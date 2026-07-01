package main

import (
	"log"

	_ "github.com/joho/godotenv/autoload"

	"erp-cbqa-global/config"
	"erp-cbqa-global/db"
)

func main() {
	gormDB, sqlDB, err := db.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err = sqlDB.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	if err = config.Router(gormDB); err != nil {
		log.Fatal(err)
	}
}
