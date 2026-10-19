#include "otterwebwidget.h"

#include <QAction>
#include <QDesktopServices>
#include <QHBoxLayout>
#include <QLineEdit>
#include <QMenu>
#include <QMessageBox>
#include <QPushButton>
#include <QSettings>
#include <QTextBrowser>
#include <QUrl>
#include <QVBoxLayout>

namespace {
const QUrl homeUrl(QStringLiteral("qrc:/otterweb-search.html"));
}

OtterWebWidget::OtterWebWidget(QWidget *parent)
    : QWidget(parent),
      m_view(new QTextBrowser(this)),
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
    m_view->setOpenExternalLinks(false);
    m_view->setOpenLinks(false);
    m_view->setReadOnly(true);
    root->addWidget(m_view, 1);

    QSettings settings(QStringLiteral("OtterLink"), QStringLiteral("OtterLink"));
    m_bookmarks = settings.value(QStringLiteral("otterweb/bookmarks")).toStringList();
    rebuildBookmarksMenu();

    connect(m_backButton, &QPushButton::clicked, m_view, &QTextBrowser::backward);
    connect(m_forwardButton, &QPushButton::clicked, m_view, &QTextBrowser::forward);
    connect(m_homeButton, &QPushButton::clicked, this, [this]() {
        m_view->setSource(homeUrl);
    });
    connect(m_goButton, &QPushButton::clicked, this, &OtterWebWidget::navigateToAddress);
    connect(m_address, &QLineEdit::returnPressed, this, &OtterWebWidget::navigateToAddress);
    connect(m_bookmarkButton, &QPushButton::clicked, this, &OtterWebWidget::addBookmark);
    connect(m_view, &QTextBrowser::sourceChanged, this, [this](const QUrl &url) {
        m_address->setText(url.toString());
        updateNavigationState();
    });
    connect(m_view, &QTextBrowser::anchorClicked, this, [this](const QUrl &url) {
        if (url.scheme() == QStringLiteral("http") || url.scheme() == QStringLiteral("https")) {
            QDesktopServices::openUrl(url);
            return;
        }
        if (url.scheme() == QStringLiteral("qrc") || url.scheme().isEmpty())
            m_view->setSource(url);
    });
    connect(m_view, &QTextBrowser::backwardAvailable, m_backButton, &QPushButton::setEnabled);
    connect(m_view, &QTextBrowser::forwardAvailable, m_forwardButton, &QPushButton::setEnabled);

    m_view->setSource(homeUrl);
    updateNavigationState();
}

void OtterWebWidget::navigateToAddress()
{
    QString text = m_address->text().trimmed();
    if (text.isEmpty())
        return;

    QUrl url = QUrl::fromUserInput(text);
    if (!url.isValid() || url.scheme().isEmpty()) {
        QMessageBox::warning(this, QStringLiteral("OtterWeb"),
                             QStringLiteral("Enter a valid web address."));
        return;
    }

    if (url.scheme() == QStringLiteral("http") || url.scheme() == QStringLiteral("https")) {
        // Qt WebEngine is not available in the project's MinGW cross-build kit.
        // Keep OtterWeb buildable and hand public websites to the system browser.
        if (!QDesktopServices::openUrl(url)) {
            QMessageBox::warning(this, QStringLiteral("OtterWeb"),
                                 QStringLiteral("The system browser could not open this address."));
        }
        return;
    }

    if (url.scheme() == QStringLiteral("qrc") &&
        url.path() == QStringLiteral("/otterweb-search.html")) {
        m_view->setSource(url);
        return;
    }

    QMessageBox::warning(this, QStringLiteral("OtterWeb"),
                         QStringLiteral("OtterWeb currently displays its bundled internal pages. "
                                        "HTTP and HTTPS sites open in your system browser."));
}

void OtterWebWidget::updateNavigationState()
{
    if (!m_view)
        return;
    m_backButton->setEnabled(m_view->isBackwardAvailable());
    m_forwardButton->setEnabled(m_view->isForwardAvailable());
}

void OtterWebWidget::addBookmark()
{
    const QUrl url = m_view->source();
    if (!url.isValid() || url.scheme() != QStringLiteral("qrc")) {
        QMessageBox::information(this, QStringLiteral("OtterWeb"),
                                 QStringLiteral("Open an internal OtterWeb page before bookmarking it."));
        return;
    }

    const QString entry = m_view->documentTitle().trimmed().isEmpty()
        ? url.toString()
        : m_view->documentTitle().trimmed() + QStringLiteral(" | ") + url.toString();
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
            const QUrl url(address);
            if (url.scheme() == QStringLiteral("http") || url.scheme() == QStringLiteral("https"))
                QDesktopServices::openUrl(url);
            else if (url.scheme() == QStringLiteral("qrc"))
                m_view->setSource(url);
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
