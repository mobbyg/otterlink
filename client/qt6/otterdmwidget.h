#pragma once

#include <QTimer>
#include <QWidget>

class QListWidget;
class QLineEdit;
class OtterLinkClient;

class OtterDmWidget final : public QWidget
{
    Q_OBJECT
public:
    explicit OtterDmWidget(OtterLinkClient *client, const QString &username,
                           QWidget *parent = nullptr);

private slots:
    void loadConversation();
    void sendMessage();
    void conversationLoaded(const QJsonObject &conversation);
    void messageSent(const QJsonObject &message);
    void refreshConversation();

private:
    OtterLinkClient *m_client = nullptr;
    QString m_username;
    QListWidget *m_messages = nullptr;
    QLineEdit *m_input = nullptr;
    QTimer m_refreshTimer;
};
