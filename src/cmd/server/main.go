package main

import (
	"alc/assets"
	"alc/handler/admin"
	"alc/handler/constancia"
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
	_ "time/tzdata"

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
	cs := service.NewConstanciaService(dbpool)

	// Initialize handlers
	ph := public.Handler{
		AuthService: us,
	}

	ch := constancia.Handler{
		ConstanciaService: cs,
	}

	ah := admin.Handler{
		ConstanciaService: cs,
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
	adminMiddleware := middle.Admin
	loggedMiddleware := middle.Logged

	// Static files
	e.StaticFS("/static", echo.MustSubFS(assets.Assets, "static"))

	// Page routes
	e.GET("/", ch.HandleIndexShow, authMiddleware, loggedMiddleware)
	e.GET("/accesorios", ch.HandleAccesoriosFormShow, authMiddleware, loggedMiddleware)
	e.GET("/devolucion", ch.HandleDevolucionFormShow, authMiddleware, loggedMiddleware)

	gc := e.Group("/clonacion")
	gc.Use(authMiddleware, loggedMiddleware)
	gc.GET("", ch.HandleClonacionFormShow)
	gc.GET("/equipo", ch.HandleClonacionEquipoFetch)
	gc.POST("", ch.HandleClonacionInsert)
	gc.GET("/report", ch.HandleEquiposReportDownload, adminMiddleware)

	gb := e.Group("/borrado")
	gb.Use(authMiddleware, loggedMiddleware)
	gb.GET("", ch.HandleBorradoFormShow)
	gb.GET("/inventario", ch.HandleBorradoInventarioFetch)
	gb.POST("", ch.HandleBorradoInsert)
	gb.GET("/report", ch.HandleBorradosReportDownload, adminMiddleware)
	gb.GET("/zip", ch.DownloadZipHandler, adminMiddleware)

	e.GET("/cliente", ch.HandleUsuarioFetch, authMiddleware, loggedMiddleware)
	e.GET("/equipo", ch.HandleEquipoFetch, authMiddleware, loggedMiddleware)

	e.POST("/constancia", ch.HandleConstanciaInsert, authMiddleware, loggedMiddleware)
	e.PUT("/constancia", ch.HandleConstanciaUpdate, authMiddleware, loggedMiddleware)

	e.GET("/download", ch.DownloadPDFHandler, authMiddleware, loggedMiddleware)

	// Auth routes
	e.GET("/login", ph.HandleLoginShow)
	e.POST("/login", ph.HandleLogin)
	e.GET("/logout", ph.HandleLogout)

	// Admin routes
	g1 := e.Group("/admin")
	g1.Use(authMiddleware, adminMiddleware)
	g1.GET("", ah.HandleIndexShow)
	g1.POST("/equipos", ah.HandleEquiposInsertion)
	g1.POST("/clientes", ah.HandleClientesInsertion)
	g1.GET("/constancias", ah.HandleConstanciasDownload)
	g1.GET("/signup", ph.HandleSignupShow)
	g1.POST("/signup", ph.HandleSignup)

	// Error handler
	e.HTTPErrorHandler = util.HTTPErrorHandler

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatalln(e.Start(":" + port))
}
