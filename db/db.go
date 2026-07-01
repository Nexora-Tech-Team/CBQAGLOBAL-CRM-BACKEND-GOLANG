package db

import (
	"database/sql"
	"erp-cbqa-global/lib/env"

	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Open() (*gorm.DB, *sql.DB, error) {
	sqlDB := PostgresqlOpen()
	var gormDB *gorm.DB
	var err error
	if env.String("GIN_MODE", "release") == "release" {
		gormDB, err = gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	} else {
		gormDB, err = gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})
	}
	if err != nil {
		return nil, nil, err
	}
	return gormDB, sqlDB, nil
}
