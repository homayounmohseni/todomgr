package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/homayounmohseni/todomgr/handler"
	"github.com/homayounmohseni/todomgr/observability"
	"github.com/homayounmohseni/todomgr/store"

	_ "github.com/homayounmohseni/todomgr/docs"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Todo Manager API
// @version 1.0
// @description Simple todo CRUD API.
// @host localhost:8080
// @BasePath /
func NewRouter(s store.TodoStore) *gin.Engine {
	r := gin.Default()
	r.Use(observability.Middleware())
	r.GET("/healthz", Healthz)
	handler.NewTodoHandler(s).RegisterRoutes(r)
	r.GET("/metrics", Metrics)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	return r
}

// @Summary Health check
// @Tags ops
// @Produce json
// @Success 200 {object} map[string]string
// @Router /healthz [get]
func Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// @Summary Prometheus metrics
// @Tags ops
// @Produce text/plain
// @Success 200 {string} string "Prometheus exposition format"
// @Router /metrics [get]
func Metrics(c *gin.Context) {
	promhttp.Handler().ServeHTTP(c.Writer, c.Request)
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
