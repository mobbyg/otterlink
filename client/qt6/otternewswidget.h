#pragma once

#include <QWidget>

class QComboBox;
class QLabel;
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
    void openOriginal();

private:
    void populateCategories(const QJsonArray &items);
    void clearArticle();

    OtterLinkClient *m_client = nullptr;
    QComboBox *m_categoryCombo = nullptr;
    QPushButton *m_refreshButton = nullptr;
    QListWidget *m_headlines = nullptr;
    QLabel *m_title = nullptr;
    QLabel *m_meta = nullptr;
    QLabel *m_summary = nullptr;
    QPushButton *m_originalButton = nullptr;
    QString m_selectedUrl;
};
