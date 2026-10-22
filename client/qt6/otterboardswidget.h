#pragma once
#include <QJsonArray>
#include <QJsonObject>
#include <QWidget>
#include <QUrl>

class QLabel;
class QListWidget;
class QListWidgetItem;
class QPushButton;
class QTextBrowser;
class QTextEdit;
class OtterLinkClient;

class OtterBoardsWidget final : public QWidget
{
    Q_OBJECT
public:
    explicit OtterBoardsWidget(OtterLinkClient *client, QWidget *parent = nullptr);

private slots:
    void loadBoards();
    void boardsLoaded(const QJsonArray &boards);
    void boardSelected(QListWidgetItem *item);
    void threadsLoaded(const QJsonObject &board, const QJsonArray &threads);
    void threadSelected(QListWidgetItem *item);
    void threadLoaded(const QJsonObject &thread, const QJsonArray &posts);
    void createThread();
    void sendReply();
    void toggleLock();
    void toggleSticky();
    void postLinkClicked(const QUrl &url);
    void showError(const QString &message);

private:
    void renderPosts(const QJsonArray &posts);
    void loadCurrentThread();
    void updateBoardItem(QListWidgetItem *item);
    void updateThreadItem(QListWidgetItem *item, const QJsonObject &thread);
    OtterLinkClient *m_client = nullptr;
    QListWidget *m_boards = nullptr;
    QListWidget *m_threads = nullptr;
    QLabel *m_threadHeading = nullptr;
    QTextBrowser *m_posts = nullptr;
    QTextEdit *m_reply = nullptr;
    QPushButton *m_newThread = nullptr;
    QPushButton *m_replyButton = nullptr;
    QPushButton *m_lockButton = nullptr;
    QPushButton *m_stickyButton = nullptr;
    QPushButton *m_banButton = nullptr;
    QPushButton *m_unbanButton = nullptr;
    QPushButton *m_deleteThreadButton = nullptr;
    qint64 m_boardId = 0;
    qint64 m_threadId = 0;
    bool m_locked = false;
    bool m_sticky = false;
    bool m_canModerate = false;
};
