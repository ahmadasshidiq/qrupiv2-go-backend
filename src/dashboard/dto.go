package dashboard

type OverviewDTO struct {
	Months    int    `form:"months" binding:"omitempty,min=1,max=24"`
	StartDate string `form:"start_date" binding:"omitempty,datetime=2006-01-02"`
	EndDate   string `form:"end_date" binding:"omitempty,datetime=2006-01-02"`
}

type Response struct {
	Cached    bool           `json:"cached"`
	Generated string         `json:"generated_at"`
	Summary   map[string]any `json:"summary"`
	Rankings  []any          `json:"rankings,omitempty"`
	Data      map[string]any `json:"data"`
}

var roleAliases = map[string]string{
	"super-admin":       "super_admin",
	"super_admin":       "super_admin",
	"institution-admin": "institution_admin",
	"institution_admin": "institution_admin",
	"school-admin":      "institution_admin",
	"school_admin":      "institution_admin",
	"instructor":        "instructor",
	"teacher":           "instructor",
	"student":           "student",
	"dinas-pendidikan":  "dinas_pendidikan",
	"dinas_pendidikan":  "dinas_pendidikan",
}
