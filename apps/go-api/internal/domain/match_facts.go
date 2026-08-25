package domain

// match_facts.go — CE QUE LA BASE SAIT D'UN MATCH ET QUE LE FILM NE DIT PAS.
//
// POURQUOI CE TYPE VIT DANS `domain` ET NON DANS `port`. Il a d'abord été écrit à côté de
// `ReplayFactsRepo`, l'interface qui le lit — un placement naturel tant qu'il ne servait qu'à
// traverser cette porte. Il en sort désormais par une SECONDE : `BuildQueuePayload` le
// transporte jusqu'à l'ouvrier distant, qui n'a aucune base pour le résoudre lui-même.
//
// Et `domain` n'importe JAMAIS `port` (c'est `port` qui importe `domain`) : le laisser dans
// `port` rendrait la file de construction impossible à typer sans cycle d'import. Le type est
// de toute façon un fait de DOMAINE — ce qui est vrai d'un match — pas un contrat de port.
// `port.MatchFacts` reste disponible en ALIAS, si bien qu'aucun appelant n'a changé.

// MatchPlayerFact est la ligne de match d'un joueur, telle que le constructeur d'artefact de
// rejeu en a besoin.
//
// LE TRIPLET EST UNE CLÉ, pas une statistique d'affichage : (frags, morts, assistances) est ce
// qui apparie le slot d'entité du film au xuid, et rien d'autre ne le fait — le slot d'entité
// et le slot de biped sont deux espaces différents (cf. objectiveevents/slotidentity.go).
type MatchPlayerFact struct {
	// XUID en décimal, même forme que la base et que le rejeu.
	XUID string `json:"xuid"`
	// Kills, Deaths, Assists : le triplet d'appariement.
	Kills   int `json:"kills"`
	Deaths  int `json:"deaths"`
	Assists int `json:"assists"`
	// TeamID est le camp du joueur ; -1 quand la base ne le porte pas. Sert à rattacher les
	// slots d'entité d'équipe aux camps quand les scores du registre sont à égalité.
	TeamID int `json:"teamId"`
}

// MatchFacts est CE QUE LA BASE SAIT DU MATCH ET QUE LE FILM NE DIT PAS.
//
// POURQUOI CE TYPE EXISTE. `replaybuild` et `analysis/replay` n'ouvrent AUCUNE base — c'est leur
// contrat, et il est ce qui rend le constructeur d'artefact utilisable hors ligne. Les deux ponts
// qui manquent au film (l'identité des joueurs et celle des camps) arrivent donc en ENTRÉE, par
// ce type, résolue par l'appelant là où il sait le faire.
//
// Sa forme vide est LÉGITIME : un appelant sans base (le CLI unitaire) construit un artefact
// valide, seulement sans compteurs de joueur ni actions d'objectif. La dégradation est
// journalisée, jamais avalée.
//
// CE QUE COÛTE SON ABSENCE, MESURÉ (témoin 7344d24f, Strongholds, 2026-08-24) : actions
// d'objectif 246 -> 0, zones du mode 3 -> 0, joueurs de la courbe de score 8 -> 0, identité des
// camps `b` -> `unresolved`, points 1706 -> 612. Le film, lui, ne perd rien : trajectoires, tirs,
// grenades et roster sont identiques, et la vie des drapeaux traverse intacte (témoin 530820e5 :
// 30 portages, 3 captures, 41 vies d'objet des deux côtés). La dette porte sur ce que la BASE
// sait, jamais sur ce que le film sait.
type MatchFacts struct {
	// Players sont les lignes de match (`match_participants`).
	Players []MatchPlayerFact `json:"players,omitempty"`
	// TeamScores porte `team_0_score` / `team_1_score` du registre. Nil = absents.
	TeamScores *[2]int `json:"teamScores,omitempty"`
	// GameVariantName est le nom de variante du match : il donne la FAMILLE d'objectif
	// (`objectiveevents.ObjectiveTypeOf`), sans laquelle aucune action ne peut être nommée.
	GameVariantName string `json:"gameVariantName,omitempty"`
	// MapID est l'asset UGC de la carte (`match_registry.map_id`) : la SEULE clé qui joint le
	// match au catalogue versionné d'objectifs de carte, d'où sortent les socles de drapeau
	// ET les zones du mode (KOTH / Strongholds).
	//
	// PAS LE MODULE, PAS LE NOM PUBLIC, et c'est une mesure : dans `map_objectives.json`,
	// `public_name` est vide sur la quasi-totalité des entrées, et le module n'y porte pas le
	// même nom que dans le catalogue de bornes (`ridgeline` contre `cliffhanger_ridgeline`).
	// Joindre sur l'un ou l'autre ne trouve rien, SILENCIEUSEMENT. C'est déjà par map_id que le
	// service sert le calque statique des objectifs.
	//
	// Vide = artefact sans socles NI zones : la vie des drapeaux reste publiée, mais sans équipe
	// propriétaire ni état `home` (leur position serait inventée).
	MapID string `json:"mapId,omitempty"`
}

// Empty dit qu'aucun fait n'a été fourni — l'appelant n'avait pas de base, ou le match n'est pas
// au registre.
func (f MatchFacts) Empty() bool {
	return len(f.Players) == 0 && f.TeamScores == nil && f.GameVariantName == "" && f.MapID == ""
}
