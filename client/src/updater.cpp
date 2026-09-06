#include "updater.h"
#include <QJsonDocument>
#include <QJsonObject>
#include <QDebug>

Updater::Updater(QObject *parent) : QObject(parent) {
    m_manager = new QNetworkAccessManager(this);
    connect(m_manager, &QNetworkAccessManager::finished, this, &Updater::onReplyFinished);
}

void Updater::checkForUpdates(const QString &currentVersion) {
    if (m_updateUrl.isEmpty()) {
        emit noUpdatesAvailable();
        return;
    }

    QUrl url(m_updateUrl);
    QNetworkRequest request(url);
    request.setTransferTimeout(10000);
    m_manager->get(request);
}

void Updater::onReplyFinished(QNetworkReply *reply) {
    if (reply->error() != QNetworkReply::NoError) {
        emit updateCheckError(reply->errorString());
        reply->deleteLater();
        return;
    }

    QByteArray data = reply->readAll();
    QJsonDocument doc = QJsonDocument::fromJson(data);

    if (doc.isObject()) {
        QJsonObject obj = doc.object();
        QString latestVersion = obj.value("version").toString();
        QString downloadUrl = obj.value("download_url").toString();

        if (!latestVersion.isEmpty()) {
            emit updateAvailable(latestVersion, downloadUrl);
        } else {
            emit noUpdatesAvailable();
        }
    } else {
        emit noUpdatesAvailable();
    }

    reply->deleteLater();
}
