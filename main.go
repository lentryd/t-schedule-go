package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"t-schedule/internal/config"
	"t-schedule/internal/pkg/gcalendar"
	"t-schedule/internal/schedule"
	"t-schedule/internal/store"
	"t-schedule/internal/telegram"

	"flag"

	"github.com/joho/godotenv"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func main() {
	debug := flag.Bool("debug", false, "Enable debug logging")
	version := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *version {
		fmt.Printf("t-schedule %s (commit: %s, built: %s)\n", Version, Commit, Date)
		return
	}

	_ = godotenv.Load()

	level := slog.LevelInfo
	if *debug {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))

	cfg := config.New(*debug)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	projectID := cfg.FirestoreProjectID
	if projectID == "" {
		projectID = store.DetectProjectID
	}

	st, err := store.New(ctx, projectID, cfg.CredentialsPath)
	if err != nil {
		log.Fatalf("failed to init firestore: %v", err)
	}
	defer func() {
		if err := st.Close(); err != nil {
			slog.Error("failed to close firestore client", "error", err)
		}
	}()

	gcal, err := gcalendar.New(ctx, cfg.CredentialsPath)
	if err != nil {
		log.Fatalf("failed to init google calendar: %v", err)
	}

	syncer := schedule.NewSyncer(st, gcal)
	studentUpdater := schedule.NewStudentListUpdater(st)
	scheduler := schedule.NewScheduler(syncer, studentUpdater)

	if err := scheduler.Start(ctx); err != nil {
		log.Fatalf("failed to start scheduler: %v", err)
	}
	defer scheduler.Stop()

	b, err := telegram.New(cfg, st, gcal, studentUpdater)
	if err != nil {
		log.Fatalf("failed to init bot: %v", err)
	}

	slog.Info("starting", "version", Version, "commit", Commit)
	b.Start(ctx)
}
