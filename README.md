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
- 消息持久化：MongoDB(TODO)
- 在线状态：Redis(TODO)
- 容器化：Docker

## 项目结构
````
MyGoChat/
├── api/
│   └── v1/
│       ├── message.pb.go
│       └── message.proto
├── cmd/
│   └── main.go              # 中央依赖组装（DI wiring）
├── configs/
│   └── config.prod.yaml
├── deployments/
│   └── docker-compose.yaml
├── internal/
│   ├── gateway/
│   │   └── socket/
│   │       ├── client.go
│   │       ├── hub.go
│   │       └── socket.go
│   ├── logic/
│   │   ├── data/            # 数据访问层（替代旧的全局 db 包）
│   │   │   └── data.go
│   │   ├── handler/         # 中央 Handler 持有服务实例
│   │   │   ├── handler.go
│   │   │   ├── group_controller.go
│   │   │   └── user_controller.go
│   │   ├── middleware/
│   │   │   ├── cors.go
│   │   │   └── jwt.go
│   │   ├── server/
│   │   │   └── http.go
│   │   └── service/         # 服务层，构造函数接收依赖
│   │       ├── group_service.go
│   │       ├── message_service.go
│   │       └── user_service.go
│   ├── model/
│   │   ├── group_member.go
│   │   ├── group.go
│   │   ├── message.go
│   │   ├── user_friend.go
│   │   └── user.go
│   └── util/
│       └── password.go
└── pkg/
├── common/
│   ├── request/
│   │   ├── group_request.go
│   │   └── user_request.go
│   └── response/
│       ├── group_response.go
│       ├── response_msg.go
│       └── user_response.go
├── config/
│   └── config.go
├── kafka/
│   ├── consumer.go
│   └── producer.go
├── log/
│   └── logger.go
├── redis/
│   └── redis.go
├── test/
│   ├── config_test.go
│   ├── db_connection_test.go
│   ├── kafka_test.go
│   └── logger_test.go
└── token/
└── token.go
```
## 快速开始

### 1. 环境准备

确保你的开发环境中安装了以下软件：

- Go (1.18+)
- PostgreSQL (12+)
- Kafka
- Redis
- MongoDB
- Docker (可选, 用于快速启动Kafka)

### 2. 克隆与安装依赖

```bash
# 克隆仓库
git clone https://github.com/RichardoKang/MyGoChat.git

# 进入项目目录
cd MyGoChat

# 安装 Go 依赖
go mod tidy
