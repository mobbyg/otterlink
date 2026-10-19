#pragma once

#include <QWidget>
#include <QStringList>
#include <QUrl>

class QLineEdit;
class QMenu;
class QPushButton;

#ifdef OTTERLINK_USE_WEBENGINE
class QWebEngineView;
#else
class QTextBrowser;
#endif

class OtterWebWidget final : public QWidget
{
    Q_OBJECT

public:
    explicit OtterWebWidget(QWidget *parent = nullptr);

private:
    void navigateToAddress();
    void loadInView(const QUrl &url);
    void updateNavigationState();
    void addBookmark();
    void rebuildBookmarksMenu();

#ifdef OTTERLINK_USE_WEBENGINE
    QWebEngineView *m_view = nullptr;
#else
    QTextBrowser *m_view = nullptr;
#endif
    QLineEdit *m_address = nullptr;
    QPushButton *m_backButton = nullptr;
    QPushButton *m_forwardButton = nullptr;
    QPushButton *m_homeButton = nullptr;
    QPushButton *m_goButton = nullptr;
    QPushButton *m_bookmarkButton = nullptr;
    QMenu *m_bookmarksMenu = nullptr;
    QStringList m_bookmarks;
};
