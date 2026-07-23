// Package domain — appearance_diag.go : DTO du diagnostic apparence Spartan ID
// (volet 2 du plan .ai/PLAN_DIAG_APPARENCE_ADMIN_2026-07.md). Surface admin, à la
// demande, par joueur suivi. Structure neutre (aucune dépendance haloclient/duckdb)
// sérialisée telle quelle par le handler Huma GET /admin/diag/appearance/{player_slug}.
package domain

// Composants du Spartan ID diagnostiqués (clés stables, consommées côté front).
const (
	AppearanceComponentBanner     = "banner"
	AppearanceComponentEmblem     = "emblem"
	AppearanceComponentBackdrop   = "backdrop"
	AppearanceComponentServiceTag = "service_tag"
)

// AppearanceComponentDiagnosis — diagnostic d'UN composant du Spartan ID.
//   - ServedValue : la valeur ACTUELLEMENT servie par l'app (dernière valeur
//     connue en DB via LoadLastCareerRank : URL pour banner/emblem/backdrop, texte
//     pour service_tag). Jamais recalculée ici — c'est ce que voit l'utilisateur.
//   - ServedFrom : "live" (la résolution live confirme la valeur servie) ou
//     "carry" (dernière valeur connue reportée, la résolution live n'a rien produit).
//   - Verdict : enum fermé (ok / upstream_missing / transient / auth_required /
//     not_supported) — le POURQUOI actionnable.
//   - Detail : clé technique non traduite (mapping_miss, no_positive_cfg, …) ; la
//     localisation FR/EN est posée côté front (Lot G).
type AppearanceComponentDiagnosis struct {
	Component   string `json:"component"`
	ServedValue string `json:"served_value"`
	ServedFrom  string `json:"served_from"`
	Verdict     string `json:"verdict"`
	Detail      string `json:"detail"`
}

// AppearanceDiagnosisResponse — réponse du diagnostic apparence d'un joueur suivi.
// LastFetchStatus reflète l'issue du DERNIER fetch live persisté
// (career_progression.last_fetch_status : ok / api_empty / forbidden_403 /
// auth_missing / failed / "" si jamais tenté) — contexte passif, distinct des
// verdicts par composant (calculés à la demande).
type AppearanceDiagnosisResponse struct {
	PlayerSlug      string                         `json:"player_slug"`
	Gamertag        string                         `json:"gamertag"`
	XUID            string                         `json:"xuid"`
	TitleSlug       string                         `json:"title_slug"`
	GeneratedAt     string                         `json:"generated_at"`
	LastFetchStatus string                         `json:"last_fetch_status"`
	Components      []AppearanceComponentDiagnosis `json:"components"`
}
