#pragma once

#include <QJsonArray>
#include <QJsonObject>
#include <QNetworkAccessManager>
#include <QNetworkReply>
#include <QSet>
#include <QString>
#include <QStringList>
#include <QObject>

class OtterLinkClient final : public QObject
{
    Q_OBJECT

public:
    explicit OtterLinkClient(QObject *parent = nullptr);

    void setBaseUrl(const QString &url);
    QString baseUrl() const;
    QString accountName() const;

    void login(const QString &username, const QString &password);
    void loadDashboard();
    void sendChatMessage(const QString &message);
    void sendChatMessage(qint64 channelId, const QString &message);
    void loadChatChannels();
    void createChatChannel(const QString &name, bool allowOpsToCreateOps);
    void joinChatChannel(qint64 channelId);
    void leaveChatChannel(qint64 channelId);
    void setChatRole(qint64 channelId, const QString &username, const QString &role);
    void moderateChatUser(qint64 channelId, const QString &username, const QString &action);
    void addBuddy(const QString &username);
    void removeBuddy(const QString &username);
    void loadDirectConversation(const QString &username);
    void sendDirectMessage(const QString &username, const QString &message);
    void markDirectMessagesRead(const QString &username);
    void setAway(bool away);
    void loadEvents(int year, int month);
    void createEvent(const QJsonObject &event);
    void updateEvent(qint64 eventId, const QJsonObject &event);
    void deleteEvent(qint64 eventId);
    void logout();

signals:
    void loggedIn(const QString &accountName);
    void dashboardLoaded(const QStringList &buddies, const QStringList &onlineUsers,
                         const QStringList &chatMessages);
    void buddyAdded(const QString &username);
    void buddyChanged();
    void directConversationLoaded(const QJsonObject &conversation);
    void directMessageSent(const QJsonObject &message);
    void directUnreadLoaded(const QJsonArray &messages);
    void presenceLoaded(const QJsonArray &users);
    void awayChanged(bool away);
    void eventsLoaded(const QJsonArray &events, int year, int month);
    void eventChanged(const QJsonObject &event);
    void eventDeleted(qint64 eventId);
    void chatMessageSent();
    void chatChannelsLoaded(const QJsonArray &channels);
    void chatChannelLoaded(const QJsonObject &channel, const QJsonArray &members,
                           const QJsonArray &messages, const QString &role);
    void chatActionCompleted();
    void loggedOut();
    void errorOccurred(const QString &message);

private:
    QNetworkRequest request(const QString &path) const;

    QNetworkAccessManager m_network;
    QString m_baseUrl = QStringLiteral("http://127.0.0.1:9090");
    QString m_token;
    QString m_username;
    QSet<qint64> m_joinedChatChannels;
};
