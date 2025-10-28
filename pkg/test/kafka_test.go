package test

import (
	"MyGoChat/pkg/config"
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	myKafka "MyGoChat/pkg/kafka"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
)

// TestMain 在所有测试之前运行，用于设置测试环境
func TestMain(m *testing.M) {

	// 创建所需的Kafka topic
	topicsToCreate := []string{"test-topic", "test-round-trip-topic"}
	if err := createKafkaTopics(topicsToCreate...); err != nil {
		fmt.Printf("Failed to create kafka topics for testing: %v\n", err)
		os.Exit(1)
	}

	// Run 所有的测试
	code := m.Run()
	os.Exit(code)
}

// createKafkaTopics 创建指定的Kafka topics
func createKafkaTopics(topics ...string) error {
	cfg := config.GetConfig()
	if len(cfg.Kafka.Brokers) == 0 {
		return fmt.Errorf("no kafka brokers configured")
	}

	conn, err := kafka.Dial("tcp", cfg.Kafka.Brokers[0])
	if err != nil {
		return fmt.Errorf("failed to dial kafka broker: %w", err)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("failed to get controller: %w", err)
	}

	var controllerConn *kafka.Conn
	controllerConn, err = kafka.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		return fmt.Errorf("failed to dial controller: %w", err)
	}
	defer controllerConn.Close()

	var topicConfigs []kafka.TopicConfig
	for _, topic := range topics {
		topicConfigs = append(topicConfigs, kafka.TopicConfig{
			Topic:             topic,
			NumPartitions:     1,
			ReplicationFactor: 1,
		})
	}

	err = controllerConn.CreateTopics(topicConfigs...)
	if err != nil {
		// 如果topic已经存在，则忽略错误
		if ke, ok := err.(kafka.Error); ok && ke == kafka.TopicAlreadyExists {
			fmt.Printf("Topics %v already exist, continuing...\n", topics)
		} else {
			return fmt.Errorf("failed to create topics: %w", err)
		}
	} else {
		fmt.Printf("Successfully created topics: %v\n", topics)
	}

	return nil
}

// TestInitProducer 测试初始化Kafka生产者
func TestInitProducer(t *testing.T) {
	writer := myKafka.InitProducer()
	if writer == nil {
		t.Error("Expected non-nil Kafka writer after initialization")
	}

	myKafka.CloseProducer()
}

// TestSendMessage 测试发送消息功能
func TestSendMessage(t *testing.T) {
	myKafka.InitProducer()
	err := myKafka.SendMessage("test-topic", []byte("Hello, Kafka!"))
	if err != nil {
		t.Errorf("Expected no error sending message, got %v", err)
	}
	myKafka.CloseProducer()
}

func TestCloseProducer(t *testing.T) {
	myKafka.InitProducer()
	myKafka.CloseProducer()
	// 这项测试有点棘手，因为writer是一个包级变量。
	// 如果writer对象有一个公共状态方法，检查状态会是一个更好的方法。
	// 目前，我们假设如果没有panic，CloseProducer()就能工作。
}

// TestKafkaMessageRoundTrip 测试从发送到接收消息的完整流程
func TestKafkaMessageRoundTrip(t *testing.T) {
	// 1. 初始化
	myKafka.InitProducer()
	defer myKafka.CloseProducer()

	topic := "test-round-trip-topic"
	groupID := "test-round-trip-group"
	expectedMessage := []byte("hello world, this is a round trip test")
	messageReceived := make(chan []byte, 1)

	// 2. 启动消费者
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handler := func(m kafka.Message) {
		messageReceived <- m.Value
	}

	r := myKafka.InitConsumer(topic, groupID)
	myKafka.StartConsumer(ctx, r, handler)

	// 给消费者一些时间来启动
	time.Sleep(3 * time.Second)

	// 3. 发送消息
	err := myKafka.SendMessage(topic, expectedMessage)
	assert.NoError(t, err, "Sending message should not produce an error")

	// 4. Wait for and Verify Message
	select {
	case receivedMessage := <-messageReceived:
		assert.Equal(t, expectedMessage, receivedMessage, "Received message should match the sent message")
	case <-time.After(15 * time.Second):
		t.Fatal("Test timed out: did not receive message from Kafka")
	}
}
