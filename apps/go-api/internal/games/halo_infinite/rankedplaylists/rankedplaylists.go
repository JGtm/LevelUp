// Package rankedplaylists — référence autoritative des playlists CLASSÉES
// (ranked) Halo Infinite. Source de vérité unique pour `is_ranked`, indépendante
// de l'historique des joueurs (cf. retour utilisateur : ne JAMAIS dériver la liste
// des parties — peu de joueurs jouent toutes les playlists, c'est fragile).
//
// Origine des données : métadonnée curée HaloDotAPI
// `/games/halo-infinite/metadata/multiplayer/playlists` (champ `ranked`), la même
// source que le projet voisin SpartanRecord. Les asset_id sont stables entre
// saisons ; seul l'identifiant de saison CSR change (résolu dynamiquement par
// LevelUp via csr_season_calendars). Le rang par saison se lit ensuite par
// playlist via l'endpoint skill `/hi/playlist/{id}/csrs?season=` (mécanisme Grunt
// Skill.GetPlaylistCsr), qui renvoie "Non classé" si la playlist n'a jamais été
// jouée.
//
// Refresh : relancer le fetch métadonnée (cf. cmd de régénération) quand 343
// ajoute/retire un hopper classé — événement rare (~1 par opération/saison).
//
// Package leaf volontairement sans dépendance interne (importable par migration
// ET sync sans cycle).
package rankedplaylists

import "strings"

// Playlist décrit une playlist classée Halo Infinite.
type Playlist struct {
	AssetID string // playlist_asset_id (UUID, stable entre saisons)
	NameEN  string // nom canonique EN
	NameFR  string // traduction FR officielle ("" si non localisée)
	Queue   string // "open-queue" | "solo-duo"
	Input   string // "crossplay" | "mnk" | "controller"
	Active  bool   // hopper actuellement actif (sinon classé mais retiré du matchmaking)
}

// all est la liste complète des playlists classées connues (actives + retirées).
// Mesure 2026-05 via le proxy public SpartanRecord (HaloDotAPI metadata, ranked=true).
var all = []Playlist{
	// --- Actives (saison en cours) ---
	{AssetID: "edfef3ac-9cbe-4fa2-b949-8f29deafd483", NameEN: "Ranked Arena", NameFR: "Arène classée", Queue: "open-queue", Input: "crossplay", Active: true},
	{AssetID: "dcb2e24e-05fb-4390-8076-32a0cdb4326e", NameEN: "Ranked Slayer", NameFR: "Assassin classé", Queue: "open-queue", Input: "crossplay", Active: true},
	{AssetID: "fa5aa2a3-2428-4912-a023-e1eeea7b877c", NameEN: "Ranked Doubles", NameFR: "Duo classé", Queue: "open-queue", Input: "crossplay", Active: true},
	{AssetID: "c94cb508-2fbd-450a-81db-bb74f7741d45", NameEN: "RANKED LEGACY", NameFR: "Legacy classé", Queue: "open-queue", Input: "crossplay", Active: true},
	// --- Classées mais inactives (retirées du matchmaking ; conservées pour l'historique CSR) ---
	{AssetID: "6233381c-fc96-40b9-b1ff-f6a4de72dd7a", NameEN: "Ranked Snipers", NameFR: "Snipers classés", Queue: "open-queue", Input: "crossplay", Active: false},
	{AssetID: "57e417dd-7366-4dda-9bdd-2802151d5e81", NameEN: "Ranked Tactical", NameFR: "Tactique classé", Queue: "open-queue", Input: "crossplay", Active: false},
	{AssetID: "71734db4-4b8e-4682-9206-62b6eff92582", NameEN: "Ranked FFA", NameFR: "Chacun pour soi classé", Queue: "open-queue", Input: "crossplay", Active: false},
	{AssetID: "28bfa5f4-89b0-47dc-86e8-1a7cc5b593fc", NameEN: "RANKED 1V1 SHOWDOWN", NameFR: "", Queue: "open-queue", Input: "crossplay", Active: false},
	{AssetID: "a4a4453c-7a91-4b27-b952-2456c5ce3205", NameEN: "Ranked S&D Extraction", NameFR: "", Queue: "open-queue", Input: "crossplay", Active: false},
	{AssetID: "6dc5f699-d6d9-41c4-bdf8-7ae11dec2d1b", NameEN: "Ranked Squad Battle", NameFR: "", Queue: "open-queue", Input: "crossplay", Active: false},
	{AssetID: "f3738fae-bd09-4fd1-9dea-e32f546bbbfd", NameEN: "Ranked Survivors", NameFR: "", Queue: "open-queue", Input: "crossplay", Active: false},
	{AssetID: "0b42053a-32c5-4c2d-b8b8-5f07274a0117", NameEN: "HaloWC FFA Qualifier", NameFR: "", Queue: "open-queue", Input: "crossplay", Active: false},
	{AssetID: "7c60fb3e-656c-4ada-a085-293562642e50", NameEN: "RANKED TACTICAL OLD", NameFR: "", Queue: "open-queue", Input: "crossplay", Active: false},
	{AssetID: "a883e7e1-9aca-4296-9009-3733a0ca8081", NameEN: "RANKED SNIPERS OLD", NameFR: "", Queue: "open-queue", Input: "crossplay", Active: false},
	{AssetID: "f7eb8c71-fedb-4696-8c0f-96025e285ffd", NameEN: "Ranked Arena (Solo-Duo MnK)", NameFR: "Arène classée (Solo-Duo CSM)", Queue: "solo-duo", Input: "mnk", Active: false},
	{AssetID: "f7f30787-f607-436b-bdec-44c65bc2ecef", NameEN: "Ranked Arena (Solo-Duo Controller)", NameFR: "Arène classée (Solo-Duo Manette)", Queue: "solo-duo", Input: "controller", Active: false},
}

// rankedIndex : lookup O(1) par asset_id normalisé (lower-case).
var rankedIndex = func() map[string]Playlist {
	m := make(map[string]Playlist, len(all))
	for _, p := range all {
		m[strings.ToLower(p.AssetID)] = p
	}
	return m
}()

// All retourne toutes les playlists classées connues (actives + inactives).
func All() []Playlist {
	out := make([]Playlist, len(all))
	copy(out, all)
	return out
}

// Active retourne uniquement les playlists classées actuellement actives.
func Active() []Playlist {
	out := make([]Playlist, 0, len(all))
	for _, p := range all {
		if p.Active {
			out = append(out, p)
		}
	}
	return out
}

// IsRanked indique si un asset_id appartient à la référence des playlists classées.
// Casse-insensible. C'est le point d'autorité que tous les chemins d'écriture du
// catalogue doivent consulter pour fixer is_ranked.
func IsRanked(assetID string) bool {
	_, ok := rankedIndex[strings.ToLower(strings.TrimSpace(assetID))]
	return ok
}

// Lookup retourne la playlist de référence pour un asset_id (ok=false si absente).
func Lookup(assetID string) (Playlist, bool) {
	p, ok := rankedIndex[strings.ToLower(strings.TrimSpace(assetID))]
	return p, ok
}
