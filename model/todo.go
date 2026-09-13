package model

import "time"

type Todo struct {
	ID        int64     `json:"id" example:"1"`
	Title     string    `json:"title" example:"Buy milk"`
	Done      bool      `json:"done" example:"false"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateTodoInput struct {
	Title string `json:"title" binding:"required,min=1,max=500" example:"Buy milk"`
}

type UpdateTodoInput struct {
	Title string `json:"title" binding:"required,min=1,max=500" example:"Buy milk"`
	Done  bool   `json:"done" example:"true"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"not found"`
}
