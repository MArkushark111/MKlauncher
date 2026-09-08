#ifndef DOWNLOADMANAGER_H
#define DOWNLOADMANAGER_H

#include <QObject>
#include <QNetworkAccessManager>
#include <QNetworkReply>
#include <QNetworkRequest>
#include <QFile>
#include <QUrl>
#include <QElapsedTimer>
#include <QTimer>

struct DownloadInfo {
    int gameId;
    QString gameName;
    QString url;
    QString savePath;
    qint64 totalBytes = 0;
    qint64 downloadedBytes = 0;
    double speed = 0;
    int progress = 0;
    bool active = false;
    bool paused = false;
    bool error = false;
    QString errorMsg;
    QString eta;
};

class DownloadManager : public QObject {
    Q_OBJECT
public:
    explicit DownloadManager(QObject *parent = nullptr);

    void startDownload(int gameId, const QString &gameName, const QUrl &url, const QString &savePath);
    void pauseDownload(int gameId);
    void resumeDownload(int gameId);
    void cancelDownload(int gameId);

    QList<DownloadInfo> activeDownloads() const {
        QList<DownloadInfo> list;
        for (auto it = m_downloads.constBegin(); it != m_downloads.constEnd(); ++it)
            list.append(it.value().info);
        return list;
    }
    bool isDownloading(int gameId) const { return m_downloads.contains(gameId) && m_downloads[gameId].info.active; }

signals:
    void downloadProgress(int gameId, int percent, double speed, const QString &eta);
    void downloadComplete(int gameId, const QString &filePath);
    void downloadError(int gameId, const QString &error);
    void downloadStarted(int gameId);

private slots:
    void onReadyRead();
    void onDownloadProgress(qint64 received, qint64 total);
    void onFinished();
    void onError(QNetworkReply::NetworkError error);
    void updateSpeed();

private:
    struct DownloadTask {
        QNetworkReply *reply = nullptr;
        QFile *file = nullptr;
        DownloadInfo info;
        QElapsedTimer elapsed;
        qint64 lastBytes = 0;
        double currentSpeed = 0;
        bool redirectPhase = true;
        QByteArray redirectBuffer;
    };

    QNetworkAccessManager *m_manager;
    QTimer *m_speedTimer;
    QMap<int, DownloadTask> m_downloads;

    QString formatSpeed(double bytesPerSec) const;
    QString formatEta(qint64 remaining, double speed) const;
    QString formatSize(qint64 bytes) const;
};

#endif
