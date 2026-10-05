#pragma once

#include <QHash>
#include <QJsonArray>
#include <QMainWindow>
#include <QTimer>

class QLabel;
class QFrame;
class QPushButton;
class QSoundEffect;
class QTreeWidget;
class QWidget;
class OtterHomePage;
class OtterPeopleWidget;
class OtterServiceWindow;
class OtterKeywordWidget;

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
    void directUnreadLoaded(const QJsonArray &messages);
    void openPrivateMessage(const QString &username);
    void enterKeyword();
    void keywordResolved(const QJsonObject &keyword);

private:
    void setLoggedIn(bool loggedIn);
    void beginConnectionPresentation();
    void toggleConnectionSfx();
    void updateConnectionPresentation(int stage);
    void updateSfxButton();
    void rebuildBuddyTree(const QStringList &buddies, const QStringList &onlineUsers);
    void openServiceWindow(const QString &service, const QString &title, QWidget *content);
    void closeServiceWindow(OtterServiceWindow *window);
    void closeAllServiceWindows();
    void restoreServicePage(QWidget *page);
    void updateServiceButtonStates(OtterServiceWindow *activeWindow = nullptr);
    void openKeywordTarget(const QString &type, qint64 id);

    Ui::MainWindow *ui = nullptr;
    OtterLinkClient *m_client = nullptr;
    QTreeWidget *m_buddyTree = nullptr;
    QFrame *m_desktop = nullptr;
    OtterHomePage *m_homePage = nullptr;
    OtterPeopleWidget *m_peopleWidget = nullptr;
    QHash<QString, QString> m_buddyGroups;
    QHash<QString, QString> m_pendingBuddyGroups;
    QHash<QString, OtterServiceWindow *> m_serviceWindows;
    QHash<OtterServiceWindow *, QString> m_windowServices;
    QTimer m_refreshTimer;
    QTimer m_homeRefreshTimer;
    QTimer m_connectionTimer;
    QTimer m_connectionFinishTimer;
    QLabel *m_connectionImageLabel = nullptr;
    QLabel *m_connectionStatusLabel = nullptr;
    QWidget *m_connectionPresentationWidget = nullptr;
    QPushButton *m_sfxButton = nullptr;
    QSoundEffect *m_connectionSfx = nullptr;
    int m_connectionStage = 0;
    int m_nextWindowOffset = 0;
    bool m_connectionReady = false;
    bool m_connectionSfxEnabled = true;
    bool m_homeScreenRequested = false;
    QString m_connectionDisplayName;
};
