#pragma once

#include <QFrame>

class QEvent;
class QLabel;
class QPushButton;
class QMouseEvent;
class QWidget;

class OtterServiceWindow final : public QFrame
{
    Q_OBJECT

public:
    explicit OtterServiceWindow(const QString &title, QWidget *content,
                                QWidget *parent = nullptr);

    QWidget *contentWidget() const { return m_content; }
    void activateWindow();

signals:
    void closeRequested(OtterServiceWindow *window);

protected:
    void mousePressEvent(QMouseEvent *event) override;
    void mouseMoveEvent(QMouseEvent *event) override;
    void mouseReleaseEvent(QMouseEvent *event) override;
    bool eventFilter(QObject *watched, QEvent *event) override;

private:
    void minimize();
    void closeWindow();
    void keepInsideDesktop();

    QWidget *m_content = nullptr;
    QLabel *m_titleLabel = nullptr;
    QPushButton *m_minimizeButton = nullptr;
    QPushButton *m_closeButton = nullptr;
    bool m_dragging = false;
    QPoint m_dragOffset;
};
