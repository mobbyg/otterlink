#pragma once

#include <QImage>
#include <QJsonObject>
#include <QPixmap>
#include <QWidget>
#include <QHash>

class QJsonArray;
class QPushButton;
class QScrollArea;
class QVBoxLayout;

class OtterHomePage final : public QWidget
{
    Q_OBJECT

public:
    explicit OtterHomePage(QWidget *parent = nullptr);

    void setServerScreen(const QJsonObject &screen);
    void setServerScreen(const QJsonObject &screen, const QImage &background);
    void setServerAsset(qint64 assetId, const QImage &image);

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
    void buildOverlay(const QJsonArray &elements);
    void layoutOverlay();
    void addImageElement(qint64 assetId, const QImage &image, const QJsonObject &item);
    void applyButtonAsset(QPushButton *button, const QImage &image);

    QScrollArea *m_scrollArea = nullptr;
    QWidget *m_page = nullptr;
    QVBoxLayout *m_layout = nullptr;
    QPixmap m_background;
    bool m_templateMode = false;
    QWidget *m_overlay = nullptr;
    QHash<qint64, QImage> m_loadedAssets;
};
