package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	apacheStatusURL = ""
	nginxStatusURL  = ""
	port            = ":8080"
	dummyMode       = false
)

type DashboardData struct {
	LoadAvg       string       `json:"load_avg"`
	Nginx         NginxStat    `json:"nginx"`
	ApacheSummary ApacheSum    `json:"apache_summary"`
	Groups        []VHostGroup `json:"groups"`
}

type NginxStat struct {
	Active   string `json:"active"`
	Accepts  string `json:"accepts"`
	Handled  string `json:"handled"`
	Requests string `json:"requests"`
	Reading  string `json:"reading"`
	Writing  string `json:"writing"`
	Waiting  string `json:"waiting"`
	Status   string `json:"status"`
}

type ApacheSum struct {
	TotalAccesses string `json:"total_accesses"`
	TotalTraffic  string `json:"total_traffic"`
	Uptime        string `json:"uptime"`
	BusyWorkers   string `json:"busy_workers"`
	IdleWorkers   string `json:"idle_workers"`
	ReqPerSec     string `json:"req_per_sec"`
	BytesPerSec   string `json:"bytes_per_sec"`
	Status        string `json:"status"`
}

type Worker struct {
	Srv      string `json:"srv"`
	PID      string `json:"pid"`
	Acc      string `json:"acc"`
	Mode     string `json:"mode"`
	CPU      string `json:"cpu"`
	SS       string `json:"ss"`
	Req      string `json:"req"`
	Dur      string `json:"dur"`
	Conn     string `json:"conn"`
	Child    string `json:"child"`
	Slot     string `json:"slot"`
	ClientIP string `json:"ip_address"`
	Protocol string `json:"protocol"`
	VHost    string `json:"vhost"`
	URLReq   string `json:"url_request"`
}

type VHostGroup struct {
	VHost       string   `json:"vhost"`
	TotalWorker int      `json:"total_worker"`
	Workers     []Worker `json:"workers"`
}

func main() {
	flag.StringVar(&apacheStatusURL, "apache", "", "URL Apache server-status (kosongkan untuk auto-detect)")
	flag.StringVar(&nginxStatusURL, "nginx", "", "URL Nginx status (kosongkan untuk auto-detect)")
	flag.StringVar(&port, "port", port, "Port Web UI")
	flag.BoolVar(&dummyMode, "dummy", false, "Gunakan data dummy")
	flag.Parse()

	if !dummyMode {
		discoverURLs()
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(htmlTemplate))
	})

	http.HandleFunc("/api/data", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var data DashboardData
		if dummyMode {
			data = getDummyData()
		} else {
			data = DashboardData{
				LoadAvg:       getLoadAvg(),
				Nginx:         fetchNginx(),
				ApacheSummary: fetchApacheAuto(),
				Groups:        fetchApacheWorkers(),
			}
		}
		json.NewEncoder(w).Encode(data)
	})

	mode := "production"
	if dummyMode {
		mode = "DUMMY"
	}
	log.Printf("Monitor running at http://127.0.0.1%s [%s]", port, mode)
	log.Fatal(http.ListenAndServe(port, nil))
}

func discoverURLs() {
	client := &http.Client{Timeout: 1 * time.Second}

	if apacheStatusURL == "" {
		candidates := []string{
			"http://127.0.0.1:8080/server-status",
			"http://127.0.0.1:8081/server-status",
			"http://localhost:8080/server-status",
			"http://localhost:8081/server-status",
			"http://127.0.0.1/server-status",
		}
		for _, url := range candidates {
			resp, err := client.Get(url + "?auto")
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == 200 {
					apacheStatusURL = url
					log.Printf("Auto-discovered Apache status at: %s", url)
					break
				}
			}
		}
		if apacheStatusURL == "" {
			apacheStatusURL = "http://127.0.0.1:8080/server-status" // fallback default
		}
	}

	if nginxStatusURL == "" {
		candidates := []string{
			"http://127.0.0.1/nginx_status",
			"http://localhost/nginx_status",
			"http://127.0.0.1:8081/nginx_status",
			"http://127.0.0.1:8080/nginx_status",
		}
		for _, url := range candidates {
			resp, err := client.Get(url)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == 200 {
					nginxStatusURL = url
					log.Printf("Auto-discovered Nginx status at: %s", url)
					break
				}
			}
		}
		if nginxStatusURL == "" {
			nginxStatusURL = "http://127.0.0.1/nginx_status" // fallback default
		}
	}
}

func getDummyData() DashboardData {
	w := func(srv, pid, acc, mode, cpu, ss, req, dur, conn, child, slot, ip, proto, vhost, url string) Worker {
		return Worker{Srv: srv, PID: pid, Acc: acc, Mode: mode, CPU: cpu, SS: ss, Req: req, Dur: dur, Conn: conn, Child: child, Slot: slot, ClientIP: ip, Protocol: proto, VHost: vhost, URLReq: url}
	}
	return DashboardData{
		LoadAvg: "2.34 2.18 1.95",
		Nginx: NginxStat{Status: "OK", Active: "312", Accepts: "4821903", Handled: "4821903", Requests: "18234021", Reading: "5", Writing: "38", Waiting: "269"},
		ApacheSummary: ApacheSum{Status: "OK", TotalAccesses: "18,432,771", TotalTraffic: "34,218.4 MB", Uptime: "412h 38m", BusyWorkers: "51", IdleWorkers: "199", ReqPerSec: "48.23", BytesPerSec: "91834"},
		Groups: []VHostGroup{
			{VHost: "apiv1.indolen.com:8443", TotalWorker: 14, Workers: []Worker{
				w("0-40", "1988291", "0/94/1871", "_", "38.27", "476", "4008", "5472175", "0.0", "0.54", "23.96", "114.10.29.42", "http/1.1", "apiv1.indolen.com:8443", "POST /api/driver/login HTTP/1.0"),
				w("0-41", "1988292", "0/97/1931", "W", "38.47", "0", "1649", "5420451", "0.0", "0.57", "25.69", "114.10.29.42", "http/1.1", "apiv1.indolen.com:8443", "POST /api/driver/login HTTP/1.0"),
				w("0-42", "1988293", "0/96/1938", "_", "38.47", "454", "4702", "5238838", "0.0", "1.63", "22.20", "114.10.29.42", "http/1.1", "apiv1.indolen.com:8443", "POST /api/driver/login HTTP/1.0"),
				w("0-43", "1988294", "0/93/1915", "W", "38.36", "0", "518", "5299272", "0.0", "0.62", "27.75", "114.10.29.42", "http/1.1", "apiv1.indolen.com:8443", "POST /api/driver/save_queue HTTP/1.0"),
				w("0-44", "1988295", "0/85/1851", "_", "38.47", "457", "3399", "5154374", "0.0", "1.71", "18.21", "114.10.29.42", "http/1.1", "apiv1.indolen.com:8443", "GET /api/driver/getBandara HTTP/1.0"),
				w("1-40", "1992257", "0/82/1449", "W", "41.01", "0", "2701", "4094963", "0.0", "0.85", "16.33", "114.10.29.42", "http/1.1", "apiv1.indolen.com:8443", "POST /api/driver/getOrderByGeofence HTTP/1.0"),
				w("1-41", "1992258", "0/88/1431", "_", "41.02", "453", "4999", "4260952", "0.0", "3.45", "17.73", "114.10.29.42", "http/1.1", "apiv1.indolen.com:8443", "POST /api/driver/login HTTP/1.0"),
				w("1-42", "1992259", "0/65/1425", "_", "40.72", "456", "2", "4146695", "0.0", "1.22", "18.90", "140.213.13.163", "http/1.1", "apiv1.indolen.com:8443", "POST /api/driver/login HTTP/1.0"),
				w("1-43", "1992260", "0/65/1423", "W", "41.01", "0", "2601", "4026738", "0.0", "0.60", "13.46", "140.213.13.163", "http/1.1", "apiv1.indolen.com:8443", "POST /api/driver/login HTTP/1.0"),
				w("1-44", "1992261", "0/84/1429", "_", "40.99", "457", "4806", "4418036", "0.0", "3.85", "18.69", "140.213.13.163", "http/1.1", "apiv1.indolen.com:8443", "POST /api/driver/login HTTP/1.0"),
				w("2-40", "1988290", "0/76/895", "W", "36.09", "0", "591", "3089505", "0.0", "0.53", "9.28", "140.213.13.163", "http/1.1", "apiv1.indolen.com:8443", "POST /api/driver/save_queue HTTP/1.0"),
				w("2-41", "1988291", "0/77/836", "_", "36.09", "1296", "793", "2503558", "0.0", "0.49", "8.57", "140.213.13.163", "http/1.1", "apiv1.indolen.com:8443", "POST /api/driver/login HTTP/1.0"),
				w("2-42", "1988292", "0/88/872", "_", "36.10", "1278", "335", "2639789", "0.0", "0.65", "10.54", "140.213.13.163", "http/1.1", "apiv1.indolen.com:8443", "GET /api/driver/get_last_queue HTTP/1.0"),
				w("2-43", "1988293", "0/83/847", "W", "36.11", "0", "4905", "2740170", "0.0", "1.59", "9.67", "114.10.29.42", "http/1.1", "apiv1.indolen.com:8443", "POST /api/driver/login HTTP/1.0"),
			}},
			{VHost: "cloud.indolen.com:8443", TotalWorker: 10, Workers: []Worker{
				w("1-45", "1992262", "0/73/1390", "_", "40.61", "455", "1055", "3988245", "0.0", "1.75", "16.42", "182.2.184.134", "http/1.1", "cloud.indolen.com:8443", "GET /api/v1/driver/get-last-queue HTTP/1.0"),
				w("1-46", "1992263", "0/80/1411", "_", "40.61", "453", "0", "4059924", "0.0", "2.71", "17.64", "182.2.184.134", "http/1.1", "cloud.indolen.com:8443", "GET /assets/images/logobbm1.png HTTP/1.0"),
				w("1-47", "1992264", "0/79/1407", "W", "40.61", "0", "3808", "3891149", "0.0", "0.62", "15.56", "182.2.184.134", "http/1.1", "cloud.indolen.com:8443", "GET /api/v1/user HTTP/1.0"),
				w("2-45", "1988295", "0/81/901", "_", "35.71", "1358", "1834", "2846124", "0.0", "1.21", "11.18", "182.2.184.134", "http/1.1", "cloud.indolen.com:8443", "POST /api/v1/driver/save-queue HTTP/1.0"),
				w("2-46", "1988296", "0/82/863", "W", "35.70", "0", "2866", "2647147", "0.0", "2.38", "11.38", "182.2.184.134", "http/1.1", "cloud.indolen.com:8443", "GET /api/v1/countries-new HTTP/1.0"),
				w("3-10", "0", "0/0/813", ".", "0.00", "17713", "0", "2523213", "0.0", "0.00", "13.36", "182.3.43.114", "http/1.1", "cloud.indolen.com:8443", "GET /api/v1/user HTTP/1.0"),
				w("3-11", "0", "0/0/822", ".", "0.00", "30095", "0", "2550077", "0.0", "0.00", "11.57", "182.3.43.114", "http/1.1", "cloud.indolen.com:8443", "GET /api/v1/countries-new HTTP/1.0"),
				w("3-12", "0", "0/0/804", ".", "0.00", "10901", "0", "2323800", "0.0", "0.00", "10.51", "182.3.43.114", "http/1.1", "cloud.indolen.com:8443", "GET /api/v1/countries-new HTTP/1.0"),
				w("3-13", "0", "0/0/794", ".", "0.00", "3807", "0", "2446693", "0.0", "0.00", "8.12", "182.3.43.114", "http/1.1", "cloud.indolen.com:8443", "POST /api/v1/driver/getPresenceStatus HTTP/1.0"),
				w("5-5", "0", "0/0/20", ".", "0.00", "16598", "0", "71159", "0.0", "0.00", "0.15", "182.2.191.70", "http/1.1", "cloud.indolen.com:8443", "POST /api/v1/driver/getPresenceStatus HTTP/1.0"),
			}},
			{VHost: "taxiairport.id:8443", TotalWorker: 3, Workers: []Worker{
				w("0-45", "1988296", "0/86/1912", "W", "37.94", "0", "2402", "5198766", "0.0", "2.75", "26.15", "140.213.6.243", "http/1.1", "taxiairport.id:8443", "GET /api/v1/countries-new HTTP/1.0"),
				w("0-46", "1988297", "0/90/1884", "_", "38.27", "453", "4593", "5609817", "0.0", "1.59", "25.86", "140.213.6.243", "http/1.1", "taxiairport.id:8443", "GET /api/v1/getBandara HTTP/1.0"),
				w("0-47", "1988298", "0/95/1914", "_", "38.36", "475", "4703", "5202611", "0.0", "2.21", "24.13", "140.213.6.243", "http/1.1", "taxiairport.id:8443", "GET /api/v1/get_last_queue HTTP/1.0"),
			}},
			{VHost: "smart.indolen.com:8443", TotalWorker: 2, Workers: []Worker{
				w("0-48", "1988299", "0/88/1875", "_", "37.33", "454", "276", "5291819", "0.0", "0.60", "24.12", "114.5.240.152", "http/1.1", "smart.indolen.com:8443", "GET /dashboard/detail/314170 HTTP/1.0"),
				w("0-49", "1988300", "0/80/1899", "W", "38.27", "0", "4295", "5266889", "0.0", "2.91", "20.65", "114.5.240.152", "http/1.1", "smart.indolen.com:8443", "GET /dashboard HTTP/1.0"),
			}},
			{VHost: "srv478774.hstgr.cloud:8081", TotalWorker: 5, Workers: []Worker{
				w("1-27", "1992257", "1/72/1433", "W", "41.00", "0", "0", "4232800", "0.0", "2.96", "19.29", "127.0.0.1", "http/1.1", "srv478774.hstgr.cloud:8081", "GET /server-status/ HTTP/1.1"),
				w("1-38", "1992257", "0/70/1389", "_", "40.62", "462", "1", "3819464", "0.0", "1.54", "16.41", "127.0.0.1", "http/1.1", "srv478774.hstgr.cloud:8081", "GET /server-status HTTP/1.1"),
				w("1-39", "1992257", "0/76/1385", "_", "40.72", "458", "0", "3964029", "0.0", "1.41", "13.74", "127.0.0.1", "http/1.1", "srv478774.hstgr.cloud:8081", "GET /server-status HTTP/1.1"),
				w("2-38", "1988290", "0/88/830", "_", "35.71", "1345", "0", "2736973", "0.0", "2.73", "10.48", "127.0.0.1", "http/1.1", "srv478774.hstgr.cloud:8081", "GET /server-status HTTP/1.1"),
				w("3-18", "0", "0/0/770", ".", "0.00", "0", "0", "2550605", "0.0", "0.00", "10.33", "127.0.0.1", "http/1.1", "srv478774.hstgr.cloud:8081", "GET /server-status HTTP/1.1"),
			}},
		},
	}
}

func httpClient() *http.Client {
	return &http.Client{Timeout: 3 * time.Second}
}

func getLoadAvg() string {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return "N/A"
	}
	parts := strings.Fields(string(data))
	if len(parts) >= 3 {
		return fmt.Sprintf("%s %s %s", parts[0], parts[1], parts[2])
	}
	return "N/A"
}

func fetchNginx() NginxStat {
	resp, err := httpClient().Get(nginxStatusURL)
	if err != nil {
		return NginxStat{Status: "Offline"}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	text := string(body)
	stat := NginxStat{Status: "OK"}
	if m := regexp.MustCompile(`Active connections:\s+(\d+)`).FindStringSubmatch(text); len(m) > 1 {
		stat.Active = m[1]
	}
	if m := regexp.MustCompile(`\s+(\d+)\s+(\d+)\s+(\d+)`).FindStringSubmatch(text); len(m) > 3 {
		stat.Accepts, stat.Handled, stat.Requests = m[1], m[2], m[3]
	}
	if m := regexp.MustCompile(`Reading:\s+(\d+)\s+Writing:\s+(\d+)\s+Waiting:\s+(\d+)`).FindStringSubmatch(text); len(m) > 3 {
		stat.Reading, stat.Writing, stat.Waiting = m[1], m[2], m[3]
	}
	return stat
}

func fetchApacheAuto() ApacheSum {
	resp, err := httpClient().Get(apacheStatusURL + "?auto")
	if err != nil {
		return ApacheSum{Status: "Offline"}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	stat := ApacheSum{Status: "OK"}
	var totalKB float64
	for _, line := range strings.Split(string(body), "\n") {
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) != 2 {
			continue
		}
		key, val := parts[0], strings.TrimSpace(parts[1])
		switch key {
		case "Total Accesses":
			stat.TotalAccesses = val
		case "Total kBytes":
			kb, _ := strconv.ParseFloat(val, 64)
			totalKB = kb
			stat.TotalTraffic = fmt.Sprintf("%.1f MB", kb/1024)
		case "Uptime":
			upSec, _ := strconv.Atoi(val)
			stat.Uptime = fmt.Sprintf("%dh %dm", upSec/3600, (upSec%3600)/60)
		case "BusyWorkers":
			stat.BusyWorkers = val
		case "IdleWorkers":
			stat.IdleWorkers = val
		case "ReqPerSec":
			stat.ReqPerSec = val
		case "BytesPerSec":
			stat.BytesPerSec = val
		}
	}
	if stat.TotalTraffic == "" {
		stat.TotalTraffic = fmt.Sprintf("%.1f MB", totalKB/1024)
	}
	return stat
}

func fetchApacheWorkers() []VHostGroup {
	resp, err := httpClient().Get(apacheStatusURL)
	if err != nil {
		return []VHostGroup{}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	rowRe := regexp.MustCompile(`(?is)<tr[^>]*>(.*?)</tr>`)
	colRe := regexp.MustCompile(`(?is)<td[^>]*>(.*?)</td>`)
	tagRe := regexp.MustCompile(`<[^>]*>`)
	clean := func(s string) string {
		return strings.TrimSpace(tagRe.ReplaceAllString(s, ""))
	}

	var workers []Worker
	for _, row := range rowRe.FindAllStringSubmatch(html, -1) {
		cols := colRe.FindAllStringSubmatch(row[1], -1)
		// Support both 13-col (old) and 15-col (new, with Acc+CPU) formats
		var vhost, urlFull, clientIP, proto, srv, pid, acc, mode, cpu, ss, req, dur, conn, child, slot string
		if len(cols) >= 15 {
			srv = clean(cols[0][1])
			pid = clean(cols[1][1])
			acc = clean(cols[2][1])
			mode = clean(cols[3][1])
			cpu = clean(cols[4][1])
			ss = clean(cols[5][1])
			req = clean(cols[6][1])
			dur = clean(cols[7][1])
			conn = clean(cols[8][1])
			child = clean(cols[9][1])
			slot = clean(cols[10][1])
			clientIP = clean(cols[11][1])
			proto = clean(cols[12][1])
			vhost = clean(cols[13][1])
			urlFull = clean(cols[14][1])
		} else if len(cols) >= 13 {
			srv = clean(cols[0][1])
			pid = clean(cols[1][1])
			mode = clean(cols[3][1])
			cpu = clean(cols[4][1])
			dur = clean(cols[6][1])
			slot = clean(cols[9][1])
			clientIP = clean(cols[10][1])
			vhost = clean(cols[11][1])
			urlFull = clean(cols[12][1])
		} else {
			continue
		}
		if vhost == "" || vhost == "VHost" {
			continue
		}
		workers = append(workers, Worker{
			Srv: srv, PID: pid, Acc: acc, Mode: mode, CPU: cpu,
			SS: ss, Req: req, Dur: dur, Conn: conn, Child: child, Slot: slot,
			ClientIP: clientIP, Protocol: proto, VHost: vhost, URLReq: urlFull,
		})
	}

	groupMap := make(map[string][]Worker)
	for _, w := range workers {
		groupMap[w.VHost] = append(groupMap[w.VHost], w)
	}
	var groups []VHostGroup
	for vhost, wList := range groupMap {
		groups = append(groups, VHostGroup{VHost: vhost, TotalWorker: len(wList), Workers: wList})
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].TotalWorker > groups[j].TotalWorker
	})
	return groups
}

var htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>SERVO - Server Monitor</title>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@400;600&display=swap" rel="stylesheet">
<script src="https://cdn.jsdelivr.net/npm/chart.js@4/dist/chart.umd.min.js"></script>
<style>
*{box-sizing:border-box;margin:0;padding:0}
:root{
  --bg0:#0d1117;--bg1:#161b22;--bg2:#1c2128;--bg3:#21262d;
  --border:#30363d;--t1:#e6edf3;--t2:#8b949e;--t3:#484f58;
  --green:#3fb950;--yellow:#d29922;--red:#f85149;--blue:#58a6ff;--purple:#bc8cff;
}
html,body{height:100%;overflow:hidden}
body{background:var(--bg0);color:var(--t1);font-family:'Inter',sans-serif;display:flex;flex-direction:column}
.topbar{height:52px;background:var(--bg1);border-bottom:1px solid var(--border);display:flex;align-items:center;justify-content:space-between;padding:0 1.25rem;flex-shrink:0}
.topbar-logo{font-size:1rem;font-weight:700;display:flex;align-items:center;gap:.5rem}
.topbar-logo span{color:var(--blue)}
.topbar-right{display:flex;align-items:center;gap:1rem;font-size:.75rem;color:var(--t2)}
.load-val{color:#ff79c6;font-weight:600;font-family:'JetBrains Mono',monospace}
.pulse{width:8px;height:8px;border-radius:50%;background:var(--green);animation:pulse 2s infinite}
@keyframes pulse{0%,100%{opacity:1;box-shadow:0 0 0 0 rgba(63,185,80,.4)}50%{opacity:.7;box-shadow:0 0 0 5px rgba(63,185,80,0)}}
.layout{display:flex;flex:1;overflow:hidden}
.sidebar{width:268px;background:var(--bg1);border-right:1px solid var(--border);display:flex;flex-direction:column;flex-shrink:0;overflow-y:auto}
.sidebar::-webkit-scrollbar{width:4px}
.sidebar::-webkit-scrollbar-thumb{background:var(--border);border-radius:2px}
.sb-section{padding:.875rem 1rem}
.sb-label{font-size:.625rem;font-weight:600;color:var(--t3);text-transform:uppercase;letter-spacing:.1em;padding:.5rem 0}
.sb-sys-card{background:var(--bg2);border:1px solid var(--border);border-radius:.5rem;padding:.625rem .75rem;display:flex;align-items:center;justify-content:space-between;margin-bottom:.375rem}
.sb-sys-name{font-size:.8125rem;font-weight:500;display:flex;align-items:center;gap:.5rem}
.sdot{width:8px;height:8px;border-radius:50%;flex-shrink:0}
.badge{font-size:.625rem;font-weight:700;padding:2px 8px;border-radius:999px}
.b-healthy{background:rgba(63,185,80,.15);color:var(--green)}
.b-warning{background:rgba(210,153,34,.15);color:var(--yellow)}
.b-critical{background:rgba(248,81,73,.15);color:var(--red)}
.b-down{background:rgba(72,79,88,.2);color:var(--t2)}
.b-offline{background:rgba(248,81,73,.15);color:var(--red)}
.vhost-item{padding:.625rem .75rem;border-radius:.5rem;cursor:pointer;transition:background .1s;margin-bottom:2px;border:1px solid transparent}
.vhost-item:hover{background:var(--bg2)}
.vhost-item.active{background:var(--bg2);border-color:var(--border)}
.vi-header{display:flex;align-items:center;gap:.5rem;margin-bottom:.375rem}
.vi-name{font-size:.75rem;font-weight:500;flex:1;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.vi-meta{font-size:.6875rem;color:var(--t2);display:flex;gap:.75rem}
.vi-bar{height:2px;background:var(--border);border-radius:1px;overflow:hidden;margin-top:.375rem}
.vi-bar-fill{height:100%;border-radius:1px;transition:width .5s ease}
.content{flex:1;overflow-y:auto;display:flex;flex-direction:column}
.content::-webkit-scrollbar{width:6px}
.content::-webkit-scrollbar-thumb{background:var(--border);border-radius:3px}
.overview{padding:1.5rem 1.75rem}
.ov-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(240px,1fr));gap:1rem}
.ov-card{background:var(--bg1);border:1px solid var(--border);border-radius:.75rem;padding:1.125rem;cursor:pointer;transition:all .15s;position:relative;overflow:hidden}
.ov-card::before{content:'';position:absolute;top:0;left:0;right:0;height:2px;background:linear-gradient(90deg,#58a6ff,#bc8cff);opacity:0;transition:opacity .2s}
.ov-card:hover{border-color:#388bfd;transform:translateY(-1px);box-shadow:0 4px 20px rgba(0,0,0,.4)}
.ov-card:hover::before{opacity:1}
.ov-title{font-size:.8125rem;font-weight:600;margin-bottom:.875rem;word-break:break-all;line-height:1.35}
.ov-stats{display:flex;gap:.875rem;flex-wrap:wrap;font-size:.75rem;color:var(--t2)}
.det-header{padding:1.25rem 1.75rem 1rem;background:var(--bg1);border-bottom:1px solid var(--border);flex-shrink:0}
.det-title{font-size:1.125rem;font-weight:700;display:flex;align-items:center;gap:.75rem;margin-bottom:1rem;word-break:break-all;line-height:1.3;flex-wrap:wrap}
.stats-row{display:flex;gap:.625rem;flex-wrap:wrap}
.stat-chip{background:var(--bg2);border:1px solid var(--border);border-radius:.5rem;padding:.5rem .875rem;min-width:90px}
.stat-lbl{font-size:.625rem;color:var(--t2);text-transform:uppercase;letter-spacing:.05em;margin-bottom:.25rem}
.stat-val{font-size:1.125rem;font-weight:700;font-family:'JetBrains Mono',monospace}
.charts-row{display:grid;grid-template-columns:1fr 300px;gap:1rem;padding:1.25rem 1.75rem}
.chart-card{background:var(--bg1);border:1px solid var(--border);border-radius:.75rem;padding:1rem 1.125rem}
.chart-ttl{font-size:.6875rem;font-weight:600;color:var(--t2);text-transform:uppercase;letter-spacing:.06em;margin-bottom:.875rem}
.tbl-section{padding:0 1.75rem 1.75rem}
.tbl-card{background:var(--bg1);border:1px solid var(--border);border-radius:.75rem;overflow:hidden}
.tbl-head{padding:.75rem 1.125rem;border-bottom:1px solid var(--border);display:flex;justify-content:space-between;align-items:center}
.tbl-ttl{font-size:.6875rem;font-weight:600;color:var(--t2);text-transform:uppercase;letter-spacing:.06em}
.tbl-scroll{overflow-x:auto}
.tbl-scroll::-webkit-scrollbar{height:4px}
.tbl-scroll::-webkit-scrollbar-thumb{background:var(--border);border-radius:2px}
table{width:100%;border-collapse:collapse;font-size:.8rem;white-space:nowrap}
thead th{background:var(--bg2);color:var(--t3);font-weight:500;font-size:.625rem;text-transform:uppercase;letter-spacing:.05em;padding:.5rem .875rem;text-align:left;border-bottom:1px solid var(--border)}
tbody tr{border-bottom:1px solid rgba(48,54,61,.5);transition:background .1s}
tbody tr:hover{background:var(--bg2)}
tbody tr:last-child{border-bottom:none}
tbody td{padding:.4375rem .875rem}
.mode-badge{display:inline-flex;align-items:center;justify-content:center;width:22px;height:22px;border-radius:4px;font-weight:700;font-size:.75rem;font-family:'JetBrains Mono',monospace}
.m-W{background:rgba(210,153,34,.2);color:#d29922}
.m-_{background:rgba(88,166,255,.1);color:#58a6ff}
.m-dot{background:rgba(72,79,88,.3);color:#8b949e}
.m-K{background:rgba(188,140,255,.15);color:#bc8cff}
.ip-tag{font-family:'JetBrains Mono',monospace;font-size:.6875rem;background:rgba(88,166,255,.08);color:#58a6ff;padding:2px 7px;border-radius:4px}
.mono{font-family:'JetBrains Mono',monospace}
[x-cloak]{display:none!important}
.modal-overlay{position:fixed;top:0;left:0;right:0;bottom:0;z-index:9999;display:flex;align-items:center;justify-content:center;padding:1.5rem;background:rgba(0,0,0,.82);backdrop-filter:blur(6px)}
.modal-overlay[x-cloak]{display:none!important}
@keyframes modalIn{from{opacity:0;transform:scale(.96) translateY(8px)}to{opacity:1;transform:scale(1) translateY(0)}}
</style>
</head>
<body x-data="app()">

<!-- ===== SUMMARY MODAL ===== -->
<template x-teleport="body">
<div x-show="showSummary" x-cloak class="modal-overlay" @click.self="showSummary=false" @keydown.escape.window="showSummary=false">
  <div style="background:#0d1117;border:1px solid #30363d;border-radius:1rem;width:100%;max-width:960px;height:82vh;display:flex;flex-direction:column;box-shadow:0 32px 100px rgba(0,0,0,.85);animation:modalIn .22s ease;overflow:hidden" @click.stop>

    <!-- Modal Header -->
    <div style="padding:1.125rem 1.5rem;border-bottom:1px solid #21262d;display:flex;align-items:center;justify-content:space-between;flex-shrink:0;background:linear-gradient(135deg,#161b22 0%,#0d1117 100%);border-radius:1rem 1rem 0 0">
      <div style="display:flex;align-items:center;gap:.75rem">
        <div style="width:32px;height:32px;border-radius:8px;background:linear-gradient(135deg,#58a6ff,#bc8cff);display:flex;align-items:center;justify-content:center">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/></svg>
        </div>
        <div>
          <div style="font-size:1rem;font-weight:700;color:#e6edf3">System Summary</div>
          <div style="font-size:.6875rem;color:#8b949e" x-text="(data.groups||[]).length+' vhosts · '+(data.groups||[]).reduce(function(s,g){return s+g.total_worker;},0)+' total workers'"></div>
        </div>
      </div>
      <button @click="showSummary=false" style="background:rgba(139,148,158,.1);border:none;color:#8b949e;cursor:pointer;padding:6px;border-radius:8px;display:flex;align-items:center;transition:all .15s" onmouseover="this.style.background='rgba(139,148,158,.2)';this.style.color='#e6edf3'" onmouseout="this.style.background='rgba(139,148,158,.1)';this.style.color='#8b949e'">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
      </button>
    </div>
    <!-- Stats chips -->
    <div style="padding:.75rem 1.25rem;border-bottom:1px solid #21262d;display:flex;gap:.5rem;flex-wrap:wrap;flex-shrink:0;background:#0d1117">
      <div style="background:#161b22;border:1px solid #21262d;border-radius:.5rem;padding:.4rem .75rem;text-align:center">
        <div style="font-size:.55rem;color:#8b949e;text-transform:uppercase;letter-spacing:.06em">VHosts</div>
        <div style="font-size:1.1rem;font-weight:700;font-family:'JetBrains Mono',monospace" x-text="(data.groups||[]).length"></div>
      </div>
      <div style="background:#161b22;border:1px solid #21262d;border-radius:.5rem;padding:.4rem .75rem;text-align:center">
        <div style="font-size:.55rem;color:#8b949e;text-transform:uppercase;letter-spacing:.06em">Workers</div>
        <div style="font-size:1.1rem;font-weight:700;font-family:'JetBrains Mono',monospace" x-text="(data.groups||[]).reduce(function(s,g){return s+g.total_worker;},0)"></div>
      </div>
      <div style="background:#161b22;border:1px solid rgba(63,185,80,.3);border-radius:.5rem;padding:.4rem .75rem;text-align:center">
        <div style="font-size:.55rem;color:#3fb950;text-transform:uppercase;letter-spacing:.06em">Healthy</div>
        <div style="font-size:1.1rem;font-weight:700;font-family:'JetBrains Mono',monospace;color:#3fb950" x-text="(data.groups||[]).filter(function(g){return healthStatus(g)==='healthy';}).length"></div>
      </div>
      <div style="background:#161b22;border:1px solid rgba(210,153,34,.3);border-radius:.5rem;padding:.4rem .75rem;text-align:center">
        <div style="font-size:.55rem;color:#d29922;text-transform:uppercase;letter-spacing:.06em">Warning</div>
        <div style="font-size:1.1rem;font-weight:700;font-family:'JetBrains Mono',monospace;color:#d29922" x-text="(data.groups||[]).filter(function(g){return healthStatus(g)==='warning';}).length"></div>
      </div>
      <div style="background:#161b22;border:1px solid rgba(248,81,73,.3);border-radius:.5rem;padding:.4rem .75rem;text-align:center">
        <div style="font-size:.55rem;color:#f85149;text-transform:uppercase;letter-spacing:.06em">Critical</div>
        <div style="font-size:1.1rem;font-weight:700;font-family:'JetBrains Mono',monospace;color:#f85149" x-text="(data.groups||[]).filter(function(g){return healthStatus(g)==='critical'||healthStatus(g)==='down';}).length"></div>
      </div>
      <div style="background:#161b22;border:1px solid #21262d;border-radius:.5rem;padding:.4rem .75rem;text-align:center">
        <div style="font-size:.55rem;color:#8b949e;text-transform:uppercase;letter-spacing:.06em">Busy</div>
        <div style="font-size:1.1rem;font-weight:700;font-family:'JetBrains Mono',monospace;color:#d29922" x-text="data.apache_summary&&data.apache_summary.busy_workers||'—'"></div>
      </div>
      <div style="background:#161b22;border:1px solid #21262d;border-radius:.5rem;padding:.4rem .75rem;text-align:center">
        <div style="font-size:.55rem;color:#8b949e;text-transform:uppercase;letter-spacing:.06em">Req/s</div>
        <div style="font-size:1.1rem;font-weight:700;font-family:'JetBrains Mono',monospace;color:#58a6ff" x-text="data.apache_summary&&data.apache_summary.req_per_sec||'—'"></div>
      </div>
      <div style="background:#161b22;border:1px solid #21262d;border-radius:.5rem;padding:.4rem .75rem;text-align:center">
        <div style="font-size:.55rem;color:#8b949e;text-transform:uppercase;letter-spacing:.06em">Uptime</div>
        <div style="font-size:1.1rem;font-weight:700;font-family:'JetBrains Mono',monospace;color:#8b949e" x-text="data.apache_summary&&data.apache_summary.uptime||'—'"></div>
      </div>
    </div>

    <!-- 2-Column body: chart left | table right -->
    <div style="display:flex;flex:1;overflow:hidden;min-height:0">

      <!-- LEFT: CPU Chart -->
      <div style="width:240px;flex-shrink:0;border-right:1px solid #21262d;padding:1rem;display:flex;flex-direction:column;gap:.625rem;overflow-y:auto">
        <div style="font-size:.625rem;font-weight:600;color:#8b949e;text-transform:uppercase;letter-spacing:.06em">CPU Allocation</div>
        <div style="position:relative;width:100%;height:180px;flex-shrink:0">
          <canvas id="cpuAllocChart"></canvas>
        </div>
        <div style="display:flex;flex-direction:column;gap:.3rem">
          <template x-for="(g,i) in (data.groups||[])" :key="g.vhost">
            <div style="display:flex;align-items:center;gap:.5rem;cursor:pointer;padding:3px 4px;border-radius:4px;transition:background .1s" @click="showSummary=false;select(g)" onmouseover="this.style.background='#161b22'" onmouseout="this.style.background=''">
              <div style="width:8px;height:8px;border-radius:2px;flex-shrink:0" :style="'background:'+cpuChartColors()[i%cpuChartColors().length]"></div>
              <div style="flex:1;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;font-size:.6875rem;color:#8b949e" x-text="g.vhost.split(':')[0]"></div>
              <div style="font-size:.6875rem;font-family:'JetBrains Mono',monospace;color:#e6edf3;font-weight:600" x-text="(cpuPct(g)*100).toFixed(1)+'%'"></div>
            </div>
          </template>
        </div>
      </div>

      <!-- RIGHT: scrollable table -->
      <div style="flex:1;overflow-y:auto;overflow-x:auto;min-width:0">
        <table style="width:100%;border-collapse:collapse;font-size:.8rem;white-space:nowrap">
          <thead style="position:sticky;top:0;z-index:5">
            <tr>
              <th style="background:#161b22;color:#484f58;font-size:.575rem;font-weight:600;text-transform:uppercase;letter-spacing:.06em;padding:.5rem 1rem;text-align:left;border-bottom:1px solid #21262d">VHost</th>
              <th style="background:#161b22;color:#484f58;font-size:.575rem;font-weight:600;text-transform:uppercase;letter-spacing:.06em;padding:.5rem .75rem;text-align:left;border-bottom:1px solid #21262d">Status</th>
              <th style="background:#161b22;color:#484f58;font-size:.575rem;font-weight:600;text-transform:uppercase;letter-spacing:.06em;padding:.5rem .75rem;text-align:center;border-bottom:1px solid #21262d">Wkr</th>
              <th style="background:#161b22;color:#484f58;font-size:.575rem;font-weight:600;text-transform:uppercase;letter-spacing:.06em;padding:.5rem .75rem;text-align:center;border-bottom:1px solid #21262d">W</th>
              <th style="background:#161b22;color:#484f58;font-size:.575rem;font-weight:600;text-transform:uppercase;letter-spacing:.06em;padding:.5rem .75rem;text-align:center;border-bottom:1px solid #21262d">Idle</th>
              <th style="background:#161b22;color:#484f58;font-size:.575rem;font-weight:600;text-transform:uppercase;letter-spacing:.06em;padding:.5rem .75rem;text-align:right;border-bottom:1px solid #21262d">CPU</th>
              <th style="background:#161b22;color:#484f58;font-size:.575rem;font-weight:600;text-transform:uppercase;letter-spacing:.06em;padding:.5rem .75rem;text-align:right;border-bottom:1px solid #21262d">CPU%</th>
              <th style="background:#161b22;color:#484f58;font-size:.575rem;font-weight:600;text-transform:uppercase;letter-spacing:.06em;padding:.5rem .75rem;text-align:right;border-bottom:1px solid #21262d">Avg Req</th>
              <th style="background:#161b22;color:#484f58;font-size:.575rem;font-weight:600;text-transform:uppercase;letter-spacing:.06em;padding:.5rem .75rem;text-align:right;border-bottom:1px solid #21262d">IPs</th>
            </tr>
          </thead>
          <tbody>
            <template x-for="g in (data.groups||[])" :key="g.vhost">
              <tr style="border-bottom:1px solid rgba(33,38,45,.7);cursor:pointer;transition:background .1s" @click="showSummary=false;select(g)" onmouseover="this.style.background='#161b22'" onmouseout="this.style.background=''">
                <td style="padding:.5rem 1rem;font-weight:500">
                  <div style="display:flex;align-items:center;gap:.5rem">
                    <div class="sdot" :style="'background:'+healthColor(g)"></div>
                    <span x-text="g.vhost"></span>
                  </div>
                </td>
                <td style="padding:.5rem .75rem"><span class="badge" :class="'b-'+healthStatus(g)" x-text="healthLabel(g)"></span></td>
                <td style="padding:.5rem .75rem;text-align:center;font-family:'JetBrains Mono',monospace" x-text="g.total_worker"></td>
                <td style="padding:.5rem .75rem;text-align:center;font-family:'JetBrains Mono',monospace;color:#d29922" x-text="activeW(g)"></td>
                <td style="padding:.5rem .75rem;text-align:center;font-family:'JetBrains Mono',monospace;color:#58a6ff" x-text="idleWorkers(g)"></td>
                <td style="padding:.5rem .75rem;text-align:right;font-family:'JetBrains Mono',monospace;color:#f85149" x-text="totalCPU(g).toFixed(2)"></td>
                <td style="padding:.5rem .75rem;text-align:right">
                  <span style="font-size:.7rem;padding:1px 7px;border-radius:4px;font-weight:600;font-family:'JetBrains Mono',monospace"
                    :style="'background:'+(cpuPct(g)*100>50?'rgba(248,81,73,.15)':cpuPct(g)*100>20?'rgba(210,153,34,.15)':'rgba(63,185,80,.12)')+';color:'+(cpuPct(g)*100>50?'#f85149':cpuPct(g)*100>20?'#d29922':'#3fb950')"
                    x-text="(cpuPct(g)*100).toFixed(1)+'%'"></span>
                </td>
                <td style="padding:.5rem .75rem;text-align:right;font-family:'JetBrains Mono',monospace;color:#8b949e" x-text="avgReq(g)"></td>
                <td style="padding:.5rem .75rem;text-align:right;font-family:'JetBrains Mono',monospace;color:#58a6ff" x-text="uniqueIPs(g).length"></td>
              </tr>
            </template>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</div>
</template>
<div class="topbar">
  <div class="topbar-logo"><span>SERVO</span></div>
  <div class="topbar-right">
    <span x-show="downCount()>0" style="background:rgba(248,81,73,.12);color:#f85149;padding:3px 10px;border-radius:999px;font-weight:600;font-size:.7rem" x-text="downCount()+' Critical'"></span>
    <span>Load Avg: <span class="load-val" x-text="data.load_avg||'—'"></span></span>
    <div style="display:flex;align-items:center;gap:6px"><div class="pulse"></div><span>Live · 3s</span></div>
  </div>
</div>
<div class="layout">

<!-- SIDEBAR -->
<aside class="sidebar">
  <div class="sb-section">
    <div class="sb-label">System</div>
    <div class="sb-sys-card">
      <div class="sb-sys-name">
        <div class="sdot" :style="'background:'+(data.nginx&&data.nginx.status==='OK'?'var(--green)':'var(--red)')"></div>
        Nginx
      </div>
      <span class="badge" :class="data.nginx&&data.nginx.status==='OK'?'b-healthy':'b-offline'" x-text="data.nginx?data.nginx.status:'—'"></span>
    </div>
    <div class="sb-sys-card">
      <div class="sb-sys-name">
        <div class="sdot" :style="'background:'+(data.apache_summary&&data.apache_summary.status==='OK'?'var(--green)':'var(--red)')"></div>
        Apache
      </div>
      <span class="badge" :class="data.apache_summary&&data.apache_summary.status==='OK'?'b-healthy':'b-offline'" x-text="data.apache_summary?data.apache_summary.status:'—'"></span>
    </div>
    <div style="font-size:.6875rem;color:var(--t2);margin-top:.625rem;display:flex;flex-direction:column;gap:.3rem" x-show="data.apache_summary">
      <div>Busy: <strong style="color:var(--t1)" x-text="data.apache_summary&&data.apache_summary.busy_workers"></strong> &nbsp;Idle: <strong style="color:var(--t1)" x-text="data.apache_summary&&data.apache_summary.idle_workers"></strong></div>
      <div>Uptime: <span style="color:var(--t1)" x-text="data.apache_summary&&data.apache_summary.uptime"></span></div>
      <div>Req/s: <span style="color:var(--yellow)" x-text="data.apache_summary&&data.apache_summary.req_per_sec"></span></div>
    </div>
    <button @click="openSummary()" style="width:100%;margin-top:.625rem;padding:.475rem .75rem;background:rgba(88,166,255,.08);border:1px solid rgba(88,166,255,.2);border-radius:.5rem;color:var(--blue);font-size:.75rem;font-weight:500;cursor:pointer;display:flex;align-items:center;justify-content:center;gap:.4rem;transition:background .15s" onmouseover="this.style.background='rgba(88,166,255,.16)'" onmouseout="this.style.background='rgba(88,166,255,.08)'">
      <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/></svg>
      Summary
    </button>
  </div>
  <div class="sb-section" style="padding-top:0;flex:1">
    <div class="sb-label">VHosts (<span x-text="(data.groups||[]).length"></span>)</div>
    <template x-for="g in (data.groups||[])" :key="g.vhost">
      <div class="vhost-item" :class="selected&&selected.vhost===g.vhost?'active':''" @click="select(g)">
        <div class="vi-header">
          <div class="sdot" :style="'background:'+healthColor(g)"></div>
          <div class="vi-name" x-text="g.vhost"></div>
          <span class="badge" :class="'b-'+healthStatus(g)" x-text="healthLabel(g)"></span>
        </div>
        <div class="vi-meta">
          <span><strong style="color:var(--t1)" x-text="g.total_worker"></strong> wkr</span>
          <span>W:<strong style="color:var(--yellow)" x-text="activeW(g)"></strong></span>
          <span>CPU:<strong style="color:var(--t1);font-family:'JetBrains Mono',monospace" x-text="totalCPU(g).toFixed(1)"></strong></span>
        </div>
        <div class="vi-bar"><div class="vi-bar-fill" :style="'width:'+Math.min(activeW(g)/Math.max(g.total_worker,1)*100,100)+'%;background:'+healthColor(g)"></div></div>
      </div>
    </template>
    <div x-show="!(data.groups&&data.groups.length)" style="font-size:.75rem;color:var(--t3);padding:.5rem .25rem">No active vhosts</div>
  </div>
</aside>

<!-- MAIN -->
<main class="content">

  <!-- Detail -->
  <template x-if="selected">
    <div style="display:flex;flex-direction:column">
      <div class="det-header">
        <div class="det-title">
          <div class="sdot" :style="'background:'+healthColor(selected)"></div>
          <span x-text="selected.vhost"></span>
          <span class="badge" :class="'b-'+healthStatus(selected)" x-text="healthLabel(selected)"></span>
          <a :href="vhostURL(selected.vhost)" target="_blank" style="display:inline-flex;align-items:center;gap:4px;font-size:.75rem;font-weight:500;color:var(--blue);background:rgba(88,166,255,.1);padding:3px 10px;border-radius:6px;text-decoration:none;transition:background .15s" onmouseover="this.style.background='rgba(88,166,255,.2)'" onmouseout="this.style.background='rgba(88,166,255,.1)'">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>
            Visit
          </a>
        </div>
        <div class="stats-row">
          <div class="stat-chip"><div class="stat-lbl">Workers</div><div class="stat-val" x-text="selected.total_worker"></div></div>
          <div class="stat-chip"><div class="stat-lbl">Active (W)</div><div class="stat-val" style="color:var(--yellow)" x-text="activeW(selected)"></div></div>
          <div class="stat-chip"><div class="stat-lbl">Idle</div><div class="stat-val" style="color:var(--blue)" x-text="idleWorkers(selected)"></div></div>
          <div class="stat-chip"><div class="stat-lbl">Total CPU</div><div class="stat-val" x-text="totalCPU(selected).toFixed(2)"></div></div>
          <div class="stat-chip"><div class="stat-lbl">Unique IPs</div><div class="stat-val" x-text="uniqueIPs(selected).length"></div></div>
          <div class="stat-chip"><div class="stat-lbl">Avg Req (ms)</div><div class="stat-val" x-text="avgReq(selected)"></div></div>
        </div>
      </div>
      <div class="charts-row">
        <div class="chart-card">
          <div class="chart-ttl">Request Duration per Worker (ms)</div>
          <div style="position:relative;height:200px"><canvas id="reqChart"></canvas></div>
        </div>
        <div class="chart-card">
          <div class="chart-ttl">Worker Mode Distribution</div>
          <div style="position:relative;height:200px"><canvas id="modeChart"></canvas></div>
        </div>
      </div>
      <div class="tbl-section">
        <div class="tbl-card">
          <div class="tbl-head">
            <div class="tbl-ttl">Worker Details</div>
            <div style="font-size:.6875rem;color:var(--t2)" x-text="(selected.workers||[]).length+' rows'"></div>
          </div>
          <div class="tbl-scroll">
            <table>
              <thead><tr>
                <th>Srv</th><th>PID</th><th>Acc</th><th>M</th><th>CPU</th>
                <th>SS (s)</th><th>Req (ms)</th><th>Dur</th><th>Conn</th><th>Child</th><th>Slot</th>
                <th>IP Address</th><th>Protocol</th><th>Request URL</th>
              </tr></thead>
              <tbody>
                <template x-for="w in (selected.workers||[])" :key="w.srv+w.pid+w.url_request">
                  <tr>
                    <td class="mono" style="color:var(--t2);font-size:.75rem" x-text="w.srv"></td>
                    <td class="mono" style="font-size:.75rem" x-text="w.pid"></td>
                    <td class="mono" style="color:var(--t2);font-size:.75rem" x-text="w.acc||'—'"></td>
                    <td><span class="mode-badge" :class="w.mode==='W'?'m-W':w.mode==='_'?'m-_':w.mode==='K'?'m-K':'m-dot'" x-text="w.mode"></span></td>
                    <td class="mono" style="color:var(--red)" x-text="w.cpu||'—'"></td>
                    <td class="mono" style="color:var(--t2)" x-text="w.ss||'—'"></td>
                    <td class="mono" :style="'color:'+(parseFloat(w.req)>5000?'var(--red)':parseFloat(w.req)>1000?'var(--yellow)':'var(--green)')" x-text="w.req||'—'"></td>
                    <td class="mono" style="color:var(--t2);font-size:.75rem" x-text="w.dur||'—'"></td>
                    <td class="mono" style="color:var(--t2)" x-text="w.conn||'—'"></td>
                    <td class="mono" style="color:var(--t2)" x-text="w.child||'—'"></td>
                    <td class="mono" style="color:var(--t2)" x-text="w.slot||'—'"></td>
                    <td><span class="ip-tag" x-text="w.ip_address"></span></td>
                    <td style="color:var(--t2);font-size:.75rem" x-text="w.protocol"></td>
                    <td style="max-width:320px;overflow:hidden;text-overflow:ellipsis;font-size:.75rem" :title="w.url_request" x-text="w.url_request"></td>
                  </tr>
                </template>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  </template>
</main>
</div>

<script>
function app() {
  return {
    data: {}, selected: null, _rc: null, _mc: null, showSummary: false, _cpuAlloc: null,
    init() {
      this.loadData().then(function(){
        if (!this.selected && this.data.groups && this.data.groups.length) {
          this.select(this.data.groups[0]);
        }
      }.bind(this));
      setInterval(function(){ this.loadData(); }.bind(this), 3000);
    },
    async loadData() {
      try {
        var res = await fetch('/api/data');
        var fresh = await res.json();
        if (this.selected && fresh.groups) {
          var self = this;
          var m = fresh.groups.find(function(g){ return g.vhost === self.selected.vhost; });
          if (m) { this.selected = m; this.$nextTick(function(){ this.buildCharts(); }.bind(this)); }
        }
        this.data = fresh;
        if (this.showSummary) {
          this.$nextTick(function(){ this.buildCpuAllocChart(); }.bind(this));
        }
        return fresh;
      } catch(e) { console.error(e); }
    },
    select(g) { this.selected = g; this.$nextTick(function(){ this.buildCharts(); }.bind(this)); },
    downCount() {
      return (this.data.groups||[]).filter(function(g){ return this.healthStatus(g)==='critical'||this.healthStatus(g)==='down'; },this).length;
    },
    vhostURL(vhost) {
      if (!vhost) return '#';
      var parts = vhost.split(':');
      var port = parts[1]||'';
      var scheme = (port==='443'||port==='8443'||port==='4443') ? 'https' : 'http';
      return scheme+'://'+vhost;
    },
    cpuChartColors() {
      return ['#58a6ff','#bc8cff','#3fb950','#d29922','#f85149','#79c0ff','#d2a8ff','#56d364','#e3b341','#ff7b72','#a5d6ff','#efb8c8','#b3f0ff','#ffa28b'];
    },
    cpuPct(g) {
      var total = (this.data.groups||[]).reduce(function(s,x){ return s+this.totalCPU(x); }.bind(this), 0);
      return total > 0 ? this.totalCPU(g) / total : 0;
    },
    openSummary() {
      this.showSummary = true;
      setTimeout(function(){ this.buildCpuAllocChart(); }.bind(this), 100);
    },
    buildCpuAllocChart() {
      if (this._cpuAlloc) { this._cpuAlloc.destroy(); this._cpuAlloc = null; }
      var canvas = document.getElementById('cpuAllocChart');
      if (!canvas) return;
      var groups = this.data.groups || [];
      var self = this;
      var labels = groups.map(function(g){ return g.vhost.split(':')[0]; });
      var vals = groups.map(function(g){ return self.totalCPU(g); });
      var colors = this.cpuChartColors();
      this._cpuAlloc = new Chart(canvas, {
        type: 'doughnut',
        data: {
          labels: labels,
          datasets: [{ data: vals, backgroundColor: colors.slice(0, groups.length), borderWidth: 0, hoverOffset: 6 }]
        },
        options: {
          responsive: true, maintainAspectRatio: false, cutout: '68%', animation: false,
          plugins: {
            legend: { display: false },
            tooltip: { callbacks: {
              label: function(c) {
                var total = vals.reduce(function(s,v){ return s+v; }, 0);
                var pct = total > 0 ? (c.parsed/total*100).toFixed(1) : '0';
                return ' '+c.label+': '+c.parsed.toFixed(2)+' ('+pct+'%)';
              }
            }}
          }
        }
      });
    },

    healthStatus(g) {
      if (!g || !g.workers || !g.workers.length) return 'down';
      var wc = g.workers.filter(function(x){ return x.mode === 'W'; }).length;
      var r = wc / g.workers.length;
      return r > 0.75 ? 'critical' : r > 0.4 ? 'warning' : 'healthy';
    },
    healthLabel(g) { return {healthy:'Healthy',warning:'Warning',critical:'Critical',down:'Down'}[this.healthStatus(g)]||'Down'; },
    healthColor(g) { return {healthy:'#3fb950',warning:'#d29922',critical:'#f85149',down:'#8b949e'}[this.healthStatus(g)]||'#8b949e'; },
    totalCPU(g) { return (g&&g.workers||[]).reduce(function(s,w){ return s+parseFloat(w.cpu||0); },0); },
    activeW(g) { return (g&&g.workers||[]).filter(function(w){ return w.mode==='W'; }).length; },
    idleWorkers(g) { return (g&&g.workers||[]).filter(function(w){ return w.mode==='.'||w.mode==='_'; }).length; },
    uniqueIPs(g) {
      var ips=(g&&g.workers||[]).map(function(w){ return w.ip_address; }).filter(Boolean);
      return ips.filter(function(v,i,a){ return a.indexOf(v)===i; });
    },
    avgReq(g) {
      var ws=(g&&g.workers||[]).filter(function(w){ return w.req&&parseFloat(w.req)>0; });
      if(!ws.length) return '—';
      return (ws.reduce(function(s,w){ return s+parseFloat(w.req); },0)/ws.length).toFixed(0);
    },
    buildCharts() {
      if(this._rc){ this._rc.destroy(); this._rc=null; }
      if(this._mc){ this._mc.destroy(); this._mc=null; }
      if(!this.selected) return;
      var workers=this.selected.workers||[];
      var GRID='#21262d', TICK='#6e7681';
      var rc=document.getElementById('reqChart');
      if(rc){
        var vals=workers.map(function(w){ return parseFloat(w.req||0); });
        var colors=vals.map(function(v){ return v>5000?'#f85149':v>1000?'#d29922':'#3fb950'; });
        this._rc=new Chart(rc,{
          type:'bar',
          data:{labels:workers.map(function(w){ return w.srv; }),datasets:[{data:vals,backgroundColor:colors,borderRadius:2}]},
          options:{responsive:true,maintainAspectRatio:false,animation:false,
            plugins:{legend:{display:false},tooltip:{callbacks:{label:function(c){ return c.parsed.y+' ms'; }}}},
            scales:{x:{grid:{color:GRID},ticks:{color:TICK,font:{size:9},maxRotation:0}},y:{grid:{color:GRID},ticks:{color:TICK,font:{size:10},callback:function(v){ return v+'ms'; }}}}}
        });
      }
      var mc=document.getElementById('modeChart');
      if(mc){
        var modes={W:0,'_':0,'.':0,K:0,o:0};
        workers.forEach(function(w){ if(w.mode==='W')modes.W++;else if(w.mode==='_')modes['_']++;else if(w.mode==='.')modes['.']++;else if(w.mode==='K')modes.K++;else modes.o++; });
        this._mc=new Chart(mc,{
          type:'doughnut',
          data:{labels:['Writing (W)','Keepalive (_)','Idle (.)','Closing (K)','Other'],
            datasets:[{data:[modes.W,modes['_'],modes['.'],modes.K,modes.o],backgroundColor:['#d29922','#58a6ff','#374151','#bc8cff','#484f58'],borderWidth:0,hoverOffset:4}]},
          options:{responsive:true,maintainAspectRatio:false,animation:false,
            plugins:{legend:{position:'right',labels:{color:'#8b949e',boxWidth:10,padding:8,font:{size:11}}}}}
        });
      }
    }
  };
}
</script>
<script defer src="https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js"></script>
</body>
</html>`
