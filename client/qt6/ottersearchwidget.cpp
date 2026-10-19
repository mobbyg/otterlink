#include "ottersearchwidget.h"

#include <QFrame>
#include <QLabel>
#include <QLineEdit>
#include <QVBoxLayout>

OtterSearchWidget::OtterSearchWidget(QWidget *parent)
    : QWidget(parent)
{
    setObjectName(QStringLiteral("otterSearchWidget"));
    auto *root = new QVBoxLayout(this);
    root->setContentsMargins(24, 24, 24, 24);
    root->setSpacing(12);
    root->addStretch(1);

    auto *masthead = new QLabel(QStringLiteral("Otter Link Search"), this);
    masthead->setObjectName(QStringLiteral("otterSearchTitle"));
    QFont titleFont = masthead->font();
    titleFont.setPointSize(titleFont.pointSize() + 8);
    titleFont.setBold(true);
    masthead->setFont(titleFont);
    masthead->setAlignment(Qt::AlignCenter);
    root->addWidget(masthead);

    auto *subtitle = new QLabel(
        QStringLiteral("Find your way around the online world of Otter Link."),
        this);
    subtitle->setAlignment(Qt::AlignCenter);
    subtitle->setWordWrap(true);
    root->addWidget(subtitle);

    auto *searchBox = new QLineEdit(this);
    searchBox->setObjectName(QStringLiteral("otterSearchComingSoon"));
    searchBox->setPlaceholderText(QStringLiteral("COMING SOON"));
    searchBox->setText(QStringLiteral("COMING SOON"));
    searchBox->setAlignment(Qt::AlignCenter);
    searchBox->setReadOnly(true);
    searchBox->setToolTip(QStringLiteral("Otter Link Search is not available yet."));
    searchBox->setMinimumHeight(42);
    root->addWidget(searchBox);
    root->addStretch(1);
}
