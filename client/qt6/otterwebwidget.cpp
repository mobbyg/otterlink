#include "otterwebwidget.h"

#include <QAction>
#include <QDesktopServices>
#include <QHBoxLayout>
#include <QLineEdit>
#include <QMenu>
#include <QMessageBox>
#include <QPushButton>
#include <QSettings>
#include <QUrl>
#include <QVBoxLayout>

#ifdef OTTERLINK_USE_WEBENGINE
#include <QWebEngineHistory>
#include <QWebEngineView>
#else
#include <QTextBrowser>
#endif

namespace {
const QUrl homeUrl(QStringLiteral("qrc:/otterweb-search.html"));

bool isWebUrl(const QUrl &url)
{
    return url.scheme() == QStringLiteral("http") ||
           url.scheme() == QStringLiteral("https");
}

bool isBundledHomeUrl(const QUrl &url)
{
    return url.scheme() == QStringLiteral("qrc") &&
           url.path() == QStringLiteral("/otterweb-search.html");
}

bool isSearchHomeUrl(const QUrl &url)
{
    return isBundledHomeUrl(url) ||
           (url.scheme().compare(QStringLiteral("otterlink"), Qt::CaseInsensitive) == 0 &&
            url.host().compare(QStringLiteral("search"), Qt::CaseInsensitive) == 0 &&
            url.path().isEmpty());
}

QString addressForUrl(const QUrl &url)
{
    return isBundledHomeUrl(url) ? QStringLiteral("otterlink://search") : url.toString();
}
}

OtterWebWidget::OtterWebWidget(QWidget *parent)
    : QWidget(parent),
#ifdef OTTERLINK_USE_WEBENGINE
      m_view(new QWebEngineView(this)),
#else
      m_view(new QTextBrowser(this)),
#endif
      m_address(new QLineEdit(this)),
      m_backButton(new QPushButton(QStringLiteral("←"), this)),
      m_forwardButton(new QPushButton(QStringLiteral("→"), this)),
      m_homeButton(new QPushButton(QStringLiteral("Home"), this)),
      m_goButton(new QPushButton(QStringLiteral("Go"), this)),
      m_bookmarkButton(new QPushButton(QStringLiteral("★"), this)),
      m_bookmarksMenu(new QMenu(this))
{
    setObjectName(QStringLiteral("otterWebWidget"));
    auto *root = new QVBoxLayout(this);
    root->setContentsMargins(8, 8, 8, 8);
    root->setSpacing(6);

    auto *toolbar = new QHBoxLayout;
    toolbar->setSpacing(4);
    m_backButton->setToolTip(QStringLiteral("Back"));
    m_forwardButton->setToolTip(QStringLiteral("Forward"));
    m_bookmarkButton->setToolTip(QStringLiteral("Bookmark current page"));
    m_address->setObjectName(QStringLiteral("otterWebAddress"));
    m_address->setPlaceholderText(QStringLiteral("Enter a web address"));
    m_address->setClearButtonEnabled(true);

    for (QPushButton *button : {m_backButton, m_forwardButton, m_homeButton}) {
        button->setMinimumHeight(30);
        toolbar->addWidget(button);
    }
    toolbar->addWidget(m_address, 1);
    for (QPushButton *button : {m_goButton, m_bookmarkButton}) {
        button->setMinimumHeight(30);
        toolbar->addWidget(button);
    }
    auto *bookmarkMenuButton = new QPushButton(QStringLiteral("Bookmarks ▾"), this);
    bookmarkMenuButton->setMinimumHeight(30);
    bookmarkMenuButton->setMenu(m_bookmarksMenu);
    toolbar->addWidget(bookmarkMenuButton);
    root->addLayout(toolbar);

    m_view->setObjectName(QStringLiteral("otterWebPage"));
#ifndef OTTERLINK_USE_WEBENGINE
    m_view->setOpenExternalLinks(false);
    m_view->setOpenLinks(false);
    m_view->setReadOnly(true);
#endif
    root->addWidget(m_view, 1);

    QSettings settings(QStringLiteral("OtterLink"), QStringLiteral("OtterLink"));
    m_bookmarks = settings.value(QStringLiteral("otterweb/bookmarks")).toStringList();
    rebuildBookmarksMenu();

#ifdef OTTERLINK_USE_WEBENGINE
    connect(m_backButton, &QPushButton::clicked, m_view, &QWebEngineView::back);
    connect(m_forwardButton, &QPushButton::clicked, m_view, &QWebEngineView::forward);
    connect(m_view, &QWebEngineView::urlChanged, this, [this](const QUrl &url) {
        m_address->setText(addressForUrl(url));
        updateNavigationState();
    });
    connect(m_view, &QWebEngineView::titleChanged, this, [this](const QString &) {
        updateNavigationState();
    });
    connect(m_view, &QWebEngineView::loadFinished, this, [this](bool) {
        updateNavigationState();
    });
#else
    connect(m_backButton, &QPushButton::clicked, m_view, &QTextBrowser::backward);
    connect(m_forwardButton, &QPushButton::clicked, m_view, &QTextBrowser::forward);
    connect(m_view, &QTextBrowser::sourceChanged, this, [this](const QUrl &url) {
        m_address->setText(addressForUrl(url));
        updateNavigationState();
    });
    connect(m_view, &QTextBrowser::anchorClicked, this, [this](const QUrl &url) {
        if (isWebUrl(url)) {
            QDesktopServices::openUrl(url);
            return;
        }
        if (url.scheme() == QStringLiteral("qrc") || url.scheme().isEmpty())
            loadInView(url);
    });
    connect(m_view, &QTextBrowser::backwardAvailable, m_backButton, &QPushButton::setEnabled);
    connect(m_view, &QTextBrowser::forwardAvailable, m_forwardButton, &QPushButton::setEnabled);
#endif

    connect(m_homeButton, &QPushButton::clicked, this, [this]() {
        loadInView(homeUrl);
    });
    connect(m_goButton, &QPushButton::clicked, this, &OtterWebWidget::navigateToAddress);
    connect(m_address, &QLineEdit::returnPressed, this, &OtterWebWidget::navigateToAddress);
    connect(m_bookmarkButton, &QPushButton::clicked, this, &OtterWebWidget::addBookmark);

    loadInView(homeUrl);
    updateNavigationState();
}

void OtterWebWidget::loadInView(const QUrl &url)
{
    const QUrl target = isSearchHomeUrl(url) ? homeUrl : url;
#ifdef OTTERLINK_USE_WEBENGINE
    m_view->load(target);
#else
    if (isBundledHomeUrl(target) || target.scheme().isEmpty())
        m_view->setSource(target);
    else if (isWebUrl(target))
        QDesktopServices::openUrl(target);
#endif
}

void OtterWebWidget::navigateToAddress()
{
    const QString text = m_address->text().trimmed();
    if (text.isEmpty())
        return;

    const QUrl url = QUrl::fromUserInput(text);
    if (isSearchHomeUrl(url)) {
        loadInView(homeUrl);
        return;
    }
    if (!url.isValid() || url.scheme().isEmpty()) {
        QMessageBox::warning(this, QStringLiteral("OtterWeb"),
                             QStringLiteral("Enter a valid web address."));
        return;
    }

    if (isWebUrl(url)) {
#ifdef OTTERLINK_USE_WEBENGINE
        loadInView(url);
#else
        if (!QDesktopServices::openUrl(url)) {
            QMessageBox::warning(this, QStringLiteral("OtterWeb"),
                                 QStringLiteral("The system browser could not open this address."));
        }
#endif
        return;
    }

    QMessageBox::warning(this, QStringLiteral("OtterWeb"),
                         QStringLiteral("Only HTTP/HTTPS addresses and bundled OtterWeb pages are supported."));
}

void OtterWebWidget::updateNavigationState()
{
    if (!m_view)
        return;
#ifdef OTTERLINK_USE_WEBENGINE
    const auto *history = m_view->history();
    m_backButton->setEnabled(history && history->canGoBack());
    m_forwardButton->setEnabled(history && history->canGoForward());
#else
    m_backButton->setEnabled(m_view->isBackwardAvailable());
    m_forwardButton->setEnabled(m_view->isForwardAvailable());
#endif
}

void OtterWebWidget::addBookmark()
{
    const QUrl url = QUrl::fromUserInput(m_address->text().trimmed());
    if (!url.isValid() || url.scheme().isEmpty()) {
        QMessageBox::information(this, QStringLiteral("OtterWeb"),
                                 QStringLiteral("Enter a valid address before bookmarking it."));
        return;
    }
    if (!isWebUrl(url) && !isSearchHomeUrl(url)) {
        QMessageBox::information(this, QStringLiteral("OtterWeb"),
                                 QStringLiteral("Only HTTP/HTTPS addresses or internal OtterWeb pages can be bookmarked."));
        return;
    }

    QString title;
#ifdef OTTERLINK_USE_WEBENGINE
    title = m_view->title().trimmed();
#else
    title = m_view->documentTitle().trimmed();
#endif
    if (isWebUrl(url) && title.isEmpty())
        title = url.host();
    const QString entry = title.isEmpty()
        ? url.toString()
        : title + QStringLiteral(" | ") + url.toString();
    if (!m_bookmarks.contains(entry))
        m_bookmarks.append(entry);
    QSettings settings(QStringLiteral("OtterLink"), QStringLiteral("OtterLink"));
    settings.setValue(QStringLiteral("otterweb/bookmarks"), m_bookmarks);
    rebuildBookmarksMenu();
}

void OtterWebWidget::rebuildBookmarksMenu()
{
    m_bookmarksMenu->clear();
    if (m_bookmarks.isEmpty()) {
        QAction *empty = m_bookmarksMenu->addAction(QStringLiteral("No bookmarks yet"));
        empty->setEnabled(false);
        return;
    }

    for (const QString &entry : m_bookmarks) {
        const int separator = entry.indexOf(QStringLiteral(" | "));
        const QString label = separator >= 0 ? entry.left(separator) : entry;
        const QString address = separator >= 0 ? entry.mid(separator + 3) : entry;
        QAction *action = m_bookmarksMenu->addAction(label);
        action->setToolTip(address);
        connect(action, &QAction::triggered, this, [this, address]() {
            const QUrl url = QUrl::fromUserInput(address);
            if (isWebUrl(url) || isSearchHomeUrl(url))
                loadInView(url);
        });
    }

    m_bookmarksMenu->addSeparator();
    QAction *clear = m_bookmarksMenu->addAction(QStringLiteral("Clear Bookmarks"));
    connect(clear, &QAction::triggered, this, [this]() {
        const auto answer = QMessageBox::question(
            this, QStringLiteral("Clear Bookmarks"),
            QStringLiteral("Remove all saved OtterWeb bookmarks?"));
        if (answer != QMessageBox::Yes)
            return;
        m_bookmarks.clear();
        QSettings settings(QStringLiteral("OtterLink"), QStringLiteral("OtterLink"));
        settings.remove(QStringLiteral("otterweb/bookmarks"));
        rebuildBookmarksMenu();
    });
}
