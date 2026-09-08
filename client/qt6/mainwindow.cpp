#include "mainwindow.h"
#include "otterlinkclient.h"
#include "ui_mainwindow.h"

#include <QMessageBox>

MainWindow::MainWindow(QWidget *parent)
    : QMainWindow(parent),
      ui(new Ui::MainWindow),
      m_client(new OtterLinkClient(this))
{
    ui->setupUi(this);
    resize(900, 600);
    setLoggedIn(false);

    connect(ui->loginButton, &QPushButton::clicked, this, &MainWindow::login);
    connect(ui->passwordEdit, &QLineEdit::returnPressed, this, &MainWindow::login);
    connect(ui->logoutButton, &QPushButton::clicked, this, &MainWindow::logout);

    connect(m_client, &OtterLinkClient::loggedIn, this, &MainWindow::showDashboard);
    connect(m_client, &OtterLinkClient::dashboardLoaded, this, &MainWindow::dashboardLoaded);
    connect(m_client, &OtterLinkClient::loggedOut, this, [this]() { setLoggedIn(false); });
    connect(m_client, &OtterLinkClient::errorOccurred, this, &MainWindow::showError);
}

MainWindow::~MainWindow()
{
    delete ui;
}

void MainWindow::login()
{
    m_client->setBaseUrl(ui->serverEdit->text());
    m_client->login(ui->usernameEdit->text(), ui->passwordEdit->text());
}

void MainWindow::logout()
{
    m_client->logout();
}

void MainWindow::showDashboard(const QString &displayName)
{
    ui->identityLabel->setText(
        QStringLiteral("Connected as <b>%1</b>").arg(displayName.toHtmlEscaped()));
    setLoggedIn(true);
    m_client->loadDashboard();
}

void MainWindow::dashboardLoaded(const QStringList &buddies, const QStringList &onlineUsers,
                                 const QStringList &chatMessages)
{
    ui->buddiesList->clear();
    ui->buddiesList->addItems(buddies);
    ui->onlineList->clear();
    ui->onlineList->addItems(onlineUsers);
    ui->chatList->clear();
    ui->chatList->addItems(chatMessages);
}

void MainWindow::showError(const QString &message)
{
    QMessageBox::warning(this, QStringLiteral("Otter Link"), message);
}

void MainWindow::setLoggedIn(bool loggedIn)
{
    ui->stackedWidget->setCurrentWidget(loggedIn ? ui->dashboardPage : ui->authPage);
}
