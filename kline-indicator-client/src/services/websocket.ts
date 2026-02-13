type MessageHandler = (data: unknown) => void;

interface WSMessage {
  type: string;
  data?: unknown;
  request_id?: string;
  timestamp: string;
}

interface SubscribeStock {
  market: number;
  code: string;
  klinetype: number;
}

class WebSocketService {
  private ws: WebSocket | null = null;
  private url: string;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 3000;
  private handlers: Map<string, Set<MessageHandler>> = new Map();
  private isConnecting = false;
  private pingInterval: ReturnType<typeof setInterval> | null = null;
  private pingIntervalMs = 25000; // 25秒发送一次ping，小于服务端30秒心跳周期

  constructor(url: string) {
    this.url = url;
  }

  // 连接WebSocket
  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        resolve();
        return;
      }

      if (this.isConnecting) {
        reject(new Error('正在连接中'));
        return;
      }

      this.isConnecting = true;

      try {
        this.ws = new WebSocket(this.url);

        this.ws.onopen = () => {
          console.log('WebSocket连接成功');
          this.isConnecting = false;
          this.reconnectAttempts = 0;
          this.startPing();
          resolve();
        };

        this.ws.onmessage = (event) => {
          this.handleMessage(event.data);
        };

        this.ws.onerror = (error) => {
          console.error('WebSocket错误:', error);
          this.isConnecting = false;
          reject(error);
        };

        this.ws.onclose = () => {
          console.log('WebSocket连接关闭');
          this.isConnecting = false;
          this.stopPing();
          this.attemptReconnect();
        };
      } catch (error) {
        this.isConnecting = false;
        reject(error);
      }
    });
  }

  // 断开连接
  disconnect(): void {
    this.stopPing();
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  // 发送消息
  send(message: WSMessage): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    } else {
      console.warn('WebSocket未连接，无法发送消息');
    }
  }

  // 订阅股票
  subscribe(stocks: SubscribeStock[]): void {
    this.send({
      type: 'subscribe',
      data: { stocks },
      timestamp: new Date().toISOString(),
    });
  }

  // 取消订阅
  unsubscribe(stocks: SubscribeStock[]): void {
    this.send({
      type: 'unsubscribe',
      data: { stocks },
      timestamp: new Date().toISOString(),
    });
  }

  // 注册消息处理器
  on(type: string, handler: MessageHandler): () => void {
    if (!this.handlers.has(type)) {
      this.handlers.set(type, new Set());
    }
    this.handlers.get(type)!.add(handler);

    // 返回取消注册函数
    return () => {
      this.handlers.get(type)?.delete(handler);
    };
  }

  // 处理接收到的消息
  private handleMessage(data: string): void {
    try {
      const message: WSMessage = JSON.parse(data);
      const handlers = this.handlers.get(message.type);
      
      if (handlers) {
        handlers.forEach((handler) => handler(message.data));
      }

      // 触发通用消息处理器
      const allHandlers = this.handlers.get('*');
      if (allHandlers) {
        allHandlers.forEach((handler) => handler(message));
      }
    } catch (error) {
      console.error('解析WebSocket消息失败:', error);
    }
  }

  // 启动客户端心跳
  private startPing(): void {
    this.stopPing();
    this.pingInterval = setInterval(() => {
      this.send({
        type: 'ping',
        timestamp: new Date().toISOString(),
      });
    }, this.pingIntervalMs);
  }

  // 停止客户端心跳
  private stopPing(): void {
    if (this.pingInterval) {
      clearInterval(this.pingInterval);
      this.pingInterval = null;
    }
  }

  // 尝试重连
  private attemptReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('WebSocket重连次数已达上限');
      return;
    }

    this.reconnectAttempts++;
    console.log(`WebSocket尝试重连 (${this.reconnectAttempts}/${this.maxReconnectAttempts})`);

    setTimeout(() => {
      // 先清理旧连接，防止多实例并存
      if (this.ws) {
        this.ws.onclose = null; // 防止触发新的重连
        this.ws.onerror = null;
        this.ws.onmessage = null;
        this.ws.onopen = null;
        if (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING) {
          this.ws.close();
        }
        this.ws = null;
      }
      this.connect().catch(() => {
        // 重连失败，继续尝试
      });
    }, this.reconnectDelay * this.reconnectAttempts);
  }

  // 获取连接状态
  get isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }

  // 完全销毁服务，清理所有资源
  destroy(): void {
    this.stopPing();
    this.handlers.clear();
    this.reconnectAttempts = this.maxReconnectAttempts; // 防止重连
    if (this.ws) {
      this.ws.onclose = null;
      this.ws.onerror = null;
      this.ws.close();
      this.ws = null;
    }
  }
}

// 创建单例
const wsService = new WebSocketService(
  `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws/kline`
);

export default wsService;
