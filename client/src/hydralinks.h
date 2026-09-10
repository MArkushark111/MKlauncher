#ifndef HYDRALINKS_H
#define HYDRALINKS_H

#include <QObject>
#include <QNetworkAccessManager>
#include <QNetworkReply>
#include <QString>
#include <QStringList>
#include <QJsonArray>

struct HydraDownload {
    QString title;
    QString fileSize;
    QStringList uris;
    QString uploadDate;
    QString repackLinkSource;
};

struct HydraSource {
    QString name;
    QString url;
    QList<HydraDownload> downloads;
};

class HydraLinks : public QObject {
    Q_OBJECT
public:
    explicit HydraLinks(QObject *parent = nullptr);

    void fetchAllSources();
    bool isLoaded() const { return m_loaded; }
    int sourceCount() const { return m_sources.size(); }
    QList<HydraSource> sources() const { return m_sources; }

    QList<HydraSource> sourcesForGame(const QString &gameName) const;
    QStringList allGameNames() const;
    static bool namesMatch(const QString &a, const QString &b);

    static const QStringList &defaultSourceUrls();
    static QString formatFileSize(const QString &sizeStr);

signals:
    void sourcesLoaded();
    void sourcesError(const QString &error);
    void progress(int loaded, int total);

private slots:
    void onSourceReply(QNetworkReply *reply);

private:
    QNetworkAccessManager *m_manager;
    QList<HydraSource> m_sources;
    int m_pendingRequests = 0;
    bool m_loaded = false;

    void parseSource(const QByteArray &data, const QString &url);

};

#endif
