#include "downloadmanager.h"
#include <QDir>
#include <QFileInfo>

DownloadManager::DownloadManager(QObject *parent) : QObject(parent) {
    m_manager = new QNetworkAccessManager(this);
    m_speedTimer = new QTimer(this);
    connect(m_speedTimer, &QTimer::timeout, this, &DownloadManager::updateSpeed);
    m_speedTimer->start(500);
}

void DownloadManager::startDownload(int gameId, const QString &gameName, const QUrl &url, const QString &savePath) {
    if (m_downloads.contains(gameId) && m_downloads[gameId].info.active) return;

    QFileInfo fi(savePath);
    QDir().mkpath(fi.absolutePath());

    DownloadTask task;
    task.info.gameId = gameId;
    task.info.gameName = gameName;
    task.info.url = url.toString();
    task.info.savePath = savePath;
    task.info.active = true;
    task.info.paused = false;
    task.info.downloadedBytes = 0;
    task.elapsed.start();

    task.file = new QFile(savePath, this);
    if (!task.file->open(QIODevice::WriteOnly)) {
        task.info.error = true;
        task.info.errorMsg = "Cannot create file: " + savePath;
        m_downloads[gameId] = task;
        emit downloadError(gameId, task.info.errorMsg);
        return;
    }

    QNetworkRequest request(url);
    request.setTransferTimeout(30000);

    task.reply = m_manager->get(request);
    connect(task.reply, &QNetworkReply::readyRead, this, &DownloadManager::onReadyRead);
    connect(task.reply, &QNetworkReply::downloadProgress, this, &DownloadManager::onDownloadProgress);
    connect(task.reply, &QNetworkReply::finished, this, &DownloadManager::onFinished);
    connect(task.reply, &QNetworkReply::errorOccurred, this, &DownloadManager::onError);

    m_downloads[gameId] = task;
    emit downloadStarted(gameId);
}

void DownloadManager::pauseDownload(int gameId) {
    if (m_downloads.contains(gameId)) {
        auto &task = m_downloads[gameId];
        if (task.reply) {
            task.reply->abort();
        }
        task.info.paused = true;
        task.info.active = false;
    }
}

void DownloadManager::resumeDownload(int gameId) {
    if (m_downloads.contains(gameId) && m_downloads[gameId].info.paused) {
        auto &task = m_downloads[gameId];
        startDownload(gameId, task.info.gameName, QUrl(task.info.url), task.info.savePath);
    }
}

void DownloadManager::cancelDownload(int gameId) {
    if (m_downloads.contains(gameId)) {
        auto &task = m_downloads[gameId];
        if (task.reply) {
            task.reply->abort();
            task.reply->deleteLater();
        }
        if (task.file) {
            task.file->close();
            task.file->remove();
            task.file->deleteLater();
        }
        m_downloads.remove(gameId);
    }
}

void DownloadManager::onReadyRead() {
    QNetworkReply *reply = qobject_cast<QNetworkReply*>(sender());
    if (!reply) return;

    for (auto it = m_downloads.begin(); it != m_downloads.end(); ++it) {
        if (it.value().reply == reply && it.value().file) {
            it.value().file->write(reply->readAll());
            break;
        }
    }
}

void DownloadManager::onDownloadProgress(qint64 received, qint64 total) {
    QNetworkReply *reply = qobject_cast<QNetworkReply*>(sender());
    if (!reply) return;

    for (auto it = m_downloads.begin(); it != m_downloads.end(); ++it) {
        if (it.value().reply == reply) {
            it.value().info.downloadedBytes = received;
            it.value().info.totalBytes = total;

            if (total > 0) {
                it.value().info.progress = static_cast<int>((received * 100) / total);
            }

            qint64 elapsed = it.value().elapsed.elapsed();
            if (elapsed > 0) {
                it.value().currentSpeed = (static_cast<double>(received) / elapsed) * 1000.0;
                it.value().info.speed = it.value().currentSpeed;

                if (it.value().currentSpeed > 0 && total > received) {
                    qint64 remaining = total - received;
                    it.value().info.eta = formatEta(remaining, it.value().currentSpeed);
                }
            }

            emit downloadProgress(it.key(), it.value().info.progress,
                                   it.value().currentSpeed, it.value().info.eta);
            break;
        }
    }
}

void DownloadManager::onFinished() {
    QNetworkReply *reply = qobject_cast<QNetworkReply*>(sender());
    if (!reply) return;

    for (auto it = m_downloads.begin(); it != m_downloads.end(); ++it) {
        if (it.value().reply == reply) {
            qDebug() << "[DOWNLOAD] Finished. Error:" << reply->errorString() << "HTTP:" << reply->attribute(QNetworkRequest::HttpStatusCodeAttribute);
            if (reply->error() == QNetworkReply::NoError) {
                it.value().file->flush();
                it.value().file->close();
                it.value().info.active = false;
                qDebug() << "[DOWNLOAD] Saved to:" << it.value().info.savePath;
                emit downloadComplete(it.key(), it.value().info.savePath);
            } else if (reply->error() != QNetworkReply::OperationCanceledError) {
                qDebug() << "[DOWNLOAD] ERROR:" << reply->errorString();
                it.value().info.error = true;
                it.value().info.errorMsg = reply->errorString();
                emit downloadError(it.key(), it.value().info.errorMsg);
            }
            reply->deleteLater();
            if (it.value().file) it.value().file->deleteLater();
            m_downloads.remove(it.key());
            break;
        }
    }
}

void DownloadManager::onError(QNetworkReply::NetworkError error) {
    if (error == QNetworkReply::OperationCanceledError) return;

    QNetworkReply *reply = qobject_cast<QNetworkReply*>(sender());
    if (!reply) return;

    for (auto it = m_downloads.begin(); it != m_downloads.end(); ++it) {
        if (it.value().reply == reply) {
            it.value().info.error = true;
            it.value().info.errorMsg = reply->errorString();
            emit downloadError(it.key(), it.value().info.errorMsg);
            break;
        }
    }
}

void DownloadManager::updateSpeed() {
    for (auto it = m_downloads.begin(); it != m_downloads.end(); ++it) {
        if (it.value().info.active && !it.value().info.paused) {
            qint64 currentBytes = it.value().info.downloadedBytes;
            qint64 diff = currentBytes - it.value().lastBytes;
            it.value().lastBytes = currentBytes;
            it.value().currentSpeed = static_cast<double>(diff) * 2.0;
            it.value().info.speed = it.value().currentSpeed;
        }
    }
}

QString DownloadManager::formatSpeed(double bytesPerSec) const {
    if (bytesPerSec < 1024) return QString::number(bytesPerSec, 'f', 0) + " B/s";
    if (bytesPerSec < 1048576) return QString::number(bytesPerSec / 1024, 'f', 1) + " KB/s";
    return QString::number(bytesPerSec / 1048576, 'f', 2) + " MB/s";
}

QString DownloadManager::formatEta(qint64 remaining, double speed) const {
    if (speed <= 0) return "calculating...";
    int seconds = static_cast<int>(remaining / speed);
    if (seconds < 60) return QString::number(seconds) + "s";
    if (seconds < 3600) return QString::number(seconds / 60) + "m " + QString::number(seconds % 60) + "s";
    return QString::number(seconds / 3600) + "h " + QString::number((seconds % 3600) / 60) + "m";
}

QString DownloadManager::formatSize(qint64 bytes) const {
    if (bytes < 1024) return QString::number(bytes) + " B";
    if (bytes < 1048576) return QString::number(bytes / 1024.0, 'f', 1) + " KB";
    if (bytes < 1073741824) return QString::number(bytes / 1048576.0, 'f', 1) + " MB";
    return QString::number(bytes / 1073741824.0, 'f', 2) + " GB";
}
