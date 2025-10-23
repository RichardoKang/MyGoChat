package request

import "github.com/golang-jwt/jwt/v5"

type CreateGroupRequest struct {
	GroupName string   `json:"groupName"`
	Members   []string `json:"members"`
	jwt.Token
}
