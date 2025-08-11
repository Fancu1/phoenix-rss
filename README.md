# Phoenix RSS - 一个 Go 实现的 RSS 聚合器

Phoenix RSS 是一个用 Go 语言编写的开源 RSS 聚合器。它提供了一个简单的 API 来添加和管理 RSS feed，并通过后台任务异步获取和存储文章。

## 架构设计

Phoenix RSS 采用了前后端分离、服务与工作进程分离的架构。核心由一个 API 服务器和一个后台工作进程组成，通过 Redis 任务队列进行解耦。

### 核心组件

-   **API 服务器**: 使用 [Gin](https://github.com/gin-gonic/gin) 框架构建，负责处理所有面向用户的 HTTP 请求。它提供了管理 RSS 源和查看文章的 RESTful API。
-   **后台工作进程**: 使用 [Asynq](https://github.com/hibiken/asynq) 框架实现，负责异步处理耗时任务，例如从 RSS 源抓取文章。这确保了 API 服务器可以快速响应用户请求。
-   **PostgreSQL 数据库**: 作为主数据存储，使用 [Gorm](https://gorm.io/) 作为 ORM，持久化存储 Feed 和文章信息。
-   **Redis**: 作为消息代理，支持 Asynq 的任务队列。所有待处理的抓取任务都在 Redis 中排队。

## 技术栈

-   **语言**: Go
-   **Web 框架**: Gin
-   **数据库**: PostgreSQL
-   **ORM**: Gorm
-   **任务队列**: Asynq
-   **消息代理**: Redis
-   **容器化**: Docker

## 主要功能

-   **用户注册与登录**：使用 JWT 进行无状态认证。
-   **订阅 RSS Feed**：用户通过 URL 订阅，系统自动去重并复用已存在的 Feed 记录。
-   **查看已订阅的 Feed 列表**（仅限当前登录用户）。
-   **取消订阅 Feed**。
-   **异步抓取文章**：基于 Asynq 的后台任务，可手动触发。
-   **阅读 Feed 文章**：仅能查看自己订阅的 Feed 下的文章。

## 目录结构

```
.
├── api/                  # OpenAPI/Swagger 规范 (当前为空)
├── cmd/                  # 应用入口
│   ├── server/           # API 服务器主程序
│   └── worker/           # 后台工作进程主程序
├── configs/              # 配置文件
├── internal/             # 私有应用和库代码
│   ├── config/           # 配置加载
│   ├── core/             # 核心业务逻辑 (Services)
│   ├── handler/          # HTTP 处理器
│   ├── models/           # GORM 数据模型
│   ├── repository/       # 数据仓库层
│   ├── server/           # Gin 服务器设置和路由
│   ├── tasks/            # Asynq 任务定义
│   └── worker/           # Asynq 工作进程实现
├── go.mod                # Go 模块文件
├── db-setup.sh           # 数据库设置脚本
├── redis-setup.sh        # Redis 设置脚本
└── docker-compose.yml    # Docker Compose 配置
```

## 运行应用

应用包含两个独立的进程，你需要分别启动它们。

### 启动 API 服务器

```bash
# 先执行数据库迁移
go run ./cmd/migrator up
# 启动 API 服务器
go run ./cmd/server/main.go

### 启动后台工作进程

```bash
# 启动 worker
go run ./cmd/worker/main.go
```

### 迁移（Migration）

```bash
go run ./cmd/migrator up
```

## 运行测试

```bash
go test -v ./...
```
