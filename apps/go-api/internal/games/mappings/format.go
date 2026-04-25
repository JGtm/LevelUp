package mappings

import (
	"fmt"
	"math"
	"strconv"
)

// Format énumère les rendus supportés pour un FieldMapping.
type Format string

const (
	FormatInteger     Format = "integer"
	FormatSignedInt   Format = "signed_int"
	FormatPercent1    Format = "percent_1"
	FormatPercent2    Format = "percent_2"
	FormatKDR2        Format = "kdr_2"
	FormatDurationHMS Format = "duration_hms"
	FormatSeconds     Format = "seconds"
	FormatString      Format = "string"
	FormatBoolean     Format = "boolean"
	FormatDateTime    Format = "datetime"
	FormatEnum        Format = "enum"
)

// IsKnownFormat retourne vrai si le format est dans l'enum.
func IsKnownFormat(f Format) bool {
	switch f {
	case FormatInteger, FormatSignedInt,
		FormatPercent1, FormatPercent2, FormatKDR2,
		FormatDurationHMS, FormatSeconds,
		FormatString, FormatBoolean, FormatDateTime, FormatEnum:
		return true
	}
	return false
}

// FormatValue applique le format à v et retourne la chaîne de présentation.
//
// Comportement :
//   - v nil          → "" (champ non disponible, à distinguer côté UI si besoin)
//   - format inconnu → erreur explicite (devrait être impossible si validation OK)
//   - NaN/Inf        → "" pour les formats numériques (donnée corrompue)
//
// FormatValue ne fait pas de conversion d'unité ; cette responsabilité revient
// à l'adapter qui applique storage_unit -> display_unit avant d'appeler ici.
func FormatValue(v any, f Format) (string, error) {
	if v == nil {
		return "", nil
	}
	if !IsKnownFormat(f) {
		return "", fmt.Errorf("format inconnu: %q", f)
	}

	switch f {
	case FormatInteger:
		return formatInteger(v, false)
	case FormatSignedInt:
		return formatInteger(v, true)
	case FormatPercent1:
		return formatPercent(v, 1)
	case FormatPercent2:
		return formatPercent(v, 2)
	case FormatKDR2:
		return formatFloat(v, 2)
	case FormatSeconds:
		return formatSeconds(v)
	case FormatDurationHMS:
		return formatDurationHMS(v)
	case FormatString:
		return formatString(v), nil
	case FormatBoolean:
		return formatBoolean(v), nil
	case FormatDateTime:
		return formatString(v), nil
	case FormatEnum:
		return formatString(v), nil
	}
	return "", fmt.Errorf("format non implémenté: %q", f)
}

func toFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case uint:
		return float64(x), true
	case uint32:
		return float64(x), true
	case uint64:
		return float64(x), true
	}
	return 0, false
}

func formatInteger(v any, signed bool) (string, error) {
	f, ok := toFloat(v)
	if !ok {
		return "", fmt.Errorf("integer attendu, reçu %T", v)
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return "", nil
	}
	n := int64(f)
	if signed && n >= 0 {
		return "+" + strconv.FormatInt(n, 10), nil
	}
	return strconv.FormatInt(n, 10), nil
}

func formatPercent(v any, decimals int) (string, error) {
	f, ok := toFloat(v)
	if !ok {
		return "", fmt.Errorf("percent attendu, reçu %T", v)
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return "", nil
	}
	return strconv.FormatFloat(f, 'f', decimals, 64) + "%", nil
}

func formatFloat(v any, decimals int) (string, error) {
	f, ok := toFloat(v)
	if !ok {
		return "", fmt.Errorf("float attendu, reçu %T", v)
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return "", nil
	}
	return strconv.FormatFloat(f, 'f', decimals, 64), nil
}

func formatSeconds(v any) (string, error) {
	f, ok := toFloat(v)
	if !ok {
		return "", fmt.Errorf("seconds attendu, reçu %T", v)
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return "", nil
	}
	return strconv.FormatFloat(f, 'f', 0, 64) + "s", nil
}

func formatDurationHMS(v any) (string, error) {
	f, ok := toFloat(v)
	if !ok {
		return "", fmt.Errorf("seconds attendu pour duration_hms, reçu %T", v)
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return "", nil
	}
	total := int64(f)
	if total < 0 {
		total = -total
	}
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	if h > 0 {
		return fmt.Sprintf("%dh%02dm%02ds", h, m, s), nil
	}
	if m > 0 {
		return fmt.Sprintf("%dm%02ds", m, s), nil
	}
	return fmt.Sprintf("%ds", s), nil
}

func formatString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case fmt.Stringer:
		return x.String()
	}
	return fmt.Sprintf("%v", v)
}

func formatBoolean(v any) string {
	switch x := v.(type) {
	case bool:
		if x {
			return "true"
		}
		return "false"
	}
	return ""
}
