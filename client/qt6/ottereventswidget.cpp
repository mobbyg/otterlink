#include "ottereventswidget.h"
#include "otterlinkclient.h"

#include <QCheckBox>
#include <QComboBox>
#include <QDateTimeEdit>
#include <QDate>
#include <QDateTime>
#include <QDialog>
#include <QDialogButtonBox>
#include <QFormLayout>
#include <QHBoxLayout>
#include <QLabel>
#include <QLineEdit>
#include <QListWidget>
#include <QMessageBox>
#include <QPlainTextEdit>
#include <QPushButton>
#include <QSpinBox>
#include <QVBoxLayout>

namespace {
QString eventWhen(const QJsonObject &event)
{
    if (event.value(QStringLiteral("all_day")).toBool()) {
        const QString start = event.value(QStringLiteral("start_at")).toString();
        const QString end = event.value(QStringLiteral("end_at")).toString();
        return start == end ? start : QStringLiteral("%1 → %2").arg(start, end);
    }
    const QDateTime start = QDateTime::fromString(event.value(QStringLiteral("start_at")).toString(), Qt::ISODate);
    const QDateTime end = QDateTime::fromString(event.value(QStringLiteral("end_at")).toString(), Qt::ISODate);
    return QStringLiteral("%1 → %2").arg(start.toLocalTime().toString(QStringLiteral("ddd MMM d, h:mm AP")),
                                          end.toLocalTime().toString(QStringLiteral("h:mm AP")));
}
}

OtterEventsWidget::OtterEventsWidget(OtterLinkClient *client, QWidget *parent)
    : QWidget(parent), m_client(client)
{
    auto *layout = new QVBoxLayout(this);
    layout->setContentsMargins(12,12,12,12);

    auto *nav = new QHBoxLayout;
    auto *prev = new QPushButton(QStringLiteral("◀"), this);
    auto *next = new QPushButton(QStringLiteral("▶"), this);
    m_monthLabel = new QLabel(this);
    m_monthLabel->setAlignment(Qt::AlignCenter);
    nav->addWidget(prev);
    nav->addWidget(m_monthLabel, 1);
    nav->addWidget(next);
    layout->addLayout(nav);

    m_list = new QListWidget(this);
    layout->addWidget(m_list, 1);

    auto *actions = new QHBoxLayout;
    auto *add = new QPushButton(QStringLiteral("Add Event"), this);
    m_editButton = new QPushButton(QStringLiteral("Edit Your Event"), this);
    m_deleteButton = new QPushButton(QStringLiteral("Delete Your Event"), this);
    m_editButton->setEnabled(false); m_deleteButton->setEnabled(false);
    actions->addWidget(add); actions->addWidget(m_editButton); actions->addWidget(m_deleteButton);
    layout->addLayout(actions);

    connect(prev,&QPushButton::clicked,this,&OtterEventsWidget::previousMonth);
    connect(next,&QPushButton::clicked,this,&OtterEventsWidget::nextMonth);
    connect(add,&QPushButton::clicked,this,&OtterEventsWidget::addEvent);
    connect(m_editButton,&QPushButton::clicked,this,&OtterEventsWidget::editSelected);
    connect(m_deleteButton,&QPushButton::clicked,this,&OtterEventsWidget::deleteSelected);
    connect(m_list,&QListWidget::itemSelectionChanged,this,[this]{
        const auto event=selectedEvent();
        const bool mine=!event.isEmpty() && event.value(QStringLiteral("created_by")).toString().compare(m_client->accountName(),Qt::CaseInsensitive)==0;
        m_editButton->setEnabled(mine); m_deleteButton->setEnabled(mine);
    });
    connect(m_client,&OtterLinkClient::eventsLoaded,this,&OtterEventsWidget::eventsLoaded);
    connect(m_client,&OtterLinkClient::eventChanged,this,&OtterEventsWidget::eventChanged);
    connect(m_client,&OtterLinkClient::eventDeleted,this,&OtterEventsWidget::eventDeleted);
    connect(m_client,&OtterLinkClient::chatChannelsLoaded,this,&OtterEventsWidget::chatChannelsLoaded);

    const QDate now=QDate::currentDate(); m_year=now.year();m_month=now.month();
    m_client->loadChatChannels();
    loadCurrentMonth();
}

void OtterEventsWidget::loadCurrentMonth(){m_monthLabel->setText(QDate(m_year,m_month,1).toString(QStringLiteral("MMMM yyyy")));m_client->loadEvents(m_year,m_month);}
void OtterEventsWidget::previousMonth(){if(--m_month<1){m_month=12;--m_year;}loadCurrentMonth();}
void OtterEventsWidget::nextMonth(){if(++m_month>12){m_month=1;++m_year;}loadCurrentMonth();}

QJsonObject OtterEventsWidget::selectedEvent() const
{
    const int row=m_list->currentRow();
    if(row<0||row>=m_events.size())return {};
    return m_events.at(row).toObject();
}

void OtterEventsWidget::eventsLoaded(const QJsonArray &events,int year,int month)
{
    m_year=year;m_month=month;m_events=events;
    m_monthLabel->setText(QDate(year,month,1).toString(QStringLiteral("MMMM yyyy")));
    m_list->clear();
    for(const auto &value:events){
        const auto e=value.toObject();
        auto *item=new QListWidgetItem(QStringLiteral("%1\n%2\n%3").arg(e.value("title").toString(),eventWhen(e),e.value("description").toString()),m_list);
        item->setToolTip(e.value("created_by").toString());
    }
}

void OtterEventsWidget::eventChanged(const QJsonObject &){loadCurrentMonth();}
void OtterEventsWidget::eventDeleted(qint64){loadCurrentMonth();}

void OtterEventsWidget::chatChannelsLoaded(const QJsonArray &channels){m_channels=channels;}

void OtterEventsWidget::addEvent(){openEditor({},false);}
void OtterEventsWidget::editSelected(){const auto e=selectedEvent();if(!e.isEmpty())openEditor(e,true);}
void OtterEventsWidget::deleteSelected()
{
    const auto e=selectedEvent();if(e.isEmpty())return;
    if(QMessageBox::question(this,QStringLiteral("Delete Event"),QStringLiteral("Delete '%1'?").arg(e.value("title").toString()))==QMessageBox::Yes)m_client->deleteEvent(e.value("id").toVariant().toLongLong());
}

void OtterEventsWidget::openEditor(const QJsonObject &event,bool editing)
{
    QDialog dialog(this);dialog.setWindowTitle(editing?QStringLiteral("Edit Event"):QStringLiteral("Add Event"));
    auto *form=new QFormLayout(&dialog);
    auto *title=new QLineEdit(&dialog);
    auto *description=new QPlainTextEdit(&dialog);
    description->setMaximumHeight(90);
    auto *type=new QComboBox(&dialog);type->addItem(QStringLiteral("Public"),"public");type->addItem(QStringLiteral("Community"),"community");
    auto *target=new QComboBox(&dialog);target->addItem(QStringLiteral("No community"),0);
    for(const auto &value:m_channels){const auto c=value.toObject();target->addItem(c.value("name").toString(),c.value("id").toVariant().toLongLong());}
    auto *allDay=new QCheckBox(QStringLiteral("All day"),&dialog);
    auto *start=new QDateTimeEdit(QDateTime::currentDateTime(),&dialog);start->setCalendarPopup(true);
    auto *end=new QDateTimeEdit(QDateTime::currentDateTime().addSecs(3600),&dialog);end->setCalendarPopup(true);
    form->addRow(QStringLiteral("Title:"),title);form->addRow(QStringLiteral("Description:"),description);form->addRow(QStringLiteral("Type:"),type);form->addRow(QStringLiteral("Community:"),target);form->addRow(allDay);form->addRow(QStringLiteral("Start:"),start);form->addRow(QStringLiteral("End:"),end);
    auto *buttons=new QDialogButtonBox(QDialogButtonBox::Ok|QDialogButtonBox::Cancel,&dialog);form->addRow(buttons);
    connect(buttons,&QDialogButtonBox::accepted,&dialog,&QDialog::accept);connect(buttons,&QDialogButtonBox::rejected,&dialog,&QDialog::reject);
    if(editing){
        title->setText(event.value("title").toString());description->setPlainText(event.value("description").toString());
        type->setCurrentIndex(event.value("event_type").toString()=="community"?1:0);
        const qint64 targetId=event.value("target_id").toVariant().toLongLong();int idx=target->findData(targetId);if(idx>=0)target->setCurrentIndex(idx);
        const bool day=event.value("all_day").toBool();allDay->setChecked(day);
        if(day){start->setDate(QDate::fromString(event.value("start_at").toString(),"yyyy-MM-dd"));end->setDate(QDate::fromString(event.value("end_at").toString(),"yyyy-MM-dd"));}
        else{start->setDateTime(QDateTime::fromString(event.value("start_at").toString(),Qt::ISODate).toLocalTime());end->setDateTime(QDateTime::fromString(event.value("end_at").toString(),Qt::ISODate).toLocalTime());}
    }
    if(dialog.exec()!=QDialog::Accepted)return;
    QJsonObject payload{{"title",title->text().trimmed()},{"description",description->toPlainText().trimmed()},{"event_type",type->currentData().toString()},{"target_type",target->currentData().toLongLong()>0?"chat":"none"},{"target_id",target->currentData().toLongLong()},{"all_day",allDay->isChecked()}};
    if(allDay){payload["start_at"]=start->date().toString("yyyy-MM-dd");payload["end_at"]=end->date().toString("yyyy-MM-dd");}
    else{
        payload["start_at"]=start->dateTime().toUTC().toString("yyyy-MM-dd'T'HH:mm:ss'Z'");
        payload["end_at"]=end->dateTime().toUTC().toString("yyyy-MM-dd'T'HH:mm:ss'Z'");
    }
    if(editing)m_client->updateEvent(event.value("id").toVariant().toLongLong(),payload);else m_client->createEvent(payload);
}
