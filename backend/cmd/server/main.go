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
	"github.com/task-underground/backend/internal/handler"
	"github.com/task-underground/backend/internal/middleware"
	"github.com/task-underground/backend/internal/repository"
	"github.com/task-underground/backend/internal/service"
	"github.com/task-underground/backend/internal/websocket"
	"golang.org/x/time/rate"
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

	// Rate limiting
	limiter := rate.NewLimiter(rate.Every(time.Second), 10)
	r.Use(func(c *gin.Context) {
		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			c.Abort()
			return
		}
		c.Next()
	})

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// WebSocket
	wsHandler := websocket.NewWSHandler(wsHub, userSvc)
	r.GET("/ws", middleware.AuthMiddleware(userSvc), wsHandler.HandleWebSocket)

	// API routes
	api := r.Group("/api/v1")
	api.Use(middleware.AuthMiddleware(userSvc))

	// Handlers
	taskHandler := handler.NewTaskHandler(taskSvc)
	claimHandler := handler.NewClaimHandler(claimSvc)
	chatHandler := handler.NewChatHandler(chatSvc, taskSvc, claimSvc)

	// Task routes
	api.POST("/tasks", taskHandler.CreateTask)
	api.GET("/tasks", taskHandler.GetOpenTasks)
	api.GET("/tasks/my", taskHandler.GetUserTasks)
	api.GET("/tasks/:id", taskHandler.GetTask)

	// Claim routes
	api.POST("/tasks/:task_id/claims", claimHandler.ClaimTask)
	api.GET("/tasks/:task_id/claims", claimHandler.GetClaimsByTask)
	api.GET("/claims/:id", claimHandler.GetClaim)
	api.POST("/claims/:id/submit", claimHandler.SubmitCompletion)
	api.POST("/claims/:id/approve", claimHandler.ApproveClaim)
	api.POST("/claims/:id/reject", claimHandler.RejectClaim)

	// Chat routes
	api.GET("/tasks/:task_id/chats", chatHandler.GetChats)
	api.POST("/tasks/:task_id/chats", chatHandler.GetOrCreateChat)
	api.DELETE("/chats/:id", chatHandler.DeleteChat)
	api.POST("/chats/:id/messages", chatHandler.SendMessage)
	api.GET("/chats/:id/messages", chatHandler.GetMessages)

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
