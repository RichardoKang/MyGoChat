#!/bin/bash

# MyGoChat 项目启动脚本

set -e

echo "🚀 Starting MyGoChat Services..."

# 构建项目
echo "🔨 Building services..."
go build -o bin/logic ./cmd/logic
go build -o bin/gateway ./cmd/gateway

echo "✅ Build completed"

# 检查依赖服务
echo "📋 Checking dependencies..."

# 检查 Redis
if ! redis-cli ping > /dev/null 2>&1; then
    echo "⚠️  Redis is not running. Starting services anyway..."
else
    echo "✅ Redis is running"
fi
    echo "⚠️  PostgreSQL connection check failed (may be using remote server)"
fi

# 检查 MongoDB (可选，根据配置调整)
if ! mongosh --eval "db.adminCommand('ismaster')" > /dev/null 2>&1; then
    echo "⚠️  MongoDB connection check failed (may be using remote server)"
fi

# 检查 Kafka (可选)
if ! nc -z localhost 9092 > /dev/null 2>&1; then
    echo "⚠️  Kafka connection check failed (may need to start Kafka)"
    echo "   Use: docker-compose -f deployments/docker-compose.yaml up -d"
fi

echo ""
echo "🔧 Building services..."

# 构建服务
go build -o bin/logic cmd/logic/main.go
go build -o bin/gateway cmd/gateway/main.go

echo "✅ Build completed"
echo ""

# 设置环境变量
export GATEWAY_ID=${GATEWAY_ID:-"gateway-01"}

echo "🎯 Starting services..."
echo "   Gateway ID: $GATEWAY_ID"
echo "   Logic Service: http://localhost:8080"
echo "   Gateway Service: ws://localhost:8081"
echo ""

# 启动 Logic 服务 (后台)
echo "🔄 Starting Logic Service..."
./bin/logic > logs/logic.log 2>&1 &
LOGIC_PID=$!
echo "   Logic Service PID: $LOGIC_PID"

# 等待 Logic 服务启动
sleep 3

# 检查 Logic 服务是否正常启动
if ! curl -s http://localhost:8080/api/ > /dev/null; then
    echo "❌ Logic Service failed to start"
    kill $LOGIC_PID 2>/dev/null || true
    exit 1
fi
echo "✅ Logic Service is running"

# 启动 Gateway 服务 (后台)
echo "🔄 Starting Gateway Service..."
./bin/gateway > logs/gateway.log 2>&1 &
GATEWAY_PID=$!
echo "   Gateway Service PID: $GATEWAY_PID"

# 等待 Gateway 服务启动
sleep 2

echo ""
echo "🎉 All services started successfully!"
echo ""
echo "📚 Quick Test Commands:"
echo "   # Test Logic API"
echo "   curl http://localhost:8080/api/"
echo ""
echo "   # Register a user"
echo "   curl -X POST http://localhost:8080/api/user/register \\"
echo "     -H 'Content-Type: application/json' \\"
echo "     -d '{\"username\":\"testuser\",\"password\":\"123456\",\"nickname\":\"Test User\"}'"
echo ""
echo "🌐 Frontend Test Pages:"
echo "   # Complete Chat Client (Recommended for demo)"
echo "   open pkg/HTML/chat_client.html"
echo ""
echo "   # Function Test Tool (For debugging)"
echo "   open pkg/HTML/test_client.html"
echo ""
echo "   # WebSocket Test Tool (For connection testing)"
echo "   open pkg/HTML/websocket_test.html"
echo ""
echo "📊 Monitor Services:"
echo "   # View logs"
echo "   tail -f logs/logic.log"
echo "   tail -f logs/gateway.log"
echo ""
echo "   # Check Redis status"
echo "   redis-cli info"
echo ""
echo "🛑 Stop Services:"
echo "   kill $LOGIC_PID $GATEWAY_PID"
echo ""

# 创建停止脚本
cat > stop.sh << EOF
#!/bin/bash
echo "🛑 Stopping MyGoChat Services..."
kill $LOGIC_PID $GATEWAY_PID 2>/dev/null || true
echo "✅ Services stopped"
EOF
chmod +x stop.sh

echo "💡 Use './stop.sh' to stop all services"
echo ""
echo "🎯 Services are ready! Check the logs for any issues."

# 保持脚本运行，等待用户中断
trap 'kill $LOGIC_PID $GATEWAY_PID 2>/dev/null; exit' INT TERM
wait
