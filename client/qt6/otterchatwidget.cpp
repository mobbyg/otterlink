#include "otterchatwidget.h"

#include "otterlinkclient.h"

#include <QCheckBox>
#include <QComboBox>
#include <QCursor>
#include <QDialog>
#include <QDialogButtonBox>
#include <QHBoxLayout>
#include <QInputDialog>
#include <QLineEdit>
#include <QListWidget>
#include <QMenu>
#include <QPushButton>
#include <QVBoxLayout>

OtterChatWidget::OtterChatWidget(OtterLinkClient *client, QWidget *parent)
    : QWidget(parent), m_client(client)
{
    auto *layout = new QVBoxLayout(this);
    layout->setContentsMargins(8, 8, 8, 8);
    layout->setSpacing(6);

    auto *channelBar = new QHBoxLayout;
    m_channelCombo = new QComboBox(this);
    m_channelCombo->setMinimumWidth(220);
    m_channelCombo->setToolTip(QStringLiteral("Choose a chat room"));
    m_createButton = new QPushButton(QStringLiteral("+"), this);
    m_createButton->setFixedWidth(32);
    m_createButton->setToolTip(QStringLiteral("Create a new chat room"));
    channelBar->addWidget(m_channelCombo, 1);
    channelBar->addWidget(m_createButton);
    layout->addLayout(channelBar);

    auto *body = new QHBoxLayout;
    m_chatList = new QListWidget(this);
    m_userList = new QListWidget(this);
    m_userList->setContextMenuPolicy(Qt::CustomContextMenu);
    m_userList->setMinimumWidth(150);
    m_userList->setMaximumWidth(220);
    body->addWidget(m_chatList, 1);
    body->addWidget(m_userList);
    layout->addLayout(body, 1);

    auto *input = new QHBoxLayout;
    m_chatEdit = new QLineEdit(this);
    m_chatEdit->setPlaceholderText(QStringLiteral("Type a message..."));
    m_emojiButton = new QPushButton(QStringLiteral("🙂"), this);
    m_emojiButton->setFixedWidth(38);
    m_sendButton = new QPushButton(QStringLiteral("SEND"), this);
    input->addWidget(m_chatEdit, 1);
    input->addWidget(m_emojiButton);
    input->addWidget(m_sendButton);
    layout->addLayout(input);

    connect(m_channelCombo, qOverload<int>(&QComboBox::currentIndexChanged), this, &OtterChatWidget::channelChanged);
    connect(m_createButton, &QPushButton::clicked, this, &OtterChatWidget::createChannel);
    connect(m_sendButton, &QPushButton::clicked, this, &OtterChatWidget::sendMessage);
    connect(m_chatEdit, &QLineEdit::returnPressed, this, &OtterChatWidget::sendMessage);
    connect(m_userList, &QListWidget::customContextMenuRequested, this, &OtterChatWidget::userContextMenu);
    connect(m_client, &OtterLinkClient::chatChannelsLoaded, this, &OtterChatWidget::channelsLoaded);
    connect(m_client, &OtterLinkClient::chatChannelLoaded, this, &OtterChatWidget::channelLoaded);
    connect(m_client, &OtterLinkClient::chatActionCompleted, this, &OtterChatWidget::actionCompleted);
    connect(m_client, &OtterLinkClient::chatMessageSent, this, &OtterChatWidget::refreshCurrentChannel);
    connect(m_client, &OtterLinkClient::errorOccurred, this, &OtterChatWidget::showError);

    connect(m_emojiButton, &QPushButton::clicked, this, [this]() {
        auto *menu = new QMenu(m_emojiButton);
        const QStringList emojis = {QStringLiteral("😀"), QStringLiteral("😃"), QStringLiteral("😄"), QStringLiteral("😁"), QStringLiteral("😂"), QStringLiteral("🤣"), QStringLiteral("😊"), QStringLiteral("😎"), QStringLiteral("😍"), QStringLiteral("🤔"), QStringLiteral("👍"), QStringLiteral("👎"), QStringLiteral("❤️"), QStringLiteral("🎉"), QStringLiteral("🔥"), QStringLiteral("🦦")};
        for (const QString &emoji : emojis) {
            auto *action = menu->addAction(emoji);
            connect(action, &QAction::triggered, this, [this, emoji]() { m_chatEdit->insert(emoji); m_chatEdit->setFocus(); });
        }
        menu->exec(m_emojiButton->mapToGlobal(QPoint(0, -menu->sizeHint().height())));
        menu->deleteLater();
    });

    loadChannels();
}

void OtterChatWidget::loadChannels() { m_client->loadChatChannels(); }

void OtterChatWidget::channelsLoaded(const QJsonArray &channels)
{
    const qint64 previous = m_channelId;
    m_channels.clear();
    m_channelCombo->blockSignals(true);
    m_channelCombo->clear();
    int select = -1;
    for (const QJsonValue &value : channels) {
        const QJsonObject channel = value.toObject();
        const qint64 id = static_cast<qint64>(channel.value(QStringLiteral("id")).toDouble());
        if (id < 1) continue;
        m_channels.insert(id, channel);
        m_channelCombo->addItem(channel.value(QStringLiteral("name")).toString(), id);
        if (id == previous) select = m_channelCombo->count() - 1;
    }
    if (select < 0 && m_channelCombo->count() > 0) select = 0;
    m_channelCombo->setCurrentIndex(select);
    m_channelCombo->blockSignals(false);
    if (select >= 0) channelChanged(select);
    else {
        m_channelId = 0; m_channelName.clear(); m_role.clear();
        m_chatList->clear(); m_userList->clear();
        m_chatList->addItem(QStringLiteral("No chat rooms exist yet. Use + to create one."));
    }
}

void OtterChatWidget::channelChanged(int index)
{
    if (index < 0) return;
    const qint64 id = m_channelCombo->itemData(index).toLongLong();
    if (id < 1) return;
    if (m_channelId > 0 && m_channelId != id) m_client->leaveChatChannel(m_channelId);
    m_channelId = id;
    m_channelName = m_channelCombo->currentText();
    m_client->joinChatChannel(m_channelId);
}

void OtterChatWidget::channelLoaded(const QJsonObject &channel, const QJsonArray &members, const QJsonArray &messages, const QString &role)
{
    m_channelId = static_cast<qint64>(channel.value(QStringLiteral("id")).toDouble());
    m_channelName = channel.value(QStringLiteral("name")).toString();
    m_role = role;
    populateUsers(members);
    populateMessages(messages);
}

void OtterChatWidget::populateUsers(const QJsonArray &members)
{
    m_userList->clear(); m_userRoles.clear();
    for (const QJsonValue &value : members) {
        const QJsonObject member = value.toObject();
        const QJsonObject user = member.value(QStringLiteral("user")).toObject();
        const QString username = user.value(QStringLiteral("username")).toString();
        const QString display = user.value(QStringLiteral("display_name")).toString(username);
        const QString role = member.value(QStringLiteral("role")).toString();
        if (username.isEmpty()) continue;
        m_userRoles.insert(username, role);
        auto *item = new QListWidgetItem(QStringLiteral("%1 %2").arg(roleIcon(role), display), m_userList);
        item->setData(Qt::UserRole, username);
        item->setData(Qt::UserRole + 1, role);
    }
}

void OtterChatWidget::populateMessages(const QJsonArray &messages)
{
    m_chatList->clear();
    for (const QJsonValue &value : messages) {
        const QJsonObject message = value.toObject();
        const QJsonObject from = message.value(QStringLiteral("from")).toObject();
        const QString name = from.value(QStringLiteral("display_name")).toString(from.value(QStringLiteral("username")).toString());
        m_chatList->addItem(QStringLiteral("%1: %2").arg(name, message.value(QStringLiteral("message")).toString()));
    }
    m_chatList->scrollToBottom();
}

QString OtterChatWidget::roleIcon(const QString &role) const
{
    if (role == QStringLiteral("original_mod")) return QStringLiteral("👑");
    if (role == QStringLiteral("mod")) return QStringLiteral("🔶");
    if (role == QStringLiteral("op")) return QStringLiteral("🟢");
    return QString();
}
QString OtterChatWidget::currentUsername() const { return m_client->accountName(); }

QString OtterChatWidget::selectedUsername() const
{
    const auto *item = m_userList->itemAt(m_userList->mapFromGlobal(QCursor::pos()));
    return item ? item->data(Qt::UserRole).toString() : QString();
}

bool OtterChatWidget::canModerate(const QString &targetRole) const
{
    if (targetRole == QStringLiteral("original_mod")) return false;
    if (m_role == QStringLiteral("original_mod")) return true;
    if (m_role == QStringLiteral("mod")) return targetRole != QStringLiteral("mod");
    if (m_role == QStringLiteral("op")) return targetRole == QStringLiteral("user");
    return false;
}

bool OtterChatWidget::canAssignRole(const QString &targetRole, const QString &newRole) const
{
    if (targetRole == QStringLiteral("original_mod")) return false;
    if (newRole == QStringLiteral("mod")) return m_role == QStringLiteral("original_mod");
    if (newRole == QStringLiteral("op")) return m_role == QStringLiteral("original_mod") || m_role == QStringLiteral("mod") || m_role == QStringLiteral("op");
    return m_role == QStringLiteral("original_mod") || m_role == QStringLiteral("mod");
}

void OtterChatWidget::createChannel()
{
    bool ok = false;
    const QString name = QInputDialog::getText(this, QStringLiteral("Create Chat Room"), QStringLiteral("Room name:"), QLineEdit::Normal, QString(), &ok).trimmed();
    if (!ok || name.isEmpty()) return;
    QDialog dialog(this);
    dialog.setWindowTitle(QStringLiteral("Create Chat Room"));
    auto *layout = new QVBoxLayout(&dialog);
    auto *check = new QCheckBox(QStringLiteral("Allow Ops to promote to Ops"), &dialog);
    auto *buttons = new QDialogButtonBox(QDialogButtonBox::Ok | QDialogButtonBox::Cancel, &dialog);
    layout->addWidget(check); layout->addWidget(buttons);
    connect(buttons, &QDialogButtonBox::accepted, &dialog, &QDialog::accept);
    connect(buttons, &QDialogButtonBox::rejected, &dialog, &QDialog::reject);
    if (dialog.exec() != QDialog::Accepted) return;
    m_client->createChatChannel(name, check->isChecked());
}

void OtterChatWidget::sendMessage()
{
    const QString message = m_chatEdit->text().trimmed();
    if (message.isEmpty() || m_channelId < 1) return;
    m_chatEdit->clear();
    m_client->sendChatMessage(m_channelId, message);
}

void OtterChatWidget::refreshCurrentChannel() { if (m_channelId > 0) m_client->joinChatChannel(m_channelId); }
void OtterChatWidget::actionCompleted() { loadChannels(); if (m_channelId > 0) m_client->joinChatChannel(m_channelId); }

void OtterChatWidget::userContextMenu(const QPoint &position)
{
    auto *item = m_userList->itemAt(position);
    if (!item) return;
    const QString username = item->data(Qt::UserRole).toString();
    const QString targetRole = item->data(Qt::UserRole + 1).toString();
    if (username.isEmpty() || username.compare(currentUsername(), Qt::CaseInsensitive) == 0) return;
    QMenu menu(this);
    if (canAssignRole(targetRole, QStringLiteral("mod"))) menu.addAction(QStringLiteral("🔶 Make Mod"), this, [this, username]() { m_client->setChatRole(m_channelId, username, QStringLiteral("mod")); });
    if (canAssignRole(targetRole, QStringLiteral("op"))) menu.addAction(QStringLiteral("🟢 Make Op"), this, [this, username]() { m_client->setChatRole(m_channelId, username, QStringLiteral("op")); });
    if (canAssignRole(targetRole, QStringLiteral("user")) && (targetRole == QStringLiteral("mod") || targetRole == QStringLiteral("op"))) menu.addAction(QStringLiteral("Remove Role"), this, [this, username]() { m_client->setChatRole(m_channelId, username, QStringLiteral("user")); });
    if (canModerate(targetRole)) {
        if (!menu.isEmpty()) menu.addSeparator();
        menu.addAction(QStringLiteral("Kick"), this, [this, username]() { m_client->moderateChatUser(m_channelId, username, QStringLiteral("kick")); });
        menu.addAction(QStringLiteral("Ban"), this, [this, username]() { m_client->moderateChatUser(m_channelId, username, QStringLiteral("ban")); });
    }
    if (!menu.isEmpty()) menu.exec(m_userList->viewport()->mapToGlobal(position));
}

void OtterChatWidget::showError(const QString &message)
{
    if (!message.isEmpty()) m_chatList->addItem(QStringLiteral("[Chat] %1").arg(message));
}
