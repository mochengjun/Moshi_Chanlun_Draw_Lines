import { useState } from 'react';
import type { KLine, IndicatorResult, BiMarker, MoshiMarkPoint } from '../../types';

interface ExportDialogProps {
  isOpen: boolean;
  onClose: () => void;
  klines: KLine[];
  indicators: IndicatorResult[];
  stockCode: string;
}

type ExportFormat = 'csv' | 'json';

// 统计指标数据摘要
function getIndicatorSummary(indicators: IndicatorResult[]): string[] {
  const lines: string[] = [];
  indicators.forEach((ind) => {
    const parts: string[] = [ind.name || ind.type];
    if (ind.series?.length) parts.push(`${ind.series.length}条线`);
    if (ind.fractal_markers?.length) parts.push(`${ind.fractal_markers.length}个标注点`);
    if (ind.bi_markers?.length) parts.push(`${ind.bi_markers.length}条笔`);
    if (ind.extra?.mark_points?.length) parts.push(`${ind.extra.mark_points.length}个详细点`);
    lines.push(parts.join(' / '));
  });
  return lines;
}

export default function ExportDialog({
  isOpen,
  onClose,
  klines,
  indicators,
  stockCode,
}: ExportDialogProps) {
  const [format, setFormat] = useState<ExportFormat>('csv');
  const [includeIndicators, setIncludeIndicators] = useState(true);

  if (!isOpen) return null;

  const handleExport = () => {
    let content: string;
    let filename: string;
    let mimeType: string;

    if (format === 'csv') {
      content = generateCSV(klines, indicators, includeIndicators);
      filename = `${stockCode}_kline_${Date.now()}.csv`;
      mimeType = 'text/csv;charset=utf-8';
    } else {
      content = generateJSON(klines, indicators, includeIndicators, stockCode);
      filename = `${stockCode}_kline_${Date.now()}.json`;
      mimeType = 'application/json;charset=utf-8';
    }

    // 添加 UTF-8 BOM 以确保中文在 Excel 中正确显示
    const bom = '\uFEFF';
    downloadFile(bom + content, filename, mimeType);
    onClose();
  };

  const summaryLines = getIndicatorSummary(indicators);

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-gray-800 p-6 rounded-lg w-96">
        <h3 className="text-white text-lg font-semibold mb-4">导出数据</h3>

        <div className="space-y-4">
          {/* 格式选择 */}
          <div>
            <label className="text-gray-300 text-sm block mb-2">导出格式</label>
            <div className="flex gap-4">
              <label className="flex items-center text-gray-300">
                <input
                  type="radio"
                  name="format"
                  value="csv"
                  checked={format === 'csv'}
                  onChange={() => setFormat('csv')}
                  className="mr-2"
                />
                CSV
              </label>
              <label className="flex items-center text-gray-300">
                <input
                  type="radio"
                  name="format"
                  value="json"
                  checked={format === 'json'}
                  onChange={() => setFormat('json')}
                  className="mr-2"
                />
                JSON
              </label>
            </div>
          </div>

          {/* 包含指标 */}
          <div>
            <label className="flex items-center text-gray-300">
              <input
                type="checkbox"
                checked={includeIndicators}
                onChange={(e) => setIncludeIndicators(e.target.checked)}
                className="mr-2"
              />
              包含指标数据
            </label>
          </div>

          {/* 数据预览 */}
          <div className="text-gray-400 text-sm space-y-1">
            <p>K线数据: {klines.length} 条</p>
            <p>指标数据: {indicators.length} 个</p>
            {summaryLines.length > 0 && (
              <div className="mt-1 pl-2 border-l-2 border-gray-600">
                {summaryLines.map((line, i) => (
                  <p key={i} className="text-xs text-gray-500">{line}</p>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* 按钮 */}
        <div className="flex gap-3 mt-6">
          <button
            onClick={onClose}
            className="flex-1 py-2 bg-gray-600 text-white rounded hover:bg-gray-500 transition-colors"
          >
            取消
          </button>
          <button
            onClick={handleExport}
            className="flex-1 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors"
          >
            导出
          </button>
        </div>
      </div>
    </div>
  );
}

// 转义 CSV 字段
function escapeCSV(value: string): string {
  if (value.includes(',') || value.includes('"') || value.includes('\n')) {
    return `"${value.replace(/"/g, '""')}"`;
  }
  return value;
}

// 生成CSV内容
function generateCSV(
  klines: KLine[],
  indicators: IndicatorResult[],
  includeIndicators: boolean
): string {
  const sections: string[] = [];

  // === 第一部分：K线数据 + 线形指标 + 标注点 ===
  const headers = ['时间', '开盘', '最高', '最低', '收盘', '成交量'];

  // 收集所有需要按K线索引映射的数据
  const seriesColumns: { name: string; getVal: (idx: number) => string }[] = [];
  // 按索引映射的标注点: index → MarkPoint[]
  const markPointsByIndex = new Map<number, MoshiMarkPoint[]>();
  let hasMoshiData = false;

  if (includeIndicators) {
    indicators.forEach((ind) => {
      // 线形数据列
      if (ind.series) {
        ind.series.forEach((s) => {
          seriesColumns.push({
            name: `${ind.name || ind.type}_${s.name}`,
            getVal: (idx) => {
              const v = s.values[idx]?.value;
              return v != null && v !== 0 ? v.toString() : '';
            },
          });
        });
      }

      // 莫氏缠论标注点按索引映射
      if (ind.type === 'moshi_chanlun' && ind.extra?.mark_points) {
        hasMoshiData = true;
        ind.extra.mark_points.forEach((mp) => {
          const list = markPointsByIndex.get(mp.index) || [];
          list.push(mp);
          markPointsByIndex.set(mp.index, list);
        });
      }
    });

    seriesColumns.forEach((col) => headers.push(col.name));

    if (hasMoshiData) {
      headers.push('莫氏缠论_标注', '莫氏缠论_价格', '莫氏缠论_级别', '莫氏缠论_倍数');
    }
  }

  const rows = klines.map((k, idx) => {
    const row = [
      escapeCSV(k.timestamp),
      k.open.toString(),
      k.high.toString(),
      k.low.toString(),
      k.close.toString(),
      k.volume.toString(),
    ];

    if (includeIndicators) {
      seriesColumns.forEach((col) => row.push(col.getVal(idx)));

      if (hasMoshiData) {
        const mps = markPointsByIndex.get(idx);
        if (mps && mps.length > 0) {
          // 合并同一索引的多个标注点
          const types = mps.map((m) => m.type).join('/');
          const prices = mps.map((m) => m.price).join('/');
          const levels = mps.map((m) => m.level).join('/');
          const mults = mps.map((m) => `x${m.multiplier}`).join('/');
          row.push(types, prices, escapeCSV(levels), mults);
        } else {
          row.push('', '', '', '');
        }
      }
    }

    return row.join(',');
  });

  sections.push([headers.join(','), ...rows].join('\n'));

  // === 第二部分：笔标记数据 ===
  if (includeIndicators) {
    const allBiMarkers: (BiMarker & { indicatorName: string })[] = [];
    indicators.forEach((ind) => {
      if (ind.bi_markers && ind.bi_markers.length > 0) {
        ind.bi_markers.forEach((bi) => {
          allBiMarkers.push({ ...bi, indicatorName: ind.name || ind.type });
        });
      }
    });

    if (allBiMarkers.length > 0) {
      const biHeaders = [
        '指标', '方向', '起点时间', '起点价格', '终点时间', '终点价格',
        'K线数', '上涨K线', '下跌K线', '级别倍数',
      ];

      const biRows = allBiMarkers.map((bi) => [
        escapeCSV(bi.indicatorName),
        bi.direction === 'up' || bi.direction === 'UP' ? '上涨' : '下跌',
        escapeCSV(bi.start_timestamp),
        bi.start_price.toString(),
        escapeCSV(bi.end_timestamp),
        bi.end_price.toString(),
        (bi.length ?? '').toString(),
        (bi.up_count ?? '').toString(),
        (bi.down_count ?? '').toString(),
        bi.multiplier ? `x${bi.multiplier}` : '',
      ].join(','));

      sections.push('');
      sections.push('=== 笔标记数据 ===');
      sections.push([biHeaders.join(','), ...biRows].join('\n'));
    }
  }

  return sections.join('\n');
}

// 生成JSON内容
function generateJSON(
  klines: KLine[],
  indicators: IndicatorResult[],
  includeIndicators: boolean,
  stockCode: string
): string {
  const data: Record<string, unknown> = {
    stock_code: stockCode,
    export_time: new Date().toISOString(),
    kline_count: klines.length,
    klines,
  };

  if (includeIndicators && indicators.length > 0) {
    data.indicators = indicators;

    // 为莫氏缠论单独提取结构化数据，方便下游使用
    const moshiIndicators = indicators.filter((i) => i.type === 'moshi_chanlun');
    if (moshiIndicators.length > 0) {
      const moshi = moshiIndicators[0];
      data.moshi_chanlun_summary = {
        mark_points: moshi.extra?.mark_points || [],
        bi_markers: moshi.bi_markers || [],
        fractal_markers: moshi.fractal_markers || [],
        levels: moshi.extra?.levels || [],
        kline_type: moshi.extra?.kline_type,
      };
    }
  }

  return JSON.stringify(data, null, 2);
}

// 下载文件
function downloadFile(content: string, filename: string, mimeType: string): void {
  const blob = new Blob([content], { type: mimeType });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}
