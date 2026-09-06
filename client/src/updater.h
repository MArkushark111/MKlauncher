#ifndef UPDATER_H
#define UPDATER_H

#include <QObject>
#include <QNetworkAccessManager>
#include <QNetworkReply>
#include <QJsonDocument>
#include <QJsonObject>

class Updater : public QObject {
    Q_OBJECT
public:
    explicit Updater(QObject *parent = nullptr);

    void checkForUpdates(const QString &currentVersion);

signals:
    void updateAvailable(const QString &newVersion, const QString &downloadUrl);
    void noUpdatesAvailable();
    void updateCheckError(const QString &error);

private slots:
    void onReplyFinished(QNetworkReply *reply);

private:
    QNetworkAccessManager *m_manager;
    QString m_updateUrl;
};

#endif
