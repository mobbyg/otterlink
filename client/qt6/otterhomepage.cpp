#include "otterhomepage.h"

#include <QFont>
#include <QColor>
#include <QPalette>
#include <QResizeEvent>
#include <QFrame>
#include <QGridLayout>
#include <QGroupBox>
#include <QHBoxLayout>
#include <QJsonArray>
#include <QJsonObject>
#include <QLabel>
#include <QPushButton>
#include <QScrollArea>
#include <QVariant>
#include <QSizePolicy>
#include <QVBoxLayout>
#include <QPixmap>

namespace {
constexpr int kHomeCanvasWidth = 1280;
constexpr int kHomeCanvasHeight = 720;
}

namespace {

QLabel *makeLabel(const QString &text, QWidget *parent, int pointSize = -1, bool bold = false)
{
    auto *label = new QLabel(text, parent);
    label->setWordWrap(true);

    if (pointSize > 0 || bold) {
        QFont font = label->font();
        if (pointSize > 0)
            font.setPointSize(pointSize);
        font.setBold(bold);
        label->setFont(font);
    }

    return label;
}

QString destinationService(const QJsonObject &value)
{
    const QJsonObject destination = value.value(QStringLiteral("destination")).toObject();
    if (destination.value(QStringLiteral("type")).toString() != QStringLiteral("service"))
        return QString();
    return destination.value(QStringLiteral("service")).toString().trimmed().toLower();
}

} // namespace

OtterHomePage::OtterHomePage(QWidget *parent)
    : QWidget(parent)
{
    auto *outerLayout = new QVBoxLayout(this);
    outerLayout->setContentsMargins(0, 0, 0, 0);
    outerLayout->setSpacing(0);

    m_scrollArea = new QScrollArea(this);
    // Home is a fixed 1280x720 publication canvas.
    m_scrollArea->setWidgetResizable(false);
    m_scrollArea->setFrameShape(QFrame::NoFrame);
    m_scrollArea->setHorizontalScrollBarPolicy(Qt::ScrollBarAlwaysOff);
    m_scrollArea->setVerticalScrollBarPolicy(Qt::ScrollBarAlwaysOff);
    m_scrollArea->setAlignment(Qt::AlignCenter);
    m_scrollArea->setObjectName(QStringLiteral("homeScrollArea"));
    outerLayout->addWidget(m_scrollArea);

    m_page = new QWidget;
    m_page->setObjectName(QStringLiteral("homeContent"));
    m_page->setFixedSize(kHomeCanvasWidth, kHomeCanvasHeight);

    m_backgroundLayer = new QLabel(m_page);
    m_backgroundLayer->setObjectName(QStringLiteral("homeBackgroundLayer"));
    m_backgroundLayer->setAlignment(Qt::AlignCenter);
    m_backgroundLayer->setAttribute(Qt::WA_TransparentForMouseEvents);
    m_backgroundLayer->lower();

    m_layout = new QVBoxLayout(m_page);
    m_layout->setContentsMargins(0, 0, 0, 0);
    m_layout->setSpacing(0);
    m_scrollArea->setWidget(m_page);

    buildFallback();
}

void OtterHomePage::clearPage()
{
    m_overlay = nullptr;
    if (m_backgroundLayer)
        m_backgroundLayer->lower();
    if (!m_layout)
        return;

    while (QLayoutItem *item = m_layout->takeAt(0)) {
        if (QWidget *widget = item->widget())
            delete widget;
        delete item;
    }
}

void OtterHomePage::setServerScreen(const QJsonObject &screen)
{
    setServerScreen(screen, QImage());
}

void OtterHomePage::setServerBackground(const QImage &background)
{
    if (background.isNull())
        return;

    m_background = QPixmap::fromImage(background);
    setUpdatesEnabled(false);
    applyBackground(m_background);
    setUpdatesEnabled(true);
    update();
}

void OtterHomePage::setServerAsset(qint64 assetId, const QImage &image)
{
    if (assetId < 1 || image.isNull())
        return;

    m_loadedAssets.insert(assetId, image);
    if (!m_overlay)
        return;

    const QJsonArray elements = m_overlay->property("overlay_elements").toJsonArray();
    for (const QJsonValue &value : elements) {
        const QJsonObject item = value.toObject();
        const QString type = item.value(QStringLiteral("type")).toString().trimmed().toLower();
        if (item.value(QStringLiteral("asset")).toInteger() != assetId)
            continue;
        if (type == QStringLiteral("image")) {
            addImageElement(assetId, image, item);
        } else if (type == QStringLiteral("button")) {
            for (QPushButton *button : m_overlay->findChildren<QPushButton *>()) {
                if (button->property("overlay_asset").toLongLong() == assetId)
                    applyButtonAsset(button, image);
            }
        }
    }
}

void OtterHomePage::setServerScreen(const QJsonObject &screen, const QImage &background)
{
    QJsonObject content = screen.value(QStringLiteral("content")).toObject();
    if (content.isEmpty())
        return;

    m_serverScreen = screen;
    m_background = background.isNull() ? QPixmap() : QPixmap::fromImage(background);
    const QJsonObject backgroundData = content.value(QStringLiteral("background")).toObject();
    m_backgroundFit = backgroundData.value(QStringLiteral("fit")).toString(QStringLiteral("cover")).trimmed().toLower();
    if (m_backgroundFit != QStringLiteral("contain") && m_backgroundFit != QStringLiteral("cover"))
        m_backgroundFit = QStringLiteral("cover");
    if (m_backgroundLayer)
        m_backgroundLayer->setPixmap(m_background);
    m_templateMode = !content.value(QStringLiteral("elements")).toArray().isEmpty();

    // The screen title is server-managed metadata. Keep the JSON content as the
    // source for the rest of the page, but let the screen title control the
    // visible Home heading when one is supplied.
    const QString screenTitle = screen.value(QStringLiteral("title")).toString().trimmed();
    if (!screenTitle.isEmpty()) {
        QJsonObject hero = content.value(QStringLiteral("hero")).toObject();
        hero.insert(QStringLiteral("title"), screenTitle);
        content.insert(QStringLiteral("hero"), hero);
    }

    // Build the server-managed layout as one atomic visual update. This avoids
    // exposing the intermediate empty/partially populated page while widgets
    // are being replaced and assets are arriving.
    setUpdatesEnabled(false);
    clearPage();
    buildFromContent(content);
    applyBackground(m_background);
    setUpdatesEnabled(true);
    update();
}

void OtterHomePage::applyBackground(const QPixmap &background)
{
    if (!m_page || !m_backgroundLayer)
        return;

    const QSize targetSize(kHomeCanvasWidth, kHomeCanvasHeight);
    if (targetSize.isEmpty())
        return;

    m_backgroundLayer->setGeometry(m_page->rect());
    if (background.isNull()) {
        m_backgroundLayer->clear();
        return;
    }

    const Qt::AspectRatioMode mode =
        m_backgroundFit == QStringLiteral("contain")
            ? Qt::KeepAspectRatio
            : Qt::KeepAspectRatioByExpanding;
    const QPixmap scaled = background.scaled(targetSize, mode, Qt::SmoothTransformation);
    m_backgroundLayer->setPixmap(scaled);
    m_backgroundLayer->lower();
}

void OtterHomePage::resizeEvent(QResizeEvent *event)
{
    QWidget::resizeEvent(event);
    applyBackground(m_background);
    layoutOverlay();
}

void OtterHomePage::buildFallback()
{
    QJsonObject content{
        {QStringLiteral("hero"), QJsonObject{
            {QStringLiteral("title"), QStringLiteral("Welcome to Otter Link")},
            {QStringLiteral("body"), QStringLiteral("Your online world for modern and retro computers. See what's new, meet other Otters, and explore the services available to you.")},
            {QStringLiteral("icon"), QStringLiteral("/\\_/\\\n( o.o )\n > ^ <")}
        }},
        {QStringLiteral("announcements"), QJsonArray{
            QJsonObject{
                {QStringLiteral("title"), QStringLiteral("Welcome to the new Otter Link desktop")},
                {QStringLiteral("body"), QStringLiteral("The Qt 6 client now opens services as movable windows, giving Otter Link a classic online-service feel.")},
                {QStringLiteral("action_label"), QStringLiteral("Learn more")}
            },
            QJsonObject{
                {QStringLiteral("title"), QStringLiteral("People & Buddy Presence")},
                {QStringLiteral("body"), QStringLiteral("See your buddies, organize them into groups, and watch their online status change while you're connected.")},
                {QStringLiteral("action_label"), QStringLiteral("Open People")},
                {QStringLiteral("destination"), QJsonObject{{QStringLiteral("type"), QStringLiteral("service")}, {QStringLiteral("service"), QStringLiteral("people")}}}
            },
            QJsonObject{
                {QStringLiteral("title"), QStringLiteral("Community Chat")},
                {QStringLiteral("body"), QStringLiteral("Talk with the Otter Link community in the shared chat service.")},
                {QStringLiteral("action_label"), QStringLiteral("Open Chat")},
                {QStringLiteral("destination"), QJsonObject{{QStringLiteral("type"), QStringLiteral("service")}, {QStringLiteral("service"), QStringLiteral("chat")}}}
            }
        }},
        {QStringLiteral("services"), QJsonArray{
            QJsonObject{{QStringLiteral("title"), QStringLiteral("People")}, {QStringLiteral("description"), QStringLiteral("Buddies and online users")}, {QStringLiteral("icon"), QStringLiteral("👥")}, {QStringLiteral("destination"), QJsonObject{{QStringLiteral("type"), QStringLiteral("service")}, {QStringLiteral("service"), QStringLiteral("people")}}}},
            QJsonObject{{QStringLiteral("title"), QStringLiteral("Community Chat")}, {QStringLiteral("description"), QStringLiteral("Talk with everyone online")}, {QStringLiteral("icon"), QStringLiteral("💬")}, {QStringLiteral("destination"), QJsonObject{{QStringLiteral("type"), QStringLiteral("service")}, {QStringLiteral("service"), QStringLiteral("chat")}}}},
            QJsonObject{{QStringLiteral("title"), QStringLiteral("Mail")}, {QStringLiteral("description"), QStringLiteral("Private messages — coming soon")}, {QStringLiteral("icon"), QStringLiteral("✉")}, {QStringLiteral("destination"), QJsonObject{{QStringLiteral("type"), QStringLiteral("service")}, {QStringLiteral("service"), QStringLiteral("mail")}}}},
            QJsonObject{{QStringLiteral("title"), QStringLiteral("Boards")}, {QStringLiteral("description"), QStringLiteral("Community discussions — coming soon")}, {QStringLiteral("icon"), QStringLiteral("▤")}, {QStringLiteral("destination"), QJsonObject{{QStringLiteral("type"), QStringLiteral("service")}, {QStringLiteral("service"), QStringLiteral("boards")}}}},
            QJsonObject{{QStringLiteral("title"), QStringLiteral("News")}, {QStringLiteral("description"), QStringLiteral("Updates and announcements")}, {QStringLiteral("icon"), QStringLiteral("📰")}, {QStringLiteral("destination"), QJsonObject{{QStringLiteral("type"), QStringLiteral("service")}, {QStringLiteral("service"), QStringLiteral("news")}}}},
            QJsonObject{{QStringLiteral("title"), QStringLiteral("Games")}, {QStringLiteral("description"), QStringLiteral("Online games — coming soon")}, {QStringLiteral("icon"), QStringLiteral("🎮")}, {QStringLiteral("destination"), QJsonObject{{QStringLiteral("type"), QStringLiteral("service")}, {QStringLiteral("service"), QStringLiteral("games")}}}}
        }},
        {QStringLiteral("footer"), QStringLiteral("Otter Link is actively being built. More services are on the way.")}
    };
    buildFromContent(content);
}

void OtterHomePage::buildFromContent(const QJsonObject &content)
{
    const QJsonArray elements = content.value(QStringLiteral("elements")).toArray();
    if (!elements.isEmpty()) {
        buildOverlay(elements);
        return;
    }

    const QJsonObject heroData = content.value(QStringLiteral("hero")).toObject();
    auto *hero = new QFrame(m_page);
    hero->setObjectName(QStringLiteral("homeHero"));
    auto *heroLayout = new QHBoxLayout(hero);
    heroLayout->setContentsMargins(16, 14, 16, 14);

    auto *otter = makeLabel(heroData.value(QStringLiteral("icon")).toString(QStringLiteral("/\\_/\\\n( o.o )\n > ^ <")), hero, 22, true);
    otter->setObjectName(QStringLiteral("homeHeroOtter"));
    otter->setAlignment(Qt::AlignCenter);
    otter->setMinimumWidth(120);
    heroLayout->addWidget(otter);

    auto *heroText = new QVBoxLayout;
    heroText->setSpacing(4);
    heroText->addWidget(makeLabel(heroData.value(QStringLiteral("title")).toString(QStringLiteral("Welcome to Otter Link")), hero, 20, true));
    heroText->addWidget(makeLabel(heroData.value(QStringLiteral("body")).toString(), hero));
    heroLayout->addLayout(heroText, 1);
    m_layout->addWidget(hero);

    const QJsonArray announcements = content.value(QStringLiteral("announcements")).toArray();
    if (!announcements.isEmpty()) {
        auto *news = new QGroupBox(QStringLiteral("What's New"), m_page);
        news->setObjectName(QStringLiteral("homeAnnouncements"));
        auto *newsLayout = new QVBoxLayout(news);
        newsLayout->setSpacing(8);
        for (const QJsonValue &value : announcements) {
            const QJsonObject item = value.toObject();
            addAnnouncement(item.value(QStringLiteral("title")).toString(),
                            item.value(QStringLiteral("body")).toString(),
                            item.value(QStringLiteral("action_label")).toString(),
                            destinationService(item), news);
        }
        m_layout->addWidget(news);
    }

    const QJsonArray services = content.value(QStringLiteral("services")).toArray();
    if (!services.isEmpty()) {
        auto *group = new QGroupBox(QStringLiteral("Explore Otter Link"), m_page);
        group->setObjectName(QStringLiteral("homeServices"));
        auto *grid = new QGridLayout(group);
        grid->setHorizontalSpacing(10);
        grid->setVerticalSpacing(10);
        for (const QJsonValue &value : services) {
            const QJsonObject item = value.toObject();
            addServiceTile(item.value(QStringLiteral("title")).toString(),
                           item.value(QStringLiteral("description")).toString(),
                           item.value(QStringLiteral("icon")).toString(),
                           destinationService(item), group);
        }
        m_layout->addWidget(group);
    }

    const QString footerText = content.value(QStringLiteral("footer")).toString();
    if (!footerText.isEmpty()) {
        auto *footer = makeLabel(footerText, m_page);
        footer->setObjectName(QStringLiteral("homeFooter"));
        footer->setAlignment(Qt::AlignCenter);
        m_layout->addWidget(footer);
    }
    m_layout->addStretch(1);
}

void OtterHomePage::addAnnouncement(const QString &title, const QString &body,
                                    const QString &actionLabel, const QString &service,
                                    QWidget *parent)
{
    auto *card = new QFrame(parent);
    card->setObjectName(QStringLiteral("homeAnnouncementCard"));
    auto *layout = new QHBoxLayout(card);
    layout->setContentsMargins(10, 8, 10, 8);
    layout->setSpacing(10);

    auto *text = new QVBoxLayout;
    text->setSpacing(2);
    text->addWidget(makeLabel(title, card, 11, true));
    text->addWidget(makeLabel(body, card));
    layout->addLayout(text, 1);

    if (!service.isEmpty()) {
        auto *button = new QPushButton(actionLabel, card);
        button->setObjectName(QStringLiteral("homeActionButton"));
        button->setSizePolicy(QSizePolicy::Fixed, QSizePolicy::Preferred);
        connect(button, &QPushButton::clicked, this, [this, service]() {
            emit serviceRequested(service);
        });
        layout->addWidget(button);
    }
    parent->layout()->addWidget(card);
}

void OtterHomePage::addServiceTile(const QString &title, const QString &description,
                                   const QString &icon, const QString &service,
                                   QWidget *parent)
{
    auto *tile = new QFrame(parent);
    tile->setObjectName(QStringLiteral("homeServiceTile"));
    tile->setMinimumHeight(86);
    tile->setSizePolicy(QSizePolicy::Expanding, QSizePolicy::Preferred);

    auto *layout = new QVBoxLayout(tile);
    layout->setContentsMargins(10, 8, 10, 8);
    layout->setSpacing(2);

    auto *button = new QPushButton(tile);
    button->setObjectName(QStringLiteral("homeTileButton"));
    button->setText(icon + QStringLiteral("  ") + title);
    button->setSizePolicy(QSizePolicy::Expanding, QSizePolicy::Preferred);
    layout->addWidget(button);
    layout->addWidget(makeLabel(description, tile));

    if (!service.isEmpty()) {
        connect(button, &QPushButton::clicked, this, [this, service]() {
            emit serviceRequested(service);
        });
    } else {
        button->setEnabled(false);
    }

    auto *grid = qobject_cast<QGridLayout *>(parent->layout());
    if (!grid)
        return;

    const int index = grid->count();
    const int row = index / 2;
    const int column = index % 2;
    grid->addWidget(tile, row, column);
}

void OtterHomePage::buildOverlay(const QJsonArray &elements)
{
    m_overlay = new QWidget(m_page);
    m_overlay->setObjectName(QStringLiteral("homeOverlay"));
    m_overlay->setAttribute(Qt::WA_TranslucentBackground);
    // Elements are positioned against a fixed 1280x720 canvas. Do not let
    // the service window resize this coordinate system.
    m_overlay->setFixedSize(kHomeCanvasWidth, kHomeCanvasHeight);
    m_overlay->setProperty("overlay_elements", elements);
    m_layout->addWidget(m_overlay, 1);

    for (const QJsonValue &value : elements) {
        const QJsonObject item = value.toObject();
        const QString type = item.value(QStringLiteral("type")).toString().trimmed().toLower();
        const QString text = item.value(QStringLiteral("text")).toString();
        QWidget *widget = nullptr;

        if (type == QStringLiteral("button")) {
            auto *button = new QPushButton(text, m_overlay);
            const qint64 assetId = item.value(QStringLiteral("asset")).toInteger();
            button->setProperty("overlay_asset", assetId);
            if (assetId > 0 && m_loadedAssets.contains(assetId)) {
                applyButtonAsset(button, m_loadedAssets.value(assetId));
            } else {
                const QString background = item.value(QStringLiteral("background")).toString(QStringLiteral("#efa00b"));
                const QString foreground = item.value(QStringLiteral("color")).toString(QStringLiteral("#591f0a"));
                button->setStyleSheet(QStringLiteral(
                    "QPushButton { background: %1; color: %2; border: 1px solid rgba(255,255,255,0.35); border-radius: 8px; padding: 4px; }")
                    .arg(background, foreground));
            }
            const QString service = destinationService(item);
            if (!service.isEmpty()) {
                connect(button, &QPushButton::clicked, this, [this, service]() {
                    emit serviceRequested(service);
                });
            } else {
                button->setEnabled(false);
            }
            widget = button;
        } else if (type == QStringLiteral("text")) {
            auto *label = new QLabel(text, m_overlay);
            label->setWordWrap(true);
            label->setSizePolicy(QSizePolicy::Ignored, QSizePolicy::Ignored);
            label->setTextInteractionFlags(Qt::NoTextInteraction);
            label->setAttribute(Qt::WA_TranslucentBackground);
            label->setStyleSheet(QStringLiteral("QLabel { background: transparent; }"));

            Qt::Alignment alignment = Qt::AlignLeft | Qt::AlignTop;
            const QString align = item.value(QStringLiteral("align")).toString().trimmed().toLower();
            if (align == QStringLiteral("center"))
                alignment = Qt::AlignHCenter | Qt::AlignTop;
            else if (align == QStringLiteral("right"))
                alignment = Qt::AlignRight | Qt::AlignTop;
            label->setAlignment(alignment);

            QFont font = label->font();
            font.setPixelSize(qBound(8, item.value(QStringLiteral("font_size")).toInt(32), 200));
            font.setWeight(static_cast<QFont::Weight>(qBound(1, item.value(QStringLiteral("weight")).toInt(700), 1000)));
            label->setFont(font);

            const QColor textColor(item.value(QStringLiteral("color")).toString(QStringLiteral("#ffffff")));
            label->setStyleSheet(QStringLiteral("QLabel { background: transparent; color: %1; }")
                                      .arg(textColor.isValid() ? textColor.name() : QStringLiteral("#ffffff")));
            widget = label;
        } else if (type == QStringLiteral("image")) {
            const qint64 assetId = item.value(QStringLiteral("asset")).toInteger();
            if (assetId > 0 && m_loadedAssets.contains(assetId))
                addImageElement(assetId, m_loadedAssets.value(assetId), item);
            continue;
        }

        if (!widget)
            continue;

        widget->setProperty("overlay_x", item.value(QStringLiteral("x")).toDouble(0.0));
        widget->setProperty("overlay_y", item.value(QStringLiteral("y")).toDouble(0.0));
        widget->setProperty("overlay_width", item.value(QStringLiteral("width")).toDouble(
            type == QStringLiteral("button") ? 0.30 : type == QStringLiteral("text") ? 0.34 : 0.24));
        widget->setProperty("overlay_height", item.value(QStringLiteral("height")).toDouble(
            type == QStringLiteral("button") ? 0.11 : type == QStringLiteral("text") ? 0.10 : 0.24));
        if (type == QStringLiteral("text")) {
            widget->setProperty("overlay_font_size", item.value(QStringLiteral("font_size")).toDouble(32.0));
        }
        widget->show();
    }

    layoutOverlay();
}

void OtterHomePage::applyButtonAsset(QPushButton *button, const QImage &image)
{
    if (!button || image.isNull())
        return;

    button->setText(QString());
    button->setIcon(QIcon(QPixmap::fromImage(image)));
    button->setFlat(true);
    button->setStyleSheet(QStringLiteral("QPushButton { border: none; padding: 0px; background: transparent; }"));
    button->setProperty("overlay_graphical_button", true);
    button->setIconSize(button->size());
}

void OtterHomePage::addImageElement(qint64 assetId, const QImage &image, const QJsonObject &item)
{
    if (!m_overlay || assetId < 1 || image.isNull())
        return;

    const QString key = QStringLiteral("overlay_image_%1_%2_%3")
                            .arg(assetId)
                            .arg(item.value(QStringLiteral("x")).toDouble())
                            .arg(item.value(QStringLiteral("y")).toDouble());
    if (m_overlay->findChild<QLabel *>(key))
        return;

    auto *label = new QLabel(m_overlay);
    label->setObjectName(key);
    label->setAlignment(Qt::AlignCenter);
    label->setAttribute(Qt::WA_TranslucentBackground);
    label->setProperty("overlay_x", item.value(QStringLiteral("x")).toDouble(0.0));
    label->setProperty("overlay_y", item.value(QStringLiteral("y")).toDouble(0.0));
    label->setProperty("overlay_width", item.value(QStringLiteral("width")).toDouble(0.20));
    label->setProperty("overlay_height", item.value(QStringLiteral("height")).toDouble(0.20));
    label->setProperty("overlay_image", true);
    label->setProperty("overlay_fit", item.value(QStringLiteral("fit")).toString(QStringLiteral("contain")));
    label->setProperty("overlay_source_image", QVariant::fromValue(image));
    label->show();
}

void OtterHomePage::layoutOverlay()
{
    if (!m_overlay)
        return;

    const int width = kHomeCanvasWidth;
    const int height = kHomeCanvasHeight;
    for (QWidget *widget : m_overlay->findChildren<QWidget *>(QString(), Qt::FindDirectChildrenOnly)) {
        const double x = qBound(0.0, widget->property("overlay_x").toDouble(), 1.0);
        const double y = qBound(0.0, widget->property("overlay_y").toDouble(), 1.0);
        const double w = qBound(0.01, widget->property("overlay_width").toDouble(), 1.0);
        const double h = qBound(0.01, widget->property("overlay_height").toDouble(), 1.0);
        const int widgetWidth = qMax(1, qRound(w * width));
        const int widgetHeight = qMax(1, qRound(h * height));
        widget->setGeometry(qRound(x * width), qRound(y * height), widgetWidth, widgetHeight);
        // Overlay children are deliberately clipped to their stored bounding box.
        // The editor owns the position/size; the client must not let text or
        // graphical assets paint outside that rectangle.

        if (auto *label = qobject_cast<QLabel *>(widget); label && widget->property("overlay_font_size").isValid()) {
            QFont font = label->font();
            const int scaledSize = qMax(8, qRound(widget->property("overlay_font_size").toDouble() * width / 1280.0));
            font.setPixelSize(scaledSize);
            label->setFont(font);
        }

        if (widget->property("overlay_image").toBool()) {
            const QImage image = widget->property("overlay_source_image").value<QImage>();
            if (!image.isNull()) {
                const Qt::AspectRatioMode mode =
                    widget->property("overlay_fit").toString() == QStringLiteral("cover")
                        ? Qt::KeepAspectRatioByExpanding
                        : Qt::KeepAspectRatio;
                const QPixmap pixmap = QPixmap::fromImage(
                    image.scaled(QSize(widgetWidth, widgetHeight), mode, Qt::SmoothTransformation));
                if (auto *label = qobject_cast<QLabel *>(widget))
                    label->setPixmap(pixmap);
            }
        } else if (widget->property("overlay_graphical_button").toBool()) {
            if (auto *button = qobject_cast<QPushButton *>(widget))
                button->setIconSize(QSize(widgetWidth, widgetHeight));
        }
    }
}
