package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/homayounmohseni/todomgr/model"
	"github.com/homayounmohseni/todomgr/observability"
	"github.com/homayounmohseni/todomgr/store"

	"github.com/gin-gonic/gin"
)

type TaskHandler struct {
	Store store.TaskStore
}

func NewTaskHandler(s store.TaskStore) *TaskHandler {
	return &TaskHandler{Store: s}
}

func (h *TaskHandler) RegisterRoutes(r *gin.Engine) {
	r.POST("/tasks", h.Create)
	r.GET("/tasks", h.List)
	r.GET("/tasks/:id", h.Get)
	r.PUT("/tasks/:id", h.Update)
	r.DELETE("/tasks/:id", h.Delete)
}

func parseID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return 0, false
	}
	return id, true
}

// @Summary Create task
// @Tags tasks
// @Accept json
// @Produce json
// @Param task body model.CreateTaskInput true "Task to create"
// @Success 201 {object} model.Task
// @Failure 400 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /tasks [post]
func (h *TaskHandler) Create(c *gin.Context) {
	var in model.CreateTaskInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	t, err := h.Store.Create(c.Request.Context(), in.Title, in.Assignee)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	observability.TasksInc()
	c.JSON(http.StatusCreated, t)
}

// @Summary List tasks
// @Tags tasks
// @Produce json
// @Param limit query int false "Page size (1-100, default 20)"
// @Param offset query int false "Rows to skip (default 0)"
// @Param status query boolean false "Filter by status"
// @Param assignee query string false "Filter by assignee (exact match)"
// @Success 200 {array} model.Task
// @Failure 400 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /tasks [get]
func (h *TaskHandler) List(c *gin.Context) {
	var q model.ListTasksQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	limit := q.Limit
	if limit == 0 {
		limit = 20
	}
	if limit < 1 || limit > 100 || q.Offset < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pagination"})
		return
	}
	tasks, err := h.Store.List(c.Request.Context(), store.ListFilter{
		Limit:    limit,
		Offset:   q.Offset,
		Status:   q.Status,
		Assignee: q.Assignee,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list failed"})
		return
	}
	if tasks == nil {
		tasks = []model.Task{}
	}
	observability.TasksSet(float64(len(tasks)))
	c.JSON(http.StatusOK, tasks)
}

// @Summary Get task
// @Tags tasks
// @Produce json
// @Param id path int true "Task ID"
// @Success 200 {object} model.Task
// @Failure 400 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /tasks/{id} [get]
func (h *TaskHandler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	t, err := h.Store.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get failed"})
		return
	}
	c.JSON(http.StatusOK, t)
}

// @Summary Update task
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path int true "Task ID"
// @Param task body model.UpdateTaskInput true "Updated fields"
// @Success 200 {object} model.Task
// @Failure 400 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /tasks/{id} [put]
func (h *TaskHandler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in model.UpdateTaskInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	t, err := h.Store.Update(c.Request.Context(), id, in.Title, in.Status, in.Assignee)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	c.JSON(http.StatusOK, t)
}

// @Summary Delete task
// @Tags tasks
// @Produce json
// @Param id path int true "Task ID"
// @Success 204 "No Content"
// @Failure 400 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /tasks/{id} [delete]
func (h *TaskHandler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.Store.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}
	observability.TasksDec()
	c.Status(http.StatusNoContent)
}
