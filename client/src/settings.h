#ifndef SETTINGS_H
#define SETTINGS_H

#include <QString>
#include <QSettings>
#include <QStandardPaths>

class Settings {
public:
    Settings() : m_settings("MKGames", "MKLauncher") {}

    QString serverUrl() const { return m_settings.value("server/url", "").toString(); }
    void setServerUrl(const QString &url) { m_settings.setValue("server/url", url); }

    QString authToken() const { return m_settings.value("auth/token", "").toString(); }
    void setAuthToken(const QString &token) { m_settings.setValue("auth/token", token); }

    bool isConnected() const { return m_settings.value("server/connected", false).toBool(); }
    void setConnected(bool connected) { m_settings.setValue("server/connected", connected); }

    QString installDir() const {
        return m_settings.value("paths/install",
            QStandardPaths::writableLocation(QStandardPaths::GenericDataLocation) + "/MKLauncher/games"
        ).toString();
    }
    void setInstallDir(const QString &dir) { m_settings.setValue("paths/install", dir); }

    bool autoCheckUpdates() const { return m_settings.value("launcher/auto_updates", true).toBool(); }
    void setAutoCheckUpdates(bool val) { m_settings.setValue("launcher/auto_updates", val); }

    bool firstRun() const { return m_settings.value("general/first_run", true).toBool(); }
    void setFirstRun(bool val) { m_settings.setValue("general/first_run", val); }

    bool showNotifications() const { return m_settings.value("notifications/show", true).toBool(); }
    void setShowNotifications(bool val) { m_settings.setValue("notifications/show", val); }

    bool closeToTray() const { return m_settings.value("general/close_to_tray", false).toBool(); }
    void setCloseToTray(bool val) { m_settings.setValue("general/close_to_tray", val); }

    bool autoExtract() const { return m_settings.value("downloads/auto_extract", true).toBool(); }
    void setAutoExtract(bool val) { m_settings.setValue("downloads/auto_extract", val); }

    bool deleteArchiveAfterInstall() const { return m_settings.value("downloads/delete_archive", true).toBool(); }
    void setDeleteArchiveAfterInstall(bool val) { m_settings.setValue("downloads/delete_archive", val); }

    int maxConcurrentDownloads() const { return m_settings.value("downloads/max_concurrent", 3).toInt(); }
    void setMaxConcurrentDownloads(int val) { m_settings.setValue("downloads/max_concurrent", val); }

    bool keepLibraryUpdated() const { return m_settings.value("launcher/keep_updated", true).toBool(); }
    void setKeepLibraryUpdated(bool val) { m_settings.setValue("launcher/keep_updated", val); }

    bool launchAfterInstall() const { return m_settings.value("launcher/launch_after_install", false).toBool(); }
    void setLaunchAfterInstall(bool val) { m_settings.setValue("launcher/launch_after_install", val); }

    bool showGameSizeInGrid() const { return m_settings.value("appearance/show_size", true).toBool(); }
    void setShowGameSizeInGrid(bool val) { m_settings.setValue("appearance/show_size", val); }

    bool darkMode() const { return m_settings.value("appearance/dark_mode", true).toBool(); }
    void setDarkMode(bool val) { m_settings.setValue("appearance/dark_mode", val); }

    void clearAll() {
        m_settings.clear();
    }

    QString userToken() const { return m_settings.value("user/token", "").toString(); }
    void setUserToken(const QString &token) { m_settings.setValue("user/token", token); }

    QString username() const { return m_settings.value("user/username", "").toString(); }
    void setUsername(const QString &name) { m_settings.setValue("user/username", name); }

private:
    QSettings m_settings;
};

#endif
