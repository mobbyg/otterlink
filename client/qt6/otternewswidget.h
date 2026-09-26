#pragma once

#include <QJsonObject>
#include <QString>
#include <QWidget>

class QComboBox;
class QLabel;
class QListWidget;
class QListWidgetItem;
class QNetworkAccessManager;
class QPushButton;
class QStackedWidget;
class QTextBrowser;
class OtterLinkClient;
class QJsonArray;

class OtterNewsWidget final : public QWidget
{
    Q_OBJECT

public:
    explicit OtterNewsWidget(OtterLinkClient *client, QWidget *parent = nullptr);

private slots:
    void loadNews();
    void showItem(QListWidgetItem *item);
    void newsLoaded(const QJsonArray &items);
    void newsSourcesLoaded(const QJsonArray &sources);
    void openOriginal();

private:
    void populateCategories(const QJsonArray &items);
    void populateSources(const QJsonArray &sources);
    void clearArticle();
    void showItemData(const QJsonObject &data);
    void loadImageIntoLabel(const QString &url, QLabel *label, int maxWidth, int maxHeight);
    void loadArticleImages(const QString &html, const QString &articleUrl);

    OtterLinkClient *m_client = nullptr;
    QComboBox *m_categoryCombo = nullptr;
    QComboBox *m_sourceCombo = nullptr;
    QPushButton *m_refreshButton = nullptr;
    QListWidget *m_headlines = nullptr;

    QLabel *m_dateLabel = nullptr;
    QLabel *m_featuredImage = nullptr;
    QPushButton *m_featuredTitle = nullptr;
    QLabel *m_featuredMeta = nullptr;
    QLabel *m_featuredSummary = nullptr;

    QStackedWidget *m_stack = nullptr;
    QPushButton *m_backButton = nullptr;
    QLabel *m_title = nullptr;
    QLabel *m_meta = nullptr;
    QTextBrowser *m_article = nullptr;
    QPushButton *m_originalButton = nullptr;

    QJsonObject m_featuredItem;
    QString m_selectedUrl;
    QString m_selectedImageUrl;
    QNetworkAccessManager *m_imageNetwork = nullptr;
};
