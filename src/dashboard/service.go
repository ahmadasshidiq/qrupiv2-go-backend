package dashboard

import (
	"clasenna-go-backend/libs/helpers"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type Aggregator struct {
	Client *http.Client
	Cache  *redis.Client
	TTL    time.Duration
}

func NewAggregator(cache *redis.Client) *Aggregator {
	ttl := time.Minute
	if d, err := time.ParseDuration(os.Getenv("DASHBOARD_CACHE_TTL")); err == nil && d > 0 {
		ttl = d
	}
	return &Aggregator{Client: &http.Client{Timeout: 8 * time.Second}, Cache: cache, TTL: ttl}
}

func (s *Aggregator) Get(ctx context.Context, headers http.Header, query url.Values) (*Response, error) {
	role := headers.Get("X-Role")
	if role == "" {
		role = headers.Get("X-Role-Name")
	}
	if role == "" && headers.Get("X-Role-ID") != "" {
		resolved, err := s.resolveRole(ctx, headers, headers.Get("X-Role-ID"))
		if err != nil {
			return nil, err
		}
		role = resolved
	}
	// Never trust a client-supplied user_id for personal dashboards.
	// The gateway injects X-User-ID from the validated token.
	effectiveQuery := cloneValues(query)
	if isStudentRole(role) {
		effectiveQuery.Set("user_id", headers.Get("X-User-ID"))
	}
	if institutionID := headers.Get("X-Institution-ID"); institutionID != "" {
		effectiveQuery.Set("institution_id", institutionID)
	}
	// Dashboard aggregates need enough rows to calculate rankings. Upstream
	// services cap this value at their own safe maximum.
	if effectiveQuery.Get("limit") == "" {
		effectiveQuery.Set("limit", "100")
	}
	if isInstructorRole(role) {
		groups, err := s.instructorGroups(ctx, headers, effectiveQuery)
		if err != nil {
			return nil, err
		}
		if len(groups) == 0 {
			effectiveQuery.Set("learning_group_id", "00000000-0000-0000-0000-000000000000")
		} else {
			effectiveQuery.Set("learning_group_id.in", strings.Join(groups, ","))
		}
	}
	if isStudentRole(role) && effectiveQuery.Get("scope") == "" {
		effectiveQuery.Set("scope", "school")
	}
	h := sha256.Sum256([]byte(role + "|" + headers.Get("X-User-ID") + "|" + headers.Get("X-Institution-ID") + "|" + effectiveQuery.Encode()))
	key := "dashboard:v1:" + hex.EncodeToString(h[:])
	if s.Cache != nil {
		if b, err := s.Cache.Get(ctx, key).Bytes(); err == nil {
			var r Response
			if json.Unmarshal(b, &r) == nil {
				r.Cached = true
				return &r, nil
			}
		}
	}
	paths := map[string]string{"users": "/users", "institutions": "/institutions", "learning_groups": "/learning-groups", "learning_group_members": "/learning-group-members", "activities": "/activities", "quiz_sessions": "/quiz-sessions", "attendance": "/attendance-logs"}
	paths["quiz_rankings"] = "/quiz-sessions/rankings"
	data := map[string]any{}
	for name, path := range paths {
		if path == "/quiz-sessions/rankings" && effectiveQuery.Get("scope") == "" {
			data[name] = map[string]any{"available": false, "reason": "ranking scope is not requested"}
			continue
		}
		v, err := s.fetch(ctx, headers, path, effectiveQuery)
		if err != nil {
			data[name] = map[string]any{"available": false, "error": err.Error()}
		} else {
			data[name] = v
		}
	}
	r := &Response{Role: role, RoleLabel: roleLabel(role), Generated: time.Now().UTC().Format(time.RFC3339), Summary: summarize(role, data), Rankings: buildRankings(role, data), Alerts: buildAlerts(role, data), Data: data}
	if b, err := json.Marshal(r); err == nil && s.Cache != nil {
		_ = s.Cache.Set(ctx, key, b, s.TTL).Err()
	}
	return r, nil
}

func roleLabel(role string) string {
	switch helpers.NormalizeRoleForDashboard(role) {
	case "super_admin":
		return helpers.RoleName("super_admin")
	case "institution_admin", "admin_sekolah", "school_admin":
		return helpers.RoleName("institution_admin")
	case "instructor", "guru", "teacher":
		return helpers.RoleName("instructor")
	case "student", "pelajar", "siswa":
		return helpers.RoleName("student")
	case "dinas_pendidikan":
		return helpers.RoleName("dinas_pendidikan")
	default:
		return role
	}
}

func cloneValues(values url.Values) url.Values {
	clone := make(url.Values, len(values))
	for key, items := range values {
		clone[key] = append([]string(nil), items...)
	}
	return clone
}

func isStudentRole(role string) bool {
	return helpers.IsRole(role, "student")
}

func isInstructorRole(role string) bool {
	return helpers.IsRole(role, "instructor")
}

func (s *Aggregator) resolveRole(ctx context.Context, headers http.Header, roleID string) (string, error) {
	value, err := s.fetch(ctx, headers, "/roles/"+url.PathEscape(roleID), url.Values{})
	if err != nil {
		return "", fmt.Errorf("cannot resolve role: %w", err)
	}
	role := findString(value, "name", "role_name")
	if role == "" {
		return "", fmt.Errorf("role %s has no name", roleID)
	}
	return role, nil
}

func (s *Aggregator) instructorGroups(ctx context.Context, headers http.Header, query url.Values) ([]string, error) {
	memberQuery := cloneValues(query)
	memberQuery.Set("user_id", headers.Get("X-User-ID"))
	memberQuery.Set("role_in_group", "instructor")
	value, err := s.fetch(ctx, headers, "/learning-group-members", memberQuery)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve instructor learning groups: %w", err)
	}
	return findStrings(value, "learning_group_id"), nil
}

func findString(value any, keys ...string) string {
	if object, ok := value.(map[string]any); ok {
		for _, key := range keys {
			if text, ok := object[key].(string); ok {
				return text
			}
		}
		for _, key := range []string{"data", "result"} {
			if nested, ok := object[key]; ok {
				if text := findString(nested, keys...); text != "" {
					return text
				}
			}
		}
	}
	return ""
}

func findStrings(value any, key string) []string {
	result := []string{}
	var walk func(any)
	walk = func(current any) {
		switch typed := current.(type) {
		case []any:
			for _, item := range typed {
				walk(item)
			}
		case map[string]any:
			if id, ok := typed[key].(string); ok {
				result = append(result, id)
			}
			for _, child := range []string{"data", "items", "rows", "results"} {
				if nested, ok := typed[child]; ok {
					walk(nested)
				}
			}
		}
	}
	walk(value)
	return result
}

func summarize(role string, data map[string]any) map[string]any {
	count := func(name string) int { return collectionCount(data[name]) }
	result := map[string]any{"total_users": count("users"), "total_institutions": count("institutions"), "total_learning_groups": count("learning_groups"), "total_learning_group_members": count("learning_group_members"), "total_activities": count("activities"), "total_quiz_sessions": count("quiz_sessions"), "total_attendance_logs": count("attendance"), "quiz_rankings_available": data["quiz_rankings"] != nil}
	if helpers.IsRole(role, "dinas_pendidikan") {
		result["scope"] = "region"
		result["region_level"] = "from_authenticated_scope"
		return result
	}
	switch strings.ToLower(role) {
	case "super_admin", "super-admin", "superadmin":
		result["scope"] = "platform"
	case "dinas_pendidikan", "dinas-pendidikan":
		result["scope"] = "region"
		result["region_level"] = "from_authenticated_scope"
	case "admin_sekolah", "school_admin", "admin-sekolah":
		result["scope"] = "institution"
	case "instructor", "guru", "teacher":
		result["scope"] = "instructor_learning_groups"
	case "pelajar", "student", "siswa":
		result["scope"] = "student"
	default:
		result["scope"] = "authenticated_user"
	}
	return result
}

func collectionCount(value any) int {
	switch typed := value.(type) {
	case []any:
		return len(typed)
	case map[string]any:
		for _, key := range []string{"data", "items", "rows", "results"} {
			if nested, ok := typed[key]; ok {
				return collectionCount(nested)
			}
		}
		for _, key := range []string{"total", "total_count", "count"} {
			if number, ok := typed[key].(float64); ok {
				return int(number)
			}
		}
	}
	return 0
}

func buildRankings(role string, data map[string]any) []any {
	if rows := records(data["quiz_rankings"]); len(rows) > 0 {
		return rows
	}
	// Activity rows are already ordered by the activity endpoint when the
	// client requests it; expose them as a useful fallback for school views.
	if roleIsSchool(role) {
		return records(data["activities"])
	}
	return []any{}
}

func buildAlerts(role string, data map[string]any) []any {
	alerts := []any{}
	for _, row := range records(data["attendance"]) {
		if object, ok := row.(map[string]any); ok && strings.EqualFold(fmt.Sprint(object["status"]), "absent") {
			alerts = append(alerts, map[string]any{"type": "attendance", "severity": "warning", "data": object})
		}
	}
	if roleIsStudent(role) {
		for _, row := range records(data["quiz_sessions"]) {
			if object, ok := row.(map[string]any); ok && strings.EqualFold(fmt.Sprint(object["status"]), "in_progress") {
				alerts = append(alerts, map[string]any{"type": "unfinished_quiz", "severity": "info", "data": object})
			}
		}
	}
	return alerts
}

func roleIsSchool(role string) bool {
	return helpers.IsRole(role, "super_admin") || helpers.IsRole(role, "institution_admin") || helpers.IsRole(role, "dinas_pendidikan")
}

func roleIsStudent(role string) bool { return isStudentRole(role) }

func records(value any) []any {
	switch typed := value.(type) {
	case []any:
		return typed
	case map[string]any:
		for _, key := range []string{"data", "items", "rows", "results", "rankings"} {
			if nested, ok := typed[key]; ok {
				if rows := records(nested); len(rows) > 0 {
					return rows
				}
			}
		}
	}
	return []any{}
}

func (s *Aggregator) fetch(ctx context.Context, headers http.Header, path string, query url.Values) (any, error) {
	query = cloneValues(query)
	if path != "/quiz-sessions/rankings" {
		query.Del("scope")
	}
	// Scope is enforced by JWT-aware upstream services. Do not forward a
	// generic user_id filter to endpoints whose database schema uses another
	// column or derives the user from the authenticated context.
	switch path {
	case "/users":
		if userID := query.Get("user_id"); userID != "" {
			query.Set("u.id", userID)
		}
		query.Del("user_id")
	case "/institutions", "/learning-groups", "/learning-resources":
		query.Del("user_id")
	}
	if institutionID := query.Get("institution_id"); institutionID != "" {
		query.Del("institution_id")
		alias := map[string]string{
			"/users":                  "u.institution_id",
			"/institutions":           "institutions.id",
			"/learning-groups":        "lg.institution_id",
			"/learning-group-members": "i.id",
			"/learning-resources":     "i.id",
			"/activities":             "a.institution_id",
			"/quiz-sessions":          "i.id",
			"/attendance-logs":        "u.institution_id",
		}[path]
		if alias != "" {
			query.Set(alias, institutionID)
		}
	}
	base := helpers.ConfigString("LEARNING_SERVICE_URL", "http://localhost:3003")
	if path == "/users" || path == "/institutions" || strings.HasPrefix(path, "/roles/") {
		base = helpers.ConfigString("CORE_SERVICE_URL", "http://localhost:3002")
	}
	if path == "/activities" {
		base = helpers.ConfigString("ACTIVITY_SERVICE_URL", "http://localhost:3006")
	}
	if base == "" {
		return nil, fmt.Errorf("upstream is not configured")
	}
	suffix := ""
	if q := query.Encode(); q != "" {
		suffix = "?" + q
	}
	upstreamPath := "/api/v1" + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+upstreamPath+suffix, nil)
	if err != nil {
		return nil, err
	}
	for _, n := range []string{"Authorization", "X-User-ID", "X-Institution-ID", "X-Role-ID", "X-Role", "X-Role-Name", "X-Region-Level", "X-Region-Code"} {
		if v := headers.Get(n); v != "" {
			req.Header.Set(n, v)
		}
	}
	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		message := strings.TrimSpace(string(b))
		if len(message) > 500 {
			message = message[:500]
		}
		if message == "" {
			message = resp.Status
		}
		return nil, fmt.Errorf("upstream returned %d: %s", resp.StatusCode, message)
	}
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}
	// All repository controllers wrap payloads as {data, message, status}.
	// Dashboard exposes the actual payload directly to avoid data.data.
	if envelope, ok := v.(map[string]any); ok {
		if payload, exists := envelope["data"]; exists {
			return payload, nil
		}
	}
	return v, nil
}
