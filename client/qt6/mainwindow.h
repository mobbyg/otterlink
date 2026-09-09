#pragma once

#include <QHash>
#include <QMainWindow>
#include <QTimer>

class QTreeWidget;

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

private:
    void setLoggedIn(bool loggedIn);
    void beginConnectionPresentation();
    void rebuildBuddyTree(const QStringList &buddies, const QStringList &onlineUsers);

    Ui::MainWindow *ui = nullptr;
    OtterLinkClient *m_client = nullptr;
    QTreeWidget *m_buddyTree = nullptr;
    QHash<QString, QString> m_buddyGroups;
    QTimer m_refreshTimer;
    QTimer m_connectionTimer;
    QTimer m_connectionFinishTimer;
    int m_connectionStage = 0;
    bool m_connectionReady = false;
    QString m_connectionDisplayName;
};
