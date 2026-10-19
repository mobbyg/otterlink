#pragma once

#include <QWidget>
#include <QStringList>

class QLineEdit;
class QMenu;
class QPushButton;
class QTextBrowser;

class OtterWebWidget final : public QWidget
{
    Q_OBJECT

public:
    explicit OtterWebWidget(QWidget *parent = nullptr);

private:
    void navigateToAddress();
    void updateNavigationState();
    void addBookmark();
    void rebuildBookmarksMenu();

    QTextBrowser *m_view = nullptr;
    QLineEdit *m_address = nullptr;
    QPushButton *m_backButton = nullptr;
    QPushButton *m_forwardButton = nullptr;
    QPushButton *m_homeButton = nullptr;
    QPushButton *m_goButton = nullptr;
    QPushButton *m_bookmarkButton = nullptr;
    QMenu *m_bookmarksMenu = nullptr;
    QStringList m_bookmarks;
};
