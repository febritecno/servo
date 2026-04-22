<?php
// ============================================================
// SERVO - Server Monitor Dashboard (PHP Single File Edition)
// Taruh file ini di /home/user/web/domain/public_html/index.php
// ============================================================

// ---- CONFIG AUTO DISCOVERY ----
function discoverApache() {
    $candidates = [
        'http://127.0.0.1:8080/server-status',
        'http://127.0.0.1:8081/server-status',
        'http://127.0.0.1/server-status',
    ];
    foreach ($candidates as $url) {
        $ctx = stream_context_create(['http'=>['timeout'=>2,'ignore_errors'=>true]]);
        $r = @file_get_contents($url.'?auto', false, $ctx);
        if ($r !== false && strpos($http_response_header[0]??'','200') !== false) return $url;
    }
    return 'http://127.0.0.1:8080/server-status';
}

function discoverNginx() {
    $candidates = [
        'http://127.0.0.1/nginx_status',
        'http://127.0.0.1:8081/nginx_status',
        'http://127.0.0.1:8080/nginx_status',
    ];
    foreach ($candidates as $url) {
        $ctx = stream_context_create(['http'=>['timeout'=>2,'ignore_errors'=>true]]);
        $r = @file_get_contents($url, false, $ctx);
        if ($r !== false && strpos($http_response_header[0]??'','200') !== false) return $url;
    }
    return 'http://127.0.0.1/nginx_status';
}

// ---- API ENDPOINT ----
if (isset($_GET['api'])) {
    header('Content-Type: application/json');
    $apacheURL = discoverApache();
    $nginxURL  = discoverNginx();
    echo json_encode([
        'load_avg'       => getLoadAvg(),
        'nginx'          => fetchNginx($nginxURL),
        'apache_summary' => fetchApacheSummary($apacheURL),
        'groups'         => fetchApacheWorkers($apacheURL),
    ]);
    exit;
}

function getLoadAvg() {
    if (file_exists('/proc/loadavg')) {
        $parts = explode(' ', trim(file_get_contents('/proc/loadavg')));
        return ($parts[0]??'N/A').' '.($parts[1]??'').' '.($parts[2]??'');
    }
    return 'N/A';
}

function httpGet($url, $timeout=3) {
    $ctx = stream_context_create(['http'=>['timeout'=>$timeout,'ignore_errors'=>true]]);
    $body = @file_get_contents($url, false, $ctx);
    return $body === false ? null : $body;
}

function fetchNginx($url) {
    $body = httpGet($url);
    if ($body === null) return ['status'=>'Offline','active'=>'','accepts'=>'','handled'=>'','requests'=>'','reading'=>'','writing'=>'','waiting'=>''];
    $stat = ['status'=>'OK'];
    if (preg_match('/Active connections:\s+(\d+)/', $body, $m)) $stat['active'] = $m[1];
    if (preg_match('/\s+(\d+)\s+(\d+)\s+(\d+)/', $body, $m)) {
        $stat['accepts']=$m[1]; $stat['handled']=$m[2]; $stat['requests']=$m[3];
    }
    if (preg_match('/Reading:\s+(\d+)\s+Writing:\s+(\d+)\s+Waiting:\s+(\d+)/', $body, $m)) {
        $stat['reading']=$m[1]; $stat['writing']=$m[2]; $stat['waiting']=$m[3];
    }
    return $stat;
}

function fetchApacheSummary($url) {
    $body = httpGet($url.'?auto');
    if ($body === null) return ['status'=>'Offline'];
    $stat = ['status'=>'OK'];
    $totalKB = 0;
    foreach (explode("\n", $body) as $line) {
        $parts = explode(': ', $line, 2);
        if (count($parts) !== 2) continue;
        [$key, $val] = [trim($parts[0]), trim($parts[1])];
        switch ($key) {
            case 'Total Accesses': $stat['total_accesses']=$val; break;
            case 'Total kBytes':
                $totalKB = (float)$val;
                $stat['total_traffic'] = number_format($totalKB/1024, 1).' MB'; break;
            case 'Uptime':
                $s=(int)$val; $stat['uptime']=floor($s/3600).'h '.floor(($s%3600)/60).'m'; break;
            case 'BusyWorkers':  $stat['busy_workers']=$val; break;
            case 'IdleWorkers':  $stat['idle_workers']=$val; break;
            case 'ReqPerSec':    $stat['req_per_sec']=$val; break;
            case 'BytesPerSec':  $stat['bytes_per_sec']=$val; break;
        }
    }
    if (empty($stat['total_traffic'])) $stat['total_traffic'] = number_format($totalKB/1024,1).' MB';
    return $stat;
}

function stripTags2($s) { return trim(strip_tags($s)); }

function fetchApacheWorkers($url) {
    $body = httpGet($url);
    if ($body === null) return [];

    $workers = [];
    preg_match_all('/<tr[^>]*>(.*?)<\/tr>/is', $body, $rows);
    foreach ($rows[1] as $row) {
        preg_match_all('/<td[^>]*>(.*?)<\/td>/is', $row, $cols);
        $c = $cols[1];
        $n = count($c);
        if ($n >= 15) {
            $vhost = stripTags2($c[13]);
            if (!$vhost || $vhost==='VHost') continue;
            $workers[] = [
                'srv'=>stripTags2($c[0]), 'pid'=>stripTags2($c[1]), 'acc'=>stripTags2($c[2]),
                'mode'=>stripTags2($c[3]), 'cpu'=>stripTags2($c[4]), 'ss'=>stripTags2($c[5]),
                'req'=>stripTags2($c[6]), 'dur'=>stripTags2($c[7]), 'conn'=>stripTags2($c[8]),
                'child'=>stripTags2($c[9]), 'slot'=>stripTags2($c[10]),
                'ip_address'=>stripTags2($c[11]), 'protocol'=>stripTags2($c[12]),
                'vhost'=>$vhost, 'url_request'=>stripTags2($c[14]),
            ];
        } elseif ($n >= 13) {
            $vhost = stripTags2($c[11]);
            if (!$vhost || $vhost==='VHost') continue;
            $workers[] = [
                'srv'=>stripTags2($c[0]), 'pid'=>stripTags2($c[1]), 'acc'=>'',
                'mode'=>stripTags2($c[3]), 'cpu'=>stripTags2($c[4]), 'ss'=>'',
                'req'=>'', 'dur'=>stripTags2($c[6]), 'conn'=>'', 'child'=>'', 'slot'=>stripTags2($c[9]),
                'ip_address'=>stripTags2($c[10]), 'protocol'=>'',
                'vhost'=>$vhost, 'url_request'=>stripTags2($c[12]),
            ];
        }
    }

    $groups = [];
    foreach ($workers as $w) {
        $groups[$w['vhost']][] = $w;
    }
    $result = [];
    foreach ($groups as $vhost => $wList) {
        $result[] = ['vhost'=>$vhost,'total_worker'=>count($wList),'workers'=>$wList];
    }
    usort($result, fn($a,$b) => $b['total_worker'] <=> $a['total_worker']);
    return $result;
}
?>
<!DOCTYPE html>
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

<!-- SUMMARY MODAL -->
<template x-teleport="body">
<div x-show="showSummary" x-cloak class="modal-overlay" @click.self="showSummary=false" @keydown.escape.window="showSummary=false">
  <div style="background:#0d1117;border:1px solid #30363d;border-radius:1rem;width:100%;max-width:960px;height:82vh;display:flex;flex-direction:column;box-shadow:0 32px 100px rgba(0,0,0,.85);animation:modalIn .22s ease;overflow:hidden" @click.stop>
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
      <div style="background:#161b22;border:1px solid #21262d;border-radius:.5rem;padding:.4rem .75rem;text-align:center"><div style="font-size:.55rem;color:#8b949e;text-transform:uppercase;letter-spacing:.06em">VHosts</div><div style="font-size:1.1rem;font-weight:700;font-family:'JetBrains Mono',monospace" x-text="(data.groups||[]).length"></div></div>
      <div style="background:#161b22;border:1px solid #21262d;border-radius:.5rem;padding:.4rem .75rem;text-align:center"><div style="font-size:.55rem;color:#8b949e;text-transform:uppercase;letter-spacing:.06em">Workers</div><div style="font-size:1.1rem;font-weight:700;font-family:'JetBrains Mono',monospace" x-text="(data.groups||[]).reduce(function(s,g){return s+g.total_worker;},0)"></div></div>
      <div style="background:#161b22;border:1px solid rgba(63,185,80,.3);border-radius:.5rem;padding:.4rem .75rem;text-align:center"><div style="font-size:.55rem;color:#3fb950;text-transform:uppercase;letter-spacing:.06em">Healthy</div><div style="font-size:1.1rem;font-weight:700;font-family:'JetBrains Mono',monospace;color:#3fb950" x-text="(data.groups||[]).filter(function(g){return healthStatus(g)==='healthy';}).length"></div></div>
      <div style="background:#161b22;border:1px solid rgba(210,153,34,.3);border-radius:.5rem;padding:.4rem .75rem;text-align:center"><div style="font-size:.55rem;color:#d29922;text-transform:uppercase;letter-spacing:.06em">Warning</div><div style="font-size:1.1rem;font-weight:700;font-family:'JetBrains Mono',monospace;color:#d29922" x-text="(data.groups||[]).filter(function(g){return healthStatus(g)==='warning';}).length"></div></div>
      <div style="background:#161b22;border:1px solid rgba(248,81,73,.3);border-radius:.5rem;padding:.4rem .75rem;text-align:center"><div style="font-size:.55rem;color:#f85149;text-transform:uppercase;letter-spacing:.06em">Critical</div><div style="font-size:1.1rem;font-weight:700;font-family:'JetBrains Mono',monospace;color:#f85149" x-text="(data.groups||[]).filter(function(g){return healthStatus(g)==='critical'||healthStatus(g)==='down';}).length"></div></div>
      <div style="background:#161b22;border:1px solid #21262d;border-radius:.5rem;padding:.4rem .75rem;text-align:center"><div style="font-size:.55rem;color:#8b949e;text-transform:uppercase;letter-spacing:.06em">Busy</div><div style="font-size:1.1rem;font-weight:700;font-family:'JetBrains Mono',monospace;color:#d29922" x-text="data.apache_summary&&data.apache_summary.busy_workers||'—'"></div></div>
      <div style="background:#161b22;border:1px solid #21262d;border-radius:.5rem;padding:.4rem .75rem;text-align:center"><div style="font-size:.55rem;color:#8b949e;text-transform:uppercase;letter-spacing:.06em">Req/s</div><div style="font-size:1.1rem;font-weight:700;font-family:'JetBrains Mono',monospace;color:#58a6ff" x-text="data.apache_summary&&data.apache_summary.req_per_sec||'—'"></div></div>
      <div style="background:#161b22;border:1px solid #21262d;border-radius:.5rem;padding:.4rem .75rem;text-align:center"><div style="font-size:.55rem;color:#8b949e;text-transform:uppercase;letter-spacing:.06em">Uptime</div><div style="font-size:1.1rem;font-weight:700;font-family:'JetBrains Mono',monospace;color:#8b949e" x-text="data.apache_summary&&data.apache_summary.uptime||'—'"></div></div>
    </div>
    <!-- 2-column body -->
    <div style="display:flex;flex:1;overflow:hidden;min-height:0">
      <!-- LEFT chart -->
      <div style="width:240px;flex-shrink:0;border-right:1px solid #21262d;padding:1rem;display:flex;flex-direction:column;gap:.625rem;overflow-y:auto">
        <div style="font-size:.625rem;font-weight:600;color:#8b949e;text-transform:uppercase;letter-spacing:.06em">CPU Allocation</div>
        <div style="position:relative;width:100%;height:180px;flex-shrink:0"><canvas id="cpuAllocChart"></canvas></div>
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
      <!-- RIGHT table -->
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
                <td style="padding:.5rem 1rem;font-weight:500"><div style="display:flex;align-items:center;gap:.5rem"><div class="sdot" :style="'background:'+healthColor(g)"></div><span x-text="g.vhost"></span></div></td>
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
<aside class="sidebar">
  <div class="sb-section">
    <div class="sb-label">System</div>
    <div class="sb-sys-card">
      <div class="sb-sys-name"><div class="sdot" :style="'background:'+(data.nginx&&data.nginx.status==='OK'?'var(--green)':'var(--red)')"></div>Nginx</div>
      <span class="badge" :class="data.nginx&&data.nginx.status==='OK'?'b-healthy':'b-offline'" x-text="data.nginx?data.nginx.status:'—'"></span>
    </div>
    <div class="sb-sys-card">
      <div class="sb-sys-name"><div class="sdot" :style="'background:'+(data.apache_summary&&data.apache_summary.status==='OK'?'var(--green)':'var(--red)')"></div>Apache</div>
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

<main class="content">
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
        var res = await fetch('?api=1');
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
        data: { labels: labels, datasets: [{ data: vals, backgroundColor: colors.slice(0, groups.length), borderWidth: 0, hoverOffset: 6 }] },
        options: {
          responsive: true, maintainAspectRatio: false, cutout: '68%', animation: false,
          plugins: {
            legend: { display: false },
            tooltip: { callbacks: { label: function(c) {
              var total = vals.reduce(function(s,v){ return s+v; }, 0);
              var pct = total > 0 ? (c.parsed/total*100).toFixed(1) : '0';
              return ' '+c.label+': '+c.parsed.toFixed(2)+' ('+pct+'%)';
            }}}
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
</html>
