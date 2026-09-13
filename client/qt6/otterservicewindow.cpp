#include "otterservicewindow.h"

#include <QEvent>
#include <QHBoxLayout>
#include <QLabel>
#include <QLineEdit>
#include <QMenu>
#include <QMouseEvent>
#include <QPushButton>
#include <QSizeGrip>
#include <QVBoxLayout>
#include <QWidget>

OtterServiceWindow::OtterServiceWindow(const QString &title, QWidget *content, QWidget *parent)
    : QFrame(parent),
      m_content(content)
{
    setObjectName(QStringLiteral("serviceWindow"));
    setFrameShape(QFrame::StyledPanel);
    setFrameShadow(QFrame::Raised);
    setMinimumSize(300, 190);
    setAttribute(Qt::WA_DeleteOnClose, false);

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

    auto *resizeBar = new QHBoxLayout;
    resizeBar->setContentsMargins(0, 0, 0, 0);
    resizeBar->addStretch(1);
    auto *sizeGrip = new QSizeGrip(this);
    sizeGrip->setObjectName(QStringLiteral("serviceWindowSizeGrip"));
    sizeGrip->setFixedSize(14, 14);
    resizeBar->addWidget(sizeGrip, 0, Qt::AlignRight | Qt::AlignBottom);
    outer->addLayout(resizeBar);

    connect(m_minimizeButton, &QPushButton::clicked, this, &OtterServiceWindow::minimize);
    connect(m_closeButton, &QPushButton::clicked, this, &OtterServiceWindow::closeWindow);

    setupChatEmojiButton();
}

void OtterServiceWindow::setupChatEmojiButton()
{
    if (!m_content || m_titleLabel->text() != QStringLiteral("Community Chat"))
        return;

    auto *chatEdit = m_content->findChild<QLineEdit *>(QStringLiteral("chatEdit"));
    auto *sendButton = m_content->findChild<QPushButton *>(QStringLiteral("sendChatButton"));
    if (!chatEdit || !sendButton)
        return;

    auto *inputLayout = qobject_cast<QHBoxLayout *>(chatEdit->parentWidget()->layout());
    if (!inputLayout)
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
        const int height = qMax(250, height());
        resize(310, height);
        m_initialSizeApplied = true;
    }

    setFocus(Qt::OtherFocusReason);
}

void OtterServiceWindow::minimize()
{
    hide();
}

void OtterServiceWindow::closeWindow()
{
    emit closeRequested(this);
}

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

void OtterServiceWindow::keepInsideDesktop()
{
    const QWidget *desktop = parentWidget();
    if (!desktop)
        return;

    const int maxX = qMax(0, desktop->width() - width());
    const int maxY = qMax(0, desktop->height() - height());
    move(qBound(0, x(), maxX), qBound(0, y(), maxY));
}

bool OtterServiceWindow::eventFilter(QObject *watched, QEvent *event)
{
    if (watched == m_titleLabel->parentWidget()) {
        if (event->type() == QEvent::MouseButtonPress) {
            auto *mouseEvent = static_cast<QMouseEvent *>(event);
            if (mouseEvent->button() == Qt::LeftButton) {
                m_dragging = true;
                m_dragOffset = mouseEvent->position().toPoint();
                raise();
                return true;
            }
        } else if (event->type() == QEvent::MouseMove && m_dragging) {
            auto *mouseEvent = static_cast<QMouseEvent *>(event);
            if (mouseEvent->buttons() & Qt::LeftButton) {
                move(mapToParent(mouseEvent->position().toPoint()) - m_dragOffset);
                keepInsideDesktop();
                return true;
            }
        } else if (event->type() == QEvent::MouseButtonRelease) {
            m_dragging = false;
        }
    }

    return QFrame::eventFilter(watched, event);
}
