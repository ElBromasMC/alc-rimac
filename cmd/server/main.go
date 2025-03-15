package main

import (
	"alc/handler/public"
	"alc/handler/util"
	middle "alc/middleware"
	"alc/service"
	"context"
	"fmt"
	"github.com/gorilla/sessions"
	"log"
	"net/http"
	"os"

	pgxuuid "github.com/jackc/pgx-gofrs-uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	e := echo.New()
	if os.Getenv("ENV") == "development" {
		e.Debug = true
	}

	// Database connection
	dburl := fmt.Sprintf("postgres://postgres:%s@db:5432/%s?sslmode=disable",
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)
	dbconfig, err := pgxpool.ParseConfig(dburl)
	if err != nil {
		log.Fatalln("Failed to parse config:", err)
	}
	dbconfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		// Register uuid type
		pgxuuid.Register(conn.TypeMap())
		return nil
	}
	dbpool, err := pgxpool.NewWithConfig(context.Background(), dbconfig)
	if err != nil {
		log.Fatalln("Failed to connect to the database:", err)
	}
	defer dbpool.Close()

	// Initialize services
	us := service.NewAuthService(dbpool)

	// Initialize handlers
	ph := public.Handler{
		AuthService: us,
	}

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.RemoveTrailingSlashWithConfig(middleware.TrailingSlashConfig{
		RedirectCode: http.StatusMovedPermanently,
	}))
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{Level: 5}))
	key := os.Getenv("SESSION_KEY")
	if key == "" {
		log.Fatalln("Missing SESSION_KEY env variable")
	}
	e.Use(session.Middleware(sessions.NewCookieStore([]byte(key))))

	authMiddleware := middle.Auth(us)

	// Static files
	static(e)

	// Page routes
	e.GET("/", ph.HandleIndexShow, authMiddleware)

	// Auth routes
	e.GET("/login", ph.HandleLoginShow)
	e.GET("/signup", ph.HandleSignupShow)
	e.POST("/login", ph.HandleLogin)
	e.POST("/signup", ph.HandleSignup)
	e.GET("/logout", ph.HandleLogout)

	// Error handler
	e.HTTPErrorHandler = util.HTTPErrorHandler

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatalln(e.Start(":" + port))
}
