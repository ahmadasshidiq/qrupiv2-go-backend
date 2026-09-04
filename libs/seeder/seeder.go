package seeder

import "clasenna-go-backend/libs/models"

var MasterPermissions = []models.PermissionItem{
	{Model: "dashboard", Action: "get-overview"},
	{Model: "dashboard", Action: "get-tenant-overview"},
	{Model: "dashboard", Action: "get-mobile-overview"},

	{Model: "users", Action: "get-all"},
	{Model: "users", Action: "get-by-id"},
	{Model: "users", Action: "create"},
	{Model: "users", Action: "update"},
	{Model: "users", Action: "delete"},
	{Model: "users", Action: "export"},
	{Model: "users", Action: "import"},

	{Model: "roles", Action: "get-all"},
	{Model: "roles", Action: "get-by-id"},
	{Model: "roles", Action: "create"},
	{Model: "roles", Action: "update"},
	{Model: "roles", Action: "delete"},
	{Model: "roles", Action: "export"},

	{Model: "institutions", Action: "get-all"},
	{Model: "institutions", Action: "get-by-id"},
	{Model: "institutions", Action: "create"},
	{Model: "institutions", Action: "update"},
	{Model: "institutions", Action: "delete"},
	{Model: "institutions", Action: "export"},

	{Model: "notifications", Action: "get-all"},
	{Model: "notifications", Action: "get-by-id"},
	{Model: "notifications", Action: "create"},
	{Model: "notifications", Action: "update"},
	{Model: "notifications", Action: "export"},

	{Model: "activities", Action: "get-all"},
	{Model: "activities", Action: "get-by-id"},
	{Model: "activities", Action: "create"},
	{Model: "activities", Action: "update"},
	{Model: "activities", Action: "delete"},

	{Model: "activity-items", Action: "get-all"},
	{Model: "activity-items", Action: "get-by-id"},
	{Model: "activity-items", Action: "create"},
	{Model: "activity-items", Action: "update"},
	{Model: "activity-items", Action: "delete"},

	{Model: "activity-categories", Action: "get-all"},
	{Model: "activity-categories", Action: "get-by-id"},
	{Model: "activity-categories", Action: "create"},
	{Model: "activity-categories", Action: "update"},
	{Model: "activity-categories", Action: "delete"},

	{Model: "learning-groups", Action: "get-all"},
	{Model: "learning-groups", Action: "get-by-id"},
	{Model: "learning-groups", Action: "create"},
	{Model: "learning-groups", Action: "update"},
	{Model: "learning-groups", Action: "delete"},

	{Model: "learning-group-members", Action: "get-all"},
	{Model: "learning-group-members", Action: "get-by-id"},
	{Model: "learning-group-members", Action: "create"},
	{Model: "learning-group-members", Action: "update"},
	{Model: "learning-group-members", Action: "delete"},

	{Model: "learning-resources", Action: "get-all"},
	{Model: "learning-resources", Action: "get-by-id"},
	{Model: "learning-resources", Action: "create"},
	{Model: "learning-resources", Action: "update"},
	{Model: "learning-resources", Action: "delete"},

	{Model: "quizzes", Action: "get-all"},
	{Model: "quizzes", Action: "get-by-id"},
	{Model: "quizzes", Action: "create"},
	{Model: "quizzes", Action: "update"},
	{Model: "quizzes", Action: "delete"},

	{Model: "quiz-sessions", Action: "get-all"},
	{Model: "quiz-sessions", Action: "get-by-id"},
	{Model: "quiz-sessions", Action: "create"},
	{Model: "quiz-sessions", Action: "update"},
	{Model: "quiz-sessions", Action: "delete"},

	{Model: "attendance-logs", Action: "get-all"},
	{Model: "attendance-logs", Action: "get-by-id"},
	{Model: "attendance-logs", Action: "create"},
	{Model: "attendance-logs", Action: "update"},
	{Model: "attendance-logs", Action: "delete"},

	{Model: "attendance-absence-reasons", Action: "get-all"},
	{Model: "attendance-absence-reasons", Action: "get-by-id"},
	{Model: "attendance-absence-reasons", Action: "create"},
	{Model: "attendance-absence-reasons", Action: "update"},
	{Model: "attendance-absence-reasons", Action: "delete"},

	{Model: "provinces", Action: "get-all"},
	{Model: "provinces", Action: "get-by-id"},
	{Model: "regencies", Action: "get-all"},
	{Model: "regencies", Action: "get-by-id"},
	{Model: "districts", Action: "get-all"},
	{Model: "districts", Action: "get-by-id"},
	{Model: "villages", Action: "get-all"},
	{Model: "villages", Action: "get-by-id"},
}
