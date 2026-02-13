import { useState, useEffect, useCallback, useRef } from 'react';
import KLineChart from './components/Chart/KLineChart';
import TimeframeSwitcher from './components/Chart/TimeframeSwitcher';
import ExportDialog from './components/DataExport/ExportDialog';
import NetworkStatus from './components/StatusBar/NetworkStatus';
import { fetchKLineData, calculateIndicators } from './services/api';
import wsService from './services/websocket';
import type { 
  KLine, 
  KLineType, 
  WeightType, 
  IndicatorConfig, 
  IndicatorResult, 
} from './types';

// 莫氏缠论默认配置：四个级别全部开启，sub-x1默认关闭
const MOSHI_CHANLUN_CONFIG: IndicatorConfig = {
  type: 'moshi_chanlun',
  params: {
    kline_type: 10,
    show_level_sub_x1: true,
    show_level_1x: true,
    show_level_2x: true,
    show_level_4x: true,
    show_level_8x: true,
  },
};

function App() {
  // 股票信息
  const [market] = useState(0);
  const [stockCode, setStockCode] = useState('000001');
  const [stockInput, setStockInput] = useState('000001');
  
  // K线参数
  const [timeframe, setTimeframe] = useState<KLineType>(10);
  const [weight] = useState<WeightType>(0);
  const [count, setCount] = useState(2000);
  const [countInput, setCountInput] = useState('2000');
  
  // 数据
  const [klines, setKlines] = useState<KLine[]>([]);
  const [indicators, setIndicators] = useState<IndicatorResult[]>([]);
  
  // 状态
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [apiConnected, setApiConnected] = useState(false);
  const [showExport, setShowExport] = useState(false);
  // 每次WebSocket (重)连接时递增，触发重新订阅
  const [wsSessionId, setWsSessionId] = useState(0);

  // 用ref保存当前参数，供WebSocket回调引用最新值
  const paramsRef = useRef({ market, stockCode, timeframe });
  // 标记klines更新来源是否为WebSocket，避免触发冗余REST指标计算
  const wsKlineUpdateRef = useRef(false);

  // 加载K线数据
  const loadKLineData = useCallback(async () => {
    setLoading(true);
    setError(null);
    
    try {
      const res = await fetchKLineData({
        market,
        code: stockCode,
        klinetype: timeframe,
        weight,
        count,
      });
      
      if (res.code === 0 && res.data) {
        setKlines(res.data.klines);
        setApiConnected(true);
      } else {
        setError(res.message || '获取K线数据失败');
      }
    } catch (err: unknown) {
      let msg = '获取K线数据失败';
      if (err && typeof err === 'object' && 'response' in err) {
        const axiosErr = err as { response?: { data?: { message?: string } } };
        if (axiosErr.response?.data?.message) {
          msg = axiosErr.response.data.message;
        }
      } else if (err && typeof err === 'object' && 'message' in err) {
        const e = err as { message: string };
        if (e.message.includes('Network Error')) {
          msg = '无法连接后端服务，请确认服务已启动 (端口8080)';
        }
      }
      setError(msg);
      setApiConnected(false);
      console.error(err);
    } finally {
      setLoading(false);
    }
  }, [market, stockCode, timeframe, weight, count]);

  // 计算莫氏缠论指标
  const loadIndicators = useCallback(async () => {
    if (klines.length === 0) {
      setIndicators([]);
      return;
    }

    // WebSocket推送已同步klines+indicators，跳过冗余的REST API调用
    if (wsKlineUpdateRef.current) {
      wsKlineUpdateRef.current = false;
      return;
    }
    
    // 同步 kline_type 参数与当前时间周期
    const config: IndicatorConfig = {
      ...MOSHI_CHANLUN_CONFIG,
      params: { ...MOSHI_CHANLUN_CONFIG.params, kline_type: timeframe },
    };

    try {
      const res = await calculateIndicators({
        market,
        code: stockCode,
        klinetype: timeframe,
        weight,
        count,
        indicators: [config],
      });
      
      if (res.code === 0) {
        setIndicators(res.indicators);
      }
    } catch (err) {
      console.error('计算指标失败:', err);
      setIndicators([]);
    }
  }, [market, stockCode, timeframe, weight, count, klines.length]);

  // 加载数据
  useEffect(() => {
    loadKLineData();
  }, [loadKLineData]);

  // 计算指标
  useEffect(() => {
    loadIndicators();
  }, [loadIndicators]);

  // 同步paramsRef
  useEffect(() => {
    paramsRef.current = { market, stockCode, timeframe };
  }, [market, stockCode, timeframe]);

  // WebSocket连接与indicator_update监听
  useEffect(() => {
    wsService.connect()
      .catch((err) => console.warn('WebSocket连接失败:', err));

    const unsubIndicator = wsService.on('indicator_update', (data: unknown) => {
      const update = data as {
        market: number;
        code: string;
        klinetype: number;
        klines?: KLine[];
        indicators: IndicatorResult[];
      };
      const { market: m, stockCode: sc, timeframe: tf } = paramsRef.current;
      if (update.code === sc && update.klinetype === tf && update.market === m) {
        wsKlineUpdateRef.current = true;
        if (update.klines && update.klines.length > 0) {
          setKlines(update.klines);
        }
        setIndicators(update.indicators);
      }
    });

    // 每次收到connected消息（包括重连后），递增sessionId触发重新订阅
    const unsubConnected = wsService.on('connected', () => {
      setWsSessionId((prev) => prev + 1);
    });

    return () => {
      unsubIndicator();
      unsubConnected();
    };
  }, []);

  // WebSocket订阅管理：连接建立或股票/周期变化时重新订阅
  useEffect(() => {
    if (wsSessionId === 0) return;
    wsService.subscribe([{ market, code: stockCode, klinetype: timeframe }]);
    return () => {
      wsService.unsubscribe([{ market, code: stockCode, klinetype: timeframe }]);
    };
  }, [market, stockCode, timeframe, wsSessionId]);

  // 处理股票代码搜索
  const handleSearch = () => {
    if (stockInput.trim()) {
      setStockCode(stockInput.trim());
    }
  };

  // 处理K线数量变更
  const handleCountChange = (value: number) => {
    if (value >= 10 && value <= 5000) {
      setCount(value);
      setCountInput(String(value));
    }
  };

  const handleCountInputConfirm = () => {
    const val = parseInt(countInput, 10);
    if (!isNaN(val) && val >= 10 && val <= 5000) {
      setCount(val);
    } else {
      setCountInput(String(count));
    }
  };

  return (
    <div className="min-h-screen bg-gray-900 text-white">
      {/* 顶部工具栏 */}
      <header className="bg-gray-800 p-4 flex items-center justify-between">
        <div className="flex items-center gap-4">
          <h1 className="text-xl font-bold">莫氏缠论画线系统</h1>
          
          {/* 股票搜索 */}
          <div className="flex items-center gap-2">
            <input
              type="text"
              value={stockInput}
              onChange={(e) => setStockInput(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
              placeholder="输入股票代码"
              className="px-3 py-1 bg-gray-700 text-white rounded border border-gray-600 focus:border-blue-500 focus:outline-none w-28"
            />
            <button
              onClick={handleSearch}
              className="px-4 py-1 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors"
            >
              查询
            </button>
          </div>
          
          {/* 时间周期切换 */}
          <TimeframeSwitcher
            currentTimeframe={timeframe}
            onTimeframeChange={setTimeframe}
          />
          
          {/* K线数量选择 */}
          <div className="flex items-center gap-1">
            <span className="text-gray-400 text-sm">数量:</span>
            {[200, 500, 1000, 2000].map((n) => (
              <button
                key={n}
                onClick={() => handleCountChange(n)}
                className={`px-2 py-1 text-sm rounded transition-colors ${
                  count === n
                    ? 'bg-blue-600 text-white'
                    : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
                }`}
              >
                {n}
              </button>
            ))}
            <input
              type="number"
              value={countInput}
              onChange={(e) => setCountInput(e.target.value)}
              onBlur={handleCountInputConfirm}
              onKeyDown={(e) => e.key === 'Enter' && handleCountInputConfirm()}
              min={10}
              max={5000}
              className="w-16 px-2 py-1 text-sm bg-gray-700 text-white rounded border border-gray-600 focus:border-blue-500 focus:outline-none text-center"
            />
          </div>
        </div>
        
        <div className="flex items-center gap-4">
          {/* 导出按钮 */}
          <button
            onClick={() => setShowExport(true)}
            className="px-4 py-1 bg-gray-600 text-white rounded hover:bg-gray-500 transition-colors"
          >
            导出数据
          </button>
          
          {/* 刷新按钮 */}
          <button
            onClick={loadKLineData}
            disabled={loading}
            className="px-4 py-1 bg-green-600 text-white rounded hover:bg-green-700 transition-colors disabled:opacity-50"
          >
            {loading ? '加载中...' : '刷新'}
          </button>
          
          {/* 网络状态 */}
          <NetworkStatus apiConnected={apiConnected} />
        </div>
      </header>

      {/* 主内容区 */}
      <main className="p-4">
        {error ? (
          <div className="bg-red-900 text-red-200 p-4 rounded-lg">
            错误: {error}
          </div>
        ) : (
          <div className="bg-gray-800 p-4 rounded-lg relative">
            <div className="mb-2 text-gray-400 flex items-center gap-4">
              <span>{stockCode} - K线数量: {klines.length}</span>
              {indicators.length > 0 && indicators[0].extra?.levels && (
                <span className="text-xs">
                  莫氏缠论级别: {indicators[0].extra.levels.map((l) => l === 0 ? 'sub-x1' : `x${l}`).join(' / ')}
                </span>
              )}
              {indicators.length > 0 && (
                <span className="text-xs text-gray-500">
                  计算耗时: {indicators[0].computation_time_ms}ms
                </span>
              )}
            </div>
            <KLineChart 
              klines={klines} 
              indicators={indicators}
              height={700}
            />
            {loading && (
              <div className="absolute inset-0 bg-gray-900/60 flex items-center justify-center rounded-lg">
                <div className="text-gray-200 text-lg">加载数据中...</div>
              </div>
            )}
          </div>
        )}
      </main>

      {/* 导出对话框 */}
      <ExportDialog
        isOpen={showExport}
        onClose={() => setShowExport(false)}
        klines={klines}
        indicators={indicators}
        stockCode={stockCode}
      />
    </div>
  );
}

export default App;
