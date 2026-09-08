#include "mainwindow.h"
#include "otterlinkclient.h"

#include <QFormLayout>
#include <QGroupBox>
#include <QHBoxLayout>
#include <QInputDialog>
#include <QLabel>
#include <QLineEdit>
#include <QListWidget>
#include <QMessageBox>
#include <QPushButton>
#include <QSplitter>
#include <QStackedWidget>
#include <QVBoxLayout>

MainWindow::MainWindow(QWidget *parent)
    : QMainWindow(parent), m_client(new OtterLinkClient(this))
{
    setWindowTitle(QStringLiteral("Otter Link"));
    resize(900, 600);
    buildUi();
    setLoggedIn(false);

    connect(m_client, &OtterLinkClient::loggedIn, this, &MainWindow::showDashboard);
    connect(m_client, &OtterLinkClient::dashboardLoaded, this, &MainWindow::dashboardLoaded);
    connect(m_client, &OtterLinkClient::loggedOut, this, [this]() { setLoggedIn(false); });
    connect(m_client, &OtterLinkClient::errorOccurred, this, &MainWindow::showError);
}

void MainWindow::buildUi()
{
    auto *stack = new QStackedWidget(this);
    setCentralWidget(stack);

    m_authPage = new QWidget;
    auto *authLayout = new QVBoxLayout(m_authPage);
    auto *title = new QLabel(QStringLiteral("<h1>Otter Link</h1><p>Connected community service for modern and retro computers.</p>"));
    authLayout->addWidget(title);

    auto *form = new QFormLayout;
    m_serverEdit = new QLineEdit(QStringLiteral("http://127.0.0.1:9090"));
    m_usernameEdit = new QLineEdit;
    m_passwordEdit = new QLineEdit;
    m_passwordEdit->setEchoMode(QLineEdit::Password);
    form->addRow(QStringLiteral("Server:"), m_serverEdit);
    form->addRow(QStringLiteral("Username:"), m_usernameEdit);
    form->addRow(QStringLiteral("Password:"), m_passwordEdit);
    authLayout->addLayout(form);
    auto *loginButton = new QPushButton(QStringLiteral("Connect"));
    authLayout->addWidget(loginButton);
    authLayout->addStretch();
    connect(loginButton, &QPushButton::clicked, this, &MainWindow::login);
    connect(m_passwordEdit, &QLineEdit::returnPressed, this, &MainWindow::login);
    stack->addWidget(m_authPage);

    m_dashboardPage = new QWidget;
    auto *dashboardLayout = new QVBoxLayout(m_dashboardPage);
    auto *top = new QHBoxLayout;
    m_identityLabel = new QLabel;
    auto *logoutButton = new QPushButton(QStringLiteral("Disconnect"));
    top->addWidget(m_identityLabel);
    top->addStretch();
    top->addWidget(logoutButton);
    dashboardLayout->addLayout(top);

    auto *splitter = new QSplitter(Qt::Horizontal);
    auto *community = new QGroupBox(QStringLiteral("Online"));
    auto *communityLayout = new QVBoxLayout(community);
    m_online = new QListWidget;
    communityLayout->addWidget(m_online);
    splitter->addWidget(community);

    auto *buddies = new QGroupBox(QStringLiteral("Buddies"));
    auto *buddyLayout = new QVBoxLayout(buddies);
    m_buddies = new QListWidget;
    buddyLayout->addWidget(m_buddies);
    splitter->addWidget(buddies);

    auto *chat = new QGroupBox(QStringLiteral("Community Chat"));
    auto *chatLayout = new QVBoxLayout(chat);
    m_chat = new QListWidget;
    chatLayout->addWidget(m_chat);
    splitter->addWidget(chat);

    dashboardLayout->addWidget(splitter, 1);
    stack->addWidget(m_dashboardPage);

    connect(logoutButton, &QPushButton::clicked, this, &MainWindow::logout);
}

void MainWindow::login()
{
    m_client->setBaseUrl(m_serverEdit->text());
    m_client->login(m_usernameEdit->text(), m_passwordEdit->text());
}

void MainWindow::logout()
{
    m_client->logout();
}

void MainWindow::showDashboard(const QString &displayName)
{
    m_identityLabel->setText(QStringLiteral("Connected as <b>%1</b>").arg(displayName.toHtmlEscaped()));
    setLoggedIn(true);
    m_client->loadDashboard();
}

void MainWindow::dashboardLoaded(const QStringList &buddies, const QStringList &onlineUsers,
                                 const QStringList &chatMessages)
{
    m_buddies->clear();
    m_buddies->addItems(buddies);
    m_online->clear();
    m_online->addItems(onlineUsers);
    m_chat->clear();
    m_chat->addItems(chatMessages);
}

void MainWindow::showError(const QString &message)
{
    QMessageBox::warning(this, QStringLiteral("Otter Link"), message);
}

void MainWindow::setLoggedIn(bool loggedIn)
{
    centralWidget()->findChild<QStackedWidget *>()->setCurrentWidget(loggedIn ? m_dashboardPage : m_authPage);
}
