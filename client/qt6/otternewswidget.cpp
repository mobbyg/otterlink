#include "otternewswidget.h"

#include "otterlinkclient.h"

#include <QComboBox>
#include <QDateTime>
#include <QDesktopServices>
#include <QFrame>
#include <QGridLayout>
#include <QHBoxLayout>
#include <QJsonArray>
#include <QJsonObject>
#include <QJsonValue>
#include <QLabel>
#include <QListWidget>
#include <QListWidgetItem>
#include <QNetworkAccessManager>
#include <QNetworkReply>
#include <QNetworkRequest>
#include <QPixmap>
#include <QPointer>
#include <QPushButton>
#include <QRegularExpression>
#include <QScrollArea>
#include <QSet>
#include <QStackedWidget>
#include <QStyle>
#include <QSignalBlocker>
#include <QStringList>
#include <QTextBrowser>
#include <QTextDocument>
#include <QUrl>
#include <QVBoxLayout>
#include <QWidget>

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

QLabel *makeImageLabel(QWidget *parent, int width, int height)
{
    auto *label = new QLabel(parent);
    label->setAlignment(Qt::AlignCenter);
    label->setFixedSize(width, height);
    label->setObjectName(QStringLiteral("newsCardImage"));
    label->setText(QStringLiteral("NEWS"));
    label->setProperty("placeholder", true);
    return label;
}

} // namespace

OtterNewsWidget::OtterNewsWidget(OtterLinkClient *client, QWidget *parent)
    : QWidget(parent),
      m_client(client),
      m_imageNetwork(new QNetworkAccessManager(this))
{
    auto *root = new QVBoxLayout(this);
    root->setContentsMargins(10, 8, 10, 10);
    root->setSpacing(6);

    // Newspaper masthead.
    auto *masthead = new QHBoxLayout;
    masthead->setSpacing(8);

    auto *newsIcon = new QLabel(this);
    newsIcon->setPixmap(QPixmap(QStringLiteral(":/images/news.png")).scaled(
        34, 34, Qt::KeepAspectRatio, Qt::SmoothTransformation));
    newsIcon->setFixedSize(34, 34);
    masthead->addWidget(newsIcon);

    auto *mastheadText = new QVBoxLayout;
    mastheadText->setSpacing(0);
    auto *mastheadTitle = new QLabel(QStringLiteral("OTTER LINK NEWS"), this);
    mastheadTitle->setObjectName(QStringLiteral("newsMastheadTitle"));
    mastheadText->addWidget(mastheadTitle);
    auto *mastheadSubtitle = new QLabel(
        QStringLiteral("The day's headlines from around the network"), this);
    mastheadSubtitle->setObjectName(QStringLiteral("newsMastheadSubtitle"));
    mastheadText->addWidget(mastheadSubtitle);
    masthead->addLayout(mastheadText);
    masthead->addStretch(1);

    m_dateLabel = new QLabel(this);
    m_dateLabel->setObjectName(QStringLiteral("newsDateLabel"));
    m_dateLabel->setText(QDateTime::currentDateTime().toString(QStringLiteral("dddd, MMMM d, yyyy")));
    masthead->addWidget(m_dateLabel, 0, Qt::AlignBottom);
    root->addLayout(masthead);

    auto *rule = new QFrame(this);
    rule->setObjectName(QStringLiteral("newsMastheadRule"));
    rule->setFrameShape(QFrame::HLine);
    rule->setFrameShadow(QFrame::Plain);
    root->addWidget(rule);

    // Navigation/filter strip. This keeps the existing source/category model,
    // but visually reads more like newspaper sections than an RSS control panel.
    auto *toolbar = new QHBoxLayout;
    toolbar->setSpacing(5);

    auto *sectionLabel = new QLabel(QStringLiteral("SECTION"), this);
    sectionLabel->setObjectName(QStringLiteral("newsSectionLabel"));
    toolbar->addWidget(sectionLabel);

    m_categoryCombo = new QComboBox(this);
    m_categoryCombo->setMinimumWidth(125);
    m_categoryCombo->addItem(QStringLiteral("All"));
    toolbar->addWidget(m_categoryCombo);

    auto *sourceLabel = new QLabel(QStringLiteral("SOURCE"), this);
    sourceLabel->setObjectName(QStringLiteral("newsSectionLabel"));
    toolbar->addWidget(sourceLabel);

    m_sourceCombo = new QComboBox(this);
    m_sourceCombo->setMinimumWidth(150);
    m_sourceCombo->addItem(QStringLiteral("All"), 0);
    toolbar->addWidget(m_sourceCombo);

    toolbar->addStretch(1);

    m_refreshButton = new QPushButton(QStringLiteral("Refresh"), this);
    m_refreshButton->setObjectName(QStringLiteral("newsRefreshButton"));
    toolbar->addWidget(m_refreshButton);
    root->addLayout(toolbar);

    // Front page.
    m_stack = new QStackedWidget(this);

    auto *frontPage = new QWidget(this);
    auto *frontLayout = new QVBoxLayout(frontPage);
    frontLayout->setContentsMargins(0, 2, 0, 0);
    frontLayout->setSpacing(8);

    auto *topStories = new QHBoxLayout;
    topStories->setSpacing(10);

    auto *featuredFrame = new QFrame(frontPage);
    featuredFrame->setObjectName(QStringLiteral("newsFeatured"));
    auto *featuredLayout = new QVBoxLayout(featuredFrame);
    featuredLayout->setContentsMargins(8, 8, 8, 8);
    featuredLayout->setSpacing(5);

    auto *featuredLabel = new QLabel(QStringLiteral("TOP STORY"), featuredFrame);
    featuredLabel->setObjectName(QStringLiteral("newsSectionHeading"));
    featuredLayout->addWidget(featuredLabel);

    m_featuredImage = makeImageLabel(featuredFrame, 250, 135);
    featuredLayout->addWidget(m_featuredImage, 0, Qt::AlignHCenter);

    m_featuredTitle = new QPushButton(QStringLiteral("Select a headline"), featuredFrame);
    m_featuredTitle->setObjectName(QStringLiteral("newsFeaturedTitle"));
    m_featuredTitle->setFlat(true);
    m_featuredTitle->setCursor(Qt::PointingHandCursor);
    m_featuredTitle->setMinimumHeight(52);
    featuredLayout->addWidget(m_featuredTitle);

    m_featuredMeta = new QLabel(featuredFrame);
    m_featuredMeta->setObjectName(QStringLiteral("newsFeaturedMeta"));
    m_featuredMeta->setWordWrap(true);
    featuredLayout->addWidget(m_featuredMeta);

    m_featuredSummary = new QLabel(featuredFrame);
    m_featuredSummary->setObjectName(QStringLiteral("newsFeaturedSummary"));
    m_featuredSummary->setWordWrap(true);
    featuredLayout->addWidget(m_featuredSummary);

    topStories->addWidget(featuredFrame, 3);

    auto *latestFrame = new QFrame(frontPage);
    latestFrame->setObjectName(QStringLiteral("newsLatest"));
    auto *latestLayout = new QVBoxLayout(latestFrame);
    latestLayout->setContentsMargins(8, 8, 8, 8);
    latestLayout->setSpacing(5);

    auto *latestLabel = new QLabel(QStringLiteral("LATEST HEADLINES"), latestFrame);
    latestLabel->setObjectName(QStringLiteral("newsSectionHeading"));
    latestLayout->addWidget(latestLabel);

    m_headlines = new QListWidget(latestFrame);
    m_headlines->setObjectName(QStringLiteral("newsHeadlines"));
    m_headlines->setMinimumWidth(270);
    m_headlines->setSpacing(2);
    latestLayout->addWidget(m_headlines, 1);

    topStories->addWidget(latestFrame, 2);
    frontLayout->addLayout(topStories, 1);

    auto *frontFooter = new QLabel(
        QStringLiteral("Select a headline to read the story. Otter Link News links back to the original publisher."),
        frontPage);
    frontFooter->setObjectName(QStringLiteral("newsFrontFooter"));
    frontFooter->setWordWrap(true);
    frontLayout->addWidget(frontFooter);

    m_stack->addWidget(frontPage);

    // Article reader.
    auto *articlePage = new QWidget(this);
    auto *articleLayout = new QVBoxLayout(articlePage);
    articleLayout->setContentsMargins(0, 0, 0, 0);
    articleLayout->setSpacing(5);

    auto *articleToolbar = new QHBoxLayout;
    articleToolbar->setSpacing(5);

    m_backButton = new QPushButton(QStringLiteral("← Headlines"), articlePage);
    m_backButton->setObjectName(QStringLiteral("newsBackButton"));
    articleToolbar->addWidget(m_backButton);
    articleToolbar->addStretch(1);

    m_originalButton = new QPushButton(QStringLiteral("Read Original"), articlePage);
    m_originalButton->setObjectName(QStringLiteral("newsOriginalButton"));
    m_originalButton->setEnabled(false);
    articleToolbar->addWidget(m_originalButton);
    articleLayout->addLayout(articleToolbar);

    m_title = new QLabel(QStringLiteral("Select a headline"), articlePage);
    m_title->setObjectName(QStringLiteral("newsArticleTitle"));
    m_title->setWordWrap(true);
    articleLayout->addWidget(m_title);

    m_meta = new QLabel(articlePage);
    m_meta->setObjectName(QStringLiteral("newsArticleMeta"));
    m_meta->setWordWrap(true);
    articleLayout->addWidget(m_meta);

    m_article = new QTextBrowser(articlePage);
    m_article->setObjectName(QStringLiteral("newsArticle"));
    m_article->setReadOnly(true);
    m_article->setOpenLinks(false);
    m_article->setOpenExternalLinks(false);
    articleLayout->addWidget(m_article, 1);

    m_stack->addWidget(articlePage);
    root->addWidget(m_stack, 1);

    connect(m_refreshButton, &QPushButton::clicked, this, &OtterNewsWidget::loadNews);
    connect(m_categoryCombo, &QComboBox::currentTextChanged,
            this, &OtterNewsWidget::loadNews);
    connect(m_sourceCombo, &QComboBox::currentIndexChanged,
            this, &OtterNewsWidget::loadNews);
    connect(m_headlines, &QListWidget::itemClicked,
            this, &OtterNewsWidget::showItem);
    connect(m_featuredTitle, &QPushButton::clicked, this, [this]() {
        if (!m_featuredItem.isEmpty())
            showItemData(m_featuredItem);
    });
    connect(m_backButton, &QPushButton::clicked, this, [this]() {
        m_stack->setCurrentIndex(0);
    });
    connect(m_originalButton, &QPushButton::clicked,
            this, &OtterNewsWidget::openOriginal);
    connect(m_article, &QTextBrowser::anchorClicked, this,
            [](const QUrl &url) { QDesktopServices::openUrl(url); });
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
    const QString current = m_categoryCombo->currentText();
    QStringList categories;

    for (const QJsonValue &value : items) {
        const QString category = value.toObject()
            .value(QStringLiteral("category")).toString().trimmed();
        if (!category.isEmpty() && !categories.contains(category, Qt::CaseInsensitive))
            categories << category;
    }

    categories.sort(Qt::CaseInsensitive);

    QSignalBlocker blocker(m_categoryCombo);
    m_categoryCombo->clear();
    m_categoryCombo->addItem(QStringLiteral("All"));
    m_categoryCombo->addItems(categories);

    const int index = m_categoryCombo->findText(current, Qt::MatchFixedString);
    m_categoryCombo->setCurrentIndex(index >= 0 ? index : 0);
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
    m_featuredItem = QJsonObject();
    m_featuredImage->clear();
    m_featuredImage->setText(QStringLiteral("NEWS"));
    m_featuredSummary->clear();
    m_featuredMeta->clear();
    m_featuredTitle->setText(QStringLiteral("Select a headline"));
    clearArticle();

    int validCount = 0;
    for (const QJsonValue &value : items) {
        const QJsonObject item = value.toObject();
        if (item.value(QStringLiteral("title")).toString().trimmed().isEmpty())
            continue;

        if (validCount == 0) {
            m_featuredItem = item;
            m_featuredTitle->setText(item.value(QStringLiteral("title")).toString().trimmed());
            m_featuredMeta->setText(sourceLine(item));
            m_featuredSummary->setText(item.value(QStringLiteral("summary")).toString().trimmed());
            const QString imageUrl = item.value(QStringLiteral("image_url")).toString().trimmed();
            if (!imageUrl.isEmpty())
                loadImageIntoLabel(imageUrl, m_featuredImage, 250, 135);
        }

        auto *listItem = new QListWidgetItem;
        listItem->setData(Qt::UserRole, item);
        listItem->setToolTip(sourceLine(item));

        auto *card = new QWidget(m_headlines);
        auto *cardLayout = new QHBoxLayout(card);
        cardLayout->setContentsMargins(5, 5, 5, 5);
        cardLayout->setSpacing(7);

        const QString imageUrl = item.value(QStringLiteral("image_url")).toString().trimmed();
        auto *image = makeImageLabel(card, 86, 58);
        if (!imageUrl.isEmpty())
            loadImageIntoLabel(imageUrl, image, 86, 58);
        cardLayout->addWidget(image);

        auto *text = new QVBoxLayout;
        text->setContentsMargins(0, 0, 0, 0);
        text->setSpacing(2);

        auto *title = new QLabel(item.value(QStringLiteral("title")).toString().trimmed(), card);
        title->setObjectName(QStringLiteral("newsCardTitle"));
        title->setWordWrap(true);
        text->addWidget(title);

        auto *meta = new QLabel(sourceLine(item), card);
        meta->setObjectName(QStringLiteral("newsCardMeta"));
        meta->setWordWrap(true);
        text->addWidget(meta);
        text->addStretch(1);

        cardLayout->addLayout(text, 1);
        card->setMinimumHeight(70);
        m_headlines->addItem(listItem);
        m_headlines->setItemWidget(listItem, card);
        listItem->setSizeHint(card->sizeHint());
        ++validCount;
    }

    if (validCount == 0) {
        m_featuredTitle->setText(QStringLiteral("No news available"));
        m_featuredSummary->setText(QStringLiteral("There are no stories for these sections yet."));
        m_headlines->addItem(QStringLiteral("No headlines available."));
    }
}

void OtterNewsWidget::showItem(QListWidgetItem *item)
{
    if (!item)
        return;

    const QJsonObject data = item->data(Qt::UserRole).toJsonObject();
    if (!data.isEmpty())
        showItemData(data);
}

void OtterNewsWidget::showItemData(const QJsonObject &data)
{
    if (data.isEmpty())
        return;

    m_title->setText(data.value(QStringLiteral("title")).toString());
    m_meta->setText(sourceLine(data));
    m_selectedUrl = data.value(QStringLiteral("url")).toString().trimmed();
    m_selectedImageUrl = data.value(QStringLiteral("image_url")).toString().trimmed();

    QString articleHtml = data.value(QStringLiteral("article_html")).toString().trimmed();
    if (articleHtml.isEmpty()) {
        articleHtml = QStringLiteral("<p>%1</p>")
            .arg(data.value(QStringLiteral("summary")).toString().toHtmlEscaped());
    }

    if (!m_selectedImageUrl.isEmpty() && !articleHtml.contains(m_selectedImageUrl)) {
        articleHtml.prepend(QStringLiteral("<p><img src="%1"></p>")
            .arg(m_selectedImageUrl.toHtmlEscaped()));
    }

    m_article->setHtml(articleHtml);
    loadArticleImages(articleHtml, m_selectedUrl);
    m_originalButton->setEnabled(!m_selectedUrl.isEmpty());
    m_stack->setCurrentIndex(1);
}

void OtterNewsWidget::loadImageIntoLabel(const QString &url, QLabel *label,
                                         int maxWidth, int maxHeight)
{
    if (!m_imageNetwork || !label || url.isEmpty())
        return;

    QPointer<QLabel> target(label);
    QNetworkRequest request{QUrl(url)};
    request.setHeader(QNetworkRequest::UserAgentHeader, QStringLiteral("OtterLink-News/1.0"));

    auto *reply = m_imageNetwork->get(request);
    connect(reply, &QNetworkReply::finished, this,
            [reply, target, url, maxWidth, maxHeight]() {
        const QByteArray data =
            reply->error() == QNetworkReply::NoError ? reply->readAll() : QByteArray();
        reply->deleteLater();

        if (!target || data.isEmpty())
            return;

        QPixmap pixmap;
        if (!pixmap.loadFromData(data) || pixmap.isNull())
            return;

        pixmap = pixmap.scaled(maxWidth, maxHeight,
                               Qt::KeepAspectRatio, Qt::SmoothTransformation);
        target->setPixmap(pixmap);
        target->setText(QString());
        target->setProperty("placeholder", false);
        target->style()->unpolish(target);
        target->style()->polish(target);
    });
}

void OtterNewsWidget::loadArticleImages(const QString &html, const QString &articleUrl)
{
    if (!m_imageNetwork || html.isEmpty())
        return;

    static const QRegularExpression imagePattern(
        QStringLiteral("<img[^>]+src=[\"']([^\"']+)[\"'][^>]*>"),
        QRegularExpression::CaseInsensitiveOption);

    QSet<QString> urls;
    auto match = imagePattern.globalMatch(html);
    while (match.hasNext()) {
        const QString url = match.next().captured(1).trimmed();
        if (!url.isEmpty())
            urls.insert(url);
    }

    for (const QString &url : urls) {
        QNetworkRequest request{QUrl(url)};
        request.setHeader(QNetworkRequest::UserAgentHeader, QStringLiteral("OtterLink-News/1.0"));

        auto *reply = m_imageNetwork->get(request);
        connect(reply, &QNetworkReply::finished, this,
                [this, reply, url, articleUrl]() {
            const QByteArray data =
                reply->error() == QNetworkReply::NoError ? reply->readAll() : QByteArray();
            reply->deleteLater();

            if (articleUrl != m_selectedUrl || data.isEmpty())
                return;

            QPixmap pixmap;
            if (!pixmap.loadFromData(data) || pixmap.isNull())
                return;

            const int maxWidth = qMax(320, m_article->viewport()->width() - 24);
            const int maxHeight = 600;
            if (pixmap.width() > maxWidth || pixmap.height() > maxHeight) {
                pixmap = pixmap.scaled(maxWidth, maxHeight,
                                       Qt::KeepAspectRatio, Qt::SmoothTransformation);
            }

            m_article->document()->addResource(
                QTextDocument::ImageResource, QUrl(url), QVariant::fromValue(pixmap));
            m_article->document()->markContentsDirty(
                0, m_article->document()->characterCount());
            m_article->viewport()->update();
        });
    }
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
    m_article->clear();
    m_selectedUrl.clear();
    m_selectedImageUrl.clear();
    m_originalButton->setEnabled(false);
    m_stack->setCurrentIndex(0);
}
