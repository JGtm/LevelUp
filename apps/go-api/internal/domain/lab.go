// Package domain — lab.go : contrats JSON du diagnostic d'instance (ex-Lab).
//
// A3.5 (DC-9, 2026-07-10) : le Lab est retiré de l'app — ne reste que le
// diagnostic d'instance (GET /lab/diagnostics), consommé par l'onglet admin
// Données (panneau parité + garde-fous médailles). Les contrats Resources /
// Contracts / Waypoint sont partis avec leurs endpoints.
package domain

import "time"

// LabFileStatus décrit l'état d'un fichier utile au diagnostic.
type LabFileStatus struct {
	Path       string     `json:"path"`
	Exists     bool       `json:"exists"`
	SizeBytes  int64      `json:"size_bytes"`
	ModifiedAt *time.Time `json:"modified_at,omitempty"`
}

// LabGuardResult est la projection sérialisable d'un garde-fou metadata.
type LabGuardResult struct {
	Passed  bool     `json:"passed"`
	Reason  string   `json:"reason"`
	Details []string `json:"details,omitempty"`
}

// LabMedalGuardsReport regroupe les garde-fous appliqués aux médailles brutes.
type LabMedalGuardsReport struct {
	EntryCount     int            `json:"entry_count"`
	Cardinality    LabGuardResult `json:"cardinality"`
	RequiredFields LabGuardResult `json:"required_fields"`
	Images         LabGuardResult `json:"images"`
	Overall        LabGuardResult `json:"overall"`
}

// LabParitySummary résume le dernier rapport de parité stocké.
type LabParitySummary struct {
	Total   int `json:"total"`
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

// LabParityResult décrit un endpoint du rapport de parité.
type LabParityResult struct {
	Name       string                   `json:"name"`
	Status     string                   `json:"status"`
	HTTPStatus *int                     `json:"http_status,omitempty"`
	Mode       string                   `json:"mode,omitempty"`
	Reason     string                   `json:"reason,omitempty"`
	Error      string                   `json:"error,omitempty"`
	Diffs      []map[string]interface{} `json:"diffs,omitempty"`
}

// LabParityReport représente le JSON produit par parity_check.py.
type LabParityReport struct {
	GeneratedAt string            `json:"generated_at"`
	GoURL       string            `json:"go_url"`
	Player      string            `json:"player"`
	Summary     LabParitySummary  `json:"summary"`
	Results     []LabParityResult `json:"results"`
}

// LabDiagnosticsResponse alimente le panneau Diagnostics (onglet Données).
type LabDiagnosticsResponse struct {
	TitleSlug        string                `json:"title_slug"`
	ParityReportFile LabFileStatus         `json:"parity_report_file"`
	ParityReport     *LabParityReport      `json:"parity_report,omitempty"`
	MedalGuards      *LabMedalGuardsReport `json:"medal_guards,omitempty"`
}
