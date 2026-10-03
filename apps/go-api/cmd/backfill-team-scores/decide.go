package main

// decide.go — LA DÉCISION DE CORRECTION, ET RIEN D'AUTRE.
//
// Ce fichier ne connaît ni le réseau, ni DuckDB, ni les flags : il prend un payload
// GetMatchStats déjà téléchargé et la ligne de registre déjà lue, et il rend un verdict.
// C'est ce découplage qui rend la règle testable sans base ni API — et c'est la seule
// partie de l'outil dont une erreur écrirait une valeur fausse en production.

import (
	"fmt"

	go_sync "levelup/go-api/internal/sync"
)

// Bornes du type de colonne. `match_registry.team_0_score` / `team_1_score` sont des
// SMALLINT (`internal/sync/schema.go:181-182`) : au-delà, l'UPDATE échoue ou tronque.
// Une valeur hors bornes est donc REFUSÉE avant d'atteindre la base, pas après.
const (
	scoreMin = 0
	scoreMax = 32767
)

// Verdict énumère les issues possibles pour un match. Toute autre valeur est un bug.
type Verdict string

const (
	// VerdictIdentical : la base porte déjà la valeur de l'API. Rien à faire.
	VerdictIdentical Verdict = "identique"
	// VerdictFix : la base diverge de l'API et la valeur de l'API est écrivable.
	VerdictFix Verdict = "a_corriger"
	// VerdictSkipNoTeams : l'API ne publie pas les deux camps 0 et 1 (FFA, payload
	// tronqué). On ne devine pas un camp manquant — on saute et on le dit.
	VerdictSkipNoTeams Verdict = "skip_sans_camps_0_et_1"
	// VerdictSkipImplausible : l'API publie une valeur qui ne peut pas être un score
	// (négative, ou hors bornes du SMALLINT de la colonne).
	VerdictSkipImplausible Verdict = "skip_valeur_invraisemblable"
)

// RegistryScores porte les valeurs ACTUELLES de la ligne de registre. Les pointeurs nil
// distinguent « colonne NULL » de « colonne à zéro » : un NULL n'est pas un zéro, et une
// ligne NULL est bien une ligne à corriger.
type RegistryScores struct {
	Team0 *int
	Team1 *int
}

// Decision est le verdict complet, prêt à logguer et à appliquer.
type Decision struct {
	Verdict Verdict
	// NewTeam0 / NewTeam1 ne sont significatifs que pour VerdictFix et VerdictIdentical.
	NewTeam0 int
	NewTeam1 int
	// Old porte l'avant, tel qu'il est en base (nil = NULL).
	Old RegistryScores
	// Reason est la phrase de journal. Toujours renseignée.
	Reason string
}

// Writable dit si la décision autorise une écriture. Un seul endroit décide, pour qu'un
// appelant ne puisse pas écrire sur un skip par inattention.
func (d Decision) Writable() bool { return d.Verdict == VerdictFix }

// Decide applique la règle de correction à un payload GetMatchStats.
//
// L'extraction n'est PAS réimplémentée ici : elle délègue à
// `sync.ExtractTeamScoresByID`, la même fonction que la sync de production. C'est
// volontaire et non négociable — deux lectures du score d'équipe re-divergeraient, et
// c'est précisément la divergence que ce backfill répare.
func Decide(matchJSON map[string]any, cur RegistryScores) Decision {
	p0, p1 := go_sync.ExtractTeamScoresByID(matchJSON)
	if p0 == nil || p1 == nil {
		return Decision{
			Verdict: VerdictSkipNoTeams,
			Old:     cur,
			Reason:  "l'API ne publie pas les deux camps TeamId 0 et 1 (FFA ou payload partiel)",
		}
	}
	t0, t1 := *p0, *p1
	if bad, why := implausible(t0, t1); bad {
		return Decision{
			Verdict: VerdictSkipImplausible,
			Old:     cur,
			Reason:  why,
		}
	}
	if cur.Team0 != nil && cur.Team1 != nil && *cur.Team0 == t0 && *cur.Team1 == t1 {
		return Decision{
			Verdict:  VerdictIdentical,
			NewTeam0: t0, NewTeam1: t1,
			Old:    cur,
			Reason: "la base porte déjà la valeur de l'API",
		}
	}
	return Decision{
		Verdict:  VerdictFix,
		NewTeam0: t0, NewTeam1: t1,
		Old:    cur,
		Reason: fmt.Sprintf("base %s -> API %d/%d", formatScores(cur), t0, t1),
	}
}

// implausible refuse ce qui ne peut pas être un score d'équipe. La garde est en amont de
// la base : une valeur hors bornes ne doit pas atteindre l'UPDATE, même pour y échouer.
func implausible(t0, t1 int) (bool, string) {
	for _, v := range [2]int{t0, t1} {
		if v < scoreMin {
			return true, fmt.Sprintf("score négatif refusé (%d/%d)", t0, t1)
		}
		if v > scoreMax {
			return true, fmt.Sprintf("score hors bornes SMALLINT refusé (%d/%d, max %d)", t0, t1, scoreMax)
		}
	}
	return false, ""
}

// formatScores rend la ligne de registre lisible, en distinguant NULL de zéro.
func formatScores(s RegistryScores) string {
	return fmt.Sprintf("%s/%s", formatScore(s.Team0), formatScore(s.Team1))
}

func formatScore(p *int) string {
	if p == nil {
		return "NULL"
	}
	return fmt.Sprintf("%d", *p)
}
