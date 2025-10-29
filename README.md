# MyGoChat

这是一个简单的即时通讯应用后端，旨在帮助我熟悉Golang的后端开发。

## 功能特性
- 用户注册与登录
- 实时消息传递
- 群组聊天
- 消息存储与检索

## 技术栈
- 后端框架：Gin
  - 中间件：JWT认证、日志记录
  - 路由管理
- ORM：Gorm
  - 数据库迁移与模型定义
  - CRUD操作
- 数据库：PostgreSQL
- 实时通信：WebSocket
    - 消息推送与接收
    - 连接管理
    - 心跳检测(TODO)
- 消息队列：Kafka
- 缓存：Redis
- 容器化：Docker

## 项目结构
```
MyGoChat/
├── api/                             # Protobuf API 定义
│   └── v1/
│       ├── message.pb.go
│       └── message.proto
├── cmd/                             # 项目主入口
│   └── main.go
├── configs/                         # 配置文件
│   └── config.prod.yaml
├── deployments/                     # 部署文件
│   └── docker-compose.yaml
├── internal/                        # 内部业务逻辑
│   ├── gateway/                     # 网关层
│   │   └── socket/                  # WebSocket 连接处理
│   │       ├── client.go            # 客户端连接
│   │       ├── hub.go               # 连接管理器
│   │       └── socket.go            # WebSocket 入口
│   ├── logic/                       # 核心业务逻辑
│   │   ├── data/                    # 数据访问层
│   │   │   └── data.go
│   │   ├── handler/                 # HTTP 请求处理器 (Controller)
│   │   │   ├── group_controller.go
│   │   │   └── user_controller.go
│   │   ├── middleware/              # 中间件
│   │   │   ├── cors.go
│   │   │   └── jwt.go
│   │   ├── server/                  # HTTP 服务
│   │   │   └── http.go
│   │   └── service/                 # 服务层
│   │       ├── group_service.go
│   │       ├── message_service.go
│   │       └── user_service.go
│   ├── model/                       # 数据模型
│   │   ├── group_member.go
│   │   ├── group.go
│   │   ├── message.go
│   │   ├── user_friend.go
│   │   └── user.go
│   └── util/                        # 工具类
│       └── password.go
└── pkg/                             # 公共包/SDK
    ├── common/                      # 通用请求/响应结构
    │   ├── request/
    │   │   ├── group_request.go
    │   │   └── user_request.go
    │   └── response/
    │       ├── group_response.go
    │       ├── response_msg.go
    │       └── user_response.go
    ├── config/                      # 配置加载
    │   └── config.go
    ├── db/                          # 数据库连接
    │   └── postgresql.go
    ├── HTML/                        # 前端测试页面
    │   ├── message_pb.js
    │   └── socket_test.html
    ├── kafka/                       # Kafka 生产者/消费者
    │   ├── consumer.go
    │   └── producer.go
    ├── log/                         # 日志
    │   └── logger.go
    ├── redis/                       # Redis 客户端
    │   └── redis.go
    ├── test/                        # 各类测试
    │   ├── config_test.go
    │   ├── db_connection_test.go
    │   ├── kafka_test.go
    │   └── logger_test.go
    └── token/                       # JWT Token
        └── token.go
```

## 快速开始

### 1. 环境准备

确保你的开发环境中安装了以下软件：

- Go (1.18+)
- PostgreSQL (12+)
- Kafka
- Redis
- Docker (可选, 用于快速启动Kafka)

### 2. 克隆与安装依赖

```bash
# 克隆仓库
git clone https://github.com/RichardoKang/MyGoChat.git

# 进入项目目录
cd MyGoChat

# 安装 Go 依赖
go mod tidy
```

### 3. 配置

项目通过 `configs/` 目录下的 `yaml` 文件进行配置。默认情况下，程序会加载 `config.dev.yaml`。 

你需要从生产配置模板 `config.prod.yaml` 复制一份来创建你的本地开发配置：

```bash
# 在项目根目录下执行
cp configs/config.prod.yaml configs/config.dev.yaml
```

然后，**修改 `configs/config.dev.yaml`** 文件，填入你本地数据库、Redis 和 Kafka 的连接信息。

### 4. 运行

你可以选择下面任意一种方式来启动项目及其依赖。

#### 方式一：本地手动启动 (完全控制)

1.  **启动依赖服务**：请确保你的本地 PostgreSQL, Redis, 和 Kafka 服务已经启动。

2.  **运行 Go 应用**：在项目根目录运行以下命令：
    ```bash
    go run ./cmd/main.go
    ```

#### 方式二：使用 Docker Compose 启动 Kafka

`deployments/docker-compose.yaml` 文件可以帮助你快速启动一个 Kafka 实例。

1.  **启动依赖服务**：请确保你的本地 PostgreSQL 和 Redis 服务已经启动。

2.  **启动 Kafka**：在项目根目录运行以下命令：
    ```bash
    docker-compose -f deployments/docker-compose.yaml up -d
    ```

3.  **运行 Go 应用**：
    ```bash
    go run ./cmd/main.go
    ```

项目启动后，默认会在 `8080` 端口监听 HTTP 请求。
