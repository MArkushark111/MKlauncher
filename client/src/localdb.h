#ifndef LOCALDB_H
#define LOCALDB_H

#include <QObject>
#include <QSqlDatabase>
#include <QSqlQuery>
#include <QSqlError>
#include <QVariant>
#include <QDebug>
#include <QDir>

struct LocalGame {
    int id = 0;
    int serverGameId = 0;
    QString name;
    QString installPath;
    QString version;
    QString exePath;
    QString coverUrl;
    QString category;
    QString tags;
    qint64 fileSize = 0;
    QDateTime installedAt;
    QDateTime lastPlayed;
};

class LocalDB : public QObject {
    Q_OBJECT
public:
    explicit LocalDB(QObject *parent = nullptr) : QObject(parent) {
        m_db = QSqlDatabase::addDatabase("QSQLITE");
        QString path = QDir::currentPath() + "/mkgames_local.db";
        m_db.setDatabaseName(path);
        if (!m_db.open()) {
            qWarning() << "Local DB error:" << m_db.lastError().text();
            return;
        }
        initSchema();
    }

    ~LocalDB() { m_db.close(); }

    void addGame(const LocalGame &game) {
        QSqlQuery q(m_db);
        q.prepare("INSERT OR REPLACE INTO local_games "
                  "(server_game_id, name, install_path, version, exe_path, cover_url, category, tags, file_size) "
                  "VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)");
        q.addBindValue(game.serverGameId);
        q.addBindValue(game.name);
        q.addBindValue(game.installPath);
        q.addBindValue(game.version);
        q.addBindValue(game.exePath);
        q.addBindValue(game.coverUrl);
        q.addBindValue(game.category);
        q.addBindValue(game.tags);
        q.addBindValue(game.fileSize);
        q.exec();
    }

    void removeGame(int serverGameId) {
        QSqlQuery q(m_db);
        q.prepare("DELETE FROM local_games WHERE server_game_id = ?");
        q.addBindValue(serverGameId);
        q.exec();
    }

    LocalGame getGame(int serverGameId) {
        QSqlQuery q(m_db);
        q.prepare("SELECT * FROM local_games WHERE server_game_id = ?");
        q.addBindValue(serverGameId);
        LocalGame game;
        if (q.next()) {
            game = fromQuery(q);
        }
        return game;
    }

    bool isInstalled(int serverGameId) {
        QSqlQuery q(m_db);
        q.prepare("SELECT COUNT(*) FROM local_games WHERE server_game_id = ?");
        q.addBindValue(serverGameId);
        if (q.next()) return q.value(0).toInt() > 0;
        return false;
    }

    QList<LocalGame> getAllGames() {
        QSqlQuery q(m_db);
        q.prepare("SELECT * FROM local_games ORDER BY name");
        q.exec();
        QList<LocalGame> games;
        while (q.next()) {
            games.append(fromQuery(q));
        }
        return games;
    }

    void updateLastPlayed(int serverGameId) {
        QSqlQuery q(m_db);
        q.prepare("UPDATE local_games SET last_played = CURRENT_TIMESTAMP WHERE server_game_id = ?");
        q.addBindValue(serverGameId);
        q.exec();
    }

    void updateVersion(int serverGameId, const QString &version) {
        QSqlQuery q(m_db);
        q.prepare("UPDATE local_games SET version = ? WHERE server_game_id = ?");
        q.addBindValue(version);
        q.addBindValue(serverGameId);
        q.exec();
    }

private:
    QSqlDatabase m_db;

    void initSchema() {
        QSqlQuery q(m_db);
        q.exec("CREATE TABLE IF NOT EXISTS local_games ("
               "id INTEGER PRIMARY KEY AUTOINCREMENT, "
               "server_game_id INTEGER UNIQUE, "
               "name TEXT, "
               "install_path TEXT, "
               "version TEXT, "
               "exe_path TEXT, "
               "cover_url TEXT, "
               "category TEXT, "
               "tags TEXT, "
               "file_size INTEGER DEFAULT 0, "
               "installed_at DATETIME DEFAULT CURRENT_TIMESTAMP, "
               "last_played DATETIME DEFAULT NULL)");
    }

    LocalGame fromQuery(QSqlQuery &q) {
        LocalGame g;
        g.id = q.value("id").toInt();
        g.serverGameId = q.value("server_game_id").toInt();
        g.name = q.value("name").toString();
        g.installPath = q.value("install_path").toString();
        g.version = q.value("version").toString();
        g.exePath = q.value("exe_path").toString();
        g.coverUrl = q.value("cover_url").toString();
        g.category = q.value("category").toString();
        g.tags = q.value("tags").toString();
        g.fileSize = q.value("file_size").toLongLong();
        g.installedAt = q.value("installed_at").toDateTime();
        g.lastPlayed = q.value("last_played").toDateTime();
        return g;
    }
};

#endif
