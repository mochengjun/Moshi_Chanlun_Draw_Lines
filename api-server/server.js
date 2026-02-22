/**
 * Moshi Chanlun API Server
 * 
 * HTTP API + WebSocket server on port 8080.
 * Generates sample K-line data and calls C++ calculator_cli for indicator computation.
 */

import express from 'express';
import { createServer } from 'http';
import { WebSocketServer, WebSocket } from 'ws';
import { execFile } from 'child_process';
import { fileURLToPath } from 'url';
import path from 'path';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const CLI_PATH = path.resolve(__dirname, '../cpp-trading-system/build/calculator_cli');
const PORT = 5173;

// Data source configuration - 点证API
const DATA_SOURCE_URL = 'http://101.201.37.86:8501';

// Request ID counter
let reqIdCounter = 1;

// ============================================================================
// Market code resolution (based on stock code prefix)
// ============================================================================

const MarketCodes = {
  SH_INDEX: 0,     // 上证指数
  SH_A: 1,         // 上证A股
  SH_B: 2,         // 上证B股
  STAR: 7,         // 科创板
  SZ_INDEX: 1000,  // 深证指数
  SZ_A: 1001,      // 深证A股
  SZ_B: 1002,      // 深证B股
  GEM: 1008,       // 创业板
};

function resolveMarket(market, code) {
  // If already precise market code (>=2), use directly
  if (market >= 2) return market;
  if (!code || code.length < 3) return market;

  const prefix3 = code.slice(0, 3);
  const prefix2 = code.slice(0, 2);

  // 创业板: 300xxx, 301xxx
  if (prefix3 === '300' || prefix3 === '301') return MarketCodes.GEM;
  // 科创板: 688xxx, 689xxx
  if (prefix3 === '688' || prefix3 === '689') return MarketCodes.STAR;
  // 深证A股: 002xxx, 003xxx
  if (prefix3 === '002' || prefix3 === '003') return MarketCodes.SZ_A;
  // 深证指数: 399xxx
  if (prefix3 === '399') return MarketCodes.SZ_INDEX;
  // 上证B股: 900xxx
  if (prefix3 === '900') return MarketCodes.SH_B;
  // 深证B股: 200xxx
  if (prefix3 === '200') return MarketCodes.SZ_B;
  // 沪市个股: 600xxx, 601xxx, 603xxx, 605xxx
  if (prefix2 === '60') return MarketCodes.SH_A;
  // 上证指数: 000xxx (when market=0)
  if (prefix3 === '000' && market === 0) return MarketCodes.SH_INDEX;
  // 深证A股: 000xxx, 001xxx (when market=1)
  if ((prefix3 === '000' || prefix3 === '001') && market === 1) return MarketCodes.SZ_A;

  return market;
}

// ============================================================================
// Real K-line data fetching from 点证 API
// ============================================================================

/**
 * Fetch K-line data from 点证 API using POST request
 */
async function fetchKLineFromAPI(market, code, klineType, count, weight = 0) {
  const resolvedMarket = resolveMarket(market, code);
  const now = new Date();
  const endTime = `${now.getFullYear()}-${now.getMonth() + 1}-${now.getDate()} ${pad2(now.getHours())}:${pad2(now.getMinutes())}:${pad2(now.getSeconds())}`;
  
  const requestBody = {
    reqtype: 150,
    reqid: reqIdCounter++,
    session: '',
    data: {
      market: resolvedMarket,
      code: code,
      klinetype: klineType,
      weight: weight,
      timetype: 2,  // 往前count条
      time0: endTime,
      count: count,
    },
  };

  try {
    const response = await fetch(DATA_SOURCE_URL, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
      },
      body: JSON.stringify(requestBody),
      signal: AbortSignal.timeout(10000), // 10s timeout
    });
    
    if (!response.ok) {
      throw new Error(`API returned HTTP ${response.status}`);
    }
    
    const data = await response.json();
    
    // Check status (0 = success)
    if (data.status !== 0) {
      throw new Error(data.msg || `API status ${data.status}`);
    }
    
    // Parse kline data from response
    const klineList = data.data?.kline;
    if (!klineList || klineList.length < 2) {
      throw new Error('Empty kline data');
    }
    
    // Skip header row (index 0), parse data rows
    const klines = [];
    for (let i = 1; i < klineList.length; i++) {
      const row = klineList[i];
      if (row.length < 6) continue;
      
      klines.push({
        timestamp: row[0],
        open: parseFloat(row[1]) || 0,
        high: parseFloat(row[2]) || 0,
        low: parseFloat(row[3]) || 0,
        close: parseFloat(row[4]) || 0,
        volume: parseInt(row[5]) || 0,
        amount: row.length > 6 ? parseFloat(row[6]) || 0 : 0,
      });
    }
    
    console.log(`Fetched ${klines.length} klines from 点证 API for ${code} (market=${resolvedMarket})`);
    return {
      klines: klines,
      name: code,
      fromAPI: true,
    };
  } catch (err) {
    console.warn(`Failed to fetch from 点证 API: ${err.message}, falling back to generated data`);
    return null;
  }
}

// ============================================================================
// Sample K-line data generation
// ============================================================================

/**
 * Generate realistic stock K-line data using random walk with mean reversion.
 */
function generateKLineData(count, klineType) {
  const klines = [];
  const now = new Date();
  let price = 3200 + Math.random() * 300; // Base price around 3200-3500
  let trend = 0; // Current trend momentum
  const volatility = getVolatility(klineType);

  for (let i = count - 1; i >= 0; i--) {
    const timestamp = getTimestamp(now, i, klineType);

    // Update trend with mean reversion
    trend = trend * 0.95 + (Math.random() - 0.5) * 0.02;
    // Add some periodic trend changes
    const cyclePhase = Math.sin(i * 0.05) * 0.005;
    const drift = trend + cyclePhase;

    const change = price * (drift + (Math.random() - 0.5) * volatility);
    const open = price;
    const close = price + change;

    // Generate high and low
    const range = Math.abs(change) + price * volatility * Math.random() * 0.5;
    const high = Math.max(open, close) + Math.abs(range * Math.random() * 0.3);
    const low = Math.min(open, close) - Math.abs(range * Math.random() * 0.3);

    const volume = Math.round(50000 + Math.random() * 200000 + Math.abs(change) * 1000);
    const amount = Math.round(volume * (open + close) / 2);

    klines.push({
      timestamp,
      open: round2(open),
      high: round2(high),
      low: round2(low),
      close: round2(close),
      volume,
      amount,
    });

    price = close;
    // Mean reversion to prevent drift too far
    if (price > 4000) trend -= 0.003;
    if (price < 2800) trend += 0.003;
  }

  return klines;
}

function getVolatility(klineType) {
  const map = {
    1: 0.002, 4: 0.003, 2: 0.004, 5: 0.006, 6: 0.008,
    3: 0.01, 8: 0.012, 7: 0.015, 10: 0.02, 11: 0.03,
    20: 0.04, 21: 0.06, 30: 0.08,
  };
  return map[klineType] || 0.02;
}

function getTimestamp(now, offsetBack, klineType) {
  const d = new Date(now);
  switch (klineType) {
    case 1: // 1 min
      d.setMinutes(d.getMinutes() - offsetBack);
      return formatDateTime(d);
    case 4: // 3 min
      d.setMinutes(d.getMinutes() - offsetBack * 3);
      return formatDateTime(d);
    case 2: // 5 min
      d.setMinutes(d.getMinutes() - offsetBack * 5);
      return formatDateTime(d);
    case 5: // 15 min
      d.setMinutes(d.getMinutes() - offsetBack * 15);
      return formatDateTime(d);
    case 6: // 30 min
      d.setMinutes(d.getMinutes() - offsetBack * 30);
      return formatDateTime(d);
    case 3: // 60 min
      d.setHours(d.getHours() - offsetBack);
      return formatDateTime(d);
    case 8: // 120 min
      d.setHours(d.getHours() - offsetBack * 2);
      return formatDateTime(d);
    case 7: // half day
      d.setHours(d.getHours() - offsetBack * 4);
      return formatDateTime(d);
    case 11: // weekly
      d.setDate(d.getDate() - offsetBack * 7);
      return formatDate(d);
    case 20: // monthly
      d.setMonth(d.getMonth() - offsetBack);
      return formatDate(d);
    case 21: // quarterly
      d.setMonth(d.getMonth() - offsetBack * 3);
      return formatDate(d);
    case 30: // yearly
      d.setFullYear(d.getFullYear() - offsetBack);
      return formatDate(d);
    default: // daily (10)
      d.setDate(d.getDate() - offsetBack);
      return formatDate(d);
  }
}

function formatDate(d) {
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}`;
}

function formatDateTime(d) {
  return `${formatDate(d)} ${pad2(d.getHours())}:${pad2(d.getMinutes())}:00`;
}

function pad2(n) { return n < 10 ? '0' + n : '' + n; }
function round2(n) { return Math.round(n * 100) / 100; }

// ============================================================================
// C++ calculator invocation
// ============================================================================

function callCalculator(klines, params) {
  return new Promise((resolve, reject) => {
    const input = JSON.stringify({ klines, params });
    const child = execFile(CLI_PATH, [], { maxBuffer: 50 * 1024 * 1024 }, (error, stdout, stderr) => {
      if (error) {
        console.error('Calculator error:', stderr);
        reject(new Error('Calculator execution failed: ' + (stderr || error.message)));
        return;
      }
      try {
        const result = JSON.parse(stdout);
        if (result.error) {
          reject(new Error(result.error));
        } else {
          resolve(result);
        }
      } catch (e) {
        reject(new Error('Failed to parse calculator output'));
      }
    });
    child.stdin.write(input);
    child.stdin.end();
  });
}

// ============================================================================
// K-line data cache (per stock+timeframe)
// ============================================================================

const klineCache = new Map();

function getCacheKey(market, code, klineType) {
  return `${market}_${code}_${klineType}`;
}

/**
 * Get K-line data: first try 点证 API, fallback to generated data
 */
async function getOrGenerateKlines(market, code, klineType, count, weight = 0) {
  const key = getCacheKey(market, code, klineType);
  
  // Try to fetch from 点证 API first
  const apiResult = await fetchKLineFromAPI(market, code, klineType, count, weight);
  
  if (apiResult && apiResult.klines && apiResult.klines.length > 0) {
    // Cache the real data
    klineCache.set(key, { klines: apiResult.klines, name: apiResult.name, fromAPI: true });
    return apiResult;
  }
  
  // Fallback to cached or generated data
  let cached = klineCache.get(key);
  if (!cached || cached.klines.length < count) {
    cached = { klines: generateKLineData(count, klineType), name: getStockName(market, code), fromAPI: false };
    klineCache.set(key, cached);
  }
  
  return { klines: cached.klines.slice(0, count), name: cached.name, fromAPI: cached.fromAPI };
}

// ============================================================================
// Express HTTP server
// ============================================================================

const app = express();
app.use(express.json({ limit: '50mb' }));

// Health check
app.get('/api/v1/health', (_req, res) => {
  res.json({ status: 'ok', server: 'moshi-chanlun-cpp', uptime: process.uptime() });
});

// Get K-line data
app.get('/api/v1/kline', async (req, res) => {
  const market = parseInt(req.query.market) || 0;
  const code = req.query.code || '000001';
  const klineType = parseInt(req.query.klinetype) || 10;
  const count = Math.min(Math.max(parseInt(req.query.count) || 2000, 10), 5000);
  const weight = parseInt(req.query.weight) || 0;

  const result = await getOrGenerateKlines(market, code, klineType, count, weight);

  res.json({
    code: 0,
    message: 'success',
    data: {
      market,
      code,
      name: result.name,
      klinetype: klineType,
      weight,
      klines: result.klines,
      count: result.klines.length,
    },
    data_source: result.fromAPI ? '点证API' : 'generated',
    cache_hit: klineCache.has(getCacheKey(market, code, klineType)),
  });
});

// Calculate indicators
app.post('/api/v1/indicators/calculate', async (req, res) => {
  const { market, code, klinetype, count, indicators: indicatorConfigs } = req.body;

  const klineType = klinetype || 10;
  const klineCount = Math.min(Math.max(count || 2000, 10), 5000);

  const klineResult = await getOrGenerateKlines(market || 0, code || '000001', klineType, klineCount);
  const klines = klineResult.klines;

  const t0 = Date.now();
  const results = [];

  for (const config of (indicatorConfigs || [])) {
    if (config.type === 'moshi_chanlun') {
      try {
        const params = {
          kline_type: klineType,
          ...(config.params || {}),
        };
        // Convert boolean params to numbers for C++
        for (const key of Object.keys(params)) {
          if (typeof params[key] === 'boolean') {
            params[key] = params[key] ? 1 : 0;
          }
        }
        const result = await callCalculator(klines, params);
        results.push(result);
      } catch (err) {
        console.error('Indicator calculation failed:', err.message);
        results.push({
          type: 'moshi_chanlun',
          name: '莫氏缠论',
          error: err.message,
          computation_time_ms: Date.now() - t0,
        });
      }
    }
  }

  res.json({
    code: 0,
    message: 'success',
    stock_code: code || '000001',
    indicators: results,
    data_source: klineResult.fromAPI ? '点证API' : 'generated',
    computation_time_ms: Date.now() - t0,
  });
});

// Get indicator list
app.get('/api/v1/indicators/list', (_req, res) => {
  res.json({
    code: 0,
    message: 'success',
    indicators: [
      {
        type: 'moshi_chanlun',
        name: '莫氏缠论',
        category: 'chanlun',
        description: '莫氏缠论画线指标 - 多级别(sub-x1/x1/x2/x4/x8)标注点、走势识别',
        params_def: [
          { name: 'kline_type', type: 'int', required: false, default_value: 10, description: 'K线类型' },
          { name: 'show_level_sub_x1', type: 'bool', required: false, default_value: true, description: '显示sub-x1级别' },
          { name: 'show_level_1x', type: 'bool', required: false, default_value: true, description: '显示x1级别' },
          { name: 'show_level_2x', type: 'bool', required: false, default_value: true, description: '显示x2级别' },
          { name: 'show_level_4x', type: 'bool', required: false, default_value: true, description: '显示x4级别' },
          { name: 'show_level_8x', type: 'bool', required: false, default_value: true, description: '显示x8级别' },
        ],
      },
    ],
  });
});

function getStockName(market, code) {
  const names = {
    '0_000001': '上证指数',
    '1_600000': '浦发银行',
    '1_601398': '工商银行',
    '1001_000002': '万科A',
    '1001_000858': '五粮液',
    '1008_300750': '宁德时代',
    '1000_399001': '深证成指',
  };
  return names[`${market}_${code}`] || `${code}`;
}

// ============================================================================
// WebSocket server
// ============================================================================

const server = createServer(app);

const wss = new WebSocketServer({ server, path: '/ws/kline' });

// Track subscriptions per client
const clientSubscriptions = new Map();

wss.on('connection', (ws) => {
  console.log('WebSocket client connected');
  clientSubscriptions.set(ws, []);

  // Send connected message
  ws.send(JSON.stringify({
    type: 'connected',
    data: { message: 'WebSocket connection established' },
    timestamp: new Date().toISOString(),
  }));

  ws.on('message', (data) => {
    try {
      const msg = JSON.parse(data.toString());

      switch (msg.type) {
        case 'ping':
          ws.send(JSON.stringify({
            type: 'pong',
            timestamp: new Date().toISOString(),
          }));
          break;

        case 'subscribe':
          if (msg.data?.stocks) {
            clientSubscriptions.set(ws, msg.data.stocks);
            console.log('Client subscribed to:', msg.data.stocks);
          }
          break;

        case 'unsubscribe':
          clientSubscriptions.set(ws, []);
          break;

        default:
          break;
      }
    } catch (e) {
      console.error('WebSocket message parse error:', e);
    }
  });

  ws.on('close', () => {
    console.log('WebSocket client disconnected');
    clientSubscriptions.delete(ws);
  });

  ws.on('error', (err) => {
    console.error('WebSocket error:', err);
    clientSubscriptions.delete(ws);
  });
});

// Periodic indicator updates for subscribed clients (every 30 seconds)
setInterval(async () => {
  for (const [ws, subscriptions] of clientSubscriptions.entries()) {
    if (ws.readyState !== WebSocket.OPEN || subscriptions.length === 0) continue;

    for (const sub of subscriptions) {
      try {
        const klineResult = await getOrGenerateKlines(sub.market, sub.code, sub.klinetype, 2000);
        const klines = [...klineResult.klines]; // Clone to avoid mutating cache

        // Add a small random change to the last kline to simulate real-time
        const last = { ...klines[klines.length - 1] };
        const change = last.close * (Math.random() - 0.5) * 0.002;
        last.close = round2(last.close + change);
        last.high = round2(Math.max(last.high, last.close));
        last.low = round2(Math.min(last.low, last.close));
        klines[klines.length - 1] = last;

        const params = {
          kline_type: sub.klinetype,
          show_level_sub_x1: 1,
          show_level_1x: 1,
          show_level_2x: 1,
          show_level_4x: 1,
          show_level_8x: 1,
        };

        const indicatorResult = await callCalculator(klines, params);

        ws.send(JSON.stringify({
          type: 'indicator_update',
          data: {
            market: sub.market,
            code: sub.code,
            klinetype: sub.klinetype,
            klines,
            indicators: [indicatorResult],
          },
          timestamp: new Date().toISOString(),
        }));
      } catch (err) {
        console.error('WebSocket update error:', err.message);
      }
    }
  }
}, 30000);

// ============================================================================
// Serve frontend static files
// ============================================================================

const DIST_PATH = path.resolve(__dirname, '../kline-indicator-client/dist');
app.use(express.static(DIST_PATH));

// SPA fallback: any non-API route serves index.html
app.get('*', (_req, res) => {
  res.sendFile(path.join(DIST_PATH, 'index.html'));
});

// ============================================================================
// Start server
// ============================================================================

server.listen(PORT, '0.0.0.0', () => {
  console.log(`Moshi Chanlun Server running on http://0.0.0.0:${PORT}`);
  console.log(`Frontend: http://localhost:${PORT}/`);
  console.log(`WebSocket: ws://localhost:${PORT}/ws/kline`);
  console.log(`Calculator CLI: ${CLI_PATH}`);
  console.log('---');
  console.log('API Endpoints:');
  console.log(`  GET  /api/v1/health`);
  console.log(`  GET  /api/v1/kline?market=0&code=000001&klinetype=10&count=2000`);
  console.log(`  POST /api/v1/indicators/calculate`);
  console.log(`  GET  /api/v1/indicators/list`);
});
