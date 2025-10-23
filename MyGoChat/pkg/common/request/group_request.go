package request

// CreateGroupRequest 创建群组请求
type CreateGroupRequest struct {
	GroupName string `json:"groupname" binding:"required,min=3,max=20"`
}
