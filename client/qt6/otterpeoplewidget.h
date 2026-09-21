#pragma once

#include <QHash>
#include <QJsonArray>
#include <QStringList>
#include <QWidget>

class QMenu;
class QTreeWidget;
class QTreeWidgetItem;
class OtterLinkClient;

class OtterPeopleWidget final : public QWidget
{
    Q_OBJECT
public:
    explicit OtterPeopleWidget(OtterLinkClient *client, QWidget *parent = nullptr);
    void setBuddies(const QStringList &buddies, const QStringList &onlineUsers);
    void setUnread(const QJsonArray &messages);
    void setPresence(const QJsonArray &users);
    bool away() const { return m_away; }
    void toggleAway();

signals:
    void privateMessageRequested(const QString &username);
    void awayRequested(bool away);

private slots:
    void addBuddy();
    void removeBuddy();
    void privateMessage();
    void buddySelectionChanged();

private:
    QString selectedUsername() const;
    void rebuild();
    void showContextMenu(const QPoint &pos);

    OtterLinkClient *m_client = nullptr;
    QTreeWidget *m_tree = nullptr;
    QHash<QString, QString> m_groups;
    QHash<QString, int> m_unread;
    QHash<QString, QString> m_status;
    QStringList m_buddies;
    QStringList m_online;
    bool m_away = false;
};
