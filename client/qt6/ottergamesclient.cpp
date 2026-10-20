#include "otterlinkclient.h"

#include <QJsonDocument>
#include <QJsonObject>
#include <QNetworkReply>
#include <QNetworkRequest>

namespace {
QString gameReplyError(QNetworkReply *reply)
{
    const QString body = QString::fromUtf8(reply->readAll()).trimmed();
    return body.isEmpty() ? reply->errorString() : body;
}
}

void OtterLinkClient::loadGames()
{
    auto *reply = m_network.get(request(QStringLiteral("/api/games")));
    connect(reply, &QNetworkReply::finished, this, [this, reply]() {
        if (reply->error() != QNetworkReply::NoError) {
            emit errorOccurred(gameReplyError(reply));
        } else {
            const QJsonObject result = QJsonDocument::fromJson(reply->readAll()).object();
            emit gamesLoaded(result.value(QStringLiteral("games")).toArray());
        }
        reply->deleteLater();
    });
}

void OtterLinkClient::startGameSession(qint64 gameId)
{
    if (gameId < 1)
        return;
    auto *reply = m_network.post(
        request(QStringLiteral("/api/games/%1/sessions").arg(gameId)),
        QJsonDocument(QJsonObject{}).toJson(QJsonDocument::Compact));
    connect(reply, &QNetworkReply::finished, this, [this, reply]() {
        if (reply->error() != QNetworkReply::NoError) {
            emit errorOccurred(gameReplyError(reply));
        } else {
            const QJsonObject session = QJsonDocument::fromJson(reply->readAll()).object();
            emit gameSessionStarted(session);
        }
        reply->deleteLater();
    });
}

void OtterLinkClient::pollGameSession(const QString &sessionId, qint64 afterSequence)
{
    if (sessionId.trimmed().isEmpty())
        return;
    const QString path = QStringLiteral("/api/game-sessions/%1/output?after=%2")
        .arg(sessionId, QString::number(qMax<qint64>(0, afterSequence)));
    auto *reply = m_network.get(request(path));
    connect(reply, &QNetworkReply::finished, this, [this, reply]() {
        if (reply->error() != QNetworkReply::NoError) {
            emit errorOccurred(gameReplyError(reply));
        } else {
            emit gameOutputLoaded(QJsonDocument::fromJson(reply->readAll()).object());
        }
        reply->deleteLater();
    });
}

void OtterLinkClient::sendGameInput(const QString &sessionId, const QString &text)
{
    if (sessionId.trimmed().isEmpty() || text.trimmed().isEmpty())
        return;
    const QJsonObject body{{QStringLiteral("text"), text}};
    auto *reply = m_network.post(
        request(QStringLiteral("/api/game-sessions/%1/input").arg(sessionId)),
        QJsonDocument(body).toJson(QJsonDocument::Compact));
    connect(reply, &QNetworkReply::finished, this, [this, reply, sessionId]() {
        if (reply->error() != QNetworkReply::NoError) {
            emit errorOccurred(gameReplyError(reply));
        } else {
            emit gameInputSent(sessionId);
            reply->readAll();
        }
        reply->deleteLater();
    });
}

void OtterLinkClient::stopGameSession(const QString &sessionId)
{
    if (sessionId.trimmed().isEmpty())
        return;
    auto *reply = m_network.deleteResource(request(QStringLiteral("/api/game-sessions/%1").arg(sessionId)));
    connect(reply, &QNetworkReply::finished, this, [this, reply, sessionId]() {
        if (reply->error() != QNetworkReply::NoError) {
            emit errorOccurred(gameReplyError(reply));
        } else {
            emit gameSessionStopped(sessionId);
        }
        reply->deleteLater();
    });
}
