#include "mainwindow.h"
#include "otterlinkclient.h"
#include "ui_mainwindow.h"

#include <QComboBox>
#include <QDialog>
#include <QDialogButtonBox>
#include <QFont>
#include <QFormLayout>
#include <QLineEdit>
#include <QMessageBox>
#include <QPushButton>
#include <QStyle>
#include <QTreeWidget>
#include <QTreeWidgetItem>

#include <initializer_list>

namespace {

void setActiveServiceButton(QPushButton *active,
                            std::initializer_list<QPushButton *> buttons)
{
    for (QPushButton *button : buttons)
        button->setProperty("active", button == active);

    for (QPushButton *button : buttons) {
        button->style()->unpolish(button);
        button->style()->polish(button);
        button->update();
    }
}

const QStringList kBuddyGroups = {
    QStringLiteral("Buddies"),
    QStringLiteral("Family"),
    QStringLiteral("Co-worker")
};

} // namespace

MainWindow::MainWindow(QWidget *parent)
    : QMainWindow(parent),
      ui(new Ui::MainWindow),
      m_client(new OtterLinkClient(this))
{
    ui->setupUi(this);
    resize(1000, 680);
    m_refreshTimer.setInterval(5000);
    m_connectionTimer.setInterval(700);

    // Replace the simple Designer placeholder with the hierarchical People view.
    m_buddyTree = new QTreeWidget(ui->buddiesGroup);
    m_buddyTree->setObjectName(QStringLiteral("buddyTree"));
    m_buddyTree->setHeaderHidden(true);
    m_buddyTree->setRootIsDecorated(true);
    m_buddyTree->setItemsExpandable(true);
    m_buddyTree->setUniformRowHeights(true);
    m_buddyTree->setSelectionMode(QAbstractItemView::SingleSelection);
    m_buddyTree->setMinimumHeight(180);
    ui->buddiesLayout->replaceWidget(ui->buddiesList, m_buddyTree);
    ui->buddiesList->hide();
    ui->buddiesList->deleteLater();

    setLoggedIn(false);

    connect(ui->loginButton, &QPushButton::clicked, this, &MainWindow::login);
    connect(ui->passwordEdit, &QLineEdit::returnPressed, this, &MainWindow::login);
    connect(ui->logoutButton, &QPushButton::clicked, this, &MainWindow::logout);
    connect(ui->sendChatButton, &QPushButton::clicked, this, &MainWindow::sendChat);
    connect(ui->chatEdit, &QLineEdit::returnPressed, this, &MainWindow::sendChat);
    connect(ui->refreshButton, &QPushButton::clicked, this, &MainWindow::refreshDashboard);
    connect(ui->addBuddyButton, &QPushButton::clicked, this, &MainWindow::addBuddy);
    connect(ui->removeBuddyButton, &QPushButton::clicked, this, &MainWindow::removeBuddy);
    connect(ui->homeButton, &QPushButton::clicked, this, &MainWindow::navigateService);
    connect(ui->peopleButton, &QPushButton::clicked, this, &MainWindow::navigateService);
    connect(ui->mailButton, &QPushButton::clicked, this, &MainWindow::navigateService);
    connect(ui->chatButton, &QPushButton::clicked, this, &MainWindow::navigateService);
    connect(ui->boardsButton, &QPushButton::clicked, this, &MainWindow::navigateService);
    connect(ui->newsButton, &QPushButton::clicked, this, &MainWindow::navigateService);
    connect(ui->filesButton, &QPushButton::clicked, this, &MainWindow::navigateService);
    connect(ui->gamesButton, &QPushButton::clicked, this, &MainWindow::navigateService);
    connect(&m_refreshTimer, &QTimer::timeout, this, &MainWindow::refreshDashboard);
    connect(&m_connectionTimer, &QTimer::timeout, this, &MainWindow::advanceConnectionStage);
    connect(&m_connectionFinishTimer, &QTimer::timeout, this, &MainWindow::finishConnectionPresentation);
    connect(m_buddyTree, &QTreeWidget::itemSelectionChanged,
            this, &MainWindow::buddySelectionChanged);

    connect(m_client, &OtterLinkClient::loggedIn, this, &MainWindow::showDashboard);
    connect(m_client, &OtterLinkClient::dashboardLoaded, this, &MainWindow::dashboardLoaded);
    connect(m_client, &OtterLinkClient::buddyAdded, this, &MainWindow::buddyAdded);
    connect(m_client, &OtterLinkClient::loggedOut, this, [this]() {
        m_refreshTimer.stop();
        m_connectionTimer.stop();
        m_connectionFinishTimer.stop();
        m_pendingBuddyGroups.clear();
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
    QDialog dialog(this);
    dialog.setWindowTitle(QStringLiteral("Add Buddy"));

    auto *layout = new QFormLayout(&dialog);
    auto *usernameEdit = new QLineEdit(&dialog);
    auto *groupCombo = new QComboBox(&dialog);
    auto *buttons = new QDialogButtonBox(QDialogButtonBox::Ok | QDialogButtonBox::Cancel,
                                         Qt::Horizontal, &dialog);

    groupCombo->addItems(kBuddyGroups);
    groupCombo->setCurrentText(QStringLiteral("Buddies"));
    usernameEdit->setPlaceholderText(QStringLiteral("Enter a username"));
    layout->addRow(QStringLiteral("Username:"), usernameEdit);
    layout->addRow(QStringLiteral("Group:"), groupCombo);
    layout->addRow(buttons);

    connect(buttons, &QDialogButtonBox::accepted, &dialog, &QDialog::accept);
    connect(buttons, &QDialogButtonBox::rejected, &dialog, &QDialog::reject);
    connect(usernameEdit, &QLineEdit::returnPressed, &dialog, &QDialog::accept);

    usernameEdit->setFocus();
    if (dialog.exec() != QDialog::Accepted)
        return;

    const QString trimmed = usernameEdit->text().trimmed();
    if (trimmed.isEmpty()) {
        showError(QStringLiteral("Enter a username to add."));
        return;
    }

    m_pendingBuddyGroups.insert(trimmed, groupCombo->currentText());
    m_client->addBuddy(trimmed);
}

void MainWindow::buddyAdded(const QString &username)
{
    const QString group = m_pendingBuddyGroups.take(username);
    if (!group.isEmpty())
        m_buddyGroups.insert(username, group);
}

void MainWindow::removeBuddy()
{
    const auto selected = m_buddyTree->selectedItems();
    if (selected.isEmpty() || selected.first()->parent() == nullptr) {
        showError(QStringLiteral("Select a buddy to remove."));
        return;
    }

    const QTreeWidgetItem *item = selected.first();
    const QString username = item->data(0, Qt::UserRole).toString();
    if (username.isEmpty()) {
        showError(QStringLiteral("The selected buddy does not have a username."));
        return;
    }

    if (QMessageBox::question(this, QStringLiteral("Remove Buddy"),
                              QStringLiteral("Remove %1 from your buddy list?").arg(username))
        == QMessageBox::Yes) {
        m_buddyGroups.remove(username);
        m_client->removeBuddy(username);
    }
}

void MainWindow::navigateService()
{
    auto *button = qobject_cast<QPushButton *>(sender());
    if (!button)
        return;

    setActiveServiceButton(button, {
        ui->homeButton, ui->peopleButton, ui->mailButton, ui->chatButton,
        ui->boardsButton, ui->newsButton, ui->filesButton, ui->gamesButton
    });

    if (button == ui->homeButton) {
        ui->serviceStack->setCurrentWidget(ui->homePage);
        ui->serviceTitleLabel->setText(QStringLiteral("Welcome to Otter Link"));
    } else if (button == ui->peopleButton) {
        ui->serviceStack->setCurrentWidget(ui->peoplePage);
        ui->serviceTitleLabel->setText(QStringLiteral("People"));
    } else if (button == ui->chatButton) {
        ui->serviceStack->setCurrentWidget(ui->chatPage);
        ui->serviceTitleLabel->setText(QStringLiteral("Community Chat"));
    } else {
        ui->serviceStack->setCurrentWidget(ui->placeholderPage);
        ui->placeholderTitleLabel->setText(button->text());
        ui->serviceTitleLabel->setText(button->text());
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
    ui->serviceStack->setCurrentWidget(ui->homePage);
    ui->serviceTitleLabel->setText(QStringLiteral("Welcome to Otter Link"));
    setActiveServiceButton(ui->homeButton, {
        ui->homeButton, ui->peopleButton, ui->mailButton, ui->chatButton,
        ui->boardsButton, ui->newsButton, ui->filesButton, ui->gamesButton
    });
    setLoggedIn(true);
    m_client->loadDashboard();
    m_refreshTimer.start();
}

void MainWindow::dashboardLoaded(const QStringList &buddies, const QStringList &onlineUsers,
                                 const QStringList &chatMessages)
{
    rebuildBuddyTree(buddies, onlineUsers);

    ui->onlineList->clear();
    for (const QString &user : onlineUsers)
        ui->onlineList->addItem(QStringLiteral("● %1").arg(user));

    ui->chatList->clear();
    ui->chatList->addItems(chatMessages);

    ui->homeBuddiesLabel->setText(
        QStringLiteral("%1 %2 in your buddy list")
            .arg(buddies.size())
            .arg(buddies.size() == 1 ? QStringLiteral("buddy") : QStringLiteral("buddies")));
    ui->homeOnlineLabel->setText(
        QStringLiteral("%1 %2 currently online")
            .arg(onlineUsers.size())
            .arg(onlineUsers.size() == 1 ? QStringLiteral("user") : QStringLiteral("users")));
}

void MainWindow::rebuildBuddyTree(const QStringList &buddies, const QStringList &onlineUsers)
{
    m_buddyTree->clear();

    QHash<QString, QTreeWidgetItem *> groupItems;
    for (const QString &group : kBuddyGroups) {
        auto *item = new QTreeWidgetItem(m_buddyTree);
        item->setText(0, group);
        item->setExpanded(true);
        item->setFont(0, QFont(QStringLiteral("Sans Serif"), -1, QFont::Bold));
        groupItems.insert(group, item);
    }

    auto *offlineGroup = new QTreeWidgetItem(m_buddyTree);
    offlineGroup->setText(0, QStringLiteral("Offline"));
    offlineGroup->setExpanded(true);
    offlineGroup->setFont(0, QFont(QStringLiteral("Sans Serif"), -1, QFont::Bold));

    const QPalette palette = m_buddyTree->palette();
    const QBrush offlineBrush(palette.color(QPalette::Disabled, QPalette::Text));

    for (const QString &buddy : buddies) {
        const bool online = onlineUsers.contains(buddy, Qt::CaseInsensitive);
        QString group = m_buddyGroups.value(buddy, QStringLiteral("Buddies"));
        if (!groupItems.contains(group))
            group = QStringLiteral("Buddies");

        QTreeWidgetItem *parent = online ? groupItems.value(group) : offlineGroup;
        auto *item = new QTreeWidgetItem(parent);
        item->setText(0, QStringLiteral("%1 %2")
                             .arg(online ? QStringLiteral("●") : QStringLiteral("○"), buddy));
        item->setData(0, Qt::UserRole, buddy);

        if (!online) {
            item->setForeground(0, offlineBrush);
            QFont font = item->font(0);
            font.setItalic(true);
            item->setFont(0, font);
        }
    }

    offlineGroup->setHidden(offlineGroup->childCount() == 0);
    for (QTreeWidgetItem *group : groupItems)
        group->setHidden(group->childCount() == 0);

    m_buddyTree->resizeColumnToContents(0);
    m_buddyTree->expandAll();
    buddySelectionChanged();
}

void MainWindow::buddySelectionChanged()
{
    const auto selected = m_buddyTree->selectedItems();
    if (selected.isEmpty() || selected.first()->parent() == nullptr) {
        ui->peopleInfoLabel->setText(
            QStringLiteral("<h3>People</h3><p>Select a buddy to see their current status.</p>"));
        return;
    }

    const QTreeWidgetItem *item = selected.first();
    const QString username = item->data(0, Qt::UserRole).toString();
    const bool online = item->parent()->text(0) != QStringLiteral("Offline");
    ui->peopleInfoLabel->setText(
        QStringLiteral("<h3>%1</h3><p>%2</p>")
            .arg(username.toHtmlEscaped())
            .arg(online ? QStringLiteral("Online") : QStringLiteral("Offline")));
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
