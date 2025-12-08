package request

type JoinGroupRelationRequest struct {
	Username    string `json:"username" binding:"required"`
	GroupNumber string `json:"group_number" binding:"required"`
}

type CreateFriendRelationRequest struct {
	Username       string `json:"username1" binding:"required"`
	FriendUsername string `json:"username2" binding:"required"`
}

type ListUserRelationRequest struct {
	Username string `json:"username" binding:"required"`
}
