# MyGoChat

一个基于 Go 语言的分布式即时通讯系统，采用 Gateway + Logic 分离架构。

## 功能特性

- 用户注册与登录（JWT 认证）
- 好友关系管理
- 私聊消息（实时 + 离线同步）
- 群聊消息
- WebSocket 实时通信
- 消息持久化存储

## 技术栈

| 组件 | 技术 | 说明 |
|------|------|------|
| Web 框架 | Gin | HTTP 路由、中间件 |
| ORM | GORM | PostgreSQL 操作 |
| 关系型数据库 | PostgreSQL | 用户、群组、关系数据 |
| 文档数据库 | MongoDB | 消息、会话存储 |
| 缓存/路由 | Redis | 用户在线状态、离线消息队列 |
| 消息队列 | Kafka | 消息投递、异步处理 |
| 序列化 | Protobuf | 消息格式定义 |
| 实时通信 | WebSocket | 消息推送 |
| 容器化 | Docker | 服务部署 |

## 系统架构

**会话流程：**
1. Client 通过 WebSocket 连接 Gateway
2. Gateway 将消息发送到 Kafka Ingest Topic
3. Logic 消费消息，处理业务逻辑（存储、路由计算）
4. Logic 查询 Redis 确定用户在线状态
5. 在线用户：投递到 Kafka Delivery Topic → Gateway 推送
6. 离线用户：存入 Redis 离线队列，上线时同步


![会话流程图](Markdown/会话流程图.png)
## 项目结构

```text
MyGoChat/
├── chat/                    # Logic 服务
│   ├── cmd/                 # 服务入口
│   ├── configs/             # 配置文件
│   └── internal/            # 聊天/关系/用户等核心模块
├── gateway/                 # Gateway 服务
│   ├── cmd/                 # 服务入口
│   └── internal/            # WebSocket 与消息分发
├── frontend/                # 前端静态资源与 Nginx 配置
├── pkg/                     # 公共组件（config/log/kafka/redis/token 等）
├── redis/                   # Redis 配置
├── docker-compose.yaml      # 容器编排（含 PostgreSQL/Mongo/Redis/Kafka）
└── Markdown/                # 项目文档
```

## 快速开始

### 环境要求

- Go 1.18+
- PostgreSQL 12+
- MongoDB 4.4+
- Redis 6+
- Kafka 2.8+
- Docker (可选)

### 安装与启动

```bash
# 克隆仓库
git clone https://github.com/RichardoKang/MyGoChat.git
cd MyGoChat

# 使用 Docker Compose 启动全部服务（推荐）
docker compose up -d --build

# 查看服务状态
docker compose ps
```

服务端口：
- frontend: `80`
- chat (logic): `8080`
- gateway: `8081`
- postgresql: `5432`
- mongo: `27017`
- redis: `6379`
- kafka: `9092`

## API 接口

### 用户模块

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/user/register | 用户注册 |
| POST | /api/user/login | 用户登录 |
| PUT | /api/user/info/update | 更新用户信息 |

### 消息模块

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/message/send | 发送消息 |
| GET | /api/message/history/:conversationId | 获取历史消息 |
| GET | /api/message/conversations | 获取会话列表 |
| POST | /api/message/conversation/private | 创建私聊会话 |
| POST | /api/message/sync-offline | 同步离线消息 |

### 关系模块

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/relations/add-friend | 添加好友 |
| POST | /api/relations/add-group | 加入群组 |
| GET | /api/relations/list | 获取关系列表 |

### 群组模块

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/group/create | 创建群组 |

### WebSocket

| 路径 | 说明 |
|------|------|
| ws://localhost:8081/ws?token={jwt} | WebSocket 连接 |

## 前端测试页面

项目提供 HTML 测试页面用于开发调试：

**文件**: `pkg/HTML/index.html`

**使用方法**:
```bash
# 启动服务后，在浏览器中打开
open pkg/HTML/index.html
```

**功能**:
- 用户登录/注册
- WebSocket 连接管理
- 会话列表展示
- 实时消息收发
- 好友添加
- 调试日志输出

**测试流程**:
1. 注册/登录用户
2. 添加好友
3. 连接 WebSocket
4. 选择会话并发送消息

## 配置说明

主要配置文件位于 `chat/configs/config.dev.yaml`。

使用 Docker Compose 时，建议按容器服务名配置依赖地址：

```yaml
# config.dev.yaml
PostgresSQL:
  user: "postgres"
  password: "password"
  host: "postgresql"
  port: 5432
  dbname: "chatdb"
  sslmode: "disable"
  timezone: "Asia/Shanghai"

Kafka:
  brokers:
    - "kafka:9093"
  topics:
    ingest: "im_message_ingest"
    sync_request: "im_sync_request"
    delivery: "im_message_delivery_"

Redis:
  addr: "redis:6379"
  password: "mygochat"
  db: 0

Mongo:
  uri: "mongodb://admin:admin@mongo:27017"
  database: "mygochat"
```

## 注意事项

1. **端口配置**: Logic 服务运行在 8080，Gateway 服务运行在 8081
2. **JWT Token**: WebSocket 连接需要携带有效的 JWT Token
3. **消息顺序**: 同一会话的消息通过 Kafka 分区键保证顺序
4. **离线消息**: 用户上线时自动同步离线消息

## License

MIT License
