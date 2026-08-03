package main

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/pressly/goose/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"infra/service-core/internal/organization"
	"infra/service-core/internal/outbox"
	"infra/service-core/internal/zitadel"
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

	baseDomain := env("INFRA_HTTP_BASE_DOMAIN", "app.localhost")
	// The master org's own host; defaults to the base domain (master on the apex).
	defaultOrgDomain := env("INFRA_DEFAULT_ORG_DOMAIN", baseDomain)

	// Seed the default (master) organization — the apex-domain org for the
	// non-SaaS case — if it does not exist yet. Idempotent.
	if err := organization.EnsureMaster(db, env("INFRA_DEFAULT_ORG_NAME", "My Organization")); err != nil {
		log.Fatalf("seed default organization: %v", err)
	}

	// Cancelled on SIGINT/SIGTERM so the outbox worker and HTTP server shut down
	// gracefully.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Outbox worker: provisions organizations in Zitadel downstream of the local
	// write. The client reads its machine-user PAT lazily (zitadel-init may not
	// have written it yet at startup).
	idp := zitadel.New(
		env("INFRA_SERVICE_CORE_M2M_FILE", "/run/infra/service-core-m2m.json"),
		env("INFRA_TENANT_APP_FILE", "/run/infra/tenant-app.json"),
	)
	orgCfg := organization.HandlerConfig{
		HTTPProtocol:     env("INFRA_HTTP_PROTOCOL", "http"),
		BaseDomain:       baseDomain,
		DefaultOrgDomain: defaultOrgDomain,
		// Master-org owner: presence of the email triggers owner provisioning on the
		// apex org (single-org deploy). Absent → SaaS-style, no seeded owner.
		DefaultOrgOwner: organization.Owner{
			Email:    env("INFRA_DEFAULT_ORG_OWNER_EMAIL", ""),
			Name:     env("INFRA_DEFAULT_ORG_OWNER_NAME", ""),
			Password: env("INFRA_DEFAULT_ORG_OWNER_PASSWORD", ""),
		},
	}
	handler := organization.NewOutboxHandler(idp, orgCfg)
	worker := outbox.NewWorker(db, handler, outbox.Config{
		PollInterval: envDuration("OUTBOX_POLL_INTERVAL", 2*time.Second),
		MaxAttempts:  envInt("OUTBOX_MAX_ATTEMPTS", 10),
	})
	go worker.Run(ctx)

	// Reconcile per-org identity-provider state in the background: a re-init can
	// reset the shared Tenant app's redirect URIs and per-org token actions. This
	// idempotent pass re-asserts them. Retry rides out the startup window before M2M
	// creds are written; it stops once a full pass clears.
	go func() {
		for {
			if err := organization.ReconcileOrgs(ctx, db, idp, orgCfg); err != nil {
				log.Printf("org reconcile incomplete, retrying in 10s: %v", err)
				select {
				case <-ctx.Done():
					return
				case <-time.After(10 * time.Second):
					continue
				}
			}
			log.Println("org reconcile complete")
			return
		}
	}()

	router := chi.NewRouter()
	router.Use(middleware.Recoverer)

	router.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	api := humachi.New(router, huma.DefaultConfig("Service Core", "v1"))
	organization.RegisterRoutes(api, db, baseDomain, defaultOrgDomain, idp)

	srv := &http.Server{
		Addr:    ":" + env("PORT", "8080"),
		Handler: router,
	}

	go func() {
		log.Printf("listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Printf("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown: %v", err)
	}
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
	// TranslateError maps driver errors to gorm sentinels (e.g. ErrDuplicatedKey)
	// so handlers can map a unique-violation to 409 without driver-specific checks.
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
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

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
		log.Printf("invalid %s=%q, using default %d", key, v, def)
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
		log.Printf("invalid %s=%q, using default %s", key, v, def)
	}
	return def
}
