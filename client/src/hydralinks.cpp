#include "hydralinks.h"
#include <QJsonDocument>
#include <QJsonObject>
#include <QDebug>
#include <QFileInfo>
#include <QRegularExpression>
#include <algorithm>

const QStringList &HydraLinks::defaultSourceUrls() {
    static const QStringList urls = {
        "https://hydralinks.cloud/sources/fitgirl.json",
        "https://hydralinks.cloud/sources/dodi.json",
        "https://hydralinks.cloud/sources/steamrip.json",
        "https://hydralinks.cloud/sources/gog.json",
        "https://hydralinks.cloud/sources/onlinefix.json",
        "https://hydralinks.cloud/sources/kaoskrew.json",
        "https://hydralinks.cloud/sources/xatab.json",
        "https://hydralinks.cloud/sources/empress.json",
    };
    return urls;
}

HydraLinks::HydraLinks(QObject *parent) : QObject(parent) {
    m_manager = new QNetworkAccessManager(this);
    connect(m_manager, &QNetworkAccessManager::finished, this, &HydraLinks::onSourceReply);
}

void HydraLinks::fetchAllSources(const QString &serverUrl) {
    m_sources.clear();
    m_loaded = false;

    QNetworkRequest request(QUrl(serverUrl + "/api/hydra/sources"));
    request.setTransferTimeout(30000);
    request.setRawHeader("User-Agent", "MKLauncher/1.0");
    m_manager->get(request);
    m_pendingRequests = 1;
}

void HydraLinks::onSourceReply(QNetworkReply *reply) {
    QString url = reply->url().toString();

    if (reply->error() == QNetworkReply::NoError) {
        QByteArray data = reply->readAll();
        QJsonDocument doc = QJsonDocument::fromJson(data);
        if (doc.isArray()) {
            for (const QJsonValue &val : doc.array()) {
                QJsonObject obj = val.toObject();
                HydraSource source;
                source.name = obj["name"].toString();
                source.url = url;
                QJsonArray downloads = obj["downloads"].toArray();
                for (const QJsonValue &dv : downloads) {
                    QJsonObject dObj = dv.toObject();
                    HydraDownload dl;
                    dl.title = dObj["title"].toString();
                    dl.fileSize = dObj["fileSize"].toString();
                    dl.uploadDate = dObj["uploadDate"].toString();
                    dl.repackLinkSource = dObj["repackLinkSource"].toString();
                    QJsonArray uris = dObj["uris"].toArray();
                    for (const QJsonValue &u : uris) {
                        QString uri = u.toString().trimmed();
                        if (!uri.isEmpty()) dl.uris.append(uri);
                    }
                    if (!dl.title.isEmpty() && !dl.uris.isEmpty()) {
                        source.downloads.append(dl);
                    }
                }
                std::sort(source.downloads.begin(), source.downloads.end(),
                    [](const HydraDownload &a, const HydraDownload &b) {
                        return a.title.toLower() < b.title.toLower();
                    });
                if (!source.downloads.isEmpty()) {
                    qDebug() << "[HYDRA] Loaded source:" << source.name << source.downloads.size() << "games";
                    m_sources.append(source);
                }
            }
        } else if (doc.isObject()) {
            parseSource(data, url);
        }
    } else {
        qDebug() << "[HYDRA] Failed to load sources:" << url << reply->errorString();
    }

    reply->deleteLater();
    m_pendingRequests--;
    emit progress(1 - m_pendingRequests, 1);

    if (m_pendingRequests <= 0) {
        m_loaded = true;
        qDebug() << "[HYDRA] Loaded" << m_sources.size() << "sources";
        for (const HydraSource &src : m_sources) {
            qDebug() << "  " << src.name << src.downloads.size() << "games";
        }
        emit sourcesLoaded();
    }
}

void HydraLinks::parseSource(const QByteArray &data, const QString &url) {
    QJsonDocument doc = QJsonDocument::fromJson(data);
    if (doc.isNull()) {
        qDebug() << "[HYDRA] Invalid JSON from:" << url;
        return;
    }

    QJsonObject root = doc.object();
    HydraSource source;
    source.name = root["name"].toString();
    source.url = url;

    if (source.name.isEmpty()) {
        QFileInfo fi(url);
        source.name = fi.baseName();
    }

    QJsonArray downloads = root["downloads"].toArray();
    for (const QJsonValue &val : downloads) {
        QJsonObject obj = val.toObject();
        HydraDownload dl;
        dl.title = obj["title"].toString();
        dl.fileSize = obj["fileSize"].toString();
        dl.uploadDate = obj["uploadDate"].toString();
        dl.repackLinkSource = obj["repackLinkSource"].toString();

        QJsonArray uris = obj["uris"].toArray();
        for (const QJsonValue &u : uris) {
            QString uri = u.toString().trimmed();
            if (!uri.isEmpty()) {
                dl.uris.append(uri);
            }
        }

        if (!dl.title.isEmpty() && !dl.uris.isEmpty()) {
            source.downloads.append(dl);
        }
    }

    std::sort(source.downloads.begin(), source.downloads.end(),
        [](const HydraDownload &a, const HydraDownload &b) {
            return a.title.toLower() < b.title.toLower();
        });

    qDebug() << "[HYDRA] Parsed source:" << source.name << source.downloads.size() << "games";
    m_sources.append(source);
}

QList<HydraSource> HydraLinks::sourcesForGame(const QString &gameName) const {
    QList<HydraSource> result;
    for (const HydraSource &src : m_sources) {
        for (const HydraDownload &dl : src.downloads) {
            if (namesMatch(dl.title, gameName)) {
                result.append(src);
                break;
            }
        }
    }
    return result;
}

QStringList HydraLinks::allGameNames() const {
    QStringList names;
    for (const HydraSource &src : m_sources) {
        for (const HydraDownload &dl : src.downloads) {
            if (!names.contains(dl.title, Qt::CaseInsensitive)) {
                names.append(dl.title);
            }
        }
    }
    names.sort(Qt::CaseInsensitive);
    return names;
}

bool HydraLinks::namesMatch(const QString &a, const QString &b) {
    QString cleanA = a.toLower().trimmed();
    QString cleanB = b.toLower().trimmed();

    if (cleanA == cleanB) return true;

    cleanA.remove(QRegularExpression("[^a-z0-9 ]"));
    cleanB.remove(QRegularExpression("[^a-z0-9 ]"));
    cleanA = cleanA.simplified();
    cleanB = cleanB.simplified();

    if (cleanA == cleanB) return true;
    if (cleanA.contains(cleanB) || cleanB.contains(cleanA)) return true;

    QStringList wordsA = cleanA.split(' ', Qt::SkipEmptyParts);
    QStringList wordsB = cleanB.split(' ', Qt::SkipEmptyParts);
    int matches = 0;
    for (const QString &w : wordsA) {
        if (wordsB.contains(w) && w.length() > 2) matches++;
    }
    return matches >= qMax(1, qMin(wordsA.size(), wordsB.size()) - 1);
}

QString HydraLinks::formatFileSize(const QString &sizeStr) {
    return sizeStr.trimmed();
}
