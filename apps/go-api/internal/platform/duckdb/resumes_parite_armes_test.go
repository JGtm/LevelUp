package duckdb

// resumes_parite_armes_test.go — PARITÉ 9.3 (transitoire) : les lecteurs de frags par arme
// basculés sur S1 rendent exactement ce que rendait la lecture du journal.

import (
	"fmt"
	"strings"
	"testing"

	"levelup/go-api/internal/port"
)

// enAncienneFormeArmes réécrit une requête d'armes sur S1 en sa forme d'avant la bascule.
func enAncienneFormeArmes(sqlText string) string {
	return strings.NewReplacer(
		"SUM(k.kills)::INTEGER", "COUNT(*)::INTEGER",
		"match_kill_pairs_latest", "match_kill_events_latest",
	).Replace(sqlText)
}

func TestParite93_ArmesDuMatch(t *testing.T) {
	db := corpusParite(t)
	if !strings.Contains(qMatchKillSourceWeapons, "match_kill_pairs_latest") {
		t.Fatal("qMatchKillSourceWeapons ne lit plus S1")
	}
	for _, m := range corpusPariteMatchs {
		n := lignesDe(t, db, qMatchKillSourceWeapons, m)
		a := lignesDe(t, db, ancien_qMatchKillSourceWeapons, m)
		if len(a) == 0 && len(n) == 0 {
			continue // match sans source mesurée (pC, pD ont des cas sans tag) : vide des deux côtés
		}
		exigerMemesLignes(t, "armes du match "+m, a, n)
	}
}

func TestParite93_ArmesDuJoueur(t *testing.T) {
	db := corpusParite(t)
	for _, slug := range []string{"halo_infinite", "halo_5"} {
		for _, xuid := range []string{pJ1, pJ2, pJ3} {
			for _, ids := range [][]string{nil, corpusPariteMatchs[:3], {"pE"}} {
				nouveau, args := buildPlayerKillSourceQuery(slug, xuid, ids)
				ancien := enAncienneFormeArmes(nouveau)
				a := lignesDe(t, db, ancien, args...)
				n := lignesDe(t, db, nouveau, args...)
				if len(a) == 0 && len(n) == 0 {
					continue
				}
				exigerMemesLignes(t, fmt.Sprintf("armes du joueur %s %s %d matchs", slug, xuid, len(ids)), a, n)
			}
		}
	}
}

func TestParite93_ArmesAgregeesEtCategories(t *testing.T) {
	db := corpusParite(t)
	for _, slug := range []string{"halo_infinite", "halo_5"} {
		for _, f := range []port.WeaponKillFilters{
			{MatchIDs: corpusPariteMatchs, XUIDs: []string{pJ1, pJ2, pJ3}},
			{MatchIDs: corpusPariteMatchs[:4], XUIDs: []string{pJ1}},
		} {
			nouveau, args := buildKillSourceWeaponQuery(f, slug)
			exigerMemesLignes(t, "armes agrégées "+slug, lignesDe(t, db, enAncienneFormeArmes(nouveau), args...), lignesDe(t, db, nouveau, args...))
		}
	}
}

func TestParite93_EngagesEtJournalEmprise(t *testing.T) {
	db := corpusParite(t)
	par, liste := clauseListeMatchs("k.match_id", corpusPariteMatchs)
	exigerMemesLignes(t, "frags d'engin par camp",
		lignesDe(t, db, fmt.Sprintf(ancien_qSquadVehicleFrags, par), liste),
		lignesDe(t, db, fmt.Sprintf(qSquadVehicleFrags, par), liste))
	exigerMemesLignes(t, "journal d'armes de l'Emprise",
		lignesDe(t, db, fmt.Sprintf(ancien_qSquadEmpriseJournal, par), liste),
		lignesDe(t, db, fmt.Sprintf(qSquadEmpriseJournal, par), liste))
}
