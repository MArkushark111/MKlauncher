#include "gamegrid.h"
#include "theme.h"
#include "localdb.h"
#include <QNetworkRequest>
#include <QJsonDocument>
#include <QJsonObject>
#include <QJsonArray>
#include <QFile>
#include <QDir>
#include <QDesktopServices>
#include <QUrl>

GameGrid::GameGrid(QWidget *parent) : QWidget(parent) {
    m_apiManager = new QNetworkAccessManager(this);
    m_coverManager = new QNetworkAccessManager(this);
    connect(m_apiManager, &QNetworkAccessManager::finished, this, &GameGrid::onGamesLoaded);
    connect(m_coverManager, &QNetworkAccessManager::finished, this, &GameGrid::onCoverLoaded);

    auto *mainLayout = new QVBoxLayout(this);
    mainLayout->setContentsMargins(0, 0, 0, 0);

    m_scrollArea = new QScrollArea(this);
    m_scrollArea->setWidgetResizable(true);
    m_scrollArea->setFrameShape(QFrame::NoFrame);

    m_gridWidget = new QWidget();
    m_gridWidget->setSizePolicy(QSizePolicy::Expanding, QSizePolicy::Preferred);
    m_grid = new QGridLayout(m_gridWidget);
    m_grid->setSpacing(16);
    m_grid->setContentsMargins(24, 24, 24, 24);

    m_scrollArea->setWidget(m_gridWidget);
    mainLayout->addWidget(m_scrollArea);
}

void GameGrid::loadGames(const QString &serverUrl, const QString &token) {
    m_serverUrl = serverUrl;
    m_token = token;

    QNetworkRequest request(QUrl(serverUrl + "/api/games"));
    request.setRawHeader("Authorization", "Bearer " + token.toUtf8());
    request.setTransferTimeout(15000);
    m_apiManager->get(request);
}

void GameGrid::onGamesLoaded(QNetworkReply *reply) {
    if (reply->error() != QNetworkReply::NoError) {
        emit gamesLoadError(reply->errorString());
        reply->deleteLater();
        return;
    }

    QByteArray data = reply->readAll();
    QJsonDocument doc = QJsonDocument::fromJson(data);
    reply->deleteLater();

    m_games.clear();
    if (doc.isArray()) {
        for (const QJsonValue &val : doc.array()) {
            QJsonObject obj = val.toObject();
            ServerGame game;
            game.id = obj["id"].toInt();
            game.name = obj["name"].toString();
            game.description = obj["description"].toString();
            game.version = obj["version"].toString();
            game.category = obj["category"].toString();
            game.tags = obj["tags"].toString();
            game.coverUrl = obj["cover_url"].toString();
            game.backgroundUrl = obj["background_url"].toString();
            game.logoUrl = obj["logo_url"].toString();
            game.wideCoverUrl = obj["wide_cover_url"].toString();
            game.archivePath = obj["archive_path"].toString();
            game.gameFolder = obj["game_folder"].toString();
            game.exePath = obj["exe_path"].toString();
            game.fileSize = obj["file_size"].toVariant().toLongLong();
            game.downloadCount = obj["download_count"].toInt();
            game.avgStars = obj["avg_stars"].toDouble();
            game.reviewCount = obj["review_count"].toInt();
            game.downloadUrl = obj["download_url"].toString();
            game.storageType = obj["storage_type"].toString();
            game.mirrorUrls = obj["mirror_urls"].toString();
            game.status = obj["status"].toString();
            if (game.status.isEmpty()) game.status = "released";
            m_games.append(game);
        }
    }

    emit gamesLoadError(QString());
    buildGrid();
}

void GameGrid::buildGrid() {
    QLayoutItem *item;
    while ((item = m_grid->takeAt(0)) != nullptr) {
        if (item->widget()) item->widget()->deleteLater();
        delete item;
    }

    int cols = 4;
    for (int i = 0; i < cols; i++) {
        m_grid->setColumnStretch(i, 0);
    }
    int row = 0, col = 0;
    int shown = 0;

    LocalDB localDB;

    for (const ServerGame &game : m_games) {
        bool textMatch = m_searchFilter.isEmpty() ||
            game.name.toLower().contains(m_searchFilter) ||
            game.tags.toLower().contains(m_searchFilter) ||
            game.category.toLower().contains(m_searchFilter);

        bool installed = localDB.isInstalled(game.id);
        bool statusMatch = m_statusFilter.isEmpty() ||
            (m_statusFilter == "installed" && installed) ||
            (m_statusFilter == "not_installed" && !installed);

        bool categoryMatch = m_categoryFilter.isEmpty() ||
            game.tags.toLower().contains(m_categoryFilter.toLower()) ||
            game.category.toLower().contains(m_categoryFilter.toLower());

        if (!textMatch || !statusMatch || !categoryMatch) continue;

        QWidget *card = createGameCard(game);
        m_grid->addWidget(card, row, col);
        col++;
        if (col >= cols) { col = 0; row++; }
        shown++;
    }

    if (col > 0 && shown > 0) {
        m_grid->addItem(new QSpacerItem(0, 0, QSizePolicy::Expanding, QSizePolicy::Minimum), row, col, 1, cols - col);
    }

    m_grid->addItem(new QSpacerItem(0, 0, QSizePolicy::Minimum, QSizePolicy::Expanding), row + 1, 0, 1, cols);

    if (shown == 0) {
        QLabel *emptyLabel = new QLabel(m_games.isEmpty() ? "No games available" : "No matching games");
        emptyLabel->setAlignment(Qt::AlignCenter);
        emptyLabel->setStyleSheet("color: #555555; font-size: 16px; padding: 60px;");
        m_grid->addWidget(emptyLabel, 0, 0, 1, cols);
    }
}

QWidget* GameGrid::createGameCard(const ServerGame &game) {
    QWidget *card = new QWidget();
    card->setObjectName("gameCard");
    card->setMinimumSize(240, 320);
    card->setMaximumSize(280, 360);
    card->setStyleSheet(R"(
        #gameCard {
            background-color: #1a1a1a;
            border: 1px solid #333333;
            border-radius: 12px;
        }
        #gameCard:hover {
            border-color: #00ff88;
        }
    )");

    auto *layout = new QVBoxLayout(card);
    layout->setContentsMargins(0, 0, 0, 0);
    layout->setSpacing(0);

    QLabel *coverLabel = new QLabel(card);
    coverLabel->setFixedHeight(160);
    coverLabel->setStyleSheet("background-color: #111111; border-top-left-radius: 12px; border-top-right-radius: 12px; color: #555555; font-size: 12px;");
    coverLabel->setAlignment(Qt::AlignCenter);
    coverLabel->setText("LOADING...");
    layout->addWidget(coverLabel);
    m_coverLabels[game.id] = coverLabel;

    connect(card, &QWidget::customContextMenuRequested, this, [this, game](const QPoint &) {
        emit gameDetails(game);
    });
    card->setContextMenuPolicy(Qt::CustomContextMenu);

    if (!game.coverUrl.isEmpty()) {
        QString url = imageUrl(game.coverUrl);
        qDebug() << "[COVER] Loading cover for" << game.name << "URL:" << url;
        QUrl coverUrl(url);
        QNetworkRequest req(coverUrl);
        req.setTransferTimeout(10000);
        QNetworkReply *reply = m_coverManager->get(req);
        m_coverReplies[reply] = game.id;
    }

    QWidget *infoWidget = new QWidget(card);
    infoWidget->setStyleSheet("background: transparent;");
    auto *infoLayout = new QVBoxLayout(infoWidget);
    infoLayout->setContentsMargins(12, 8, 12, 4);
    infoLayout->setSpacing(2);

    QLabel *nameLabel = new QLabel(game.name, infoWidget);
    nameLabel->setStyleSheet("color: #e0e0e0; font-weight: bold; font-size: 14px; background: transparent;");
    nameLabel->setWordWrap(true);
    nameLabel->setMaximumHeight(36);
    infoLayout->addWidget(nameLabel);

    QLabel *metaLabel = new QLabel(
        QString("%1 | %2")
            .arg(game.version)
            .arg(game.category.isEmpty() ? "N/A" : game.category),
        infoWidget);
    metaLabel->setStyleSheet("color: #888888; font-size: 11px; background: transparent;");
    infoLayout->addWidget(metaLabel);

    if (game.status == "coming_soon") {
        QLabel *statusLabel = new QLabel("COMING SOON", infoWidget);
        statusLabel->setStyleSheet("color: #ffaa00; font-size: 10px; font-weight: bold; background: transparent; letter-spacing: 1px;");
        infoLayout->addWidget(statusLabel);
    }

    layout->addWidget(infoWidget);

    QWidget *btnWidget = new QWidget(card);
    btnWidget->setStyleSheet("background: transparent;");
    auto *btnLayout = new QHBoxLayout(btnWidget);
    btnLayout->setContentsMargins(12, 4, 12, 12);
    btnLayout->setSpacing(8);

    LocalDB localDB;
    bool installed = localDB.isInstalled(game.id);
    LocalGame localGame = localDB.getGame(game.id);

    if (installed) {
        bool needsUpdate = localGame.version != game.version;

        QPushButton *playBtn = new QPushButton("PLAY", btnWidget);
        playBtn->setObjectName("playBtn");
        playBtn->setCursor(Qt::PointingHandCursor);
        playBtn->setStyleSheet(
            "QPushButton#playBtn { background-color: #00ff88; color: #000000; border: none; border-radius: 4px; font-size: 14px; padding: 10px 24px; }"
            "QPushButton#playBtn:hover { background-color: #00cc6a; }");
        connect(playBtn, &QPushButton::clicked, [this, game, localGame]() {
            emit gamePlay(game.id, game.name, game.exePath, localGame.installPath);
        });
        btnLayout->addWidget(playBtn);

        QPushButton *menuBtn = new QPushButton("...", btnWidget);
        menuBtn->setObjectName("menuBtn");
        menuBtn->setCursor(Qt::PointingHandCursor);
        menuBtn->setMaximumWidth(40);

        QMenu *menu = new QMenu(menuBtn);
        menuBtn->setMenu(menu);

        if (needsUpdate) {
            QAction *updateAction = menu->addAction("UPDATE");
            QFont f = updateAction->font();
            f.setBold(true);
            updateAction->setFont(f);
            connect(updateAction, &QAction::triggered, [this, game, localGame]() {
                emit gameUpdate(game.id, game.name, localGame.version, game.version,
                               m_serverUrl + "/api/games/" + QString::number(game.id) + "/download");
            });
        }

        QAction *uninstallAction = menu->addAction("UNINSTALL");
        QFont uf = uninstallAction->font();
        uf.setBold(true);
        uninstallAction->setFont(uf);
        connect(uninstallAction, &QAction::triggered, [this, game, localGame]() {
            emit gameUninstall(game.id, game.name, localGame.installPath);
        });

        btnLayout->addWidget(menuBtn);
    } else if (game.status == "coming_soon") {
        QPushButton *wishlistBtn = new QPushButton("WISHLIST", btnWidget);
        wishlistBtn->setObjectName("wishlistBtn");
        wishlistBtn->setCursor(Qt::PointingHandCursor);
        wishlistBtn->setStyleSheet(
            "QPushButton#wishlistBtn { background-color: transparent; color: #ff4488; border: 1px solid #ff4488; border-radius: 4px; font-size: 12px; padding: 10px 24px; font-weight: bold; }"
            "QPushButton#wishlistBtn:hover { background-color: #ff4488; color: #000000; }");
        connect(wishlistBtn, &QPushButton::clicked, [this, game]() {
            emit gameDetails(game);
        });
        btnLayout->addWidget(wishlistBtn);
    } else {
        QPushButton *downloadBtn = new QPushButton(
            QString("INSTALL - %1").arg(formatSize(game.fileSize)), btnWidget);
        downloadBtn->setObjectName("installBtn");
        downloadBtn->setCursor(Qt::PointingHandCursor);
        downloadBtn->setStyleSheet(
            "QPushButton#installBtn { background-color: #ffffff; color: #000000; border: none; border-radius: 4px; font-size: 12px; padding: 10px 24px; font-weight: bold; }"
            "QPushButton#installBtn:hover { background-color: #e0e0e0; }");
        connect(downloadBtn, &QPushButton::clicked, [this, game]() {
            QString dlUrl;
            if (game.storageType == "url" && !game.downloadUrl.isEmpty()) {
                dlUrl = game.downloadUrl;
            } else {
                dlUrl = m_serverUrl + "/api/games/" + QString::number(game.id) + "/download";
            }
            emit gameDownload(game.id, game.name, dlUrl, game.fileSize);
        });
        btnLayout->addWidget(downloadBtn);
    }

    btnLayout->addStretch();
    layout->addWidget(btnWidget);

    return card;
}

void GameGrid::onCoverLoaded(QNetworkReply *reply) {
    int gameId = m_coverReplies.take(reply);
    if (reply->error() != QNetworkReply::NoError) {
        qDebug() << "[COVER] Failed to load cover for game" << gameId << ":" << reply->errorString() << "URL:" << reply->url();
        reply->deleteLater();
        return;
    }

    QByteArray imgData = reply->readAll();
    QPixmap pixmap;
    pixmap.loadFromData(imgData);
    reply->deleteLater();

    if (pixmap.isNull()) {
        qDebug() << "[COVER] Invalid pixmap for game" << gameId << "size:" << imgData.size();
        return;
    }
    m_covers[gameId] = pixmap;
    if (m_coverLabels.contains(gameId)) {
        QLabel *label = m_coverLabels[gameId];
        QPixmap scaled = pixmap.scaled(label->width() > 0 ? label->width() : 280,
                                       label->height() > 0 ? label->height() : 160,
                                       Qt::KeepAspectRatioByExpanding, Qt::SmoothTransformation);
        label->setPixmap(scaled);
        label->setText(QString());
        label->update();
    }
}

void GameGrid::refreshGrid() {
    if (!m_serverUrl.isEmpty() && !m_token.isEmpty()) {
        loadGames(m_serverUrl, m_token);
    }
}

QString GameGrid::formatSize(qint64 bytes) const {
    if (bytes < 1024) return QString::number(bytes) + " B";
    if (bytes < 1048576) return QString::number(bytes / 1024.0, 'f', 1) + " KB";
    if (bytes < 1073741824) return QString::number(bytes / 1048576.0, 'f', 1) + " MB";
    return QString::number(bytes / 1073741824.0, 'f', 2) + " GB";
}

QString GameGrid::imageUrl(const QString &url) const {
    if (url.startsWith("http://") || url.startsWith("https://")) return url;
    QString path = url.trimmed().replace(QRegularExpression("^/+"), "");
    if (path.startsWith("storage/covers/")) {
        path = path.mid(QString("storage/").length());
    }
    return m_serverUrl + "/" + path;
}

void GameGrid::filterByText(const QString &text) {
    m_searchFilter = text.toLower();
    buildGrid();
}

void GameGrid::filterByStatus(const QString &status) {
    m_statusFilter = status;
    buildGrid();
}

void GameGrid::filterByCategory(const QString &category) {
    m_categoryFilter = category;
    buildGrid();
}
