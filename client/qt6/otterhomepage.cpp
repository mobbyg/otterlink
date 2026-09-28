#include "otterhomepage.h"

#include <QFont>
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
    m_scrollArea->setWidgetResizable(true);
    m_scrollArea->setFrameShape(QFrame::NoFrame);
    m_scrollArea->setHorizontalScrollBarPolicy(Qt::ScrollBarAlwaysOff);
    m_scrollArea->setObjectName(QStringLiteral("homeScrollArea"));
    outerLayout->addWidget(m_scrollArea);

    m_page = new QWidget;
    m_page->setObjectName(QStringLiteral("homeContent"));
    m_layout = new QVBoxLayout(m_page);
    m_layout->setContentsMargins(16, 16, 16, 16);
    m_layout->setSpacing(12);
    m_scrollArea->setWidget(m_page);

    buildFallback();
}

void OtterHomePage::clearPage()
{
    m_overlay = nullptr;
    if (!m_layout)
        return;

    while (QLayoutItem *item = m_layout->takeAt(0)) {
        if (QWidget *widget = item->widget())
            widget->deleteLater();
        delete item;
    }
}

void OtterHomePage::setServerScreen(const QJsonObject &screen)
{
    setServerScreen(screen, QImage());
}

void OtterHomePage::setServerScreen(const QJsonObject &screen, const QImage &background)
{
    QJsonObject content = screen.value(QStringLiteral("content")).toObject();
    if (content.isEmpty())
        return;

    m_background = background.isNull() ? QPixmap() : QPixmap::fromImage(background);
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

    clearPage();
    buildFromContent(content);
    applyBackground(m_background);
}

void OtterHomePage::applyBackground(const QPixmap &background)
{
    if (background.isNull() || !m_page)
        return;

    const QSize targetSize = m_page->size().expandedTo(size());
    if (targetSize.isEmpty())
        return;

    const Qt::AspectRatioMode mode = m_templateMode
        ? Qt::KeepAspectRatio
        : Qt::KeepAspectRatioByExpanding;
    const QPixmap scaled = background.scaled(targetSize, mode, Qt::SmoothTransformation);
    QPalette palette = m_page->palette();
    palette.setBrush(QPalette::Window, QBrush(scaled));
    m_page->setAutoFillBackground(true);
    m_page->setPalette(palette);
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
    m_overlay->setMinimumSize(640, 360);
    m_overlay->setSizePolicy(QSizePolicy::Expanding, QSizePolicy::Expanding);
    m_layout->addWidget(m_overlay, 1);

    for (const QJsonValue &value : elements) {
        const QJsonObject item = value.toObject();
        const QString type = item.value(QStringLiteral("type")).toString().trimmed().toLower();
        const QString text = item.value(QStringLiteral("text")).toString();
        QWidget *widget = nullptr;

        if (type == QStringLiteral("button")) {
            auto *button = new QPushButton(text, m_overlay);
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
            label->setAlignment(Qt::AlignLeft | Qt::AlignTop);
            widget = label;
        }

        if (!widget)
            continue;

        widget->setProperty("overlay_x", item.value(QStringLiteral("x")).toDouble(0.0));
        widget->setProperty("overlay_y", item.value(QStringLiteral("y")).toDouble(0.0));
        widget->setProperty("overlay_width", item.value(QStringLiteral("width")).toDouble(type == QStringLiteral("button") ? 0.22 : 0.30));
        widget->setProperty("overlay_height", item.value(QStringLiteral("height")).toDouble(type == QStringLiteral("button") ? 0.09 : 0.12));
        widget->show();
    }

    layoutOverlay();
}

void OtterHomePage::layoutOverlay()
{
    if (!m_overlay)
        return;

    const int width = m_overlay->width();
    const int height = m_overlay->height();
    for (QWidget *widget : m_overlay->findChildren<QWidget *>(QString(), Qt::FindDirectChildrenOnly)) {
        const double x = qBound(0.0, widget->property("overlay_x").toDouble(), 1.0);
        const double y = qBound(0.0, widget->property("overlay_y").toDouble(), 1.0);
        const double w = qBound(0.01, widget->property("overlay_width").toDouble(), 1.0);
        const double h = qBound(0.01, widget->property("overlay_height").toDouble(), 1.0);
        widget->setGeometry(qRound(x * width), qRound(y * height),
                            qMax(1, qRound(w * width)), qMax(1, qRound(h * height)));
    }
}
