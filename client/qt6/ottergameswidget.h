#pragma once

#include <QJsonArray>
#include <QJsonObject>
#include <QString>
#include <QWidget>

class QLabel;
class QListWidget;
class QListWidgetItem;
class QPushButton;
class QLineEdit;
class QTextBrowser;
class QTimer;
class OtterLinkClient;
class OtterHomePage;
class QTabWidget;

class OtterGamesWidget final : public QWidget
{
    Q_OBJECT
public:
    explicit OtterGamesWidget(OtterLinkClient *client, QWidget *parent = nullptr);
    ~OtterGamesWidget() override;
    void launchGameSlug(const QString &slug);

signals:
    void serviceRequested(const QString &service);

private slots:
    void loadGames();
    void gamesLoaded(const QJsonArray &games);
    void selectGame(QListWidgetItem *item);
    void startGame();
    void sendCommand();
    void pollSession();
    void outputLoaded(const QJsonObject &result);
    void sessionStarted(const QJsonObject &session);
    void sessionStopped(const QString &sessionId);
    void showError(const QString &message);

private:
    void appendOutput(const QString &text);
    void updateControls();
    OtterLinkClient *m_client = nullptr;
    QTabWidget *m_tabs = nullptr;
    OtterHomePage *m_gamesHome = nullptr;
    QListWidget *m_games = nullptr;
    QLabel *m_title = nullptr;
    QLabel *m_metadata = nullptr;
    QLabel *m_description = nullptr;
    QLabel *m_status = nullptr;
    QTextBrowser *m_output = nullptr;
    QLineEdit *m_command = nullptr;
    QPushButton *m_playButton = nullptr;
    QPushButton *m_stopButton = nullptr;
    QPushButton *m_sendButton = nullptr;
    QTimer *m_pollTimer = nullptr;
    qint64 m_gameId = 0;
    qint64 m_lastSequence = 0;
    QString m_sessionId;
    QString m_requestedSlug;
    bool m_gameAvailable = false;
    bool m_pollInFlight = false;
    bool m_gamesLoaded = false;
};
