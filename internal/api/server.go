package api

import (
	"context"
	"fmt"
	"net/http"
	"time"
	"uptime-go/internal/net/database"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type Server struct {
	db         *database.Database
	router     *gin.Engine
	server     *http.Server
	configPath string
}

type ServerConfig struct {
	Bind       string
	Port       string
	ConfigPath string
}

func NewServer(cfg ServerConfig, db *database.Database) *Server {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(accessLogger())

	server := &Server{
		db:         db,
		router:     router,
		configPath: cfg.ConfigPath,
		server: &http.Server{
			Addr:         fmt.Sprintf("%s:%s", cfg.Bind, cfg.Port),
			Handler:      router.Handler(),
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}

	server.setupRoutes()

	return server
}

func (s *Server) Start() error {
	log.Info().Str("address", s.server.Addr).Msg("Starting api server")

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start api server: %w", err)
	}

	return nil
}

func (s *Server) Shutdown() {
	log.Info().Msg("Stopping API server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Error stopping API server")
	}

	log.Info().Msg("API server stopped successfully")
}

func (s *Server) setupRoutes() {
	s.router.GET("/health", s.HealthCheckHandler)

	api := s.router.Group("/api/uptime-go")
	// api.GET("/config")
	api.POST("/config", s.UpdateConfigHandler)

	reportGroup := api.Group("/reports")
	reportGroup.GET("", s.GetMonitoringReport)
}

func accessLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		method := c.Request.Method

		if query != "" {
			path = path + "?" + query
		}

		logEvent := log.Info()
		if statusCode >= 400 {
			logEvent = log.Error()
		}

		logEvent.Str("method", method).
			Str("path", path).
			Int("status", statusCode).
			Str("latency", latency.String()).
			Msg("API request")
	}
}
