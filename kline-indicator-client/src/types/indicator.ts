// 指标类型
export type IndicatorType = 'moshi_chanlun';

// 指标分类
export type IndicatorCategory = 'chanlun';

// 参数定义
export interface ParameterDef {
  name: string;
  type: 'int' | 'float' | 'bool' | 'string';
  required: boolean;
  default_value: number | string | boolean;
  min?: number;
  max?: number;
  description: string;
}

// 指标元信息
export interface IndicatorMetadata {
  type: IndicatorType;
  name: string;
  category: IndicatorCategory;
  description: string;
  params_def: ParameterDef[];
}

// 指标配置
export interface IndicatorConfig {
  type: IndicatorType;
  params: Record<string, number | string | boolean>;
}

// 指标值
export interface IndicatorValue {
  timestamp: string;
  value: number;
}

// 指标序列
export interface IndicatorSeries {
  name: string;
  values: IndicatorValue[];
}

// 分型类型
export type FractalType = 'top' | 'bottom' | 'TOP' | 'BOTTOM';

// 分型标记
export interface FractalMarker {
  index: number;
  timestamp: string;
  type: FractalType;
  price: number;
  zone?: [number, number];
}

// 笔方向
export type BiDirection = 'up' | 'down' | 'UP' | 'DOWN';

// 笔标记
export interface BiMarker {
  start_index: number;
  end_index: number;
  start_timestamp: string;
  end_timestamp: string;
  start_price: number;
  end_price: number;
  direction: BiDirection;
  length?: number;
  up_count?: number;    // 上涨K线数量
  down_count?: number;  // 下跌K线数量
  multiplier?: number;  // 级别倍数 0/1/2/4/8
  actual_retrace_time?: number; // 实际回调/反弹时间(分钟)
}

// 莫氏缠论标注点
export interface MoshiMarkPoint {
  type: 'L' | 'H';
  index: number;
  timestamp: string;
  price: number;
  level: string;
  multiplier: number;
}

// x1同级别走势
// pattern: "trend"(趋势), "convergent"(收敛型中枢), "divergent"(扩张型中枢)
export interface SameLevelTrend {
  type: 'up' | 'down';                          // 走势方向
  pattern: 'trend' | 'convergent' | 'divergent'; // 形态类型
  multiplier: number;                            // 走势所属级别: 1(x1), 2(x2), 4(x4), 8(x8)
  start_index: number;                          // 起始K线索引
  end_index: number;                            // 结束K线索引
  start_timestamp: string;                      // 起始时间
  end_timestamp: string;                        // 结束时间
  high_point: MoshiMarkPoint;                   // 走势最高点（上涨=最后H，下跌=第一H）
  low_point: MoshiMarkPoint;                    // 走势最低点（上涨=第一L，下跌=最后L）
  points: MoshiMarkPoint[];                     // 组成走势的所有L/H点序列
  // 级别升级相关字段
  upgraded?: boolean;                           // 是否已升级为父级别
  parent_points?: MoshiMarkPoint[];             // 升级后的父级别点序列
}

// 指标计算结果
export interface IndicatorResult {
  type: IndicatorType;
  name?: string;
  series?: IndicatorSeries[];
  fractal_markers?: FractalMarker[];
  bi_markers?: BiMarker[];
  extra?: {
    mark_points?: MoshiMarkPoint[];
    kline_type?: number;
    levels?: number[];
    same_level_trends?: SameLevelTrend[];  // x1同级别走势
  };
  computation_time_ms: number;
}

// 指标计算请求
export interface IndicatorCalculateRequest {
  market: number;
  code: string;
  klinetype: number;
  weight: number;
  count: number;
  indicators: IndicatorConfig[];
}

// 指标计算响应
export interface IndicatorCalculateResponse {
  code: number;
  message: string;
  stock_code: string;
  indicators: IndicatorResult[];
  computation_time_ms: number;
}

// 指标列表响应
export interface IndicatorListResponse {
  code: number;
  message: string;
  indicators: IndicatorMetadata[];
}
