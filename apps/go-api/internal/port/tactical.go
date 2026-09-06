// Package port — tactical.go : les contrats de l'onglet Tactique, lecteur ET
// service.
//
// Fichier separe de repository_data.go et de services.go, qui depassent DEJA le
// seuil de 500 lignes : ce paquet range ses contrats par sujet (achievements.go,
// medals.go, replay_availability.go, ...), et grossir un god-file au motif que le
// voisin de famille y vit accroitrait une dette gelee par la baseline
// (CLAUDE.md n 5).
package port

import (
	"context"

	"levelup/go-api/internal/domain"
)

// TacticalService sert les deux lectures de l'onglet Tactique : la grille des
// cartes jouees, et la lecture de placement d'une carte.
//
// Capability-gated : `Raster` retourne games.ErrCapabilityNotSupported quand le
// titre ne mesure pas les positions de kill (le handler degrade en 503).
// `MapsPlayed`, lui, ne lit que le registre des matchs — il n'a rien a gater.
//
// LE PERIMETRE ARRIVE RESOLU (phase 4 bis, 2026-09-06) : le client fait resoudre sa
// selection par le endpoint de filtres (service.FilteredMatchIDs, base JOUEUR) et passe
// des match_id en LISTE BLANCHE. Ce service ne filtre plus rien lui-meme — une seconde
// definition du perimetre donnerait deux comptes de matchs pour la meme question, et
// c'est elle qui laissait les sessions sans effet sur cet onglet.
//
// Refus typés, traduits en statut par le handler :
//   - domain.ErrTacticalCarteInconnue             -> 404 ;
//   - domain.ErrTacticalQuestionInconnue          -> 400 ;
//   - domain.ErrTacticalQuiInconnu                -> 400 ;
//   - domain.ErrTacticalEscouadeSansComposition   -> 400.
type TacticalService interface {
	// MapsPlayed rend les cartes jouees dans le perimetre, triees par nombre de
	// matchs decroissant, chacune portant son verdict de lisibilite.
	MapsPlayed(ctx context.Context, scope domain.TacticalScope) (domain.TacticalMapsPage, error)

	// Raster rend la lecture de placement d'une carte pour une question
	// (domain.TacticalQuestion*) et un axe (domain.TacticalQui*).
	Raster(ctx context.Context, req domain.TacticalRasterRequest) (domain.TacticalRaster, error)
}

// TacticalRepository — onglet Tactique (plan .ai/PLAN_TACTIQUE_2026-09-06.md,
// phase 2) : les trois lectures de base d'une analyse de placement par CARTE.
// Implémenté par platform/duckdb.TacticalRepo.
//
// Source : shared.match_registry × shared.match_participants pour l'UNIVERS
// (les matchs retenus par le filtre) ; shared.kill_positions_latest ×
// shared.match_kill_events_latest pour les positions et le journal des morts —
// vues `_latest` UNIQUEMENT (règle ART n°2, jamais la table brute).
//
// L'UNIVERS VOYAGE AVEC LES POINTS. KillPositions et KillEvents rendent l'un ET
// l'autre, parce qu'un match retenu SANS point (aucune mort mesurée) doit compter
// au dénominateur « par match » de la lecture. Le déduire des points l'effacerait
// — défaut mesuré et corrigé en phase 1 du plan.
//
// Capability gating : retourne games.ErrCapabilityNotSupported si les tables du
// film sont absentes (titre/schéma sans décodeur). Zéro ligne est l'état NOMINAL
// d'un joueur dont aucun match n'est encore décodé — pas une panne.
// LE PERIMETRE EST UNE LISTE BLANCHE (`domain.TacticalQuery.Matchs`), eventuellement
// resserrée par une COMPOSITION (`Coequipiers`) : les trois lectures l'appliquent au
// MEME endroit — le SELECT de l'univers — pour qu'aucune ne puisse mesurer sur une
// population que les autres ne voient pas.
type TacticalRepository interface {
	// MapsPlayed liste les cartes jouées par le joueur dans le périmètre, avec le
	// nombre de matchs et sa décomposition victoires / défaites. Ordre
	// déterministe (matchs décroissants, puis map_id). `filtre.MapID` est ignoré :
	// cette lecture est l'écran d'entrée, elle porte sur toutes les cartes.
	MapsPlayed(ctx context.Context, filtre domain.TacticalQuery) ([]domain.TacticalMapRow, error)

	// KillPositions rend l'univers des matchs retenus ET les positions mesurées
	// (tueur ET victime connus) de ces matchs. TOUS les joueurs du match sont
	// rendus : l'axe « qui » (moi / escouade / adversaires) se tranche dans le
	// service, à partir des équipes de l'univers — une requête par axe multiplierait
	// les scans de la même fenêtre.
	KillPositions(ctx context.Context, filtre domain.TacticalQuery) (domain.TacticalPositions, error)

	// KillEvents rend l'univers des matchs retenus ET le journal de leurs morts,
	// sous la forme que analysis/coordination consomme (victime, tueur crédité,
	// instant). Même portée que KillPositions : tous les joueurs.
	//
	// `filtre.MapID` est OPTIONNEL ici (vide = toutes les cartes) : la page
	// Escouade mesure l'échange d'une COMPOSITION, qui n'a pas de carte, et
	// resserre le périmètre sur ses matchs partagés côté service. Une seconde
	// requête pour ce seul cas aurait donné deux définitions du journal des
	// morts d'un joueur.
	KillEvents(ctx context.Context, filtre domain.TacticalQuery) (domain.TacticalKillEvents, error)
}
