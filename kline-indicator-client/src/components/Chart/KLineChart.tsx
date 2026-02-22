import { useEffect, useRef, useCallback } from 'react';
import { createChart, IChartApi, ISeriesApi, CandlestickData, LineData, Time, SeriesMarker } from 'lightweight-charts';
import type { KLine, IndicatorResult, FractalMarker, BiMarker, SameLevelTrend } from '../../types';

interface KLineChartProps {
  klines: KLine[];
  indicators?: IndicatorResult[];
  showTrends?: boolean;
  height?: number;
  onCrosshairMove?: (time: Time | null, price: number | null) => void;
}

// 颜色配置
const COLORS = {
  upColor: 'rgba(0,0,0,0)',  // 阳线柱体透明（空心效果）
  downColor: '#00b070',       // 阴线实心绿色
  upBorder: '#ef5350',        // 阳线边框红色
  downBorder: '#00b070',      // 阴线边框绿色
  wickUpColor: '#ef5350',     // 阳线影线红色
  wickDownColor: '#00b070',   // 阴线影线绿色
  ma5: '#f0a000',
  ma10: '#00a0f0',
  ma20: '#f000a0',
  bollMid: '#2962FF',
  bollUpper: '#FF6D00',
  bollLower: '#FF6D00',
  fractalTop: '#ef5350',
  fractalBottom: '#26a69a',
  biUp: '#26a69a',
  biDown: '#ef5350',
};

// 莫氏缠论级别颜色配置
const MOSHI_COLORS: Record<number, { line: string; marker: string; name: string }> = {
  0: { line: '#888888', marker: '#888888', name: 'sub' }, // 灰色 sub-x1
  1: { line: '#FFD700', marker: '#FFD700', name: 'x1' },  // 金色
  2: { line: '#FF6B6B', marker: '#FF6B6B', name: 'x2' },  // 红色
  4: { line: '#4ECDC4', marker: '#4ECDC4', name: 'x4' },  // 青色
  8: { line: '#9B59B6', marker: '#9B59B6', name: 'x8' },  // 紫色
};

// 获取级别线宽
const getLevelLineWidth = (multiplier: number): 1 | 2 | 3 | 4 => {
  switch (multiplier) {
    case 1: return 1;
    case 2: return 2;
    case 4: return 3;
    case 8: return 4;
    default: return 2;
  }
};

export default function KLineChart({ 
  klines, 
  indicators = [],
  showTrends = true,
  height = 600,
  onCrosshairMove 
}: KLineChartProps) {
  const chartContainerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const candlestickSeriesRef = useRef<ISeriesApi<'Candlestick'> | null>(null);
  const lineSeriesRef = useRef<Map<string, ISeriesApi<'Line'>>>(new Map());
  const labelContainerRef = useRef<HTMLDivElement | null>(null);
  const ohlcvOverlayRef = useRef<HTMLDivElement | null>(null);
  const klineMapRef = useRef<Map<number, KLine>>(new Map());
  // 存储走势数据和DOM元素引用，用于缩放/平移时更新位置
  const trendsDataRef = useRef<SameLevelTrend[]>([]);
  const trendElementsRef = useRef<{ bg: HTMLDivElement; label: HTMLDivElement; trend: SameLevelTrend }[]>([]);
  // 存储高低点标签元素引用
  const pointLabelElementsRef = useRef<{ label: HTMLDivElement; time: Time; price: number; isUp: boolean; color: string }[]>([]);

  // 初始化图表
  useEffect(() => {
    if (!chartContainerRef.current) return;

    const containerWidth = chartContainerRef.current.clientWidth;

    const chart = createChart(chartContainerRef.current, {
      width: containerWidth,
      height,
      layout: {
        background: { color: '#1e1e1e' },
        textColor: '#d1d4dc',
      },
      grid: {
        vertLines: { color: '#2B2B43' },
        horzLines: { color: '#2B2B43' },
      },
      crosshair: {
        mode: 1,
      },
      rightPriceScale: {
        borderColor: '#2B2B43',
      },
      timeScale: {
        borderColor: '#2B2B43',
        timeVisible: true,
        secondsVisible: false,
      },
    });

    chartRef.current = chart;

    // 创建K线系列
    const candlestickSeries = chart.addCandlestickSeries({
      upColor: COLORS.upColor,
      downColor: COLORS.downColor,
      wickUpColor: COLORS.wickUpColor,
      wickDownColor: COLORS.wickDownColor,
      borderVisible: true,
      borderUpColor: COLORS.upBorder,
      borderDownColor: COLORS.downBorder,
    });
    candlestickSeriesRef.current = candlestickSeries;

    // 创建标签容器
    const labelContainer = document.createElement('div');
    labelContainer.style.position = 'absolute';
    labelContainer.style.top = '0';
    labelContainer.style.left = '0';
    labelContainer.style.width = '100%';
    labelContainer.style.height = '100%';
    labelContainer.style.pointerEvents = 'none';
    labelContainer.style.overflow = 'hidden';
    chartContainerRef.current.appendChild(labelContainer);
    labelContainerRef.current = labelContainer;

    // 创建OHLCV信息浮层
    const ohlcvOverlay = document.createElement('div');
    ohlcvOverlay.style.cssText = 'position:absolute;top:8px;left:8px;z-index:10;pointer-events:none;' +
      'font-size:12px;font-family:monospace;line-height:1.6;color:#d1d4dc;display:none;';
    chartContainerRef.current.appendChild(ohlcvOverlay);
    ohlcvOverlayRef.current = ohlcvOverlay;

    // 订阅十字线移动事件
    const crosshairHandler = (param: any) => {
      // 更新OHLCV浮层
      if (param.time && param.seriesData.get(candlestickSeries)) {
        const candle = param.seriesData.get(candlestickSeries) as CandlestickData;
        const kline = klineMapRef.current.get(param.time as number);
        const change = candle.close - candle.open;
        const changePct = candle.open !== 0 ? (change / candle.open * 100) : 0;
        const isUp = candle.close >= candle.open;
        const color = isUp ? '#ef5350' : '#00b070';

        let html = `<span style="color:#d1d4dc">${kline ? kline.timestamp : ''} </span>` +
          `<span style="color:${color}">` +
          `O:${candle.open.toFixed(2)} ` +
          `H:${candle.high.toFixed(2)} ` +
          `L:${candle.low.toFixed(2)} ` +
          `C:${candle.close.toFixed(2)} ` +
          `${change >= 0 ? '+' : ''}${change.toFixed(2)}(${changePct >= 0 ? '+' : ''}${changePct.toFixed(2)}%)`;
        if (kline) {
          html += ` V:${formatVolume(kline.volume)}`;
        }
        html += '</span>';
        ohlcvOverlay.innerHTML = html;
        ohlcvOverlay.style.display = 'block';
      } else {
        ohlcvOverlay.style.display = 'none';
      }

      // 调用外部回调
      if (onCrosshairMove) {
        if (param.time) {
          const price = param.seriesData.get(candlestickSeries);
          onCrosshairMove(param.time, price ? (price as CandlestickData).close : null);
        } else {
          onCrosshairMove(null, null);
        }
      }
    };
    chart.subscribeCrosshairMove(crosshairHandler);

    // 使用 ResizeObserver 响应容器尺寸变化
    const resizeObserver = new ResizeObserver((entries) => {
      for (const entry of entries) {
        const { width: newWidth } = entry.contentRect;
        if (newWidth > 0) {
          chart.applyOptions({ width: newWidth });
        }
      }
    });
    resizeObserver.observe(chartContainerRef.current);

    // 订阅时间轴变化事件，用于更新走势区域位置和高低点标签位置
    const updateTrendPositions = () => {
      if (!chartRef.current || !candlestickSeriesRef.current) return;
      
      const timeScale = chartRef.current.timeScale();
      
      // 更新走势区域位置
      trendElementsRef.current.forEach(({ bg, label, trend }) => {
        const startTime = (new Date(trend.start_timestamp).getTime() / 1000) as Time;
        const endTime = (new Date(trend.end_timestamp).getTime() / 1000) as Time;
        
        const startX = timeScale.timeToCoordinate(startTime);
        const endX = timeScale.timeToCoordinate(endTime);
        
        if (startX === null || endX === null) {
          bg.style.display = 'none';
          label.style.display = 'none';
          return;
        }
        
        const topY = candlestickSeriesRef.current!.priceToCoordinate(trend.high_point.price);
        const bottomY = candlestickSeriesRef.current!.priceToCoordinate(trend.low_point.price);
        
        if (topY === null || bottomY === null) {
          bg.style.display = 'none';
          label.style.display = 'none';
          return;
        }
        
        const isUp = trend.type === 'up';
        
        bg.style.display = 'block';
        bg.style.left = `${Math.min(startX, endX)}px`;
        bg.style.width = `${Math.abs(endX - startX)}px`;
        bg.style.top = `${Math.min(topY, bottomY)}px`;
        bg.style.height = `${Math.abs(bottomY - topY)}px`;
        
        label.style.display = 'block';
        label.style.left = `${Math.min(startX, endX) + 4}px`;
        label.style.top = `${isUp ? Math.min(topY, bottomY) + 4 : Math.max(topY, bottomY) - 20}px`;
      });

      // 更新高低点标签位置
      pointLabelElementsRef.current.forEach((item: any) => {
        const { label, time, price, isUp, stackOffset = 0 } = item;
        const x = timeScale.timeToCoordinate(time);
        const y = candlestickSeriesRef.current!.priceToCoordinate(price);
        
        if (x === null || y === null) {
          label.style.display = 'none';
          return;
        }
        
        label.style.display = 'block';
        label.style.left = `${x}px`;
        // 高点标签在K线上方，低点标签在K线下方，加上堆叠偏移
        const baseOffset = isUp ? -20 : 8;
        label.style.top = `${y + baseOffset + stackOffset}px`;
      });
    };

    // 订阅可视范围变化（缩放、平移）
    chart.timeScale().subscribeVisibleTimeRangeChange(updateTrendPositions);
    
    // 订阅尺寸变化
    chart.timeScale().subscribeSizeChange(updateTrendPositions);

    return () => {
      resizeObserver.disconnect();
      chart.timeScale().unsubscribeVisibleTimeRangeChange(updateTrendPositions);
      chart.timeScale().unsubscribeSizeChange(updateTrendPositions);
      chart.unsubscribeCrosshairMove(crosshairHandler);
      if (labelContainerRef.current && chartContainerRef.current?.contains(labelContainerRef.current)) {
        chartContainerRef.current.removeChild(labelContainerRef.current);
      }
      if (ohlcvOverlayRef.current && chartContainerRef.current?.contains(ohlcvOverlayRef.current)) {
        chartContainerRef.current.removeChild(ohlcvOverlayRef.current);
      }
      chart.remove();
      chartRef.current = null;
      candlestickSeriesRef.current = null;
      lineSeriesRef.current.clear();
      labelContainerRef.current = null;
      ohlcvOverlayRef.current = null;
    };
  }, [height, onCrosshairMove]);

  // 更新K线数据
  useEffect(() => {
    if (!candlestickSeriesRef.current || klines.length === 0) return;

    // 构建 time -> KLine 映射（用于OHLCV浮层查找成交量）
    const timeMap = new Map<number, KLine>();
    const candlestickData: CandlestickData[] = klines.map((k) => {
      const t = (new Date(k.timestamp).getTime() / 1000) as Time;
      timeMap.set(t as number, k);
      return {
        time: t,
        open: k.open,
        high: k.high,
        low: k.low,
        close: k.close,
      };
    });
    klineMapRef.current = timeMap;

    candlestickSeriesRef.current.setData(candlestickData);
    chartRef.current?.timeScale().fitContent();
  }, [klines]);

  // 更新指标数据
  useEffect(() => {
    if (!chartRef.current) return;

    // 清除旧的指标线
    lineSeriesRef.current.forEach((series) => {
      chartRef.current?.removeSeries(series);
    });
    lineSeriesRef.current.clear();

    // 清除旧的标签
    if (labelContainerRef.current) {
      labelContainerRef.current.innerHTML = '';
    }
    // 清除走势元素引用
    trendElementsRef.current = [];
    trendsDataRef.current = [];
    // 清除高低点标签元素引用
    pointLabelElementsRef.current = [];

    // 添加新的指标线
    indicators.forEach((indicator) => {
      if (indicator.series) {
        indicator.series.forEach((s, idx) => {
          const color = getSeriesColor(indicator.type, s.name, idx);
          const lineSeries = chartRef.current!.addLineSeries({
            color,
            lineWidth: 1,
            priceLineVisible: false,
            lastValueVisible: false,
          });

          const lineData: LineData[] = s.values
            .filter((v) => v.value !== 0)
            .map((v) => ({
              time: (new Date(v.timestamp).getTime() / 1000) as Time,
              value: v.value,
            }));

          lineSeries.setData(lineData);
          lineSeriesRef.current.set(`${indicator.type}_${s.name}`, lineSeries);
        });
      }

      // 莫氏缠论指标特殊处理
      if (indicator.type === 'moshi_chanlun' && indicator.bi_markers) {
        addMoshiBiMarkers(indicator.bi_markers, klines);
        // 渲染同级别走势区域（受showTrends开关控制）
        if (showTrends && indicator.extra?.same_level_trends) {
          renderSameLevelTrends(indicator.extra.same_level_trends);
        }
      } else {
        // 添加普通分型标记
        if (indicator.fractal_markers) {
          addFractalMarkers(indicator.fractal_markers);
        }
        // 添加普通笔标记
        if (indicator.bi_markers) {
          addBiMarkers(indicator.bi_markers);
        }
      }
    });
  }, [indicators, klines, showTrends]);

  // 添加分型标记
  const addFractalMarkers = useCallback((markers: FractalMarker[]) => {
    if (!candlestickSeriesRef.current) return;

    const seriesMarkers: SeriesMarker<Time>[] = markers.map((m) => {
      const isTop = m.type === 'TOP' || m.type === 'top';
      return {
        time: (new Date(m.timestamp).getTime() / 1000) as Time,
        position: isTop ? 'aboveBar' as const : 'belowBar' as const,
        color: isTop ? COLORS.fractalTop : COLORS.fractalBottom,
        shape: isTop ? 'arrowDown' as const : 'arrowUp' as const,
        text: isTop ? 'H' : 'L',
      };
    });

    candlestickSeriesRef.current.setMarkers(seriesMarkers);
  }, []);

  // 添加普通笔标记（使用线段）
  const addBiMarkers = useCallback((markers: BiMarker[]) => {
    if (!chartRef.current) return;

    markers.forEach((bi, idx) => {
      const isUp = bi.direction === 'UP' || bi.direction === 'up';
      const lineSeries = chartRef.current!.addLineSeries({
        color: isUp ? COLORS.biUp : COLORS.biDown,
        lineWidth: 2,
        lineStyle: 0,
        priceLineVisible: false,
        lastValueVisible: false,
      });

      const lineData: LineData[] = [
        {
          time: (new Date(bi.start_timestamp).getTime() / 1000) as Time,
          value: bi.start_price,
        },
        {
          time: (new Date(bi.end_timestamp).getTime() / 1000) as Time,
          value: bi.end_price,
        },
      ];

      lineSeries.setData(lineData);
      lineSeriesRef.current.set(`bi_${idx}`, lineSeries);
    });
  }, []);

  // 添加莫氏缠论笔标记（按级别分组，显示K线数量）
  const addMoshiBiMarkers = useCallback((markers: BiMarker[], _klineData: KLine[]) => {
    if (!chartRef.current || !candlestickSeriesRef.current || !labelContainerRef.current) return;

    // 按级别分组
    const groupedMarkers: Record<number, BiMarker[]> = {};
    markers.forEach((bi) => {
      const mult = bi.multiplier ?? 1;
      if (!groupedMarkers[mult]) {
        groupedMarkers[mult] = [];
      }
      groupedMarkers[mult].push(bi);
    });

    // 用于创建高低点数字标签
    const timeScale = chartRef.current.timeScale();

    // 收集所有标签信息，用于处理重叠
    const labelInfos: {
      time: Time;
      price: number;
      isUp: boolean;
      mult: number;
      text: string;
      color: string;
    }[] = [];

    // 按级别从小到大排序处理（小级别线在下层）
    const sortedMultipliers = Object.keys(groupedMarkers).map(Number).sort((a, b) => a - b);

    sortedMultipliers.forEach((mult) => {
      const biList = groupedMarkers[mult];
      const colorConfig = MOSHI_COLORS[mult] || MOSHI_COLORS[1];
      const lineWidth = getLevelLineWidth(mult);

      biList.forEach((bi, idx) => {
        // 跳过双向突破产生的零长度段（起止时间相同）
        const startTime = new Date(bi.start_timestamp).getTime();
        const endTime = new Date(bi.end_timestamp).getTime();
        if (startTime === endTime) return;

        // sub-x1级别(multiplier=0)：画虚线连接线 + 长度数值标注
        if (mult === 0) {
          const lineSeries = chartRef.current!.addLineSeries({
            color: colorConfig.line,
            lineWidth: 1,
            lineStyle: 2, // Dashed
            priceLineVisible: false,
            lastValueVisible: false,
            crosshairMarkerVisible: false,
          });

          const lineData: LineData[] = [
            {
              time: (new Date(bi.start_timestamp).getTime() / 1000) as Time,
              value: bi.start_price,
            },
            {
              time: (new Date(bi.end_timestamp).getTime() / 1000) as Time,
              value: bi.end_price,
            },
          ];

          lineSeries.setData(lineData);
          lineSeriesRef.current.set(`moshi_bi_${mult}_${idx}`, lineSeries);
        } else {
          // 添加连接线（仅 x1 及以上级别）
          const lineSeries = chartRef.current!.addLineSeries({
            color: colorConfig.line,
            lineWidth: lineWidth,
            lineStyle: 0,
            priceLineVisible: false,
            lastValueVisible: false,
            crosshairMarkerVisible: false,
          });

          const lineData: LineData[] = [
            {
              time: (new Date(bi.start_timestamp).getTime() / 1000) as Time,
              value: bi.start_price,
            },
            {
              time: (new Date(bi.end_timestamp).getTime() / 1000) as Time,
              value: bi.end_price,
            },
          ];

          lineSeries.setData(lineData);
          lineSeriesRef.current.set(`moshi_bi_${mult}_${idx}`, lineSeries);
        }

        // 收集标签信息
        const isUp = bi.direction === 'UP' || bi.direction === 'up';
        const totalCount = bi.length || ((bi.up_count || 0) + (bi.down_count || 0));
        const time = (new Date(bi.end_timestamp).getTime() / 1000) as Time;

        labelInfos.push({
          time,
          price: bi.end_price,
          isUp,
          mult,
          text: `${totalCount}`,
          color: colorConfig.marker,
        });
      });
    });

    // 按位置分组处理重叠标签（相同时间+方向为一组）
    const positionGroups: Map<string, typeof labelInfos> = new Map();
    labelInfos.forEach((info) => {
      const key = `${info.time}_${info.isUp ? 'up' : 'down'}`;
      if (!positionGroups.has(key)) {
        positionGroups.set(key, []);
      }
      positionGroups.get(key)!.push(info);
    });

    // 为每组标签创建元素，处理垂直偏移
    positionGroups.forEach((group) => {
      // 按级别从大到小排序（高级别在最外侧）
      group.sort((a, b) => b.mult - a.mult);

      group.forEach((info, groupIdx) => {
        const labelEl = document.createElement('div');
        labelEl.style.cssText = `
          position: absolute;
          color: ${info.color};
          font-size: 10px;
          font-weight: bold;
          pointer-events: none;
          z-index: 3;
          transform: translateX(-50%);
          text-shadow: 0 0 2px #1e1e1e, 0 0 2px #1e1e1e;
        `;
        labelEl.textContent = info.text;

        const x = timeScale.timeToCoordinate(info.time);
        const y = candlestickSeriesRef.current!.priceToCoordinate(info.price);

        if (x !== null && y !== null) {
          // 计算垂直偏移：每个标签错开14px
          const baseOffset = info.isUp ? -20 : 8;
          const stackOffset = info.isUp ? -groupIdx * 14 : groupIdx * 14;
          labelEl.style.left = `${x}px`;
          labelEl.style.top = `${y + baseOffset + stackOffset}px`;
        } else {
          labelEl.style.display = 'none';
        }

        labelContainerRef.current!.appendChild(labelEl);

        // 保存元素引用，包含堆叠偏移信息
        pointLabelElementsRef.current.push({
          label: labelEl,
          time: info.time,
          price: info.price,
          isUp: info.isUp,
          color: info.color,
          stackOffset: info.isUp ? -groupIdx * 14 : groupIdx * 14,
        } as any);
      });
    });
  }, []);

  // 渲染x1同级别走势区域标记
  const renderSameLevelTrends = useCallback((trends: SameLevelTrend[]) => {
    if (!chartRef.current || !labelContainerRef.current || trends.length === 0) return;

    // 清空之前的元素引用
    trendElementsRef.current = [];
    // 保存走势数据
    trendsDataRef.current = trends;

    trends.forEach((trend) => {
      const isUp = trend.type === 'up';
      
      // 根据形态类型设置不同颜色
      let color: string;
      let borderColor: string;
      let labelText: string;
      
      switch (trend.pattern) {
        case 'convergent':
          // 收敛型中枢：橙色（升级后使用金色边框）
          if (trend.upgraded) {
            color = 'rgba(255, 193, 7, 0.20)';  // 金色背景（升级）
            borderColor = '#ffc107';
            labelText = isUp ? '↗收敛中枢⬆' : '↘收敛中枢⬆';  // 添加升级标记
          } else {
            color = 'rgba(255, 152, 0, 0.15)';
            borderColor = '#ff9800';
            labelText = isUp ? '↗收敛中枢' : '↘收敛中枢';
          }
          break;
        case 'divergent':
          // 扩张型中枢：紫色
          color = 'rgba(156, 39, 176, 0.15)';
          borderColor = '#9c27b0';
          labelText = isUp ? '↗扩张中枢' : '↘扩张中枢';
          break;
        default:
          // 趋势型：绿色/红色
          color = isUp ? 'rgba(38, 166, 154, 0.15)' : 'rgba(239, 83, 80, 0.15)';
          borderColor = isUp ? '#26a69a' : '#ef5350';
          labelText = isUp ? '↗上涨' : '↘下跌';
      }
      
      // 创建走势区域背景层
      const trendBg = document.createElement('div');
      trendBg.className = 'trend-region';
      trendBg.style.cssText = `
        position: absolute;
        background: ${color};
        pointer-events: none;
        z-index: 1;
      `;

      // 计算区域位置（需要转换时间戳到图表坐标）
      const startTime = (new Date(trend.start_timestamp).getTime() / 1000) as Time;
      const endTime = (new Date(trend.end_timestamp).getTime() / 1000) as Time;
      
      // 使用图表API获取坐标
      const timeScale = chartRef.current!.timeScale();
      
      const startX = timeScale.timeToCoordinate(startTime);
      const endX = timeScale.timeToCoordinate(endTime);
      
      // 创建走势方向标签（先创建，后面统一设置位置）
      const label = document.createElement('div');
      label.style.cssText = `
        position: absolute;
        color: ${borderColor};
        font-size: 11px;
        font-weight: bold;
        pointer-events: none;
        z-index: 2;
      `;
      label.textContent = labelText;

      // 如果坐标有效，设置初始位置
      if (startX !== null && endX !== null) {
        const topY = candlestickSeriesRef.current!.priceToCoordinate(trend.high_point.price);
        const bottomY = candlestickSeriesRef.current!.priceToCoordinate(trend.low_point.price);

        if (topY !== null && bottomY !== null) {
          trendBg.style.left = `${Math.min(startX, endX)}px`;
          trendBg.style.width = `${Math.abs(endX - startX)}px`;
          trendBg.style.top = `${Math.min(topY, bottomY)}px`;
          trendBg.style.height = `${Math.abs(bottomY - topY)}px`;

          label.style.left = `${Math.min(startX, endX) + 4}px`;
          label.style.top = `${isUp ? Math.min(topY, bottomY) + 4 : Math.max(topY, bottomY) - 20}px`;
        } else {
          trendBg.style.display = 'none';
          label.style.display = 'none';
        }
      } else {
        trendBg.style.display = 'none';
        label.style.display = 'none';
      }

      labelContainerRef.current!.appendChild(trendBg);
      labelContainerRef.current!.appendChild(label);

      // 保存元素引用，用于后续更新位置
      trendElementsRef.current.push({ bg: trendBg, label, trend });
    });
  }, []);

  return (
    <div 
      ref={chartContainerRef} 
      className="kline-chart"
      style={{ width: '100%', height: `${height}px`, position: 'relative' }}
    />
  );
}

// 获取指标线颜色
function getSeriesColor(type: string, name: string, idx: number): string {
  if (type === 'ma' || type === 'ema') {
    const colors = [COLORS.ma5, COLORS.ma10, COLORS.ma20];
    return colors[idx % colors.length];
  }
  if (type === 'boll') {
    if (name === 'MID') return COLORS.bollMid;
    return COLORS.bollUpper;
  }
  // 默认颜色
  const defaultColors = ['#2962FF', '#FF6D00', '#2E7D32', '#D50000'];
  return defaultColors[idx % defaultColors.length];
}

// 格式化成交量（大数简写）
function formatVolume(vol: number): string {
  if (vol >= 1e8) return (vol / 1e8).toFixed(2) + '亿';
  if (vol >= 1e4) return (vol / 1e4).toFixed(0) + '万';
  return vol.toFixed(0);
}
