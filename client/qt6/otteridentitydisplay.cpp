#include "otteridentitydisplay.h"

#include "mainwindow.h"
#include "otterlinkclient.h"

#include <QLabel>
#include <QSizePolicy>

namespace {

void updateIdentityLabel(MainWindow &window, const QString &accountName)
{
    auto *label = window.findChild<QLabel *>(QStringLiteral("identityLabel"));
    if (!label)
        return;

    const QString username = accountName.trimmed();
    label->setText(username.isEmpty()
                       ? QStringLiteral("Connected")
                       : QStringLiteral("Connected as <b>%1</b>")
                             .arg(username.toHtmlEscaped()));
    label->setWordWrap(false);
    label->setSizePolicy(QSizePolicy::Preferred, QSizePolicy::Fixed);
    label->setVisible(true);
}

} // namespace

namespace OtterIdentityDisplay {

void install(MainWindow &window)
{
    auto *client = window.findChild<OtterLinkClient *>();
    if (!client)
        return;

    QObject::connect(client, &OtterLinkClient::loggedIn, &window,
                     [&window](const QString &displayName) {
                         updateIdentityLabel(window, displayName);
                     });

    QObject::connect(client, &OtterLinkClient::dashboardLoaded, &window,
                     [&window, client](const QStringList &, const QStringList &, const QStringList &) {
                         updateIdentityLabel(window, client->accountName());
                     });
}

} // namespace OtterIdentityDisplay
