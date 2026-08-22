package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"ticket-system/api"
	"ticket-system/config"
	"ticket-system/internal/database"
	"ticket-system/internal/handler"
	"ticket-system/internal/middleware"
	"ticket-system/internal/repository"
	"ticket-system/internal/service"
)

// main 是组合根：装配依赖并启动服务。
// 单一职责：仅负责依赖组装与生命周期管理，不含业务逻辑。
func main() {
	// 1. 配置
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 2. 日志
	log := newLogger(cfg.Log)
	defer log.Sync()

	// 3. 上传目录
	if err := os.MkdirAll(cfg.Upload.Dir, 0o755); err != nil {
		log.Fatal("创建上传目录失败", zap.Error(err))
	}

	// 4. 数据库
	db, err := database.Connect(cfg.Database.DSN, log)
	if err != nil {
		log.Fatal("连接数据库失败", zap.Error(err))
	}
	if err := database.Seed(db, log); err != nil {
		log.Fatal("初始化数据失败", zap.Error(err))
	}

	// 5. 仓库
	userRepo := repository.NewUserRepository(db)
	ticketRepo := repository.NewTicketRepository(db)
	catRepo := repository.NewCategoryRepository(db)
	cmtRepo := repository.NewCommentRepository(db)
	attRepo := repository.NewAttachmentRepository(db)
	reviewRepo := repository.NewReviewRepository(db)

	// 6. 服务
	authSvc := service.NewAuthService(userRepo, cfg.JWT.Secret, cfg.JWT.ExpireHours)
	ticketSvc := service.NewTicketService(db, ticketRepo, catRepo, cmtRepo, attRepo, cfg.Upload.Dir)
	assignSvc := service.NewAssignService(db, ticketRepo, userRepo)
	reviewSvc := service.NewReviewService(db, reviewRepo, ticketRepo)
	catSvc := service.NewCategoryService(catRepo)
	statsSvc := service.NewStatsService(ticketRepo, userRepo)

	// 7. Handler
	authH := handler.NewAuthHandler(authSvc)
	ticketH := handler.NewTicketHandler(ticketSvc, assignSvc, reviewSvc, catSvc)
	statsH := handler.NewStatsHandler(statsSvc)

	// 8. Gin 引擎
	gin.SetMode(cfg.Server.Mode)
	e := gin.New()
	e.Use(middleware.Logger(log))
	e.Use(middleware.CORS())
	e.Use(gin.Recovery())

	// 9. 路由
	r := api.NewRouter(authH, ticketH, statsH, authSvc)
	r.Register(e)

	// 10. 静态前端页面
	e.Static("/static", filepath.Join("web", "static"))
	e.StaticFile("/", filepath.Join("web", "index.html"))
	e.StaticFile("/login", filepath.Join("web", "login.html"))
	e.StaticFile("/register", filepath.Join("web", "register.html"))
	e.StaticFile("/tickets", filepath.Join("web", "tickets.html"))
	e.StaticFile("/tickets/new", filepath.Join("web", "ticket-new.html"))
	e.StaticFile("/tickets/detail", filepath.Join("web", "ticket-detail.html"))
	e.StaticFile("/my-tickets", filepath.Join("web", "my-tickets.html"))
	e.StaticFile("/pending", filepath.Join("web", "pending.html"))
	e.StaticFile("/dashboard", filepath.Join("web", "dashboard.html"))
	e.StaticFile("/overdue", filepath.Join("web", "overdue.html"))
	e.Static("/uploads", cfg.Upload.Dir)

	// 11. 启动
	addr := ":" + cfg.Server.Port
	log.Info("服务启动", zap.String("addr", addr))
	if err := e.Run(addr); err != nil {
		log.Fatal("服务退出", zap.Error(err))
	}
}

// newLogger 创建按配置输出的 zap 日志器。
// 单一职责：仅负责日志器构造。
func newLogger(cfg config.LogConfig) *zap.Logger {
	_ = os.MkdirAll(cfg.Dir, 0o755)
	path := filepath.Join(cfg.Dir, cfg.Filename)

	lvl := zap.NewAtomicLevelAt(zap.InfoLevel)
	switch cfg.Level {
	case "debug":
		lvl = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "warn":
		lvl = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		lvl = zap.NewAtomicLevelAt(zap.ErrorLevel)
	}

	zcfg := zap.NewProductionConfig()
	zcfg.Level = lvl
	zcfg.OutputPaths = []string{path, "stdout"}
	return zap.Must(zcfg.Build())
}
