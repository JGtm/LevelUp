package port

// match_range.go — LE PORT DE LA PORTÉE DE TOUT UN LOBBY, PAR MATCH (plan
// .ai/V7.5/PLAN_AJUSTEMENTS_SUPP_PRE_V75_2026-09-21.md, lot N2 ; réserve R3 de la maquette
// MAQUETTE_TRANSVERSES_RIPOSTE_PORTEE_APPUI_2026-09-21.html §5).
//
// # POURQUOI UN CONTRAT SÉPARÉ DE `WeaponRangeRepository`
//
// Ce n'est ni le même scope ni le même sens. `WeaponRangeRepository` lit les frags D'UN
// JOUEUR des DEUX CÔTÉS (« où je frague », « où je meurs ») pour les agréger PAR ARME.
// Celui-ci lit les frags de TOUS LES JOUEURS des matchs du scope, du SEUL côté tueur, pour
// les agréger PAR (MATCH, JOUEUR) — l'arme n'y entre pas. Une méthode de plus sur le premier
// contrat aurait forcé tous ses montages (Synthèse, Explorateur, Comparaison, Timeseries) à
// implémenter une lecture dont ils n'ont pas l'usage. Même doctrine que
// `flagGrabsNetLoader` / `objectiveRoleRowsLoader` côté session.
//
// L'implémenteur, lui, est le MÊME objet (`duckdb.WeaponRangeRepo`) : la jointure « mort
// mesurée » est unique dans le dépôt (kill_measured.go, garde-rail
// kill_measured_guard_test.go) et il n'y en aura pas une seconde.
//
// # LE DÉNOMINATEUR VOYAGE AVEC LES FRAGS, ET CE N'EST PAS UN LUXE
//
// La couverture des positions décodées tourne autour de 76 % : publier des médianes sans
// dire sur combien de frags elles portent laisserait croire à l'exhaustivité. Le repo rend
// donc le nombre de frags PUBLIABLES du scope à côté des frags MESURÉS — les deux termes du
// quotient, jamais le quotient lui-même (qui se calcule là où il s'affiche).

import (
	"context"

	"levelup/go-api/internal/analysis"
)

// MatchRangeRead est le résultat brut d'une lecture de portée « tout le lobby ».
type MatchRangeRead struct {
	// Kills sont les frags MESURÉS (arme connue ET les deux positions décodées) des matchs
	// du scope, tous joueurs confondus, du côté TUEUR uniquement. Chacun porte sa clé
	// (MatchID, KillerXUID, TimeMS) : c'est par elle que l'agrégat groupe.
	Kills []analysis.MeasuredKill
	// KillsTotal est le nombre de frags PUBLIABLES des mêmes matchs — le dénominateur de
	// la couverture. Il inclut les frags dont les positions manquent, qui sont précisément
	// ceux que `Kills` ne porte pas.
	KillsTotal int
}

// MatchRangeRepository lit la portée de frag de tous les joueurs des matchs d'un scope.
//
// Capability gating identique à WeaponRangeRepository : `games.ErrCapabilityNotSupported`
// quand la table de positions est ABSENTE (titre sans décodeur de film, base non migrée).
// Une table PRÉSENTE MAIS VIDE rend zéro frag sans erreur — l'état nominal d'un scope dont
// aucun film n'a été décodé.
type MatchRangeRepository interface {
	// LoadMatchRangeKills rend les frags mesurés de tous les joueurs des matchs du scope.
	// Les filtres doivent porter `MatchIDs` et `AllPlayers` ; le repo re-valide en défense.
	LoadMatchRangeKills(ctx context.Context, slug string, filters WeaponRangeFilters) (MatchRangeRead, error)
}
