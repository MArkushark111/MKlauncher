#ifndef GAMEGRID_H
#define GAMEGRID_H

#include <QWidget>
#include <QVBoxLayout>
#include <QHBoxLayout>
#include <QGridLayout>
#include <QLabel>
#include <QPushButton>
#include <QScrollArea>
#include <QMenu>
#include <QAction>
#include <QFrame>
#include <QNetworkAccessManager>
#include <QNetworkReply>
#include <QPixmap>
#include <QJsonDocument>
#include <QJsonObject>
#include <QJsonArray>

struct ServerGame {
    int id = 0;
    QString name;
    QString description;
    QString version;
    QString category;
    QString tags;
    QString coverUrl;
    QString backgroundUrl;
    QString logoUrl;
    QString wideCoverUrl;
    QString archivePath;
    QString gameFolder;
    QString exePath;
    qint64 fileSize = 0;
    int downloadCount = 0;
};

class GameGrid : public QWidget {
    Q_OBJECT
public:
    explicit GameGrid(QWidget *parent = nullptr);

    void loadGames(const QString &serverUrl, const QString &token);
    void refreshGrid();
    QList<ServerGame> games() const { return m_games; }
    void filterByText(const QString &text);
    void filterByStatus(const QString &status);

signals:
    void gamePlay(int gameId, const QString &name, const QString &exePath, const QString &installPath);
    void gameDownload(int gameId, const QString &name, const QString &url, qint64 size);
    void gameUpdate(int gameId, const QString &name, const QString &localVersion, const QString &serverVersion, const QString &url);
    void gameUninstall(int gameId, const QString &name, const QString &installPath);
    void gamesLoadError(const QString &error);

private slots:
    void onGamesLoaded(QNetworkReply *reply);
    void onCoverLoaded(QNetworkReply *reply);

private:
    QGridLayout *m_grid;
    QScrollArea *m_scrollArea;
    QWidget *m_gridWidget;
    QNetworkAccessManager *m_apiManager;
    QNetworkAccessManager *m_coverManager;

private:
    QList<ServerGame> m_games;
    QMap<int, QPixmap> m_covers;
    QString m_serverUrl;
    QString m_token;
    QMap<QNetworkReply*, int> m_coverReplies;
    QString m_searchFilter;
    QString m_statusFilter;

    void buildGrid();
    QWidget* createGameCard(const ServerGame &game);
    QString formatSize(qint64 bytes) const;
};

#endif
