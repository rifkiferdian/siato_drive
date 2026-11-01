package main

import (
	"bufio"
	"cloud-drive/internal/config"
	"cloud-drive/internal/database"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	down := flag.Bool("down", false, "run down migration")
	flag.Parse()
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	db, err := database.Open(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	file := "migrations/000001_initial.up.sql"
	if *down {
		file = "migrations/000001_initial.down.sql"
	}
	data, err := os.ReadFile(file)
	if err != nil {
		log.Fatal(err)
	}
	for i, statement := range splitSQL(string(data)) {
		if _, err = db.ExecContext(context.Background(), statement); err != nil {
			log.Fatalf("migration statement %d: %v", i+1, err)
		}
	}
	fmt.Println("migration applied:", file)
}

func splitSQL(source string) []string {
	delimiter := ";"
	var statements []string
	var current strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(source))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToUpper(trimmed), "DELIMITER ") {
			delimiter = strings.TrimSpace(trimmed[len("DELIMITER "):])
			continue
		}
		current.WriteString(line)
		current.WriteByte('\n')
		if strings.HasSuffix(trimmed, delimiter) {
			statement := strings.TrimSpace(current.String())
			statement = strings.TrimSpace(strings.TrimSuffix(statement, delimiter))
			if statement != "" {
				statements = append(statements, statement)
			}
			current.Reset()
		}
	}
	if tail := strings.TrimSpace(current.String()); tail != "" {
		statements = append(statements, tail)
	}
	return statements
}
