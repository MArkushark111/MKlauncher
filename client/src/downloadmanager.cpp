#include "downloadmanager.h"
#include <QDir>
#include <QFileInfo>
#include <QJsonDocument>
#include <QJsonObject>
#include <QJsonArray>

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
    task.redirectPhase = true;
    task.elapsed.start();

    QNetworkRequest request(url);
    request.setTransferTimeout(30000);
    request.setRawHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36");

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
        if (it.value().reply == reply) {
            if (it.value().redirectPhase) {
                it.value().redirectBuffer.append(reply->readAll());
            } else if (it.value().file) {
                it.value().file->write(reply->readAll());
            }
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

            if (reply->error() != QNetworkReply::NoError && reply->error() != QNetworkReply::OperationCanceledError) {
                qDebug() << "[DOWNLOAD] ERROR:" << reply->errorString();
                it.value().info.error = true;
                it.value().info.errorMsg = reply->errorString();
                emit downloadError(it.key(), it.value().info.errorMsg);
                reply->deleteLater();
                if (it.value().file) it.value().file->deleteLater();
                m_downloads.remove(it.key());
                break;
            }

            if (it.value().redirectPhase) {
                QByteArray data = it.value().redirectBuffer;
                it.value().redirectBuffer.clear();
                reply->deleteLater();
                it.value().reply = nullptr;

                QString contentStr = QString::fromUtf8(data);
                qDebug() << "[DOWNLOAD] Redirect phase. Response size:" << data.size() << "starts with:" << contentStr.left(100);

                if (data.size() > 0 && (data[0] == '{' || data[0] == '[')) {
                    QJsonDocument doc = QJsonDocument::fromJson(data);
                    if (!doc.isNull()) {
                        QJsonObject obj = doc.object();
                        qDebug() << "[DOWNLOAD] JSON response keys:" << obj.keys();

                        QString realUrl;
                        if (obj.contains("data")) {
                            QJsonValue dataVal = obj["data"];
                            if (dataVal.isObject()) {
                                QJsonObject dataObj = dataVal.toObject();
                                if (dataObj.contains("url")) realUrl = dataObj["url"].toString();
                                else if (dataObj.contains("downloadUrl")) realUrl = dataObj["downloadUrl"].toString();
                                else if (dataObj.contains("directUrl")) realUrl = dataObj["directUrl"].toString();
                                else if (dataObj.contains("downloadPage")) {
                                    qDebug() << "[DOWNLOAD] Gofile-style response. downloadPage:" << dataObj["downloadPage"].toString();
                                }
                            }
                        }
                        if (realUrl.isEmpty() && obj.contains("url")) realUrl = obj["url"].toString();
                        if (realUrl.isEmpty() && obj.contains("download_url")) realUrl = obj["download_url"].toString();

                        if (!realUrl.isEmpty()) {
                            qDebug() << "[DOWNLOAD] Follow redirect to:" << realUrl;
                            it.value().info.url = realUrl;
                            it.value().redirectPhase = true;

                            QNetworkRequest newRequest{QUrl(realUrl)};
                            newRequest.setTransferTimeout(120000);
                            newRequest.setRawHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36");
                            it.value().reply = m_manager->get(newRequest);
                            connect(it.value().reply, &QNetworkReply::readyRead, this, &DownloadManager::onReadyRead);
                            connect(it.value().reply, &QNetworkReply::downloadProgress, this, &DownloadManager::onDownloadProgress);
                            connect(it.value().reply, &QNetworkReply::finished, this, &DownloadManager::onFinished);
                            connect(it.value().reply, &QNetworkReply::errorOccurred, this, &DownloadManager::onError);
                            break;
                        }
                    }
                }

                qDebug() << "[DOWNLOAD] Not JSON redirect, treating as direct file response. Size:" << data.size();
                it.value().redirectPhase = false;

                QFileInfo fi(it.value().info.savePath);
                QDir().mkpath(fi.absolutePath());
                it.value().file = new QFile(it.value().info.savePath, this);
                if (!it.value().file->open(QIODevice::WriteOnly)) {
                    it.value().info.error = true;
                    it.value().info.errorMsg = "Cannot create file: " + it.value().info.savePath;
                    emit downloadError(it.key(), it.value().info.errorMsg);
                    m_downloads.remove(it.key());
                    break;
                }
                it.value().file->write(data);

                QNetworkRequest newRequest{QUrl(it.value().info.url)};
                newRequest.setTransferTimeout(120000);
                newRequest.setRawHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36");
                it.value().reply = m_manager->get(newRequest);
                connect(it.value().reply, &QNetworkReply::readyRead, this, &DownloadManager::onReadyRead);
                connect(it.value().reply, &QNetworkReply::downloadProgress, this, &DownloadManager::onDownloadProgress);
                connect(it.value().reply, &QNetworkReply::finished, this, &DownloadManager::onFinished);
                connect(it.value().reply, &QNetworkReply::errorOccurred, this, &DownloadManager::onError);
                break;
            }

            if (it.value().file) {
                it.value().file->flush();
                it.value().file->close();
            }
            it.value().info.active = false;
            qDebug() << "[DOWNLOAD] Saved to:" << it.value().info.savePath;
            emit downloadComplete(it.key(), it.value().info.savePath);

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
