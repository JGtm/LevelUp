// Package mappings — loader TOML + types pour la couche sémantique des titres.
//
// Le loader transforme les fichiers config/titles/{slug}/mappings/*.toml en
// FieldMappingSet immutables consommés par le TitleSemanticAdapter de chaque
// titre. Une erreur de validation refuse le boot du titre concerné mais ne
// bloque pas les autres titres.
package mappings

// Unit énumère les unités physiques/logiques supportées par le canonique.
//
// L'adapter applique automatiquement la conversion StorageUnit -> DisplayUnit
// avant de remplir un champ canonique services. Toute unité absente de cette
// liste est rejetée par le loader.
type Unit string

const (
	UnitCount        Unit = "count"
	UnitRatio        Unit = "ratio"   // 0..1
	UnitPercent      Unit = "percent" // 0..100
	UnitSeconds      Unit = "seconds"
	UnitMilliseconds Unit = "milliseconds"
	UnitNoneUnit     Unit = "" // pour types non numériques (string, datetime, bool, enum)
)

// IsKnownUnit retourne vrai si l'unité est dans l'enum.
func IsKnownUnit(u Unit) bool {
	switch u {
	case UnitCount, UnitRatio, UnitPercent, UnitSeconds, UnitMilliseconds, UnitNoneUnit:
		return true
	}
	return false
}

// ConvertValue convertit une valeur numérique de from -> to. Retourne ok=false
// si la conversion n'est pas définie. Identité retournée si from == to.
//
// Conversions définies :
//   - ratio       <-> percent
//   - milliseconds -> seconds  (et inverse)
func ConvertValue(value float64, from, to Unit) (out float64, ok bool) {
	if from == to {
		return value, true
	}
	switch {
	case from == UnitRatio && to == UnitPercent:
		return value * 100.0, true
	case from == UnitPercent && to == UnitRatio:
		return value / 100.0, true
	case from == UnitMilliseconds && to == UnitSeconds:
		return value / 1000.0, true
	case from == UnitSeconds && to == UnitMilliseconds:
		return value * 1000.0, true
	}
	return 0, false
}
