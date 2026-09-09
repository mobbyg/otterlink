#include "otterlinkstyle.h"

#include <QApplication>

namespace OtterLinkStyle {

void install(QApplication &app)
{
    app.setStyleSheet(QStringLiteral(R"STYLE(
        QMainWindow, QWidget {
            background: #dfeaf3;
            color: #17324d;
        }

        QFrame#serviceHeader {
            background: qlineargradient(x1:0, y1:0, x2:0, y2:1,
                                        stop:0 #0d6f9d, stop:0.48 #075b86, stop:1 #06496d);
            border: 1px solid #063d5c;
            border-bottom: 2px solid #022e46;
        }

        QLabel#brandLabel {
            color: white;
            background: transparent;
        }

        QLabel#identityLabel {
            color: #d8f1ff;
            background: transparent;
        }

        QPushButton#refreshButton, QPushButton#logoutButton {
            background: #e8f2f8;
            color: #16435f;
            border: 1px solid #7d9db2;
            border-radius: 3px;
            padding: 4px 10px;
            min-height: 20px;
        }

        QPushButton#refreshButton:hover, QPushButton#logoutButton:hover {
            background: #ffffff;
            border-color: #3f7898;
        }

        QPushButton#refreshButton:pressed, QPushButton#logoutButton:pressed {
            background: #c8dce8;
        }

        QFrame#navigationBar {
            background: #b8d0df;
            border-bottom: 1px solid #6f8fa4;
        }

        QFrame#serviceRail {
            background: #b8d0df;
            border-right: 1px solid #6f8fa4;
        }

        QLabel#railTitle, QLabel#railServices {
            color: #174563;
            background: transparent;
        }

        QPushButton#homeButton, QPushButton#peopleButton, QPushButton#mailButton,
        QPushButton#chatButton, QPushButton#boardsButton, QPushButton#newsButton,
        QPushButton#filesButton, QPushButton#gamesButton {
            text-align: left;
            background: qlineargradient(x1:0, y1:0, x2:0, y2:1,
                                        stop:0 #f5fbff, stop:0.48 #d8e8f2, stop:1 #b7cfdd);
            color: #173b55;
            border: 1px solid #7897a9;
            border-radius: 3px;
            padding: 6px 8px;
            margin: 1px 4px;
            min-height: 24px;
        }

        QPushButton#homeButton:hover, QPushButton#peopleButton:hover, QPushButton#mailButton:hover,
        QPushButton#chatButton:hover, QPushButton#boardsButton:hover, QPushButton#newsButton:hover,
        QPushButton#filesButton:hover, QPushButton#gamesButton:hover {
            background: #ffffff;
            border-color: #3d7797;
        }

        QPushButton#homeButton:pressed, QPushButton#peopleButton:pressed, QPushButton#mailButton:pressed,
        QPushButton#chatButton:pressed, QPushButton#boardsButton:pressed, QPushButton#newsButton:pressed,
        QPushButton#filesButton:pressed, QPushButton#gamesButton:pressed,
        QPushButton#homeButton[active="true"], QPushButton#peopleButton[active="true"],
        QPushButton#mailButton[active="true"], QPushButton#chatButton[active="true"],
        QPushButton#boardsButton[active="true"], QPushButton#newsButton[active="true"],
        QPushButton#filesButton[active="true"], QPushButton#gamesButton[active="true"] {
            background: qlineargradient(x1:0, y1:0, x2:0, y2:1,
                                        stop:0 #79b8d7, stop:1 #4f91b5);
            color: white;
            border-color: #245e7d;
        }

        QLabel#serviceTitleLabel {
            color: #12476a;
            background: #eaf3f8;
            border-bottom: 1px solid #a0b9c8;
            padding: 5px 8px;
        }

        QGroupBox {
            background: #edf5f9;
            border: 1px solid #9db6c5;
            border-radius: 3px;
            margin-top: 10px;
            padding-top: 6px;
        }

        QGroupBox::title {
            subcontrol-origin: margin;
            left: 8px;
            padding: 0 4px;
            color: #0c5279;
            background: #edf5f9;
        }

        QListWidget, QTextEdit, QLineEdit {
            background: white;
            color: #19364b;
            border: 1px solid #8ca8b8;
            border-radius: 2px;
            selection-background-color: #4c91b6;
            selection-color: white;
        }

        QListWidget {
            padding: 2px;
        }

        QListWidget::item {
            padding: 3px 5px;
            border-bottom: 1px solid #e3edf2;
        }

        QListWidget::item:selected {
            background: #4c91b6;
            color: white;
        }

        QPushButton#addBuddyButton, QPushButton#removeBuddyButton, QPushButton#sendChatButton {
            background: #d9e9f2;
            color: #174563;
            border: 1px solid #7d9daf;
            border-radius: 3px;
            padding: 4px 9px;
        }

        QPushButton#addBuddyButton:hover, QPushButton#removeBuddyButton:hover,
        QPushButton#sendChatButton:hover {
            background: #f7fbfd;
            border-color: #3f7898;
        }

        QPushButton#loginButton {
            background: qlineargradient(x1:0, y1:0, x2:0, y2:1,
                                        stop:0 #3b9bc5, stop:0.45 #147aa8, stop:1 #075b86);
            color: white;
            border: 1px solid #06496d;
            border-radius: 4px;
            padding: 8px 14px;
            font-weight: bold;
            min-height: 30px;
        }

        QPushButton#loginButton:hover {
            background: #2b8db8;
        }

        QPushButton#loginButton:pressed {
            background: #075b86;
        }

        QLabel#homeWelcomeLabel {
            color: #0c5279;
        }

        QLabel#homeNewsLabel, QLabel#homeBuddiesLabel, QLabel#homeOnlineLabel,
        QLabel#peopleInfoLabel, QLabel#placeholderInfoLabel {
            color: #31556b;
        }

        QLabel#statusLabel {
            background: #d1e3ec;
            color: #24566f;
            border-top: 1px solid #91adbd;
            padding: 3px 7px;
        }

        QProgressBar#connectionProgress {
            background: #d1e1ea;
            border: 1px solid #7394a8;
            border-radius: 3px;
            text-align: center;
            min-height: 12px;
        }

        QProgressBar#connectionProgress::chunk {
            background: #147aa8;
            border-radius: 2px;
        }

        QLabel#connectionStageLabel {
            color: #075b86;
            letter-spacing: 1px;
        }

        QLabel#placeholderTitleLabel {
            color: #075b86;
        }

        QToolTip {
            background: #173b55;
            color: white;
            border: 1px solid #78a0b6;
            padding: 4px;
        }
    )STYLE"));
}

} // namespace OtterLinkStyle
