package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	_ "time/tzdata" // embeds the time zone database, so APP_TIMEZONE works on Windows too

	"golang.org/x/crypto/bcrypt"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/database"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/handler"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/model"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/repository"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/service"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/simulator"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/storage"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/validation"
)

type config struct {
	port          string
	dbPath        string
	cookieSecure  bool
	adminEmail    string
	adminPassword string
	uploadDir     string
	// timezone decides which calendar day "today" is for installation status.
	timezone *time.Location
	// simulateSensors writes dummy readings for IoT devices until real ones send data.
	simulateSensors bool
}

func loadConfig() (config, error) {
	secure, err := strconv.ParseBool(getenv("COOKIE_SECURE", "false"))
	if err != nil {
		return config{}, fmt.Errorf("COOKIE_SECURE must be true or false: %w", err)
	}
	simulate, err := strconv.ParseBool(getenv("SIMULATE_SENSORS", "true"))
	if err != nil {
		return config{}, fmt.Errorf("SIMULATE_SENSORS must be true or false: %w", err)
	}
	tz, err := time.LoadLocation(getenv("APP_TIMEZONE", "Asia/Jakarta"))
	if err != nil {
		return config{}, fmt.Errorf("APP_TIMEZONE must be an IANA time zone such as Asia/Jakarta: %w", err)
	}
	return config{
		port:            getenv("PORT", "8080"),
		dbPath:          getenv("DB_PATH", "./data/app.db"),
		cookieSecure:    secure,
		adminEmail:      os.Getenv("ADMIN_EMAIL"),
		adminPassword:   os.Getenv("ADMIN_PASSWORD"),
		uploadDir:       getenv("UPLOAD_DIR", "./data/uploads"),
		timezone:        tz,
		simulateSensors: simulate,
	}, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Open(ctx, cfg.dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := database.Migrate(ctx, db); err != nil {
		return err
	}
	if err := database.Seed(ctx, db); err != nil {
		return err
	}

	val, err := validation.New()
	if err != nil {
		return err
	}
	files, err := storage.NewFiles(cfg.uploadDir)
	if err != nil {
		return err
	}
	entities := service.NewEntityService(repository.NewEntityRepository(db), files)
	photos := service.NewPhotoService(repository.NewPhotoRepository(db), files, entities)
	installations := service.NewInstallationService(repository.NewInstallationRepository(db), entities, cfg.timezone)
	sensors := service.NewSensorService(repository.NewSensorRepository(db), entities)
	auth, err := service.NewAuthService(repository.NewUserRepository(db), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := seedAdmin(ctx, cfg, auth, val); err != nil {
		return err
	}
	router := handler.NewRouter(handler.Services{
		Entities:      entities,
		Auth:          auth,
		Photos:        photos,
		Installations: installations,
		Sensors:       sensors,
	}, val, handler.Options{SecureCookie: cfg.cookieSecure})

	srv := &http.Server{
		Addr:              ":" + cfg.port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	if cfg.simulateSensors {
		slog.Info("sensor simulator enabled: dummy readings every minute (SIMULATE_SENSORS=false to turn off)")
		go simulator.New(sensors, val, uint64(time.Now().UnixNano())).Run(ctx, time.Minute)
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("server listening", "addr", srv.Addr, "db", cfg.dbPath, "uploads", cfg.uploadDir, "timezone", cfg.timezone.String())
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("listen: %w", err)
	case <-ctx.Done():
	}

	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}

// seedAdmin creates the first admin from ADMIN_EMAIL and ADMIN_PASSWORD.
// Without them the server still runs, but nobody can change entities.
func seedAdmin(ctx context.Context, cfg config, auth *service.AuthService, val *validation.Validator) error {
	if cfg.adminEmail == "" && cfg.adminPassword == "" {
		slog.Warn("ADMIN_EMAIL and ADMIN_PASSWORD are not set: no admin account, entities are read-only")
		return nil
	}

	if cfg.adminEmail == "" || cfg.adminPassword == "" {
		return errors.New("ADMIN_EMAIL and ADMIN_PASSWORD must be set together (only one of them is set)")
	}

	in := model.RegisterInput{Email: cfg.adminEmail, Password: cfg.adminPassword}
	fields, err := val.Register(&in)
	if err != nil {
		return err
	}
	if fields != nil {
		// Report the rule that failed, never the password itself.
		return fmt.Errorf("invalid ADMIN_EMAIL/ADMIN_PASSWORD: %v", fields)
	}

	u, created, err := auth.EnsureAdmin(ctx, in)
	if err != nil {
		return fmt.Errorf("seed admin: %w", err)
	}
	switch {
	case created:
		slog.Info("admin account created", "email", u.Email)
	case u.Role != model.RoleAdmin:
		slog.Warn("ADMIN_EMAIL is already registered as a non-admin user; it was not promoted", "email", u.Email)
	default:
		slog.Info("admin account already exists", "email", u.Email)
	}
	return nil
}
