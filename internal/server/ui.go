package server

import (
	"net/http"
)

const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Aegis AI Gateway — Live Playground & Inspector</title>
  <script src="https://cdn.tailwindcss.com"></script>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; }
    .pulse-dot { animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite; }
    @keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: .4; } }
  </style>
</head>
<body class="bg-slate-950 text-slate-100 min-h-screen flex flex-col">
  <header class="border-b border-slate-800 bg-slate-900/80 backdrop-blur sticky top-0 z-50 px-6 py-3.5 flex flex-wrap justify-between items-center">
    <div class="flex items-center gap-3">
      <div class="w-8 h-8 rounded-lg bg-gradient-to-tr from-blue-600 to-indigo-500 flex items-center justify-center font-bold text-white shadow-lg shadow-blue-500/20">
        Æ
      </div>
      <div>
        <div class="flex items-center gap-2">
          <h1 class="text-base font-bold text-white tracking-tight">Aegis AI Gateway</h1>
          <span class="text-[10px] px-2 py-0.5 rounded-full bg-blue-500/10 text-blue-400 border border-blue-500/20 font-mono">v1.0 Live</span>
        </div>
        <p class="text-xs text-slate-400">High-Performance Reverse Proxy & 2-Tier Caching System</p>
      </div>
    </div>
    <div class="flex items-center gap-3 text-xs font-mono mt-2 sm:mt-0">
      <div id="status-badge" class="flex items-center gap-2 px-3 py-1 rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
        <span class="w-2 h-2 rounded-full bg-emerald-500 pulse-dot"></span>
        <span id="status-text">Checking health...</span>
      </div>
      <a href="/metrics" target="_blank" class="px-2.5 py-1 rounded bg-slate-800 hover:bg-slate-700 text-slate-300 border border-slate-700 transition">Prometheus Metrics</a>
    </div>
  </header>

  <main class="flex-1 max-w-7xl w-full mx-auto p-4 sm:p-6 grid grid-cols-1 lg:grid-cols-12 gap-6">
    <section class="lg:col-span-7 flex flex-col gap-4">
      <div class="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-sm">
        <div class="flex items-center justify-between mb-3">
          <h2 class="text-sm font-semibold text-slate-200">Interactive Gateway Playground</h2>
          <div class="flex items-center gap-2 text-xs">
            <label class="flex items-center gap-1.5 cursor-pointer text-slate-400 hover:text-slate-200">
              <input type="checkbox" id="bypass-exact" class="rounded bg-slate-800 border-slate-700 text-blue-500 focus:ring-0">
              <span>Bypass Exact Cache</span>
            </label>
          </div>
        </div>

        <div class="flex flex-wrap gap-2 mb-3">
          <span class="text-xs text-slate-500 self-center">Try Quick Scenarios:</span>
          <button onclick="setPrompt('Explain quantum computing in one sentence.')" class="text-xs px-2.5 py-1 rounded bg-slate-800/80 hover:bg-slate-700 text-slate-300 border border-slate-700/60 transition">
            1. Original Query
          </button>
          <button onclick="setPrompt('Can you describe quantum computing in a single sentence?')" class="text-xs px-2.5 py-1 rounded bg-slate-800/80 hover:bg-slate-700 text-cyan-300 border border-cyan-800/60 transition">
            2. Paraphrase (Semantic Test)
          </button>
          <button onclick="setPrompt('Implement a production concurrent worker pool in Go with bounded channels and context cancellation.')" class="text-xs px-2.5 py-1 rounded bg-slate-800/80 hover:bg-slate-700 text-purple-300 border border-purple-800/60 transition">
            3. Deep Technical (Router Test)
          </button>
        </div>

        <div class="space-y-3">
          <div>
            <label class="block text-xs text-slate-400 mb-1">Model Route Mode</label>
            <select id="model-select" class="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-xs text-slate-200 focus:outline-none focus:border-blue-500">
              <option value="">Auto-Route via Aegis Classifier (Hybrid Cosine + Heuristic)</option>
              <option value="gemini-2.5-flash-lite">Force gemini-2.5-flash-lite (Fast)</option>
              <option value="gemini-2.5-flash">Force gemini-2.5-flash (Balanced)</option>
              <option value="gemini-2.5-pro">Force gemini-2.5-pro (Deep)</option>
            </select>
          </div>

          <div>
            <label class="block text-xs text-slate-400 mb-1">Prompt / Message</label>
            <textarea id="prompt-input" rows="4" class="w-full bg-slate-950 border border-slate-800 rounded-lg p-3 text-sm text-slate-100 placeholder-slate-600 focus:outline-none focus:border-blue-500 transition resize-none font-sans" placeholder="Type a prompt to send through Aegis..."></textarea>
          </div>

          <div class="flex items-center justify-between pt-1">
            <span class="text-[11px] text-slate-500 font-mono">Press Ctrl + Enter to dispatch</span>
            <button id="send-btn" onclick="sendRequest()" class="px-5 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-xs font-semibold shadow-lg shadow-blue-600/20 transition flex items-center gap-2">
              <span>Send Request</span>
            </button>
          </div>
        </div>
      </div>

      <div class="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-sm flex-1 flex flex-col">
        <div class="flex items-center justify-between mb-3 border-b border-slate-800/80 pb-2">
          <h3 class="text-xs font-semibold text-slate-300 uppercase tracking-wider">Gateway Response Output</h3>
          <span id="response-time" class="text-xs font-mono text-slate-400">Ready</span>
        </div>
        <div id="response-box" class="flex-1 bg-slate-950 border border-slate-800/80 rounded-lg p-4 font-mono text-xs text-slate-300 overflow-y-auto whitespace-pre-wrap min-h-[220px]">
Click "Send Request" or select a scenario above to test the gateway live.
        </div>
      </div>
    </section>

    <section class="lg:col-span-5 flex flex-col gap-4">
      <div class="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-sm">
        <h3 class="text-xs font-semibold text-slate-300 uppercase tracking-wider mb-4">Live Gateway Telemetry Inspector</h3>

        <div class="space-y-4">
          <div class="bg-slate-950 border border-slate-800 rounded-lg p-3.5">
            <div class="flex items-center justify-between mb-2">
              <span class="text-xs text-slate-400">Cache Layer Resolution</span>
              <span id="badge-cache" class="text-xs font-mono px-2 py-0.5 rounded bg-slate-800 text-slate-400">IDLE</span>
            </div>
            <div class="grid grid-cols-2 gap-2 text-[11px] font-mono mt-3 border-t border-slate-800/60 pt-2">
              <div>
                <span class="text-slate-500">Exact Cache (Redis):</span>
                <span id="stat-exact" class="text-slate-300 block">-</span>
              </div>
              <div>
                <span class="text-slate-500">Semantic Cache (pgvector):</span>
                <span id="stat-semantic" class="text-slate-300 block">-</span>
              </div>
            </div>
          </div>

          <div class="bg-slate-950 border border-slate-800 rounded-lg p-3.5">
            <div class="flex items-center justify-between mb-2">
              <span class="text-xs text-slate-400">Round-Trip Latency</span>
              <span id="stat-latency" class="text-xs font-mono font-bold text-slate-300">- ms</span>
            </div>
            <div class="w-full bg-slate-900 rounded-full h-1.5 overflow-hidden">
              <div id="latency-bar" class="bg-blue-500 h-full w-0 transition-all duration-300"></div>
            </div>
            <div class="grid grid-cols-2 gap-2 text-[11px] font-mono mt-3 border-t border-slate-800/60 pt-2">
              <div>
                <span class="text-slate-500">Embedding Hop:</span>
                <span id="stat-embed-dur" class="text-slate-300 block">-</span>
              </div>
              <div>
                <span class="text-slate-500">HNSW Vector Search:</span>
                <span id="stat-vec-dur" class="text-slate-300 block">-</span>
              </div>
            </div>
          </div>

          <div class="bg-slate-950 border border-slate-800 rounded-lg p-3.5">
            <div class="flex items-center justify-between mb-2">
              <span class="text-xs text-slate-400">Model Routing Decision</span>
              <span id="badge-routing" class="text-xs font-mono px-2 py-0.5 rounded bg-slate-800 text-slate-400">IDLE</span>
            </div>
            <div class="text-[11px] font-mono space-y-1 mt-2 border-t border-slate-800/60 pt-2">
              <div class="flex justify-between">
                <span class="text-slate-500">Dispatched Model:</span>
                <span id="stat-model" class="text-slate-300">-</span>
              </div>
              <div class="flex justify-between">
                <span class="text-slate-500">Routing Scores:</span>
                <span id="stat-scores" class="text-slate-300 truncate max-w-[200px]">-</span>
              </div>
            </div>
          </div>

          <div class="bg-slate-950 border border-slate-800 rounded-lg p-3.5">
            <div class="flex items-center justify-between mb-2">
              <span class="text-xs text-slate-400">Upstream Circuit & Failover</span>
              <span id="badge-provider" class="text-xs font-mono px-2 py-0.5 rounded bg-slate-800 text-slate-400">PRIMARY</span>
            </div>
            <div class="text-[11px] font-mono flex justify-between mt-2 border-t border-slate-800/60 pt-2">
              <span class="text-slate-500">Failover Engaged:</span>
              <span id="stat-failover" class="text-emerald-400">No (Healthy)</span>
            </div>
          </div>
        </div>
      </div>

      <div class="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-sm">
        <h3 class="text-xs font-semibold text-slate-300 uppercase tracking-wider mb-2">Inspect Response Headers</h3>
        <pre id="headers-box" class="bg-slate-950 border border-slate-800/80 rounded-lg p-3 text-[11px] font-mono text-slate-400 overflow-x-auto max-h-48 whitespace-pre">No request sent yet.</pre>
      </div>
    </section>
  </main>

  <footer class="border-t border-slate-800/80 py-4 px-6 text-center text-xs text-slate-500">
    Aegis AI Gateway — Reverse Proxy with Redis Exact Caching & PostgreSQL pgvector Semantic HNSW Search
  </footer>

  <script>
    async function checkHealth() {
      try {
        const res = await fetch('/health');
        const data = await res.json();
        const badge = document.getElementById('status-badge');
        const text = document.getElementById('status-text');
        if (data.status === 'available') {
          badge.className = 'flex items-center gap-2 px-3 py-1 rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20';
          text.textContent = 'Healthy (Postgres: ' + data.postgres + ', Redis: ' + data.redis + ')';
        } else {
          badge.className = 'flex items-center gap-2 px-3 py-1 rounded-full bg-amber-500/10 text-amber-400 border border-amber-500/20';
          text.textContent = 'Degraded (Postgres: ' + data.postgres + ', Redis: ' + data.redis + ')';
        }
      } catch (e) {
        document.getElementById('status-text').textContent = 'Server Unreachable';
      }
    }
    checkHealth();

    function setPrompt(text) {
      document.getElementById('prompt-input').value = text;
    }

    document.getElementById('prompt-input').addEventListener('keydown', function(e) {
      if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
        sendRequest();
      }
    });

    async function sendRequest() {
      const prompt = document.getElementById('prompt-input').value.trim();
      if (!prompt) return;

      const sendBtn = document.getElementById('send-btn');
      const responseBox = document.getElementById('response-box');
      const responseTime = document.getElementById('response-time');
      const bypassExact = document.getElementById('bypass-exact').checked;
      const selectedModel = document.getElementById('model-select').value;

      sendBtn.disabled = true;
      sendBtn.classList.add('opacity-50');
      responseBox.textContent = 'Routing request through Aegis Gateway...';
      responseTime.textContent = 'In Flight...';

      const startTime = performance.now();

      const headers = {
        'Content-Type': 'application/json'
      };
      if (bypassExact) {
        headers['X-Bypass-Exact-Cache'] = 'true';
      }

      const payload = {
        messages: [{ role: 'user', content: prompt }]
      };
      if (selectedModel) {
        payload.model = selectedModel;
      }

      try {
        const res = await fetch('/v1/chat/completions', {
          method: 'POST',
          headers: headers,
          body: JSON.stringify(payload)
        });

        const elapsed = (performance.now() - startTime).toFixed(1);
        responseTime.textContent = elapsed + ' ms';
        document.getElementById('stat-latency').textContent = elapsed + ' ms';

        let barPct = Math.min(100, Math.max(5, (elapsed / 2000) * 100));
        document.getElementById('latency-bar').style.width = barPct + '%';

        const cacheHeader = res.headers.get('X-Cache') || 'UNKNOWN';
        const similarity = res.headers.get('X-Cache-Similarity');
        const embDuration = res.headers.get('X-Embedding-Duration-Ms');
        const vecDuration = res.headers.get('X-Vector-Duration-Ms');
        const routeClass = res.headers.get('X-Routing-Class');
        const routeModel = res.headers.get('X-Routing-Model');
        const routeScores = res.headers.get('X-Routing-Scores');
        const provider = res.headers.get('X-Provider') || 'primary';
        const failover = res.headers.get('X-Provider-Failover') === 'true';

        const badgeCache = document.getElementById('badge-cache');
        if (cacheHeader === 'EXACT_HIT') {
          badgeCache.className = 'text-xs font-mono px-2 py-0.5 rounded bg-emerald-500/20 text-emerald-400 border border-emerald-500/30';
          badgeCache.textContent = 'EXACT HIT (Redis)';
          document.getElementById('stat-exact').textContent = 'HIT (<2ms)';
          document.getElementById('stat-semantic').textContent = 'Bypassed (Exact Match)';
        } else if (cacheHeader === 'SEMANTIC_HIT') {
          badgeCache.className = 'text-xs font-mono px-2 py-0.5 rounded bg-cyan-500/20 text-cyan-400 border border-cyan-500/30';
          badgeCache.textContent = 'SEMANTIC HIT (pgvector)';
          document.getElementById('stat-exact').textContent = bypassExact ? 'Bypassed by Header' : 'MISS';
          document.getElementById('stat-semantic').textContent = 'HIT (' + (similarity ? 'Cosine: ' + similarity : 'Similar') + ')';
        } else {
          badgeCache.className = 'text-xs font-mono px-2 py-0.5 rounded bg-amber-500/20 text-amber-400 border border-amber-500/30';
          badgeCache.textContent = 'COLD MISS (Forwarded)';
          document.getElementById('stat-exact').textContent = 'MISS';
          document.getElementById('stat-semantic').textContent = 'MISS';
        }

        document.getElementById('stat-embed-dur').textContent = embDuration ? embDuration + ' ms' : '-';
        document.getElementById('stat-vec-dur').textContent = vecDuration ? vecDuration + ' ms' : '-';

        const badgeRouting = document.getElementById('badge-routing');
        if (routeClass) {
          badgeRouting.className = 'text-xs font-mono px-2 py-0.5 rounded bg-purple-500/20 text-purple-400 border border-purple-500/30';
          badgeRouting.textContent = routeClass;
          document.getElementById('stat-model').textContent = routeModel || '-';
          document.getElementById('stat-scores').textContent = routeScores || '-';
        } else {
          badgeRouting.textContent = selectedModel || 'DEFAULT';
          document.getElementById('stat-model').textContent = selectedModel || 'gemini-2.5-flash';
        }

        const badgeProvider = document.getElementById('badge-provider');
        const statFailover = document.getElementById('stat-failover');
        if (failover) {
          badgeProvider.className = 'text-xs font-mono px-2 py-0.5 rounded bg-rose-500/20 text-rose-400 border border-rose-500/30';
          badgeProvider.textContent = 'FALLBACK SECONDARY';
          statFailover.className = 'text-rose-400';
          statFailover.textContent = 'Engaged (Primary Failed)';
        } else {
          badgeProvider.className = 'text-xs font-mono px-2 py-0.5 rounded bg-emerald-500/20 text-emerald-400 border border-emerald-500/30';
          badgeProvider.textContent = 'PRIMARY (' + provider.toUpperCase() + ')';
          statFailover.className = 'text-emerald-400';
          statFailover.textContent = 'No (Healthy)';
        }

        let headerDump = 'Status: ' + res.status + ' ' + res.statusText + '\n';
        res.headers.forEach((v, k) => {
          if (k.startsWith('x-') || k === 'content-type') {
            headerDump += k + ': ' + v + '\n';
          }
        });
        document.getElementById('headers-box').textContent = headerDump;

        const data = await res.json();
        if (data.choices && data.choices.length > 0 && data.choices[0].message) {
          responseBox.textContent = data.choices[0].message.content;
        } else {
          responseBox.textContent = JSON.stringify(data, null, 2);
        }
      } catch (err) {
        responseBox.textContent = 'Error sending request: ' + err.message;
        responseTime.textContent = 'Failed';
      } finally {
        sendBtn.disabled = false;
        sendBtn.classList.remove('opacity-50');
      }
    }
  </script>
</body>
</html>`

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(indexHTML))
}
