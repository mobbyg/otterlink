#pragma once

#include <QWidget>
#include <QString>

class QNetworkAccessManager;

class QComboBox;
class QLabel;
class QTextBrowser;
class QListWidget;
class QListWidgetItem;
class QPushButton;
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
    void loadArticleImages(const QString &html, const QString &articleUrl);

    OtterLinkClient *m_client = nullptr;
    QComboBox *m_categoryCombo = nullptr;
    QComboBox *m_sourceCombo = nullptr;
    QPushButton *m_refreshButton = nullptr;
    QListWidget *m_headlines = nullptr;
    QLabel *m_title = nullptr;
    QLabel *m_meta = nullptr;
    QTextBrowser *m_article = nullptr;
    QPushButton *m_originalButton = nullptr;
    QString m_selectedUrl;
    QString m_selectedImageUrl;
    QNetworkAccessManager *m_imageNetwork = nullptr;
};
