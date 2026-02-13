import axios, { AxiosInstance } from 'axios';
import type { 
  KLineRequest, 
  KLineResponse, 
  IndicatorCalculateRequest, 
  IndicatorCalculateResponse,
  IndicatorListResponse
} from '../types';

// 创建axios实例
const api: AxiosInstance = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// 请求拦截器
api.interceptors.request.use(
  (config) => {
    // 添加请求ID
    config.headers['X-Request-ID'] = generateRequestId();
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// 响应拦截器
api.interceptors.response.use(
  (response) => {
    return response;
  },
  (error) => {
    console.error('API Error:', error);
    return Promise.reject(error);
  }
);

// 生成请求ID
function generateRequestId(): string {
  return Date.now().toString(36) + Math.random().toString(36).substring(2, 11);
}

// 获取K线数据
export async function fetchKLineData(params: KLineRequest): Promise<KLineResponse> {
  const response = await api.get<KLineResponse>('/kline', { params });
  return response.data;
}

// 计算指标
export async function calculateIndicators(
  request: IndicatorCalculateRequest
): Promise<IndicatorCalculateResponse> {
  const response = await api.post<IndicatorCalculateResponse>('/indicators/calculate', request);
  return response.data;
}

// 获取指标列表
export async function fetchIndicatorList(): Promise<IndicatorListResponse> {
  const response = await api.get<IndicatorListResponse>('/indicators/list');
  return response.data;
}

// 健康检查
export async function healthCheck(): Promise<{ status: string }> {
  const response = await api.get('/health');
  return response.data;
}

export default api;
