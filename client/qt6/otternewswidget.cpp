#include "otternewswidget.h"

#include "otterlinkclient.h"

#include <QComboBox>
#include <QDesktopServices>
#include <QHBoxLayout>
#include <QJsonArray>
#include <QJsonObject>
#include <QJsonValue>
#include <QLabel>
#include <QListWidget>
#include <QListWidgetItem>
#include <QPushButton>
#include <QPixmap>
#include <QSignalBlocker>
#include <QStringList>
#include <QUrl>
#include <QVBoxLayout>
#include <QDateTime>
#include <QNetworkAccessManager>
#include <QNetworkRequest>
#include <QNetworkReply>

namespace {

QString displayPublished(const QString &value)
{
    const QDateTime parsed = QDateTime::fromString(value, Qt::ISODate);
    if (!parsed.isValid())
        return value;
    return parsed.toLocalTime().toString(QStringLiteral("MMM d, yyyy h:mm AP"));
}

QString sourceLine(const QJsonObject &item)
{
    const QString source = item.value(QStringLiteral("source_name")).toString();
    const QString category = item.value(QStringLiteral("category")).toString();
    const QString published = displayPublished(item.value(QStringLiteral("published_at")).toString());

    QStringList parts;
    if (!source.isEmpty())
        parts << source;
    if (!category.isEmpty())
        parts << category;
    if (!published.isEmpty())
        parts << published;
    return parts.join(QStringLiteral(" • "));
}

} // namespace

OtterNewsWidget::OtterNewsWidget(OtterLinkClient *client, QWidget *parent)
    : QWidget(parent),
      m_client(client),
      m_imageNetwork(new QNetworkAccessManager(this))
{
    auto *root = new QVBoxLayout(this);
    root->setContentsMargins(12, 10, 12, 12);
    root->setSpacing(8);

    auto *masthead = new QHBoxLayout;
    masthead->setSpacing(8);

    auto *newsIcon = new QLabel(this);
    newsIcon->setPixmap(QPixmap(QStringLiteral(":/images/news.png")).scaled(
        32, 32, Qt::KeepAspectRatio, Qt::SmoothTransformation));
    newsIcon->setFixedSize(32, 32);
    masthead->addWidget(newsIcon);

    auto *mastheadText = new QVBoxLayout;
    mastheadText->setSpacing(0);
    auto *mastheadTitle = new QLabel(QStringLiteral("Otter Link News"), this);
    mastheadTitle->setObjectName(QStringLiteral("newsMastheadTitle"));
    mastheadText->addWidget(mastheadTitle);
    auto *mastheadSubtitle = new QLabel(QStringLiteral("Headlines from around the network"), this);
    mastheadSubtitle->setObjectName(QStringLiteral("newsMastheadSubtitle"));
    mastheadText->addWidget(mastheadSubtitle);
    masthead->addLayout(mastheadText);
    masthead->addStretch(1);
    root->addLayout(masthead);
    auto *toolbar = new QHBoxLayout;
    toolbar->setSpacing(6);

    auto *categoryLabel = new QLabel(QStringLiteral("Category:"), this);
    toolbar->addWidget(categoryLabel);

    m_categoryCombo = new QComboBox(this);
    m_categoryCombo->setMinimumWidth(140);
    m_categoryCombo->addItem(QStringLiteral("All"));
    toolbar->addWidget(m_categoryCombo);

    auto *sourceLabel = new QLabel(QStringLiteral("Source:"), this);
    toolbar->addWidget(sourceLabel);
    m_sourceCombo = new QComboBox(this);
    m_sourceCombo->setMinimumWidth(170);
    m_sourceCombo->addItem(QStringLiteral("All"), 0);
    toolbar->addWidget(m_sourceCombo);
    toolbar->addStretch(1);

    m_refreshButton = new QPushButton(QStringLiteral("Refresh"), this);
    m_refreshButton->setObjectName(QStringLiteral("newsRefreshButton"));
    toolbar->addWidget(m_refreshButton);
    root->addLayout(toolbar);

    auto *content = new QHBoxLayout;
    content->setSpacing(10);

    m_headlines = new QListWidget(this);
    m_headlines->setObjectName(QStringLiteral("newsHeadlines"));
    m_headlines->setMinimumWidth(290);
    m_headlines->setMaximumWidth(430);
    content->addWidget(m_headlines, 1);

    auto *article = new QVBoxLayout;
    article->setSpacing(6);

    m_title = new QLabel(QStringLiteral("Select a headline"), this);
    m_title->setObjectName(QStringLiteral("newsArticleTitle"));
    m_title->setWordWrap(true);
    article->addWidget(m_title);

    m_meta = new QLabel(this);
    m_meta->setObjectName(QStringLiteral("newsArticleMeta"));
    m_meta->setWordWrap(true);
    article->addWidget(m_meta);

    m_image = new QLabel(this);
    m_image->setObjectName(QStringLiteral("newsArticleImage"));
    m_image->setAlignment(Qt::AlignCenter);
    m_image->setMaximumSize(320, 180);
    m_image->setScaledContents(false);
    m_image->hide();
    article->addWidget(m_image, 0, Qt::AlignLeft);

    m_summary = new QLabel(this);
    m_summary->setObjectName(QStringLiteral("newsArticleSummary"));
    m_summary->setWordWrap(true);
    m_summary->setAlignment(Qt::AlignTop | Qt::AlignLeft);
    article->addWidget(m_summary, 1);

    m_originalButton = new QPushButton(QStringLiteral("Read Original"), this);
    m_originalButton->setObjectName(QStringLiteral("newsOriginalButton"));
    m_originalButton->setEnabled(false);
    article->addWidget(m_originalButton, 0, Qt::AlignLeft);

    content->addLayout(article, 2);
    root->addLayout(content, 1);

    connect(m_refreshButton, &QPushButton::clicked, this, &OtterNewsWidget::loadNews);
    connect(m_categoryCombo, &QComboBox::currentTextChanged,
            this, &OtterNewsWidget::loadNews);
    connect(m_sourceCombo, &QComboBox::currentIndexChanged,
            this, &OtterNewsWidget::loadNews);
    connect(m_headlines, &QListWidget::itemClicked,
            this, &OtterNewsWidget::showItem);
    connect(m_originalButton, &QPushButton::clicked,
            this, &OtterNewsWidget::openOriginal);
    connect(m_client, &OtterLinkClient::newsLoaded,
            this, &OtterNewsWidget::newsLoaded);
    connect(m_client, &OtterLinkClient::newsSourcesLoaded,
            this, &OtterNewsWidget::newsSourcesLoaded);
    connect(m_client, &OtterLinkClient::errorOccurred,
            this, [this](const QString &) {
                if (m_refreshButton)
                    m_refreshButton->setEnabled(true);
            });

    clearArticle();
    if (m_client)
        m_client->loadNewsSources();
    loadNews();
}

void OtterNewsWidget::loadNews()
{
    if (!m_client)
        return;
    const QString category = m_categoryCombo->currentText() == QStringLiteral("All")
        ? QString()
        : m_categoryCombo->currentText();
    const qint64 sourceId = m_sourceCombo->currentData().toLongLong();
    m_refreshButton->setEnabled(false);
    m_client->loadNews(100, category, sourceId);
}

void OtterNewsWidget::populateCategories(const QJsonArray &items)
{
    QString current = m_categoryCombo->currentText();
    QStringList categories;
    for (const QJsonValue &value : items) {
        const QString category = value.toObject().value(QStringLiteral("category")).toString().trimmed();
        if (!category.isEmpty() && !categories.contains(category, Qt::CaseInsensitive))
            categories << category;
    }
    categories.sort(Qt::CaseInsensitive);

    QSignalBlocker blocker(m_categoryCombo);
    m_categoryCombo->clear();
    m_categoryCombo->addItem(QStringLiteral("All"));
    m_categoryCombo->addItems(categories);
    int index = m_categoryCombo->findText(current, Qt::MatchFixedString);
    if (index < 0)
        index = 0;
    m_categoryCombo->setCurrentIndex(index);
}

void OtterNewsWidget::populateSources(const QJsonArray &sources)
{
    const qint64 current = m_sourceCombo->currentData().toLongLong();
    QSignalBlocker blocker(m_sourceCombo);
    m_sourceCombo->clear();
    m_sourceCombo->addItem(QStringLiteral("All"), 0);
    for (const QJsonValue &value : sources) {
        const QJsonObject source = value.toObject();
        const QString name = source.value(QStringLiteral("name")).toString().trimmed();
        const qint64 id = source.value(QStringLiteral("id")).toVariant().toLongLong();
        if (!name.isEmpty() && id > 0)
            m_sourceCombo->addItem(name, id);
    }
    const int index = m_sourceCombo->findData(current);
    m_sourceCombo->setCurrentIndex(index >= 0 ? index : 0);
}

void OtterNewsWidget::newsSourcesLoaded(const QJsonArray &sources)
{
    populateSources(sources);
}
void OtterNewsWidget::newsLoaded(const QJsonArray &items)
{
    m_refreshButton->setEnabled(true);
    if (m_categoryCombo->currentText() == QStringLiteral("All")
        && m_sourceCombo->currentData().toLongLong() == 0) {
        populateCategories(items);
    }

    m_headlines->clear();
    clearArticle();

    for (const QJsonValue &value : items) {
        const QJsonObject item = value.toObject();
        const QString title = item.value(QStringLiteral("title")).toString().trimmed();
        if (title.isEmpty())
            continue;

        auto *listItem = new QListWidgetItem(title, m_headlines);
        listItem->setData(Qt::UserRole, item);
        listItem->setToolTip(sourceLine(item));
    }

    if (m_headlines->count() > 0) {
        m_headlines->setCurrentRow(0);
        showItem(m_headlines->item(0));
    } else {
        m_title->setText(QStringLiteral("No news available"));
        m_meta->setText(QStringLiteral("There are no articles for these filters."));
    }
}

void OtterNewsWidget::showItem(QListWidgetItem *item)
{
    if (!item)
        return;
    const QJsonObject data = item->data(Qt::UserRole).toJsonObject();
    m_title->setText(data.value(QStringLiteral("title")).toString());
    m_meta->setText(sourceLine(data));
    m_summary->setText(data.value(QStringLiteral("summary")).toString().trimmed());
    const QString imageUrl = data.value(QStringLiteral("image_url")).toString().trimmed();
    m_image->clear();
    m_image->hide();
    if (!imageUrl.isEmpty())
        loadArticleImage(imageUrl);
    m_selectedUrl = data.value(QStringLiteral("url")).toString().trimmed();
    m_originalButton->setEnabled(!m_selectedUrl.isEmpty());
}

void OtterNewsWidget::loadArticleImage(const QString &url)
{
    if (!m_imageNetwork || url.isEmpty())
        return;

    QNetworkRequest request(QUrl(url));
    request.setHeader(QNetworkRequest::UserAgentHeader, QStringLiteral("OtterLink-News/1.0"));
    auto *reply = m_imageNetwork->get(request);
    connect(reply, &QNetworkReply::finished, this, [this, reply, url]() {
        const QByteArray data = reply->error() == QNetworkReply::NoError ? reply->readAll() : QByteArray();
        reply->deleteLater();
        if (url != m_headlines->currentItem()->data(Qt::UserRole).toJsonObject().value(QStringLiteral("image_url")).toString().trimmed())
            return;
        QPixmap pixmap;
        if (pixmap.loadFromData(data) && !pixmap.isNull()) {
            m_image->setPixmap(pixmap.scaled(320, 180, Qt::KeepAspectRatio, Qt::SmoothTransformation));
            m_image->show();
        }
    });
}

void OtterNewsWidget::openOriginal()
{
    if (!m_selectedUrl.isEmpty())
        QDesktopServices::openUrl(QUrl(m_selectedUrl));
}

void OtterNewsWidget::clearArticle()
{
    m_title->setText(QStringLiteral("Select a headline"));
    m_meta->clear();
    m_image->clear();
    m_image->hide();
    m_summary->clear();
    m_selectedUrl.clear();
    m_originalButton->setEnabled(false);
}
