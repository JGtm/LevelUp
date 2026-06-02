package lab

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"levelup/go-api/internal/domain"
	metadata_guard "levelup/go-api/internal/metadata"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

func loadParityReport(file domain.LabFileStatus) (*domain.LabParityReport, error) {
	if !file.Exists {
		return nil, nil
	}
	data, err := os.ReadFile(file.Path)
	if err != nil {
		return nil, fmt.Errorf("read parity report: %w", err)
	}
	var report domain.LabParityReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("parse parity report: %w", err)
	}
	return &report, nil
}

func loadOpenAPIRoutes(path string) (map[string][]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read openapi %s: %w", path, err)
	}
	var doc struct {
		Paths map[string]map[string]interface{} `yaml:"paths"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse openapi %s: %w", path, err)
	}
	routes := make(map[string][]string, len(doc.Paths))
	for path, methods := range doc.Paths {
		var declared []string
		for method := range methods {
			if openAPIMethods[strings.ToLower(method)] {
				declared = append(declared, strings.ToLower(method))
			}
		}
		if len(declared) == 0 {
			continue
		}
		sort.Strings(declared)
		routes[path] = declared
	}
	return routes, nil
}

func compareOpenAPIRoutes(
	fastapiRoutes map[string][]string,
	goRoutes map[string][]string,
) ([]domain.LabRouteMethods, []domain.LabRouteMethods, []domain.LabMethodMismatch) {
	faNorm := normalizeRoutes(fastapiRoutes)
	goNorm := normalizeRoutes(goRoutes)

	missing := make([]domain.LabRouteMethods, 0, len(faNorm))
	extra := make([]domain.LabRouteMethods, 0, len(goNorm))
	var mismatches []domain.LabMethodMismatch

	for normalized, route := range faNorm {
		if goRoute, ok := goNorm[normalized]; ok {
			if !sameMethods(route.Methods, goRoute.Methods) {
				mismatches = append(mismatches, domain.LabMethodMismatch{
					FastAPIPath:    route.Path,
					GoPath:         goRoute.Path,
					FastAPIMethods: route.Methods,
					GoMethods:      goRoute.Methods,
					MissingMethods: diffMethods(route.Methods, goRoute.Methods),
					ExtraMethods:   diffMethods(goRoute.Methods, route.Methods),
				})
			}
			continue
		}
		missing = append(missing, domain.LabRouteMethods{Path: route.Path, Methods: route.Methods})
	}
	for normalized, route := range goNorm {
		if _, ok := faNorm[normalized]; ok {
			continue
		}
		extra = append(extra, domain.LabRouteMethods{Path: route.Path, Methods: route.Methods})
	}

	sortRoutes(missing)
	sortRoutes(extra)
	sortMethodMismatches(mismatches)
	return missing, extra, mismatches
}

func normalizeRoutes(routes map[string][]string) map[string]domain.LabRouteMethods {
	normalized := make(map[string]domain.LabRouteMethods, len(routes))
	for path, methods := range routes {
		normalized[pathParamRE.ReplaceAllString(path, "{*}")] = domain.LabRouteMethods{
			Path:    path,
			Methods: methods,
		}
	}
	return normalized
}

func sortRoutes(routes []domain.LabRouteMethods) {
	sort.Slice(routes, func(i, j int) bool { return routes[i].Path < routes[j].Path })
}

func sortMethodMismatches(items []domain.LabMethodMismatch) {
	sort.Slice(items, func(i, j int) bool { return items[i].FastAPIPath < items[j].FastAPIPath })
}

func scanSeasonCalendar(rows *sql.Rows) (domain.SeasonCalendar, error) {
	var season domain.SeasonCalendar
	var endDate sql.NullTime
	if err := rows.Scan(
		&season.TitleID, &season.SeasonID, &season.Version, &season.Name,
		&season.StartDate, &endDate, &season.FetchedAt,
		&season.ContentHash, &season.ETag, &season.SourceURL,
	); err != nil {
		return domain.SeasonCalendar{}, fmt.Errorf("scan season calendar: %w", err)
	}
	if endDate.Valid {
		season.EndDate = &endDate.Time
	}
	return season, nil
}

func orEmptyCSR(items []domain.CSRSeasonCalendar) []domain.CSRSeasonCalendar {
	if items == nil {
		return []domain.CSRSeasonCalendar{}
	}
	return items
}

func likeQuery(search string) string {
	trimmed := strings.TrimSpace(strings.ToLower(search))
	if trimmed == "" {
		return ""
	}
	return "%" + trimmed + "%"
}

func isMissingRelationError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "catalog error") && strings.Contains(msg, "does not exist")
}

func toGuardResult(result metadata_guard.GuardResult) domain.LabGuardResult {
	return domain.LabGuardResult{Passed: result.Passed, Reason: result.Reason, Details: result.Details}
}

func sameMethods(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func diffMethods(left, right []string) []string {
	lookup := make(map[string]bool, len(right))
	for _, method := range right {
		lookup[method] = true
	}
	var diff []string
	for _, method := range left {
		if !lookup[method] {
			diff = append(diff, method)
		}
	}
	return diff
}

func labContractStatus(missing []domain.LabRouteMethods, mismatches []domain.LabMethodMismatch) string {
	if len(missing) == 0 && len(mismatches) == 0 {
		return "OK"
	}
	return "DIVERGENCES"
}
