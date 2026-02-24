import { useState } from 'react';
import type { IndicatorMetadata, IndicatorConfig } from '../../types';

interface IndicatorPanelProps {
  availableIndicators: IndicatorMetadata[];
  activeIndicators: IndicatorConfig[];
  onAddIndicator: (config: IndicatorConfig) => void;
  onRemoveIndicator: (type: string) => void;
}

export default function IndicatorPanel({
  availableIndicators,
  activeIndicators,
  onAddIndicator,
  onRemoveIndicator,
}: IndicatorPanelProps) {
  const [selectedType, setSelectedType] = useState<string>('');
  const [params, setParams] = useState<Record<string, number | string | boolean>>({});

  const selectedIndicator = availableIndicators.find((i) => i.type === selectedType);

  const handleAdd = () => {
    if (!selectedIndicator) return;

    const config: IndicatorConfig = {
      type: selectedIndicator.type,
      params: { ...params },
    };

    // 设置默认值
    selectedIndicator.params_def.forEach((p) => {
      if (config.params[p.name] === undefined) {
        config.params[p.name] = p.default_value;
      }
    });

    onAddIndicator(config);
    setSelectedType('');
    setParams({});
  };

  const handleParamChange = (name: string, value: string) => {
    const paramDef = selectedIndicator?.params_def.find((p) => p.name === name);
    if (!paramDef) return;

    let parsedValue: number | string | boolean = value;
    if (paramDef.type === 'int') {
      parsedValue = parseInt(value, 10) || 0;
    } else if (paramDef.type === 'float') {
      parsedValue = parseFloat(value) || 0;
    } else if (paramDef.type === 'bool') {
      parsedValue = value === 'true';
    }

    setParams((prev) => ({ ...prev, [name]: parsedValue }));
  };

  // 按分类分组
  const groupedIndicators = availableIndicators.reduce((acc, indicator) => {
    if (!acc[indicator.category]) {
      acc[indicator.category] = [];
    }
    acc[indicator.category].push(indicator);
    return acc;
  }, {} as Record<string, IndicatorMetadata[]>);

  const categoryNames: Record<string, string> = {
    basic: '基础指标',
    chanlun: '缠论指标',
    qiangshi: '强势调整',
    zhuli: '主力行为',
  };

  return (
    <div className="bg-gray-800 p-4 rounded-lg">
      <h3 className="text-white text-lg font-semibold mb-4">技术指标</h3>

      {/* 当前激活的指标 */}
      <div className="mb-4">
        <h4 className="text-gray-400 text-sm mb-2">已添加</h4>
        <div className="flex flex-wrap gap-2">
          {activeIndicators.map((ind) => (
            <span
              key={ind.type}
              className="inline-flex items-center px-2 py-1 bg-blue-600 text-white text-sm rounded"
            >
              {ind.type}
              <button
                onClick={() => onRemoveIndicator(ind.type)}
                className="ml-2 text-blue-200 hover:text-white"
              >
                ×
              </button>
            </span>
          ))}
          {activeIndicators.length === 0 && (
            <span className="text-gray-500 text-sm">暂无指标</span>
          )}
        </div>
      </div>

      {/* 指标选择 */}
      <div className="mb-4">
        <h4 className="text-gray-400 text-sm mb-2">添加指标</h4>
        <select
          value={selectedType}
          onChange={(e) => {
            setSelectedType(e.target.value);
            setParams({});
          }}
          className="w-full p-2 bg-gray-700 text-white rounded border border-gray-600"
        >
          <option value="">选择指标...</option>
          {Object.entries(groupedIndicators).map(([category, indicators]) => (
            <optgroup key={category} label={categoryNames[category] || category}>
              {indicators.map((ind) => (
                <option key={ind.type} value={ind.type}>
                  {ind.name}
                </option>
              ))}
            </optgroup>
          ))}
        </select>
      </div>

      {/* 参数配置 */}
      {selectedIndicator && (
        <div className="mb-4">
          <h4 className="text-gray-400 text-sm mb-2">参数设置</h4>
          <div className="space-y-2">
            {selectedIndicator.params_def.map((p) => (
              <div key={p.name} className="flex items-center gap-2">
                <label className="text-gray-300 text-sm w-24">{p.name}:</label>
                {p.type === 'bool' ? (
                  <select
                    value={String(params[p.name] ?? p.default_value)}
                    onChange={(e) => handleParamChange(p.name, e.target.value)}
                    className="flex-1 p-1 bg-gray-700 text-white rounded border border-gray-600 text-sm"
                  >
                    <option value="true">是</option>
                    <option value="false">否</option>
                  </select>
                ) : (
                  <input
                    type={p.type === 'int' || p.type === 'float' ? 'number' : 'text'}
                    value={String(params[p.name] ?? p.default_value)}
                    onChange={(e) => handleParamChange(p.name, e.target.value)}
                    min={p.min as number}
                    max={p.max as number}
                    className="flex-1 p-1 bg-gray-700 text-white rounded border border-gray-600 text-sm"
                  />
                )}
              </div>
            ))}
          </div>

          <button
            onClick={handleAdd}
            className="mt-3 w-full py-2 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors"
          >
            添加指标
          </button>
        </div>
      )}

      {/* 指标说明 */}
      {selectedIndicator && (
        <div className="text-gray-400 text-xs mt-2">
          {selectedIndicator.description}
        </div>
      )}
    </div>
  );
}
