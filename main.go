package main

import (
	"context"
	"log"
	"os"

	"github.com/homayounmohseni/todomgr/handler"
	"github.com/homayounmohseni/todomgr/store"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewRouter(s store.TodoStore) *gin.Engine {
	r := gin.Default()
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	handler.NewTodoHandler(s).RegisterRoutes(r)
	return r
}

func configFromEnv() (dbURL, port string) {
	dbURL = os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://todo:todo@localhost:5432/todo?sslmode=disable"
	}
	port = os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return dbURL, port
}

func main() {
	dbURL, port := configFromEnv()

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("db ping: %v", err)
	}
	if err := store.Migrate(ctx, pool); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	r := NewRouter(store.NewPostgresStore(pool))

	log.Printf("listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
