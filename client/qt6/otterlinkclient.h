#pragma once

#include <QObject>
#include <QNetworkAccessManager>
#include <QNetworkReply>
#include <QString>
#include <QStringList>

class OtterLinkClient final : public QObject
{
    Q_OBJECT

public:
    explicit OtterLinkClient(QObject *parent = nullptr);

    void setBaseUrl(const QString &url);
    QString baseUrl() const;

    void login(const QString &username, const QString &password);
    void loadDashboard();
    void sendChatMessage(const QString &message);
    void addBuddy(const QString &username);
    void removeBuddy(const QString &username);
    void logout();

signals:
    void loggedIn(const QString &displayName);
    void dashboardLoaded(const QStringList &buddies, const QStringList &onlineUsers,
                         const QStringList &chatMessages);
    void buddyChanged();
    void chatMessageSent();
    void loggedOut();
    void errorOccurred(const QString &message);

private:
    QNetworkRequest request(const QString &path) const;

    QNetworkAccessManager m_network;
    QString m_baseUrl = QStringLiteral("http://127.0.0.1:9090");
    QString m_token;
};
