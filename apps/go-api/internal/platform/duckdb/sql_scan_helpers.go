package duckdb

import "database/sql"

// sql_scan_helpers.go — helpers SQL génériques partagés par les repos duckdb-root
// ET le sous-package duckdb/prestige (extraction K3a). RowScanner + NullableStr
// n'ont rien de spécifique au domaine Prestige : ils vivaient historiquement dans
// prestige_player_helpers.go et ont été remontés ici pour que les deux packages y
// accèdent sans cycle (prestige importe duckdb-root, jamais l'inverse).

// RowScanner abstrait *sql.Row et *sql.Rows pour partager la logique de scan entre
// les fonctions scanX (Get sur une ligne unique vs List sur *sql.Rows).
type RowScanner interface {
	Scan(dest ...any) error
}

// NullableStr convertit une string vide en sql.NullString (NULL en base) et
// laisse toute autre valeur telle quelle. Utilisé pour les colonnes optionnelles.
func NullableStr(s string) any {
	if s == "" {
		return sql.NullString{}
	}
	return s
}
