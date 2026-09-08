#include "mainwindow.h"
#include "otterlinkclient.h"
#include "ui_mainwindow.h"

#include <QInputDialog>
#include <QMessageBox>

MainWindow::MainWindow(QWidget *parent)
    : QMainWindow(parent),
      ui(new Ui::MainWindow),
      m_client(new OtterLinkClient(this))
{
    ui->setupUi(this);
    resize(900, 600);
    m_refreshTimer.setInterval(5000);
    m_connectionTimer.setInterval(700);
    setLoggedIn(false);

    connect(ui->loginButton, &QPushButton::clicked, this, &MainWindow::login);
    connect(ui->passwordEdit, &QLineEdit::returnPressed, this, &MainWindow::login);
    connect(ui->logoutButton, &QPushButton::clicked, this, &MainWindow::logout);
    connect(ui->sendChatButton, &QPushButton::clicked, this, &MainWindow::sendChat);
    connect(ui->chatEdit, &QLineEdit::returnPressed, this, &MainWindow::sendChat);
    connect(ui->refreshButton, &QPushButton::clicked, this, &MainWindow::refreshDashboard);
    connect(ui->addBuddyButton, &QPushButton::clicked, this, &MainWindow::addBuddy);
    connect(ui->removeBuddyButton, &QPushButton::clicked, this, &MainWindow::removeBuddy);
    connect(&m_refreshTimer, &QTimer::timeout, this, &MainWindow::refreshDashboard);
    connect(&m_connectionTimer, &QTimer::timeout, this, &MainWindow::advanceConnectionStage);
    connect(&m_connectionFinishTimer, &QTimer::timeout, this, &MainWindow::finishConnectionPresentation);

    connect(m_client, &OtterLinkClient::loggedIn, this, &MainWindow::showDashboard);
    connect(m_client, &OtterLinkClient::dashboardLoaded, this, &MainWindow::dashboardLoaded);
    connect(m_client, &OtterLinkClient::loggedOut, this, [this]() {
        m_refreshTimer.stop();
        m_connectionTimer.stop();
        m_connectionFinishTimer.stop();
        setLoggedIn(false);
    });
    connect(m_client, &OtterLinkClient::errorOccurred, this, &MainWindow::showError);
}

MainWindow::~MainWindow()
{
    delete ui;
}

void MainWindow::login()
{
    m_client->setBaseUrl(ui->serverEdit->text());
    beginConnectionPresentation();
    m_client->login(ui->usernameEdit->text(), ui->passwordEdit->text());
}

void MainWindow::logout()
{
    m_refreshTimer.stop();
    m_client->logout();
}

void MainWindow::sendChat()
{
    const QString message = ui->chatEdit->text().trimmed();
    if (message.isEmpty())
        return;

    ui->chatEdit->clear();
    m_client->sendChatMessage(message);
}

void MainWindow::refreshDashboard()
{
    m_client->loadDashboard();
}

void MainWindow::addBuddy()
{
    bool accepted = false;
    const QString username = QInputDialog::getText(this, QStringLiteral("Add Buddy"),
                                                    QStringLiteral("Username:"),
                                                    QLineEdit::Normal, QString(), &accepted);
    if (accepted && !username.trimmed().isEmpty())
        m_client->addBuddy(username);
}

void MainWindow::removeBuddy()
{
    const QListWidgetItem *item = ui->buddiesList->currentItem();
    if (!item) {
        showError(QStringLiteral("Select a buddy to remove."));
        return;
    }

    const QString username = item->data(Qt::UserRole).toString();
    if (username.isEmpty()) {
        showError(QStringLiteral("The selected buddy does not have a username."));
        return;
    }

    if (QMessageBox::question(this, QStringLiteral("Remove Buddy"),
                              QStringLiteral("Remove %1 from your buddy list?").arg(item->text()))
        == QMessageBox::Yes) {
        m_client->removeBuddy(username);
    }
}

void MainWindow::beginConnectionPresentation()
{
    m_connectionReady = false;
    m_connectionDisplayName.clear();
    m_connectionStage = 0;
    ui->connectionStageLabel->setText(QStringLiteral("CALLING"));
    ui->connectionDetailLabel->setText(QStringLiteral("Dialing Otter Link..."));
    ui->connectionProgress->setValue(10);
    ui->connectionOtterLabel->setText(QStringLiteral("( o.o )\n /|\\\n  / \\\n\n~ ~ ~"));
    ui->stackedWidget->setCurrentWidget(ui->connectionPage);
    m_connectionTimer.start();
}

void MainWindow::advanceConnectionStage()
{
    ++m_connectionStage;

    switch (m_connectionStage) {
    case 1:
        ui->connectionStageLabel->setText(QStringLiteral("CONNECTING"));
        ui->connectionDetailLabel->setText(QStringLiteral("Establishing carrier..."));
        ui->connectionProgress->setValue(55);
        ui->connectionOtterLabel->setText(QStringLiteral("( o.o )\n /|\\\n  / \\\n\n~ ~ ~"));
        break;
    case 2:
        ui->connectionStageLabel->setText(QStringLiteral("CONNECTED"));
        ui->connectionDetailLabel->setText(QStringLiteral("Welcome to Otter Link."));
        ui->connectionProgress->setValue(100);
        ui->connectionOtterLabel->setText(QStringLiteral("  /\\_/\\\n ( o.o )\n  > ^ <"));
        m_connectionTimer.stop();
        if (m_connectionReady)
            m_connectionFinishTimer.start(450);
        break;
    default:
        m_connectionTimer.stop();
        break;
    }
}

void MainWindow::finishConnectionPresentation()
{
    m_connectionFinishTimer.stop();
    if (m_connectionReady)
        showDashboard(m_connectionDisplayName);
}

void MainWindow::showDashboard(const QString &displayName)
{
    if (ui->stackedWidget->currentWidget() == ui->connectionPage && m_connectionStage < 2) {
        m_connectionReady = true;
        m_connectionDisplayName = displayName;
        return;
    }

    m_connectionReady = false;
    m_connectionDisplayName.clear();
    m_connectionTimer.stop();
    m_connectionFinishTimer.stop();
    ui->identityLabel->setText(
        QStringLiteral("Connected as <b>%1</b>").arg(displayName.toHtmlEscaped()));
    setLoggedIn(true);
    m_client->loadDashboard();
    m_refreshTimer.start();
}

void MainWindow::dashboardLoaded(const QStringList &buddies, const QStringList &onlineUsers,
                                 const QStringList &chatMessages)
{
    ui->buddiesList->clear();
    for (const QString &buddy : buddies) {
        auto *item = new QListWidgetItem(buddy, ui->buddiesList);
        item->setData(Qt::UserRole, buddy);
    }

    ui->onlineList->clear();
    ui->onlineList->addItems(onlineUsers);
    ui->chatList->clear();
    ui->chatList->addItems(chatMessages);
}

void MainWindow::showError(const QString &message)
{
    m_connectionReady = false;
    m_connectionDisplayName.clear();
    m_connectionTimer.stop();
    m_connectionFinishTimer.stop();
    setLoggedIn(false);
    QMessageBox::warning(this, QStringLiteral("Otter Link"), message);
}

void MainWindow::setLoggedIn(bool loggedIn)
{
    ui->stackedWidget->setCurrentWidget(loggedIn ? ui->dashboardPage : ui->authPage);
}
