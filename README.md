# SERVO — Server Monitoring Dashboard

A lightweight, zero-dependency server monitoring dashboard for **Apache** and **Nginx**.  
Built for **HestiaCP** and Linux-based hosting environments.

Available in two editions:
- ⚡ **Go Binary** — single binary, runs as a service on any port
- 🐘 **PHP** — single `index.php`, drop into any Apache/Nginx vhost

---

## ✨ Features

- **Auto-Discovery** — detects Apache & Nginx status endpoints automatically
- **VHost Dashboard** — per-domain CPU, workers, requests, unique IPs
- **CPU Allocation Chart** — doughnut chart of system-wide CPU per domain
- **Worker Mode Distribution** — Writing / Keepalive / Idle breakdown
- **Critical Alerts** — live badge when any vhost hits critical load
- **Summary Modal** — full system overview with scrollable table
- **Live Refresh** — data updates every 3 seconds
- **Dark Mode UI** — premium GitHub-dark themed interface
- **Zero Configuration** — just run it

---

## 🚀 Go Binary Edition

### Method 1 — One-line Installer (Recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/febritecno/servo/main/install.sh | bash
```

> Automatically detects OS/architecture, installs to `/usr/local/bin/servo`,  
> and creates a **systemd service** (Linux) or **LaunchAgent** (macOS).

**Custom port:**
```bash
curl -fsSL https://raw.githubusercontent.com/febritecno/servo/main/install.sh | bash -s -- --port :9090
```

**Uninstall:**
```bash
curl -fsSL https://raw.githubusercontent.com/febritecno/servo/main/install.sh | bash -s -- --uninstall
```

---

### Method 2 — Manual Download

Download the binary for your platform from the [Releases](https://github.com/febritecno/servo/releases) page:

| Platform | File |
|----------|------|
| Linux x86_64 (most VPS) | `servo_*_linux_amd64.tar.gz` |
| Linux ARM64 (Oracle, RPi) | `servo_*_linux_arm64.tar.gz` |
| Linux ARM | `servo_*_linux_arm.tar.gz` |
| macOS Intel | `servo_*_darwin_amd64.tar.gz` |
| macOS Apple Silicon | `servo_*_darwin_arm64.tar.gz` |
| Windows x64 | `servo_*_windows_amd64.zip` |
| FreeBSD | `servo_*_freebsd_amd64.tar.gz` |

```bash
# Example: Linux amd64
wget https://github.com/febritecno/servo/releases/latest/download/servo_linux_amd64.tar.gz
tar xzf servo_linux_amd64.tar.gz
chmod +x servo
sudo mv servo /usr/local/bin/
```

---

### Method 3 — Build from Source

```bash
git clone https://github.com/febritecno/servo.git
cd servo
go build -o servo .

# Run
./servo --port :8080
```

**Build all platforms:**
```bash
chmod +x build.sh
./build.sh
# Output: dist/
```

---

### Running the Go Binary

```bash
# Production (auto-detect Apache & Nginx)
./servo --port :8080

# Specify endpoints manually
./servo --port :8080 \
  --apache http://127.0.0.1:8080/server-status \
  --nginx  http://127.0.0.1/nginx_status

# Test with dummy data
./servo --port :8080 --dummy

# Run in background
nohup ./servo --port :8080 > servo.log 2>&1 &

# Or use the included helper script
chmod +x start.sh && ./start.sh
```

**CLI flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--port` | Port to listen on | `:8080` |
| `--apache` | Apache `server-status` URL | *auto-detect* |
| `--nginx` | Nginx `nginx_status` URL | *auto-detect* |
| `--dummy` | Use sample data for testing | `false` |

---

### Systemd Service (Linux)

The installer sets this up automatically. Manual setup:

```bash
sudo tee /etc/systemd/system/servo.service << 'EOF'
[Unit]
Description=SERVO Server Monitor Dashboard
After=network.target

[Service]
ExecStart=/usr/local/bin/servo --port :8080
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now servo
```

**Manage the service:**
```bash
systemctl status servo      # Check status
systemctl restart servo     # Restart
journalctl -u servo -f      # Live logs
systemctl stop servo        # Stop
```

---

## 🐘 PHP Edition

A single-file PHP version with **identical UI and functionality**.  
No build step, no dependencies — just drop and run.

### Install

**Method 1 — One-liner download (Recommended)**

```bash
# Download langsung ke document root vhost
wget -O /home/user/web/yourdomain.com/public_html/index.php \
  https://raw.githubusercontent.com/febritecno/servo/main/index.php
```

Or using `curl`:
```bash
curl -fsSL https://raw.githubusercontent.com/febritecno/servo/main/index.php \
  -o /home/user/web/yourdomain.com/public_html/index.php
```

**Method 2 — Manual copy** (jika sudah clone repo)

```bash
cp index.php /home/user/web/yourdomain.com/public_html/index.php
```

Then open: `https://yourdomain.com/`

> **Default password:** `servo@admin123` — **Ganti setelah install!**
>
> Edit baris `SERVO_PASSWORD_HASH` di `index.php`. Generate hash baru dengan:
> ```bash
> php -r "echo password_hash('PasswordBaru', PASSWORD_BCRYPT, ['cost'=>12]);"
> ```

> **Security Note:** Pastikan domain ini tidak dapat diakses publik — gunakan IP whitelist via `.htaccess` atau firewall.

### PHP Requirements

| Requirement | Detail |
|-------------|--------|
| PHP version | 7.4 or later |
| `allow_url_fopen` | `On` (default in HestiaCP) |
| Apache mod_status | Enabled with `ExtendedStatus On` |
| Nginx stub_status | Enabled |

### How it Works

- `GET /` → renders the dashboard HTML
- `GET /?api=1` → returns JSON data (called by Alpine.js every 3s)
- Auto-discovers Apache on ports `8080`, `8081`, `80`
- Auto-discovers Nginx on `/nginx_status`

### Restrict Access (Recommended)

Add to `.htaccess` in the same directory:

```apache
# Allow only localhost and your own IP
Order Deny,Allow
Deny from all
Allow from 127.0.0.1
Allow from YOUR.IP.HERE
```

---

## 🔧 Apache & Nginx Configuration

### Enable Apache mod_status (HestiaCP)

Edit Apache config (usually `/etc/apache2/apache2.conf` or via HestiaCP):

```apache
<Location /server-status>
    SetHandler server-status
    Require local
</Location>
ExtendedStatus On
```

```bash
sudo a2enmod status
sudo systemctl reload apache2
# Verify:
curl http://127.0.0.1:8080/server-status?auto
```

### Enable Nginx stub_status (HestiaCP)

Add to your Nginx server block or `/etc/nginx/conf.d/status.conf`:

```nginx
server {
    listen 127.0.0.1:80;
    location /nginx_status {
        stub_status on;
        allow 127.0.0.1;
        deny all;
    }
}
```

```bash
sudo nginx -t && sudo systemctl reload nginx
# Verify:
curl http://127.0.0.1/nginx_status
```

---

## 📊 Dashboard Overview

| Section | Description |
|---------|-------------|
| **Topbar** | Load average, live indicator, critical count |
| **Sidebar** | System status (Nginx/Apache), VHost list with mini-bars |
| **Summary button** | Opens full modal with CPU chart + summary table |
| **Main panel** | Selected VHost detail: stats chips, bar chart, doughnut chart, worker table |

---

## 📁 Project Structure

```
.
├── main.go        # Go source (single file)
├── index.php      # PHP single-file edition
├── build.sh       # Cross-platform release builder
├── install.sh     # Universal one-line installer
├── start.sh       # Quick start / restart helper
├── go.mod         # Go module
├── servo          # Compiled binary (after build)
└── dist/          # Release binaries (after build.sh)
    ├── servo_*_linux_amd64.tar.gz
    ├── servo_*_windows_amd64.zip
    ├── checksums.txt
    └── ...
```

---

## 📜 Logs

```bash
# Go binary (background)
tail -f servo.log

# Systemd
journalctl -u servo -f

# macOS LaunchAgent
tail -f /tmp/servo.log
```

---

*Built with ❤️ for server administrators running HestiaCP and Linux hosting.*
