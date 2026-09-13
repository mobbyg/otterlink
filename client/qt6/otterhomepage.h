#pragma once

#include <QWidget>

class OtterHomePage final : public QWidget
{
    Q_OBJECT

public:
    explicit OtterHomePage(QWidget *parent = nullptr);

signals:
    void serviceRequested(const QString &service);

private:
    void addAnnouncement(const QString &title, const QString &body, const QString &actionLabel,
                         const QString &service, QWidget *parent);
    void addServiceTile(const QString &title, const QString &description, const QString &icon,
                        const QString &service, QWidget *parent);
};
