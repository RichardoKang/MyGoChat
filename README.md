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
- 消息队列：Kafka(TODO)
- 缓存：Redis(TODO)
- 容器化：Docker(TODO)

## 项目结构
```
MyGoChat/
├── api/                             # API接口层
│   └── v1/                          # v1版本的API
│       ├── group_controller.go      # 群组相关的控制器
│       └── user_controller.go       # 用户相关的控制器
├── cmd/                             # 项目主入口
│   └── main.go                      # 主函数
├── configs/                         # 配置文件目录
│   └── config.prod.yaml             # 生产环境配置文件
├── deployments/                     # 部署相关文件
│   └── docker-compose.yaml          # Docker Compose部署文件
├── internal/                        # 内部业务逻辑
│   ├── gateway/                     # 网关，处理WebSocket等
│   │   └── socket.go                # WebSocket处理逻辑
│   ├── logic/                       # 业务逻辑层
│   │   ├── data/                    # 数据访问层
│   │   │   └── data.go              # 数据访问逻辑
│   │   ├── middleware/              # 中间件
│   │   │   ├── cors.go              # 跨域资源共享中间件
│   │   │   └── jwt.go               # JWT认证中间件
│   │   ├── mq/                      # 消息队列
│   │   │   ├── client.go            # 消息队列客户端
│   │   │   └── server.go            # 消息队列服务端
│   │   ├── server/                  # HTTP服务
│   │   │   └── http.go              # HTTP服务实现
│   │   └── service/                 # 服务层
│   │       ├── group_service.go     # 群组服务
│   │       ├── message_service.go   # 消息服务
│   │       └── user_service.go      # 用户服务
│   ├── model/                       # 数据模型
│   │   ├── group_member.go          # 群组成员模型
│   │   ├── group.go                 # 群组模型
│   │   ├── message.go               # 消息模型
│   │   ├── user_friend.go           # 用户好友关系模型
│   │   └── user.go                  # 用户模型
│   └── util/                        # 工具类
│       └── password.go              # 密码处理工具
└── pkg/                             # 公共包
    ├── common/                      # 通用工具
    │   ├── request/                 # 请求体封装
    │   │   ├── group_request.go     # 群组请求体
    │   │   └── user_request.go      # 用户请求体
    │   └── response/                # 响应体封装
    │       ├── group_response.go    # 群组响应体
    │       ├── response_msg.go      # 通用响应消息
    │       └── user_response.go     # 用户响应体
    ├── config/                      # 配置包
    │   ├── config_test.go           # 配置包测试
    │   └── config.go                # 配置包实现
    ├── db/                          # 数据库包
    │   ├── db_connection_test.go    # 数据库连接测试
    │   └── postgresql.go            # PostgreSQL数据库连接
    ├── HTML/                        # HTML文件
    │   └── socket_test.html         # WebSocket测试页面
    ├── kafka/                       # Kafka包
    │   ├── consumer.go              # Kafka消费者
    │   ├── producer.go              # Kafka生产者
    │   └── kafka_test.go            # Kafka测试
    ├── log/                         # 日志包
    │   ├── logger_test.go           # 日志包测试
    │   └── logger.go                # 日志包实现
    ├── redis/                       # Redis包
    └── token/                       # Token包
        └── token.go                 # Token生成和解析
```

## 快速开始（当前尚未完善）
### 环境要求
- Go 1.18+
- PostgreSQL 12+
- Kafka
- Docker (可选)

### 安装步骤
1. 克隆仓库
   ```bash
   git clone https://github.com/RichardoKang/MyGoChat.git
    ```
   
2. 进入项目目录
    ```bash
    cd MyGoChat
    ```

3. 安装依赖
   ```bash
   go mod tidy
   ```
   
4. 配置数据库连接
   在`configs/config.dev.yaml`中配置数据库连接信息。

5. 运行项目
   ```bash
   cd cmd/
   go run main.go
   ```
