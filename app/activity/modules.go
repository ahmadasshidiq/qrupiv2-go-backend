package main

import (
	"clasenna-go-backend/src/activities"
	activitycategories "clasenna-go-backend/src/activity-categories"
	activityitems "clasenna-go-backend/src/activity-items"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterAllModules(router *gin.RouterGroup, db *gorm.DB) {
	activities.RegisterActivityModule(router, db)
	activityitems.RegisterActivityItemModule(router, db)
	activitycategories.RegisterActivityCategoryModule(router, db)
}
