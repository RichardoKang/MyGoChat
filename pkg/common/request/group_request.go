package request

// CreateGroupRequest 创建群组请求
type CreateGroupRequest struct {
	GroupName string `json:"groupname" binding:"required,min=3,max=20"`
}

// JoinGroupRequest 加入群组请求
type JoinGroupRequest struct {
	GroupNumber string `json:"groupnumber" binding:"required,len=6"`
	Nickname    string `json:"nickname" binding:"required,min=3,max=20"`
}

// GetGroupMembersRequest 获取群组成员请求
type GetGroupMembersRequest struct {
	GroupNumber string `uri:"groupnumber" binding:"required,len=6"`
}
