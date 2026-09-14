#pragma once

#include <QHash>
#include <QJsonArray>
#include <QJsonObject>
#include <QWidget>

class QComboBox;
class QLineEdit;
class QListWidget;
class QPushButton;
class QListWidgetItem;
class OtterLinkClient;

class OtterChatWidget final : public QWidget
{
    Q_OBJECT

public:
    explicit OtterChatWidget(OtterLinkClient *client, QWidget *parent = nullptr);

private slots:
    void loadChannels();
    void channelsLoaded(const QJsonArray &channels);
    void channelChanged(int index);
    void channelLoaded(const QJsonObject &channel, const QJsonArray &members,
                       const QJsonArray &messages, const QString &role);
    void createChannel();
    void sendMessage();
    void refreshCurrentChannel();
    void userContextMenu(const QPoint &position);
    void actionCompleted();
    void showError(const QString &message);

private:
    void populateUsers(const QJsonArray &members);
    void populateMessages(const QJsonArray &messages);
    QString roleIcon(const QString &role) const;
    QString currentUsername() const;
    QString selectedUsername() const;
    bool canModerate(const QString &targetRole) const;
    bool canAssignRole(const QString &targetRole, const QString &newRole) const;

    OtterLinkClient *m_client = nullptr;
    QComboBox *m_channelCombo = nullptr;
    QPushButton *m_createButton = nullptr;
    QListWidget *m_chatList = nullptr;
    QListWidget *m_userList = nullptr;
    QLineEdit *m_chatEdit = nullptr;
    QPushButton *m_emojiButton = nullptr;
    QPushButton *m_sendButton = nullptr;
    qint64 m_channelId = 0;
    QString m_role;
    QString m_channelName;
    QHash<qint64, QJsonObject> m_channels;
    QHash<QString, QString> m_userRoles;
};
