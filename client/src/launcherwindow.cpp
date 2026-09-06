#include "launcherwindow.h"
#include "theme.h"

#include <QVBoxLayout>
#include <QHBoxLayout>
#include <QFrame>
#include <QMessageBox>
#include <QDesktopServices>
#include <QProcess>
#include <QJsonDocument>
#include <QJsonObject>
#include <QFile>
#include <QDir>
#include <QFileInfo>
#include <QDebug>

LauncherWindow::LauncherWindow(QWidget *parent) : QMainWindow(parent) {
    setWindowTitle("MKLAUNCHER");
    setMinimumSize(1000, 700);
    setStyleSheet(Theme::getStyleSheet());

    m_downloadManager = new DownloadManager(this);
    m_extractor = new ArchiveExtractor(this);
    m_uninstaller = new Uninstaller(this);
    m_localDB = new LocalDB(this);
    m_updater = new Updater(this);
    m_authManager = new QNetworkAccessManager(this);

    m_stack = new QStackedWidget(this);
    setCentralWidget(m_stack);

    setupConnectPage();
    setupMainPage();
    m_stack->addWidget(m_connectPage);
    m_stack->addWidget(m_mainPage);

    if (!m_settings.firstRun() && !m_settings.serverUrl().isEmpty()) {
        m_urlInput->setText(m_settings.serverUrl());
        m_stack->setCurrentIndex(1);
        connectToServer(m_settings.serverUrl(), "");
    }

    m_updateTimer = new QTimer(this);
    connect(m_updateTimer, &QTimer::timeout, this, &LauncherWindow::checkLauncherUpdates);
    m_updateTimer->start(3600000);

    connect(m_authManager, &QNetworkAccessManager::finished, this, &LauncherWindow::onAuthResult);
    connect(m_downloadManager, &DownloadManager::downloadProgress, this, &LauncherWindow::onDownloadProgress);
    connect(m_downloadManager, &DownloadManager::downloadComplete, this, &LauncherWindow::onDownloadComplete);
    connect(m_downloadManager, &DownloadManager::downloadError, this, &LauncherWindow::onDownloadError);
    connect(m_extractor, &ArchiveExtractor::extractionProgress, this, &LauncherWindow::onExtractionProgress);
    connect(m_extractor, &ArchiveExtractor::extractionComplete, this, &LauncherWindow::onExtractionComplete);
    connect(m_extractor, &ArchiveExtractor::extractionError, this, &LauncherWindow::onExtractionError);

    m_trayIcon = new QSystemTrayIcon(this);
}

void LauncherWindow::setupConnectPage() {
    m_connectPage = new QWidget();

    auto *layout = new QVBoxLayout(m_connectPage);
    layout->setAlignment(Qt::AlignCenter);

    auto *card = new QWidget();
    card->setFixedSize(400, 290);
    card->setStyleSheet(Theme::cardStyle());

    auto *cardLayout = new QVBoxLayout(card);
    cardLayout->setContentsMargins(32, 32, 32, 32);
    cardLayout->setSpacing(16);

    auto *logo = new QLabel("MK<span style='color:#00ff88'>LAUNCHER</span>");
    logo->setTextFormat(Qt::RichText);
    logo->setAlignment(Qt::AlignCenter);
    logo->setStyleSheet("font-size: 28px; font-weight: bold; letter-spacing: 3px; background: transparent;");
    cardLayout->addWidget(logo);

    auto *subtitle = new QLabel("CONNECT TO SERVER");
    subtitle->setAlignment(Qt::AlignCenter);
    subtitle->setStyleSheet("color: #888888; font-size: 11px; letter-spacing: 4px; background: transparent;");
    cardLayout->addWidget(subtitle);

    cardLayout->addSpacing(16);

    m_urlInput = new QLineEdit();
    m_urlInput->setPlaceholderText("WAN:port (e.g. 203.0.113.50:8080)");
    cardLayout->addWidget(m_urlInput);

    m_connectBtn = new QPushButton("CONNECT");
    m_connectBtn->setStyleSheet(
        "background-color: #00ff88; color: #000000; font-size: 14px; padding: 12px; border: none;");
    m_connectBtn->setCursor(Qt::PointingHandCursor);
    connect(m_connectBtn, &QPushButton::clicked, this, &LauncherWindow::onConnectClicked);
    cardLayout->addWidget(m_connectBtn);

    m_statusLabel = new QLabel();
    m_statusLabel->setAlignment(Qt::AlignCenter);
    m_statusLabel->setStyleSheet("color: #ff4444; font-size: 12px; background: transparent;");
    cardLayout->addWidget(m_statusLabel);

    layout->addWidget(card);
}

void LauncherWindow::setupMainPage() {
    m_mainPage = new QWidget();

    auto *layout = new QVBoxLayout(m_mainPage);
    layout->setContentsMargins(0, 0, 0, 0);
    layout->setSpacing(0);

    auto *header = new QWidget();
    header->setFixedHeight(60);
    header->setStyleSheet("background-color: #111111; border-bottom: 1px solid #2a2a2a;");
    auto *headerLayout = new QHBoxLayout(header);
    headerLayout->setContentsMargins(24, 0, 24, 0);

    auto *headerLogo = new QLabel("MK<span style='color:#00ff88'>LAUNCHER</span>");
    headerLogo->setTextFormat(Qt::RichText);
    headerLogo->setStyleSheet("font-size: 18px; font-weight: bold; background: transparent;");
    headerLayout->addWidget(headerLogo);

    headerLayout->addSpacing(24);

    m_searchInput = new QLineEdit();
    m_searchInput->setPlaceholderText("Search games...");
    m_searchInput->setFixedWidth(250);
    m_searchInput->setStyleSheet(
        "background-color: #1a1a1a; color: #e0e0e0; border: 1px solid #2a2a2a;"
        "border-radius: 4px; padding: 8px 12px; font-size: 13px;");
    connect(m_searchInput, &QLineEdit::textChanged, this, &LauncherWindow::onSearchChanged);
    headerLayout->addWidget(m_searchInput);

    m_filterCombo = new QComboBox();
    m_filterCombo->setFixedWidth(140);
    m_filterCombo->addItems({"All", "Installed", "Not Installed"});
    m_filterCombo->setStyleSheet(
        "background-color: #1a1a1a; color: #e0e0e0; border: 1px solid #2a2a2a;"
        "border-radius: 4px; padding: 8px; font-size: 12px;");
    connect(m_filterCombo, QOverload<int>::of(&QComboBox::currentIndexChanged),
            this, &LauncherWindow::onFilterChanged);
    headerLayout->addWidget(m_filterCombo);

    headerLayout->addStretch();

    m_progressBar = new QProgressBar();
    m_progressBar->setFixedWidth(300);
    m_progressBar->setVisible(false);
    headerLayout->addWidget(m_progressBar);

    m_progressLabel = new QLabel();
    m_progressLabel->setStyleSheet("color: #888888; font-size: 11px; background: transparent;");
    headerLayout->addWidget(m_progressLabel);

    headerLayout->addSpacing(12);

    auto *refreshBtn = new QPushButton("REFRESH");
    refreshBtn->setCursor(Qt::PointingHandCursor);
    connect(refreshBtn, &QPushButton::clicked, this, &LauncherWindow::refreshGames);
    headerLayout->addWidget(refreshBtn);

    auto *settingsBtn = new QPushButton("SETTINGS");
    settingsBtn->setCursor(Qt::PointingHandCursor);
    connect(settingsBtn, &QPushButton::clicked, [this]() {
        m_stack->setCurrentIndex(0);
    });
    headerLayout->addWidget(settingsBtn);

    layout->addWidget(header);

    m_gameGrid = new GameGrid(m_mainPage);
    connect(m_gameGrid, &GameGrid::gamePlay, this, &LauncherWindow::onGamePlay);
    connect(m_gameGrid, &GameGrid::gameDownload, this, &LauncherWindow::onGameDownload);
    connect(m_gameGrid, &GameGrid::gameUpdate, this, &LauncherWindow::onGameUpdate);
    connect(m_gameGrid, &GameGrid::gameUninstall, this, &LauncherWindow::onGameUninstall);
    connect(m_gameGrid, &GameGrid::gamesLoadError, this, [this](const QString &error) {
        m_progressLabel->setText(error.isEmpty() ? QString() : "Server error: " + error);
    });
    layout->addWidget(m_gameGrid);
}

void LauncherWindow::onConnectClicked() {
    QString url = m_urlInput->text().trimmed();

    if (url.isEmpty()) {
        m_statusLabel->setText("Enter server URL");
        return;
    }

    if (!url.startsWith("http://") && !url.startsWith("https://")) {
        url = "http://" + url;
    }

    connectToServer(url, "");
}

void LauncherWindow::connectToServer(const QString &url, const QString &passcode) {
    m_serverUrl = url;
    m_statusLabel->setText("Connecting...");
    m_connectBtn->setEnabled(false);

    if (passcode.isEmpty()) {
        m_stack->setCurrentIndex(1);
        m_settings.setServerUrl(url);
        m_settings.setConnected(true);
        refreshGames();
        return;
    }

    QJsonObject obj;
    obj["passcode"] = passcode;
    QNetworkRequest request(QUrl(url + "/api/auth"));
    request.setHeader(QNetworkRequest::ContentTypeHeader, "application/json");
    request.setTransferTimeout(10000);
    m_authManager->post(request, QJsonDocument(obj).toJson());
}

void LauncherWindow::onAuthResult(QNetworkReply *reply) {
    m_connectBtn->setEnabled(true);

    if (reply->error() != QNetworkReply::NoError) {
        m_statusLabel->setText("Connection failed: " + reply->errorString());
        reply->deleteLater();
        return;
    }

    QByteArray data = reply->readAll();
    QJsonDocument doc = QJsonDocument::fromJson(data);
    QJsonObject obj = doc.object();
    reply->deleteLater();

    if (obj["success"].toBool()) {
        m_authToken = obj["token"].toString();
        m_settings.setAuthToken(m_authToken);
        m_settings.setServerUrl(m_serverUrl);
        m_settings.setConnected(true);
        m_settings.setFirstRun(false);

        m_stack->setCurrentIndex(1);
        refreshGames();
    } else {
        m_statusLabel->setText(obj["error"].toString());
    }
}

void LauncherWindow::refreshGames() {
    if (m_authToken.isEmpty()) {
        m_authToken = m_settings.authToken();
    }
    m_gameGrid->loadGames(m_serverUrl, m_authToken);
}

void LauncherWindow::onGamePlay(int gameId, const QString &name, const QString &exePath, const QString &installPath) {
    launchGame(exePath, installPath);
    m_localDB->updateLastPlayed(gameId);
}

void LauncherWindow::launchGame(const QString &exePath, const QString &installPath) {
    QString fullPath = installPath + "/" + exePath;
    QFileInfo fi(fullPath);
    if (!fi.exists()) {
        QMessageBox::warning(this, "Error", "Game executable not found: " + fullPath);
        return;
    }

#ifdef Q_OS_WIN
    QProcess::startDetached("cmd", {"/c", "start", "", QDir::toNativeSeparators(fullPath)});
#else
    QDesktopServices::openUrl(QUrl::fromLocalFile(fullPath));
#endif
}

void LauncherWindow::onGameDownload(int gameId, const QString &name, const QString &url, qint64 size) {
    QString savePath = gameInstallPath(name) + ".zip";
    m_currentDownloadGameId = gameId;
    m_currentDownloadName = name;

    m_progressBar->setVisible(true);
    m_progressBar->setValue(0);
    m_progressLabel->setText("Downloading: " + name);

    m_downloadManager->startDownload(gameId, name, QUrl(url), savePath);
}

void LauncherWindow::onGameUpdate(int gameId, const QString &name, const QString &localVer, const QString &serverVer, const QString &url) {
    QMessageBox::StandardButton reply = QMessageBox::question(this, "Update Available",
        QString("%1 v%2 is available (current: v%3)\n\nDownload update?")
            .arg(name, serverVer, localVer),
        QMessageBox::Yes | QMessageBox::No);

    if (reply == QMessageBox::Yes) {
        LocalGame game = m_localDB->getGame(gameId);
        m_localDB->removeGame(gameId);
        if (!game.installPath.isEmpty()) {
            QDir(game.installPath).removeRecursively();
        }
        onGameDownload(gameId, name, url, 0);
    }
}

void LauncherWindow::onGameUninstall(int gameId, const QString &name, const QString &installPath) {
    QMessageBox::StandardButton reply = QMessageBox::question(this, "Uninstall Game",
        QString("Are you sure you want to uninstall \"%1\"?\nAll game files will be deleted.")
            .arg(name),
        QMessageBox::Yes | QMessageBox::No);

    if (reply == QMessageBox::Yes) {
        m_progressBar->setVisible(true);
        m_progressBar->setValue(0);
        m_progressLabel->setText("Uninstalling: " + name);

        m_localDB->removeGame(gameId);
        m_uninstaller->uninstallGame(installPath, name);
        m_gameGrid->refreshGrid();
    }
}

void LauncherWindow::onDownloadProgress(int gameId, int percent, double speed, const QString &eta) {
    m_progressBar->setValue(percent);

    QString speedStr;
    if (speed < 1024) speedStr = QString::number(speed, 'f', 0) + " B/s";
    else if (speed < 1048576) speedStr = QString::number(speed / 1024, 'f', 1) + " KB/s";
    else speedStr = QString::number(speed / 1048576, 'f', 2) + " MB/s";

    m_progressLabel->setText(QString("%1 - %2% - %3 - ETA: %4")
        .arg(m_currentDownloadName).arg(percent).arg(speedStr).arg(eta));
}

void LauncherWindow::onDownloadComplete(int gameId, const QString &filePath) {
    m_progressLabel->setText("Extracting: " + m_currentDownloadName);
    m_progressBar->setValue(0);

    QString destDir = gameInstallPath(m_currentDownloadName);
    m_pendingArchivePath = filePath;
    m_extractor->extract(filePath, destDir);
}

void LauncherWindow::onDownloadError(int gameId, const QString &error) {
    m_progressBar->setVisible(false);
    m_progressLabel->setText("");
    QMessageBox::warning(this, "Download Error", "Failed to download: " + error);
}

void LauncherWindow::onExtractionProgress(int percent, const QString &currentFile) {
    if (percent >= 0) {
        m_progressBar->setValue(percent);
    }
    if (!currentFile.isEmpty()) {
        m_progressLabel->setText("Extracting: " + currentFile);
    }
}

void LauncherWindow::onExtractionComplete(const QString &destDir) {
    if (!m_pendingArchivePath.isEmpty()) {
        QFile::remove(m_pendingArchivePath);
        m_pendingArchivePath.clear();
    }

    m_progressBar->setVisible(false);
    m_progressLabel->setText("");

    for (const auto &game : m_gameGrid->games()) {
        if (game.id == m_currentDownloadGameId) {
            LocalGame local;
            local.serverGameId = game.id;
            local.name = game.name;
            local.installPath = destDir;
            local.version = game.version;
            local.exePath = game.exePath;
            local.coverUrl = game.coverUrl;
            local.category = game.category;
            local.tags = game.tags;
            local.fileSize = game.fileSize;
            local.installedAt = QDateTime::currentDateTime();
            m_localDB->addGame(local);
            break;
        }
    }

    m_gameGrid->refreshGrid();
    showNotification("Installation Complete", m_currentDownloadName + " is ready to play!");
}

void LauncherWindow::onExtractionError(const QString &error) {
    m_progressBar->setVisible(false);
    m_progressLabel->setText("");
    QMessageBox::warning(this, "Extraction Error", "Failed to extract: " + error);
}

void LauncherWindow::onUninstallComplete(const QString &gameName) {
    m_progressBar->setVisible(false);
    m_progressLabel->setText("");
    showNotification("Uninstall Complete", gameName + " has been removed");
}

void LauncherWindow::showNotification(const QString &title, const QString &msg) {
    m_trayIcon->showMessage(title, msg, QSystemTrayIcon::Information, 3000);
}

void LauncherWindow::checkLauncherUpdates() {
    if (m_settings.autoCheckUpdates() && !m_serverUrl.isEmpty()) {
        m_updater->checkForUpdates("1.0.0");
    }
}

QString LauncherWindow::gameInstallPath(const QString &name) {
    return m_settings.installDir() + "/" + name;
}

void LauncherWindow::onSearchChanged(const QString &text) {
    m_gameGrid->filterByText(text);
}

void LauncherWindow::onFilterChanged(int index) {
    QString filter;
    switch (index) {
        case 1: filter = "installed"; break;
        case 2: filter = "not_installed"; break;
        default: filter = ""; break;
    }
    m_gameGrid->filterByStatus(filter);
}
