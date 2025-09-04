package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/turfaa/apotek-hris/internal/attendance"
	"github.com/turfaa/apotek-hris/internal/hris"
	"github.com/turfaa/apotek-hris/internal/salary"
	"github.com/turfaa/apotek-hris/pkg/httpx"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jmoiron/sqlx"
)

type Server struct {
	config Config
	db     *sqlx.DB
	server *http.Server
}

func New(config Config, db *sqlx.DB) *Server {
	return &Server{
		config: config,
		db:     db,
	}
}

func (s *Server) Start() error {
	r := chi.NewRouter()
	s.setupMiddleware(r)
	s.setupRoutes(r)

	s.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.config.Host, s.config.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *Server) setupMiddleware(r *chi.Mux) {
	// Basic CORS setup to allow all origins
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))
	r.Use(middleware.Timeout(60 * time.Second))
}

func (s *Server) setupRoutes(r *chi.Mux) {
	r.Get("/health", s.handleHealth())

	hrisService := hris.NewService(s.db)
	attendanceService := attendance.NewService(s.db)
	salaryService := salary.NewService(s.db, hrisService, attendanceService)

	hrisHandler := hris.NewHandler(hrisService)
	attendanceHandler := attendance.NewHandler(attendanceService)
	salaryHandler := salary.NewHandler(salaryService)

	r.Group(func(r chi.Router) {
		r.Route("/api/v1", func(r chi.Router) {
			hrisHandler.RegisterRoutes(r)
			attendanceHandler.RegisterRoutes(r)
			salaryHandler.RegisterRoutes(r)
		})
	})
}

func (s *Server) handleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := s.db.Ping(); err != nil {
			httpx.Error(w, err, http.StatusServiceUnavailable)
			return
		}

		httpx.Ok(w, map[string]string{"status": "ok"})
	}
}
