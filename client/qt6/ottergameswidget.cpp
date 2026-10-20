#include "ottergameswidget.h"
#include "otterlinkclient.h"
#include "otterhomepage.h"

#include <QFont>
#include <QHBoxLayout>
#include <QLabel>
#include <QLineEdit>
#include <QListWidget>
#include <QPushButton>
#include <QSplitter>
#include <QTextBrowser>
#include <QTabWidget>
#include <QTextCursor>
#include <QTimer>
#include <QVBoxLayout>
#include <QMessageBox>

OtterGamesWidget::OtterGamesWidget(OtterLinkClient *client, QWidget *parent)
    : QWidget(parent), m_client(client)
{
    auto *root = new QVBoxLayout(this);
    root->setContentsMargins(8, 8, 8, 8);

    auto *heading = new QLabel(QStringLiteral("Games — Interactive Fiction"), this);
    QFont headingFont = heading->font();
    headingFont.setBold(true);
    headingFont.setPointSize(headingFont.pointSize() + 2);
    heading->setFont(headingFont);
    auto *headingRow = new QHBoxLayout;
    auto *refreshButton = new QPushButton(QStringLiteral("Refresh"), this);
    headingRow->addWidget(heading, 1);
    headingRow->addWidget(refreshButton);
    root->addLayout(headingRow);
    connect(refreshButton, &QPushButton::clicked, this, &OtterGamesWidget::loadGames);

    m_tabs = new QTabWidget(this);
    m_gamesHome = new OtterHomePage(m_tabs);
    m_tabs->addTab(m_gamesHome, QStringLiteral("Games Home"));
    auto *catalogPage = new QWidget(m_tabs);
    auto *catalogLayout = new QVBoxLayout(catalogPage);
    catalogLayout->setContentsMargins(0, 0, 0, 0);
    auto *split = new QSplitter(Qt::Horizontal, catalogPage);
    m_games = new QListWidget(split);
    m_games->setMinimumWidth(220);

    auto *details = new QWidget(split);
    auto *detailsLayout = new QVBoxLayout(details);
    detailsLayout->setContentsMargins(0, 0, 0, 0);

    m_title = new QLabel(QStringLiteral("Select a game"), details);
    QFont titleFont = m_title->font();
    titleFont.setBold(true);
    titleFont.setPointSize(titleFont.pointSize() + 1);
    m_title->setFont(titleFont);
    m_title->setWordWrap(true);
    m_title->setTextFormat(Qt::PlainText);
    detailsLayout->addWidget(m_title);

    m_metadata = new QLabel(details);
    m_metadata->setWordWrap(true);
    m_metadata->setTextFormat(Qt::PlainText);
    detailsLayout->addWidget(m_metadata);

    m_description = new QLabel(QStringLiteral("Choose a game from the catalog to see its description."), details);
    m_description->setWordWrap(true);
    m_description->setTextFormat(Qt::PlainText);
    detailsLayout->addWidget(m_description);

    m_output = new QTextBrowser(details);
    m_output->setOpenLinks(false);
    m_output->setOpenExternalLinks(false);
    m_output->setPlaceholderText(QStringLiteral("Game output will appear here."));
    detailsLayout->addWidget(m_output, 1);

    m_status = new QLabel(QStringLiteral("Select an available game to begin."), details);
    m_status->setWordWrap(true);
    detailsLayout->addWidget(m_status);

    auto *gameActions = new QHBoxLayout;
    m_playButton = new QPushButton(QStringLiteral("Play"), details);
    m_stopButton = new QPushButton(QStringLiteral("Stop Game"), details);
    gameActions->addWidget(m_playButton);
    gameActions->addWidget(m_stopButton);
    gameActions->addStretch(1);
    detailsLayout->addLayout(gameActions);

    auto *commandRow = new QHBoxLayout;
    m_command = new QLineEdit(details);
    m_command->setPlaceholderText(QStringLiteral("Enter a command, such as LOOK or INVENTORY"));
    m_sendButton = new QPushButton(QStringLiteral("Send"), details);
    commandRow->addWidget(m_command, 1);
    commandRow->addWidget(m_sendButton);
    detailsLayout->addLayout(commandRow);

    split->addWidget(m_games);
    split->addWidget(details);
    split->setStretchFactor(1, 1);
    catalogLayout->addWidget(split, 1);
    m_tabs->addTab(catalogPage, QStringLiteral("Catalog & Play"));
    root->addWidget(m_tabs, 1);

    m_pollTimer = new QTimer(this);
    m_pollTimer->setInterval(400);

    connect(m_games, &QListWidget::currentItemChanged, this,
            [this](QListWidgetItem *current, QListWidgetItem *) { selectGame(current); });
    connect(m_playButton, &QPushButton::clicked, this, &OtterGamesWidget::startGame);
    connect(m_stopButton, &QPushButton::clicked, this, [this]() {
        if (m_sessionId.isEmpty())
            return;
        m_status->setText(QStringLiteral("Stopping game…"));
        m_command->setEnabled(false);
        m_sendButton->setEnabled(false);
        m_client->stopGameSession(m_sessionId);
    });
    connect(m_sendButton, &QPushButton::clicked, this, &OtterGamesWidget::sendCommand);
    connect(m_command, &QLineEdit::returnPressed, this, &OtterGamesWidget::sendCommand);
    connect(m_pollTimer, &QTimer::timeout, this, &OtterGamesWidget::pollSession);

    connect(m_client, &OtterLinkClient::gamesHomeScreenLoaded, m_gamesHome, [this](const QJsonObject &screen) { m_gamesHome->setServerScreen(screen); });
    connect(m_client, &OtterLinkClient::gamesHomeBackgroundLoaded, m_gamesHome, [this](const QJsonObject &, const QImage &image) { m_gamesHome->setServerBackground(image); });
    connect(m_client, &OtterLinkClient::gamesHomeAssetLoaded, m_gamesHome, &OtterHomePage::setServerAsset);
    connect(m_gamesHome, &OtterHomePage::serviceRequested, this, [this](const QString &service) {
        if (service == QStringLiteral("games-catalog") || service == QStringLiteral("games")) {
            m_tabs->setCurrentIndex(1);
        } else if (service.startsWith(QStringLiteral("game:"))) {
            launchGameSlug(service.mid(5));
        } else {
            emit serviceRequested(service);
        }
    });
    connect(m_client, &OtterLinkClient::gamesLoaded, this, &OtterGamesWidget::gamesLoaded);
    connect(m_client, &OtterLinkClient::gameSessionStarted, this, &OtterGamesWidget::sessionStarted);
    connect(m_client, &OtterLinkClient::gameOutputLoaded, this, &OtterGamesWidget::outputLoaded);
    connect(m_client, &OtterLinkClient::gameSessionStopped, this, &OtterGamesWidget::sessionStopped);
    connect(m_client, &OtterLinkClient::errorOccurred, this, &OtterGamesWidget::showError);

    updateControls();
    loadGames();
}

OtterGamesWidget::~OtterGamesWidget()
{
    m_pollTimer->stop();
    if (!m_sessionId.isEmpty())
        m_client->stopGameSession(m_sessionId);
}

void OtterGamesWidget::loadGames()
{
    m_client->loadGames();
    m_client->loadGamesHomeScreen();
}

void OtterGamesWidget::launchGameSlug(const QString &slug)
{
    if (!m_sessionId.isEmpty()) {
        m_status->setText(QStringLiteral("Stop the active game before launching another."));
        m_tabs->setCurrentIndex(1);
        return;
    }
    m_requestedSlug = slug.trimmed().toLower();
    if (m_requestedSlug.isEmpty())
        return;
    m_tabs->setCurrentIndex(1);
    if (!m_gamesLoaded)
        return;
    for (int i = 0; i < m_games->count(); ++i) {
        auto *item = m_games->item(i);
        if (item->data(Qt::UserRole + 2).toString().compare(m_requestedSlug, Qt::CaseInsensitive) == 0) {
            m_requestedSlug.clear();
            m_games->setCurrentItem(item);
            startGame();
            return;
        }
    }
    m_requestedSlug.clear();
    m_status->setText(QStringLiteral("That game is not published in the catalog."));
}

void OtterGamesWidget::gamesLoaded(const QJsonArray &games)
{
    m_gamesLoaded = true;
    const qint64 previous = m_gameId;
    m_games->clear();
    for (const auto &value : games) {
        const QJsonObject game = value.toObject();
        const qint64 id = qint64(game.value(QStringLiteral("id")).toDouble());
        const QString title = game.value(QStringLiteral("title")).toString();
        const QString category = game.value(QStringLiteral("category")).toString();
        const QString author = game.value(QStringLiteral("author")).toString();
        const bool available = game.value(QStringLiteral("available")).toBool();
        auto *item = new QListWidgetItem(m_games);
        item->setData(Qt::UserRole, id);
        item->setData(Qt::UserRole + 1, title);
        item->setData(Qt::UserRole + 2, game.value(QStringLiteral("slug")).toString());
        item->setData(Qt::UserRole + 3, game.value(QStringLiteral("description")).toString());
        item->setData(Qt::UserRole + 4, author);
        item->setData(Qt::UserRole + 5, category);
        item->setData(Qt::UserRole + 6, available);
        QString label = title;
        if (!category.isEmpty())
            label += QStringLiteral("\n") + category;
        if (!author.isEmpty())
            label += QStringLiteral(" · ") + author;
        label += available ? QStringLiteral("\nAvailable") : QStringLiteral("\nRuntime/story unavailable");
        item->setText(label);
        item->setToolTip(game.value(QStringLiteral("description")).toString());
        if (id == previous)
            m_games->setCurrentItem(item);
    }

    if (!m_requestedSlug.isEmpty()) {
        for (int i = 0; i < m_games->count(); ++i) {
            auto *item = m_games->item(i);
            if (item->data(Qt::UserRole + 2).toString().compare(m_requestedSlug, Qt::CaseInsensitive) == 0) {
                m_requestedSlug.clear();
                m_tabs->setCurrentIndex(1);
                m_games->setCurrentItem(item);
                startGame();
                updateControls();
                return;
            }
        }
        m_requestedSlug.clear();
        m_status->setText(QStringLiteral("That game is not published in the catalog."));
    }

    if (m_games->count() == 0) {
        m_gameId = 0;
        m_gameAvailable = false;
        m_title->setText(QStringLiteral("No games published yet"));
        m_metadata->clear();
        m_description->setText(QStringLiteral("An administrator can register a game after installing its story file on the server."));
        m_status->setText(QStringLiteral("The catalog is empty."));
    } else if (!m_games->currentItem()) {
        m_games->setCurrentRow(0);
    } else {
        selectGame(m_games->currentItem());
    }
    updateControls();
}

void OtterGamesWidget::selectGame(QListWidgetItem *item)
{
    if (!item)
        return;
    m_gameId = item->data(Qt::UserRole).toLongLong();
    m_gameAvailable = item->data(Qt::UserRole + 6).toBool();
    m_title->setText(item->data(Qt::UserRole + 1).toString());
    const QString author = item->data(Qt::UserRole + 4).toString();
    const QString category = item->data(Qt::UserRole + 5).toString();
    m_metadata->setText(category + (author.isEmpty() ? QString() : QStringLiteral(" · ") + author));
    m_description->setText(item->data(Qt::UserRole + 3).toString());
    if (m_sessionId.isEmpty()) {
        m_status->setText(m_gameAvailable
            ? QStringLiteral("Ready to play. Your progress is saved in your own session directory.")
            : QStringLiteral("This game is listed but cannot be launched on the server yet."));
    }
    updateControls();
}

void OtterGamesWidget::startGame()
{
    if (m_gameId < 1 || !m_gameAvailable || !m_sessionId.isEmpty())
        return;
    m_output->clear();
    m_status->setText(QStringLiteral("Starting game…"));
    m_playButton->setEnabled(false);
    m_client->startGameSession(m_gameId);
}

void OtterGamesWidget::sessionStarted(const QJsonObject &session)
{
    m_sessionId = session.value(QStringLiteral("id")).toString();
    if (m_sessionId.isEmpty()) {
        showError(QStringLiteral("The server did not return a game session ID."));
        updateControls();
        return;
    }
    m_lastSequence = 0;
    m_games->setEnabled(false);
    m_status->setText(QStringLiteral("Playing: %1").arg(session.value(QStringLiteral("game_title")).toString()));
    updateControls();
    m_pollTimer->start();
    pollSession();
}

void OtterGamesWidget::pollSession()
{
    if (m_sessionId.isEmpty() || m_pollInFlight)
        return;
    m_pollInFlight = true;
    m_client->pollGameSession(m_sessionId, m_lastSequence);
}

void OtterGamesWidget::outputLoaded(const QJsonObject &result)
{
    m_pollInFlight = false;
    const QJsonObject session = result.value(QStringLiteral("session")).toObject();
    if (session.value(QStringLiteral("id")).toString() != m_sessionId || m_sessionId.isEmpty())
        return;
    if (result.value(QStringLiteral("truncated")).toBool())
        appendOutput(QStringLiteral("\n[Earlier output was trimmed to keep this session bounded.]\n"));
    const QJsonArray chunks = result.value(QStringLiteral("chunks")).toArray();
    for (const auto &value : chunks) {
        const QJsonObject chunk = value.toObject();
        appendOutput(chunk.value(QStringLiteral("text")).toString());
        m_lastSequence = qMax(m_lastSequence, qint64(chunk.value(QStringLiteral("sequence")).toDouble()));
    }
    m_lastSequence = qMax(m_lastSequence, qint64(result.value(QStringLiteral("next_sequence")).toDouble()));

    const QString status = session.value(QStringLiteral("status")).toString();
    if (status != QStringLiteral("running") && status != QStringLiteral("stopping")) {
        m_pollTimer->stop();
        m_sessionId.clear();
        m_games->setEnabled(true);
        m_status->setText(status == QStringLiteral("finished")
            ? QStringLiteral("Game ended.")
            : QStringLiteral("Game session ended with an error."));
        updateControls();
    }
}

void OtterGamesWidget::sendCommand()
{
    const QString command = m_command->text().trimmed();
    if (command.isEmpty() || m_sessionId.isEmpty())
        return;
    appendOutput(QStringLiteral("\n> %1\n").arg(command));
    m_client->sendGameInput(m_sessionId, command);
    m_command->clear();
}

void OtterGamesWidget::sessionStopped(const QString &sessionId)
{
    if (sessionId != m_sessionId)
        return;
    m_pollTimer->stop();
    m_sessionId.clear();
    m_games->setEnabled(true);
    m_status->setText(QStringLiteral("Game session stopped."));
    updateControls();
}

void OtterGamesWidget::appendOutput(const QString &text)
{
    if (text.isEmpty())
        return;
    QTextCursor cursor = m_output->textCursor();
    cursor.movePosition(QTextCursor::End);
    cursor.insertText(text);
    m_output->setTextCursor(cursor);
    m_output->ensureCursorVisible();
}

void OtterGamesWidget::updateControls()
{
    const bool active = !m_sessionId.isEmpty();
    m_playButton->setEnabled(!active && m_gameId > 0 && m_gameAvailable);
    m_stopButton->setEnabled(active);
    m_command->setEnabled(active);
    m_sendButton->setEnabled(active);
}

void OtterGamesWidget::showError(const QString &message)
{
    m_pollInFlight = false;
    if (message.isEmpty())
        return;
    if (!m_sessionId.isEmpty())
        m_pollTimer->stop();
    if (!m_sessionId.isEmpty() &&
        (message.contains(QStringLiteral("game session not found"), Qt::CaseInsensitive) ||
         message.contains(QStringLiteral("unauthorized"), Qt::CaseInsensitive))) {
        m_pollTimer->stop();
        m_sessionId.clear();
        m_games->setEnabled(true);
        updateControls();
    }
    updateControls();
    QMessageBox::warning(this, QStringLiteral("Games"), message);
}
