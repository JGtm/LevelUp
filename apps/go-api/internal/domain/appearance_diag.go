// Package domain — appearance_diag.go : DTO du diagnostic apparence Spartan ID.
// Surface admin, à la
// demande, par joueur suivi. Structure neutre (aucune dépendance vers un titre ni duckdb)
// sérialisée telle quelle par le handler Huma GET /admin/diag/appearance/{player_slug}.
package domain

import "strings"

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

// AppearanceVerdict — enum FERMÉ (5 valeurs) du diagnostic d'un composant d'apparence. Le résolveur
// d'un titre n'émet que ok, upstream_missing et transient ; auth_required et not_supported sont posés
// par le service du diagnostic.
type AppearanceVerdict string

const (
	// AppearanceVerdictOK — le live résout, la valeur servie est à jour.
	AppearanceVerdictOK AppearanceVerdict = "ok"
	// AppearanceVerdictUpstreamMissing — absence DÉFINITIVE côté éditeur (nameplate absente de la table
	// de correspondance et aucune configuration positive au CMS). Rien à faire : la dernière valeur
	// connue est servie par design ; elle se réparera seule si l'éditeur publie l'image.
	AppearanceVerdictUpstreamMissing AppearanceVerdict = "upstream_missing"
	// AppearanceVerdictTransient — échec réseau, HTTP ou de lecture indéterminé. Se répare seul au
	// prochain refresh.
	AppearanceVerdictTransient AppearanceVerdict = "transient"
	// AppearanceVerdictAuthRequired — jetons absents ou morts (401/403 propriétaire). Posé par le
	// service, jamais par un résolveur.
	AppearanceVerdictAuthRequired AppearanceVerdict = "auth_required"
	// AppearanceVerdictNotSupported — le titre ne fournit pas ce composant par ce pipeline (capacité
	// absente). Posé par le service, jamais par un résolveur.
	AppearanceVerdictNotSupported AppearanceVerdict = "not_supported"
)

// AppearanceServedFrom indique d'où provient la valeur actuellement servie d'un composant : de la
// résolution live (live) ou du report de la dernière valeur connue (carry). Un verdict autre que ok
// donne carry.
type AppearanceServedFrom string

const (
	AppearanceServedFromLive  AppearanceServedFrom = "live"
	AppearanceServedFromCarry AppearanceServedFrom = "carry"
)

// AppearanceDetail — clé technique qui explique le verdict (diagnostic fin, NON traduit : la
// localisation est posée côté front).
type AppearanceDetail string

const (
	// Émis par le résolveur de nameplate d'un titre.
	AppearanceDetailMappingHit    AppearanceDetail = "mapping_hit"     // ok par la table de correspondance (palette exacte)
	AppearanceDetailMappingMiss   AppearanceDetail = "mapping_miss"    // ok par une configuration positive (palette approchée)
	AppearanceDetailNoPositiveCfg AppearanceDetail = "no_positive_cfg" // upstream_missing : CMS 200 sans configuration positive
	AppearanceDetailCMSHTTPError  AppearanceDetail = "cms_http_error"  // transient : lecture du CMS en échec
	AppearanceDetailNoEmblemPath  AppearanceDetail = "no_emblem_path"  // aucun chemin d'emblème fourni au résolveur
	AppearanceDetailNonEmblemPath AppearanceDetail = "non_emblem_path" // chemin fourni qui n'est pas un emblème

	// Émis par le diagnostic d'une image de customisation (emblème, backdrop).
	AppearanceDetailImageResolved   AppearanceDetail = "image_resolved"   // ok : descripteur CMS -> URL d'image
	AppearanceDetailImageUnresolved AppearanceDetail = "image_unresolved" // transient : résolution d'image en échec

	// Émis par DiagnoseServiceTag.
	AppearanceDetailServiceTagPresent AppearanceDetail = "service_tag_present" // ok : tag présent au payload
	AppearanceDetailNoServiceTag      AppearanceDetail = "no_service_tag"      // tag absent du payload
)

// AppearanceDiagnosis — diagnostic d'UN composant d'apparence (bannière, emblème, backdrop ou service
// tag), sans condition croisée entre composants.
type AppearanceDiagnosis struct {
	// ServedFrom : live (résolu maintenant) ou carry (dernière valeur connue).
	ServedFrom AppearanceServedFrom
	// ResolvedURL : URL résolue en live ; vide si rien n'est résolu, ou pour un composant sans URL
	// (service tag).
	ResolvedURL string
	Verdict     AppearanceVerdict
	Detail      AppearanceDetail
}

// AppearanceInputs — les champs bruts de l'apparence que le diagnostic par composant lit : chemins
// d'inventaire et identifiant de configuration de l'emblème, pas des URL résolues.
type AppearanceInputs struct {
	ServiceTag        string
	BannerImagePath   string
	BackdropImagePath string
	EmblemPath        string
	// EmblemConfigID : ConfigurationId de l'emblème (0 si absent ou illisible).
	EmblemConfigID int64
}

// DiagnoseServiceTag diagnostique le service tag : présent au payload -> ok (live) ; absent ->
// transient (la dernière valeur connue reste servie). Composant SANS URL : le service porte la valeur
// textuelle.
func DiagnoseServiceTag(serviceTag string) AppearanceDiagnosis {
	if strings.TrimSpace(serviceTag) == "" {
		return AppearanceDiagnosis{
			ServedFrom: AppearanceServedFromCarry,
			Verdict:    AppearanceVerdictTransient,
			Detail:     AppearanceDetailNoServiceTag,
		}
	}
	return AppearanceDiagnosis{
		ServedFrom: AppearanceServedFromLive,
		Verdict:    AppearanceVerdictOK,
		Detail:     AppearanceDetailServiceTagPresent,
	}
}
