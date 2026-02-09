package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/task-underground/backend/internal/middleware"
	"github.com/task-underground/backend/internal/service"
)

type ClaimHandler struct {
	claimSvc service.ClaimService
}

func NewClaimHandler(claimSvc service.ClaimService) *ClaimHandler {
	return &ClaimHandler{claimSvc: claimSvc}
}

func (h *ClaimHandler) ClaimTask(c *gin.Context) {
	userID := middleware.GetUserID(c)
	taskID := c.Param("tid")

	claim, err := h.claimSvc.ClaimTask(c.Request.Context(), parseUUID(taskID), userID)
	if err != nil {
		if err == service.ErrTaskNotFound || err == service.ErrTaskNotClaimable || err == service.ErrClaimLimitReached || err == service.ErrAlreadyClaimed {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, claim)
}

func (h *ClaimHandler) GetClaim(c *gin.Context) {
	claimID := c.Param("id")

	claim, err := h.claimSvc.GetClaim(c.Request.Context(), parseUUID(claimID))
	if err != nil {
		if err == service.ErrClaimNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, claim)
}

func (h *ClaimHandler) GetClaimsByTask(c *gin.Context) {
	taskID := c.Param("tid")

	claims, err := h.claimSvc.GetClaimsByTaskID(c.Request.Context(), parseUUID(taskID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"claims": claims})
}

type SubmitCompletionRequest struct {
	Text     string `json:"text" binding:"required"`
	ImageURL string `json:"image_url"`
}

func (h *ClaimHandler) SubmitCompletion(c *gin.Context) {
	userID := middleware.GetUserID(c)
	claimID := c.Param("id")

	var req SubmitCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	claim, err := h.claimSvc.SubmitCompletion(c.Request.Context(), parseUUID(claimID), userID, req.Text, req.ImageURL)
	if err != nil {
		if err == service.ErrClaimNotFound || err == service.ErrUnauthorized {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, claim)
}

func (h *ClaimHandler) ApproveClaim(c *gin.Context) {
	ownerID := middleware.GetUserID(c)
	claimID := c.Param("id")

	err := h.claimSvc.ApproveClaim(c.Request.Context(), parseUUID(claimID), ownerID)
	if err != nil {
		if err == service.ErrClaimNotFound || err == service.ErrUnauthorized {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "claim approved"})
}

func (h *ClaimHandler) RejectClaim(c *gin.Context) {
	ownerID := middleware.GetUserID(c)
	claimID := c.Param("id")

	err := h.claimSvc.RejectClaim(c.Request.Context(), parseUUID(claimID), ownerID)
	if err != nil {
		if err == service.ErrClaimNotFound || err == service.ErrUnauthorized {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "claim rejected"})
}
