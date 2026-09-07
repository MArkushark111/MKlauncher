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
#include <QDialog>
#include <QCheckBox>
#include <QSpinBox>
#include <QGroupBox>
#include <QFileDialog>
#include <QTabWidget>
#include <QFormLayout>
#include <QScrollArea>

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
    m_reviewManager = new QNetworkAccessManager(this);

    m_stack = new QStackedWidget(this);
    setCentralWidget(m_stack);

    setupConnectPage();
    setupMainPage();
    setupSettingsPage();
    m_stack->addWidget(m_connectPage);
    m_stack->addWidget(m_mainPage);
    m_stack->addWidget(m_settingsPage);

    if (!m_settings.firstRun() && !m_settings.serverUrl().isEmpty()) {
        m_urlInput->setText(m_settings.serverUrl());
        m_stack->setCurrentIndex(1);
        connectToServer(m_settings.serverUrl(), "");
    }

    addDefenderExclusion(m_settings.installDir());

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
        m_stack->setCurrentIndex(2);
    });
    headerLayout->addWidget(settingsBtn);

    auto *libraryBtn = new QPushButton("LIBRARY");
    libraryBtn->setCursor(Qt::PointingHandCursor);
    connect(libraryBtn, &QPushButton::clicked, [this]() {
        setupLibraryTab();
    });
    headerLayout->addWidget(libraryBtn);

    layout->addWidget(header);

    m_gameGrid = new GameGrid(m_mainPage);
    connect(m_gameGrid, &GameGrid::gamePlay, this, &LauncherWindow::onGamePlay);
    connect(m_gameGrid, &GameGrid::gameDownload, this, &LauncherWindow::onGameDownload);
    connect(m_gameGrid, &GameGrid::gameUpdate, this, &LauncherWindow::onGameUpdate);
    connect(m_gameGrid, &GameGrid::gameUninstall, this, &LauncherWindow::onGameUninstall);
    connect(m_gameGrid, &GameGrid::gamesLoadError, this, [this](const QString &error) {
        m_progressLabel->setText(error.isEmpty() ? QString() : "Server error: " + error);
    });
    connect(m_gameGrid, &GameGrid::gameDetails, this, &LauncherWindow::onGameDetails);
    layout->addWidget(m_gameGrid);
}

void LauncherWindow::setupSettingsPage() {
    m_settingsPage = new QWidget();
    m_settingsPage->setStyleSheet("background-color: #0a0a0a;");

    auto *outerLayout = new QVBoxLayout(m_settingsPage);
    outerLayout->setContentsMargins(0, 0, 0, 0);
    outerLayout->setSpacing(0);

    auto *topBar = new QWidget();
    topBar->setFixedHeight(60);
    topBar->setStyleSheet("background-color: #111111; border-bottom: 1px solid #2a2a2a;");
    auto *topBarLayout = new QHBoxLayout(topBar);
    topBarLayout->setContentsMargins(24, 0, 24, 0);

    auto *backBtn = new QPushButton("BACK");
    backBtn->setCursor(Qt::PointingHandCursor);
    connect(backBtn, &QPushButton::clicked, [this]() {
        m_stack->setCurrentIndex(1);
        refreshGames();
    });
    topBarLayout->addWidget(backBtn);

    topBarLayout->addSpacing(16);

    auto *settingsTitle = new QLabel("SETTINGS");
    settingsTitle->setStyleSheet("font-size: 18px; font-weight: bold; letter-spacing: 3px; color: #00ff88; background: transparent;");
    topBarLayout->addWidget(settingsTitle);

    topBarLayout->addStretch();
    outerLayout->addWidget(topBar);

    auto *tabs = new QTabWidget();
    tabs->setStyleSheet(R"(
        QTabWidget::pane { border: none; background: #0a0a0a; }
        QTabBar::tab {
            background: #111111; color: #888888; border: none;
            padding: 12px 24px; font-size: 12px; font-weight: bold;
            letter-spacing: 1px; border-bottom: 2px solid transparent;
        }
        QTabBar::tab:selected { color: #00ff88; border-bottom: 2px solid #00ff88; }
        QTabBar::tab:hover { color: #e0e0e0; }
    )");

    QWidget *generalTab = new QWidget();
    QWidget *downloadsTab = new QWidget();
    QWidget *appearanceTab = new QWidget();
    QWidget *storageTab = new QWidget();
    QWidget *aboutTab = new QWidget();

    tabs->addTab(generalTab, "GENERAL");
    tabs->addTab(downloadsTab, "DOWNLOADS");
    tabs->addTab(appearanceTab, "APPEARANCE");
    tabs->addTab(storageTab, "STORAGE");
    tabs->addTab(aboutTab, "ABOUT");

    auto makeScroll = [](QWidget *tab, QWidget *content) {
        auto *scroll = new QScrollArea();
        scroll->setWidgetResizable(true);
        scroll->setFrameShape(QFrame::NoFrame);
        scroll->setStyleSheet("background: transparent; border: none;");
        content->setStyleSheet("background: transparent;");
        scroll->setWidget(content);
        auto *layout = new QVBoxLayout(tab);
        layout->setContentsMargins(0, 0, 0, 0);
        layout->addWidget(scroll);
    };

    auto cardStyle = R"(
        QGroupBox {
            background-color: #1a1a1a; border: 1px solid #2a2a2a;
            border-radius: 8px; margin-top: 12px; padding: 20px 16px 16px 16px;
            font-size: 13px; font-weight: bold; letter-spacing: 1px; color: #00ff88;
        }
        QGroupBox::title { subcontrol-origin: margin; left: 16px; top: 4px; padding: 0 8px; }
    )";

    auto labelStyle = "color: #888888; font-size: 11px; letter-spacing: 1px; background: transparent;";
    auto valueStyle = "color: #e0e0e0; font-size: 13px; background: transparent;";

    {   // === GENERAL TAB ===
        auto *content = new QWidget();
        auto *layout = new QVBoxLayout(content);
        layout->setContentsMargins(32, 24, 32, 24);
        layout->setSpacing(16);

        auto *serverGroup = new QGroupBox("SERVER CONNECTION");
        serverGroup->setStyleSheet(cardStyle);
        auto *serverLayout = new QFormLayout(serverGroup);
        serverLayout->setSpacing(12);

        auto *urlLabel = new QLabel(m_settings.serverUrl().isEmpty() ? "Not connected" : m_settings.serverUrl());
        urlLabel->setStyleSheet(valueStyle);
        serverLayout->addRow("SERVER URL", urlLabel);

        auto *statusLabel = new QLabel(m_settings.isConnected() ? "Connected" : "Disconnected");
        statusLabel->setStyleSheet(m_settings.isConnected() ? "color: #00ff88; font-size: 13px; background: transparent;" : "color: #ff4444; font-size: 13px; background: transparent;");
        serverLayout->addRow("STATUS", statusLabel);

        layout->addWidget(serverGroup);

        auto *installGroup = new QGroupBox("INSTALL DIRECTORY");
        installGroup->setStyleSheet(cardStyle);
        auto *installLayout = new QVBoxLayout(installGroup);
        installLayout->setSpacing(12);

        auto *installPathLabel = new QLabel(m_settings.installDir());
        installPathLabel->setStyleSheet(valueStyle);
        installPathLabel->setWordWrap(true);
        installLayout->addWidget(installPathLabel);

        auto *changeDirBtn = new QPushButton("CHANGE DIRECTORY");
        changeDirBtn->setCursor(Qt::PointingHandCursor);
        connect(changeDirBtn, &QPushButton::clicked, [this, installPathLabel]() {
            QString dir = QFileDialog::getExistingDirectory(this, "Select Install Directory", m_settings.installDir());
            if (!dir.isEmpty()) {
                m_settings.setInstallDir(dir);
                installPathLabel->setText(dir);
            }
        });
        installLayout->addWidget(changeDirBtn);
        layout->addWidget(installGroup);

        auto *updatesGroup = new QGroupBox("UPDATES");
        updatesGroup->setStyleSheet(cardStyle);
        auto *updatesLayout = new QFormLayout(updatesGroup);
        updatesLayout->setSpacing(12);

        auto *autoUpdateCheck = new QCheckBox("Automatically check for launcher updates");
        autoUpdateCheck->setChecked(m_settings.autoCheckUpdates());
        autoUpdateCheck->setStyleSheet("color: #e0e0e0; font-size: 13px; background: transparent;");
        connect(autoUpdateCheck, &QCheckBox::toggled, [this](bool val) { m_settings.setAutoCheckUpdates(val); });
        updatesLayout->addRow(autoUpdateCheck);

        auto *keepUpdatedCheck = new QCheckBox("Keep game library updated on launch");
        keepUpdatedCheck->setChecked(m_settings.keepLibraryUpdated());
        keepUpdatedCheck->setStyleSheet("color: #e0e0e0; font-size: 13px; background: transparent;");
        connect(keepUpdatedCheck, &QCheckBox::toggled, [this](bool val) { m_settings.setKeepLibraryUpdated(val); });
        updatesLayout->addRow(keepUpdatedCheck);

        layout->addWidget(updatesGroup);
        layout->addStretch();
        makeScroll(generalTab, content);
    }

    {   // === DOWNLOADS TAB ===
        auto *content = new QWidget();
        auto *layout = new QVBoxLayout(content);
        layout->setContentsMargins(32, 24, 32, 24);
        layout->setSpacing(16);

        auto *queueGroup = new QGroupBox("DOWNLOAD QUEUE");
        queueGroup->setStyleSheet(cardStyle);
        auto *queueLayout = new QFormLayout(queueGroup);
        queueLayout->setSpacing(12);

        auto *maxDownloadsSpin = new QSpinBox();
        maxDownloadsSpin->setRange(1, 10);
        maxDownloadsSpin->setValue(m_settings.maxConcurrentDownloads());
        maxDownloadsSpin->setStyleSheet("background-color: #111111; color: #e0e0e0; border: 1px solid #2a2a2a; border-radius: 4px; padding: 8px; font-size: 14px;");
        connect(maxDownloadsSpin, QOverload<int>::of(&QSpinBox::valueChanged), [this](int val) { m_settings.setMaxConcurrentDownloads(val); });
        queueLayout->addRow("MAX CONCURRENT DOWNLOADS", maxDownloadsSpin);

        layout->addWidget(queueGroup);

        auto *extractGroup = new QGroupBox("EXTRACTION");
        extractGroup->setStyleSheet(cardStyle);
        auto *extractLayout = new QFormLayout(extractGroup);
        extractLayout->setSpacing(12);

        auto *autoExtractCheck = new QCheckBox("Automatically extract archives after download");
        autoExtractCheck->setChecked(m_settings.autoExtract());
        autoExtractCheck->setStyleSheet("color: #e0e0e0; font-size: 13px; background: transparent;");
        connect(autoExtractCheck, &QCheckBox::toggled, [this](bool val) { m_settings.setAutoExtract(val); });
        extractLayout->addRow(autoExtractCheck);

        auto *deleteArchiveCheck = new QCheckBox("Delete archive after successful extraction");
        deleteArchiveCheck->setChecked(m_settings.deleteArchiveAfterInstall());
        deleteArchiveCheck->setStyleSheet("color: #e0e0e0; font-size: 13px; background: transparent;");
        connect(deleteArchiveCheck, &QCheckBox::toggled, [this](bool val) { m_settings.setDeleteArchiveAfterInstall(val); });
        extractLayout->addRow(deleteArchiveCheck);

        layout->addWidget(extractGroup);

        auto *launchGroup = new QGroupBox("LAUNCH BEHAVIOR");
        launchGroup->setStyleSheet(cardStyle);
        auto *launchLayout = new QFormLayout(launchGroup);
        launchLayout->setSpacing(12);

        auto *launchAfterCheck = new QCheckBox("Launch game automatically after install");
        launchAfterCheck->setChecked(m_settings.launchAfterInstall());
        launchAfterCheck->setStyleSheet("color: #e0e0e0; font-size: 13px; background: transparent;");
        connect(launchAfterCheck, &QCheckBox::toggled, [this](bool val) { m_settings.setLaunchAfterInstall(val); });
        launchLayout->addRow(launchAfterCheck);

        layout->addWidget(launchGroup);
        layout->addStretch();
        makeScroll(downloadsTab, content);
    }

    {   // === APPEARANCE TAB ===
        auto *content = new QWidget();
        auto *layout = new QVBoxLayout(content);
        layout->setContentsMargins(32, 24, 32, 24);
        layout->setSpacing(16);

        auto *themeGroup = new QGroupBox("THEME");
        themeGroup->setStyleSheet(cardStyle);
        auto *themeLayout = new QFormLayout(themeGroup);
        themeLayout->setSpacing(12);

        auto *darkModeCheck = new QCheckBox("Dark mode (recommended)");
        darkModeCheck->setChecked(m_settings.darkMode());
        darkModeCheck->setStyleSheet("color: #e0e0e0; font-size: 13px; background: transparent;");
        connect(darkModeCheck, &QCheckBox::toggled, [this](bool val) { m_settings.setDarkMode(val); });
        themeLayout->addRow(darkModeCheck);

        layout->addWidget(themeGroup);

        auto *gridGroup = new QGroupBox("GAME GRID");
        gridGroup->setStyleSheet(cardStyle);
        auto *gridLayout = new QFormLayout(gridGroup);
        gridLayout->setSpacing(12);

        auto *showSizeCheck = new QCheckBox("Show game size on install button");
        showSizeCheck->setChecked(m_settings.showGameSizeInGrid());
        showSizeCheck->setStyleSheet("color: #e0e0e0; font-size: 13px; background: transparent;");
        connect(showSizeCheck, &QCheckBox::toggled, [this](bool val) { m_settings.setShowGameSizeInGrid(val); });
        gridLayout->addRow(showSizeCheck);

        layout->addWidget(gridGroup);

        auto *notifGroup = new QGroupBox("NOTIFICATIONS");
        notifGroup->setStyleSheet(cardStyle);
        auto *notifLayout = new QFormLayout(notifGroup);
        notifLayout->setSpacing(12);

        auto *notifCheck = new QCheckBox("Show desktop notifications");
        notifCheck->setChecked(m_settings.showNotifications());
        notifCheck->setStyleSheet("color: #e0e0e0; font-size: 13px; background: transparent;");
        connect(notifCheck, &QCheckBox::toggled, [this](bool val) { m_settings.setShowNotifications(val); });
        notifLayout->addRow(notifCheck);

        auto *trayCheck = new QCheckBox("Minimize to system tray on close");
        trayCheck->setChecked(m_settings.closeToTray());
        trayCheck->setStyleSheet("color: #e0e0e0; font-size: 13px; background: transparent;");
        connect(trayCheck, &QCheckBox::toggled, [this](bool val) { m_settings.setCloseToTray(val); });
        notifLayout->addRow(trayCheck);

        layout->addWidget(notifGroup);
        layout->addStretch();
        makeScroll(appearanceTab, content);
    }

    {   // === STORAGE TAB ===
        auto *content = new QWidget();
        auto *layout = new QVBoxLayout(content);
        layout->setContentsMargins(32, 24, 32, 24);
        layout->setSpacing(16);

        auto *usageGroup = new QGroupBox("DISK USAGE");
        usageGroup->setStyleSheet(cardStyle);
        auto *usageLayout = new QVBoxLayout(usageGroup);
        usageLayout->setSpacing(12);

        auto *pathLabel = new QLabel(m_settings.installDir());
        pathLabel->setStyleSheet(valueStyle);
        pathLabel->setWordWrap(true);
        usageLayout->addWidget(pathLabel);

        auto *usageInfo = new QLabel("Calculating...");
        usageInfo->setStyleSheet(valueStyle);
        usageLayout->addWidget(usageInfo);

        auto calculateUsage = [usageInfo, this]() {
            qint64 totalSize = 0;
            QDir installDir(m_settings.installDir());
            if (installDir.exists()) {
                for (const auto &entry : installDir.entryInfoList(QDir::Files | QDir::Dirs | QDir::NoDotAndDotDot, QDir::DirsFirst)) {
                    if (entry.isDir()) {
                        QDir subDir(entry.absoluteFilePath());
                        for (const auto &f : subDir.entryInfoList(QDir::Files))
                            totalSize += f.size();
                    } else {
                        totalSize += entry.size();
                    }
                }
            }
            QString sizeStr;
            if (totalSize < 1024) sizeStr = QString::number(totalSize) + " B";
            else if (totalSize < 1048576) sizeStr = QString::number(totalSize / 1024.0, 'f', 1) + " KB";
            else if (totalSize < 1073741824) sizeStr = QString::number(totalSize / 1048576.0, 'f', 1) + " MB";
            else sizeStr = QString::number(totalSize / 1073741824.0, 'f', 2) + " GB";

            int gameCount = m_localDB->getAllGames().size();
            usageInfo->setText(QString("%1 games installed - %2 total").arg(gameCount).arg(sizeStr));
        };
        calculateUsage();

        layout->addWidget(usageGroup);

        auto *actionsGroup = new QGroupBox("ACTIONS");
        actionsGroup->setStyleSheet(cardStyle);
        auto *actionsLayout = new QVBoxLayout(actionsGroup);
        actionsLayout->setSpacing(12);

        auto *refreshUsageBtn = new QPushButton("REFRESH DISK USAGE");
        refreshUsageBtn->setCursor(Qt::PointingHandCursor);
        connect(refreshUsageBtn, &QPushButton::clicked, calculateUsage);
        actionsLayout->addWidget(refreshUsageBtn);

        auto *clearDbBtn = new QPushButton("RESET LOCAL DATABASE");
        clearDbBtn->setCursor(Qt::PointingHandCursor);
        clearDbBtn->setStyleSheet("background-color: #1a1a1a; color: #ffaa00; border: 1px solid #ffaa00; border-radius: 4px; padding: 10px; font-size: 12px; font-weight: bold;");
        connect(clearDbBtn, &QPushButton::clicked, [this, calculateUsage]() {
            auto reply = QMessageBox::question(this, "Reset Database",
                "This will remove all local install records.\nGame files will NOT be deleted.\n\nContinue?",
                QMessageBox::Yes | QMessageBox::No);
            if (reply == QMessageBox::Yes) {
                QList<LocalGame> games = m_localDB->getAllGames();
                for (const auto &g : games)
                    m_localDB->removeGame(g.serverGameId);
                calculateUsage();
                QMessageBox::information(this, "Done", "Local database cleared.");
            }
        });
        actionsLayout->addWidget(clearDbBtn);

        auto *uninstallAllBtn = new QPushButton("UNINSTALL ALL GAMES");
        uninstallAllBtn->setCursor(Qt::PointingHandCursor);
        uninstallAllBtn->setStyleSheet("background-color: #ff4444; color: #ffffff; border: none; border-radius: 4px; padding: 10px; font-size: 12px; font-weight: bold;");
        connect(uninstallAllBtn, &QPushButton::clicked, [this, calculateUsage]() {
            auto reply = QMessageBox::question(this, "Uninstall All Games",
                "This will permanently delete ALL installed games.\nThis cannot be undone!\n\nContinue?",
                QMessageBox::Yes | QMessageBox::No);
            if (reply == QMessageBox::Yes) {
                QList<LocalGame> games = m_localDB->getAllGames();
                for (const auto &g : games) {
                    if (!g.installPath.isEmpty())
                        QDir(g.installPath).removeRecursively();
                    m_localDB->removeGame(g.serverGameId);
                }
                calculateUsage();
                QMessageBox::information(this, "Done", "All games uninstalled.");
            }
        });
        actionsLayout->addWidget(uninstallAllBtn);

        layout->addWidget(actionsGroup);
        layout->addStretch();
        makeScroll(storageTab, content);
    }

    {   // === ABOUT TAB ===
        auto *content = new QWidget();
        auto *layout = new QVBoxLayout(content);
        layout->setContentsMargins(32, 24, 32, 24);
        layout->setSpacing(16);

        auto *aboutGroup = new QGroupBox("MKLAUNCHER");
        aboutGroup->setStyleSheet(cardStyle);
        auto *aboutLayout = new QFormLayout(aboutGroup);
        aboutLayout->setSpacing(12);

        auto *versionLabel = new QLabel("1.0.0");
        versionLabel->setStyleSheet(valueStyle);
        aboutLayout->addRow("VERSION", versionLabel);

        auto *qtLabel = new QLabel(QT_VERSION_STR);
        qtLabel->setStyleSheet(valueStyle);
        aboutLayout->addRow("QT VERSION", qtLabel);

        auto *platformLabel = new QLabel(
#ifdef Q_OS_WIN
            "Windows"
#elif defined(Q_OS_MAC)
            "macOS"
#else
            "Linux"
#endif
        );
        platformLabel->setStyleSheet(valueStyle);
        aboutLayout->addRow("PLATFORM", platformLabel);

        layout->addWidget(aboutGroup);

        auto *resetGroup = new QGroupBox("RESET");
        resetGroup->setStyleSheet(cardStyle);
        auto *resetLayout = new QVBoxLayout(resetGroup);
        resetLayout->setSpacing(12);

        auto *resetBtn = new QPushButton("RESET ALL SETTINGS");
        resetBtn->setCursor(Qt::PointingHandCursor);
        resetBtn->setStyleSheet("background-color: #ff4444; color: #ffffff; border: none; border-radius: 4px; padding: 10px; font-size: 12px; font-weight: bold;");
        connect(resetBtn, &QPushButton::clicked, [this]() {
            auto reply = QMessageBox::question(this, "Reset Settings",
                "This will reset ALL settings to defaults.\nYou will need to reconnect to your server.\n\nContinue?",
                QMessageBox::Yes | QMessageBox::No);
            if (reply == QMessageBox::Yes) {
                m_settings.clearAll();
                resetConnection();
                QMessageBox::information(this, "Done", "Settings reset. Please reconnect.");
            }
        });
        resetLayout->addWidget(resetBtn);

        layout->addWidget(resetGroup);
        layout->addStretch();
        makeScroll(aboutTab, content);
    }

    outerLayout->addWidget(tabs);
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

void LauncherWindow::resetConnection() {
    m_authToken.clear();
    m_settings.setAuthToken(QString());
    m_settings.setConnected(false);
    m_connectBtn->setEnabled(true);
    m_statusLabel->setText("Enter a server address");
    m_stack->setCurrentIndex(0);
}

void LauncherWindow::onGameDetails(const ServerGame &game) {
    showGameDetail(game);
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
        QDir searchDir(installPath);
        QString exeName = QFileInfo(exePath).fileName();
        QStringList found = searchDir.entryList(QStringList() << exeName, QDir::Files, QDir::Name);
        if (!found.isEmpty()) {
            fullPath = searchDir.absoluteFilePath(found.first());
        } else {
            QMessageBox::warning(this, "Error", "Game executable not found: " + fullPath);
            return;
        }
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
    qDebug() << "[DOWNLOAD] Starting download:" << name << "URL:" << url << "Save:" << savePath << "Size:" << size;

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
    qDebug() << "[DOWNLOAD] Complete for game" << gameId << "File:" << filePath;
    QFileInfo fi(filePath);
    qDebug() << "[DOWNLOAD] File exists:" << fi.exists() << "Size:" << fi.size();
    m_progressLabel->setText("Extracting: " + m_currentDownloadName);
    m_progressBar->setValue(0);

    QString destDir = gameInstallPath(m_currentDownloadName);
    qDebug() << "[EXTRACT] Extracting to:" << destDir;
    m_pendingArchivePath = filePath;
    m_extractor->extract(filePath, destDir);
}

void LauncherWindow::onDownloadError(int gameId, const QString &error) {
    m_progressBar->setVisible(false);
    m_progressLabel->setText("");
    QMessageBox::warning(this, "Download Error", "Failed to download: " + error);
}

void LauncherWindow::onExtractionProgress(int percent, const QString &currentFile) {
    qDebug() << "[EXTRACT] Progress:" << percent << "File:" << currentFile;
    if (percent >= 0) {
        m_progressBar->setValue(percent);
    }
    if (!currentFile.isEmpty()) {
        m_progressLabel->setText("Extracting: " + currentFile);
    }
}

void LauncherWindow::onExtractionComplete(const QString &destDir) {
    qDebug() << "[EXTRACT] Complete. Dest:" << destDir;
    if (!m_pendingArchivePath.isEmpty()) {
        QFile::remove(m_pendingArchivePath);
        m_pendingArchivePath.clear();
    }

    m_progressBar->setVisible(false);
    m_progressLabel->setText("");

    QString actualInstallPath = destDir;
    QDir dest(destDir);
    QStringList entries = dest.entryList(QDir::Dirs | QDir::NoDotAndDotDot);
    QStringList files = dest.entryList(QDir::Files | QDir::NoDotAndDotDot);
    qDebug() << "[EXTRACT] Dirs in dest:" << entries << "Files:" << files;

    if (entries.size() == 1 && files.isEmpty()) {
        actualInstallPath = dest.absoluteFilePath(entries.first());
        qDebug() << "[EXTRACT] Detected single subfolder, adjusting install path to:" << actualInstallPath;
        QDir subDest(actualInstallPath);
        qDebug() << "[EXTRACT] Subfolder contents:" << subDest.entryList(QDir::Files | QDir::NoDotAndDotDot);
    }

    bool saved = false;
    for (const auto &game : m_gameGrid->games()) {
        if (game.id == m_currentDownloadGameId) {
            LocalGame local;
            local.serverGameId = game.id;
            local.name = game.name;
            local.installPath = actualInstallPath;
            local.version = game.version;
            local.exePath = game.exePath;
            local.coverUrl = game.coverUrl;
            local.category = game.category;
            local.tags = game.tags;
            local.fileSize = game.fileSize;
            local.installedAt = QDateTime::currentDateTime();

            QString testPath = actualInstallPath + "/" + game.exePath;
            if (!QFileInfo::exists(testPath) && !game.exePath.isEmpty()) {
                QDir searchDir(actualInstallPath);
                QStringList found = searchDir.entryList(QStringList() << QFileInfo(game.exePath).fileName(), QDir::Files);
                if (!found.isEmpty()) {
                    local.exePath = found.first();
                }
            }

            m_localDB->addGame(local);
            saved = true;
            break;
        }
    }

    if (!saved && !m_currentDownloadName.isEmpty()) {
        LocalGame local;
        local.serverGameId = m_currentDownloadGameId;
        local.name = m_currentDownloadName;
        local.installPath = actualInstallPath;
        local.installedAt = QDateTime::currentDateTime();
        m_localDB->addGame(local);
    }

    m_gameGrid->refreshGrid();
    showNotification("Installation Complete", m_currentDownloadName + " is ready to play!");
}

void LauncherWindow::onExtractionError(const QString &error) {
    qDebug() << "[EXTRACT] ERROR:" << error;
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

void LauncherWindow::addDefenderExclusion(const QString &path) {
#ifdef Q_OS_WIN
    QProcess proc;
    proc.start("powershell.exe", {"-NoProfile", "-Command",
        "Add-MpExclusion -ExclusionPath '" + path + "' -ErrorAction SilentlyContinue"});
    proc.waitForFinished(5000);
#endif
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

void LauncherWindow::showGameDetail(const ServerGame &game) {
    if (m_detailPage) { m_detailPage->deleteLater(); m_detailPage = nullptr; }

    m_detailPage = new QWidget();
    m_detailPage->setStyleSheet("background-color: #0a0a0a;");
    auto *outerLayout = new QVBoxLayout(m_detailPage);
    outerLayout->setContentsMargins(0, 0, 0, 0);
    outerLayout->setSpacing(0);

    auto *topBar = new QWidget();
    topBar->setFixedHeight(60);
    topBar->setStyleSheet("background-color: #111111; border-bottom: 1px solid #2a2a2a;");
    auto *topBarLayout = new QHBoxLayout(topBar);
    topBarLayout->setContentsMargins(24, 0, 24, 0);
    auto *backBtn = new QPushButton("BACK");
    backBtn->setCursor(Qt::PointingHandCursor);
    connect(backBtn, &QPushButton::clicked, [this]() {
        m_stack->setCurrentIndex(1);
        m_gameGrid->refreshGrid();
    });
    topBarLayout->addWidget(backBtn);
    topBarLayout->addSpacing(16);
    auto *title = new QLabel(game.name);
    title->setStyleSheet("font-size: 18px; font-weight: bold; color: #00ff88; background: transparent; letter-spacing: 2px;");
    topBarLayout->addWidget(title);
    topBarLayout->addStretch();
    outerLayout->addWidget(topBar);

    auto *scroll = new QScrollArea();
    scroll->setWidgetResizable(true);
    scroll->setFrameShape(QFrame::NoFrame);
    scroll->setStyleSheet("background: transparent; border: none;");

    auto *content = new QWidget();
    content->setStyleSheet("background: transparent;");
    auto *layout = new QVBoxLayout(content);
    layout->setContentsMargins(32, 24, 32, 24);
    layout->setSpacing(16);

    auto *heroWidget = new QWidget();
    heroWidget->setFixedHeight(300);
    heroWidget->setStyleSheet("background-color: #111111; border-radius: 12px; border: 1px solid #2a2a2a;");
    auto *heroLayout = new QHBoxLayout(heroWidget);
    heroLayout->setContentsMargins(0, 0, 0, 0);

    auto *coverLabel = new QLabel();
    coverLabel->setFixedWidth(200);
    coverLabel->setStyleSheet("background-color: #1a1a1a; border-top-left-radius: 12px; border-bottom-left-radius: 12px; color: #555555; font-size: 12px;");
    coverLabel->setAlignment(Qt::AlignCenter);
    coverLabel->setText("LOADING...");
    if (!game.coverUrl.isEmpty()) {
        QUrl coverUrl(m_gameGrid->imageUrl(game.coverUrl));
        QNetworkRequest req(coverUrl);
        req.setTransferTimeout(10000);
        QNetworkReply *reply = m_reviewManager->get(req);
        connect(reply, &QNetworkReply::finished, this, [this, coverLabel, reply]() {
            reply->deleteLater();
            if (reply->error() != QNetworkReply::NoError) return;
            QPixmap pixmap;
            pixmap.loadFromData(reply->readAll());
            if (!pixmap.isNull()) {
                coverLabel->setPixmap(pixmap.scaled(coverLabel->size(), Qt::KeepAspectRatioByExpanding, Qt::SmoothTransformation));
                coverLabel->setText(QString());
            }
        });
    }
    heroLayout->addWidget(coverLabel);

    auto *infoLayout = new QVBoxLayout();
    infoLayout->setContentsMargins(24, 16, 24, 16);
    infoLayout->setSpacing(8);

    auto *nameLabel = new QLabel(game.name);
    nameLabel->setStyleSheet("color: #e0e0e0; font-size: 24px; font-weight: bold; background: transparent;");
    infoLayout->addWidget(nameLabel);

    auto *versionLabel = new QLabel(QString("Version %1 | %2").arg(game.version, game.category.isEmpty() ? "Uncategorized" : game.category));
    versionLabel->setStyleSheet("color: #888888; font-size: 13px; background: transparent;");
    infoLayout->addWidget(versionLabel);

    if (game.reviewCount > 0) {
        QString stars;
        int full = (int)game.avgStars;
        for (int i = 0; i < full; i++) stars += QChar(0x2605);
        for (int i = full; i < 5; i++) stars += QChar(0x2606);
        auto *starLabel = new QLabel(stars + QString("  %1 (%2 reviews)").arg(QString::number(game.avgStars, 'f', 1)).arg(game.reviewCount));
        starLabel->setStyleSheet("color: #ffaa00; font-size: 16px; background: transparent;");
        infoLayout->addWidget(starLabel);
    }

    auto *dlLabel = new QLabel(QString("%1 downloads | %2").arg(game.downloadCount).arg(formatSize(game.fileSize)));
    dlLabel->setStyleSheet("color: #888888; font-size: 12px; background: transparent;");
    infoLayout->addWidget(dlLabel);

    infoLayout->addSpacing(8);

    if (!game.tags.isEmpty()) {
        auto *tagsLabel = new QLabel("Tags: " + game.tags);
        tagsLabel->setStyleSheet("color: #00ff88; font-size: 12px; background: transparent;");
        tagsLabel->setWordWrap(true);
        infoLayout->addWidget(tagsLabel);
    }

    auto *descLabel = new QLabel(game.description.isEmpty() ? "No description available." : game.description);
    descLabel->setStyleSheet("color: #cccccc; font-size: 13px; background: transparent; line-height: 1.4;");
    descLabel->setWordWrap(true);
    infoLayout->addWidget(descLabel);

    infoLayout->addStretch();
    heroLayout->addLayout(infoLayout);
    layout->addWidget(heroWidget);

    LocalDB localDB;
    bool installed = localDB.isInstalled(game.id);

    auto *btnWidget = new QWidget();
    btnWidget->setStyleSheet("background: transparent;");
    auto *btnLayout = new QHBoxLayout(btnWidget);
    btnLayout->setContentsMargins(0, 0, 0, 0);

    if (installed) {
        LocalGame localGame = localDB.getGame(game.id);
        bool needsUpdate = localGame.version != game.version;

        auto *playBtn = new QPushButton("PLAY");
        playBtn->setObjectName("playBtn");
        playBtn->setCursor(Qt::PointingHandCursor);
        playBtn->setStyleSheet(
            "QPushButton#playBtn { background-color: #00ff88; color: #000000; border: none; border-radius: 6px; font-size: 16px; padding: 14px 48px; font-weight: bold; }"
            "QPushButton#playBtn:hover { background-color: #00cc6a; }");
        connect(playBtn, &QPushButton::clicked, [this, game, localGame]() {
            emit m_gameGrid->gamePlay(game.id, game.name, game.exePath, localGame.installPath);
        });
        btnLayout->addWidget(playBtn);

        if (needsUpdate) {
            auto *updateBtn = new QPushButton("UPDATE");
            updateBtn->setObjectName("updateBtn");
            updateBtn->setCursor(Qt::PointingHandCursor);
            updateBtn->setStyleSheet(
                "QPushButton#updateBtn { background-color: #ffaa00; color: #000000; border: none; border-radius: 6px; font-size: 14px; padding: 14px 32px; font-weight: bold; }"
                "QPushButton#updateBtn:hover { background-color: #cc8800; }");
            connect(updateBtn, &QPushButton::clicked, [this, game, localGame]() {
                m_gameGrid->gameUpdate(game.id, game.name, localGame.version, game.version,
                                       m_serverUrl + "/api/games/" + QString::number(game.id) + "/download");
            });
            btnLayout->addWidget(updateBtn);
        }

        auto *uninstallBtn = new QPushButton("UNINSTALL");
        uninstallBtn->setObjectName("uninstallBtn");
        uninstallBtn->setCursor(Qt::PointingHandCursor);
        uninstallBtn->setStyleSheet(
            "QPushButton#uninstallBtn { background-color: #ff4444; color: #ffffff; border: none; border-radius: 6px; font-size: 14px; padding: 14px 32px; font-weight: bold; }"
            "QPushButton#uninstallBtn:hover { background-color: #cc3333; }");
        connect(uninstallBtn, &QPushButton::clicked, [this, game, localGame]() {
            m_gameGrid->gameUninstall(game.id, game.name, localGame.installPath);
        });
        btnLayout->addWidget(uninstallBtn);
    } else {
        auto *installBtn = new QPushButton(QString("INSTALL - %1").arg(formatSize(game.fileSize)));
        installBtn->setObjectName("installBtn");
        installBtn->setCursor(Qt::PointingHandCursor);
        installBtn->setStyleSheet(
            "QPushButton#installBtn { background-color: #ffffff; color: #000000; border: none; border-radius: 6px; font-size: 14px; padding: 14px 48px; font-weight: bold; }"
            "QPushButton#installBtn:hover { background-color: #e0e0e0; }");
        connect(installBtn, &QPushButton::clicked, [this, game]() {
            m_gameGrid->gameDownload(game.id, game.name,
                                    m_serverUrl + "/api/games/" + QString::number(game.id) + "/download",
                                    game.fileSize);
        });
        btnLayout->addWidget(installBtn);
    }

    btnLayout->addStretch();
    layout->addWidget(btnWidget);

    auto *reviewsTitle = new QLabel("REVIEWS");
    reviewsTitle->setStyleSheet("color: #00ff88; font-size: 16px; font-weight: bold; letter-spacing: 2px; background: transparent; margin-top: 16px;");
    layout->addWidget(reviewsTitle);

    auto *reviewsContainer = new QWidget();
    reviewsContainer->setStyleSheet("background: transparent;");
    auto *reviewsLayout = new QVBoxLayout(reviewsContainer);
    reviewsLayout->setContentsMargins(0, 0, 0, 0);
    reviewsLayout->setSpacing(8);

    auto *loadingLabel = new QLabel("Loading reviews...");
    loadingLabel->setStyleSheet("color: #888888; font-size: 12px; background: transparent;");
    reviewsLayout->addWidget(loadingLabel);
    layout->addWidget(reviewsContainer);

    auto *addReviewWidget = new QWidget();
    addReviewWidget->setStyleSheet("background-color: #1a1a1a; border: 1px solid #2a2a2a; border-radius: 8px; padding: 16px;");
    auto *reviewFormLayout = new QVBoxLayout(addReviewWidget);
    reviewFormLayout->setSpacing(8);

    auto *reviewTitle = new QLabel("WRITE A REVIEW");
    reviewTitle->setStyleSheet("color: #00ff88; font-size: 13px; font-weight: bold; background: transparent; letter-spacing: 1px;");
    reviewFormLayout->addWidget(reviewTitle);

    auto *reviewerName = new QLineEdit();
    reviewerName->setPlaceholderText("Your name");
    reviewerName->setStyleSheet("background-color: #111111; color: #e0e0e0; border: 1px solid #2a2a2a; border-radius: 4px; padding: 8px 12px; font-size: 13px;");
    reviewFormLayout->addWidget(reviewerName);

    auto *starsWidget = new QWidget();
    starsWidget->setStyleSheet("background: transparent;");
    auto *starsLayout = new QHBoxLayout(starsWidget);
    starsLayout->setContentsMargins(0, 0, 0, 0);
    starsLayout->setSpacing(4);
    int selectedStars = 5;
    QList<QPushButton*> starBtns;
    for (int i = 1; i <= 5; i++) {
        auto *starBtn = new QPushButton(QString(QChar(0x2605)));
        starBtn->setStyleSheet("color: #ffaa00; font-size: 24px; background: transparent; border: none; padding: 2px;");
        starBtn->setCursor(Qt::PointingHandCursor);
        starBtns.append(starBtn);
        starsLayout->addWidget(starBtn);
    }
    starsLayout->addStretch();
    reviewFormLayout->addWidget(starsWidget);

    auto *reviewInput = new QLineEdit();
    reviewInput->setPlaceholderText("Review title (optional)");
    reviewInput->setStyleSheet("background-color: #111111; color: #e0e0e0; border: 1px solid #2a2a2a; border-radius: 4px; padding: 8px 12px; font-size: 13px;");
    reviewFormLayout->addWidget(reviewInput);

    auto *reviewTextInput = new QLineEdit();
    reviewTextInput->setPlaceholderText("Write your review...");
    reviewTextInput->setStyleSheet("background-color: #111111; color: #e0e0e0; border: 1px solid #2a2a2a; border-radius: 4px; padding: 8px 12px; font-size: 13px;");
    reviewFormLayout->addWidget(reviewTextInput);

    auto *submitReviewBtn = new QPushButton("SUBMIT REVIEW");
    submitReviewBtn->setCursor(Qt::PointingHandCursor);
    submitReviewBtn->setStyleSheet(
        "QPushButton { background-color: #00ff88; color: #000000; border: none; border-radius: 4px; padding: 10px; font-size: 12px; font-weight: bold; }"
        "QPushButton:hover { background-color: #00cc6a; }");
    reviewFormLayout->addWidget(submitReviewBtn);
    layout->addWidget(addReviewWidget);

    for (int i = 0; i < 5; i++) {
        connect(starBtns[i], &QPushButton::clicked, this, [starBtns, i, &selectedStars]() mutable {
            selectedStars = i + 1;
            for (int j = 0; j < 5; j++) {
                starBtns[j]->setText(j <= i ? QString(QChar(0x2605)) : QString(QChar(0x2606)));
            }
        });
    }

    connect(submitReviewBtn, &QPushButton::clicked, this, [this, game, reviewerName, reviewInput, reviewTextInput, &selectedStars, reviewsContainer, reviewsLayout, loadingLabel]() {
        if (reviewerName->text().trimmed().isEmpty()) {
            QMessageBox::warning(m_detailPage, "Error", "Enter your name");
            return;
        }
        QJsonObject obj;
        obj["username"] = reviewerName->text().trimmed();
        obj["stars"] = selectedStars;
        obj["title"] = reviewInput->text().trimmed();
        obj["text"] = reviewTextInput->text().trimmed();
        QNetworkRequest request(QUrl(m_serverUrl + "/api/games/" + QString::number(game.id) + "/reviews"));
        request.setHeader(QNetworkRequest::ContentTypeHeader, "application/json");
        request.setRawHeader("Authorization", "Bearer " + m_authToken.toUtf8());
        QNetworkReply *reply = m_reviewManager->post(request, QJsonDocument(obj).toJson());
        connect(reply, &QNetworkReply::finished, this, [this, reply, game, reviewsContainer, reviewsLayout, loadingLabel, reviewInput, reviewTextInput]() {
            reply->deleteLater();
            reviewInput->clear();
            reviewTextInput->clear();
            showNotification("Review", "Review submitted!");
        });
    });

    layout->addStretch();
    scroll->setWidget(content);
    outerLayout->addWidget(scroll);

    m_stack->addWidget(m_detailPage);
    m_stack->setCurrentWidget(m_detailPage);

    QNetworkRequest reviewReq(QUrl(m_serverUrl + "/api/games/" + QString::number(game.id) + "/reviews"));
    reviewReq.setRawHeader("Authorization", "Bearer " + m_authToken.toUtf8());
    reviewReq.setTransferTimeout(10000);
    QNetworkReply *reviewReply = m_reviewManager->get(reviewReq);
    connect(reviewReply, &QNetworkReply::finished, this, [this, reviewReply, game, reviewsContainer, reviewsLayout, loadingLabel]() {
        reviewReply->deleteLater();
        delete loadingLabel;
        QByteArray data = reviewReply->readAll();
        QJsonDocument doc = QJsonDocument::fromJson(data);
        QJsonObject obj = doc.object();
        QJsonArray reviews = obj["reviews"].toArray();
        for (const QJsonValue &val : reviews) {
            QJsonObject r = val.toObject();
            auto *reviewCard = new QWidget();
            reviewCard->setStyleSheet("background-color: #1a1a1a; border: 1px solid #2a2a2a; border-radius: 8px; padding: 12px;");
            auto *rcLayout = new QVBoxLayout(reviewCard);
            rcLayout->setSpacing(4);
            rcLayout->setContentsMargins(12, 8, 12, 8);

            QString stars;
            int s = r["stars"].toInt();
            for (int i = 0; i < s; i++) stars += QChar(0x2605);
            for (int i = s; i < 5; i++) stars += QChar(0x2606);

            auto *starsLbl = new QLabel(stars);
            starsLbl->setStyleSheet("color: #ffaa00; font-size: 14px; background: transparent;");
            rcLayout->addWidget(starsLbl);

            QString titleText = r["username"].toString();
            if (!r["title"].toString().isEmpty()) titleText += " - " + r["title"].toString();
            auto *authorLbl = new QLabel(titleText);
            authorLbl->setStyleSheet("color: #e0e0e0; font-size: 13px; font-weight: bold; background: transparent;");
            rcLayout->addWidget(authorLbl);

            if (!r["text"].toString().isEmpty()) {
                auto *textLbl = new QLabel(r["text"].toString());
                textLbl->setStyleSheet("color: #cccccc; font-size: 12px; background: transparent;");
                textLbl->setWordWrap(true);
                rcLayout->addWidget(textLbl);
            }

            reviewsLayout->addWidget(reviewCard);
        }
        if (reviews.isEmpty()) {
            auto *noReviews = new QLabel("No reviews yet. Be the first!");
            noReviews->setStyleSheet("color: #555555; font-size: 13px; background: transparent; padding: 16px;");
            reviewsLayout->addWidget(noReviews);
        }
    });
}

void LauncherWindow::setupLibraryTab() {
    if (m_libraryPage) { m_libraryPage->deleteLater(); m_libraryPage = nullptr; }

    m_libraryPage = new QWidget();
    m_libraryPage->setStyleSheet("background-color: #0a0a0a;");
    auto *outerLayout = new QVBoxLayout(m_libraryPage);
    outerLayout->setContentsMargins(0, 0, 0, 0);
    outerLayout->setSpacing(0);

    auto *topBar = new QWidget();
    topBar->setFixedHeight(60);
    topBar->setStyleSheet("background-color: #111111; border-bottom: 1px solid #2a2a2a;");
    auto *topBarLayout = new QHBoxLayout(topBar);
    topBarLayout->setContentsMargins(24, 0, 24, 0);
    auto *backBtn = new QPushButton("BACK");
    backBtn->setCursor(Qt::PointingHandCursor);
    connect(backBtn, &QPushButton::clicked, [this]() {
        m_stack->setCurrentIndex(1);
        m_gameGrid->refreshGrid();
    });
    topBarLayout->addWidget(backBtn);
    topBarLayout->addSpacing(16);
    auto *libTitle = new QLabel("MY LIBRARY");
    libTitle->setStyleSheet("font-size: 18px; font-weight: bold; color: #00ff88; background: transparent; letter-spacing: 3px;");
    topBarLayout->addWidget(libTitle);
    topBarLayout->addStretch();
    outerLayout->addWidget(topBar);

    auto *scroll = new QScrollArea();
    scroll->setWidgetResizable(true);
    scroll->setFrameShape(QFrame::NoFrame);
    scroll->setStyleSheet("background: transparent; border: none;");

    auto *content = new QWidget();
    content->setStyleSheet("background: transparent;");
    auto *layout = new QVBoxLayout(content);
    layout->setContentsMargins(24, 24, 24, 24);
    layout->setSpacing(12);

    QList<LocalGame> installedGames = m_localDB->getAllGames();
    if (installedGames.isEmpty()) {
        auto *empty = new QLabel("No games installed yet.\nDownload games from the store!");
        empty->setAlignment(Qt::AlignCenter);
        empty->setStyleSheet("color: #555555; font-size: 14px; padding: 80px;");
        layout->addWidget(empty);
    } else {
        for (const LocalGame &game : installedGames) {
            auto *card = new QWidget();
            card->setStyleSheet("background-color: #1a1a1a; border: 1px solid #2a2a2a; border-radius: 8px;");
            auto *cardLayout = new QHBoxLayout(card);
            cardLayout->setContentsMargins(16, 12, 16, 12);
            cardLayout->setSpacing(16);

            auto *coverLbl = new QLabel();
            coverLbl->setFixedSize(80, 80);
            coverLbl->setStyleSheet("background-color: #111111; border-radius: 8px;");
            coverLbl->setAlignment(Qt::AlignCenter);
            if (!game.coverUrl.isEmpty()) {
                QString url = game.coverUrl;
                if (url.startsWith("storage/covers/")) url = url.mid(QString("storage/").length());
                QNetworkRequest req(QUrl(m_serverUrl + "/" + url));
                req.setTransferTimeout(10000);
                QNetworkReply *reply = m_reviewManager->get(req);
                connect(reply, &QNetworkReply::finished, this, [coverLbl, reply]() {
                    reply->deleteLater();
                    if (reply->error() == QNetworkReply::NoError) {
                        QPixmap pixmap;
                        pixmap.loadFromData(reply->readAll());
                        if (!pixmap.isNull()) {
                            coverLbl->setPixmap(pixmap.scaled(80, 80, Qt::KeepAspectRatioByExpanding, Qt::SmoothTransformation));
                        }
                    }
                });
            } else {
                coverLbl->setText("GAME");
            }
            cardLayout->addWidget(coverLbl);

            auto *infoLayout = new QVBoxLayout();
            infoLayout->setSpacing(4);
            auto *nameLbl = new QLabel(game.name);
            nameLbl->setStyleSheet("color: #e0e0e0; font-size: 16px; font-weight: bold; background: transparent;");
            infoLayout->addWidget(nameLbl);

            auto *metaLbl = new QLabel("v" + game.version + " | " + game.category);
            metaLbl->setStyleSheet("color: #888888; font-size: 12px; background: transparent;");
            infoLayout->addWidget(metaLbl);

            if (!game.lastPlayed.isNull()) {
                auto *playedLbl = new QLabel("Last played: " + game.lastPlayed.toString("MMM d, h:mm AP"));
                playedLbl->setStyleSheet("color: #555555; font-size: 11px; background: transparent;");
                infoLayout->addWidget(playedLbl);
            }
            infoLayout->addStretch();
            cardLayout->addLayout(infoLayout);

            auto *playBtn = new QPushButton("PLAY");
            playBtn->setObjectName("playBtn");
            playBtn->setCursor(Qt::PointingHandCursor);
            playBtn->setStyleSheet(
                "QPushButton#playBtn { background-color: #00ff88; color: #000000; border: none; border-radius: 6px; font-size: 14px; padding: 12px 32px; font-weight: bold; }"
                "QPushButton#playBtn:hover { background-color: #00cc6a; }");
            connect(playBtn, &QPushButton::clicked, this, [this, game]() {
                launchGame(game.exePath, game.installPath);
                m_localDB->updateLastPlayed(game.serverGameId);
            });
            cardLayout->addWidget(playBtn);

            layout->addWidget(card);
        }
    }

    layout->addStretch();
    scroll->setWidget(content);
    outerLayout->addWidget(scroll);

    m_stack->addWidget(m_libraryPage);
}
