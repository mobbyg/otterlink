#include "otteridentitydisplay.h"

#include "mainwindow.h"
#include "otterlinkclient.h"

#include <QHBoxLayout>
#include <QLabel>
#include <QPushButton>
#include <QSizePolicy>
#include <QWidget>

namespace {

void installIdentityHeader(MainWindow &window)
{
    auto *headerLayout = window.findChild<QHBoxLayout *>(QStringLiteral("headerLayout"));
    auto *identityLabel = window.findChild<QLabel *>(QStringLiteral("identityLabel"));
    auto *refreshButton = window.findChild<QPushButton *>(QStringLiteral("refreshButton"));
    auto *logoutButton = window.findChild<QPushButton *>(QStringLiteral("logoutButton"));
    if (!headerLayout || !identityLabel || !refreshButton || !logoutButton)
        return;

    if (window.findChild<QWidget *>(QStringLiteral("identityHeaderControls")))
        return;

    headerLayout->removeWidget(identityLabel);
    headerLayout->removeWidget(refreshButton);
    headerLayout->removeWidget(logoutButton);

    auto *controls = new QWidget(identityLabel->parentWidget());
    controls->setObjectName(QStringLiteral("identityHeaderControls"));
    controls->setSizePolicy(QSizePolicy::Minimum, QSizePolicy::Fixed);

    auto *controlsLayout = new QHBoxLayout(controls);
    controlsLayout->setContentsMargins(0, 0, 0, 0);
    controlsLayout->setSpacing(6);
    controlsLayout->addWidget(identityLabel);
    controlsLayout->addWidget(refreshButton);
    controlsLayout->addWidget(logoutButton);

    headerLayout->addWidget(controls, 0, Qt::AlignVCenter);
}

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
    installIdentityHeader(window);

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
