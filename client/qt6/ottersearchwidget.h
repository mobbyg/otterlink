#pragma once

#include <QWidget>

class QLineEdit;

class OtterSearchWidget final : public QWidget
{
    Q_OBJECT

public:
    explicit OtterSearchWidget(QWidget *parent = nullptr);
};
