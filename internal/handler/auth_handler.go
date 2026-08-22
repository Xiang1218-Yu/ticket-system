package handler

import (
	"github.com/gin-gonic/gin"

	"ticket-system/internal/model"
	"ticket-system/internal/service"
)

// AuthHandler 处理认证相关 HTTP 接口。
// 单一职责：仅做 HTTP I/O，业务在 AuthService。
type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type registerReq struct {
	Username string `json:"username" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role"`
	Group    string `json:"group"`
	IsLeader bool   `json:"is_leader"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerReq
	if !bindJSON(c, &req) {
		return
	}
	u, err := h.auth.Register(service.RegisterInput{
		Username: req.Username, Name: req.Name, Password: req.Password,
		Role: req.Role, Group: req.Group, IsLeader: req.IsLeader,
	})
	if err != nil {
		fail(c, err)
		return
	}
	created(c, userDTO(u))
}

type loginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginReq
	if !bindJSON(c, &req) {
		return
	}
	token, u, err := h.auth.Login(req.Username, req.Password)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{"token": token, "user": userDTO(u)})
}

// Me 返回当前登录用户。
func (h *AuthHandler) Me(c *gin.Context) {
	u := currentUser(c)
	ok(c, userDTO(u))
}

// userDTO 把 user 转为对外 DTO（不含密码，模型已用 json:"-" 兜底，这里裁剪冗余字段）。
func userDTO(u *model.User) gin.H {
	return gin.H{
		"id":        u.ID,
		"username":  u.Username,
		"name":      u.Name,
		"role":      u.Role,
		"group":     u.Group,
		"is_leader": u.IsLeader,
	}
}

// currentUser 从上下文取出已认证用户。
func currentUser(c *gin.Context) *model.User {
	v, _ := c.Get("user")
	u, _ := v.(*model.User)
	return u
}

// actorFromUser 把 model.User 转为 service.Actor，供工单业务调用。
func actorFromUser(u *model.User) service.Actor {
	return service.Actor{
		ID: u.ID, Role: u.Role, Name: u.Name, Group: u.Group, IsLeader: u.IsLeader,
	}
}
