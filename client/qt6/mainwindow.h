#pragma once

#include <QMainWindow>

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
    void showDashboard(const QString &displayName);
    void dashboardLoaded(const QStringList &buddies, const QStringList &onlineUsers,
                         const QStringList &chatMessages);
    void showError(const QString &message);

private:
    void setLoggedIn(bool loggedIn);

    Ui::MainWindow *ui = nullptr;
    OtterLinkClient *m_client = nullptr;
};
