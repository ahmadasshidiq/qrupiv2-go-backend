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
	"sort"
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

func (s *Aggregator) GetForRole(ctx context.Context, headers http.Header, query url.Values, expectedRole string) (*Response, error) {
	role := helpers.RoleKeyFromID(headers.Get("X-Role-ID"))
	if expectedRole != "" && role == "" {
		return nil, fmt.Errorf("role UUID is not configured for dashboard endpoint")
	}
	if role == "" && headers.Get("X-Role-ID") != "" {
		resolved, err := s.resolveRole(ctx, headers, headers.Get("X-Role-ID"))
		if err != nil {
			return nil, err
		}
		role = resolved
	}
	if expectedRole != "" && role != expectedRole {
		return nil, fmt.Errorf("dashboard endpoint requires role %q", expectedRole)
	}
	effectiveQuery := cloneValues(query)
	if isStudentRole(role) {
		effectiveQuery.Set("user_id", headers.Get("X-User-ID"))
	}
	if institutionID := headers.Get("X-Institution-ID"); institutionID != "" {
		effectiveQuery.Set("institution_id", institutionID)
	}
	if effectiveQuery.Get("limit") == "" {
		if helpers.IsRole(role, "super_admin") {
			effectiveQuery.Set("limit", "999")
		} else {
			effectiveQuery.Set("limit", "100")
		}
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
	if roleIsSchool(role) && effectiveQuery.Get("scope") == "" {
		effectiveQuery.Set("scope", "school")
	}
	h := sha256.Sum256([]byte(role + "|" + headers.Get("X-User-ID") + "|" + headers.Get("X-Institution-ID") + "|" + effectiveQuery.Encode()))
	key := "dashboard:v13:" + hex.EncodeToString(h[:])
	if s.Cache != nil {
		if b, err := s.Cache.Get(ctx, key).Bytes(); err == nil {
			var r Response
			if json.Unmarshal(b, &r) == nil {
				r.Cached = true
				return &r, nil
			}
		}
	}
	paths := dashboardPaths(role)
	data := map[string]any{}
	hasUpstreamErrors := false
	for name, path := range paths {
		requestQuery := cloneValues(effectiveQuery)
		if path == "/activities/chart" {
			if name == "positive_activity_chart" {
				requestQuery.Set("type", "positive")
			} else if name == "violation_activity_chart" {
				requestQuery.Set("type", "violation")
			}
			requestQuery.Set("top_limit", "5")
		}
		if path == "/quiz-sessions/rankings" && requestQuery.Get("scope") == "" {
			data[name] = map[string]any{"available": false, "reason": "ranking scope is not requested"}
			continue
		}
		if path == "/quiz-sessions/rankings" {
			requestQuery.Set("limit", "5")
		}
		v, err := s.fetch(ctx, headers, path, requestQuery)
		if err != nil {
			hasUpstreamErrors = true
			data[name] = map[string]any{"available": false, "error": err.Error()}
		} else {
			data[name] = v
		}
	}
	r := &Response{Generated: time.Now().UTC().Format(time.RFC3339), Summary: summarize(role, data), Rankings: buildRankings(role, data), Data: dashboardData(role, data)}
	if b, err := json.Marshal(r); err == nil && s.Cache != nil && !hasUpstreamErrors {
		_ = s.Cache.Set(ctx, key, b, s.TTL).Err()
	}
	return r, nil
}

func dashboardData(role string, data map[string]any) map[string]any {
	result := make(map[string]any)
	for _, key := range []string{"positive_activity_chart", "violation_activity_chart"} {
		if value, ok := data[key]; ok {
			result[key] = value
		}
	}
	if groups := dashboardGroups(data["learning_groups"]); len(groups) > 0 {
		result["active_learning_groups"] = groups
	}
	if teachers := dashboardTeachers(data["positive_activity_chart"]); len(teachers) > 0 {
		result["top_teachers"] = teachers
	}
	if locations := institutionLocations(data["institutions"], data["users"]); len(locations) > 0 {
		result["institution_map"] = locations
	}
	if isInstructorRole(role) {
		result["learning_groups"] = instructorGroupSummary(data["learning_groups"], data["learning_group_members"], data["attendance"], data["quiz_sessions"])
		result["students_need_attention"] = studentsNeedAttention(data["attendance"], data["quiz_sessions"])
	}
	return result
}

func instructorGroupSummary(groups, members, attendance, quizzes any) []any {
	memberCount := map[string]int{}
	for _, row := range records(members) {
		if object, ok := row.(map[string]any); ok {
			if id := firstString(object, "learning_group_id", "group_id"); id != "" {
				memberCount[id]++
			}
		}
	}
	result := []any{}
	for _, row := range records(groups) {
		object, ok := row.(map[string]any)
		if !ok {
			continue
		}
		item := map[string]any{}
		for _, key := range []string{"id", "name", "learning_group_name", "total_students", "student_count"} {
			if value, exists := object[key]; exists {
				item[key] = value
			}
		}
		id := firstString(object, "id", "learning_group_id")
		if _, exists := item["student_count"]; !exists && memberCount[id] > 0 {
			item["student_count"] = memberCount[id]
		}
		if id != "" {
			item["id"] = id
		}
		if len(item) > 0 {
			result = append(result, item)
		}
	}
	return result
}

func studentsNeedAttention(attendance, quizzes any) []any {
	result := []any{}
	seen := map[string]bool{}
	for _, row := range append(records(attendance), records(quizzes)...) {
		object, ok := row.(map[string]any)
		if !ok {
			continue
		}
		status := strings.ToLower(firstString(object, "status", "attendance_status", "result"))
		score := numberValue(object, "score", "final_score", "average_score")
		if status != "absent" && status != "alpha" && status != "sick" && status != "permit" && (score == nil || *score >= 60) {
			continue
		}
		id := firstString(object, "user_id", "student_id", "id")
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		item := map[string]any{"user_id": id}
		for _, key := range []string{"user_name", "student_name", "name", "status", "attendance_status", "score", "final_score"} {
			if value, exists := object[key]; exists {
				item[key] = value
			}
		}
		result = append(result, item)
		if len(result) == 10 {
			break
		}
	}
	return result
}

func firstString(object map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := object[key].(string); ok && value != "" {
			return value
		}
	}
	return ""
}
func numberValue(object map[string]any, key ...string) *float64 {
	for _, k := range key {
		if value, ok := object[k].(float64); ok {
			return &value
		}
		if value, ok := object[k].(int); ok {
			v := float64(value)
			return &v
		}
	}
	return nil
}

func institutionLocations(value, users any) []any {
	studentCounts := map[string]int{}
	for _, row := range records(users) {
		if object, ok := row.(map[string]any); ok {
			if id, ok := object["institution_id"].(string); ok && strings.EqualFold(fmt.Sprint(object["type"]), "student") {
				studentCounts[id]++
			}
		}
	}
	result := []any{}
	for _, row := range records(value) {
		object, ok := row.(map[string]any)
		if !ok {
			continue
		}
		lat, latOK := object["latitude"]
		lng, lngOK := object["longitude"]
		if !latOK || !lngOK {
			continue
		}
		item := map[string]any{"id": object["id"], "name": object["name"], "latitude": lat, "longitude": lng}
		if id, ok := object["id"].(string); ok {
			item["student_count"] = studentCounts[id]
		}
		for _, key := range []string{"province_name", "regency_name", "status"} {
			if v, ok := object[key]; ok {
				item[key] = v
			}
		}
		result = append(result, item)
	}
	return result
}

func dashboardGroups(value any) []any {
	rows := records(value)
	result := make([]any, 0, len(rows))
	for _, row := range rows {
		object, ok := row.(map[string]any)
		if !ok {
			continue
		}
		item := map[string]any{}
		for _, key := range []string{"id", "name", "learning_group_name", "instructor_name", "teacher_name", "total_students", "student_count"} {
			if field, exists := object[key]; exists {
				item[key] = field
			}
		}
		if len(item) > 0 {
			result = append(result, item)
		}
	}
	return result
}

func dashboardTeachers(value any) []any {
	object, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return records(object["top_teachers"])
}

func dashboardPaths(role string) map[string]string {
	switch {
	case helpers.IsRole(role, "student"):
		return map[string]string{"learning_groups": "/learning-groups", "activities": "/activities", "quiz_sessions": "/quiz-sessions", "attendance": "/attendance-logs", "quiz_rankings": "/quiz-sessions/rankings"}
	case helpers.IsRole(role, "instructor"):
		return map[string]string{"learning_groups": "/learning-groups", "learning_group_members": "/learning-group-members", "activities": "/activities", "quiz_sessions": "/quiz-sessions", "attendance": "/attendance-logs", "quiz_rankings": "/quiz-sessions/rankings"}
	case helpers.IsRole(role, "institution_admin"):
		return map[string]string{"users": "/users", "learning_groups": "/learning-groups", "learning_group_members": "/learning-group-members", "activities": "/activities", "positive_activity_chart": "/activities/chart", "violation_activity_chart": "/activities/chart", "quiz_sessions": "/quiz-sessions", "attendance": "/attendance-logs"}
	case helpers.IsRole(role, "dinas_pendidikan"):
		return map[string]string{"institutions": "/institutions", "users": "/users", "learning_groups": "/learning-groups", "activities": "/activities", "attendance": "/attendance-logs"}
	default:
		return map[string]string{"users": "/users", "institutions": "/institutions", "learning_groups": "/learning-groups", "learning_group_members": "/learning-group-members", "activities": "/activities", "quiz_sessions": "/quiz-sessions", "attendance": "/attendance-logs"}
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
	groupQuery := cloneValues(query)
	groupQuery.Del("learning_group_id")
	groupQuery.Del("learning_group_id.in")
	value, err := s.fetch(ctx, headers, "/learning-groups", groupQuery)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve instructor learning groups: %w", err)
	}
	ids := findStrings(value, "id")
	result := make([]string, 0, len(ids))
	seen := map[string]bool{}
	for _, id := range ids {
		if id != "" && !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}
	return result, nil
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
	totalActivities := count("activities")
	positiveActivities := chartSummaryCount(data["positive_activity_chart"], "positive_activities")
	violationActivities := chartSummaryCount(data["violation_activity_chart"], "violation_activities")
	if _, positiveChartAvailable := data["positive_activity_chart"]; positiveChartAvailable {
		totalActivities = positiveActivities + violationActivities
	}
	result := map[string]any{"total_users": count("users"), "total_institutions": count("institutions"), "total_learning_groups": count("learning_groups"), "total_learning_group_members": count("learning_group_members"), "total_activities": totalActivities, "positive_activities": positiveActivities, "violation_activities": violationActivities, "total_quiz_sessions": count("quiz_sessions"), "total_attendance_logs": count("attendance"), "quiz_rankings_available": data["quiz_rankings"] != nil}
	if isInstructorRole(role) {
		result["total_students"] = count("learning_group_members")
		result["scope"] = "instructor_learning_groups"
	}
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

func chartSummaryCount(value any, key string) int {
	object, ok := value.(map[string]any)
	if !ok {
		return 0
	}
	summary, ok := object["summary"].(map[string]any)
	if !ok {
		return 0
	}
	number, ok := summary[key].(float64)
	if !ok {
		return 0
	}
	return int(number)
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
	if helpers.IsRole(role, "super_admin") {
		return institutionRankings(data)
	}
	if rows := records(data["quiz_rankings"]); len(rows) > 0 {
		return rows
	}
	return []any{}
}

func institutionRankings(data map[string]any) []any {
	users, groups := map[string]map[string]any{}, map[string]map[string]any{}
	activities := map[string]map[string]any{}
	userInstitutions := map[string]string{}
	for _, row := range records(data["users"]) {
		if o, ok := row.(map[string]any); ok {
			if userID, ok := o["id"].(string); ok {
				if institutionID, ok := o["institution_id"].(string); ok {
					userInstitutions[userID] = institutionID
				}
			}
			if id, ok := o["institution_id"].(string); ok && strings.EqualFold(fmt.Sprint(o["type"]), "student") {
				if _, exists := users[id]; !exists {
					users[id] = map[string]any{"institution_id": id, "institution_name": o["institution_name"]}
				}
				users[id]["student_count"] = intValue(users[id]["student_count"]) + 1
			}
		}
	}
	for _, row := range records(data["learning_groups"]) {
		if o, ok := row.(map[string]any); ok {
			if id, ok := o["institution_id"].(string); ok {
				if _, exists := groups[id]; !exists {
					groups[id] = map[string]any{"institution_id": id, "institution_name": o["institution_name"]}
				}
				groups[id]["group_count"] = intValue(groups[id]["group_count"]) + 1
			}
		}
	}
	for _, row := range records(data["activities"]) {
		if o, ok := row.(map[string]any); ok {
			id, ok := o["institution_id"].(string)
			if !ok || id == "" {
				id, ok = userInstitutions[fmt.Sprint(o["user_id"])]
			}
			if ok && id != "" {
				if _, exists := activities[id]; !exists {
					activities[id] = map[string]any{"institution_id": id, "institution_name": o["institution_name"]}
				}
				activities[id]["activity_count"] = intValue(activities[id]["activity_count"]) + 1
			}
		}
	}
	return []any{
		map[string]any{"key": "top_institutions_by_students", "items": sortRanking(users, "student_count")},
		map[string]any{"key": "top_institutions_by_activity", "items": sortRanking(activities, "activity_count")},
		map[string]any{"key": "top_institutions_by_groups", "items": sortRanking(groups, "group_count")},
	}
}

func intValue(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	}
	return 0
}
func sortRanking(values map[string]map[string]any, field string) []any {
	result := []any{}
	for _, value := range values {
		result = append(result, value)
	}
	sort.SliceStable(result, func(i, j int) bool {
		return intValue(result[i].(map[string]any)[field]) > intValue(result[j].(map[string]any)[field])
	})
	return result
}

func roleIsSchool(role string) bool {
	return helpers.IsRole(role, "super_admin") || helpers.IsRole(role, "institution_admin") || helpers.IsRole(role, "dinas_pendidikan")
}

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
	if path != "/activities/chart" && path != "/quiz-sessions/rankings" {
		applyListFilters(path, query)
	}
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
	if path == "/activities" || path == "/activities/chart" {
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
		return nil, fmt.Errorf("request to %s failed: %w", req.URL.String(), err)
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
	if envelope, ok := v.(map[string]any); ok {
		if payload, exists := envelope["data"]; exists {
			return payload, nil
		}
	}
	return v, nil
}

func applyListFilters(path string, query url.Values) {
	dateField := map[string]string{
		"/activities":      "a.occurred_at",
		"/attendance-logs": "al.created_at",
	}[path]
	if dateField != "" {
		if start := query.Get("start_date"); start != "" {
			query.Set(dateField+".gte", start)
		}
		if end := query.Get("end_date"); end != "" {
			if value, err := time.Parse("2006-01-02", end); err == nil {
				query.Set(dateField+".lt", value.AddDate(0, 0, 1).Format("2006-01-02"))
			}
		}
	}
	for _, key := range []string{"period", "start_date", "end_date", "category_id", "top_limit"} {
		query.Del(key)
	}
	if groupID := query.Get("learning_group_id"); groupID != "" {
		field := map[string]string{"/activities": "a.learning_group_id", "/attendance-logs": "al.learning_group_id"}[path]
		if field != "" {
			query.Set(field, groupID)
		}
	}
	if path != "/activities" && path != "/quiz-sessions" && path != "/attendance-logs" {
		query.Del("learning_group_id")
	}
}
