// Package weaponv3 — fondations de l'attribution d'armes v3 (source-first).
//
// Contrairement au pipeline v2 (corrélation fire-event uniquement, dans
// internal/analysis/weapon_*.go), la v3 ajoute des signaux DIRECTS issus des
// marqueurs film (melee, grenade) et durcit la corrélation fire (timing µs,
// canon high-32). Cette couche 1 ne contient que les types + l'outillage
// bas-niveau (canon arme, résolveur pi, estimateur de timing) ; les scanners et
// l'orchestrateur arrivent dans les couches suivantes.
//
// Réfs : .ai/PLAN_WEAPON_ATTRIBUTION_V3.md, .ai/REFERENCE_WEAPON_IDS.md,
// .ai/RESEARCH_THEATER_RE.md.
package weaponv3

// SourceSignal — provenance du signal d'attribution v3 (champ AttributionV3.SourceSignal).
//
// Hiérarchie de confiance : un melee/grenade issu d'un marqueur film est un
// signal DIRECT (haute confiance), un fire-event reste indirect (corrélation
// temporelle), formula_a est le fallback bruité v2, none = non résolu.
const (
	SignalMelee    = "melee"
	SignalGrenade  = "grenade"
	SignalFire     = "fire"
	SignalFormulaA = "formula_a"
	SignalNone     = "none"
)

// Niveaux de confiance v3 (mêmes libellés que la v2 analysis.confidence*, mais ce
// dernier est unexporté). Réutilisés par l'orchestrateur correlate.go.
const (
	confidenceHighV3   = "high"
	confidenceMediumV3 = "medium"
	confidenceLowV3    = "low"
)

// AttributionV3 — résultat d'attribution d'un kill à une arme (superset de
// analysis.KillAttribution).
//
// Les champs identité/confiance/path reprennent la sémantique v2 (les consts
// analysis.AttributionPath* restent la source de vérité pour AttributionPath).
// Les champs v3 ajoutent la provenance du signal et les bits décodés du
// fire-event (high-32 de l'arme, hit/miss du coup fatal, burst-final, compteur
// de tirs) — cf. .ai/REFERENCE_WEAPON_IDS.md §"Structure du WEAPON event".
type AttributionV3 struct {
	// --- Coeur (aligné sur analysis.KillAttribution) ---
	MatchID         string
	XUID            string
	TimeMS          int
	WeaponID        *uint64 // Attribution film 64-bit complète (nil si non résolu)
	DeltaMS         *int    // Écart signal → kill (nil si pas de signal temporel)
	Confidence      string  // "high", "medium", "low", "none"
	AttributionPath string  // consts analysis.AttributionPath* (fire_event, formula_a, none)
	PlayerIndex     *int    // pi du film (0-31), nil si non résolu
	SourceChunkIdx  *int    // index du chunk source du signal

	// --- Champs v3 ---
	SourceSignal   string  // const Signal* — provenance du signal d'attribution
	HighWeaponID   *uint32 // high-32 canonique de l'arme (identité, cf. CanonWeaponID)
	KillingShotHit *bool   // bit hit/miss du coup fatal (true = touché), nil si inconnu
	BurstFinal     *bool   // bit burst-final du fire-event (true = dernier tir du burst)
	ShotCounter    *int    // compteur de tirs du fire-event (0..127, reset par arme)
}

// MeleeHit — marqueur melee décodé depuis un chunk film (réf §K-bis).
//
// Donne la VRAIE arme de corps-à-corps (épée, marteau, crosse de pistolet) via
// son weapon-id, ancrée sur le pi (octet@+20 bits 0-4) et le timestamp.
//
// Deux type-bytes distincts cohabitent dans le marqueur :
//   - HitType  = type-byte @+76 (§K-bis) ∈ {0x42, 0x47, 0x60} = nature du coup :
//     0x42 = miss/unpowered NON-LÉTAL (whiff, pistol-whip raté), 0x47 = HIT
//     marteau, 0x60 = HIT épée/coup chargé. Sélectionne aussi l'offset weapon-id.
//   - AnimType = nibble de direction d'animation (@weapon-id-4), 0x5/0xd = sens
//     du swing PAR ARME (table acurtis). N'indique PAS hit/miss.
//
// La récupération melee→kill (correlate.go) ne considère candidats QUE les
// HitType LÉTAUX (0x47/0x60) ; 0x42 est un whiff exclu.
type MeleeHit struct {
	PI       int
	TimeMS   int
	WeaponID uint64
	HitType  byte // type-byte @+76 : 0x42 miss / 0x47 hammer hit / 0x60 sword hit
	AnimType byte // nibble direction d'animation (par arme), pas hit/miss
}

// MeleeHitLethal indique si le type-byte @+76 correspond à un coup LÉTAL (HIT
// marteau 0x47 ou HIT épée/chargé 0x60). 0x42 (miss/unpowered) est exclu : un
// whiff ne tue pas. Référence §K-bis.
func MeleeHitLethal(hitType byte) bool {
	return hitType == meleeHitHammer || hitType == meleeHitSword
}

// GrenadeThrow — marqueur de lancer de grenade décodé depuis un chunk film
// (réf §C, marqueur 0x4c0c00, weapon@+24, allowlist 4 grenades).
//
// WeaponID est sur 32 bits ici (high-32 directement) car le marqueur grenade
// n'expose pas le suffixe bas comme un fire-event.
type GrenadeThrow struct {
	TimeMS   int
	WeaponID uint32
}
