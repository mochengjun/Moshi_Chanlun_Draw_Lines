import { KLineType, KLineTypeNames } from '../../types';

interface TimeframeSwitcherProps {
  currentTimeframe: KLineType;
  onTimeframeChange: (timeframe: KLineType) => void;
}

// 外部API实际K线类型: 1:1分 4:3分 2:5分 5:15分 6:30分 3:60分 8:120分 10:日K 11:周K
const TIMEFRAMES: KLineType[] = [1, 4, 2, 5, 6, 3, 8, 10, 11];

export default function TimeframeSwitcher({ 
  currentTimeframe, 
  onTimeframeChange 
}: TimeframeSwitcherProps) {
  return (
    <div className="flex gap-1 p-2 bg-gray-800 rounded-lg">
      {TIMEFRAMES.map((tf) => (
        <button
          key={tf}
          onClick={() => onTimeframeChange(tf)}
          className={`px-3 py-1 text-sm rounded transition-colors ${
            currentTimeframe === tf
              ? 'bg-blue-600 text-white'
              : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
          }`}
        >
          {KLineTypeNames[tf]}
        </button>
      ))}
    </div>
  );
}
