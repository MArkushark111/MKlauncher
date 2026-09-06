#include <QApplication>
#include <QDesktopServices>
#include <QFormLayout>
#include <QLineEdit>
#include <QMessageBox>
#include <QPushButton>
#include <QUrl>
#include <QVBoxLayout>
#include <QWidget>

int main(int argc, char *argv[]) {
    QApplication app(argc, argv);

    QWidget window;
    window.setWindowTitle("MKGames Admin");
    window.setMinimumWidth(420);

    auto *layout = new QVBoxLayout(&window);
    auto *form = new QFormLayout();
    auto *serverUrl = new QLineEdit("http://127.0.0.1:8080");
    serverUrl->setPlaceholderText("http://WAN:port");
    form->addRow("Server address", serverUrl);
    layout->addLayout(form);

    auto *openButton = new QPushButton("OPEN ADMIN PANEL");
    layout->addWidget(openButton);

    QObject::connect(openButton, &QPushButton::clicked, [&]() {
        QString address = serverUrl->text().trimmed();
        if (address.isEmpty()) {
            QMessageBox::warning(&window, "Missing address", "Enter the server address first.");
            return;
        }
        if (!address.startsWith("http://") && !address.startsWith("https://")) {
            address = "http://" + address;
        }
        if (!QDesktopServices::openUrl(QUrl(address))) {
            QMessageBox::warning(&window, "Unable to open", "No browser could open the admin panel.");
        }
    });

    window.show();
    return app.exec();
}