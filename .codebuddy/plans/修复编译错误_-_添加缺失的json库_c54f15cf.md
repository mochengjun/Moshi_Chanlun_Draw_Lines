---
name: 修复编译错误 - 添加缺失的json库
overview: 项目缺少 third_party/json.hpp 文件，导致编译失败。需要创建目录并添加 nlohmann/json 单头文件。
todos:
  - id: create-third-party-dir
    content: 创建 cpp-trading-system/third_party 目录
    status: completed
  - id: download-json-hpp
    content: 下载 nlohmann/json 单头文件到 third_party/json.hpp
    status: completed
    dependencies:
      - create-third-party-dir
  - id: update-cmake
    content: 修改 CMakeLists.txt 添加 third_party include 路径
    status: completed
    dependencies:
      - download-json-hpp
  - id: verify-build
    content: 验证编译通过
    status: completed
    dependencies:
      - update-cmake
---

## 用户需求

修复 C++ 项目的编译错误，使编译能够通过。

## 问题分析

- `apps/server/calculator_cli.cpp:13` 包含 `#include "third_party/json.hpp"`
- 项目中不存在 `third_party` 目录，也没有 `json.hpp` 文件
- `json.hpp` 是 nlohmann/json 库的单头文件，用于 JSON 解析

## 解决方案

创建 `third_party` 目录并添加 nlohmann/json 单头文件，同时更新 CMakeLists.txt 配置 include 路径。

## 技术方案

### 问题定位

- 缺失依赖：nlohmann/json 单头库（`json.hpp`）
- 编译错误：找不到 `third_party/json.hpp` 头文件

### 解决方案

1. **创建目录结构**：在 `cpp-trading-system/` 下创建 `third_party/` 目录
2. **添加 JSON 库**：下载 nlohmann/json 单头文件到 `third_party/json.hpp`
3. **更新 CMake 配置**：修改 CMakeLists.txt，添加 `third_party` 到 include 路径

### 目录结构变更

```
cpp-trading-system/
├── CMakeLists.txt          # [MODIFY] 添加 third_party include 路径
├── third_party/            # [NEW] 第三方库目录
│   └── json.hpp            # [NEW] nlohmann/json 单头文件 (v3.11.3)
```

### 实现细节

- 使用 nlohmann/json 单头文件版本（Header-only），无需编译链接
- json.hpp 文件约 23000+ 行，提供完整的 JSON 解析和序列化功能
- CMakeLists.txt 需添加：`include_directories(${CMAKE_SOURCE_DIR}/third_party)`