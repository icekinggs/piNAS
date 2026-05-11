// Command pinas — entrypoint do backend.
//
// Responsabilidades de bootstrap:
//   1. Carrega config (.env / env vars).
//   2. Abre banco SQLite (WAL).
//   3. Roda migrations.
//   4. Carrega/cria chave JWT.
//   5. Bootstrap do admin inicial se o banco estiver vazio.
//   6. Monta dependências e injeta nas camadas (Composition Root).
//   7. Sobe HTTP server + WS hub + tarefas de background.
//   8. Graceful shutdown em SIGINT/SIGTERM.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/pinas/pinas/internal/api"
	"github.com/pinas/pinas/internal/auth"
	"github.com/pinas/pinas/internal/config"
	"github.com/pinas/pinas/internal/database"
	"github.com/pinas/pinas/internal/files"
	"github.com/pinas/pinas/internal/logger"
	"github.com/pinas/pinas/internal/samba"
	"github.com/pinas/pinas/internal/storage"
	"github.com/pinas/pinas/internal/system"
	"github.com/pinas/pinas/internal/users"
	"github.com/pinas/pinas/internal/websocket"
	pjwt "github.com/pinas/pinas/pkg/jwt"
)

// migrationsDir — em produção, dentro do container.
// Pode ser sobrescrito com -X em build se necessário.
var migrationsDir = "/usr/local/share/pinas/migrations"

func main() {
	// Subcomandos administrativos (reset-admin, version, help).
	// Se foi um subcomando, sai sem subir HTTP.
	if handled, code := dispatchCLI(); handled {
		os.Exit(code)
	}

	cfg, err := config.Load()
	if err != nil {
		// Logger ainda não existe; saída direta.
		os.Stderr.WriteString("config: " + err.Error() + "\n")
		os.Exit(2)
	}

	log := logger.New(cfg.LogLevel, cfg.Env)
	log.Info("PiNAS starting", "env", cfg.Env, "bind", cfg.Bind)

	// Banco.
	db, err := database.Open(cfg.DBPath)
	if err != nil {
		log.Error("db open", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Migrations — se o diretório embarcado no container não existir,
	// tenta um fallback local (útil em dev).
	mDir := migrationsDir
	if _, err := os.Stat(mDir); err != nil {
		alt := filepath.Join("backend", "migrations")
		if _, err := os.Stat(alt); err == nil {
			mDir = alt
		} else if _, err := os.Stat("migrations"); err == nil {
			mDir = "migrations"
		}
	}
	if err := database.RunMigrations(ctx, db, mDir); err != nil {
		log.Error("migrations", "err", err)
		os.Exit(1)
	}

	// Storage jail.
	jail, err := storage.NewJail(cfg.DataDir)
	if err != nil {
		log.Error("jail", "err", err)
		os.Exit(1)
	}

	// JWT key.
	key, err := pjwt.LoadOrCreateKey(cfg.SecretsDir)
	if err != nil {
		log.Error("jwt key", "err", err)
		os.Exit(1)
	}
	jwtIssuer := pjwt.NewIssuer(key, cfg.JWTTTL)

	// Repositories e services (Repository Pattern + Service Layer).
	userRepo := users.NewRepository(db)
	userSvc := users.NewService(userRepo)

	sessionRepo := auth.NewSessionRepository(db)
	authSvc := auth.NewService(userRepo, sessionRepo, jwtIssuer, cfg.RefreshTTL)

	// Bootstrap do admin inicial.
	if err := bootstrapAdmin(ctx, log, userRepo, userSvc, cfg); err != nil {
		log.Error("bootstrap admin", "err", err)
		os.Exit(1)
	}

	// Hub WS.
	hub := websocket.NewHub(log)
	go hub.Run(ctx)
	hub.StartMetricsBroadcaster(ctx, 5*time.Second)

	// Cleanup periódico de sessões revogadas/expiradas.
	go cleanupSessions(ctx, log, sessionRepo)

	// Handlers.
	authHandler := auth.NewHandler(authSvc, userRepo, cfg.IsProduction())
	usersHandler := users.NewHandler(userSvc)
	filesHandler := files.NewHandler(jail)
	systemHandler := system.NewHandler(jail)

	// Samba — store + service + handler.
	// O store grava em SambaStateDir (default /var/lib/pinas/samba), que é
	// bind-mountado no host em /srv/pinas/samba. Um watcher no host
	// (pinas-samba-sync.path) detecta mudanças e aplica via samba-sync.sh.
	sambaStore, err := samba.NewStore(cfg.SambaStateDir)
	if err != nil {
		log.Error("samba store", "err", err)
		os.Exit(1)
	}
	sambaSvc := samba.NewService(sambaStore)
	sambaHandler := samba.NewHandler(sambaSvc)

	// Router.
	router := api.NewRouter(api.Deps{
		Logger:        log,
		JWTIssuer:     jwtIssuer,
		AuthHandler:   authHandler,
		UsersHandler:  usersHandler,
		FilesHandler:  filesHandler,
		SystemHandler: systemHandler,
		SambaHandler:  sambaHandler,
		WSHub:         hub,
		AllowedOrigin: cfg.AllowedOrigin,
	})

	server := &http.Server{
		Addr:              cfg.Bind,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		// Sem ReadTimeout/WriteTimeout globais — uploads grandes seriam mortos.
		// chi/timeout middleware + per-handler controlam.
		IdleTimeout: 120 * time.Second,
	}

	// Graceful shutdown.
	go func() {
		log.Info("HTTP listening", "addr", cfg.Bind)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server", "err", err)
			cancel()
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	select {
	case s := <-sig:
		log.Info("signal received", "sig", s.String())
	case <-ctx.Done():
	}

	log.Info("shutting down...")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutCancel()
	if err := server.Shutdown(shutCtx); err != nil {
		log.Error("server shutdown", "err", err)
	}
	cancel()
	log.Info("bye")
}

// bootstrapAdmin cria o usuário admin se o banco estiver vazio.
func bootstrapAdmin(ctx context.Context, log *slog.Logger, repo users.Repository, svc *users.Service, cfg *config.Config) error {
	count, err := repo.Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	if cfg.AdminPassword == "" {
		log.Warn("nenhum usuário no banco e PINAS_ADMIN_PASSWORD vazio — pulei bootstrap")
		return nil
	}
	u, err := svc.Create(ctx, users.CreateInput{
		Username: cfg.AdminUsername,
		Password: cfg.AdminPassword,
		Role:     users.RoleAdmin,
	})
	if err != nil {
		return err
	}
	log.Info("admin inicial criado", "username", u.Username)
	return nil
}

func cleanupSessions(ctx context.Context, log *slog.Logger, repo auth.SessionRepository) {
	t := time.NewTicker(1 * time.Hour)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := repo.Cleanup(ctx); err != nil {
				log.Warn("session cleanup", "err", err)
			}
		}
	}
}
