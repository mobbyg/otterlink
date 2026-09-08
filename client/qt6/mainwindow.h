#pragma once

#include <QMainWindow>

class QLabel;
class QLineEdit;
class QListWidget;
class QPushButton;
class OtterLinkClient;

class MainWindow final : public QMainWindow
{
    Q_OBJECT

public:
    explicit MainWindow(QWidget *parent = nullptr);

private slots:
    void login();
    void logout();
    void showDashboard(const QString &displayName);
    void dashboardLoaded(const QStringList &buddies, const QStringList &onlineUsers,
                         const QStringList &chatMessages);
    void showError(const QString &message);

private:
    void buildUi();
    void setLoggedIn(bool loggedIn);

    OtterLinkClient *m_client = nullptr;
    QWidget *m_authPage = nullptr;
    QWidget *m_dashboardPage = nullptr;
    QLineEdit *m_serverEdit = nullptr;
    QLineEdit *m_usernameEdit = nullptr;
    QLineEdit *m_passwordEdit = nullptr;
    QLabel *m_identityLabel = nullptr;
    QListWidget *m_buddies = nullptr;
    QListWidget *m_online = nullptr;
    QListWidget *m_chat = nullptr;
};
