package response

type UserResponse struct {
	ID       int32  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`
}
