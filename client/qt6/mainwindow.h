#pragma once

#include <QHash>
#include <QMainWindow>
#include <QTimer>

class QFrame;
class QTreeWidget;
class OtterServiceWindow;

namespace Ui {
class MainWindow;
}

class OtterLinkClient;

class MainWindow final : public QMainWindow
{
    Q_OBJECT

public:
    explicit MainWindow(QWidget *parent = nullptr);
    ~MainWindow() override;

private slots:
    void login();
    void logout();
    void sendChat();
    void refreshDashboard();
    void addBuddy();
    void removeBuddy();
    void navigateService();
    void advanceConnectionStage();
    void finishConnectionPresentation();
    void showDashboard(const QString &displayName);
    void dashboardLoaded(const QStringList &buddies, const QStringList &onlineUsers,
                         const QStringList &chatMessages);
    void showError(const QString &message);
    void buddySelectionChanged();
    void buddyAdded(const QString &username);

private:
    void setLoggedIn(bool loggedIn);
    void beginConnectionPresentation();
    void rebuildBuddyTree(const QStringList &buddies, const QStringList &onlineUsers);
    void openServiceWindow(const QString &service, const QString &title, QWidget *content);
    void closeServiceWindow(OtterServiceWindow *window);
    void closeAllServiceWindows();
    void restoreServicePage(QWidget *page);
    void updateServiceButtonStates(OtterServiceWindow *activeWindow = nullptr);

    Ui::MainWindow *ui = nullptr;
    OtterLinkClient *m_client = nullptr;
    QTreeWidget *m_buddyTree = nullptr;
    QFrame *m_desktop = nullptr;
    QHash<QString, QString> m_buddyGroups;
    QHash<QString, QString> m_pendingBuddyGroups;
    QHash<QString, OtterServiceWindow *> m_serviceWindows;
    QHash<OtterServiceWindow *, QString> m_windowServices;
    QTimer m_refreshTimer;
    QTimer m_connectionTimer;
    QTimer m_connectionFinishTimer;
    int m_connectionStage = 0;
    int m_nextWindowOffset = 0;
    bool m_connectionReady = false;
    QString m_connectionDisplayName;
};
