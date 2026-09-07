#ifndef UNINSTALLER_H
#define UNINSTALLER_H

#include <QObject>
#include <QDir>
#include <QFileInfo>
#include <QProcess>
#include <QCoreApplication>

class Uninstaller : public QObject {
    Q_OBJECT
public:
    explicit Uninstaller(QObject *parent = nullptr) : QObject(parent) {}

    bool uninstallGame(const QString &installPath, const QString &gameName) {
        if (installPath.isEmpty()) {
            emit uninstallError("No install path specified");
            return false;
        }

        QDir dir(installPath);
        if (!dir.exists()) {
            emit uninstallComplete(gameName);
            return true;
        }

        emit uninstallProgress(0, "Removing files...");

#ifdef Q_OS_WIN
        QProcess proc;
        QString systemRoot = qEnvironmentVariable("SystemRoot", "C:/Windows");
        proc.start(systemRoot + "/System32/cmd.exe", {"/c", "rmdir", "/s", "/q", QDir::toNativeSeparators(installPath)});
        proc.waitForFinished(30000);
        if (dir.exists()) {
            emit uninstallError("Failed to remove: " + installPath + " (" + proc.readAllStandardError().trimmed() + ")");
            return false;
        }
#else
        if (!removeDirectory(dir)) {
            emit uninstallError("Failed to remove: " + installPath);
            return false;
        }
#endif

        emit uninstallProgress(100, "Done");
        emit uninstallComplete(gameName);
        return true;
    }

signals:
    void uninstallProgress(int percent, const QString &currentFile);
    void uninstallComplete(const QString &gameName);
    void uninstallError(const QString &error);

private:
    bool removeDirectory(const QDir &dir) {
        QStringList entries = dir.entryList(QDir::Files | QDir::Dirs | QDir::NoDotAndDotDot);
        int count = entries.size();
        int processed = 0;

        for (const QString &entry : entries) {
            QString fullPath = dir.absoluteFilePath(entry);
            QFileInfo fi(fullPath);

            if (fi.isDir()) {
                if (!removeDirectory(QDir(fullPath))) {
                    return false;
                }
            } else {
                QFile::remove(fullPath);
            }

            processed++;
            int pct = count > 0 ? (processed * 100 / count) : 100;
            emit uninstallProgress(pct, entry);
        }

        dir.rmdir(dir.absolutePath());
        return true;
    }
};

#endif
