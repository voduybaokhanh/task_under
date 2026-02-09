package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/task-underground/backend/internal/middleware"
	"github.com/task-underground/backend/internal/service"
)

type ChatHandler struct {
	chatSvc  service.ChatService
	taskSvc  service.TaskService
	claimSvc service.ClaimService
}

func NewChatHandler(chatSvc service.ChatService, taskSvc service.TaskService, claimSvc service.ClaimService) *ChatHandler {
	return &ChatHandler{
		chatSvc:  chatSvc,
		taskSvc:  taskSvc,
		claimSvc: claimSvc,
	}
}

func (h *ChatHandler) GetChats(c *gin.Context) {
	userID := middleware.GetUserID(c)
	taskID := c.Param("tid")

	chats, err := h.chatSvc.GetChatsByTaskID(c.Request.Context(), parseUUID(taskID), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"chats": chats})
}

func (h *ChatHandler) GetOrCreateChat(c *gin.Context) {
	userID := middleware.GetUserID(c)
	taskID := parseUUID(c.Param("tid"))

	// Get task to find owner
	task, err := h.taskSvc.GetTask(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	var otherUserID uuid.UUID
	if userID == task.OwnerID {
		// User is owner, need to get claimer ID from query
		claimerIDStr := c.Query("claimer_id")
		if claimerIDStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "claimer_id required when owner requests chat"})
			return
		}
		otherUserID = parseUUID(claimerIDStr)
	} else {
		// User is claimer, other is owner
		otherUserID = task.OwnerID
	}

	chat, err := h.chatSvc.GetOrCreateChat(c.Request.Context(), taskID, userID, otherUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, chat)
}

func (h *ChatHandler) DeleteChat(c *gin.Context) {
	userID := middleware.GetUserID(c)
	chatID := c.Param("id")

	err := h.chatSvc.DeleteChat(c.Request.Context(), parseUUID(chatID), userID)
	if err != nil {
		if err == service.ErrChatNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "chat deleted"})
}

type SendMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

func (h *ChatHandler) SendMessage(c *gin.Context) {
	userID := middleware.GetUserID(c)
	chatID := c.Param("id")

	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	message, err := h.chatSvc.SendMessage(c.Request.Context(), parseUUID(chatID), userID, req.Content)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, message)
}

func (h *ChatHandler) GetMessages(c *gin.Context) {
	chatID := c.Param("id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	messages, err := h.chatSvc.GetMessages(c.Request.Context(), parseUUID(chatID), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"messages": messages})
}
