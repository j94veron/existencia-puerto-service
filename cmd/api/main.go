package main

import (
	"aca/existencias-puerto/internal/config"
	"aca/existencias-puerto/internal/handler"
	"aca/existencias-puerto/internal/middleware"
	"aca/existencias-puerto/internal/repository"
	"aca/existencias-puerto/internal/service"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	c, e := config.Load()
	if e != nil {
		log.Fatal(e)
	}
	repo, e := repository.Open(c)
	if e != nil {
		log.Fatal(e)
	}
	defer repo.DB.Close()
	gin.SetMode(os.Getenv("GIN_MODE"))
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), middleware.CORS(c.CorsOrigins))
	r.GET("/health", func(x *gin.Context) { x.JSON(200, gin.H{"status": "ok"}) })
	h := handler.Handler{Svc: service.Service{Config: c, Repo: repo}}
	api := r.Group("/api/v1")
	api.Use(middleware.JWT(c.JWTSecret, c.JWTIssuer, c.JWTAudience))
	api.GET("/origenes", h.Origins)
	api.GET("/existencias", h.One)
	api.GET("/existencias/consolidado", h.Consolidated)
	srv := &http.Server{Addr: ":" + c.Port, Handler: r, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: c.ReadTimeout, WriteTimeout: c.WriteTimeout, IdleTimeout: 60 * time.Second}
	go func() {
		log.Printf("listening on %s", c.Port)
		if e := srv.ListenAndServe(); e != nil && e != http.ErrServerClosed {
			log.Fatal(e)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
