#include <QApplication>
#include <QDir>
#include <QFileDialog>
#include <QFormLayout>
#include <QHBoxLayout>
#include <QLabel>
#include <QLineEdit>
#include <QMessageBox>
#include <QPushButton>
#include <QSaveFile>
#include <QVBoxLayout>
#include <QWidget>

int main(int argc, char *argv[]) {
    QApplication app(argc, argv);

    QWidget window;
    window.setWindowTitle("MKGames Server Setup");
    window.setMinimumWidth(480);

    auto *layout = new QVBoxLayout(&window);
    auto *title = new QLabel("MKGames Server Setup");
    title->setStyleSheet("font-size: 22px; font-weight: bold;");
    layout->addWidget(title);
    layout->addWidget(new QLabel("Choose the admin password used by the admin panel."));

    auto *form = new QFormLayout();
    auto *serverPath = new QLineEdit(QDir::currentPath());
    auto *browse = new QPushButton("Browse...");
    auto *pathRow = new QWidget();
    auto *pathLayout = new QHBoxLayout(pathRow);
    pathLayout->setContentsMargins(0, 0, 0, 0);
    pathLayout->addWidget(serverPath);
    pathLayout->addWidget(browse);
    form->addRow("Server folder", pathRow);

    auto *password = new QLineEdit();
    password->setEchoMode(QLineEdit::Password);
    auto *confirm = new QLineEdit();
    confirm->setEchoMode(QLineEdit::Password);
    form->addRow("Admin password", password);
    form->addRow("Confirm password", confirm);
    layout->addLayout(form);

    auto *save = new QPushButton("Save Setup");
    layout->addWidget(save);

    QObject::connect(browse, &QPushButton::clicked, [&]() {
        const QString path = QFileDialog::getExistingDirectory(&window, "Select server folder", serverPath->text());
        if (!path.isEmpty()) {
            serverPath->setText(path);
        }
    });

    QObject::connect(save, &QPushButton::clicked, [&]() {
        const QString selectedPath = serverPath->text().trimmed();
        const QString selectedPassword = password->text();
        if (selectedPath.isEmpty() || selectedPassword.isEmpty()) {
            QMessageBox::warning(&window, "Missing information", "Server folder and password are required.");
            return;
        }
        if (selectedPassword != confirm->text()) {
            QMessageBox::warning(&window, "Password mismatch", "The passwords do not match.");
            return;
        }

        QDir().mkpath(selectedPath);
        QSaveFile envFile(QDir(selectedPath).filePath("server.env"));
        if (!envFile.open(QIODevice::WriteOnly | QIODevice::Text)) {
            QMessageBox::critical(&window, "Save failed", envFile.errorString());
            return;
        }
        envFile.write("MKGAMES_ADMIN_PASSCODE=" + selectedPassword.toUtf8() + "\n");
        envFile.write("MKGAMES_ADMIN_WEB_DIR=" + QDir(selectedPath).filePath("..\\admin\\web").toUtf8() + "\n");
        if (!envFile.commit()) {
            QMessageBox::critical(&window, "Save failed", envFile.errorString());
            return;
        }
        QMessageBox::information(&window, "Setup saved", "The server.env file was created. You can now start the server.");
        window.close();
    });

    window.show();
    return app.exec();
}