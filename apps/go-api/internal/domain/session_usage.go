// Package domain — session_usage.go : le vocabulaire partagé du résumé d'usage du film — les raisons
// machine d'un bloc indisponible, l'identité d'un joueur des fiches, et les niveaux d'arme écrits en
// base (lus par l'Emprise et les formes retenues).
package domain

// Raisons machine d'un bloc indisponible (jamais un libellé : l'i18n vit au front).
const (
	// SessionUsageUnsupported : le titre ne déclare pas la capability
	// film.usage_summary — réponse partielle propre, jamais un 500 (ADR 0011).
	SessionUsageUnsupported = "unsupported"
	// SessionUsageLoadFailed : la lecture des vues a échoué (best-effort dégradé,
	// l'erreur est loggée côté service).
	SessionUsageLoadFailed = "load_failed"
)

// SessionUsageSquadPlayer — l'identité d'UN joueur des fiches (xuid, gamertag) ; l'ordre d'une
// liste est l'ordre d'affichage (jetons squad-player-1..3 côté front).
type SessionUsageSquadPlayer struct {
	XUID     string `json:"xuid"`
	Gamertag string `json:"gamertag"`
}

// Les NIVEAUX D'ARME, tels qu'ils sont écrits en base (`match_pad_pickups_by_tier`) et lus par
// l'Emprise. UNE SEULE SOURCE : le paquet `persist` réexporte CES constantes (il ne peut pas
// diverger par construction). Ce sont des valeurs de DONNÉE, jamais des libellés.
const (
	PadTierBase         = "base"
	PadTierGround       = "terrain"
	PadTierPower        = "puissance"
	PadTierPowerup      = "bonus"
	PadTierUnclassified = "non_classe"
	// PadTierNoPickup — LE ZÉRO MESURÉ d'un joueur qui n'a pris aucun socle. Ce n'est pas un
	// niveau de contrôle : c'est ce qui sépare « ce joueur n'a rien pris » de « on n'a pas
	// regardé ».
	PadTierNoPickup = "aucune_prise"
)
