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

private:
    QSettings m_settings;
};

#endif
