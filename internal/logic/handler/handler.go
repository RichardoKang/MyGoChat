package handler

import "MyGoChat/internal/logic/service"

// Handler holds all the dependencies for the application handlers.
// It acts as the receiver for all handler functions.
type Handler struct {
	UserService    *service.UserService
	GroupService   *service.GroupService
	MessageService *service.MessageService
}
