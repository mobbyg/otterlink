#include "otterlinkclient.h"

#include <QJsonArray>
#include <QJsonDocument>
#include <QJsonObject>
#include <QNetworkReply>
#include <QNetworkRequest>
#include <QUrl>

#include <functional>
#include <memory>

namespace {

QString serverErrorMessage(QNetworkReply *reply, const QString &fallback)
{
    const QByteArray body = reply->readAll().trimmed();
    if (!body.isEmpty()) {
        const QJsonDocument doc = QJsonDocument::fromJson(body);
        if (doc.isObject()) {
            const QString message = doc.object().value(QStringLiteral("error")).toString().trimmed();
            if (!message.isEmpty())
                return message;
        }

        const QString text = QString::fromUtf8(body).trimmed();
        if (!text.isEmpty() && text.size() <= 500)
            return text;
    }

    return fallback;
}

} // namespace

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

QString OtterLinkClient::accountName() const
{
    return m_username;
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
    connect(reply, &QNetworkReply::finished, this, [this, reply, username]() {
        if (reply->error() != QNetworkReply::NoError) {
            emit errorOccurred(serverErrorMessage(reply, reply->errorString()));
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
            m_username = user.value(QStringLiteral("username")).toString().trimmed();
            if (m_username.isEmpty())
                m_username = user.value(QStringLiteral("display_name")).toString().trimmed();
            if (m_username.isEmpty())
                m_username = username.trimmed();
            emit loggedIn(m_username);
        }
        reply->deleteLater();
    });
}

void OtterLinkClient::loadDashboard()
{
    if (m_token.isEmpty())
        return;

    struct Pending {
        QStringList buddies;
        QStringList online;
        QStringList chat;
        int remaining = 4;
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
                emit errorOccurred(serverErrorMessage(reply, reply->errorString()));
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
            pending->buddies << buddy.value(QStringLiteral("username")).toString();
        }
    });
    load(QStringLiteral("/api/presence"), [pending, this](const QJsonObject &obj) {
        emit presenceLoaded(obj.value(QStringLiteral("users")).toArray());
        for (const auto value : obj.value(QStringLiteral("users")).toArray()) {
            const QJsonObject user = value.toObject();
            const QString username = user.value(QStringLiteral("username")).toString().trimmed();
            if (!username.isEmpty()
                && username.compare(m_username, Qt::CaseInsensitive) != 0) {
                pending->online << username;
            }
        }
        pending->online.removeDuplicates();
        pending->online.sort(Qt::CaseInsensitive);
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
    load(QStringLiteral("/api/messages/unread"), [this](const QJsonObject &obj) {
        emit directUnreadLoaded(obj.value(QStringLiteral("messages")).toArray());
    });
}

void OtterLinkClient::sendChatMessage(const QString &message)
{
    QJsonObject body{{QStringLiteral("message"), message}};
    auto *reply = m_network.post(request(QStringLiteral("/api/chat")),
                                 QJsonDocument(body).toJson(QJsonDocument::Compact));
    connect(reply, &QNetworkReply::finished, this, [this, reply]() {
        if (reply->error() != QNetworkReply::NoError)
            emit errorOccurred(serverErrorMessage(reply, reply->errorString()));
        else
            emit chatMessageSent();
        reply->deleteLater();
    });
}

void OtterLinkClient::sendChatMessage(qint64 channelId, const QString &message)
{
    QJsonObject body{{QStringLiteral("message"), message}};
    auto *reply = m_network.post(
        request(QStringLiteral("/api/chat/channels/%1/messages").arg(channelId)),
        QJsonDocument(body).toJson(QJsonDocument::Compact));
    connect(reply, &QNetworkReply::finished, this, [this, reply]() {
        if (reply->error() != QNetworkReply::NoError)
            emit errorOccurred(serverErrorMessage(reply, reply->errorString()));
        else
            emit chatMessageSent();
        reply->deleteLater();
    });
}

void OtterLinkClient::loadChatChannels()
{
    auto *reply = m_network.get(request(QStringLiteral("/api/chat/channels")));
    connect(reply, &QNetworkReply::finished, this, [this, reply]() {
        if (reply->error() != QNetworkReply::NoError) {
            emit errorOccurred(serverErrorMessage(reply, reply->errorString()));
        } else {
            const QJsonDocument doc = QJsonDocument::fromJson(reply->readAll());
            emit chatChannelsLoaded(doc.object().value(QStringLiteral("channels")).toArray());
        }
        reply->deleteLater();
    });
}

void OtterLinkClient::createChatChannel(const QString &name, bool allowOpsToCreateOps)
{
    QJsonObject body{{QStringLiteral("name"), name},
                     {QStringLiteral("allow_ops_to_create_ops"), allowOpsToCreateOps}};
    auto *reply = m_network.post(request(QStringLiteral("/api/chat/channels")),
                                 QJsonDocument(body).toJson(QJsonDocument::Compact));
    connect(reply, &QNetworkReply::finished, this, [this, reply]() {
        if (reply->error() != QNetworkReply::NoError) {
            emit errorOccurred(serverErrorMessage(reply, reply->errorString()));
        } else {
            emit chatActionCompleted();
            loadChatChannels();
        }
        reply->deleteLater();
    });
}

void OtterLinkClient::joinChatChannel(qint64 channelId)
{
    auto *reply = m_network.post(
        request(QStringLiteral("/api/chat/channels/%1/join").arg(channelId)), QByteArray());
    connect(reply, &QNetworkReply::finished, this, [this, reply, channelId]() {
        if (reply->error() != QNetworkReply::NoError) {
            emit errorOccurred(serverErrorMessage(reply, reply->errorString()));
        } else {
            m_joinedChatChannels.insert(channelId);
            const QJsonObject obj = QJsonDocument::fromJson(reply->readAll()).object();
            emit chatChannelLoaded(
                obj.value(QStringLiteral("channel")).toObject(),
                obj.value(QStringLiteral("members")).toArray(),
                obj.value(QStringLiteral("messages")).toArray(),
                obj.value(QStringLiteral("member")).toObject().value(QStringLiteral("role")).toString());
        }
        reply->deleteLater();
    });
}

void OtterLinkClient::leaveChatChannel(qint64 channelId)
{
    auto *reply = m_network.post(
        request(QStringLiteral("/api/chat/channels/%1/leave").arg(channelId)), QByteArray());
    connect(reply, &QNetworkReply::finished, this, [this, reply, channelId]() {
        if (reply->error() != QNetworkReply::NoError) {
            emit errorOccurred(serverErrorMessage(reply, reply->errorString()));
        } else {
            m_joinedChatChannels.remove(channelId);
            emit chatActionCompleted();
        }
        reply->deleteLater();
    });
}

void OtterLinkClient::setChatRole(qint64 channelId, const QString &username, const QString &role)
{
    QJsonObject body{{QStringLiteral("role"), role}};
    const QString encoded = QString::fromUtf8(QUrl::toPercentEncoding(username));
    const QString path = QStringLiteral("/api/chat/channels/%1/users/%2/role")
                             .arg(channelId)
                             .arg(encoded);
    auto *reply = m_network.post(request(path),
                                 QJsonDocument(body).toJson(QJsonDocument::Compact));
    connect(reply, &QNetworkReply::finished, this, [this, reply]() {
        if (reply->error() != QNetworkReply::NoError)
            emit errorOccurred(serverErrorMessage(reply, reply->errorString()));
        else
            emit chatActionCompleted();
        reply->deleteLater();
    });
}

void OtterLinkClient::moderateChatUser(qint64 channelId, const QString &username,
                                       const QString &action)
{
    QJsonObject body{{QStringLiteral("action"), action}};
    const QString encoded = QString::fromUtf8(QUrl::toPercentEncoding(username));
    const QString path = QStringLiteral("/api/chat/channels/%1/users/%2/moderate")
                             .arg(channelId)
                             .arg(encoded);
    auto *reply = m_network.post(request(path),
                                 QJsonDocument(body).toJson(QJsonDocument::Compact));
    connect(reply, &QNetworkReply::finished, this, [this, reply]() {
        if (reply->error() != QNetworkReply::NoError)
            emit errorOccurred(serverErrorMessage(reply, reply->errorString()));
        else
            emit chatActionCompleted();
        reply->deleteLater();
    });
}

void OtterLinkClient::addBuddy(const QString &username)
{
    const QString trimmed = username.trimmed();
    if (trimmed.isEmpty()) {
        emit errorOccurred(QStringLiteral("Enter a username to add."));
        return;
    }

    QJsonObject body{{QStringLiteral("username"), trimmed}};
    auto *reply = m_network.post(request(QStringLiteral("/api/buddies")),
                                 QJsonDocument(body).toJson(QJsonDocument::Compact));
    connect(reply, &QNetworkReply::finished, this, [this, reply]() {
        if (reply->error() != QNetworkReply::NoError) {
            emit errorOccurred(serverErrorMessage(reply, reply->errorString()));
        } else {
            const QJsonDocument doc = QJsonDocument::fromJson(reply->readAll());
            const QJsonObject buddy = doc.object();
            const QString username = buddy.value(QStringLiteral("username")).toString().trimmed();
            emit buddyAdded(username.isEmpty() ? QString() : username);
            emit buddyChanged();
            loadDashboard();
        }
        reply->deleteLater();
    });
}

void OtterLinkClient::removeBuddy(const QString &username)
{
    const QString trimmed = username.trimmed();
    if (trimmed.isEmpty()) {
        emit errorOccurred(QStringLiteral("Select a buddy to remove."));
        return;
    }

    QNetworkRequest req = request(QStringLiteral("/api/buddies?username=")
                                  + QString::fromUtf8(QUrl::toPercentEncoding(trimmed)));
    auto *reply = m_network.deleteResource(req);
    connect(reply, &QNetworkReply::finished, this, [this, reply]() {
        if (reply->error() != QNetworkReply::NoError) {
            emit errorOccurred(serverErrorMessage(reply, reply->errorString()));
        } else {
            emit buddyChanged();
            loadDashboard();
        }
        reply->deleteLater();
    });
}

void OtterLinkClient::logout()
{
    if (m_token.isEmpty()) {
        m_username.clear();
        m_joinedChatChannels.clear();
        emit loggedOut();
        return;
    }

    auto *reply = m_network.post(request(QStringLiteral("/api/auth/logout")), QByteArray());
    connect(reply, &QNetworkReply::finished, this, [this, reply]() {
        m_token.clear();
        m_username.clear();
        m_joinedChatChannels.clear();
        emit loggedOut();
        reply->deleteLater();
    });
}

void OtterLinkClient::loadDirectConversation(const QString &username)
{
    const QString encoded = QString::fromUtf8(QUrl::toPercentEncoding(username.trimmed()));
    auto *reply = m_network.get(request(QStringLiteral("/api/messages?with=") + encoded));
    connect(reply, &QNetworkReply::finished, this, [this, reply]() {
        if (reply->error() != QNetworkReply::NoError)
            emit errorOccurred(serverErrorMessage(reply, reply->errorString()));
        else
            emit directConversationLoaded(QJsonDocument::fromJson(reply->readAll()).object());
        reply->deleteLater();
    });
}

void OtterLinkClient::sendDirectMessage(const QString &username, const QString &message)
{
    QJsonObject body{{QStringLiteral("username"), username}, {QStringLiteral("message"), message}};
    auto *reply = m_network.post(request(QStringLiteral("/api/messages")),
                                 QJsonDocument(body).toJson(QJsonDocument::Compact));
    connect(reply, &QNetworkReply::finished, this, [this, reply]() {
        if (reply->error() != QNetworkReply::NoError)
            emit errorOccurred(serverErrorMessage(reply, reply->errorString()));
        else
            emit directMessageSent(QJsonDocument::fromJson(reply->readAll()).object());
        reply->deleteLater();
    });
}

void OtterLinkClient::markDirectMessagesRead(const QString &username)
{
    QJsonObject body{{QStringLiteral("username"), username}};
    auto *reply = m_network.post(request(QStringLiteral("/api/messages/read")),
                                 QJsonDocument(body).toJson(QJsonDocument::Compact));
    connect(reply, &QNetworkReply::finished, this, [this, reply]() {
        if (reply->error() != QNetworkReply::NoError)
            emit errorOccurred(serverErrorMessage(reply, reply->errorString()));
        reply->deleteLater();
    });
}

void OtterLinkClient::setAway(bool away)
{
    QJsonObject body{{QStringLiteral("away"), away}};
    auto *reply = m_network.post(request(QStringLiteral("/api/presence/away")),
                                 QJsonDocument(body).toJson(QJsonDocument::Compact));
    connect(reply, &QNetworkReply::finished, this, [this, reply, away]() {
        if (reply->error() != QNetworkReply::NoError)
            emit errorOccurred(serverErrorMessage(reply, reply->errorString()));
        else
            emit awayChanged(away);
        reply->deleteLater();
    });
}
