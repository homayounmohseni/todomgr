package model

import "time"

type Todo struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateTodoInput struct {
	Title string `json:"title" binding:"required,min=1,max=500"`
}

type UpdateTodoInput struct {
	Title string `json:"title" binding:"required,min=1,max=500"`
	Done  bool   `json:"done"`
}
