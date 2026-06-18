// Package domain — lab.go : contrats JSON du Lab interne d'instance.
package domain

import "time"

// LabResourcesQuery représente les filtres de l'explorateur interne.
type LabResourcesQuery struct {
	SnapshotKey string `json:"snapshot_key,omitempty"`
	AssetID     string `json:"asset_id,omitempty"`
	AssetSearch string `json:"asset_search,omitempty"`
	MedalID     int64  `json:"medal_id,omitempty"`
	MedalSearch string `json:"medal_search,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}

// LabFileStatus décrit l'état d'un fichier utile au Lab.
type LabFileStatus struct {
	Path       string     `json:"path"`
	Exists     bool       `json:"exists"`
	SizeBytes  int64      `json:"size_bytes"`
	ModifiedAt *time.Time `json:"modified_at,omitempty"`
}

// LabSnapshotSummary résume un snapshot Waypoint archivé.
type LabSnapshotSummary struct {
	ResourceKey string    `json:"resource_key"`
	Version     string    `json:"version"`
	FetchedAt   time.Time `json:"fetched_at"`
	ContentHash string    `json:"content_hash"`
	ETag        string    `json:"etag,omitempty"`
	SourceURL   string    `json:"source_url,omitempty"`
	PayloadSize int       `json:"payload_size"`
}

// LabSnapshotDetail expose le payload complet d'un snapshot sélectionné.
type LabSnapshotDetail struct {
	ResourceKey string    `json:"resource_key"`
	Version     string    `json:"version"`
	FetchedAt   time.Time `json:"fetched_at"`
	ContentHash string    `json:"content_hash"`
	ETag        string    `json:"etag,omitempty"`
	SourceURL   string    `json:"source_url,omitempty"`
	Payload     string    `json:"payload"`
}

// LabAssetSummary résume un asset Waypoint brut.
type LabAssetSummary struct {
	AssetID   string    `json:"asset_id"`
	AssetType string    `json:"asset_type"`
	VersionID string    `json:"version_id"`
	Name      string    `json:"name"`
	FetchedAt time.Time `json:"fetched_at"`
}

// LabAssetDetail expose le JSON brut d'un asset Waypoint.
type LabAssetDetail struct {
	AssetID     string    `json:"asset_id"`
	AssetType   string    `json:"asset_type"`
	VersionID   string    `json:"version_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	FetchedAt   time.Time `json:"fetched_at"`
	ContentHash string    `json:"content_hash"`
	RawJSON     string    `json:"raw_json"`
}

// LabAssetExplorer contient les résultats de recherche d'assets.
type LabAssetExplorer struct {
	Total    int               `json:"total"`
	Search   string            `json:"search,omitempty"`
	Items    []LabAssetSummary `json:"items"`
	Selected *LabAssetDetail   `json:"selected,omitempty"`
}

// LabMedalSummary résume une médaille Waypoint brute.
type LabMedalSummary struct {
	MedalID       int64     `json:"medal_id"`
	NameID        string    `json:"name_id"`
	DescriptionID string    `json:"description_id"`
	MedalType     string    `json:"medal_type"`
	Difficulty    string    `json:"difficulty"`
	SpriteIndex   int       `json:"sprite_index"`
	FetchedAt     time.Time `json:"fetched_at"`
}

// LabMedalDetail expose le JSON brut d'une médaille Waypoint.
type LabMedalDetail struct {
	MedalID       int64     `json:"medal_id"`
	NameID        string    `json:"name_id"`
	DescriptionID string    `json:"description_id"`
	MedalType     string    `json:"medal_type"`
	Difficulty    string    `json:"difficulty"`
	SpriteIndex   int       `json:"sprite_index"`
	PersonalScore int       `json:"personal_score"`
	FetchedAt     time.Time `json:"fetched_at"`
	ContentHash   string    `json:"content_hash"`
	RawJSON       string    `json:"raw_json"`
}

// LabMedalExplorer contient les résultats de recherche de médailles.
type LabMedalExplorer struct {
	Total    int               `json:"total"`
	Search   string            `json:"search,omitempty"`
	Items    []LabMedalSummary `json:"items"`
	Selected *LabMedalDetail   `json:"selected,omitempty"`
}

// LabResourcesResponse alimente l'onglet Explorateur interne.
type LabResourcesResponse struct {
	TitleSlug        string               `json:"title_slug"`
	MetadataDBPath   string               `json:"metadata_db_path"`
	CurrentSeason    *SeasonCalendar      `json:"current_season,omitempty"`
	Seasons          []SeasonCalendar     `json:"seasons"`
	CSRSeasons       []CSRSeasonCalendar  `json:"csr_seasons"`
	Snapshots        []LabSnapshotSummary `json:"snapshots"`
	SelectedSnapshot *LabSnapshotDetail   `json:"selected_snapshot,omitempty"`
	Assets           LabAssetExplorer     `json:"assets"`
	Medals           LabMedalExplorer     `json:"medals"`
}

// LabRouteMethods représente un path et ses méthodes HTTP.
type LabRouteMethods struct {
	Path    string   `json:"path"`
	Methods []string `json:"methods"`
}

// LabMethodMismatch décrit une divergence de méthodes entre FastAPI et Go.
type LabMethodMismatch struct {
	FastAPIPath    string   `json:"fastapi_path"`
	GoPath         string   `json:"go_path"`
	FastAPIMethods []string `json:"fastapi_methods"`
	GoMethods      []string `json:"go_methods"`
	MissingMethods []string `json:"missing_methods"`
	ExtraMethods   []string `json:"extra_methods"`
}

// LabOpenAPISummary résume l'état du diff OpenAPI.
type LabOpenAPISummary struct {
	FastAPIRouteCount int    `json:"fastapi_route_count"`
	GoRouteCount      int    `json:"go_route_count"`
	MissingInGo       int    `json:"missing_in_go"`
	ExtraInGo         int    `json:"extra_in_go"`
	MethodMismatches  int    `json:"method_mismatches"`
	Status            string `json:"status"`
}

// LabContractsResponse alimente l'onglet Contrats API.
type LabContractsResponse struct {
	GoOpenAPI        LabFileStatus       `json:"go_openapi"`
	FastAPIReference LabFileStatus       `json:"fastapi_reference"`
	Summary          LabOpenAPISummary   `json:"summary"`
	MissingInGo      []LabRouteMethods   `json:"missing_in_go"`
	ExtraInGo        []LabRouteMethods   `json:"extra_in_go"`
	MethodMismatches []LabMethodMismatch `json:"method_mismatches"`
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

// LabDiagnosticsResponse alimente l'onglet Diagnostics.
type LabDiagnosticsResponse struct {
	TitleSlug        string                `json:"title_slug"`
	ParityReportFile LabFileStatus         `json:"parity_report_file"`
	ParityReport     *LabParityReport      `json:"parity_report,omitempty"`
	MedalGuards      *LabMedalGuardsReport `json:"medal_guards,omitempty"`
}

// LabWaypointQuery paramètre une exploration live de l'API Discovery UGC depuis
// l'Atelier (segment = AssetType : map | playlist | pair | game_variant).
type LabWaypointQuery struct {
	Segment   string `json:"segment"`
	AssetID   string `json:"asset_id"`
	VersionID string `json:"version_id"`
	Lang      string `json:"lang,omitempty"`
}

// LabWaypointResponse expose le résultat d'un appel Discovery UGC live (Atelier).
// Les erreurs d'appel (404, auth, token indisponible) sont portées dans Error
// avec ResolvedOK=false (réponse 200 côté HTTP : l'exploration a abouti, l'asset
// non) — pas une erreur HTTP, pour que le panneau affiche le détail.
type LabWaypointResponse struct {
	Segment     string `json:"segment"`
	Endpoint    string `json:"endpoint"`
	AssetID     string `json:"asset_id"`
	VersionID   string `json:"version_id"`
	Lang        string `json:"lang"`
	ResolvedOK  bool   `json:"resolved_ok"`
	AssetName   string `json:"asset_name,omitempty"`
	Description string `json:"description,omitempty"`
	ImageURL    string `json:"image_url,omitempty"`
	Error       string `json:"error,omitempty"`
	LatencyMS   int64  `json:"latency_ms"`
}
