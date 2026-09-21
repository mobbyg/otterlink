#include "otterdmwidget.h"
#include "otterlinkclient.h"

#include <QHBoxLayout>
#include <QJsonArray>
#include <QJsonObject>
#include <QLabel>
#include <QLineEdit>
#include <QListWidget>
#include <QPushButton>
#include <QVBoxLayout>

OtterDmWidget::OtterDmWidget(OtterLinkClient *client, const QString &username, QWidget *parent)
    : QWidget(parent), m_client(client), m_username(username)
{
    auto *layout = new QVBoxLayout(this);
    auto *title = new QLabel(QStringLiteral("<b>Private Message</b> with %1").arg(username.toHtmlEscaped()), this);
    layout->addWidget(title);

    m_messages = new QListWidget(this);
    layout->addWidget(m_messages, 1);

    auto *input = new QHBoxLayout;
    m_input = new QLineEdit(this);
    m_input->setPlaceholderText(QStringLiteral("Type a private message..."));
    auto *send = new QPushButton(QStringLiteral("Send"), this);
    input->addWidget(m_input, 1);
    input->addWidget(send);
    layout->addLayout(input);

    connect(send, &QPushButton::clicked, this, &OtterDmWidget::sendMessage);
    connect(m_input, &QLineEdit::returnPressed, this, &OtterDmWidget::sendMessage);
    if (m_client) {
        connect(m_client, &OtterLinkClient::directConversationLoaded,
                this, &OtterDmWidget::conversationLoaded);
        connect(m_client, &OtterLinkClient::directMessageSent,
                this, &OtterDmWidget::messageSent);
        loadConversation();
    }
}

void OtterDmWidget::loadConversation()
{
    if (!m_client) return;
    m_client->loadDirectConversation(m_username);
    m_client->markDirectMessagesRead(m_username);
}

void OtterDmWidget::sendMessage()
{
    const QString message = m_input->text().trimmed();
    if (message.isEmpty() || !m_client) return;
    m_input->clear();
    m_client->sendDirectMessage(m_username, message);
}

void OtterDmWidget::conversationLoaded(const QJsonObject &conversation)
{
    const QJsonObject with = conversation.value(QStringLiteral("with")).toObject();
    const QString username = with.value(QStringLiteral("username")).toString();
    if (username.compare(m_username, Qt::CaseInsensitive) != 0) return;

    m_messages->clear();
    for (const QJsonValue &value : conversation.value(QStringLiteral("messages")).toArray()) {
        const QJsonObject message = value.toObject();
        const QJsonObject from = message.value(QStringLiteral("from")).toObject();
        const QString sender = from.value(QStringLiteral("username")).toString();
        const QString text = message.value(QStringLiteral("message")).toString();
        m_messages->addItem(QStringLiteral("%1: %2").arg(sender, text));
    }
    m_messages->scrollToBottom();
}

void OtterDmWidget::messageSent(const QJsonObject &message)
{
    const QJsonObject to = message.value(QStringLiteral("to")).toObject();
    if (to.value(QStringLiteral("username")).toString().compare(m_username, Qt::CaseInsensitive) != 0)
        return;
    loadConversation();
}
