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

## 安装与运行(目前)
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
   在`config/config.dev.yaml`中配置数据库连接信息。

5. 运行项目
   ```bash
   cd cmd/
   go run main.go
   ```
   