#include <QApplication>

#include "mainwindow.h"

int main(int argc, char *argv[])
{
    QApplication app(argc, argv);
    app.setApplicationName("Otter Link");
    app.setOrganizationName("Otter Link");

    MainWindow window;
    window.show();
    return app.exec();
}
