// Package domain — admin_data_quality.go : payloads qualité données du
// dashboard monitoring admin (compteurs d'inconnus, listes détaillées,
// requêtes/réponses des actions de résolution).
package domain

// AdminDataQualityCounts est la réponse de GET /admin/monitoring/data-quality.
type AdminDataQualityCounts struct {
	TitleSlug   string `json:"title_slug"`
	GeneratedAt string `json:"generated_at"` // RFC3339
	// Locale de traduction visée par le compteur untranslated_modes (défaut « fr » —
	// paramètre ?locale=). Rend le libellé front honnête (« Modes sans traduction (fr) »).
	Locale string `json:"locale"`

	// Assets dont *_name == *_id dans match_registry (cible de l'action
	// registry-names/backfill).
	RawUUIDPlaylists int `json:"raw_uuid_playlists"`
	RawUUIDMaps      int `json:"raw_uuid_maps"`
	RawUUIDPairs     int `json:"raw_uuid_pairs"`
	RawUUIDVariants  int `json:"raw_uuid_variants"`
	RawUUIDTotal     int `json:"raw_uuid_total"`

	// Modes (clé normalisée) absents de mode_name_tr[fr] — cible de l'action
	// translations/mode.
	UntranslatedModes int `json:"untranslated_modes"`
	// Playlists de match_registry absentes de metadata.playlists_catalog.
	OrphanPlaylists int `json:"orphan_playlists"`
	// Participants sans alias gamertag (informatif — résorbé par la
	// convergence PSA opportuniste).
	OrphanXUIDs int `json:"orphan_xuids"`
	// Bits de complétion posés alors que la table cible est vide.
	LyingBitsEvents  int `json:"lying_bits_events"`
	LyingBitsWeapons int `json:"lying_bits_weapons"`
}

// AdminDataQualityIssue est une ligne détaillée d'inconnu.
type AdminDataQualityIssue struct {
	Kind        string `json:"kind"` // raw_uuid | untranslated_mode | orphan_playlist | orphan_xuid
	AssetKind   string `json:"asset_kind,omitempty"`
	ID          string `json:"id"`
	Label       string `json:"label,omitempty"`
	Occurrences int    `json:"occurrences"`
	LastSeen    string `json:"last_seen,omitempty"` // RFC3339
	// ExampleMatchIDs : jusqu'à 3 match_id concrets où l'inconnu apparaît, pour
	// ouvrir la vue de match et décider en connaissance de cause (nouvel onglet).
	ExampleMatchIDs []string `json:"example_match_ids,omitempty"`
}

// AdminDataQualityIssues est la réponse de GET .../data-quality/issues.
// Items est la fenêtre paginée [offset, offset+limit) ; Total est le nombre total
// d'inconnus de ce kind (avant fenêtrage) — alimente la pagination serveur du
// front (table longue des xuids orphelins).
type AdminDataQualityIssues struct {
	TitleSlug   string                  `json:"title_slug"`
	GeneratedAt string                  `json:"generated_at"`
	Kind        string                  `json:"kind"`
	// Locale de traduction visée (défaut « fr », paramètre ?locale=) — pertinente
	// pour untranslated_modes, échotée pour le libellé front honnête.
	Locale string                  `json:"locale"`
	Items  []AdminDataQualityIssue `json:"items"`
	Total  int                     `json:"total"`
}

// RegistryNamesBackfillRequest — corps de POST .../registry-names/backfill.
type RegistryNamesBackfillRequest struct {
	DryRun bool `json:"dry_run"`
}

// LyingBitsResetRequest — corps de POST .../lying-bits/reset.
type LyingBitsResetRequest struct {
	DryRun bool `json:"dry_run"`
}

// RegistryNamesBackfillResult — compteurs du backfill (ou du scan dry-run).
type RegistryNamesBackfillResult struct {
	DryRun           bool `json:"dry_run"`
	PlaylistsScanned int  `json:"playlists_scanned"`
	PlaylistsFixed   int  `json:"playlists_fixed"`
	MapsScanned      int  `json:"maps_scanned"`
	MapsFixed        int  `json:"maps_fixed"`
	PairsScanned     int  `json:"pairs_scanned"`
	PairsFixed       int  `json:"pairs_fixed"`
	VariantsScanned  int  `json:"variants_scanned"`
	VariantsFixed    int  `json:"variants_fixed"`
	TotalFixed       int  `json:"total_fixed"`
}

// ModeTranslationRequest — corps de POST .../translations/mode.
type ModeTranslationRequest struct {
	ModeEN string `json:"mode_en"`
	NameFR string `json:"name_fr"`
}

// AssetTranslationRequest — corps de POST .../translations/asset. Au moins un
// des deux noms doit être fourni ; chaque langue fournie est upsertée
// (en-US / fr-FR dans asset_translations).
type AssetTranslationRequest struct {
	AssetKind string `json:"asset_kind"` // playlist | map | pair | game_variant
	AssetID   string `json:"asset_id"`
	NameEN    string `json:"name_en,omitempty"`
	NameFR    string `json:"name_fr,omitempty"`
}

// ResolveResult — réponse des actions de résolution metadata.
type ResolveResult struct {
	// Action : "created" | "updated" (par langue pour les assets).
	Action string `json:"action"`
	// ModeEN : clé normalisée effectivement écrite (translations/mode).
	ModeEN string `json:"mode_en,omitempty"`
	// Langs : langues écrites (translations/asset).
	Langs []string `json:"langs,omitempty"`
}

// PlayerConvergenceRequest — corps de POST .../convergence/run.
type PlayerConvergenceRequest struct {
	PlayerSlug string `json:"player_slug"`
}

// CatalogRefreshResult — compteurs de POST .../catalog/refresh (seed des
// tables catalog metadata depuis match_registry, zéro réseau).
type CatalogRefreshResult struct {
	Playlists    int `json:"playlists"`
	Pairs        int `json:"pairs"`
	Maps         int `json:"maps"`
	GameVariants int `json:"game_variants"`
}

// CatalogUGCDrainResult — compteurs de POST .../catalog/ugc-drain (job async).
// Seeded = assets recensés dans la file ; les autres = résolus via l'API
// DiscoveryUGC (réseau, rate-limité). Errors = assets en échec après retries.
type CatalogUGCDrainResult struct {
	Seeded       int `json:"seeded"`
	Playlists    int `json:"playlists"`
	Pairs        int `json:"pairs"`
	Maps         int `json:"maps"`
	GameVariants int `json:"game_variants"`
	Errors       int `json:"errors"`
}

// LyingBitsResetResult — compteurs de POST .../lying-bits/reset. Les champs
// *Cleared comptent les matchs concernés : en dry-run ce qui SERAIT corrigé,
// en exécution ce qui A ÉTÉ corrigé (le WHERE est identique, le writer est
// exclusif → pas de divergence). Reset débloque le heal events/weapons au
// prochain sync delta.
type LyingBitsResetResult struct {
	DryRun              bool `json:"dry_run"`
	EventsBitsCleared   int  `json:"events_bits_cleared"`   // MBitEvents posé, highlight_events vide
	WeaponsBitsCleared  int  `json:"weapons_bits_cleared"`  // MBitWeaponKills posé, weapon_kills vide
	EventsLoadedCleared int  `json:"events_loaded_cleared"` // events_loaded=TRUE, highlight_events vide
	Total               int  `json:"total"`
}
