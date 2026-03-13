package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/cors"
)

// Server API服务器
type Server struct {
	port       int
	host       string
	router     *gin.Engine
	httpServer *http.Server
	handler    *Handler
}

// NewServer 创建新的API服务器
func NewServer(port int, host string, handler *Handler) *Server {
	// 设置Gin为发布模式
	gin.SetMode(gin.ReleaseMode)
	
	router := gin.New()
	router.Use(gin.Recovery())
	
	server := &Server{
		port:    port,
		host:    host,
		router:  router,
		handler: handler,
	}
	
	// 设置路由
	server.setupRoutes()
	
	return server
}

// setupRoutes 设置路由
func (s *Server) setupRoutes() {
	// 健康检查
	s.router.GET("/health", s.healthCheck)
	
	// API路由组
	api := s.router.Group("/api")
	{
		// 机器人状态
		api.GET("/status", s.handler.GetStatus)

		// 收益数据
		api.GET("/earnings", s.handler.GetEarnings)
		api.GET("/earnings/daily", s.handler.GetDailyEarnings)
		api.GET("/earnings/weekly", s.handler.GetWeeklyEarnings)
		api.GET("/earnings/yearly", s.handler.GetYearlyEarnings)

		// FRR 利率
		api.GET("/frr-rates", s.handler.GetFRRRates)

		// 放贷订单
		api.GET("/offers", s.handler.GetOffers)
		api.GET("/offers/active", s.handler.GetActiveOffers)

		// 已贷出订单
		api.GET("/credits", s.handler.GetFundingCredits)

		// 钱包信息
		api.GET("/wallets", s.handler.GetWallets)

		// 配置管理
		api.GET("/config", s.handler.GetConfig)
		api.POST("/config", s.handler.UpdateConfig)

		// 机器人控制
		api.POST("/control", s.handler.Control)

		// 日志
		api.GET("/logs", s.handler.GetLogs)

		// 系统信息
		api.GET("/info", s.handler.GetSystemInfo)
	}
	
	// 静态文件服务 (Web界面)
	s.router.Static("/static", "./web/static")
	s.router.StaticFile("/", "./web/index.html")
	s.router.StaticFile("/favicon.ico", "./web/static/favicon.ico")
}

// healthCheck 健康检查
func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: "BitfinexBot API is running",
		Data: map[string]interface{}{
			"timestamp": time.Now(),
			"version":   "2.7.0",
		},
	})
}

// Start 启动服务器
func (s *Server) Start() error {
	// 配置CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
		ExposedHeaders: []string{"*"},
		MaxAge:         300,
	})
	
	handler := c.Handler(s.router)
	
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.host, s.port),
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	
	fmt.Printf("🌐 API服务器启动在 http://%s:%d\n", s.host, s.port)
	fmt.Printf("📊 Web界面访问: http://%s:%d\n", s.host, s.port)
	
	return s.httpServer.ListenAndServe()
}

// Stop 停止服务器
func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// GetRouter 获取路由器 (用于测试)
func (s *Server) GetRouter() *gin.Engine {
	return s.router
}