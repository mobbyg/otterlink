#pragma once

#include <QJsonArray>
#include <QWidget>

class QComboBox;
class QLabel;
class QListWidget;
class QPushButton;
class OtterLinkClient;

class OtterEventsWidget final : public QWidget
{
    Q_OBJECT
public:
    explicit OtterEventsWidget(OtterLinkClient *client, QWidget *parent = nullptr);

private slots:
    void previousMonth();
    void nextMonth();
    void addEvent();
    void editSelected();
    void deleteSelected();
    void eventsLoaded(const QJsonArray &events, int year, int month);
    void eventChanged(const QJsonObject &event);
    void eventDeleted(qint64 eventId);
    void chatChannelsLoaded(const QJsonArray &channels);

private:
    void loadCurrentMonth();
    QJsonObject selectedEvent() const;
    void openEditor(const QJsonObject &event, bool editing);

    OtterLinkClient *m_client = nullptr;
    QLabel *m_monthLabel = nullptr;
    QListWidget *m_list = nullptr;
    QPushButton *m_editButton = nullptr;
    QPushButton *m_deleteButton = nullptr;
    QComboBox *m_targetCombo = nullptr;
    int m_year = 0;
    int m_month = 0;
    QJsonArray m_events;
    QJsonArray m_channels;
};
