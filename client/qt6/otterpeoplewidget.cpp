#include "otterpeoplewidget.h"
#include "otterlinkclient.h"

#include <QFont>
#include <QHBoxLayout>
#include <QInputDialog>
#include <QLabel>
#include <QLineEdit>
#include <QMenu>
#include <QPushButton>
#include <QSignalBlocker>
#include <QTreeWidget>
#include <QTreeWidgetItem>
#include <QVBoxLayout>

namespace {
const QStringList kGroups = {QStringLiteral("Buddies"), QStringLiteral("Family"), QStringLiteral("Co-worker")};
}

OtterPeopleWidget::OtterPeopleWidget(OtterLinkClient *client, QWidget *parent)
    : QWidget(parent), m_client(client)
{
    auto *layout = new QVBoxLayout(this);
    layout->setContentsMargins(8, 8, 8, 8);

    auto *status = new QLabel(QStringLiteral("Status: Online"), this);
    status->setObjectName(QStringLiteral("peopleStatusLabel"));
    layout->addWidget(status);

    m_tree = new QTreeWidget(this);
    m_tree->setHeaderHidden(true);
    m_tree->setRootIsDecorated(true);
    m_tree->setItemsExpandable(true);
    m_tree->setUniformRowHeights(true);
    m_tree->setSelectionMode(QAbstractItemView::SingleSelection);
    m_tree->setHorizontalScrollBarPolicy(Qt::ScrollBarAsNeeded);
    m_tree->setContextMenuPolicy(Qt::CustomContextMenu);
    layout->addWidget(m_tree, 1);

    auto *top = new QHBoxLayout;
    auto *add = new QPushButton(QStringLiteral("＋ Add Buddy"), this);
    auto *remove = new QPushButton(QStringLiteral("− Remove"), this);
    top->addWidget(add);
    top->addWidget(remove);
    layout->addLayout(top);

    auto *bottom = new QHBoxLayout;
    auto *away = new QPushButton(QStringLiteral("😴 Away"), this);
    away->setObjectName(QStringLiteral("awayButton"));
    auto *message = new QPushButton(QStringLiteral("📨 Private Message"), this);
    bottom->addWidget(away);
    bottom->addWidget(message);
    layout->addLayout(bottom);

    connect(add, &QPushButton::clicked, this, &OtterPeopleWidget::addBuddy);
    connect(remove, &QPushButton::clicked, this, &OtterPeopleWidget::removeBuddy);
    connect(away, &QPushButton::clicked, this, &OtterPeopleWidget::toggleAway);
    connect(message, &QPushButton::clicked, this, &OtterPeopleWidget::privateMessage);
    connect(m_tree, &QTreeWidget::itemSelectionChanged, this, &OtterPeopleWidget::buddySelectionChanged);
    connect(m_tree, &QTreeWidget::itemDoubleClicked, this,
            [this](QTreeWidgetItem *item) {
                const QString username = item->data(0, Qt::UserRole).toString();
                if (!username.isEmpty()) emit privateMessageRequested(username);
            });
    connect(m_tree, &QTreeWidget::customContextMenuRequested, this, &OtterPeopleWidget::showContextMenu);

    if (m_client) {
        connect(m_client, &OtterLinkClient::buddyAdded, this, [this](const QString &) {
            m_client->loadDashboard();
        });
    }
}

void OtterPeopleWidget::setBuddies(const QStringList &buddies, const QStringList &onlineUsers)
{
    m_buddies = buddies;
    m_online = onlineUsers;
    rebuild();
}

void OtterPeopleWidget::setPresence(const QJsonArray &users)
{
    m_status.clear();
    m_away = false;
    for (const QJsonValue &value : users) {
        const QJsonObject object = value.toObject();
        const QString username = object.value(QStringLiteral("username")).toString().trimmed();
        const QString status = object.value(QStringLiteral("status")).toString().trimmed();
        if (!username.isEmpty()) {
            m_status.insert(username, status.isEmpty() ? QStringLiteral("online") : status);
            if (m_client && username.compare(m_client->accountName(), Qt::CaseInsensitive) == 0)
                m_away = status.compare(QStringLiteral("away"), Qt::CaseInsensitive) == 0;
        }
    }
    auto *statusLabel = findChild<QLabel *>(QStringLiteral("peopleStatusLabel"));
    if (statusLabel)
        statusLabel->setText(m_away ? QStringLiteral("Status: Away") : QStringLiteral("Status: Online"));
    rebuild();
}

void OtterPeopleWidget::setUnread(const QJsonArray &messages)
{
    m_unread.clear();
    for (const QJsonValue &value : messages) {
        const QJsonObject object = value.toObject();
        const QString username = object.value(QStringLiteral("username")).toString();
        const int count = object.value(QStringLiteral("count")).toInt();
        if (!username.isEmpty() && count > 0)
            m_unread.insert(username, count);
    }
    rebuild();
}

QString OtterPeopleWidget::selectedUsername() const
{
    const auto items = m_tree->selectedItems();
    if (items.isEmpty()) return {};
    return items.first()->data(0, Qt::UserRole).toString();
}

void OtterPeopleWidget::rebuild()
{
    QString selected = selectedUsername();
    QSignalBlocker blocker(m_tree);
    m_tree->clear();

    QHash<QString, QTreeWidgetItem *> groups;
    for (const QString &name : kGroups) {
        auto *item = new QTreeWidgetItem(m_tree);
        item->setText(0, name);
        item->setExpanded(true);
        item->setFont(0, QFont(QStringLiteral("Sans Serif"), -1, QFont::Bold));
        groups.insert(name, item);
    }
    auto *offline = new QTreeWidgetItem(m_tree);
    offline->setText(0, QStringLiteral("Offline"));
    offline->setExpanded(true);
    offline->setFont(0, QFont(QStringLiteral("Sans Serif"), -1, QFont::Bold));

    for (const QString &buddy : m_buddies) {
        const bool online = m_online.contains(buddy, Qt::CaseInsensitive);
        QString group = m_groups.value(buddy, QStringLiteral("Buddies"));
        if (!groups.contains(group)) group = QStringLiteral("Buddies");
        auto *item = new QTreeWidgetItem(online ? groups.value(group) : offline);
        const bool away = online && m_status.value(buddy).compare(QStringLiteral("away"), Qt::CaseInsensitive) == 0;
        QString left = away ? QStringLiteral("😴 ") : (online ? QStringLiteral("● ") : QStringLiteral("○ "));
        QString right;
        const int unread = m_unread.value(buddy, 0);
        if (unread > 0) right = QStringLiteral(" 📨");
        item->setText(0, left + buddy + right);
        item->setData(0, Qt::UserRole, buddy);
        if (!online) {
            QFont font = item->font(0);
            font.setItalic(true);
            item->setFont(0, font);
        }
    }

    offline->setHidden(offline->childCount() == 0);
    for (QTreeWidgetItem *group : groups)
        group->setHidden(group->childCount() == 0);
    m_tree->expandAll();

    if (!selected.isEmpty()) {
        const auto matches = m_tree->findItems(QStringLiteral("*%1").arg(selected),
                                               Qt::MatchWildcard | Qt::MatchRecursive);
        for (auto *item : matches) {
            if (item->data(0, Qt::UserRole).toString().compare(selected, Qt::CaseInsensitive) == 0) {
                m_tree->setCurrentItem(item);
                break;
            }
        }
    }
}

void OtterPeopleWidget::addBuddy()
{
    bool ok = false;
    const QString username = QInputDialog::getText(this, QStringLiteral("Add Buddy"),
                                                    QStringLiteral("Username:"), QLineEdit::Normal,
                                                    QString(), &ok).trimmed();
    if (ok && !username.isEmpty() && m_client)
        m_client->addBuddy(username);
}

void OtterPeopleWidget::removeBuddy()
{
    const QString username = selectedUsername();
    if (!username.isEmpty() && m_client)
        m_client->removeBuddy(username);
}

void OtterPeopleWidget::privateMessage()
{
    const QString username = selectedUsername();
    if (!username.isEmpty())
        emit privateMessageRequested(username);
}

void OtterPeopleWidget::toggleAway()
{
    m_away = !m_away;
    if (auto *statusLabel = findChild<QLabel *>(QStringLiteral("peopleStatusLabel")))
        statusLabel->setText(m_away ? QStringLiteral("Status: Away") : QStringLiteral("Status: Online"));
    if (auto *awayButton = findChild<QPushButton *>(QStringLiteral("awayButton")))
        awayButton->setText(m_away ? QStringLiteral("● Online") : QStringLiteral("😴 Away"));
    if (m_client) m_client->setAway(m_away);
    emit awayRequested(m_away);
}

void OtterPeopleWidget::buddySelectionChanged()
{
    // Selection state is intentionally owned by this widget; MainWindow only
    // needs the username when a DM is requested.
}

void OtterPeopleWidget::showContextMenu(const QPoint &pos)
{
    const QString username = selectedUsername();
    if (username.isEmpty()) return;
    QMenu menu(this);
    QAction *message = menu.addAction(QStringLiteral("Send Private Message"));
    connect(message, &QAction::triggered, this, &OtterPeopleWidget::privateMessage);
    menu.exec(m_tree->viewport()->mapToGlobal(pos));
}
