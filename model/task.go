package model

import "time"

type Task struct {
	ID        int64     `json:"id" example:"1"`
	Title     string    `json:"title" example:"Buy milk"`
	Status    bool      `json:"status" example:"false"`
	Assignee  string    `json:"assignee" example:"Sara"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateTaskInput struct {
	Title    string `json:"title" binding:"required,min=1,max=500" example:"Buy milk"`
	Assignee string `json:"assignee" example:"Sara"`
}

type UpdateTaskInput struct {
	Title    string `json:"title" binding:"required,min=1,max=500" example:"Buy milk"`
	Status   bool   `json:"status" example:"true"`
	Assignee string `json:"assignee" example:"Sara"`
}

type ListTasksQuery struct {
	Limit    int    `form:"limit"`
	Offset   int    `form:"offset"`
	Status   *bool  `form:"status"`
	Assignee string `form:"assignee"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"not found"`
}
