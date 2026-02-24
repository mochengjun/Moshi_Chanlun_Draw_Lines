// K线类型 (基于外部API reqtype=150 实际测试验证)
// 1:1分钟 2:5分钟 4:3分钟 5:15分钟 6:30分钟 3:60分钟 8:120分钟 7:半日线 10:日K 11:周K 20:月K 21:季K 30:年K
export type KLineType = 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 10 | 11 | 20 | 21 | 30;

// K线类型名称映射
export const KLineTypeNames: Record<KLineType, string> = {
  1: '1分钟',
  4: '3分钟',
  2: '5分钟',
  5: '15分钟',
  6: '30分钟',
  3: '60分钟',
  8: '120分钟',
  7: '半日线',
  10: '日K',
  11: '周K',
  20: '月K',
  21: '季K',
  30: '年K',
};

// 复权类型
export type WeightType = 0 | 1 | 2;

export const WeightTypeNames: Record<WeightType, string> = {
  0: '不复权',
  1: '前复权',
  2: '后复权',
};

// K线数据
export interface KLine {
  timestamp: string;
  open: number;
  high: number;
  low: number;
  close: number;
  volume: number;
  amount?: number;
}

// K线请求参数
export interface KLineRequest {
  market: number;
  code: string;
  klinetype: KLineType;
  weight: WeightType;
  count: number;
  endtime?: string;
}

// K线数据响应
export interface KLineData {
  market: number;
  code: string;
  name?: string;
  klinetype: KLineType;
  weight: WeightType;
  klines: KLine[];
  count: number;
}

// K线响应
export interface KLineResponse {
  code: number;
  message: string;
  data?: KLineData;
  cache_hit: boolean;
}
