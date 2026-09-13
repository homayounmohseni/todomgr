package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/homayounmohseni/todomgr/handler"
	"github.com/homayounmohseni/todomgr/observability"
	"github.com/homayounmohseni/todomgr/store"

	_ "github.com/homayounmohseni/todomgr/docs"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Task Manager API
// @version 1.0
// @description Simple task CRUD API.
// @host localhost:8080
// @BasePath /
func NewRouter(s store.TaskStore) *gin.Engine {
	r := gin.Default()
	r.Use(observability.Middleware())
	r.GET("/healthz", Healthz)
	handler.NewTaskHandler(s).RegisterRoutes(r)
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

func redisURLFromEnv() string {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/0"
	}
	return redisURL
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
	opt, err := redis.ParseURL(redisURLFromEnv())
	if err != nil {
		log.Fatalf("redis url: %v", err)
	}
	rdb := redis.NewClient(opt)
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("redis unavailable (%v): list cache will fail open", err)
	}

	var ts store.TaskStore = store.NewPostgresStore(pool)
	ts = store.NewCachedTaskStore(ts, store.NewRedisCache(rdb), time.Minute)
	r := NewRouter(ts)

	log.Printf("listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
