interface LevelConfig {
  show_level_sub_x1: boolean;
  show_level_1x: boolean;
  show_level_2x: boolean;
  show_level_4x: boolean;
  show_level_8x: boolean;
}

interface LevelToggleProps {
  levels: LevelConfig;
  showTrends: boolean;
  showFractal: boolean;
  onLevelsChange: (levels: LevelConfig) => void;
  onShowTrendsChange: (show: boolean) => void;
  onShowFractalChange: (show: boolean) => void;
}

const LEVEL_ITEMS: { key: keyof LevelConfig; label: string; color: string }[] = [
  { key: 'show_level_sub_x1', label: 'sub-x1', color: '#888888' },
  { key: 'show_level_1x', label: 'x1', color: '#FFD700' },
  { key: 'show_level_2x', label: 'x2', color: '#FF6B6B' },
  { key: 'show_level_4x', label: 'x4', color: '#4ECDC4' },
  { key: 'show_level_8x', label: 'x8', color: '#9B59B6' },
];

export default function LevelToggle({
  levels,
  showTrends,
  showFractal,
  onLevelsChange,
  onShowTrendsChange,
  onShowFractalChange,
}: LevelToggleProps) {
  const handleToggle = (key: keyof LevelConfig) => {
    onLevelsChange({ ...levels, [key]: !levels[key] });
  };

  return (
    <div className="flex items-center gap-1">
      <span className="text-gray-400 text-sm mr-1">级别:</span>
      {LEVEL_ITEMS.map(({ key, label, color }) => (
        <button
          key={key}
          onClick={() => handleToggle(key)}
          className={`px-2 py-0.5 text-xs rounded border transition-colors ${
            levels[key]
              ? 'border-opacity-80 text-white'
              : 'border-gray-600 text-gray-500 opacity-50'
          }`}
          style={{
            borderColor: levels[key] ? color : undefined,
            backgroundColor: levels[key] ? `${color}22` : undefined,
            color: levels[key] ? color : undefined,
          }}
          title={`${levels[key] ? '隐藏' : '显示'} ${label} 级别`}
        >
          {label}
        </button>
      ))}
      <span className="text-gray-600 mx-1">|</span>
      <button
        onClick={() => onShowTrendsChange(!showTrends)}
        className={`px-2 py-0.5 text-xs rounded border transition-colors ${
          showTrends
            ? 'border-orange-500 bg-orange-500/10 text-orange-400'
            : 'border-gray-600 text-gray-500 opacity-50'
        }`}
        title={`${showTrends ? '隐藏' : '显示'}走势区域`}
      >
        走势
      </button>
      <span className="text-gray-600 mx-1">|</span>
      <button
        onClick={() => onShowFractalChange(!showFractal)}
        className={`px-2 py-0.5 text-xs rounded border transition-colors ${
          showFractal
            ? 'border-green-500 bg-green-500/10 text-green-400'
            : 'border-gray-600 text-gray-500 opacity-50'
        }`}
        title={`${showFractal ? '隐藏' : '显示'}顶底分型`}
      >
        分型
      </button>
    </div>
  );
}

export type { LevelConfig };
