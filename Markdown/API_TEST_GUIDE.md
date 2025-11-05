# MyGoChat API 测试指南

## 系统架构

MyGoChat 采用微服务架构：
- **Gateway 服务** (端口 8081): 处理 WebSocket 连接和实时消息分发
- **Logic 服务** (端口 8080): 处理 HTTP API 请求和业务逻辑

## 启动服务

### 1. 启动依赖服务
```bash
# 启动 PostgreSQL, Redis, MongoDB, Kafka
docker-compose -f deployments/docker-compose.yaml up -d
```

### 2. 启动应用服务
```bash
# 启动 Logic 服务
go run cmd/logic/main.go

# 启动 Gateway 服务  
GATEWAY_ID=gateway-01 go run cmd/gateway/main.go
```

## API 端点

### 用户相关
```bash
# 注册用户
curl -X POST http://localhost:8080/api/user/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "alice",
    "password": "123456",
    "nickname": "Alice"
  }'

# 用户登录
curl -X POST http://localhost:8080/api/user/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "alice", 
    "password": "123456"
  }'
# 返回 JWT token，后续请求需要在 Header 中携带
```

### 会话相关 (需要 JWT Token)
```bash
# 获取会话列表
curl -X GET http://localhost:8080/api/conversation/list \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# 创建私聊会话
curl -X POST http://localhost:8080/api/conversation/private \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "target_user_uuid": "target_user_uuid_here"
  }'
```

### 消息相关 (需要 JWT Token)
```bash
# 获取消息历史
curl -X GET http://localhost:8080/api/message/history/CONVERSATION_ID?limit=20&offset=0 \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# 同步离线消息
curl -X POST http://localhost:8080/api/message/sync-offline \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# 标记消息已读
curl -X POST http://localhost:8080/api/message/mark-read \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "message_ids": ["msg_id_1", "msg_id_2"]
  }'
```

## WebSocket 连接

### 连接到网关
```javascript
// JavaScript 示例
const ws = new WebSocket("ws://localhost:8081/ws?token=YOUR_JWT_TOKEN");

ws.onopen = function() {
    console.log("Connected to WebSocket");
};

ws.onmessage = function(event) {
    const message = JSON.parse(event.data);
    console.log("Received message:", message);
};

// 发送消息
const message = {
    conversationID: "conversation_id_here",
    contentType: 1, // 1=文本，2=图片，3=文件，4=语音
    body: "Hello World!",
    messageType: 1, // 1=私聊，2=群聊
    recipientUUID: "recipient_uuid_here"
};
ws.send(JSON.stringify(message));
```

## 消息流程

1. **发送消息**: Client → WebSocket → Gateway → Kafka(ingest) → Logic → MongoDB
2. **消息推送**: Logic → Kafka(delivery_gateway-id) → Gateway → WebSocket → Client
3. **离线消息**: Logic → Redis → 用户上线时自动同步

## 数据模型

### 消息结构
```json
{
  "id": "message_id",
  "conversationID": "conversation_id", 
  "senderUUID": "sender_uuid",
  "sendAt": 1699000000,
  "contentType": 1,
  "body": "message_content",
  "messageType": 1,
  "recipientUUID": "recipient_uuid"
}
```

### 会话结构
```json
{
  "id": "conversation_id",
  "type": 1,
  "participants": ["user1_uuid", "user2_uuid"],
  "lastMessage": "last_message_content",
  "lastMessageTimestamp": 1699000000,
  "createdAt": "2025-11-03T00:00:00Z",
  "updatedAt": "2025-11-03T00:00:00Z"
}
```

## 测试场景

### 1. 基本消息收发
1. 注册两个用户 Alice 和 Bob
2. 获取 JWT tokens
3. Alice 和 Bob 分别连接 WebSocket
4. Alice 发送消息给 Bob
5. 验证 Bob 收到消息

### 2. 离线消息
1. Alice 在线，Bob 离线
2. Alice 发送消息给 Bob
3. Bob 上线连接 WebSocket
4. 验证 Bob 自动收到离线消息

### 3. 群聊功能
1. 创建群组
2. 多个用户加入群组
3. 发送群消息
4. 验证所有群成员收到消息

## 故障排除

### 常见问题
1. **WebSocket 连接失败**: 检查 JWT token 是否有效
2. **消息不推送**: 检查 Kafka 和 Redis 连接状态
3. **离线消息丢失**: 检查 Redis 配置和数据持久化

### 日志查看
```bash
# Logic 服务日志
tail -f ./pkg/log/log

# Kafka 消费情况
kafka-console-consumer --bootstrap-server localhost:9092 --topic im_message_ingest

# Redis 状态
redis-cli monitor
```
