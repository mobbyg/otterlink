#include "otterwebwidget.h"

#include <QDesktopServices>
#include <QHBoxLayout>
#include <QIcon>
#include <QLineEdit>
#include <QMenu>
#include <QMessageBox>
#include <QPushButton>
#include <QSettings>
#include <QUrl>
#include <QVBoxLayout>
#include <QWebEngineHistory>
#include <QWebEngineView>

OtterWebWidget::OtterWebWidget(QWidget *parent)
    : QWidget(parent),
      m_view(new QWebEngineView(this)),
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
    m_bookmarkButton->setToolTip(QStringLiteral("Add current page to bookmarks"));
    m_address->setObjectName(QStringLiteral("otterWebAddress"));
    m_address->setPlaceholderText(QStringLiteral("Enter a web address"));
    m_address->setClearButtonEnabled(true);

    for (QPushButton *button : {m_backButton, m_forwardButton, m_homeButton,
                                m_goButton, m_bookmarkButton}) {
        button->setMinimumHeight(30);
        toolbar->addWidget(button);
    }
    toolbar->addWidget(m_address, 1);
    // Keep the bookmark menu separate so saved destinations remain easy to find.
    auto *bookmarkMenuButton = new QPushButton(QStringLiteral("Bookmarks ▾"), this);
    bookmarkMenuButton->setMinimumHeight(30);
    bookmarkMenuButton->setMenu(m_bookmarksMenu);
    toolbar->addWidget(bookmarkMenuButton);
    root->addLayout(toolbar);
    root->addWidget(m_view, 1);

    QSettings settings(QStringLiteral("OtterLink"), QStringLiteral("OtterLink"));
    m_bookmarks = settings.value(QStringLiteral("otterweb/bookmarks")).toStringList();
    rebuildBookmarksMenu();

    connect(m_backButton, &QPushButton::clicked, m_view, &QWebEngineView::back);
    connect(m_forwardButton, &QPushButton::clicked, m_view, &QWebEngineView::forward);
    connect(m_homeButton, &QPushButton::clicked, this, [this]() {
        navigateTo(QStringLiteral("https://www.amigaz.org/otterlink/"));
    });
    connect(m_goButton, &QPushButton::clicked, this, &OtterWebWidget::navigateToAddress);
    connect(m_address, &QLineEdit::returnPressed, this, &OtterWebWidget::navigateToAddress);
    connect(m_bookmarkButton, &QPushButton::clicked, this, &OtterWebWidget::addBookmark);
    connect(m_view, &QWebEngineView::urlChanged, this, [this](const QUrl &url) {
        m_address->setText(url.toString());
        updateNavigationState();
    });
    connect(m_view, &QWebEngineView::titleChanged, this, [this](const QString &title) {
        if (!title.isEmpty())
            setToolTip(title);
    });
    connect(m_view, &QWebEngineView::loadFinished, this, [this](bool) {
        updateNavigationState();
        m_address->setText(m_view->url().toString());
    });

    m_view->setUrl(QUrl(QStringLiteral("https://www.amigaz.org/otterlink/")));
    updateNavigationState();
}

void OtterWebWidget::navigateToAddress()
{
    navigateTo(m_address->text());
}

void OtterWebWidget::navigateTo(const QString &address)
{
    QString text = address.trimmed();
    if (text.isEmpty())
        return;
    QUrl url = QUrl::fromUserInput(text);
    if (!url.isValid() || url.scheme().isEmpty()) {
        QMessageBox::warning(this, QStringLiteral("OtterWeb"),
                             QStringLiteral("Enter a valid web address."));
        return;
    }
    if (url.scheme() != QStringLiteral("http") &&
        url.scheme() != QStringLiteral("https")) {
        QMessageBox::warning(this, QStringLiteral("OtterWeb"),
                             QStringLiteral("OtterWeb currently supports HTTP and HTTPS addresses."));
        return;
    }
    m_view->setUrl(url);
}

void OtterWebWidget::updateNavigationState()
{
    if (!m_view || !m_view->history())
        return;
    m_backButton->setEnabled(m_view->history()->canGoBack());
    m_forwardButton->setEnabled(m_view->history()->canGoForward());
}

void OtterWebWidget::addBookmark()
{
    const QUrl url = m_view->url();
    if (!url.isValid() || (url.scheme() != QStringLiteral("http") &&
                           url.scheme() != QStringLiteral("https"))) {
        QMessageBox::information(this, QStringLiteral("OtterWeb"),
                                 QStringLiteral("Open a web page before bookmarking it."));
        return;
    }
    const QString entry = m_view->title().trimmed().isEmpty()
        ? url.toString()
        : m_view->title().trimmed() + QStringLiteral(" | ") + url.toString();
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
            navigateTo(address);
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
