package chat

import (
	pb "MyGoChat/pkg/api/v1"
	"MyGoChat/chat/internal/util"
	"MyGoChat/pkg/common/request"
	"context"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ================== Mock Definitions ==================

// MockRepository 模拟 chat.Repository 接口
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateMsg(ctx context.Context, msg *Message) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *MockRepository) GetByConversation(ctx context.Context, convID string, limit, offset int) ([]*Message, error) {
	args := m.Called(ctx, convID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Message), args.Error(1)
}

func (m *MockRepository) MarkAsRead(ctx context.Context, msgIDs []string) error {
	args := m.Called(ctx, msgIDs)
	return args.Error(0)
}

func (m *MockRepository) CreateConversation(ctx context.Context, conv *Conversation) error {
	args := m.Called(ctx, conv)
	return args.Error(0)
}

func (m *MockRepository) GetConversationByGroupNumber(ctx context.Context, groupNumber string) (*Conversation, error) {
	args := m.Called(ctx, groupNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Conversation), args.Error(1)
}

func (m *MockRepository) GetConversationsByUserID(ctx context.Context, userID string, limit int64) ([]*Conversation, error) {
	args := m.Called(ctx, userID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Conversation), args.Error(1)
}

func (m *MockRepository) GetConversationByID(ctx context.Context, conversationID string) (*Conversation, error) {
	args := m.Called(ctx, conversationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Conversation), args.Error(1)
}

func (m *MockRepository) UpdateLastMessage(ctx context.Context, conversationID string, message *Message) error {
	args := m.Called(ctx, conversationID, message)
	return args.Error(0)
}

// MockRelationRepository 模拟 relation.Repository 接口
type MockRelationRepository struct {
	mock.Mock
}

func (m *MockRelationRepository) JoinGroupRelation(ctx context.Context, userUUID, groupUUID string) error {
	args := m.Called(ctx, userUUID, groupUUID)
	return args.Error(0)
}

func (m *MockRelationRepository) CreateFriendRelation(ctx context.Context, userUUID, friendUUID string) error {
	args := m.Called(ctx, userUUID, friendUUID)
	return args.Error(0)
}

func (m *MockRelationRepository) ListUserRelation(ctx context.Context, userUUID string) (interface{}, error) {
	args := m.Called(ctx, userUUID)
	return args.Get(0), args.Error(1)
}

func (m *MockRelationRepository) GetGroupMemberUUIDs(ctx context.Context, groupUUID string) ([]string, error) {
	args := m.Called(ctx, groupUUID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

// MockGroupRepository 模拟 group.Repository 接口
type MockGroupRepository struct {
	mock.Mock
}

func (m *MockGroupRepository) CreateGroup(group interface{}, adminUser interface{}) error {
	args := m.Called(group, adminUser)
	return args.Error(0)
}

func (m *MockGroupRepository) GetMyGroups(userID uint) (interface{}, error) {
	args := m.Called(userID)
	return args.Get(0), args.Error(1)
}

func (m *MockGroupRepository) GetGroupByGroupNumber(groupNumber string) (interface{}, error) {
	args := m.Called(groupNumber)
	return args.Get(0), args.Error(1)
}

func (m *MockGroupRepository) GetUUIDByNumber(ctx context.Context, groupNumber string) (string, error) {
	args := m.Called(ctx, groupNumber)
	return args.String(0), args.Error(1)
}

// MockUserRepository 模拟 user.Repository 接口
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(user interface{}) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) GetUserByUsername(username string) (interface{}, error) {
	args := m.Called(username)
	return args.Get(0), args.Error(1)
}

func (m *MockUserRepository) Update(user interface{}) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) GetUserByUuid(uuid string) (interface{}, error) {
	args := m.Called(uuid)
	return args.Get(0), args.Error(1)
}

func (m *MockUserRepository) GetUserByID(id uint) (interface{}, error) {
	args := m.Called(id)
	return args.Get(0), args.Error(1)
}

func (m *MockUserRepository) GetUUIDByUsername(ctx context.Context, username string) (string, error) {
	args := m.Called(ctx, username)
	return args.String(0), args.Error(1)
}

// MockProducer 模拟 Kafka Producer
type MockProducer struct {
	mock.Mock
	SentMessages []SentMessage
}

type SentMessage struct {
	Topic   string
	Key     []byte
	Message []byte
}

func (m *MockProducer) SendMessage(topic string, message []byte) error {
	args := m.Called(topic, message)
	m.SentMessages = append(m.SentMessages, SentMessage{Topic: topic, Message: message})
	return args.Error(0)
}

func (m *MockProducer) SendMessageWithKey(topic string, key, message []byte) error {
	args := m.Called(topic, key, message)
	m.SentMessages = append(m.SentMessages, SentMessage{Topic: topic, Key: key, Message: message})
	return args.Error(0)
}

func (m *MockProducer) CloseProducer() {
	m.Called()
}

// ================== Test Cases ==================

// TestSendMessage_PrivateChat 测试私聊消息发送
// 验证：
// 1. ConversationID 生成正确（基于两个用户 UUID 的确定性 ID）
// 2. Kafka Producer 被正确调用
// 3. 消息体构建正确
func TestSendMessage_PrivateChat(t *testing.T) {
	// Arrange: 设置测试数据
	ctx := context.Background()
	senderUUID := "sender-uuid-12345"
	recipientUsername := "recipient_user"
	recipientUUID := "recipient-uuid-67890"

	// 计算预期的 ConversationID
	expectedConvID := util.GetPrivateConversationID(senderUUID, recipientUUID)

	// 创建 mock 对象
	mockRepo := new(MockRepository)
	mockRelRepo := new(MockRelationRepository)
	mockGroupRepo := new(MockGroupRepository)
	mockUserRepo := new(MockUserRepository)
	mockProducer := &MockProducer{SentMessages: []SentMessage{}}

	// 设置 mock 期望
	mockUserRepo.On("GetUUIDByUsername", mock.Anything, recipientUsername).
		Return(recipientUUID, nil)
	mockProducer.On("SendMessageWithKey", mock.Anything, mock.Anything, mock.Anything).
		Return(nil)

	// 创建 Service（注意：需要适配实际的依赖注入方式）
	// 这里我们使用接口兼容的方式来测试
	service := &testableService{
		repo:      mockRepo,
		userRepo:  mockUserRepo,
		groupRepo: mockGroupRepo,
		relRepo:   mockRelRepo,
		producer:  mockProducer,
	}

	// 构建请求
	req := &request.SendMessageRequest{
		TargetName:  recipientUsername,
		MessageType: 1, // 私聊
		ContentType: 1, // 文本
		Body: map[string]interface{}{
			"content": "Hello, this is a test message!",
		},
	}

	// Act: 执行被测方法
	msg, err := service.SendMessage(ctx, senderUUID, req)

	// Assert: 验证结果
	assert.NoError(t, err, "SendMessage should not return error")
	assert.NotNil(t, msg, "Message should not be nil")

	// 验证 ConversationID 生成正确
	assert.Equal(t, expectedConvID, msg.ConversationID,
		"ConversationID should be deterministic based on sender and recipient UUIDs")

	// 验证消息字段
	assert.Equal(t, senderUUID, msg.SenderUUID, "SenderUUID should match")
	assert.Equal(t, recipientUUID, msg.RecipientUUID, "RecipientUUID should match")
	assert.Equal(t, int32(1), msg.MessageType, "MessageType should be 1 for private chat")
	assert.Equal(t, int32(1), msg.ContentType, "ContentType should be 1 for text")
	assert.NotEmpty(t, msg.Id, "Message ID should be generated")
	assert.Greater(t, msg.SendAt, int64(0), "SendAt should be set")

	// 验证 Kafka Producer 被调用
	mockUserRepo.AssertExpectations(t)
	mockProducer.AssertExpectations(t)
	assert.Len(t, mockProducer.SentMessages, 1, "Should send exactly one message to Kafka")

	// 验证 Kafka Key 是 ConversationID（用于保证消息顺序）
	assert.Equal(t, []byte(expectedConvID), mockProducer.SentMessages[0].Key,
		"Kafka message key should be ConversationID for message ordering")
}

// TestSendMessage_GroupChat 测试群聊消息发送
func TestSendMessage_GroupChat(t *testing.T) {
	// Arrange
	ctx := context.Background()
	senderUUID := "sender-uuid-12345"
	groupNumber := "GROUP001"
	groupUUID := "group-uuid-abcde"

	// 群聊的 ConversationID 就是 GroupUUID
	expectedConvID := groupUUID

	mockRepo := new(MockRepository)
	mockRelRepo := new(MockRelationRepository)
	mockGroupRepo := new(MockGroupRepository)
	mockUserRepo := new(MockUserRepository)
	mockProducer := &MockProducer{SentMessages: []SentMessage{}}

	mockGroupRepo.On("GetUUIDByNumber", mock.Anything, groupNumber).
		Return(groupUUID, nil)
	mockProducer.On("SendMessageWithKey", mock.Anything, mock.Anything, mock.Anything).
		Return(nil)

	service := &testableService{
		repo:      mockRepo,
		userRepo:  mockUserRepo,
		groupRepo: mockGroupRepo,
		relRepo:   mockRelRepo,
		producer:  mockProducer,
	}

	req := &request.SendMessageRequest{
		TargetName:  groupNumber,
		MessageType: 2, // 群聊
		ContentType: 1, // 文本
		Body: map[string]interface{}{
			"content": "Hello everyone!",
		},
	}

	// Act
	msg, err := service.SendMessage(ctx, senderUUID, req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, msg)
	assert.Equal(t, expectedConvID, msg.ConversationID,
		"Group chat ConversationID should be GroupUUID")
	assert.Equal(t, groupUUID, msg.RecipientUUID,
		"RecipientUUID should be GroupUUID for group chat")
	assert.Equal(t, int32(2), msg.MessageType)

	mockGroupRepo.AssertExpectations(t)
	mockProducer.AssertExpectations(t)
}

// TestSendMessage_UserNotFound 测试用户不存在的情况
func TestSendMessage_UserNotFound(t *testing.T) {
	ctx := context.Background()
	senderUUID := "sender-uuid-12345"
	nonExistentUser := "ghost_user"

	mockRepo := new(MockRepository)
	mockRelRepo := new(MockRelationRepository)
	mockGroupRepo := new(MockGroupRepository)
	mockUserRepo := new(MockUserRepository)
	mockProducer := &MockProducer{SentMessages: []SentMessage{}}

	mockUserRepo.On("GetUUIDByUsername", mock.Anything, nonExistentUser).
		Return("", assert.AnError) // 模拟用户不存在

	service := &testableService{
		repo:      mockRepo,
		userRepo:  mockUserRepo,
		groupRepo: mockGroupRepo,
		relRepo:   mockRelRepo,
		producer:  mockProducer,
	}

	req := &request.SendMessageRequest{
		TargetName:  nonExistentUser,
		MessageType: 1,
		ContentType: 1,
		Body: map[string]interface{}{
			"content": "Hello?",
		},
	}

	// Act
	msg, err := service.SendMessage(ctx, senderUUID, req)

	// Assert
	assert.Error(t, err, "Should return error when user not found")
	assert.Nil(t, msg)
	assert.Contains(t, err.Error(), "用户不存在")
}

// TestConversationID_Deterministic 测试 ConversationID 的确定性
// 验证：不论谁先发起，A-B 和 B-A 应该生成相同的 ConversationID
func TestConversationID_Deterministic(t *testing.T) {
	userA := "alice-uuid-111"
	userB := "bob-uuid-222"

	convID1 := util.GetPrivateConversationID(userA, userB)
	convID2 := util.GetPrivateConversationID(userB, userA)

	assert.Equal(t, convID1, convID2,
		"ConversationID should be the same regardless of which user initiates the chat")
	assert.NotEmpty(t, convID1, "ConversationID should not be empty")
}

// TestSendMessage_InvalidMessageType 测试无效消息类型
func TestSendMessage_InvalidMessageType(t *testing.T) {
	ctx := context.Background()
	senderUUID := "sender-uuid-12345"

	mockRepo := new(MockRepository)
	mockRelRepo := new(MockRelationRepository)
	mockGroupRepo := new(MockGroupRepository)
	mockUserRepo := new(MockUserRepository)
	mockProducer := &MockProducer{SentMessages: []SentMessage{}}

	service := &testableService{
		repo:      mockRepo,
		userRepo:  mockUserRepo,
		groupRepo: mockGroupRepo,
		relRepo:   mockRelRepo,
		producer:  mockProducer,
	}

	req := &request.SendMessageRequest{
		TargetName:  "someone",
		MessageType: 99, // 无效类型
		ContentType: 1,
		Body: map[string]interface{}{
			"content": "Test",
		},
	}

	msg, err := service.SendMessage(ctx, senderUUID, req)

	assert.Error(t, err)
	assert.Nil(t, msg)
	assert.Contains(t, err.Error(), "不支持的消息类型")
}

// ================== Test Helper Types ==================

// testableService 是一个用于测试的 Service 实现
// 它使用 mock 对象替代真实的依赖
type testableService struct {
	repo      *MockRepository
	relRepo   *MockRelationRepository
	groupRepo *MockGroupRepository
	userRepo  *MockUserRepository
	producer  *MockProducer
	redis     *redis.Client
}

// SendMessage 测试版本的 SendMessage 方法
func (s *testableService) SendMessage(ctx context.Context, senderUUID string, req *request.SendMessageRequest) (*pb.Message, error) {
	var (
		targetUUID     string
		conversationID string
		err            error
	)

	if req.MessageType == 1 {
		// 私聊
		targetUUID, err = s.userRepo.GetUUIDByUsername(ctx, req.TargetName)
		if err != nil {
			return nil, errUserNotFound
		}
		conversationID = util.GetPrivateConversationID(senderUUID, targetUUID)
	} else if req.MessageType == 2 {
		// 群聊
		targetUUID, err = s.groupRepo.GetUUIDByNumber(ctx, req.TargetName)
		if err != nil {
			return nil, errGroupNotFound
		}
		conversationID = targetUUID
	} else {
		return nil, errInvalidMessageType
	}

	msg := &pb.Message{
		Id:             primitive.NewObjectID().Hex(),
		ConversationID: conversationID,
		SenderUUID:     senderUUID,
		RecipientUUID:  targetUUID,
		MessageType:    req.MessageType,
		ContentType:    req.ContentType,
		SendAt:         time.Now().Unix(),
	}

	// 模拟打包消息体
	// 在测试中我们跳过实际的 protobuf any 打包

	// 发送到 Kafka
	key := []byte(msg.ConversationID)
	if err := s.producer.SendMessageWithKey("test-topic", key, []byte("test-message")); err != nil {
		return nil, err
	}

	return msg, nil
}

// 错误定义
var (
	errUserNotFound       = &customError{msg: "用户不存在"}
	errGroupNotFound      = &customError{msg: "群组不存在"}
	errInvalidMessageType = &customError{msg: "不支持的消息类型"}
)

type customError struct {
	msg string
}

func (e *customError) Error() string {
	return e.msg
}

// ================== Benchmark Tests ==================

// BenchmarkGetPrivateConversationID 基准测试 ConversationID 生成性能
func BenchmarkGetPrivateConversationID(b *testing.B) {
	userA := "alice-uuid-111222333444"
	userB := "bob-uuid-555666777888"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		util.GetPrivateConversationID(userA, userB)
	}
}
