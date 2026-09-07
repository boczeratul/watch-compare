// Command migrate applies (up) or reverts (down) embedded SQL migrations.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/hsuanlee/watch-compare/backend/internal/config"
	"github.com/hsuanlee/watch-compare/backend/internal/db"
)

func main() {
	dir := "up"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(2)
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "database:", err)
		os.Exit(1)
	}
	defer pool.Close()
	switch dir {
	case "up":
		err = db.Migrate(ctx, pool)
	case "down":
		err = db.Rollback(ctx, pool)
	default:
		err = fmt.Errorf("usage: migrate [up|down]")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("migrate", dir, "ok")
}
