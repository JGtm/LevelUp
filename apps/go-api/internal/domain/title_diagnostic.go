// Package domain — title_diagnostic.go : rapport de diagnostic « santé » d'un
// titre (PMT-14 volet A — productisation Phase 1.8 côté admin).
//
// Compare la CONFIG attendue (mappings TOML) et la RÉALITÉ DB (présence des
// bases + lignes des tables cœur) pour un titre donné. Read-only.
package domain

// TitleDiagnostic est le rapport de santé d'un titre exposé à l'admin.
type TitleDiagnostic struct {
	TitleSlug   string             `json:"title_slug"`
	ConfigFiles []ConfigFileStatus `json:"config_files"`
	Databases   []DatabaseStatus   `json:"databases"`
	// Drifts = écarts entre la CONFIG déclarée (capabilities.toml, via la cascade
	// feature-matrix) et la RÉALITÉ DB. Vide = pas de drift (ou capabilities non
	// fournies au service). Cf. TitleDrift.
	Drifts []TitleDrift `json:"drifts,omitempty"`
}

// TitleDrift = un écart déclaré-vs-réalité pour une feature produit.
//
//   - Kind "data"    : la feature est calculée available (capability déclarée)
//     mais sa table de données est absente/vide → le titre PROMET une surface
//     sans données pour l'alimenter.
//   - Kind "feature" : la feature est calculée degraded (capability primaire
//     déclarée mais un enrichissement manque) → surface partielle.
type TitleDrift struct {
	Feature  string `json:"feature"`  // ex: "match_history"
	Kind     string `json:"kind"`     // "data" | "feature"
	Computed string `json:"computed"` // statut feature calculé (available|degraded|unavailable)
	Reason   string `json:"reason"`   // explication lisible
}

// ConfigFileStatus = présence d'un fichier de mapping TOML pour le titre.
type ConfigFileStatus struct {
	Name     string `json:"name"`     // ex: "fields.toml"
	Present  bool   `json:"present"`  // fichier présent sur disque
	Required bool   `json:"required"` // requis pour un titre fonctionnel
}

// DatabaseStatus = présence d'une base DuckDB du titre + état de ses tables cœur.
type DatabaseStatus struct {
	Name   string        `json:"name"`   // ex: "shared_matches_v2.duckdb"
	Exists bool          `json:"exists"` // fichier présent sur disque
	Tables []TableStatus `json:"tables,omitempty"`
	Error  string        `json:"error,omitempty"` // erreur d'ouverture/lecture (best-effort)
}

// TableStatus = présence + nombre de lignes d'une table cœur.
type TableStatus struct {
	Name   string `json:"name"`
	Exists bool   `json:"exists"`
	Rows   int64  `json:"rows"`
}
