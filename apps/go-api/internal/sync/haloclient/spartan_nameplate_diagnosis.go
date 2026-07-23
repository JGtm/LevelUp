// Package haloclient — spartan_nameplate_diagnosis.go : diagnostic structuré de
// la résolution d'apparence Spartan (bannière/nameplate, emblème, backdrop,
// service tag) pour la surface admin « diagnostic apparence » (volet 2 du plan
// .ai/PLAN_DIAG_APPARENCE_ADMIN_2026-07.md).
//
// Ce fichier N'AJOUTE AUCUNE logique de fetch : il expose le POURQUOI de la
// résolution existante. DiagnoseNameplate partage la fonction interne
// resolveNameplate avec ResolveNameplateURL (aucune duplication du fetch
// mapping/CMS) ; DiagnoseCustomizationImage réutilise resolveCustomizationImageURL.
//
// La sémantique de LECTURE de l'apparence ne bouge PAS d'un octet : chaque
// composant est diagnostiqué INDÉPENDAMMENT, aucune condition croisée
// bannière↔emblème (verrouillé par les tests du 2026-07-08). Un verdict non-ok
// signifie SEULEMENT « le live n'a rien résolu » — la dernière valeur connue
// reste servie par le read-path (directive « jamais vide »).
//
// Note frontière hosts (garde-rail archlint des littéraux d'hôte) : aucun host
// en dur ici — les URLs sont produites par resolveNameplate et
// resolveCustomizationImageURL, qui passent par le resolver d'endpoints
// (nameplateHostFor / gameCMSHost).
package haloclient

import (
	"context"
	"strings"
)

// Verdict — enum FERMÉ (5 valeurs) du diagnostic d'un composant d'apparence.
// Le resolver haloclient n'émet QUE ok/upstream_missing/transient ;
// auth_required/not_supported sont posés par la couche service (Lot F) — ils
// sont définis ici pour que le type soit complet.
type Verdict string

const (
	// VerdictOK — le live résout, la valeur servie est à jour.
	VerdictOK Verdict = "ok"
	// VerdictUpstreamMissing — absence DÉFINITIVE côté Microsoft (nameplate
	// absente de mapping.json + aucune cfg positive au CMS). Rien à faire : la
	// dernière valeur connue est servie par design ; se réparera seul si
	// Microsoft publie l'image.
	VerdictUpstreamMissing Verdict = "upstream_missing"
	// VerdictTransient — échec réseau/HTTP/parse indéterminé. Se répare seul au
	// prochain refresh.
	VerdictTransient Verdict = "transient"
	// VerdictAuthRequired — tokens absents/morts (401/403 owner). Posé par le
	// service (Lot F) ; JAMAIS par le resolver.
	VerdictAuthRequired Verdict = "auth_required"
	// VerdictNotSupported — le titre ne fournit pas ce composant via ce pipeline
	// (capability absente). Posé par le service (Lot F) ; JAMAIS par le resolver.
	VerdictNotSupported Verdict = "not_supported"
)

// ServedFrom indique d'où provient la valeur actuellement servie pour le
// composant : de la résolution live (live) ou du report de la dernière valeur
// connue (carry). Dérivée du verdict par le resolver ; le service (Lot F) peut
// l'affiner avec l'état réel des dernières valeurs servies (LoadLastCareerRank).
type ServedFrom string

const (
	ServedFromLive  ServedFrom = "live"
	ServedFromCarry ServedFrom = "carry"
)

// Detail — clé technique expliquant le verdict (diagnostic fin, NON traduit ;
// la localisation FR/EN est posée côté front au Lot G).
type Detail string

const (
	// Émis par le resolver nameplate (resolveNameplate) :
	DetailMappingHit    Detail = "mapping_hit"     // ok via mapping.json (palette exacte)
	DetailMappingMiss   Detail = "mapping_miss"    // ok via cfg positive (mapping absent — palette approchée)
	DetailNoPositiveCfg Detail = "no_positive_cfg" // upstream_missing : CMS 200 sans cfg positive
	DetailCMSHTTPError  Detail = "cms_http_error"  // transient : fetch/parse CMS KO
	DetailNoEmblemPath  Detail = "no_emblem_path"  // aucun emblem path fourni au resolver
	DetailNonEmblemPath Detail = "non_emblem_path" // chemin fourni non /Spartan/Emblems/

	// Émis par DiagnoseCustomizationImage (emblème / backdrop) :
	DetailImageResolved   Detail = "image_resolved"   // ok : descriptor CMS → URL image
	DetailImageUnresolved Detail = "image_unresolved" // transient : resolve image KO (HTTP/parse/media path absent)

	// Émis par DiagnoseServiceTag :
	DetailServiceTagPresent Detail = "service_tag_present" // ok : tag présent au payload
	DetailNoServiceTag      Detail = "no_service_tag"      // tag absent du payload

	// Posé par la couche service (Lot F), JAMAIS par le resolver : ni
	// BannerImagePath ni Emblem exploitables pour dériver une bannière.
	DetailNoBannerField Detail = "no_banner_field"
)

// AppearanceDiagnosis — diagnostic d'UN composant d'apparence (bannière,
// emblème, backdrop ou service tag). Assemblé par composant par le service du
// Lot F ; produit ici sans aucune condition croisée entre composants.
type AppearanceDiagnosis struct {
	// ServedFrom : live (résolu maintenant) ou carry (dernière valeur connue).
	ServedFrom ServedFrom
	// ResolvedURL : URL live résolue ("" si non résolue, ou composant sans URL
	// comme le service tag).
	ResolvedURL string
	Verdict     Verdict
	Detail      Detail
}

// servedFromForVerdict dérive ServedFrom du verdict : ok signifie que le live a
// résolu (live) ; tout autre verdict signifie que la valeur servie est la
// dernière connue reportée (carry).
func servedFromForVerdict(v Verdict) ServedFrom {
	if v == VerdictOK {
		return ServedFromLive
	}
	return ServedFromCarry
}

// DiagnoseNameplate diagnostique la résolution de la BANNIÈRE dérivée de
// l'emblème (fallback nameplate — voir ResolveNameplateURL). Il s'appuie sur la
// MÊME fonction interne resolveNameplate : aucune duplication du fetch mapping/CMS.
// Verdicts possibles : ok / upstream_missing / transient.
func DiagnoseNameplate(
	ctx context.Context,
	emblemPath string,
	cfg int64,
	spartanToken, clearanceToken string,
) AppearanceDiagnosis {
	url, verdict, detail := resolveNameplate(ctx, emblemPath, cfg, spartanToken, clearanceToken)
	return AppearanceDiagnosis{
		ServedFrom:  servedFromForVerdict(verdict),
		ResolvedURL: url,
		Verdict:     verdict,
		Detail:      detail,
	}
}

// DiagnoseCustomizationImage diagnostique la résolution d'une image de
// customisation (emblème ou backdrop) via resolveCustomizationImageURL — le MÊME
// chemin que GetSpartanCustomization. Succès → ok ; tout échec (HTTP / parse /
// media path absent) → transient (indéterminé, se répare au prochain refresh).
func (c *HaloAPIClient) DiagnoseCustomizationImage(ctx context.Context, inventoryPath string) AppearanceDiagnosis {
	url, err := c.resolveCustomizationImageURL(ctx, inventoryPath)
	if err != nil || url == "" {
		return AppearanceDiagnosis{
			ServedFrom: ServedFromCarry,
			Verdict:    VerdictTransient,
			Detail:     DetailImageUnresolved,
		}
	}
	return AppearanceDiagnosis{
		ServedFrom:  ServedFromLive,
		ResolvedURL: url,
		Verdict:     VerdictOK,
		Detail:      DetailImageResolved,
	}
}

// DiagnoseServiceTag diagnostique le service tag : présent au payload → ok
// (live) ; absent → transient (la dernière valeur connue reste servie). Composant
// SANS URL (ResolvedURL vide) — le service (Lot F) porte la valeur textuelle.
func DiagnoseServiceTag(serviceTag string) AppearanceDiagnosis {
	if strings.TrimSpace(serviceTag) == "" {
		return AppearanceDiagnosis{
			ServedFrom: ServedFromCarry,
			Verdict:    VerdictTransient,
			Detail:     DetailNoServiceTag,
		}
	}
	return AppearanceDiagnosis{
		ServedFrom: ServedFromLive,
		Verdict:    VerdictOK,
		Detail:     DetailServiceTagPresent,
	}
}
