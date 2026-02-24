import { useState, useEffect } from 'react';
import wsService from '../../services/websocket';

interface NetworkStatusProps {
  apiConnected: boolean;
}

export default function NetworkStatus({ apiConnected }: NetworkStatusProps) {
  const [wsConnected, setWsConnected] = useState(false);

  useEffect(() => {
    // 监听WebSocket连接状态
    const unsubscribeConnected = wsService.on('connected', () => {
      setWsConnected(true);
    });

    const unsubscribeHeartbeat = wsService.on('heartbeat', () => {
      setWsConnected(true);
    });

    // 定期检查连接状态
    const interval = setInterval(() => {
      setWsConnected(wsService.isConnected);
    }, 5000);

    return () => {
      unsubscribeConnected();
      unsubscribeHeartbeat();
      clearInterval(interval);
    };
  }, []);

  return (
    <div className="flex items-center gap-4 text-sm">
      <div className="flex items-center gap-2">
        <span
          className={`w-2 h-2 rounded-full ${
            apiConnected ? 'bg-green-500' : 'bg-red-500'
          }`}
        />
        <span className="text-gray-300">API</span>
      </div>
      <div className="flex items-center gap-2">
        <span
          className={`w-2 h-2 rounded-full ${
            wsConnected ? 'bg-green-500' : 'bg-yellow-500'
          }`}
        />
        <span className="text-gray-300">WebSocket</span>
      </div>
    </div>
  );
}
