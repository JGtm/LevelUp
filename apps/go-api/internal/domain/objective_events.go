// Package domain — objective_events.go : structs neutres représentant les rows
// des tables shared.match_objective_events + shared.match_objective_event_players
// (pipeline v3 film, mode-agnostique).
//
// **Localisation domain** : comme WeaponKillRow (cf. match_rows.go), ces structs
// sont consommées par plusieurs packages — le backfill diagnostic v3
// (internal/sync ou un CLI dédié) qui décode les films + produit les events, et
// la couche persistence (internal/platform/duckdb). Pour éviter les cycles
// d'import, elles vivent ici en zone neutre.
//
// Schéma : voir internal/migration/steps_shared_objective_events.go. Les pointeurs
// (TeamID, ObjectiveID, Value) modélisent les colonnes NULL-able : team unreliable
// sur certains matchs (mappé xuid->team via match_participants en amont),
// objective_id (identité zone/colline) non récupérable du film -> toujours NULL.

package domain

// ObjectiveEvent représente une ligne de shared.match_objective_events plus ses
// joueurs associés (shared.match_objective_event_players).
//
// PK de la table parente : (MatchID, Seq). Seq est un compteur dense 0..N-1
// ordonnant les events d'un même match (assigné par le producteur).
//
// ObjectiveType = parent mode-agnostique (flag|zone|hill|skull|bomb) ; EventType =
// action (ex. capture, score). Source/Confidence tracent la provenance et la
// précision du décodage (CTF ms-exact vs Strongholds/KOTH/Oddball ~5-20s).
// Details est un JSON-as-VARCHAR (échappatoire pour les champs non modélisés).
type ObjectiveEvent struct {
	MatchID       string
	Seq           int
	TimeMS        *int
	ObjectiveType string
	EventType     string
	TeamID        *int
	ObjectiveID   *int
	Value         *int
	Source        string
	Confidence    string
	Details       string
	Players       []ObjectiveEventPlayer
}

// ObjectiveEventPlayer représente une ligne de
// shared.match_objective_event_players (un joueur impliqué dans un event).
//
// PK de la table : (MatchID, Seq, XUID). MatchID/Seq sont portés par
// l'ObjectiveEvent parent — seuls XUID + Role sont stockés ici.
type ObjectiveEventPlayer struct {
	XUID string
	Role string
}
