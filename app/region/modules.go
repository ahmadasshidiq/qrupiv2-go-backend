package main

import (
	"clasenna-go-backend/src/regions"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterAllModules(router *gin.RouterGroup, db *gorm.DB) {
	regions.RegisterRegionModule(router, db)
}
