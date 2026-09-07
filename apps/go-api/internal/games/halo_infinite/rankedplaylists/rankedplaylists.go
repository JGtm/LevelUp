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
// Package leaf volontairement sans dépendance interne AU RESTE DE L'APPLICATION
// (importable par migration ET sync sans cycle) — la seule exception est
// internal/games/mappings, un loader de config pur (zéro dépendance vers sync/
// migrations en retour, vérifié) qui porte les libellés EN/FR embarqués
// (cf. ranked_playlists_labels.toml, lot libellés L3 2026-09-07 : les noms
// étaient auparavant des littéraux Go, cf. .ai/PLAN_LIBELLES_EN_DUR_GO_2026-09-07.md).
package rankedplaylists

import (
	"embed"
	"fmt"
	"strings"

	"levelup/go-api/internal/games/mappings"
)

//go:embed ranked_playlists_labels.toml
var rankedPlaylistLabelsFS embed.FS

// rankedPlaylistLabelKind : kind TOML portant les libellés EN/FR par asset_id
// de playlist classée (ranked_playlists_labels.toml, embarqué — pas le
// assets.toml canonique du titre, cf. commentaire de ce fichier TOML pour la
// justification et le chemin d'évolution).
const rankedPlaylistLabelKind = "playlist_ranked"

// rankedPlaylistLabels charge une fois au chargement du package (embed = contenu
// figé au build, donc une erreur ici est un bug de compilation à corriger, pas
// une dégradation runtime — même contrat que les autres catalogues embarqués).
var rankedPlaylistLabels = mustLoadRankedPlaylistLabels()

func mustLoadRankedPlaylistLabels() *mappings.AssetMappingSet {
	raw, err := rankedPlaylistLabelsFS.ReadFile("ranked_playlists_labels.toml")
	if err != nil {
		panic(fmt.Errorf("rankedplaylists: lecture ranked_playlists_labels.toml embarqué: %w", err))
	}
	set, err := mappings.LoadAssetsFromBytes("ranked_playlists_labels.toml", raw)
	if err != nil {
		panic(fmt.Errorf("rankedplaylists: validation ranked_playlists_labels.toml embarqué: %w", err))
	}
	return set
}

// Valeurs de Queue/Input — externalisées en constantes (goconst : "crossplay"
// répété sur les 14 playlists open-queue de all ci-dessous ; CLAUDE.md règle 6,
// ≤ 2 copies d'un même littéral).
const (
	queueOpen    = "open-queue"
	queueSoloDuo = "solo-duo"

	inputCrossplay  = "crossplay"
	inputMnK        = "mnk"
	inputController = "controller"
)

// Playlist décrit une playlist classée Halo Infinite.
type Playlist struct {
	AssetID string // playlist_asset_id (UUID, stable entre saisons)
	Queue   string // queueOpen | queueSoloDuo
	Input   string // inputCrossplay | inputMnK | inputController
	Active  bool   // hopper actuellement actif (sinon classé mais retiré du matchmaking)
}

// NameEN retourne le nom canonique EN (ranked_playlists_labels.toml). "" si
// l'asset_id n'a pas d'entrée (ne devrait pas arriver pour un Playlist de all).
func (p Playlist) NameEN() string { return rankedPlaylistLabelText(p.AssetID, mappings.LocaleEN) }

// NameFR retourne la traduction FR officielle (ranked_playlists_labels.toml).
// "" si non localisée.
func (p Playlist) NameFR() string { return rankedPlaylistLabelText(p.AssetID, mappings.LocaleFR) }

// rankedPlaylistLabelText lit le libellé BRUT pour la locale demandée, SANS le
// fallback EN de mappings.AssetMapping.Label (NameEN/NameFR doivent pouvoir
// retourner "" — contrat historique de ce package — plutôt que se rabattre
// silencieusement sur l'autre langue).
func rankedPlaylistLabelText(assetID, locale string) string {
	a, ok := rankedPlaylistLabels.Get(rankedPlaylistLabelKind, strings.ToLower(strings.TrimSpace(assetID)))
	if !ok {
		return ""
	}
	return a.Labels[locale]
}

// all est la liste complète des playlists classées connues (actives + retirées).
// Mesure 2026-05 via le proxy public SpartanRecord (HaloDotAPI metadata, ranked=true).
// Noms EN/FR : ranked_playlists_labels.toml (même AssetID en clé).
var all = []Playlist{
	// --- Actives (saison en cours) ---
	{AssetID: "edfef3ac-9cbe-4fa2-b949-8f29deafd483", Queue: queueOpen, Input: inputCrossplay, Active: true},
	{AssetID: "dcb2e24e-05fb-4390-8076-32a0cdb4326e", Queue: queueOpen, Input: inputCrossplay, Active: true},
	{AssetID: "fa5aa2a3-2428-4912-a023-e1eeea7b877c", Queue: queueOpen, Input: inputCrossplay, Active: true},
	{AssetID: "c94cb508-2fbd-450a-81db-bb74f7741d45", Queue: queueOpen, Input: inputCrossplay, Active: true},
	// --- Classées mais inactives (retirées du matchmaking ; conservées pour l'historique CSR) ---
	{AssetID: "6233381c-fc96-40b9-b1ff-f6a4de72dd7a", Queue: queueOpen, Input: inputCrossplay, Active: false},
	{AssetID: "57e417dd-7366-4dda-9bdd-2802151d5e81", Queue: queueOpen, Input: inputCrossplay, Active: false},
	{AssetID: "71734db4-4b8e-4682-9206-62b6eff92582", Queue: queueOpen, Input: inputCrossplay, Active: false},
	{AssetID: "28bfa5f4-89b0-47dc-86e8-1a7cc5b593fc", Queue: queueOpen, Input: inputCrossplay, Active: false},
	{AssetID: "a4a4453c-7a91-4b27-b952-2456c5ce3205", Queue: queueOpen, Input: inputCrossplay, Active: false},
	{AssetID: "6dc5f699-d6d9-41c4-bdf8-7ae11dec2d1b", Queue: queueOpen, Input: inputCrossplay, Active: false},
	{AssetID: "f3738fae-bd09-4fd1-9dea-e32f546bbbfd", Queue: queueOpen, Input: inputCrossplay, Active: false},
	{AssetID: "0b42053a-32c5-4c2d-b8b8-5f07274a0117", Queue: queueOpen, Input: inputCrossplay, Active: false},
	{AssetID: "7c60fb3e-656c-4ada-a085-293562642e50", Queue: queueOpen, Input: inputCrossplay, Active: false},
	{AssetID: "a883e7e1-9aca-4296-9009-3733a0ca8081", Queue: queueOpen, Input: inputCrossplay, Active: false},
	{AssetID: "f7eb8c71-fedb-4696-8c0f-96025e285ffd", Queue: queueSoloDuo, Input: inputMnK, Active: false},
	{AssetID: "f7f30787-f607-436b-bdec-44c65bc2ecef", Queue: queueSoloDuo, Input: inputController, Active: false},
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
