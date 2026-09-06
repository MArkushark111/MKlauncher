#ifndef THEME_H
#define THEME_H

#include <QString>

class Theme {
public:
    static QString getStyleSheet() {
        return R"(
            * {
                font-family: 'Consolas', 'Courier New', monospace;
            }
            QWidget {
                background-color: #0a0a0a;
                color: #e0e0e0;
            }
            QMainWindow {
                background-color: #0a0a0a;
            }
            QLabel {
                color: #e0e0e0;
                background: transparent;
            }
            QPushButton {
                background-color: #1a1a1a;
                color: #e0e0e0;
                border: 1px solid #2a2a2a;
                border-radius: 4px;
                padding: 8px 16px;
                font-size: 13px;
                font-weight: bold;
                letter-spacing: 1px;
            }
            QPushButton:hover {
                background-color: #222222;
                border-color: #00ff88;
            }
            QPushButton:pressed {
                background-color: #00ff88;
                color: #000000;
            }
            QPushButton#playBtn {
                background-color: #00ff88;
                color: #000000;
                border: none;
                font-size: 14px;
                padding: 10px 24px;
            }
            QPushButton#playBtn:hover {
                background-color: #00cc6a;
            }
            QPushButton#installBtn {
                background-color: #ffffff;
                color: #000000;
                border: none;
                font-size: 12px;
                padding: 10px 24px;
            }
            QPushButton#installBtn:hover {
                background-color: #e0e0e0;
            }
            QPushButton#updateBtn {
                background-color: #ffaa00;
                color: #000000;
                border: none;
            }
            QPushButton#updateBtn:hover {
                background-color: #cc8800;
            }
            QPushButton#uninstallBtn {
                background-color: #ff4444;
                color: #ffffff;
                border: none;
            }
            QPushButton#uninstallBtn:hover {
                background-color: #cc3333;
            }
            QPushButton#menuBtn {
                background: transparent;
                color: #888888;
                font-size: 18px;
                padding: 4px 8px;
                border: none;
            }
            QPushButton#menuBtn:hover {
                color: #00ff88;
            }
            QLineEdit {
                background-color: #111111;
                color: #e0e0e0;
                border: 1px solid #2a2a2a;
                border-radius: 4px;
                padding: 10px 14px;
                font-size: 14px;
            }
            QLineEdit:focus {
                border-color: #00ff88;
            }
            QProgressBar {
                background-color: #111111;
                border: 1px solid #2a2a2a;
                border-radius: 4px;
                text-align: center;
                color: #e0e0e0;
                font-size: 12px;
                height: 24px;
            }
            QProgressBar::chunk {
                background: qlineargradient(x1:0, y1:0, x2:1, y2:0,
                    stop:0 #00ff88, stop:1 #00cc6a);
                border-radius: 3px;
            }
            QScrollArea {
                border: none;
                background: transparent;
            }
            QScrollBar:vertical {
                background: #111111;
                width: 8px;
                border-radius: 4px;
            }
            QScrollBar::handle:vertical {
                background: #2a2a2a;
                border-radius: 4px;
                min-height: 30px;
            }
            QScrollBar::handle:vertical:hover {
                background: #00ff88;
            }
            QScrollBar::add-line:vertical, QScrollBar::sub-line:vertical {
                height: 0px;
            }
            QMenu {
                background-color: #1a1a1a;
                border: 1px solid #2a2a2a;
                border-radius: 4px;
                padding: 4px;
            }
            QMenu::item {
                padding: 8px 24px;
                color: #e0e0e0;
            }
            QMenu::item:selected {
                background-color: #00ff88;
                color: #000000;
            }
            QToolTip {
                background-color: #1a1a1a;
                color: #e0e0e0;
                border: 1px solid #00ff88;
                padding: 4px;
            }
            QMessageBox {
                background-color: #0a0a0a;
            }
            QDialog {
                background-color: #111111;
                border: 1px solid #2a2a2a;
            }
            QComboBox {
                background-color: #111111;
                color: #e0e0e0;
                border: 1px solid #2a2a2a;
                border-radius: 4px;
                padding: 8px 12px;
            }
            QComboBox::drop-down {
                border: none;
            }
            QComboBox QAbstractItemView {
                background-color: #1a1a1a;
                color: #e0e0e0;
                selection-background-color: #00ff88;
                selection-color: #000000;
            }
        )";
    }

    static QString cardStyle() {
        return R"(
            background-color: #1a1a1a;
            border: 1px solid #2a2a2a;
            border-radius: 8px;
        )";
    }

    static QString accentColor() { return "#00ff88"; }
    static QString dangerColor() { return "#ff4444"; }
    static QString warningColor() { return "#ffaa00"; }
    static QString bgPrimary() { return "#0a0a0a"; }
    static QString bgSecondary() { return "#111111"; }
    static QString bgCard() { return "#1a1a1a"; }
    static QString textPrimary() { return "#e0e0e0"; }
    static QString textSecondary() { return "#888888"; }
    static QString border() { return "#2a2a2a"; }
};

#endif
