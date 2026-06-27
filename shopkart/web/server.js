// ShopKart Web Frontend / BFF (Backend-for-Frontend)
// Serves a minimal storefront UI and proxies browser calls to the API gateway.
// Stateless, scalable. Exposes /health, /ready, / (UI), /api/* (proxy to gateway).
const http = require('http');
const { request } = require('http');

const PORT = process.env.PORT || 3000;
const GATEWAY_URL = process.env.GATEWAY_URL || 'http://gateway:8080';

const UI = `<!doctype html><html><head><title>ShopKart</title>
<style>body{font-family:system-ui;max-width:760px;margin:40px auto;padding:0 16px}
h1{color:#2d6}button{padding:8px 14px;margin:4px;cursor:pointer}
.p{border:1px solid #ddd;border-radius:8px;padding:12px;margin:8px 0}</style></head>
<body><h1>🛒 ShopKart</h1><p>Microservices storefront on Kubernetes.</p>
<button onclick="load()">Load Products</button><div id="out"></div>
<script>async function load(){const r=await fetch('/api/catalog/products');
const ps=await r.json();document.getElementById('out').innerHTML=
ps.map(p=>'<div class=p><b>'+p.name+'</b> — $'+p.price+' ('+p.stock+' in stock)</div>').join('')}
</script></body></html>`;

const server = http.createServer((req, res) => {
  if (req.url === '/health') { res.writeHead(200).end('{"status":"ok"}'); return; }
  if (req.url === '/ready')  { res.writeHead(200).end('{"status":"ready"}'); return; }
  if (req.url.startsWith('/api/')) { return proxyToGateway(req, res); }
  res.writeHead(200, {'Content-Type':'text/html'}).end(UI);
});

function proxyToGateway(req, res) {
  const target = new URL(GATEWAY_URL + req.url);
  const upstream = request({
    hostname: target.hostname, port: target.port, path: target.pathname + target.search,
    method: req.method, headers: req.headers, timeout: 5000,
  }, (up) => { res.writeHead(up.statusCode, up.headers); up.pipe(res); });
  upstream.on('error', () => { res.writeHead(502).end('{"error":"gateway unavailable"}'); });
  req.pipe(upstream);
}

server.listen(PORT, () => console.log('shopkart web listening on :' + PORT));
