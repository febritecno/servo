# SERVO - Server Monitoring Dashboard

SERVO is a lightweight, high-performance server monitoring dashboard built with Go. it provides real-time visibility into your Apache and Nginx web servers, specifically designed to work seamlessly with HestiaCP and other Linux-based hosting environments.

![SERVO Dashboard](https://via.placeholder.com/800x450?text=SERVO+Dashboard+UI) *(Note: Replace with actual screenshot if available)*

## ✨ Features

- **Single Binary**: No complex dependencies, just one file to rule them all.
- **Auto-Discovery**: Automatically detects Apache (`server-status`) and Nginx (`nginx_status`) endpoints on common ports (8080/8081).
- **VHost Intelligence**: Breaks down performance metrics (CPU, Workers, Avg Request, Unique IPs) per Virtual Host.
- **CPU Allocation Chart**: Beautiful doughnut visualization of system-wide CPU usage distribution across all domains.
- **Critical Status Alerts**: Dynamic badge in the topbar that updates if any service or VHost goes down.
- **Responsive UI**: Premium dark-mode dashboard built with Alpine.js and Tailwind-inspired styling.
- **Zero Configuration**: Just run it, and it finds your stats.

## 🚀 Quick Start

### 1. Prerequisites
- Go 1.20 or later (for building)
- Web Browser

### 2. Manual Run
```bash
# Build the project
go build -o servo .

# Run directly (Auto-Detect mode)
./servo --port :8080
```

### 3. Using Start Script (Recommended)
We provide a `start.sh` script that handles building, clearing old processes, and running in the background.

```bash
chmod +x start.sh
./start.sh
```

## 🛠️ Command Line Options

| Flag | Description | Default |
|------|-------------|---------|
| `--apache` | Manually specify Apache `server-status` URL | *Auto-detected* |
| `--nginx` | Manually specify Nginx `nginx_status` URL | *Auto-detected* |
| `--port` | The port SERVO will listen on | `:8080` |
| `--dummy` | Run with sample data for testing purposes | `false` |

## 📊 Monitoring View
- **Sidebar**: Overview of active workers for Nginx and Apache.
- **VHost List**: Real-time stats for every active domain.
- **Summary Modal**: Press the **Summary** button on the sidebar to see the global CPU Allocation chart and a consolidated performance table.

## 📜 Log Management
When using `./start.sh`, logs are written to `servo.log`. You can monitor them in real-time:
```bash
tail -f servo.log
```

## 🤝 Contributing
Feel free to fork this project and submit pull requests for any features or bug fixes.

---
Created with ❤️ for server administrators.
