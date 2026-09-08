#ifndef LAUNCHERWINDOW_H
#define LAUNCHERWINDOW_H

#include <QMainWindow>
#include <QStackedWidget>
#include <QLineEdit>
#include <QPushButton>
#include <QLabel>
#include <QProgressBar>
#include <QComboBox>
#include <QNetworkAccessManager>
#include <QNetworkReply>
#include <QSystemTrayIcon>
#include <QTimer>
#include <QTextEdit>
#include <QProcess>
#include <QInputDialog>

#include "gamegrid.h"
#include "downloadmanager.h"
#include "archiveextractor.h"
#include "uninstaller.h"
#include "localdb.h"
#include "updater.h"
#include "settings.h"

class LauncherWindow : public QMainWindow {
    Q_OBJECT
public:
    explicit LauncherWindow(QWidget *parent = nullptr);

private slots:
    void onConnectClicked();
    void onAuthResult(QNetworkReply *reply);
    void onLoginClicked();
    void onRegisterClicked();
    void onGamePlay(int gameId, const QString &name, const QString &exePath, const QString &installPath);
    void onGameDownload(int gameId, const QString &name, const QString &url, qint64 size);
    void onGameUpdate(int gameId, const QString &name, const QString &localVer, const QString &serverVer, const QString &url);
    void onGameUninstall(int gameId, const QString &name, const QString &installPath);
    void onDownloadProgress(int gameId, int percent, double speed, const QString &eta);
    void onDownloadComplete(int gameId, const QString &filePath);
    void onDownloadError(int gameId, const QString &error);
    void onExtractionProgress(int percent, const QString &currentFile);
    void onExtractionComplete(const QString &destDir);
    void onExtractionError(const QString &error);
    void onUninstallComplete(const QString &gameName);
    void checkLauncherUpdates();
    void onSearchChanged(const QString &text);
    void onFilterChanged(int index);
    void onGameDetails(const ServerGame &game);
    void checkForUpdates(bool manual = false);

private:
    QStackedWidget *m_stack;
    QWidget *m_connectPage;
    QWidget *m_mainPage;
    QWidget *m_settingsPage;
    QLineEdit *m_urlInput;
    QLineEdit *m_usernameInput;
    QLineEdit *m_passwordInput;
    QPushButton *m_connectBtn;
    QPushButton *m_loginBtn;
    QPushButton *m_registerBtn;
    QLabel *m_statusLabel;
    QLineEdit *m_searchInput;
    QComboBox *m_filterCombo;
    GameGrid *m_gameGrid;
    QProgressBar *m_progressBar;
    QLabel *m_progressLabel;
    QSystemTrayIcon *m_trayIcon;
    QTimer *m_updateTimer;

    DownloadManager *m_downloadManager;
    ArchiveExtractor *m_extractor;
    Uninstaller *m_uninstaller;
    LocalDB *m_localDB;
    Updater *m_updater;
    Settings m_settings;

    QNetworkAccessManager *m_authManager;
    QNetworkAccessManager *m_reviewManager;
    QString m_serverUrl;
    QString m_authToken;
    QString m_userToken;
    QString m_username;

    int m_currentDownloadGameId = 0;
    QString m_currentDownloadName;
    QString m_pendingArchivePath;
    QWidget *m_detailPage = nullptr;
    QWidget *m_libraryPage = nullptr;

    // Feature tracking
    QTimer *m_playtimeTimer = nullptr;
    int m_currentGameId = 0;

    // Running game tracking
    QProcess *m_runningProcess = nullptr;
    int m_runningGameId = 0;
    QString m_runningGameName;
    QPushButton *m_detailPlayBtn = nullptr;

    // Chat
    QWidget *m_chatPage = nullptr;
    QTextEdit *m_chatMessages = nullptr;
    QLineEdit *m_chatInput = nullptr;
    QComboBox *m_chatChannel = nullptr;
    QTimer *m_chatPollTimer = nullptr;
    int m_lastChatId = 0;

    // News
    QWidget *m_newsPage = nullptr;

    // Categories filter
    QComboBox *m_categoryFilter = nullptr;

    void setupConnectPage();
    void setupMainPage();
    void setupSettingsPage();
    void connectToServer(const QString &url, const QString &passcode);
    void refreshGames();
    void resetConnection();
    void launchGame(const QString &exePath, const QString &installPath);
    void showNotification(const QString &title, const QString &msg);
    void addDefenderExclusion(const QString &path);
    QString gameInstallPath(const QString &name);
    void showGameDetail(const ServerGame &game);
    void setupLibraryTab();
    void checkWishlistNotifications();
    void setupNewsTab();
    void setupChatTab();
    void loadNews();
    void loadCategories();
    void filterByCategory(int categoryId);
    void loadFeatured();
    void loadRecentlyPlayed();
    void startPlaytimeTracking(int gameId);
    void stopPlaytimeTracking();
    void pollChat();
    void sendChatMessage();
    void loadChatMessages();
    void toggleWishlist(int gameId);
    void loadWishlist();
    void loadLeaderboard(int gameId);
    void loadAchievements(int gameId);
    void loadScreenshots(int gameId);
    void stopRunningGame();
    void onGameProcessFinished(int exitCode, QProcess::ExitStatus exitStatus);
    void submitReport(int gameId, const QString &gameName, const QString &description);
};

#endif
