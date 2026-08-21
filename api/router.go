package api

import (
	"github.com/gin-gonic/gin"

	"ticket-system/internal/handler"
	"ticket-system/internal/middleware"
	"ticket-system/internal/service"
)

// Router 负责把路由绑定到对应 handler。
// 单一职责：仅注册路由与中间件，不含任何业务逻辑。
type Router struct {
	auth    *handler.AuthHandler
	ticket  *handler.TicketHandler
	stats   *handler.StatsHandler
	authSvc *service.AuthService
}

func NewRouter(auth *handler.AuthHandler, ticket *handler.TicketHandler, stats *handler.StatsHandler, authSvc *service.AuthService) *Router {
	return &Router{auth: auth, ticket: ticket, stats: stats, authSvc: authSvc}
}

// Register 在 gin engine 上注册全部路由。
func (r *Router) Register(e *gin.Engine) {
	api := e.Group("/api/v1")

	// 认证（公开）
	authG := api.Group("/auth")
	authG.POST("/register", r.auth.Register)
	authG.POST("/login", r.auth.Login)

	// 需要登录
	authed := api.Group("")
	authed.Use(middleware.Auth(r.authSvc))
	{
		authed.GET("/auth/me", r.auth.Me)

		// 参考数据
		authed.GET("/categories", r.ticket.Categories)
		authed.GET("/users/handlers", r.ticket.Handlers)

		// 工单：特殊路径优先注册（避免 :id 匹配）
		authed.GET("/tickets/my", r.ticket.MySubmitted)
		authed.GET("/tickets/pending", r.ticket.Pending)
		authed.GET("/tickets/processed", r.ticket.Processed)
		authed.GET("/tickets/overdue", r.ticket.Overdue)
		authed.GET("/tickets", r.ticket.List)

		authed.POST("/tickets", r.ticket.Create)
		authed.GET("/tickets/:id", r.ticket.Get)
		authed.PUT("/tickets/:id", r.ticket.Update)
		authed.PUT("/tickets/:id/status", r.ticket.UpdateStatus)
		authed.PUT("/tickets/:id/assign", r.ticket.Assign)
		authed.POST("/tickets/:id/comments", r.ticket.AddComment)
		authed.POST("/tickets/:id/attachments", r.ticket.AddAttachment)
		authed.POST("/tickets/:id/review", r.ticket.SubmitReview)

		authed.GET("/attachments/:id", r.ticket.DownloadAttachment)

		// 统计
		authed.GET("/stats", r.stats.Dashboard)
	}
}
