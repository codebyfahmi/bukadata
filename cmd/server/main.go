package main

import (
	"bufio"
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"

	"bukadata/app"
	"bukadata/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	loadDotEnv(".env")

	cfg := config.MustLoad()

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("database ping failed: %v", err)
	}
	log.Println("database connected:", cfg.DatabaseURL[:strings.Index(cfg.DatabaseURL, "@")+1]+"***")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	log.Println("server listening on http://localhost" + addr)
	if err := http.ListenAndServe(addr, app.New()); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func loadDotEnv(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}
