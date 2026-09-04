package helpers

import (
	"fmt"
	"strings"
)

type WhereClauseResult struct {
	Clause string
	Params []interface{}
}

func BuildDynamicWhereClause(filters map[string]interface{}) WhereClauseResult {
	where := "where 1=1"
	params := []interface{}{}

	operatorMap := map[string]string{
		"eq":      "=",
		"neq":     "!=",
		"lt":      "<",
		"lte":     "<=",
		"gt":      ">",
		"gte":     ">=",
		"ilike":   "ilike",
		"like":    "like",
		"in":      "in",
		"notin":   "not in",
		"isnull":  "is null",
		"notnull": "is not null",
	}

	for key, value := range filters {
		parts := strings.Split(key, ".")
		field := ""
		op := "eq"

		if len(parts) == 3 {
			field = fmt.Sprintf("%s.%s", parts[0], parts[1])
			op = parts[2]
		} else if len(parts) == 2 {
			if _, ok := operatorMap[parts[1]]; ok {
				field = parts[0]
				op = parts[1]
			} else {
				field = fmt.Sprintf("%s.%s", parts[0], parts[1])
			}
		} else if len(parts) == 1 {
			field = parts[0]
		} else {
			continue
		}

		sqlOp, ok := operatorMap[op]
		if !ok {
			continue
		}

		switch sqlOp {
		case "is null", "is not null":
			where += fmt.Sprintf(" and %s %s", field, sqlOp)
		case "in", "not in":
			if arr, ok := value.([]interface{}); ok && len(arr) > 0 {
				placeholders := make([]string, len(arr))
				for i, v := range arr {
					placeholders[i] = "?"
					params = append(params, v)
				}
				where += fmt.Sprintf(" and %s %s (%s)", field, sqlOp, strings.Join(placeholders, ", "))
			}
		case "like":
			where += fmt.Sprintf(" and %s %s ?", field, sqlOp)
			params = append(params, fmt.Sprintf("%%%v%%", value))
		case "ilike":
			where += fmt.Sprintf(" and lower(%s) %s lower(?)", field, sqlOp)
			params = append(params, fmt.Sprintf("%%%v%%", value))
		default:
			// automatic clock logic for gte, gt, lte, lt
			if op == "gte" || op == "gt" || op == "lte" || op == "lt" {
				if strVal, ok := value.(string); ok && len(strVal) == 10 {
					switch op {
					case "gte", "gt":
						value = strVal + " 00:00:00"
					case "lte", "lt":
						value = strVal + " 23:59:59"
					}
				}
			}

			where += fmt.Sprintf(" and %s %s ?", field, sqlOp)
			params = append(params, value)
		}
	}

	return WhereClauseResult{
		Clause: where,
		Params: params,
	}
}
