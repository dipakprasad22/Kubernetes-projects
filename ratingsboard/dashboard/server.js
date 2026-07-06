// RatingsBoard Dashboard — Node.js server serving a single-page ratings view.
// Proxies /api/* to the reporting API; serves the dashboard UI at /.
// Exposes /healthz and /readyz for Kubernetes probes.
const http = require('http');
const { request } = require('http');

const PORT = process.env.PORT || 3000;
const API_URL = process.env.REPORTING_API_URL || 'http://reporting-api:8000';

const UI = `<!doctype html><html><head><title>RatingsBoard</title>
<style>body{font-family:system-ui;max-width:900px;margin:32px auto;padding:0 16px;color:#1a1a1a}
h1{color:#5b4bd0}table{width:100%;border-collapse:collapse;margin-top:16px}
th,td{text-align:left;padding:8px 10px;border-bottom:1px solid #eee}
th{background:#f6f5ff}button{padding:8px 14px;cursor:pointer;background:#5b4bd0;color:#fff;border:0;border-radius:6px}
.bar{height:10px;background:#5b4bd0;border-radius:4px;display:inline-block}</style></head>
<body><h1>📺 RatingsBoard</h1><p>Media measurement ratings &amp; share by channel and daypart.</p>
<button onclick="load()">Refresh Ratings</button><div id="out">Loading…</div>
<script>
async function load(){
  try{
    const r = await fetch('/api/ratings/top?limit=15');
    const d = await r.json();
    const rows = (d.top||[]).map(x =>
      '<tr><td>'+x.channel_id+'</td><td>'+x.daypart+'</td><td>'+x.rating+
      '</td><td><span class=bar style="width:'+(x.share*3)+'px"></span> '+x.share+'%</td></tr>').join('');
    document.getElementById('out').innerHTML = rows
      ? '<table><tr><th>Channel</th><th>Daypart</th><th>Rating</th><th>Share</th></tr>'+rows+'</table>'
      : '<p>No ratings yet — run the rating-calc job.</p>';
  }catch(e){ document.getElementById('out').innerHTML = '<p>API unavailable.</p>'; }
}
load();
</script></body></html>`;

const server = http.createServer((req, res) => {
  if (req.url === '/healthz') { res.writeHead(200).end('{"status":"ok"}'); return; }
  if (req.url === '/readyz')  { res.writeHead(200).end('{"status":"ready"}'); return; }
  if (req.url.startsWith('/api/')) { return proxy(req, res); }
  res.writeHead(200, {'Content-Type':'text/html'}).end(UI);
});

function proxy(req, res) {
  const t = new URL(API_URL + req.url);
  const up = request({ hostname: t.hostname, port: t.port, path: t.pathname + t.search,
    method: req.method, headers: req.headers, timeout: 5000 },
    (r) => { res.writeHead(r.statusCode, r.headers); r.pipe(res); });
  up.on('error', () => { res.writeHead(502).end('{"error":"api unavailable"}'); });
  req.pipe(up);
}

server.listen(PORT, () => console.log('ratingsboard dashboard on :' + PORT));
