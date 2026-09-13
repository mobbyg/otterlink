#include "mainwindow.h"
#include "otterlinkclient.h"
#include "otterservicewindow.h"
#include "ui_mainwindow.h"

#include <QComboBox>
#include <QDialog>
#include <QDialogButtonBox>
#include <QFont>
#include <QFormLayout>
#include <QHeaderView>
#include <QLabel>
#include <QLineEdit>
#include <QMessageBox>
#include <QPushButton>
#include <QSignalBlocker>
#include <QStyle>
#include <QTreeWidget>
#include <QTreeWidgetItem>
#include <QVBoxLayout>

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

    // The old service stack remains the source for the existing service widgets.
    // The new desktop simply presents those widgets as movable internal windows.
    ui->serviceRail->hide();
    ui->serviceTitleLabel->hide();
    ui->serviceStack->hide();

    m_desktop = new QFrame(ui->contentLayout->parentWidget());
    m_desktop->setObjectName(QStringLiteral("otterDesktop"));
    m_desktop->setFrameShape(QFrame::StyledPanel);
    m_desktop->setFrameShadow(QFrame::Sunken);
    m_desktop->setMinimumSize(520, 320);
    ui->contentLayout->addWidget(m_desktop, 1);

    // Replace the simple Designer placeholder with the hierarchical People view.
    m_buddyTree = new QTreeWidget(ui->buddiesGroup);
    m_buddyTree->setObjectName(QStringLiteral("buddyTree"));
    m_buddyTree->setHeaderHidden(true);
    m_buddyTree->setRootIsDecorated(true);
    m_buddyTree->setItemsExpandable(true);
    m_buddyTree->setUniformRowHeights(true);
    m_buddyTree->setSelectionMode(QAbstractItemView::SingleSelection);
    m_buddyTree->setMinimumHeight(180);
    m_buddyTree->header()->setSectionResizeMode(0, QHeaderView::Stretch);
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
        closeAllServiceWindows();
        setLoggedIn(false);
    });
    connect(m_client, &OtterLinkClient::errorOccurred, this, &MainWindow::showError);
}

MainWindow::~MainWindow()
{
    closeAllServiceWindows();
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
    if (username.isEmpty())
        return;

    QString pendingKey;
    for (auto it = m_pendingBuddyGroups.cbegin(); it != m_pendingBuddyGroups.cend(); ++it) {
        if (it.key().compare(username, Qt::CaseInsensitive) == 0) {
            pendingKey = it.key();
            break;
        }
    }

    if (!pendingKey.isEmpty())
        m_buddyGroups.insert(username, m_pendingBuddyGroups.take(pendingKey));
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

    if (button == ui->homeButton) {
        openServiceWindow(QStringLiteral("home"), QStringLiteral("Welcome to Otter Link"),
                          ui->homePage);
    } else if (button == ui->peopleButton) {
        openServiceWindow(QStringLiteral("people"), QStringLiteral("People"), ui->peoplePage);
    } else if (button == ui->chatButton) {
        openServiceWindow(QStringLiteral("chat"), QStringLiteral("Community Chat"), ui->chatPage);
    } else {
        const QString title = button->text();
        auto *page = new QWidget(m_desktop);
        auto *layout = new QVBoxLayout(page);
        layout->setContentsMargins(18, 18, 18, 18);
        layout->addStretch(1);

        auto *titleLabel = new QLabel(title, page);
        titleLabel->setObjectName(QStringLiteral("placeholderTitle"));
        titleLabel->setAlignment(Qt::AlignCenter);
        titleLabel->setProperty("placeholder", true);
        layout->addWidget(titleLabel);

        auto *infoLabel = new QLabel(
            QStringLiteral("This Otter Link service is planned for a future release."), page);
        infoLabel->setAlignment(Qt::AlignCenter);
        infoLabel->setWordWrap(true);
        layout->addWidget(infoLabel);
        layout->addStretch(1);

        openServiceWindow(title.toLower(), title, page);
    }
}

void MainWindow::openServiceWindow(const QString &service, const QString &title, QWidget *content)
{
    auto existing = m_serviceWindows.value(service, nullptr);
    if (existing) {
        existing->activateWindow();
        updateServiceButtonStates(existing);
        return;
    }

    if (content == ui->homePage || content == ui->peoplePage || content == ui->chatPage)
        ui->serviceStack->removeWidget(content);

    auto *window = new OtterServiceWindow(title, content, m_desktop);
    m_serviceWindows.insert(service, window);
    m_windowServices.insert(window, service);

    const int offset = m_nextWindowOffset;
    m_nextWindowOffset = (m_nextWindowOffset + 28) % 140;
    const int width = qMin(620, qMax(360, m_desktop->width() - 70));
    const int height = qMin(440, qMax(250, m_desktop->height() - 70));
    window->resize(width, height);
    window->move(24 + offset, 20 + offset);

    connect(window, &OtterServiceWindow::closeRequested,
            this, &MainWindow::closeServiceWindow);

    window->show();
    window->activateWindow();
    updateServiceButtonStates(window);
}

void MainWindow::closeServiceWindow(OtterServiceWindow *window)
{
    if (!window)
        return;

    const QString service = m_windowServices.take(window);
    m_serviceWindows.remove(service);

    if (service == QStringLiteral("home"))
        restoreServicePage(ui->homePage);
    else if (service == QStringLiteral("people"))
        restoreServicePage(ui->peoplePage);
    else if (service == QStringLiteral("chat"))
        restoreServicePage(ui->chatPage);

    window->deleteLater();
    updateServiceButtonStates();
}

void MainWindow::closeAllServiceWindows()
{
    const auto windows = m_serviceWindows.values();
    for (OtterServiceWindow *window : windows)
        closeServiceWindow(window);

    m_serviceWindows.clear();
    m_windowServices.clear();
    m_nextWindowOffset = 0;
}

void MainWindow::restoreServicePage(QWidget *page)
{
    if (!page)
        return;

    page->setParent(ui->serviceStack);
    ui->serviceStack->addWidget(page);
    page->hide();
}

void MainWindow::updateServiceButtonStates(OtterServiceWindow *activeWindow)
{
    QPushButton *activeButton = nullptr;
    const QString activeService = activeWindow ? m_windowServices.value(activeWindow) : QString();

    if (activeService == QStringLiteral("home"))
        activeButton = ui->homeButton;
    else if (activeService == QStringLiteral("people"))
        activeButton = ui->peopleButton;
    else if (activeService == QStringLiteral("chat"))
        activeButton = ui->chatButton;

    setActiveServiceButton(activeButton, {
        ui->homeButton, ui->peopleButton, ui->mailButton, ui->chatButton,
        ui->boardsButton, ui->newsButton, ui->filesButton, ui->gamesButton
    });
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
    setLoggedIn(true);
    m_client->loadDashboard();
    m_refreshTimer.start();

    // Start with the desktop itself as the home state. Home remains available from the bar.
    updateServiceButtonStates();
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
    QString selectedUsername;
    if (const QTreeWidgetItem *selected = m_buddyTree->currentItem())
        selectedUsername = selected->data(0, Qt::UserRole).toString();

    QSignalBlocker blocker(m_buddyTree);
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

    m_buddyTree->expandAll();

    if (!selectedUsername.isEmpty()) {
        const auto matches = m_buddyTree->findItems(
            QStringLiteral("*%1").arg(selectedUsername), Qt::MatchWildcard | Qt::MatchRecursive);
        for (QTreeWidgetItem *item : matches) {
            if (item->data(0, Qt::UserRole).toString().compare(selectedUsername, Qt::CaseInsensitive) == 0) {
                m_buddyTree->setCurrentItem(item);
                break;
            }
        }
    }

    blocker.unblock();
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
    const bool wasConnecting = ui->stackedWidget->currentWidget() == ui->connectionPage;
    if (wasConnecting) {
        m_connectionReady = false;
        m_connectionDisplayName.clear();
        m_connectionTimer.stop();
        m_connectionFinishTimer.stop();
        setLoggedIn(false);
    }
    QMessageBox::warning(this, QStringLiteral("Otter Link"), message);
}

void MainWindow::setLoggedIn(bool loggedIn)
{
    ui->stackedWidget->setCurrentWidget(loggedIn ? ui->dashboardPage : ui->authPage);
}
