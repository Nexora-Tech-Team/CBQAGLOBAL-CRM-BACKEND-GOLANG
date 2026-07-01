package db

import (
	"database/sql"
	"log"

	_ "github.com/joho/godotenv/autoload"

	"erp-cbqa-global/lib/env"
)

func PostgresqlOpen() *sql.DB {
	sqlDb, err := sql.Open("postgres", env.String("POSTGRESQL_URL", "postgresql://127.0.0.1:5432/base_code?sslmode=disable"))
	if err != nil {
		log.Fatal(err)
	}
	return sqlDb
}
