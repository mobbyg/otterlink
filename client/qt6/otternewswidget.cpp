#include "otternewswidget.h"

#include "otterlinkclient.h"

#include <QComboBox>
#include <QDesktopServices>
#include <QHBoxLayout>
#include <QJsonArray>
#include <QJsonObject>
#include <QLabel>
#include <QListWidget>
#include <QListWidgetItem>
#include <QPushButton>
#include <QUrl>
#include <QVBoxLayout>
#include <QDateTime>

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
      m_client(client)
{
    auto *root = new QVBoxLayout(this);
    root->setContentsMargins(12, 12, 12, 12);
    root->setSpacing(8);

    auto *toolbar = new QHBoxLayout;
    toolbar->setSpacing(6);

    auto *categoryLabel = new QLabel(QStringLiteral("Category:"), this);
    toolbar->addWidget(categoryLabel);

    m_categoryCombo = new QComboBox(this);
    m_categoryCombo->setMinimumWidth(140);
    m_categoryCombo->addItem(QStringLiteral("All"));
    toolbar->addWidget(m_categoryCombo);
    toolbar->addStretch(1);

    m_refreshButton = new QPushButton(QStringLiteral("Refresh"), this);
    m_refreshButton->setObjectName(QStringLiteral("newsRefreshButton"));
    toolbar->addWidget(m_refreshButton);
    root->addLayout(toolbar);

    auto *content = new QHBoxLayout;
    content->setSpacing(10);

    m_headlines = new QListWidget(this);
    m_headlines->setObjectName(QStringLiteral("newsHeadlines"));
    m_headlines->setMinimumWidth(250);
    m_headlines->setMaximumWidth(360);
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
    connect(m_headlines, &QListWidget::itemClicked,
            this, &OtterNewsWidget::showItem);
    connect(m_originalButton, &QPushButton::clicked,
            this, &OtterNewsWidget::openOriginal);
    connect(m_client, &OtterLinkClient::newsLoaded,
            this, &OtterNewsWidget::newsLoaded);

    clearArticle();
    loadNews();
}

void OtterNewsWidget::loadNews()
{
    if (!m_client)
        return;
    const QString category = m_categoryCombo->currentText() == QStringLiteral("All")
        ? QString()
        : m_categoryCombo->currentText();
    m_refreshButton->setEnabled(false);
    m_client->loadNews(100, category);
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

void OtterNewsWidget::newsLoaded(const QJsonArray &items)
{
    m_refreshButton->setEnabled(true);
    populateCategories(items);

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
        m_meta->setText(QStringLiteral("There are no articles for this category."));
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
    m_selectedUrl = data.value(QStringLiteral("url")).toString().trimmed();
    m_originalButton->setEnabled(!m_selectedUrl.isEmpty());
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
    m_summary->clear();
    m_selectedUrl.clear();
    m_originalButton->setEnabled(false);
}
