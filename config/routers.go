package config

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"erp-cbqa-global/app/controllers/root"
	"erp-cbqa-global/config/collection"
)

func Router(DB *gorm.DB) error {
	router := gin.Default()
	corsConfig(router)
	router.Static("/public", "./public")
	router.GET("/", root.Index)
	dashboard := router.Group("/api")
	collection.Router(DB, dashboard)

	if err := router.Run(); err != nil {
		return err
	}
	return nil
}
