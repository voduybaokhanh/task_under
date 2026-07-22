package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/task-underground/backend/internal/cache"
	"github.com/task-underground/backend/internal/handler"
	"github.com/task-underground/backend/internal/middleware"
	"github.com/task-underground/backend/internal/repository"
	"github.com/task-underground/backend/internal/service"
	"github.com/task-underground/backend/internal/websocket"
)

func main() {
	// Database connection
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/task_underground?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Redis (optional). nil when REDIS_URL is unset/unreachable → single-instance mode.
	redisClient := cache.NewClient(os.Getenv("REDIS_URL"))
	if redisClient != nil {
		defer redisClient.Close()
	}

	// Repositories
	userRepo := repository.NewUserRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	claimRepo := repository.NewClaimRepository(db)
	chatRepo := repository.NewChatRepository(db)
	escrowRepo := repository.NewEscrowRepository(db)

	// Services
	userSvc := service.NewUserService(userRepo)
	escrowSvc := service.NewEscrowService(escrowRepo, taskRepo)
	taskSvc := service.NewTaskService(taskRepo, claimRepo, escrowSvc)
	chatSvc := service.NewChatService(chatRepo)
	claimSvc := service.NewClaimService(claimRepo, taskRepo, chatRepo, escrowSvc, userRepo)

	// WebSocket Hub
	wsHub := websocket.NewHub()
	go wsHub.Run()

	// Background job for auto-cancelling expired tasks
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if err := taskSvc.AutoCancelExpiredTasks(context.Background()); err != nil {
				log.Printf("Error auto-cancelling tasks: %v", err)
			}
		}
	}()

	// Router
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Device-ID, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Prometheus request metrics
	r.Use(middleware.Metrics())

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Prometheus scrape endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// WebSocket
	wsHandler := websocket.NewWSHandler(wsHub, userSvc)
	r.GET("/ws", middleware.AuthMiddleware(userSvc), wsHandler.HandleWebSocket)

	// API routes
	api := r.Group("/api/v1")
	api.Use(middleware.AuthMiddleware(userSvc))
	// Per-device sliding-window rate limit: 60 requests / minute (no-op without Redis).
	api.Use(middleware.PerUserRateLimit(redisClient, 60, time.Minute))

	// Handlers
	taskHandler := handler.NewTaskHandler(taskSvc)
	claimHandler := handler.NewClaimHandler(claimSvc)
	chatHandler := handler.NewChatHandler(chatSvc, taskSvc, claimSvc)
	userHandler := handler.NewUserHandler()

	// Task routes
	api.POST("/tasks", taskHandler.CreateTask)
	api.GET("/tasks", taskHandler.GetOpenTasks)
	api.GET("/tasks/search", taskHandler.SearchTasks)
	api.GET("/tasks/my", taskHandler.GetUserTasks)

	// Claim routes
	api.POST("/tasks/:tid/claims", claimHandler.ClaimTask)
	api.GET("/tasks/:tid/claims", claimHandler.GetClaimsByTask)
	api.GET("/claims/:id", claimHandler.GetClaim)
	api.POST("/claims/:id/submit", claimHandler.SubmitCompletion)
	api.POST("/claims/:id/approve", claimHandler.ApproveClaim)
	api.POST("/claims/:id/reject", claimHandler.RejectClaim)

	// Chat routes
	api.GET("/tasks/:tid/chats", chatHandler.GetChats)
	api.POST("/tasks/:tid/chats", chatHandler.GetOrCreateChat)
	api.DELETE("/chats/:id", chatHandler.DeleteChat)
	api.POST("/chats/:id/messages", chatHandler.SendMessage)
	api.GET("/chats/:id/messages", chatHandler.GetMessages)

	// Task routes (continued)
	api.GET("/task/:id", taskHandler.GetTask)

	// User routes
	api.GET("/users/me", userHandler.GetMe)

	// Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Printf("Server started on port %s", port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
