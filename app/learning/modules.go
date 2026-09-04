package main

import (
	attendancereasons "clasenna-go-backend/src/attendance-absence-reasons"
	attendancelogs "clasenna-go-backend/src/attendance-logs"
	learninggroupmembers "clasenna-go-backend/src/learning-group-members"
	learninggroups "clasenna-go-backend/src/learning-groups"
	learningresources "clasenna-go-backend/src/learning-resources"
	quizsessions "clasenna-go-backend/src/quiz-sessions"
	"clasenna-go-backend/src/quizzes"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterAllModules(router *gin.RouterGroup, db *gorm.DB) {
	attendancelogs.RegisterAttendanceLogModule(router, db)
	attendancereasons.RegisterAttendanceAbsenceReasonModule(router, db)
	learninggroups.RegisterLearningGroupModule(router, db)
	learninggroupmembers.RegisterLearningGroupMemberModule(router, db)
	learningresources.RegisterLearningResourceModule(router, db)
	quizzes.RegisterQuizModule(router, db)
	quizsessions.RegisterQuizSessionModule(router, db)
}
