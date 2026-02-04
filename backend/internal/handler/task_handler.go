package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/task-underground/backend/internal/middleware"
	"github.com/task-underground/backend/internal/service"
)

type TaskHandler struct {
	taskSvc service.TaskService
}

func NewTaskHandler(taskSvc service.TaskService) *TaskHandler {
	return &TaskHandler{taskSvc: taskSvc}
}

type CreateTaskRequest struct {
	Title         string  `json:"title" binding:"required"`
	Description   string  `json:"description" binding:"required"`
	RewardAmount  float64 `json:"reward_amount" binding:"required,gt=0"`
	MaxClaimants  int     `json:"max_claimants" binding:"required,gt=0"`
	ClaimDeadline string  `json:"claim_deadline" binding:"required"`
	OwnerDeadline string  `json:"owner_deadline" binding:"required"`
}

func (h *TaskHandler) CreateTask(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	claimDeadline, err := parseTime(req.ClaimDeadline)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid claim_deadline format"})
		return
	}

	ownerDeadline, err := parseTime(req.OwnerDeadline)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid owner_deadline format"})
		return
	}

	svcReq := service.CreateTaskRequest{
		Title:         req.Title,
		Description:   req.Description,
		RewardAmount:  req.RewardAmount,
		MaxClaimants:  req.MaxClaimants,
		ClaimDeadline: claimDeadline,
		OwnerDeadline: ownerDeadline,
	}

	task, err := h.taskSvc.CreateTask(c.Request.Context(), userID, svcReq)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, task)
}

func (h *TaskHandler) GetTask(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task id required"})
		return
	}

	task, err := h.taskSvc.GetTask(c.Request.Context(), parseUUID(taskID))
	if err != nil {
		if err == service.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, task)
}

func (h *TaskHandler) GetOpenTasks(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	tasks, err := h.taskSvc.GetOpenTasks(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tasks": tasks})
}

func (h *TaskHandler) GetUserTasks(c *gin.Context) {
	userID := middleware.GetUserID(c)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	tasks, err := h.taskSvc.GetUserTasks(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tasks": tasks})
}
