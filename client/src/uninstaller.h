#ifndef UNINSTALLER_H
#define UNINSTALLER_H

#include <QObject>
#include <QDir>
#include <QFileInfo>

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

        if (!removeDirectory(dir)) {
            emit uninstallError("Failed to remove: " + installPath);
            return false;
        }

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
