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
```

项目启动后，默认会在 8080 端口监听 HTTP 请求。


## 前端测试页面

项目提供了多个 HTML 测试页面，可以直接在浏览器中测试各项功能：

### 📱 完整聊天客户端
**文件**: `pkg/HTML/chat_client.html`
- **功能**: 完整的聊天界面，包含登录、注册、聊天列表、实时消息等
- **特点**: 类似真实聊天应用的用户界面
- **适用**: 综合功能测试和演示

### 🔧 功能测试工具
**文件**: `pkg/HTML/test_client.html`  
- **功能**: 分模块测试各个 API 接口和功能
- **特点**: 详细的日志输出，便于调试
- **适用**: 开发调试和 API 测试

### 🌐 WebSocket 专用测试
**文件**: `pkg/HTML/websocket_test.html`
- **功能**: 专门测试 WebSocket 连接和实时消息
- **特点**: 实时连接状态显示，消息收发记录
- **适用**: WebSocket 功能调试

### 🚀 使用方法

1. **启动服务**
   ```bash
   # 启动后端服务
   ./start.sh
   ```

2. **打开测试页面**
   ```bash
   # 在浏览器中打开任意测试页面
   open pkg/HTML/chat_client.html
   open pkg/HTML/test_client.html  
   open pkg/HTML/websocket_test.html
   ```

3. **测试流程**
   ```
   注册用户 → 登录获取Token → 连接WebSocket → 创建群聊/私聊 → 发送消息
   ```

### 📋 测试场景示例

#### 基础功能测试
```javascript
// 1. 注册两个测试用户
用户A: username: "alice", password: "123456", nickname: "Alice"
用户B: username: "bob", password: "123456", nickname: "Bob"

// 2. 分别登录获取 JWT Token
// 3. 建立 WebSocket 连接
// 4. 创建私聊会话
// 5. 发送消息测试
```

#### 群聊功能测试  
```javascript
// 1. 用户A创建群聊 "测试群"
// 2. 获取群号（如：123456）
// 3. 用户B加入群聊
// 4. 群内发送消息测试
// 5. 验证群成员都能收到消息
```

#### 离线消息测试
```javascript
// 1. 用户A在线，用户B离线
// 2. 用户A发送消息给用户B
// 3. 用户B上线并连接WebSocket
// 4. 验证用户B自动收到离线消息
```

### ⚠️ 注意事项

1. **服务端口确认**: 确保服务运行在正确端口
   - Logic 服务: `http://localhost:8080`
   - Gateway 服务: `ws://localhost:8081`

2. **CORS 设置**: 如果遇到跨域问题，请检查 CORS 中间件配置

3. **WebSocket 连接**: 需要先登录获取 JWT Token 才能建立 WebSocket 连接

4. **浏览器兼容**: 建议使用现代浏览器（Chrome、Firefox、Safari、Edge）

### 🐛 常见问题

- **连接失败**: 检查服务是否启动，端口是否正确
- **WebSocket 错误**: 确认 JWT Token 有效性  
- **消息不显示**: 检查会话ID是否正确，用户是否在线
- **API 调用失败**: 查看浏览器开发者工具的 Network 选项卡
