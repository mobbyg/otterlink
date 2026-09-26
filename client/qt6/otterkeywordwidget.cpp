#include "otterkeywordwidget.h"

#include <QFont>
#include <QHBoxLayout>
#include <QLabel>
#include <QPushButton>
#include <QVBoxLayout>

OtterKeywordWidget::OtterKeywordWidget(const QJsonObject &keyword, QWidget *parent)
    : QWidget(parent), m_keyword(keyword)
{
    auto *layout = new QVBoxLayout(this);
    layout->setContentsMargins(18, 18, 18, 18);
    layout->setSpacing(10);

    auto *title = new QLabel(keyword.value(QStringLiteral("display_name")).toString(), this);
    QFont titleFont = title->font();
    titleFont.setPointSize(titleFont.pointSize() + 5);
    titleFont.setBold(true);
    title->setFont(titleFont);
    title->setAlignment(Qt::AlignCenter);
    layout->addWidget(title);

    const QString description = keyword.value(QStringLiteral("description")).toString();
    if (!description.isEmpty()) {
        auto *desc = new QLabel(description, this);
        desc->setWordWrap(true);
        desc->setAlignment(Qt::AlignCenter);
        layout->addWidget(desc);
    }

    auto *services = new QVBoxLayout;
    for (const QJsonValue &value : keyword.value(QStringLiteral("targets")).toArray()) {
        const QJsonObject target = value.toObject();
        const QString type = target.value(QStringLiteral("type")).toString();
        const qint64 id = target.value(QStringLiteral("id")).toVariant().toLongLong();
        const QString label = target.value(QStringLiteral("label")).toString();
        if (type.isEmpty() || id < 1 || label.isEmpty())
            continue;

        auto *button = new QPushButton(label, this);
        button->setMinimumHeight(36);
        services->addWidget(button);
        connect(button, &QPushButton::clicked, this, [this, type, id]() {
            emit serviceRequested(type, id);
        });
    }

    if (services->count() == 0) {
        auto *empty = new QLabel(QStringLiteral("No services are currently attached to this keyword."), this);
        empty->setAlignment(Qt::AlignCenter);
        layout->addWidget(empty);
    } else {
        auto *heading = new QLabel(QStringLiteral("Services"), this);
        heading->setAlignment(Qt::AlignCenter);
        layout->addWidget(heading);
        layout->addLayout(services);
    }

    layout->addStretch(1);
}
