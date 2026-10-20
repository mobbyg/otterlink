#include "otterlinkclient.h"

#include <QNetworkReply>
#include <QNetworkRequest>

void OtterLinkClient::deleteBoardPost(qint64 postId)
{
    if (postId < 1)
        return;

    auto *reply = m_network.deleteResource(
        request(QStringLiteral("/api/board-posts/%1").arg(postId)));
    connect(reply, &QNetworkReply::finished, this, [this, reply]() {
        if (reply->error() != QNetworkReply::NoError) {
            QString message = QString::fromUtf8(reply->readAll()).trimmed();
            if (message.isEmpty())
                message = reply->errorString();
            emit errorOccurred(message);
        } else {
            emit boardActionCompleted();
        }
        reply->deleteLater();
    });
}
