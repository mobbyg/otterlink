#include "otterboardswidget.h"
#include "otterlinkclient.h"

#include <QHBoxLayout>
#include <QFont>
#include <QInputDialog>
#include <QLabel>
#include <QLineEdit>
#include <QListWidget>
#include <QMessageBox>
#include <QPushButton>
#include <QSplitter>
#include <QTextBrowser>
#include <QTextEdit>
#include <QVBoxLayout>
#include <QUrl>

namespace {
QString shortTimestamp(QString value)
{
    value.replace(QLatin1Char('T'), QLatin1Char(' '));
    if (value.size() > 16)
        value = value.left(16);
    return value;
}
}

OtterBoardsWidget::OtterBoardsWidget(OtterLinkClient *client, QWidget *parent)
    : QWidget(parent), m_client(client)
{
    auto *root = new QVBoxLayout(this);
    root->setContentsMargins(8, 8, 8, 8);

    auto *heading = new QLabel(QStringLiteral("Community Boards — persistent conversations"), this);
    QFont font = heading->font();
    font.setBold(true);
    font.setPointSize(font.pointSize() + 2);
    heading->setFont(font);
    auto *titleRow = new QHBoxLayout;
    auto *refreshButton = new QPushButton(QStringLiteral("Refresh"), this);
    titleRow->addWidget(heading, 1);
    titleRow->addWidget(refreshButton);
    root->addLayout(titleRow);
    connect(refreshButton, &QPushButton::clicked, this, &OtterBoardsWidget::loadBoards);

    auto *split = new QSplitter(Qt::Horizontal, this);
    m_boards = new QListWidget(split);
    m_boards->setMinimumWidth(155);
    m_boards->setToolTip(QStringLiteral("Boards created by the Otter Link administrators"));

    m_threads = new QListWidget(split);
    m_threads->setMinimumWidth(230);

    auto *right = new QWidget(split);
    auto *rightLayout = new QVBoxLayout(right);
    rightLayout->setContentsMargins(0, 0, 0, 0);

    m_threadHeading = new QLabel(QStringLiteral("Select a thread"), right);
    QFont threadFont = m_threadHeading->font();
    threadFont.setBold(true);
    threadFont.setPointSize(threadFont.pointSize() + 1);
    m_threadHeading->setFont(threadFont);
    m_threadHeading->setWordWrap(true);
    m_threadHeading->setTextFormat(Qt::PlainText);
    rightLayout->addWidget(m_threadHeading);

    m_posts = new QTextBrowser(right);
    m_posts->setOpenLinks(false);
    m_posts->setOpenExternalLinks(false);
    connect(m_posts, &QTextBrowser::anchorClicked, this, &OtterBoardsWidget::postLinkClicked);
    rightLayout->addWidget(m_posts, 1);

    auto *moderation = new QHBoxLayout;
    m_newThread = new QPushButton(QStringLiteral("New Thread"), right);
    m_lockButton = new QPushButton(QStringLiteral("Lock Thread"), right);
    m_stickyButton = new QPushButton(QStringLiteral("Make Sticky"), right);
    m_banButton = new QPushButton(QStringLiteral("Ban User"), right);
    m_unbanButton = new QPushButton(QStringLiteral("Unban User"), right);
    m_deleteThreadButton = new QPushButton(QStringLiteral("Remove Thread"), right);
    moderation->addWidget(m_newThread);
    moderation->addStretch(1);
    moderation->addWidget(m_lockButton);
    moderation->addWidget(m_stickyButton);
    moderation->addWidget(m_banButton);
    moderation->addWidget(m_unbanButton);
    moderation->addWidget(m_deleteThreadButton);
    rightLayout->addLayout(moderation);

    auto *replyRow = new QHBoxLayout;
    m_reply = new QTextEdit(right);
    m_reply->setPlaceholderText(QStringLiteral("Write a reply…"));
    m_reply->setMaximumHeight(90);
    m_replyButton = new QPushButton(QStringLiteral("Reply"), right);
    replyRow->addWidget(m_reply, 1);
    replyRow->addWidget(m_replyButton);
    rightLayout->addLayout(replyRow);

    split->addWidget(m_boards);
    split->addWidget(m_threads);
    split->addWidget(right);
    split->setStretchFactor(2, 1);
    root->addWidget(split, 1);

    m_lockButton->setEnabled(false);
    m_stickyButton->setEnabled(false);
    m_banButton->setEnabled(false);
    m_unbanButton->setEnabled(false);
    m_deleteThreadButton->setEnabled(false);
    m_reply->setEnabled(false);
    m_replyButton->setEnabled(false);
    m_newThread->setEnabled(false);

    connect(m_boards, &QListWidget::itemClicked, this, &OtterBoardsWidget::boardSelected);
    connect(m_threads, &QListWidget::itemClicked, this, &OtterBoardsWidget::threadSelected);
    connect(m_newThread, &QPushButton::clicked, this, &OtterBoardsWidget::createThread);
    connect(m_replyButton, &QPushButton::clicked, this, &OtterBoardsWidget::sendReply);
    connect(m_lockButton, &QPushButton::clicked, this, &OtterBoardsWidget::toggleLock);
    connect(m_stickyButton, &QPushButton::clicked, this, &OtterBoardsWidget::toggleSticky);

    connect(m_banButton, &QPushButton::clicked, this, [this]() {
        bool ok = false;
        const QString username = QInputDialog::getText(
            this, QStringLiteral("Ban User"), QStringLiteral("Username to ban"),
            QLineEdit::Normal, QString(), &ok).trimmed();
        if (!ok || username.isEmpty())
            return;
        const QString reason = QInputDialog::getText(
            this, QStringLiteral("Ban User"), QStringLiteral("Reason (optional)"),
            QLineEdit::Normal, QString(), &ok);
        if (ok)
            m_client->moderateBoardUser(m_boardId, username, QStringLiteral("ban"), reason);
    });
    connect(m_unbanButton, &QPushButton::clicked, this, [this]() {
        bool ok = false;
        const QString username = QInputDialog::getText(
            this, QStringLiteral("Unban User"), QStringLiteral("Username to unban"),
            QLineEdit::Normal, QString(), &ok).trimmed();
        if (ok && !username.isEmpty())
            m_client->moderateBoardUser(m_boardId, username, QStringLiteral("unban"));
    });
    connect(m_deleteThreadButton, &QPushButton::clicked, this, [this]() {
        if (m_threadId < 1 ||
            QMessageBox::question(this, QStringLiteral("Remove Thread"),
                QStringLiteral("Permanently remove this thread and all replies?")) != QMessageBox::Yes)
            return;
        const qint64 doomedThread = m_threadId;
        m_threadId = 0;
        m_client->deleteBoardThread(doomedThread);
    });

    connect(m_client, &OtterLinkClient::boardsLoaded, this, &OtterBoardsWidget::boardsLoaded);
    connect(m_client, &OtterLinkClient::boardThreadsLoaded, this, &OtterBoardsWidget::threadsLoaded);
    connect(m_client, &OtterLinkClient::boardThreadLoaded, this, &OtterBoardsWidget::threadLoaded);
    connect(m_client, &OtterLinkClient::boardThreadCreated, this,
            [this](const QJsonObject &thread, const QJsonArray &) {
                m_threadId = qint64(thread.value("id").toDouble());
                m_client->loadBoardThreads(m_boardId);
            });
    connect(m_client, &OtterLinkClient::boardReplyCreated, this,
            [this](const QJsonObject &) {
                m_reply->clear();
                m_client->loadBoardThreads(m_boardId);
            });
    connect(m_client, &OtterLinkClient::boardActionCompleted, this, [this]() {
        loadBoards();
    });
    connect(m_client, &OtterLinkClient::errorOccurred, this, &OtterBoardsWidget::showError);

    m_posts->setHtml(QStringLiteral(
        "<h3>Welcome to Community Boards</h3>"
        "<p>Select a board, then a thread. Conversations remain available after everyone leaves.</p>"));
    loadBoards();
}

void OtterBoardsWidget::loadBoards()
{
    m_client->loadBoards();
}

void OtterBoardsWidget::updateBoardItem(QListWidgetItem *item)
{
    if (!item)
        return;
    const QString name = item->data(Qt::UserRole + 1).toString();
    const int threads = item->data(Qt::UserRole + 2).toInt();
    const int unread = item->data(Qt::UserRole + 3).toInt();
    QString detail = QStringLiteral("%1 thread%2").arg(threads).arg(threads == 1 ? "" : "s");
    if (unread > 0)
        detail += QStringLiteral(" · %1 unread").arg(unread);
    item->setText(name + QStringLiteral("\n") + detail);
}

void OtterBoardsWidget::updateThreadItem(QListWidgetItem *item, const QJsonObject &thread)
{
    if (!item)
        return;

    const bool wasUnread = item->data(Qt::UserRole + 4).toBool();
    const bool unread = thread.value("unread").toBool();
    item->setData(Qt::UserRole + 4, unread);

    if (wasUnread && !unread) {
        for (int i = 0; i < m_boards->count(); ++i) {
            auto *boardItem = m_boards->item(i);
            if (boardItem->data(Qt::UserRole).toLongLong() != m_boardId)
                continue;
            const int count = boardItem->data(Qt::UserRole + 3).toInt();
            boardItem->setData(Qt::UserRole + 3, qMax(0, count - 1));
            updateBoardItem(boardItem);
            break;
        }
    }

    QString subject = thread.value("subject").toString();
    if (unread)
        subject = QStringLiteral("● ") + subject;
    if (thread.value("sticky").toBool())
        subject = QStringLiteral("📌 ") + subject;
    if (thread.value("locked").toBool())
        subject = QStringLiteral("🔒 ") + subject;

    const QString author = thread.value("author").toString();
    const QString lastAuthor = thread.value("last_post_author").toString();
    const int replies = thread.value("reply_count").toInt();
    const QString activity = shortTimestamp(thread.value("last_post_at").toString());
    item->setText(QStringLiteral("%1\nby %2 · %3 repl%4 · %5")
        .arg(subject, author)
        .arg(replies)
        .arg(replies == 1 ? QStringLiteral("y") : QStringLiteral("ies"))
        .arg(activity.isEmpty() ? lastAuthor : QStringLiteral("%1 · %2").arg(lastAuthor, activity)));
    item->setToolTip(thread.value("subject").toString());
}

void OtterBoardsWidget::boardsLoaded(const QJsonArray &boards)
{
    const qint64 old = m_boardId;
    const qint64 oldThread = m_threadId;
    m_boards->clear();
    for (const auto &value : boards) {
        const QJsonObject board = value.toObject();
        auto *item = new QListWidgetItem(m_boards);
        item->setData(Qt::UserRole, qint64(board.value("id").toDouble()));
        item->setData(Qt::UserRole + 1, board.value("name").toString());
        item->setData(Qt::UserRole + 2, board.value("thread_count").toInt());
        item->setData(Qt::UserRole + 3, board.value("unread_count").toInt());
        item->setToolTip(board.value("description").toString() +
            (board.value("rules").toString().isEmpty() ? QString() :
             QStringLiteral("\n\nRules: ") + board.value("rules").toString()));
        updateBoardItem(item);
    }

    for (int i = 0; i < m_boards->count(); ++i) {
        if (m_boards->item(i)->data(Qt::UserRole).toLongLong() == old) {
            m_boards->setCurrentRow(i);
            boardSelected(m_boards->item(i));
            m_threadId = oldThread;
            return;
        }
    }
    if (m_boards->count()) {
        m_boards->setCurrentRow(0);
        boardSelected(m_boards->item(0));
    } else {
        m_boardId = 0;
        m_threadId = 0;
        m_threads->clear();
        m_threadHeading->setText(QStringLiteral("No board selected"));
        m_posts->setHtml(QStringLiteral(
            "<h3>No boards yet</h3><p>An administrator needs to create a board in the web administration panel.</p>"));
        m_newThread->setEnabled(false);
        m_reply->setEnabled(false);
        m_replyButton->setEnabled(false);
    }
}

void OtterBoardsWidget::boardSelected(QListWidgetItem *item)
{
    if (!item)
        return;
    m_boardId = item->data(Qt::UserRole).toLongLong();
    m_threadId = 0;
    m_canModerate = false;
    m_threads->clear();
    m_threadHeading->setText(item->data(Qt::UserRole + 1).toString());
    m_posts->setHtml(QStringLiteral("<p>Select a thread or start a new conversation.</p>"));
    m_newThread->setEnabled(m_boardId > 0);
    m_reply->clear();
    m_reply->setEnabled(false);
    m_replyButton->setEnabled(false);
    m_lockButton->setEnabled(false);
    m_stickyButton->setEnabled(false);
    m_banButton->setEnabled(false);
    m_unbanButton->setEnabled(false);
    m_deleteThreadButton->setEnabled(false);
    m_client->loadBoardThreads(m_boardId);
}

void OtterBoardsWidget::threadsLoaded(const QJsonObject &board, const QJsonArray &threads)
{
    if (qint64(board.value("id").toDouble()) != m_boardId)
        return;

    m_threads->clear();
    for (const auto &value : threads) {
        const QJsonObject thread = value.toObject();
        auto *item = new QListWidgetItem(m_threads);
        item->setData(Qt::UserRole, qint64(thread.value("id").toDouble()));
        updateThreadItem(item, thread);
        if (qint64(thread.value("id").toDouble()) == m_threadId)
            m_threads->setCurrentItem(item);
    }

    if (m_threads->count()) {
        if (m_threadId > 0) {
            for (int i = 0; i < m_threads->count(); ++i) {
                if (m_threads->item(i)->data(Qt::UserRole).toLongLong() == m_threadId) {
                    m_threads->setCurrentItem(m_threads->item(i));
                    threadSelected(m_threads->item(i));
                    return;
                }
            }
        }
        m_threads->setCurrentRow(0);
        threadSelected(m_threads->item(0));
    } else {
        m_threadId = 0;
        m_canModerate = false;
        m_threadHeading->setText(board.value("name").toString());
        QString html = QStringLiteral("<h3>%1</h3><p>%2</p>")
            .arg(board.value("name").toString().toHtmlEscaped(),
                 board.value("description").toString().toHtmlEscaped());
        const QString rules = board.value("rules").toString().trimmed();
        if (!rules.isEmpty())
            html += QStringLiteral("<p><b>Board rules</b><br>%1</p>").arg(rules.toHtmlEscaped().replace("\n", "<br>"));
        html += QStringLiteral("<p>No threads yet. Start the first conversation.</p>");
        m_posts->setHtml(html);
        m_reply->setEnabled(false);
        m_replyButton->setEnabled(false);
        m_lockButton->setEnabled(false);
        m_stickyButton->setEnabled(false);
        m_banButton->setEnabled(false);
        m_unbanButton->setEnabled(false);
        m_deleteThreadButton->setEnabled(false);
    }
}

void OtterBoardsWidget::threadSelected(QListWidgetItem *item)
{
    if (!item)
        return;
    m_threadId = item->data(Qt::UserRole).toLongLong();
    loadCurrentThread();
}

void OtterBoardsWidget::loadCurrentThread()
{
    if (m_threadId > 0)
        m_client->loadBoardThread(m_threadId);
}

void OtterBoardsWidget::threadLoaded(const QJsonObject &thread, const QJsonArray &posts)
{
    if (qint64(thread.value("id").toDouble()) != m_threadId)
        return;

    m_locked = thread.value("locked").toBool();
    m_sticky = thread.value("sticky").toBool();
    QString title = thread.value("subject").toString();
    title += QStringLiteral(" — by %1").arg(thread.value("author").toString());
    if (m_locked)
        title += QStringLiteral("  🔒 Locked");
    if (m_sticky)
        title += QStringLiteral("  📌 Sticky");
    m_threadHeading->setText(title);

    for (int i = 0; i < m_threads->count(); ++i) {
        auto *item = m_threads->item(i);
        if (item->data(Qt::UserRole).toLongLong() == m_threadId) {
            updateThreadItem(item, thread);
            break;
        }
    }

    m_lockButton->setText(m_locked ? QStringLiteral("Unlock Thread") : QStringLiteral("Lock Thread"));
    m_stickyButton->setText(m_sticky ? QStringLiteral("Unstick Thread") : QStringLiteral("Make Sticky"));
    const bool canModerate = thread.value("can_moderate").toBool();
    m_canModerate = canModerate;
    m_reply->setEnabled(!m_locked);
    m_replyButton->setEnabled(!m_locked);
    m_lockButton->setEnabled(canModerate);
    m_stickyButton->setEnabled(canModerate);
    m_banButton->setEnabled(canModerate);
    m_unbanButton->setEnabled(canModerate);
    m_deleteThreadButton->setEnabled(canModerate);
    renderPosts(posts);
}

void OtterBoardsWidget::renderPosts(const QJsonArray &posts)
{
    QString html;
    for (const auto &value : posts) {
        const QJsonObject post = value.toObject();
        const QString name = post.value("display_name").toString(post.value("author").toString()).toHtmlEscaped();
        const QString created = post.value("created_at").toString().toHtmlEscaped();
        const QString body = post.value("body").toString().toHtmlEscaped().replace("\n", "<br>");
        html += QStringLiteral(
            "<section style='border-bottom:1px solid #b7c7dc;padding:8px 2px'>"
            "<b>%1</b> <small>%2</small><p style='white-space:pre-wrap'>%3</p></section>")
            .arg(name, created, body);
        if (m_canModerate) {
            const QString postId = QString::number(qint64(post.value("id").toDouble()));
            html += QStringLiteral("<p><small><a href='otterlink-post://delete/%1'>Remove this post</a></small></p>").arg(postId);
        }
    }
    m_posts->setHtml(html.isEmpty() ? QStringLiteral("<p>No posts.</p>") : html);
}

void OtterBoardsWidget::createThread()
{
    if (m_boardId < 1)
        return;
    bool ok = false;
    const QString subject = QInputDialog::getText(
        this, QStringLiteral("New Thread"), QStringLiteral("Subject"),
        QLineEdit::Normal, QString(), &ok);
    if (!ok)
        return;
    const QString body = QInputDialog::getMultiLineText(
        this, QStringLiteral("New Thread"), QStringLiteral("Message"), QString(), &ok);
    if (!ok)
        return;
    m_client->createBoardThread(m_boardId, subject, body);
}

void OtterBoardsWidget::sendReply()
{
    const QString body = m_reply->toPlainText();
    if (m_threadId < 1 || body.trimmed().isEmpty())
        return;
    m_client->replyBoardThread(m_threadId, body);
}

void OtterBoardsWidget::toggleLock()
{
    if (m_threadId > 0)
        m_client->moderateBoardThread(m_threadId, !m_locked, m_sticky);
}

void OtterBoardsWidget::toggleSticky()
{
    if (m_threadId > 0)
        m_client->moderateBoardThread(m_threadId, m_locked, !m_sticky);
}

void OtterBoardsWidget::postLinkClicked(const QUrl &url)
{
    if (!m_canModerate || url.scheme() != QStringLiteral("otterlink-post") ||
        url.host() != QStringLiteral("delete"))
        return;

    bool ok = false;
    const qint64 postId = url.path().mid(1).toLongLong(&ok);
    if (!ok || postId < 1)
        return;
    if (QMessageBox::question(this, QStringLiteral("Remove Post"),
            QStringLiteral("Permanently remove this post? Removing the opening post also deletes the entire thread and its replies."))
        != QMessageBox::Yes)
        return;
    m_client->deleteBoardPost(postId);
}

void OtterBoardsWidget::showError(const QString &message)
{
    if (!message.isEmpty())
        QMessageBox::warning(this, QStringLiteral("Community Boards"), message);
}
