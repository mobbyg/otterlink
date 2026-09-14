#pragma once

#include <QFrame>
#include <QPoint>

class QEvent;
class QLabel;
class QPushButton;
class QMouseEvent;
class QResizeEvent;
class QWidget;

class OtterServiceWindow final : public QFrame
{
    Q_OBJECT

public:
    explicit OtterServiceWindow(const QString &title, QWidget *content,
                                bool resizable = false, QWidget *parent = nullptr);

    QWidget *contentWidget() const { return m_content; }
    void activateWindow();

signals:
    void closeRequested(OtterServiceWindow *window);

protected:
    void mousePressEvent(QMouseEvent *event) override;
    void mouseMoveEvent(QMouseEvent *event) override;
    void mouseReleaseEvent(QMouseEvent *event) override;
    void resizeEvent(QResizeEvent *event) override;
    bool eventFilter(QObject *watched, QEvent *event) override;

private:
    void minimize();
    void closeWindow();
    void keepInsideDesktop();
    void setupChatEmojiButton();

    QWidget *m_content = nullptr;
    QWidget *m_sizeGrip = nullptr;
    QLabel *m_titleLabel = nullptr;
    QPushButton *m_minimizeButton = nullptr;
    QPushButton *m_closeButton = nullptr;
    bool m_resizable = false;
    bool m_dragging = false;
    bool m_initialSizeApplied = false;
    QPoint m_dragOffset;
};
