package main

// decide.go — LA DÉCISION D'ÉCRITURE, ET RIEN D'AUTRE.
//
// Ce fichier ne connaît ni le réseau, ni DuckDB, ni les flags : il prend un payload
// GetMatchStats déjà téléchargé et la ligne de registre déjà lue, et il rend un verdict.
// C'est la seule partie de l'outil dont une erreur écrirait une valeur fausse en
// production — d'où son isolement, testable sans base ni API.

import (
	"fmt"

	go_sync "levelup/go-api/internal/sync"
)

// Bornes du type de colonne. `team_0_rounds_won` / `team_1_rounds_won` / `rounds_total`
// sont des SMALLINT : au-delà, l'UPDATE échoue ou tronque. Une valeur hors bornes est donc
// REFUSÉE avant d'atteindre la base, pas après. Le plafond est celui du type ; la mesure du
// corpus (2026-08-29) plafonne à 5 manches, mais rien dans l'API ne le garantit.
const (
	roundsMin = 0
	roundsMax = 32767
)

// Verdict énumère les issues possibles pour un match. Toute autre valeur est un bug.
type Verdict string

const (
	// VerdictIdentical : la base porte déjà les valeurs de l'API. Rien à faire.
	VerdictIdentical Verdict = "identique"
	// VerdictFix : la base diverge de l'API (typiquement NULL) et l'API est écrivable.
	VerdictFix Verdict = "a_ecrire"
	// VerdictSkipNoTeams : l'API ne publie pas les deux camps 0 et 1 (FFA, payload
	// tronqué, bloc CoreStats absent). On ne devine pas un camp manquant.
	VerdictSkipNoTeams Verdict = "skip_sans_camps_0_et_1"
	// VerdictSkipImplausible : l'API publie des manches qui ne peuvent pas en être
	// (négatives, hors bornes SMALLINT, ou un total inférieur aux manches gagnées).
	VerdictSkipImplausible Verdict = "skip_valeur_invraisemblable"
)

// RegistryRounds porte les valeurs ACTUELLES de la ligne de registre. Les pointeurs nil
// distinguent « colonne NULL » de « colonne à zéro » : un NULL n'est pas un zéro, et une
// ligne NULL est exactement ce que ce backfill vient remplir.
type RegistryRounds struct {
	Team0Won *int
	Team1Won *int
	Total    *int
}

// Decision est le verdict complet, prêt à journaliser et à appliquer.
type Decision struct {
	Verdict Verdict
	// NewTeam0Won / NewTeam1Won / NewTotal ne sont significatifs que pour VerdictFix et
	// VerdictIdentical.
	NewTeam0Won int
	NewTeam1Won int
	NewTotal    int
	// Old porte l'avant, tel qu'il est en base (nil = NULL).
	Old RegistryRounds
	// Reason est la phrase de journal. Toujours renseignée.
	Reason string
}

// Writable dit si la décision autorise une écriture. Un seul endroit décide, pour qu'un
// appelant ne puisse pas écrire sur un skip par inattention.
func (d Decision) Writable() bool { return d.Verdict == VerdictFix }

// Decide applique la règle d'écriture à un payload GetMatchStats.
//
// L'extraction n'est PAS réimplémentée ici : elle délègue à
// `sync.ExtractTeamRoundsByID`, la même fonction que la sync de production. C'est
// volontaire et non négociable — deux lectures des manches re-divergeraient, et c'est
// précisément la divergence que ce genre d'outil est censé fermer.
func Decide(matchJSON map[string]any, cur RegistryRounds) Decision {
	w0, w1, total := go_sync.ExtractTeamRoundsByID(matchJSON)
	if w0 == nil || w1 == nil || total == nil {
		return Decision{
			Verdict: VerdictSkipNoTeams,
			Old:     cur,
			Reason:  "l'API ne publie pas les manches des deux camps TeamId 0 et 1 (FFA ou payload partiel)",
		}
	}
	t0, t1, tot := *w0, *w1, *total
	if bad, why := implausible(t0, t1, tot); bad {
		return Decision{Verdict: VerdictSkipImplausible, Old: cur, Reason: why}
	}
	if cur.Team0Won != nil && cur.Team1Won != nil && cur.Total != nil &&
		*cur.Team0Won == t0 && *cur.Team1Won == t1 && *cur.Total == tot {
		return Decision{
			Verdict: VerdictIdentical, NewTeam0Won: t0, NewTeam1Won: t1, NewTotal: tot,
			Old: cur, Reason: "la base porte déjà les manches de l'API",
		}
	}
	return Decision{
		Verdict: VerdictFix, NewTeam0Won: t0, NewTeam1Won: t1, NewTotal: tot, Old: cur,
		Reason: fmt.Sprintf("manches de l'API %d-%d sur %d manches jouées", t0, t1, tot),
	}
}

// implausible refuse ce qui ne peut pas être un compte de manches.
//
// Le contrôle « total >= manches gagnées cumulées » n'est pas décoratif : le total est le
// MAX des deux camps (cf. ExtractTeamRoundsByID), donc un total inférieur à rw0+rw1
// signalerait un payload incohérent — et écrirait un couple qui ferait mentir la règle
// d'affichage (deux camps gagnants de plus de manches qu'il n'y en a eu).
func implausible(t0, t1, total int) (bool, string) {
	for _, v := range []int{t0, t1, total} {
		if v < roundsMin || v > roundsMax {
			return true, fmt.Sprintf("manches hors bornes SMALLINT (%d-%d sur %d)", t0, t1, total)
		}
	}
	if total < t0+t1 {
		return true, fmt.Sprintf("total de manches incohérent : %d joué(es) pour %d+%d gagnées", total, t0, t1)
	}
	return false, ""
}

// formatRounds rend l'avant lisible dans le journal, NULL compris.
func formatRounds(r RegistryRounds) string {
	return fmt.Sprintf("%s-%s sur %s", nullable(r.Team0Won), nullable(r.Team1Won), nullable(r.Total))
}

func nullable(p *int) string {
	if p == nil {
		return "NULL"
	}
	return fmt.Sprintf("%d", *p)
}
