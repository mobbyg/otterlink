#include <QApplication>

#include "mainwindow.h"
#include "otteridentitydisplay.h"
#include "otterlinkstyle.h"

int main(int argc, char *argv[])
{
    QApplication app(argc, argv);
    app.setApplicationName("Otter Link");
    app.setOrganizationName("Otter Link");
    OtterLinkStyle::install(app);

    MainWindow window;
    OtterIdentityDisplay::install(window);
    window.show();
    return app.exec();
}
