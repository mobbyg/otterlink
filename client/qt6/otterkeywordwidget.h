#pragma once

#include <QJsonObject>
#include <QWidget>

class QLabel;
class QVBoxLayout;

class OtterKeywordWidget final : public QWidget
{
    Q_OBJECT

public:
    explicit OtterKeywordWidget(const QJsonObject &keyword, QWidget *parent = nullptr);

signals:
    void serviceRequested(const QString &type, qint64 id);

private:
    QJsonObject m_keyword;
};
