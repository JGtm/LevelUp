// Package objectiveevent porte L'EVENEMENT OBJECTIF tel qu'il entre en base : les rows
// des tables shared.match_objective_events + shared.match_objective_event_players
// (pipeline v3 film, mode-agnostique).
//
// POURQUOI UN PAQUET FEUILLE (2026-09-26, jalon J3 de la suite d'audit du decodeur de
// film). Ces deux structs vivaient dans `internal/domain`. Or le perimetre de revision
// d'une couche du decodeur est la FERMETURE DE SES IMPORTS : la couche
// `film/internal/facts/objectives`, qui produit ces events, importait `internal/domain`
// pour ces deux seuls types, et tirait avec lui `internal/domain/title` et
// `internal/games/canonical` — toute modification de `internal/domain` faisait alors
// rougir le gate de `objectives.Rev`. Isoler les types ici, sans aucune dependance, garde
// ce perimetre etroit. Precedent : `internal/domain/highlightevent`. Ce paquet ne doit
// importer AUCUN paquet du depot.
//
// Consommateurs : le decodeur (qui produit les events), la persistence
// (internal/platform/duckdb), le service et le handler de la vue match, l'analyse
// (comeback objectif) et le CLI diagnostic v3.
//
// Schema : voir internal/migration/steps_shared_objective_events.go. Les pointeurs
// (TeamID, ObjectiveID, Value) modelisent les colonnes NULL-able.
//
// `TeamID` VIENT DU FILM DEPUIS LE LOT 1.7.3 (2026-09-14) : l'octet 37 du bloc de pied, lu par
// `objectives`. La phrase d'origine de ce commentaire — « team unreliable sur certains
// matchs (mappé xuid->team via match_participants en amont) » — decrivait l'octet 55, qui vaut 0
// partout ; elle est REFUTEE (665/665 sur quatorze films). Le champ reste NULL-able parce qu'un
// evenement sans acteur identifie n'a pas d'equipe a porter. `objective_id` (identite
// zone/colline) n'est toujours pas recuperable du film -> toujours NULL.
package objectiveevent

// Event represente une ligne de shared.match_objective_events plus ses
// joueurs associes (shared.match_objective_event_players).
//
// PK de la table parente : (MatchID, Seq). Seq est un compteur dense 0..N-1
// ordonnant les events d'un meme match (assigne par le producteur).
//
// ObjectiveType = parent mode-agnostique (flag|zone|hill|skull|bomb) ; EventType =
// action (ex. capture, score). Source/Confidence tracent la provenance et la
// precision du decodage (CTF ms-exact vs Strongholds/KOTH/Oddball ~5-20s).
// Details est un JSON-as-VARCHAR (echappatoire pour les champs non modelises).
type Event struct {
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
	Players       []Player
}

// Player represente une ligne de shared.match_objective_event_players (un joueur
// implique dans un event).
//
// PK de la table : (MatchID, Seq, XUID). MatchID/Seq sont portes par
// l'Event parent — seuls XUID + Role sont stockes ici.
type Player struct {
	XUID string
	Role string
}
