#include <QApplication>
#include "launcherwindow.h"
#include "theme.h"

int main(int argc, char *argv[]) {
    QApplication app(argc, argv);
    app.setApplicationName("MKLauncher");
    app.setOrganizationName("MKGames");
    app.setApplicationVersion("1.0.1");

    app.setStyleSheet(Theme::getStyleSheet());

    LauncherWindow window;
    window.show();

    return app.exec();
}
