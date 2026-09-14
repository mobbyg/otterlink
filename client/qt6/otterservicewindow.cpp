#include "otterservicewindow.h"
#include "otterchatwidget.h"
#include "otterlinkclient.h"

#include <QAction>
#include <QEvent>
#include <QHBoxLayout>
#include <QLabel>
#include <QLineEdit>
#include <QMenu>
#include <QMouseEvent>
#include <QPushButton>
#include <QResizeEvent>
#include <QVBoxLayout>
#include <QWidget>

namespace {

class ServiceResizeGrip final : public QWidget
{
public:
    ServiceResizeGrip(QWidget *window, QWidget *parent)
        : QWidget(parent), m_window(window)
    {
        setCursor(Qt::SizeFDiagCursor);
        setFixedSize(18, 18);
        setMouseTracking(true);
        setAttribute(Qt::WA_Hover, true);
    }

protected:
    void mousePressEvent(QMouseEvent *event) override
    {
        if (event->button() == Qt::LeftButton && m_window) {
            m_resizing = true;
            m_startGlobal = event->globalPosition().toPoint();
            m_startSize = m_window->size();
            grabMouse(Qt::SizeFDiagCursor);
            event->accept();
            return;
        }
        QWidget::mousePressEvent(event);
    }

    void mouseMoveEvent(QMouseEvent *event) override
    {
        if (!m_resizing || !(event->buttons() & Qt::LeftButton) || !m_window) {
            QWidget::mouseMoveEvent(event);
            return;
        }
        const QPoint delta = event->globalPosition().toPoint() - m_startGlobal;
        const int width = qMax(m_window->minimumWidth(), m_startSize.width() + delta.x());
        const int height = qMax(m_window->minimumHeight(), m_startSize.height() + delta.y());
        m_window->resize(width, height);
        event->accept();
    }

    void mouseReleaseEvent(QMouseEvent *event) override
    {
        if (m_resizing) {
            m_resizing = false;
            releaseMouse();
            event->accept();
            return;
        }
        QWidget::mouseReleaseEvent(event);
    }

private:
    QWidget *m_window = nullptr;
    bool m_resizing = false;
    QPoint m_startGlobal;
    QSize m_startSize;
};

} // namespace

OtterServiceWindow::OtterServiceWindow(const QString &title, QWidget *content, QWidget *parent)
    : QFrame(parent), m_content(content), m_resizable(title == QStringLiteral("Community Chat"))
{
    setObjectName(QStringLiteral("serviceWindow"));
    setFrameShape(QFrame::StyledPanel);
    setFrameShadow(QFrame::Raised);
    setMinimumSize(300, 190);
    setAttribute(Qt::WA_DeleteOnClose, false);

    if (title == QStringLiteral("Community Chat") && content) {
        QWidget *topLevel = content->window();
        auto *client = topLevel ? topLevel->findChild<OtterLinkClient *>() : nullptr;
        auto *chatWidget = topLevel ? topLevel->findChild<OtterChatWidget *>() : nullptr;
        if (!chatWidget && client)
            chatWidget = new OtterChatWidget(client, nullptr);
        if (chatWidget)
            m_content = chatWidget;
    }

    auto *outer = new QVBoxLayout(this);
    outer->setContentsMargins(2, 2, 2, 2);
    outer->setSpacing(0);

    auto *titleBar = new QFrame(this);
    titleBar->setObjectName(QStringLiteral("serviceWindowTitleBar"));
    auto *titleLayout = new QHBoxLayout(titleBar);
    titleLayout->setContentsMargins(6, 3, 3, 3);
    titleLayout->setSpacing(3);

    m_titleLabel = new QLabel(title, titleBar);
    m_titleLabel->setObjectName(QStringLiteral("serviceWindowTitle"));
    titleLayout->addWidget(m_titleLabel);
    titleLayout->addStretch(1);

    m_minimizeButton = new QPushButton(QStringLiteral("_"), titleBar);
    m_minimizeButton->setObjectName(QStringLiteral("serviceWindowMinimize"));
    m_minimizeButton->setFixedSize(22, 20);
    m_minimizeButton->setToolTip(QStringLiteral("Minimize"));
    titleLayout->addWidget(m_minimizeButton);

    m_closeButton = new QPushButton(QStringLiteral("×"), titleBar);
    m_closeButton->setObjectName(QStringLiteral("serviceWindowClose"));
    m_closeButton->setFixedSize(22, 20);
    m_closeButton->setToolTip(QStringLiteral("Close"));
    titleLayout->addWidget(m_closeButton);

    titleBar->installEventFilter(this);
    outer->addWidget(titleBar);

    if (m_content) {
        m_content->setParent(this);
        m_content->show();
        outer->addWidget(m_content, 1);
    }

    if (m_resizable) {
        m_sizeGrip = new ServiceResizeGrip(this, this);
        m_sizeGrip->setObjectName(QStringLiteral("serviceWindowSizeGrip"));
        m_sizeGrip->raise();
    }

    connect(m_minimizeButton, &QPushButton::clicked, this, &OtterServiceWindow::minimize);
    connect(m_closeButton, &QPushButton::clicked, this, &OtterServiceWindow::closeWindow);

    setupChatEmojiButton();
}

void OtterServiceWindow::setupChatEmojiButton()
{
    if (!m_content || m_titleLabel->text() != QStringLiteral("Community Chat"))
        return;

    // The channel-aware chat widget owns its own input and emoji button.
    if (qobject_cast<OtterChatWidget *>(m_content))
        return;

    auto *chatEdit = m_content->findChild<QLineEdit *>(QStringLiteral("chatEdit"));
    auto *sendButton = m_content->findChild<QPushButton *>(QStringLiteral("sendChatButton"));
    auto *inputLayout = m_content->findChild<QHBoxLayout *>(QStringLiteral("chatInputLayout"));
    auto *chatLayout = m_content->findChild<QVBoxLayout *>(QStringLiteral("chatLayout"));
    if (!chatEdit || !sendButton || !inputLayout || !chatLayout)
        return;

    if (m_resizable)
        chatLayout->setContentsMargins(0, 0, 18, 18);

    if (m_content->findChild<QPushButton *>(QStringLiteral("chatEmojiButton")))
        return;

    auto *emojiButton = new QPushButton(QStringLiteral("😊"), m_content);
    emojiButton->setObjectName(QStringLiteral("chatEmojiButton"));
    emojiButton->setToolTip(QStringLiteral("Choose an emoji"));
    emojiButton->setFixedWidth(38);
    inputLayout->insertWidget(inputLayout->indexOf(sendButton), emojiButton);

    connect(emojiButton, &QPushButton::clicked, this, [emojiButton, chatEdit]() {
        auto *menu = new QMenu(emojiButton);
        const QStringList emojis = {
            QStringLiteral("😀"), QStringLiteral("😃"), QStringLiteral("😄"),
            QStringLiteral("😁"), QStringLiteral("😂"), QStringLiteral("🤣"),
            QStringLiteral("😊"), QStringLiteral("😎"), QStringLiteral("😍"),
            QStringLiteral("🤔"), QStringLiteral("👍"), QStringLiteral("👎"),
            QStringLiteral("❤️"), QStringLiteral("🎉"), QStringLiteral("🔥"),
            QStringLiteral("🦦")
        };
        for (const QString &emoji : emojis) {
            auto *action = menu->addAction(emoji);
            connect(action, &QAction::triggered, chatEdit, [chatEdit, emoji]() {
                chatEdit->insert(emoji);
                chatEdit->setFocus();
            });
        }
        menu->exec(emojiButton->mapToGlobal(QPoint(0, -menu->sizeHint().height())));
        menu->deleteLater();
    });
}

void OtterServiceWindow::activateWindow()
{
    show();
    raise();
    if (m_content)
        m_content->show();

    if (!m_initialSizeApplied && m_titleLabel
        && m_titleLabel->text() == QStringLiteral("People")) {
        const int initialHeight = qMax(250, this->height());
        resize(310, initialHeight);
        m_initialSizeApplied = true;
    }

    if (m_sizeGrip)
        m_sizeGrip->raise();
    setFocus(Qt::OtherFocusReason);
}

void OtterServiceWindow::minimize() { hide(); }
void OtterServiceWindow::closeWindow() { emit closeRequested(this); }

void OtterServiceWindow::mousePressEvent(QMouseEvent *event)
{
    if (event->button() == Qt::LeftButton) {
        m_dragging = true;
        m_dragOffset = event->position().toPoint();
        raise();
        event->accept();
        return;
    }
    QFrame::mousePressEvent(event);
}

void OtterServiceWindow::mouseMoveEvent(QMouseEvent *event)
{
    if (!m_dragging || !(event->buttons() & Qt::LeftButton)) {
        QFrame::mouseMoveEvent(event);
        return;
    }
    move(mapToParent(event->position().toPoint()) - m_dragOffset);
    keepInsideDesktop();
    event->accept();
}

void OtterServiceWindow::mouseReleaseEvent(QMouseEvent *event)
{
    m_dragging = false;
    QFrame::mouseReleaseEvent(event);
}

void OtterServiceWindow::resizeEvent(QResizeEvent *event)
{
    QFrame::resizeEvent(event);
    if (m_sizeGrip) {
        const int x = qMax(0, width() - m_sizeGrip->width());
        const int y = qMax(0, height() - m_sizeGrip->height());
        m_sizeGrip->move(x, y);
        m_sizeGrip->raise();
    }
}

void OtterServiceWindow::keepInsideDesktop()
{
    const QWidget *desktop = parentWidget();
    if (!desktop)
        return;
    const QRect bounds = desktop->rect();
    const int maxX = qMax(0, bounds.width() - width());
    const int maxY = qMax(0, bounds.height() - height());
    move(qBound(0, x(), maxX), qBound(0, y(), maxY));
}

bool OtterServiceWindow::eventFilter(QObject *watched, QEvent *event)
{
    if (watched == m_titleLabel->parentWidget() && event->type() == QEvent::MouseButtonPress) {
        auto *mouse = static_cast<QMouseEvent *>(event);
        if (mouse->button() == Qt::LeftButton) {
            m_dragging = true;
            m_dragOffset = mouse->position().toPoint();
            raise();
            return true;
        }
    }
    if (watched == m_titleLabel->parentWidget() && event->type() == QEvent::MouseMove) {
        auto *mouse = static_cast<QMouseEvent *>(event);
        if (m_dragging && (mouse->buttons() & Qt::LeftButton)) {
            move(mapToParent(mouse->position().toPoint()) - m_dragOffset);
            keepInsideDesktop();
            return true;
        }
    }
    if (watched == m_titleLabel->parentWidget() && event->type() == QEvent::MouseButtonRelease) {
        m_dragging = false;
        return true;
    }
    return QFrame::eventFilter(watched, event);
}
