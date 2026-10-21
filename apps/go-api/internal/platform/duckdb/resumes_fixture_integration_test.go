//go:build integration

package duckdb

// resumes_fixture_integration_test.go — la suite de l'aide des résumés pour les fixtures des tests
// d'intégration (deux bases d'un PlayerDB de test, bornes de fenêtre des vues filles).

import (
	"database/sql"
	"testing"
	"time"
)

// ligneJournal : UNE mort à poser au journal. Les chaînes vides valent NULL (bot, inconnu).
// La passe d'un match est unique et sa publiabilité est constante : c'est l'invariant de
// l'écrivain réel (`KillSourceBatch.Publishable`), une fixture qui le contredit échoue.
type ligneJournal struct {
	match          string
	publiable      bool
	victime        string // victim_xuid
	victimeGT      string // gamertag de la victime (défaut « Victime »)
	victimeSansNom bool   // gamertag de la victime vide (chaîne vide, colonne NOT NULL)
	tueur          string // feed_killer_xuid
	tueurGT        string
	assistant      string // assist_xuid
	assistantGT    string
	assistKnown    bool
	tag            uint32 // 0 = source non mesurée
	pctTueur       *uint8
	pctAssist      *uint8
	timeMS         int
	// passe : l'identifiant de la passe (défaut : une passe par match). ecart décale
	// l'horodatage de la passe : une passe plus récente se pose avec un écart plus grand.
	passe string
	ecart time.Duration
}

func pct(v uint8) *uint8 { return &v }

// poserJournal écrit les morts au journal (une passe par match) PUIS leurs résumés.
func poserJournal(t *testing.T, db *sql.DB, lignes ...ligneJournal) {
	t.Helper()
	poserJournalOpts(t, db, false, lignes...)
}

// poserJournalAJour pose une passe NEUVE d'un match déjà résumé et GARDE la génération de
// résumés de l'ancienne passe, comme l'écrivain réel : les vues servent la dernière.
func poserJournalAJour(t *testing.T, db *sql.DB, lignes ...ligneJournal) {
	t.Helper()
	poserJournalOpts(t, db, true, lignes...)
}

func poserJournalOpts(t *testing.T, db *sql.DB, garder bool, lignes ...ligneJournal) {
	t.Helper()
	publiable := map[string]bool{}
	matchs := []string{}
	for _, l := range lignes {
		if p, vu := publiable[l.match]; vu && p != l.publiable {
			t.Fatalf("match %s : publiabilité mixte dans une passe — impossible en production, "+
				"scinder la fixture en deux matchs", l.match)
		}
		if _, vu := publiable[l.match]; !vu {
			matchs = append(matchs, l.match)
		}
		publiable[l.match] = l.publiable
	}
	at := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	for _, l := range lignes {
		passe := l.passe
		if passe == "" {
			passe = "pass-" + l.match
		}
		victimeGT := l.victimeGT
		if victimeGT == "" && !l.victimeSansNom {
			victimeGT = "Victime"
		}
		var tag, cat any
		if l.tag != 0 {
			tag, cat = int64(l.tag), "None"
		}
		if _, err := db.Exec(`
			INSERT INTO match_kill_events
				(match_id, decode_pass, decoder_rev, written_at, publishable, time_ms,
				 victim_gamertag, victim_xuid, feed_killer_gamertag, feed_killer_xuid, feed_present,
				 assist_gamertag, assist_xuid, assist_known, source_tag, source_category,
				 killer_damage_pct, assist_damage_pct, read_path, read_origin)
			VALUES (?, ?, 'rev_test', ?, ?, ?, ?, ?, ?, ?, TRUE, ?, ?, ?, ?, ?, ?, ?, 'film-walk', 'credit-concordant')`,
			l.match, passe, at.Add(l.ecart), l.publiable, l.timeMS, victimeGT, nullSiVide(l.victime),
			nullSiVide(l.tueurGT), nullSiVide(l.tueur), nullSiVide(l.assistantGT), nullSiVide(l.assistant),
			l.assistKnown, tag, cat, l.pctTueur, l.pctAssist); err != nil {
			t.Fatalf("poserJournal %s : %v", l.match, err)
		}
	}
	ecrireResumesOpts(t, db, garder, matchs...)
}

// poserJournalSurLesDeux pose le même journal (et ses résumés) sur les deux bases d'un PlayerDB de
// test dont les fixtures historiques écrivent à la fois sur le joueur et sur le partagé
// (execOnSharedDBs). Une seule base quand elles sont la même.
func poserJournalSurLesDeux(t *testing.T, pdb *PlayerDB, lignes ...ligneJournal) {
	t.Helper()
	poserJournal(t, pdb.Player.SQLDb(), lignes...)
	if pdb.Shared != pdb.Player {
		poserJournal(t, pdb.Shared.SQLDb(), lignes...)
	}
}

// poserJournalSurLesDeuxAJour : [poserJournalAJour] sur les deux bases (passe neuve d'un match déjà résumé).
func poserJournalSurLesDeuxAJour(t *testing.T, pdb *PlayerDB, lignes ...ligneJournal) {
	t.Helper()
	poserJournalAJour(t, pdb.Player.SQLDb(), lignes...)
	if pdb.Shared != pdb.Player {
		poserJournalAJour(t, pdb.Shared.SQLDb(), lignes...)
	}
}
