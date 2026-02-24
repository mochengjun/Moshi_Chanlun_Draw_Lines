package moshi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"kline-indicator-service/internal/calculator"
	"kline-indicator-service/internal/models"
)

// cppCLIPath C++ calculator_cli可执行文件路径
var cppCLIPath string

// init 初始化时注册指标并查找CLI路径
func init() {
	cppCLIPath = findCLIPath()
	calculator.Register(models.IndicatorTypeMoshiChanlun, NewMoshiChanlunIndicator)
}

// findCLIPath 查找C++ calculator_cli可执行文件路径
func findCLIPath() string {
	// 可能的相对路径
	possiblePaths := []string{
		"../cpp-trading-system/build/calculator_cli",
		"../../cpp-trading-system/build/calculator_cli",
		"../../../cpp-trading-system/build/calculator_cli",
		"cpp-trading-system/build/calculator_cli",
	}

	// Windows下添加.exe后缀
	if runtime.GOOS == "windows" {
		windowsPaths := make([]string, 0, len(possiblePaths)*2)
		for _, p := range possiblePaths {
			windowsPaths = append(windowsPaths, p+".exe")
			windowsPaths = append(windowsPaths, p)
		}
		possiblePaths = windowsPaths
	}

	// 从环境变量获取
	if envPath := os.Getenv("CPP_CLI_PATH"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath
		}
	}

	// 获取当前工作目录
	cwd, _ := os.Getwd()

	// 尝试相对路径
	for _, p := range possiblePaths {
		absPath := filepath.Join(cwd, p)
		if _, err := os.Stat(absPath); err == nil {
			return absPath
		}
	}

	// 默认路径
	defaultPath := "../cpp-trading-system/build/calculator_cli"
	if runtime.GOOS == "windows" {
		defaultPath += ".exe"
	}
	return defaultPath
}

// MoshiChanlunIndicator 莫氏缠论指标（通过C++ CLI桥接）
type MoshiChanlunIndicator struct{}

// NewMoshiChanlunIndicator 创建莫氏缠论指标实例
func NewMoshiChanlunIndicator() calculator.Indicator {
	return &MoshiChanlunIndicator{}
}

// Metadata 获取指标元信息
func (m *MoshiChanlunIndicator) Metadata() models.IndicatorMetadata {
	return models.IndicatorMetadata{
		Type:        models.IndicatorTypeMoshiChanlun,
		Name:        "莫氏缠论",
		Category:    models.CategoryChanlun,
		Description: "莫氏缠论画线指标 - 多级别(sub-x1/x1/x2/x4/x8)标注点、走势识别（C++计算引擎）",
		ParamsDef: []models.ParameterDef{
			{
				Name:         "kline_type",
				Type:         "int",
				Required:     false,
				DefaultValue: 10,
				Description:  "K线类型",
			},
			{
				Name:         "show_level_sub_x1",
				Type:         "bool",
				Required:     false,
				DefaultValue: true,
				Description:  "显示sub-x1级别",
			},
			{
				Name:         "show_level_1x",
				Type:         "bool",
				Required:     false,
				DefaultValue: true,
				Description:  "显示x1级别",
			},
			{
				Name:         "show_level_2x",
				Type:         "bool",
				Required:     false,
				DefaultValue: true,
				Description:  "显示x2级别",
			},
			{
				Name:         "show_level_4x",
				Type:         "bool",
				Required:     false,
				DefaultValue: true,
				Description:  "显示x4级别",
			},
			{
				Name:         "show_level_8x",
				Type:         "bool",
				Required:     false,
				DefaultValue: true,
				Description:  "显示x8级别",
			},
		},
	}
}

// Validate 验证参数
func (m *MoshiChanlunIndicator) Validate(params map[string]interface{}) error {
	// 参数验证由C++端处理，这里只做基本检查
	if params == nil {
		return nil
	}

	if klineType, ok := params["kline_type"]; ok {
		switch v := klineType.(type) {
		case int:
			if v < 0 {
				return fmt.Errorf("kline_type不能为负数")
			}
		case float64:
			if v < 0 {
				return fmt.Errorf("kline_type不能为负数")
			}
		}
	}

	return nil
}

// cliInput C++ CLI输入格式
type cliInput struct {
	KLines []cliKLine             `json:"klines"`
	Params map[string]interface{} `json:"params"`
}

// cliKLine C++ CLI K线格式
type cliKLine struct {
	Timestamp string  `json:"timestamp"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Close     float64 `json:"close"`
	Volume    float64 `json:"volume"`
}

// cliOutput C++ CLI输出格式
type cliOutput struct {
	Type              string                 `json:"type"`
	Name              string                 `json:"name"`
	BiMarkers         []models.BiMarker      `json:"bi_markers"`
	Extra             map[string]interface{} `json:"extra"`
	ComputationTimeMs int64                  `json:"computation_time_ms"`
	Error             string                 `json:"error,omitempty"`
}

// Calculate 计算指标（通过调用C++ CLI）
func (m *MoshiChanlunIndicator) Calculate(klines []models.KLine, params map[string]interface{}) (*models.IndicatorResult, error) {
	startTime := time.Now()

	// 检查CLI路径
	if _, err := os.Stat(cppCLIPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("C++ calculator_cli不存在: %s (请先编译C++项目)", cppCLIPath)
	}

	// 准备输入数据
	input := cliInput{
		KLines: make([]cliKLine, len(klines)),
		Params: params,
	}

	for i, k := range klines {
		input.KLines[i] = cliKLine{
			Timestamp: k.Timestamp,
			Open:      k.Open,
			High:      k.High,
			Low:       k.Low,
			Close:     k.Close,
			Volume:    k.Volume,
		}
	}

	// 序列化输入
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("序列化输入失败: %w", err)
	}

	// 调用C++ CLI
	cmd := exec.Command(cppCLIPath)
	cmd.Stdin = bytes.NewReader(inputJSON)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		return nil, fmt.Errorf("C++ CLI执行失败: %s", errMsg)
	}

	// 解析输出
	var output cliOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		return nil, fmt.Errorf("解析C++ CLI输出失败: %w (output: %s)", err, stdout.String())
	}

	// 检查错误
	if output.Error != "" {
		return nil, fmt.Errorf("C++ 计算错误: %s", output.Error)
	}

	// 构建结果
	result := &models.IndicatorResult{
		Type:            models.IndicatorTypeMoshiChanlun,
		Name:            output.Name,
		BiMarkers:       output.BiMarkers,
		Extra:           output.Extra,
		ComputationTime: time.Since(startTime).Milliseconds(),
	}

	// 如果C++返回了计算时间，使用它
	if output.ComputationTimeMs > 0 {
		result.ComputationTime = output.ComputationTimeMs
	}

	return result, nil
}

// SetCLIPath 设置CLI路径（用于测试或自定义配置）
func SetCLIPath(path string) {
	cppCLIPath = path
}

// GetCLIPath 获取当前CLI路径
func GetCLIPath() string {
	return cppCLIPath
}
