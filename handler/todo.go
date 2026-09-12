package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/homayounmohseni/todomgr/model"
	"github.com/homayounmohseni/todomgr/store"

	"github.com/gin-gonic/gin"
)

type TodoHandler struct {
	Store store.TodoStore
}

func NewTodoHandler(s store.TodoStore) *TodoHandler {
	return &TodoHandler{Store: s}
}

func (h *TodoHandler) RegisterRoutes(r *gin.Engine) {
	r.POST("/todos", h.Create)
	r.GET("/todos", h.List)
	r.GET("/todos/:id", h.Get)
	r.PUT("/todos/:id", h.Update)
	r.DELETE("/todos/:id", h.Delete)
}

func parseID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return 0, false
	}
	return id, true
}

// Create creates a new todo.
//
// @Summary Create todo
// @Tags todos
// @Accept json
// @Produce json
// @Param todo body model.CreateTodoInput true "Todo to create"
// @Success 201 {object} model.Todo
// @Failure 400 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /todos [post]
func (h *TodoHandler) Create(c *gin.Context) {
	var in model.CreateTodoInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	t, err := h.Store.Create(c.Request.Context(), in.Title)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	c.JSON(http.StatusCreated, t)
}

// List returns all todos.
//
// @Summary List todos
// @Tags todos
// @Produce json
// @Success 200 {array} model.Todo
// @Failure 500 {object} model.ErrorResponse
// @Router /todos [get]
func (h *TodoHandler) List(c *gin.Context) {
	todos, err := h.Store.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list failed"})
		return
	}
	if todos == nil {
		todos = []model.Todo{}
	}
	c.JSON(http.StatusOK, todos)
}

// Get returns a single todo by ID.
//
// @Summary Get todo
// @Tags todos
// @Produce json
// @Param id path int true "Todo ID"
// @Success 200 {object} model.Todo
// @Failure 400 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /todos/{id} [get]
func (h *TodoHandler) Get(c *gin.Context) {
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

// Update replaces a todo's title and done flag.
//
// @Summary Update todo
// @Tags todos
// @Accept json
// @Produce json
// @Param id path int true "Todo ID"
// @Param todo body model.UpdateTodoInput true "Updated fields"
// @Success 200 {object} model.Todo
// @Failure 400 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /todos/{id} [put]
func (h *TodoHandler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in model.UpdateTodoInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	t, err := h.Store.Update(c.Request.Context(), id, in.Title, in.Done)
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

// Delete removes a todo by ID.
//
// @Summary Delete todo
// @Tags todos
// @Produce json
// @Param id path int true "Todo ID"
// @Success 204 "No Content"
// @Failure 400 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /todos/{id} [delete]
func (h *TodoHandler) Delete(c *gin.Context) {
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
	c.Status(http.StatusNoContent)
}
