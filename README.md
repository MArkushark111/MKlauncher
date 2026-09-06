# MKGames

MKGames is a game distribution system with a player launcher, an admin panel, and a Go server.

## Components

- `client/` - Qt player launcher for browsing, downloading, installing, updating, launching, and uninstalling games.
- `admin/` - Qt admin launcher and passcode-protected web admin panel.
- `server/` - Go server, API, SQLite database code, and game storage.
- `install.sh` - Ubuntu/Debian server installer.

## How It Works

1. The Ubuntu server hosts the game files and API.
2. An administrator opens the admin panel and signs in with the admin passcode.
3. The administrator adds games, uploads archives, configures the WAN address, and manages versions.
4. Players enter the public `WAN:port` in the Qt launcher. Players do not need an admin password.

## Ubuntu Server Installation

```bash
sudo apt update
sudo apt install -y git
git clone https://github.com/MArkushark111/MKlauncher.git
cd MKlauncher
chmod +x install.sh
sudo ./install.sh
```

The installer installs Go, builds the server, asks for the initial admin passcode, installs a `systemd` service, and creates the `mkgames` and `mklauncher` commands.

Start and inspect the server:

```bash
sudo systemctl start mkgames
sudo systemctl status mkgames
sudo mklauncher wan
```

View live server logs:

```bash
sudo journalctl -u mkgames -f
```

After configuring the WAN host and port in the admin panel, players connect to `WAN_HOST:WAN_PORT`, for example `203.0.113.50:8080`.

The server must be reachable through the firewall and router port forwarding if it is behind a private network.

## Admin Panel

Open the server address in a browser:

```text
http://YOUR_WAN_HOST:YOUR_WAN_PORT
```

The admin panel supports adding, editing, and deleting games; uploading archives, images, and new versions; managing admins; configuring WAN settings; viewing server health; and applying configuration with a server restart.

## Qt Builds

Qt 6 with MinGW is required for the Windows desktop applications.

Open `client/CMakeLists.txt` in Qt Creator to build the `MKLauncher` player target.

Open `admin/CMakeLists.txt` in Qt Creator to build the `MKGamesAdmin` target. It asks for a server address and opens the web admin panel in the default browser.

The optional `server/installer/CMakeLists.txt` project builds `MKGamesServerInstaller`. It is intended for computers with a graphical desktop and is not needed on a headless Ubuntu server.

## Manual Go Build

```bash
cd server
go mod download
go build -o mkgames-server .
```

The default server database path is `/var/lib/mkgames`, and the default port is `8080`.

## Git Workflow

```bash
git pull
git add .
git commit -m "Describe your change"
git push
```

Generated build folders, local databases, runtime storage, credentials, and server binaries are excluded by `.gitignore`.