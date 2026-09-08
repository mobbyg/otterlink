#include "otterlinkclient.h"

#include <QJsonArray>
#include <QJsonDocument>
#include <QJsonObject>
#include <QNetworkRequest>
#include <QUrl>

#include <functional>
#include <memory>

OtterLinkClient::OtterLinkClient(QObject *parent)
    : QObject(parent)
{
}

void OtterLinkClient::setBaseUrl(const QString &url)
{
    m_baseUrl = url.trimmed();
    while (m_baseUrl.endsWith('/'))
        m_baseUrl.chop(1);
}

QString OtterLinkClient::baseUrl() const
{
    return m_baseUrl;
}

QNetworkRequest OtterLinkClient::request(const QString &path) const
{
    QNetworkRequest result(QUrl(m_baseUrl + path));
    if (!m_token.isEmpty())
        result.setRawHeader("Authorization", QByteArray("Bearer ") + m_token.toUtf8());
    result.setHeader(QNetworkRequest::ContentTypeHeader, QStringLiteral("application/json"));
    return result;
}

void OtterLinkClient::login(const QString &username, const QString &password)
{
    QJsonObject body{{QStringLiteral("username"), username},
                     {QStringLiteral("password"), password}};
    QNetworkRequest req = request(QStringLiteral("/api/auth/login"));
    auto *reply = m_network.post(req, QJsonDocument(body).toJson(QJsonDocument::Compact));
    connect(reply, &QNetworkReply::finished, this, [this, reply]() {
        if (reply->error() != QNetworkReply::NoError) {
            emit errorOccurred(reply->errorString());
            reply->deleteLater();
            return;
        }
        const QJsonDocument doc = QJsonDocument::fromJson(reply->readAll());
        const QJsonObject obj = doc.object();
        const QString token = obj.value(QStringLiteral("token")).toString();
        const QJsonObject user = obj.value(QStringLiteral("user")).toObject();
        if (token.isEmpty()) {
            emit errorOccurred(QStringLiteral("Login response did not contain a token."));
        } else {
            m_token = token;
            emit loggedIn(user.value(QStringLiteral("display_name")).toString(
                user.value(QStringLiteral("username")).toString()));
        }
        reply->deleteLater();
    });
}

void OtterLinkClient::loadDashboard()
{
    struct Pending {
        QStringList buddies;
        QStringList online;
        QStringList chat;
        int remaining = 3;
    };
    auto pending = std::make_shared<Pending>();

    const auto finish = [this, pending]() {
        if (--pending->remaining == 0)
            emit dashboardLoaded(pending->buddies, pending->online, pending->chat);
    };

    const auto load = [this, pending, finish](const QString &path,
                                              const std::function<void(const QJsonObject &)> &handler) {
        auto *reply = m_network.get(request(path));
        connect(reply, &QNetworkReply::finished, this, [this, reply, handler, finish]() {
            if (reply->error() != QNetworkReply::NoError) {
                emit errorOccurred(reply->errorString());
                reply->deleteLater();
                finish();
                return;
            }
            const QJsonDocument doc = QJsonDocument::fromJson(reply->readAll());
            handler(doc.object());
            reply->deleteLater();
            finish();
        });
    };

    load(QStringLiteral("/api/buddies"), [pending](const QJsonObject &obj) {
        for (const auto value : obj.value(QStringLiteral("buddies")).toArray()) {
            const QJsonObject buddy = value.toObject();
            pending->buddies << buddy.value(QStringLiteral("display_name")).toString(
                buddy.value(QStringLiteral("username")).toString());
        }
    });
    load(QStringLiteral("/api/presence"), [pending](const QJsonObject &obj) {
        for (const auto value : obj.value(QStringLiteral("users")).toArray()) {
            const QJsonObject user = value.toObject();
            pending->online << user.value(QStringLiteral("display_name")).toString(
                user.value(QStringLiteral("username")).toString());
        }
    });
    load(QStringLiteral("/api/chat"), [pending](const QJsonObject &obj) {
        for (const auto value : obj.value(QStringLiteral("messages")).toArray()) {
            const QJsonObject message = value.toObject();
            const QJsonObject from = message.value(QStringLiteral("from")).toObject();
            pending->chat << QStringLiteral("%1: %2")
                                 .arg(from.value(QStringLiteral("display_name")).toString(
                                          from.value(QStringLiteral("username")).toString()),
                                      message.value(QStringLiteral("message")).toString());
        }
    });
}

void OtterLinkClient::logout()
{
    if (m_token.isEmpty()) {
        emit loggedOut();
        return;
    }
    auto *reply = m_network.post(request(QStringLiteral("/api/auth/logout")), QByteArray());
    connect(reply, &QNetworkReply::finished, this, [this, reply]() {
        m_token.clear();
        emit loggedOut();
        reply->deleteLater();
    });
}
