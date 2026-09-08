#include "archiveextractor.h"
#include <QFileInfo>
#include <QProcess>
#include <QDebug>
#include <QRegularExpression>

ArchiveExtractor::ArchiveExtractor(QObject *parent) : QObject(parent) {}

static QString findTool(const QString &name) {
#ifdef Q_OS_WIN
    QStringList paths = {
        "C:/Program Files/7-Zip/7z.exe",
        "C:/Program Files (x86)/7-Zip/7z.exe",
        "C:/Program Files/WinRAR/UnRAR.exe",
        "C:/Program Files/WinRAR/unrar.exe",
    };
    for (const QString &p : paths) {
        if (QFileInfo::exists(p)) return p;
    }
    return name;
#else
    return name;
#endif
}

ArchiveExtractor::ArchiveType ArchiveExtractor::detectType(const QString &path) {
    QString lower = path.toLower();
    if (lower.endsWith(".zip")) return ZIP;
    if (lower.endsWith(".rar")) return RAR;
    if (lower.endsWith(".7z")) return SEVENZ;
    if (lower.endsWith(".tar.gz") || lower.endsWith(".tgz") || lower.endsWith(".tar")) return TARGZ;
    return UNKNOWN;
}

bool ArchiveExtractor::isSupported(const QString &path) {
    return detectType(path) != UNKNOWN;
}

QStringList ArchiveExtractor::supportedExtensions() {
    return {"*.zip", "*.rar", "*.7z", "*.tar.gz", "*.tgz", "*.tar"};
}

bool ArchiveExtractor::extract(const QString &archivePath, const QString &destDir, const QString &subfolder) {
    if (m_extracting) {
        emit extractionError("Already extracting");
        return false;
    }

    QFileInfo fi(archivePath);
    if (!fi.exists()) {
        qDebug() << "[EXTRACTOR] Archive not found:" << archivePath;
        emit extractionError("Archive not found: " + archivePath);
        return false;
    }

    qDebug() << "[EXTRACTOR] Starting extraction:" << archivePath << "->" << destDir << "Size:" << fi.size();
    QDir().mkpath(destDir);
    m_destDir = destDir;
    m_extracting = true;
    m_totalFiles = 0;
    m_extractedFiles = 0;

    ArchiveType type = detectType(archivePath);
    qDebug() << "[EXTRACTOR] Archive type:" << type;
    bool result = false;

    switch (type) {
        case ZIP: result = extractZip(archivePath, destDir); break;
        case RAR: result = extractRar(archivePath, destDir); break;
        case SEVENZ: result = extract7z(archivePath, destDir); break;
        case TARGZ: result = extractTarGz(archivePath, destDir); break;
        default:
            m_extracting = false;
            emit extractionError("Unsupported archive format");
            return false;
    }

    return result;
}

bool ArchiveExtractor::extractZip(const QString &archive, const QString &dest) {
#ifdef Q_OS_WIN
    QProcess *ps = new QProcess(this);
    connect(ps, &QProcess::readyReadStandardOutput, this, &ArchiveExtractor::onProcessReadyRead);
    connect(ps, &QProcess::readyReadStandardError, this, &ArchiveExtractor::onProcessReadyRead);
    connect(ps, &QProcess::finished, this, &ArchiveExtractor::onProcessFinished);
    m_process = ps;

    QString psCmd = QString("Expand-Archive -Path '%1' -DestinationPath '%2' -Force").arg(
        QDir::toNativeSeparators(archive), QDir::toNativeSeparators(dest));
    qDebug() << "[EXTRACTOR] Running PowerShell:" << psCmd;
    ps->start("powershell.exe", {"-NoProfile", "-NonInteractive", "-Command", psCmd});

    if (!ps->waitForStarted()) {
        m_extracting = false;
        emit extractionError("Failed to start PowerShell for ZIP extraction");
        ps->deleteLater();
        m_process = nullptr;
        return false;
    }
    return true;
#else
    m_process = new QProcess(this);
    connect(m_process, &QProcess::readyReadStandardOutput, this, &ArchiveExtractor::onProcessReadyRead);
    connect(m_process, &QProcess::readyReadStandardError, this, &ArchiveExtractor::onProcessReadyRead);
    connect(m_process, &QProcess::finished, this, &ArchiveExtractor::onProcessFinished);

    QStringList args;
    args << "-o" << archive << "-d" << dest;
    qDebug() << "[EXTRACTOR] Running: unzip" << args.join(" ");
    m_process->start("unzip", args);

    if (!m_process->waitForStarted()) {
        m_extracting = false;
        emit extractionError("Failed to start unzip");
        m_process->deleteLater();
        m_process = nullptr;
        return false;
    }
    return true;
#endif
}

bool ArchiveExtractor::extractRar(const QString &archive, const QString &dest) {
    m_process = new QProcess(this);
    connect(m_process, &QProcess::readyReadStandardOutput, this, &ArchiveExtractor::onProcessReadyRead);
    connect(m_process, &QProcess::readyReadStandardError, this, &ArchiveExtractor::onProcessReadyRead);
    connect(m_process, &QProcess::finished, this, &ArchiveExtractor::onProcessFinished);

    QStringList args;
#ifdef Q_OS_WIN
    QString tool = findTool("unrar");
    if (tool == "unrar") {
        tool = findTool("7z");
        args << "x" << "-y" << ("-o" + dest) << archive;
    } else {
        args << "x" << "-o+" << "-y" << archive << dest;
    }
#else
    QString tool = "unrar";
    args << "x" << "-o+" << "-y" << archive << dest;
#endif
    qDebug() << "[EXTRACTOR] Running:" << tool << args.join(" ");
    m_process->start(tool, args);

    if (!m_process->waitForStarted()) {
        m_extracting = false;
        emit extractionError("Failed to start unrar/7z. Install 7-Zip, WinRAR, or run: sudo apt install unrar");
        m_process->deleteLater();
        m_process = nullptr;
        return false;
    }
    return true;
}

bool ArchiveExtractor::extract7z(const QString &archive, const QString &dest) {
    m_process = new QProcess(this);
    connect(m_process, &QProcess::readyReadStandardOutput, this, &ArchiveExtractor::onProcessReadyRead);
    connect(m_process, &QProcess::readyReadStandardError, this, &ArchiveExtractor::onProcessReadyRead);
    connect(m_process, &QProcess::finished, this, &ArchiveExtractor::onProcessFinished);

    QStringList args;
#ifdef Q_OS_WIN
    QString tool = findTool("7z");
    if (tool == "7z") {
        m_extracting = false;
        emit extractionError("7-Zip not found. Install 7-Zip to extract .7z files.");
        m_process->deleteLater();
        m_process = nullptr;
        return false;
    }
#else
    QString tool = "7z";
#endif
    args << "x" << "-y" << ("-o" + dest) << archive;
    qDebug() << "[EXTRACTOR] Running:" << tool << args.join(" ");
    m_process->start(tool, args);

    if (!m_process->waitForStarted()) {
        m_extracting = false;
        emit extractionError("Failed to start 7z");
        m_process->deleteLater();
        m_process = nullptr;
        return false;
    }
    return true;
}

bool ArchiveExtractor::extractTarGz(const QString &archive, const QString &dest) {
    m_process = new QProcess(this);
    connect(m_process, &QProcess::readyReadStandardOutput, this, &ArchiveExtractor::onProcessReadyRead);
    connect(m_process, &QProcess::finished, this, &ArchiveExtractor::onProcessFinished);

    QStringList args;
#ifdef Q_OS_WIN
    QString tool = findTool("7z");
    args << "x" << "-y" << ("-o" + dest) << archive;
#else
    QString tool = "tar";
    args << "-xzf" << archive << "-C" << dest;
#endif
    m_process->start(tool, args);

    if (!m_process->waitForStarted()) {
        m_extracting = false;
        emit extractionError("Failed to start extraction tool");
        m_process->deleteLater();
        m_process = nullptr;
        return false;
    }
    return true;
}

bool ArchiveExtractor::cancelExtraction() {
    if (m_process && m_extracting) {
        m_process->kill();
        m_process->waitForFinished(3000);
        m_extracting = false;
        return true;
    }
    return false;
}

void ArchiveExtractor::onProcessReadyRead() {
    if (!m_process) return;

    QString output = QString::fromUtf8(m_process->readAllStandardOutput());
    QRegularExpression re("(\\d+)\\%");
    QRegularExpressionMatch match = re.match(output);
    if (match.hasMatch()) {
        int pct = match.captured(1).toInt();
        emit extractionProgress(pct, "");
    }

    QRegularExpression inflating("inflating:\\s+(.+)$");
    QRegularExpression extracting("extracting:\\s+(.+)$");
    for (const QString &line : output.split('\n')) {
        QString trimmed = line.trimmed();
        QRegularExpressionMatch fm = inflating.match(trimmed);
        if (!fm.hasMatch()) fm = extracting.match(trimmed);
        if (fm.hasMatch() && !fm.captured(1).isEmpty()) {
            m_extractedFiles++;
            QFileInfo fi(fm.captured(1).trimmed());
            emit extractionProgress(-1, fi.fileName());
        }
    }
}

void ArchiveExtractor::onProcessFinished(int exitCode, QProcess::ExitStatus exitStatus) {
    QByteArray errOutput;
    if (m_process) {
        errOutput = m_process->readAllStandardError();
    }

    qDebug() << "[EXTRACTOR] Process finished. Exit code:" << exitCode << "Status:" << exitStatus;
    qDebug() << "[EXTRACTOR] Files extracted:" << m_extractedFiles;
    if (!errOutput.isEmpty()) {
        qDebug() << "[EXTRACTOR] Stderr:" << QString::fromUtf8(errOutput).left(500);
    }

    m_extracting = false;
    if (m_process) {
        m_process->deleteLater();
        m_process = nullptr;
    }

    if (exitStatus == QProcess::NormalExit && exitCode == 0) {
        emit extractionProgress(100, "");
        emit extractionComplete(m_destDir);
    } else {
        emit extractionError("Extraction failed (exit code " + QString::number(exitCode) + "): " + QString::fromUtf8(errOutput));
    }
}
