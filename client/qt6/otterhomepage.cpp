#include "otterhomepage.h"

#include <QFont>
#include <QFrame>
#include <QGridLayout>
#include <QGroupBox>
#include <QHBoxLayout>
#include <QLabel>
#include <QPushButton>
#include <QScrollArea>
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

} // namespace

OtterHomePage::OtterHomePage(QWidget *parent)
    : QWidget(parent)
{
    auto *outerLayout = new QVBoxLayout(this);
    outerLayout->setContentsMargins(0, 0, 0, 0);
    outerLayout->setSpacing(0);

    auto *scrollArea = new QScrollArea(this);
    scrollArea->setWidgetResizable(true);
    scrollArea->setFrameShape(QFrame::NoFrame);
    scrollArea->setHorizontalScrollBarPolicy(Qt::ScrollBarAlwaysOff);
    scrollArea->setObjectName(QStringLiteral("homeScrollArea"));
    outerLayout->addWidget(scrollArea);

    auto *page = new QWidget;
    page->setObjectName(QStringLiteral("homeContent"));
    auto *layout = new QVBoxLayout(page);
    layout->setContentsMargins(16, 16, 16, 16);
    layout->setSpacing(12);

    auto *hero = new QFrame(page);
    hero->setObjectName(QStringLiteral("homeHero"));
    auto *heroLayout = new QHBoxLayout(hero);
    heroLayout->setContentsMargins(16, 14, 16, 14);

    auto *otter = makeLabel(QStringLiteral("/\\_/\\\n( o.o )\n > ^ <"), hero, 22, true);
    otter->setObjectName(QStringLiteral("homeHeroOtter"));
    otter->setAlignment(Qt::AlignCenter);
    otter->setMinimumWidth(120);
    heroLayout->addWidget(otter);

    auto *heroText = new QVBoxLayout;
    heroText->setSpacing(4);
    heroText->addWidget(makeLabel(QStringLiteral("Welcome to Otter Link"), hero, 20, true));
    heroText->addWidget(makeLabel(
        QStringLiteral("Your online world for modern and retro computers. See what's new, "
                       "meet other Otters, and explore the services available to you."), hero));
    heroLayout->addLayout(heroText, 1);
    layout->addWidget(hero);

    auto *news = new QGroupBox(QStringLiteral("What's New"), page);
    news->setObjectName(QStringLiteral("homeAnnouncements"));
    auto *newsLayout = new QVBoxLayout(news);
    newsLayout->setSpacing(8);
    addAnnouncement(QStringLiteral("Welcome to the new Otter Link desktop"),
                    QStringLiteral("The Qt 6 client now opens services as movable windows, "
                                   "giving Otter Link a classic online-service feel."),
                    QStringLiteral("Learn more"), QString(), news);
    addAnnouncement(QStringLiteral("People & Buddy Presence"),
                    QStringLiteral("See your buddies, organize them into groups, and watch "
                                   "their online status change while you're connected."),
                    QStringLiteral("Open People"), QStringLiteral("people"), news);
    addAnnouncement(QStringLiteral("Community Chat"),
                    QStringLiteral("Talk with the Otter Link community in the shared chat service."),
                    QStringLiteral("Open Chat"), QStringLiteral("chat"), news);
    layout->addWidget(news);

    auto *services = new QGroupBox(QStringLiteral("Explore Otter Link"), page);
    services->setObjectName(QStringLiteral("homeServices"));
    auto *servicesLayout = new QGridLayout(services);
    servicesLayout->setHorizontalSpacing(10);
    servicesLayout->setVerticalSpacing(10);

    addServiceTile(QStringLiteral("People"), QStringLiteral("Buddies and online users"),
                   QStringLiteral("👥"), QStringLiteral("people"), services);
    addServiceTile(QStringLiteral("Community Chat"), QStringLiteral("Talk with everyone online"),
                   QStringLiteral("💬"), QStringLiteral("chat"), services);
    addServiceTile(QStringLiteral("Mail"), QStringLiteral("Private messages — coming soon"),
                   QStringLiteral("✉"), QStringLiteral("mail"), services);
    addServiceTile(QStringLiteral("Boards"), QStringLiteral("Community discussions — coming soon"),
                   QStringLiteral("▤"), QStringLiteral("boards"), services);
    addServiceTile(QStringLiteral("News"), QStringLiteral("Updates and announcements"),
                   QStringLiteral("📰"), QStringLiteral("news"), services);
    addServiceTile(QStringLiteral("Games"), QStringLiteral("Online games — coming soon"),
                   QStringLiteral("🎮"), QStringLiteral("games"), services);
    layout->addWidget(services);

    auto *footer = makeLabel(
        QStringLiteral("Otter Link is actively being built. More services are on the way."),
        page);
    footer->setObjectName(QStringLiteral("homeFooter"));
    footer->setAlignment(Qt::AlignCenter);
    layout->addWidget(footer);
    layout->addStretch(1);

    scrollArea->setWidget(page);
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

    connect(button, &QPushButton::clicked, this, [this, service]() {
        emit serviceRequested(service);
    });

    auto *grid = qobject_cast<QGridLayout *>(parent->layout());
    if (!grid)
        return;

    const int index = grid->count();
    const int row = index / 2;
    const int column = index % 2;
    grid->addWidget(tile, row, column);
}
