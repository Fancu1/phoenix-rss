# Phoenix RSS - 一个 Go 实现的 RSS 聚合器

Phoenix RSS 是一个用 Go 语言编写的开源 RSS 聚合器。它提供了一个简单的 API 来添加和管理 RSS feed，并通过后台任务异步获取和存储文章。

## 架构设计

Phoenix RSS 采用了前后端分离、服务与工作进程分离的架构。核心由一个 API 服务器和一个后台工作进程组成，通过 Redis 任务队列进行解耦。

```mermaid
graph TD
    subgraph 用户交互
        Client[客户端/用户]
    end

    subgraph API 服务器 (Go/Gin)
        Client -- "HTTP API 请求 (例如 POST /api/v1/feeds)" --> GinRouter(Gin 路由器)
        GinRouter -- " " --> FeedHandler(Feed 处理器)
        GinRouter -- " " --> ArticleHandler(文章处理器)
        FeedHandler -- "管理 Feed" --> FeedService(Feed 服务)
        ArticleHandler -- "触发抓取" --> AsynqClient(Asynq 客户端)
        FeedService -- "操作 Feed 数据" --> FeedRepo(Feed 仓库)
        ArticleHandler -- "列出文章" --> ArticleService(文章服务)
        ArticleService -- "操作文章数据" --> ArticleRepo(文章仓库)
    end

    subgraph 数据库
        PostgreSQL[(PostgreSQL)]
    end
    
    subgraph 任务队列
        Redis[(Redis)]
    end
    
    subgraph 后台工作进程 (Go/Asynq)
        Worker(Asynq Worker) -- "处理任务" --> TaskHandler(Feed 抓取处理器)
        TaskHandler -- "抓取并保存文章" --> ArticleService
    end

    AsynqClient -- "推送'Feed抓取'任务" --> Redis
    Worker -- "拉取任务" --> Redis
    FeedRepo -- "读/写" --> PostgreSQL
    ArticleRepo -- "读/写" --> PostgreSQL
    ArticleService -- "读取 Feed" --> FeedRepo
```


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

## 快速开始

### 环境要求

-   Go 1.18+
-   Docker

### 安装与运行

1.  **克隆仓库**

    ```bash
    git clone https://github.com/Fancu1/phoenix-rss.git
    cd phoenix-rss
    ```

2.  **启动依赖服务**

    项目提供了便捷的脚本来通过 Docker 启动 PostgreSQL 和 Redis。

    ```bash
    # 启动 PostgreSQL 容器
    ./db-setup.sh

    # 启动 Redis 容器
    ./redis-setup.sh
    ```
    
    你也可以使用 `docker-compose.yml` 来统一管理这些服务：
    ```bash
    docker-compose up -d
    ```

3.  **安装 Go 依赖**

    ```bash
    go mod tidy
    ```

4.  **运行应用**

    应用包含两个独立的进程，你需要分别启动它们。

    *   **启动 API 服务器:**
        ```bash
        go run ./cmd/server/main.go
        ```
        服务器默认在 `8080` 端口上运行。

    *   **启动后台工作进程:**
        ```bash
        go run ./cmd/worker/main.go
        ```

    应用启动时会自动执行数据库迁移，创建所需的表。

## 运行测试

在运行测试之前，请确保 PostgreSQL 和 Redis 正在通过 `db-setup.sh` 和 `redis-setup.sh` 或 `docker-compose` 运行。

执行以下命令来运行所有测试，包括集成测试：

```bash
go test -v ./...
```

## API 端点

> `🟢` 表示公共接口，`🔒` 表示需要在 `Authorization: Bearer <token>` 头中携带 JWT。

| 方法   | 路径                                           | 权限 | 描述                               |
| ------ | ---------------------------------------------- | ---- | ---------------------------------- |
| `GET`  | `/api/v1/health`                               | 🟢    | 健康检查                           |
| `POST` | `/api/v1/users/register`                       | 🟢    | 用户注册                           |
| `POST` | `/api/v1/users/login`                          | 🟢    | 用户登录，返回 JWT                 |
| `GET`  | `/api/v1/feeds`                                | 🔒    | 获取当前用户订阅的 Feed 列表       |
| `POST` | `/api/v1/feeds`                                | 🔒    | 订阅新的 RSS Feed                  |
| `DELETE`| `/api/v1/feeds/{feed_id}`                     | 🔒    | 取消订阅                           |
| `POST` | `/api/v1/feeds/{feed_id}/fetch`                | 🔒    | 触发异步抓取指定 Feed 的文章       |
| `GET`  | `/api/v1/feeds/{feed_id}/articles`             | 🔒    | 获取指定 Feed 的文章               | 
