#pragma once

#include <QImage>
#include <QJsonObject>
#include <QPixmap>
#include <QWidget>

class QScrollArea;
class QVBoxLayout;

class OtterHomePage final : public QWidget
{
    Q_OBJECT

public:
    explicit OtterHomePage(QWidget *parent = nullptr);

    void setServerScreen(const QJsonObject &screen);
    void setServerScreen(const QJsonObject &screen, const QImage &background);

signals:
    void serviceRequested(const QString &service);

protected:
    void resizeEvent(QResizeEvent *event) override;

private:
    void buildFallback();
    void buildFromContent(const QJsonObject &content);
    void addAnnouncement(const QString &title, const QString &body, const QString &actionLabel,
                         const QString &service, QWidget *parent);
    void addServiceTile(const QString &title, const QString &description, const QString &icon,
                        const QString &service, QWidget *parent);
    void clearPage();
    void applyBackground(const QPixmap &background);

    QScrollArea *m_scrollArea = nullptr;
    QWidget *m_page = nullptr;
    QVBoxLayout *m_layout = nullptr;
    QPixmap m_background;
};
