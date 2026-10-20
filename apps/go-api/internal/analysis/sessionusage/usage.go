// Package sessionusage — LES LIGNES DU RÉSUMÉ D'USAGE DU FILM et leur assemblage par match.
//
// Fonctions PURES (zéro DB, zéro HTTP) : la couche repo (platform/duckdb, SessionUsageRepo) lit
// les vues `match_usage_players_latest` / `match_usage_films_latest` (ADR 0026 — JAMAIS les tables
// brutes) et `match_participants` ; ce package porte ces lignes, le contexte de camp de chaque match
// (team_context.go), les issues d'équipement d'un joueur (usage_outcomes*.go), les niveaux de
// socle (pad_tiers.go) et les amis d'un scope (squad.go). Les blocs qui les lisent — l'Emprise
// (analysis/squademprise) et les formes retenues (analysis/squadformes) — vivent ailleurs.
//
// UN MATCH EST MESURÉ S'IL A UNE LIGNE FILM ; le camp du joueur vient de match_participants (camp
// inconnu : FFA ou participant absent — jamais un camp inventé).
package sessionusage

// PlayerRow — la ligne (match, joueur) telle que servie par
// match_usage_players_latest. Les grenades n'y figurent PAS : produites en S1,
// exclues du contrat de session (décision utilisateur 2026-09-04).
type PlayerRow struct {
	MatchID            string
	XUID               string
	GrapplePulls       int
	CamoEpisodes       int
	OvershieldEpisodes int
	// CamoMS / CamoKills / OvershieldMS / OvershieldKills : durée cumulée des épisodes actifs
	// (ms réelles, 0 si l'artefact n'a pas d'échelle de temps) et frags du porteur PENDANT
	// ces épisodes. Lus pour l'Emprise (D11 : production des bonus), jamais avant.
	CamoMS             int64
	CamoKills          int
	OvershieldMS       int64
	OvershieldKills    int
	DroppedObjects     int
	PadPickups         int
	DeployedByFamily   map[string]int
	PadPickupsByFamily map[string]int
	// Les QUATRE ventilations d'issue (colonnes taken/spent/kept/dropped_json,
	// révision de projection `us4`). Toutes dans le vocabulaire des POSES : les
	// quatre se joignent sur UNE clé de famille, sans pont. Une ligne écrite par
	// une passe antérieure les porte vides — ce qui rend la grandeur absente, pas
	// nulle.
	TakenByFamily   map[string]int
	SpentByFamily   map[string]int
	KeptByFamily    map[string]int
	DroppedByFamily map[string]int
}

// FilmRow — la ligne de grain match de match_usage_films_latest (l'existence de
// cette ligne EST la définition de « match mesuré »). Seules les colonnes
// CONSOMMÉES sont portées (0 code mort) : l'échelle de temps et les prises de
// bonus par famille, lues par l'Emprise.
type FilmRow struct {
	MatchID        string
	DurationMS     int64
	PowerupPickups map[string]int
}

// ParticipantRow — un participant du match (match_participants) : l'appartenance
// de camp pour l'attribution, la présence à la fin pour les effectifs, le
// gamertag pour nommer les joueurs.
type ParticipantRow struct {
	MatchID             string
	XUID                string
	Gamertag            string
	TeamID              *int
	PresentAtCompletion bool
}

// MatchInput — un match du scope, assemblé par BuildMatchInputs.
type MatchInput struct {
	MatchID  string
	Measured bool
	// PlayerTeam : camp du joueur suivi (nil = inconnu — les parts d'équipe de ce
	// match sont hors calcul). TeamOf : xuid -> camp, pour classer chaque ligne.
	PlayerTeam *int
	TeamOf     map[string]int
	Players    []PlayerRow
}

// BuildMatchInputs — un MatchInput par match du scope, DANS L'ORDRE DONNÉ, mesuré
// s'il porte une ligne film. SOURCE UNIQUE de l'assemblage : ses lecteurs n'ont pas
// à regrouper les lignes joueur ni à recopier le contexte de camp.
func BuildMatchInputs(
	matchIDs []string, films map[string]FilmRow, players []PlayerRow, tc TeamContext,
) []MatchInput {
	playersByMatch := make(map[string][]PlayerRow, len(films))
	for _, p := range players {
		playersByMatch[p.MatchID] = append(playersByMatch[p.MatchID], p)
	}
	out := make([]MatchInput, 0, len(matchIDs))
	for _, id := range matchIDs {
		_, measured := films[id]
		m := MatchInput{
			MatchID:  id,
			Measured: measured,
			TeamOf:   tc.TeamOf[id],
			Players:  playersByMatch[id],
		}
		if team, ok := tc.PlayerTeam[id]; ok {
			t := team
			m.PlayerTeam = &t
		}
		out = append(out, m)
	}
	return out
}
