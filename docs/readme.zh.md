# GoLiteCron

[English](../readme.md) | [使用指南](usage.zh.md) | [链式API](chain_usage_zh.md)

## 概述
GoLiteCron 是一个轻量级、高性能的 cron 任务调度框架，适用于 Go 应用程序。它提供了简单而强大的接口来管理定时任务，支持各种 cron 表达式、时区、任务超时和重试功能。该框架提供了灵活的存储选项（TimeWheel 和 Heap），以适应不同的应用场景。

## 特性
- **Cron 表达式支持**
  - 标准 cron 语法（分钟、小时、日、月、星期）
  - 扩展语法支持秒级调度（使用 WithSeconds() 选项）
  - 支持年份指定（使用 WithYears() 选项）
  - 预定义宏（@yearly, @monthly, @weekly, @daily, @hourly, @minutely）
- **灵活的存储选项**
  - TimeWheel：高效处理高频任务和大量定时作业
  - Heap：简单实现，适用于通用调度场景
- **任务管理功能**
  - 自定义时区支持任务执行
  - 可配置的任务执行超时时间
  - 失败任务的自动重试机制
  - 通过 ID 注册任务，便于管理
  - 支持从配置文件（YAML/JSON）加载任务
- **可靠性**
  - 单个任务的 panic 恢复
  - 任务状态的原子操作管理
  - 适当的资源清理和优雅关闭

## 安装
```bash
go get -u github.com/hansir-hsj/GoLiteCron
```

## 文档
- **[使用指南](usage.zh.md)** - 完整的使用示例和API文档
- **[链式API](chain_usage_zh.md)** - 自然语言风格的任务调度API
- **[English Documentation](../readme.md)** - 英文文档

## 架构
GoLiteCron 由几个关键组件组成：
- **调度器（Scheduler）**：核心组件，管理任务执行并与存储后端协调
- **任务存储（Task Storage）**：实现任务的存储和检索。提供两种实现：
  - TimeWheel: 多级时间轮实现，高效处理大量不同间隔的任务
  - Heap: 优先级队列实现，按下次执行时间排序任务
- **Cron 解析器（Cron Parser）**：解析 cron 表达式并计算任务的下次执行时间
- **作业注册表（Job Registry）**：管理可在配置文件中通过名称引用的作业函数
- **配置加载器（Config Loader）**：从 YAML 或 JSON 文件加载任务配置

## 许可证
GoLiteCron 以 MIT 许可证发布。详情请参见 [LICENSE](LICENSE) 文件。
