package main

import (
	"database/sql"
	"embed"
	"log"
	"net/http"
	"os"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/pressly/goose/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"infra/service-core/internal/organization"
)

//go:embed migrations/*.sql
var migrations embed.FS

func main() {
	log.SetFlags(0)
	log.SetPrefix("[service-core] ")

	dsn := env("DATABASE_URL", "")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}

	mustMigrate(dsn)

	db := mustDB(dsn)

	router := chi.NewRouter()
	router.Use(middleware.Recoverer)

	router.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	api := humachi.New(router, huma.DefaultConfig("Service Core", "v1"))
	organization.RegisterRoutes(api, db)

	addr := ":" + env("PORT", "8080")
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, router))
}

func mustMigrate(dsn string) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("migrate: open: %v", err)
	}
	defer db.Close()

	goose.SetBaseFS(migrations)
	goose.SetLogger(goose.NopLogger())

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("migrate: dialect: %v", err)
	}
	if err := goose.Up(db, "migrations"); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	log.Printf("migrations applied")
}

func mustDB(dsn string) *gorm.DB {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	return db
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
