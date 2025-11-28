# GoLiteCron

[中文文档](docs/readme.zh.md) | [使用指南](docs/usage.md) | [链式API](docs/chain_usage.md)

## Overview
GoLiteCron is a lightweight, high-performance cron job scheduling framework for Go applications. It provides a simple yet powerful interface for managing scheduled tasks with support for various cron expressions, time zones, task timeouts, and retries. The framework offers flexible storage options (TimeWheel and Heap) to suit different application scenarios.

## Features
- **Cron Expression Support**
  - Standard cron syntax (minutes, hours, day of month, month, day of week)
  - Extended syntax with seconds (using WithSeconds() option)
  - Year specification (using WithYears() option)
  - Predefined macros (@yearly, @monthly, @weekly, @daily, @hourly, @minutely)
- **Flexible Storage Options**
  - TimeWheel: Efficient for high-frequency tasks and large numbers of scheduled jobs
  - Heap: Simple implementation suitable for general-purpose scheduling
- **Task Management Features**
  - Custom time zones for task execution
  - Configurable timeout for task execution
  - Automatic retry mechanism for failed tasks
  - Task registration by ID for easy management
  - Support for loading tasks from configuration files (YAML/JSON)
- **Reliability**
  - Panic recovery for individual tasks
  - Atomic operations for task status management
  - Proper resource cleanup and graceful shutdown

## Installation
```bash
go get -u github.com/hansir-hsj/GoLiteCron
```

## Documentation
- **[Usage Guide](docs/usage.md)** - Complete usage examples and API documentation
- **[Chain API](docs/chain_usage.md)** - Natural language style task scheduling API
- **[中文文档](docs/readme.zh.md)** - Chinese documentation

## Architecture
GoLiteCron consists of several key components:
- **Scheduler**: The core component that manages task execution and coordinates with storage backend.
- **Task Storage**: Implements the storage and retrieval of tasks. Two implementations are provided:
  - TimeWheel: A multi-level time wheel implementation that efficiently handles large numbers of tasks with varying intervals.
  - Heap: A priority queue implementation that orders tasks by their next execution time.
- **Cron Parser**: Parses cron expressions and calculates the next execution time for tasks.
- **Job Registry**: Manages job functions that can be referenced by name in configuration files.
- **Config Loader**: Loads task configurations from YAML or JSON files.

## License
GoLiteCron is released under the MIT License. See the [LICENSE](LICENSE) file for details.
