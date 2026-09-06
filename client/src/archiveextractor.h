#ifndef ARCHIVEEXTRACTOR_H
#define ARCHIVEEXTRACTOR_H

#include <QObject>
#include <QProcess>
#include <QDir>
#include <QTemporaryDir>

class ArchiveExtractor : public QObject {
    Q_OBJECT
public:
    explicit ArchiveExtractor(QObject *parent = nullptr);

    enum ArchiveType { ZIP, RAR, SEVENZ, TARGZ, UNKNOWN };

    bool extract(const QString &archivePath, const QString &destDir, const QString &subfolder = "");
    bool cancelExtraction();
    bool isExtracting() const { return m_extracting; }

    static ArchiveType detectType(const QString &path);
    static bool isSupported(const QString &path);
    static QStringList supportedExtensions();

signals:
    void extractionProgress(int percent, const QString &currentFile);
    void extractionComplete(const QString &destDir);
    void extractionError(const QString &error);

private slots:
    void onProcessReadyRead();
    void onProcessFinished(int exitCode, QProcess::ExitStatus exitStatus);

private:
    QProcess *m_process = nullptr;
    bool m_extracting = false;
    QString m_destDir;
    int m_totalFiles = 0;
    int m_extractedFiles = 0;

    bool extractZip(const QString &archive, const QString &dest);
    bool extractRar(const QString &archive, const QString &dest);
    bool extract7z(const QString &archive, const QString &dest);
    bool extractTarGz(const QString &archive, const QString &dest);
};

#endif
