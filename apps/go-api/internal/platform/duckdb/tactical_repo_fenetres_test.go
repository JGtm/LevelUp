package duckdb

// tactical_repo_fenetres_test.go — LE PERIMETRE DES LECTURES TACTIQUES EST PAYE AU PERIMETRE
// (lot L5a du plan perf, 2026-09-23).
//
// Ce que ces tests verrouillent, et pourquoi chacun peut echouer :
//
//  1. aucune fenetre `_latest` ne voit plus que les lignes du perimetre, pour les quatre
//     lectures (univers, journal des morts, positions, morts en contexte). Le defaut que
//     le lot corrige ne changeait AUCUN chiffre : l'univers re-selectionne en sous-requete
//     et le EXISTS sans liste laissaient la fenetre du journal se calculer sur toute la
//     table. Seul le nombre de lignes vues par la fenetre le montre (cf.
//     fenetres_perimetre_helpers_test.go) ;
//  2. la liste recopiee dans le EXISTS de l'univers ne change pas le drapeau `mesure` ;
//  3. la lecture d'isolement, qui porte desormais la liste sur ses TROIS vues, garde ses
//     gardes (contexte exige, double kill au meme instant ecarte, voisinage NULL servi nil).

import (
	"context"
	"fmt"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// mortsParMatchFenetres : les morts semees dans chaque match du corpus des fenetres.
const mortsParMatchFenetres = 3

// tacContexte pose le voisinage d'une mort (match_death_context). `proche` nil = aucun
// coequipier visible.
func tacContexte(t *testing.T, pdb *PlayerDB, matchID, victimXUID string, timeMS int, proche any) {
	t.Helper()
	tacExec(t, pdb, `INSERT INTO match_death_context
		(match_id, decode_pass, decoder_rev, victim_xuid, time_ms, nearest_teammate_m,
		 teammates_visible, teammates_waiting, teammates_out_of_sight, teammates_left, teammates_total)
		VALUES (?, 'pass_test', 'rev_test', ?, ?, ?, 1, 0, 2, 0, 3)`,
		matchID, victimXUID, timeMS, proche)
}

// seedFenetresTactiques monte `n` matchs de MOI sur la carte A, chacun avec
// mortsParMatchFenetres morts publiables, leurs positions et leur contexte.
func seedFenetresTactiques(t *testing.T, pdb *PlayerDB, n int) []string {
	t.Helper()
	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	ids := make([]string, 0, n)
	for i := 1; i <= n; i++ {
		id := fmt.Sprintf("f%02d", i)
		ids = append(ids, id)
		tacMatch(t, pdb, id, tacCarteA, base.Add(time.Duration(i)*time.Hour))
		tacParticipant(t, pdb, id, tacXUIDMoi, 0, domain.OutcomeWin)
		tacParticipant(t, pdb, id, tacXUIDAdv, 1, domain.OutcomeLoss)
		for k := 0; k < mortsParMatchFenetres; k++ {
			ts := 1000 * (k + 1)
			tueur, victime := tacXUIDMoi, tacXUIDAdv
			if k%2 == 1 {
				tueur, victime = tacXUIDAdv, tacXUIDMoi
			}
			tacKill(t, pdb, id, tueur, victime, ts, true)
			tacPos(t, pdb, id, tueur, ts, 1.0, 1.0, 2.0, 2.0)
			tacContexte(t, pdb, id, victime, ts, 4.0)
		}
	}
	return ids
}

// TestTacticalRepo_PerimetreRestreint_FenetresBornees : DEUX matchs demandes sur DIX ; chaque
// fenetre de chaque requete des quatre lectures ne doit voir que les lignes de ces deux-la.
func TestTacticalRepo_PerimetreRestreint_FenetresBornees(t *testing.T) {
	b := newBaseNotee(t)
	ids := seedFenetresTactiques(t, b.pdb, 10)
	repo := NewTacticalRepo(b.pdb)
	ctx := context.Background()
	q := domain.TacticalQuery{PlayerXUID: tacXUIDMoi, Matchs: domain.RestreindreAux(ids[:2])}
	qCarte := q
	qCarte.MapID = tacCarteA
	// Les lignes des deux matchs du perimetre, dans chacune des vues lues.
	borne := 2 * mortsParMatchFenetres
	b.carnet.vider()

	ev, err := repo.KillEvents(ctx, q)
	if err != nil {
		t.Fatalf("KillEvents: %v", err)
	}
	if len(ev.Univers.Matchs) != 2 || len(ev.Events) != borne {
		t.Fatalf("KillEvents = %d matchs / %d morts, want 2 / %d", len(ev.Univers.Matchs), len(ev.Events), borne)
	}
	// Univers (EXISTS sur le journal) + journal des morts.
	exigerFenetresBornees(t, b, "KillEvents", borne, 2)

	morts, err := repo.MortsAvecContexte(ctx, qCarte)
	if err != nil {
		t.Fatalf("MortsAvecContexte: %v", err)
	}
	if len(morts.Morts) != borne {
		t.Fatalf("MortsAvecContexte = %d morts, want %d", len(morts.Morts), borne)
	}
	exigerFenetresBornees(t, b, "MortsAvecContexte", borne, 2)

	pos, err := repo.KillPositions(ctx, qCarte)
	if err != nil {
		t.Fatalf("KillPositions: %v", err)
	}
	if len(pos.Points) != borne {
		t.Fatalf("KillPositions = %d points, want %d", len(pos.Points), borne)
	}
	exigerFenetresBornees(t, b, "KillPositions", borne, 2)

	univ, err := repo.Univers(ctx, q)
	if err != nil {
		t.Fatalf("Univers: %v", err)
	}
	if len(univ.Matchs) != 2 {
		t.Fatalf("Univers = %d matchs, want 2", len(univ.Matchs))
	}
	exigerFenetresBornees(t, b, "Univers", borne, 1)
}

// TestTacticalRepo_ListeBlanche_MesureInchangee : la liste recopiee dans le EXISTS ne peut
// rien retirer que le WHERE externe n'ait deja retire — m1 reste MESURE, m2 (aucune mort)
// reste MUET, m3 (autre carte, mais demande sans carte) reste mesure.
func TestTacticalRepo_ListeBlanche_MesureInchangee(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	seedTacticalCorpus(t, pdb)

	q := domain.TacticalQuery{PlayerXUID: tacXUIDMoi, Matchs: domain.RestreindreAux([]string{"m1", "m2", "m3"})}
	got, err := NewTacticalRepo(pdb).KillEvents(context.Background(), q)
	if err != nil {
		t.Fatalf("KillEvents: %v", err)
	}
	want := map[string]bool{"m1": true, "m2": false, "m3": true}
	if len(got.Univers.Matchs) != len(want) {
		t.Fatalf("univers = %v, want m1, m2, m3", matchIDs(got.Univers.Matchs))
	}
	for _, m := range got.Univers.Matchs {
		if m.Mesure != want[m.MatchID] {
			t.Errorf("%s : mesure = %v, want %v", m.MatchID, m.Mesure, want[m.MatchID])
		}
	}
	if len(got.Events) != 3 {
		t.Errorf("evenements = %d, want 3 (deux de m1, un de m3) : %+v", len(got.Events), got.Events)
	}
}

// TestTacticalRepo_MortsAvecContexte_GardesSurLePerimetre : la lecture d'isolement, restreinte
// a un match, garde ses trois gardes — une mort sans contexte sort, le double kill au meme
// instant sort en entier, un voisinage NULL est servi nil (jamais zero).
func TestTacticalRepo_MortsAvecContexte_GardesSurLePerimetre(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	for _, id := range []string{"i1", "i2"} {
		tacMatch(t, pdb, id, tacCarteA, base)
		tacParticipant(t, pdb, id, tacXUIDMoi, 0, domain.OutcomeWin)
		tacParticipant(t, pdb, id, tacXUIDAdv, 1, domain.OutcomeLoss)
	}
	// i1, t=1000 : ma mort, contexte SANS coequipier visible (NULL).
	tacKill(t, pdb, "i1", tacXUIDAdv, tacXUIDMoi, 1000, true)
	tacPos(t, pdb, "i1", tacXUIDAdv, 1000, 1.0, 1.0, 2.0, 2.0)
	tacContexte(t, pdb, "i1", tacXUIDMoi, 1000, nil)
	// i1, t=2000 : mort de l'adversaire SANS contexte — elle sort.
	tacKill(t, pdb, "i1", tacXUIDMoi, tacXUIDAdv, 2000, true)
	tacPos(t, pdb, "i1", tacXUIDMoi, 2000, 3.0, 3.0, 4.0, 4.0)
	// i1, t=3000 : DOUBLE KILL au meme instant (deux victimes, une ligne de positions).
	tacKill(t, pdb, "i1", tacXUIDMoi, tacXUIDAdv, 3000, true)
	tacKill(t, pdb, "i1", tacXUIDMoi, tacXUIDTier, 3000, true)
	tacPos(t, pdb, "i1", tacXUIDMoi, 3000, 5.0, 5.0, 6.0, 6.0)
	tacContexte(t, pdb, "i1", tacXUIDAdv, 3000, 7.5)
	tacContexte(t, pdb, "i1", tacXUIDTier, 3000, 7.5)
	// i2 : hors perimetre — sa mort ne doit apparaitre nulle part.
	tacKill(t, pdb, "i2", tacXUIDAdv, tacXUIDMoi, 1000, true)
	tacPos(t, pdb, "i2", tacXUIDAdv, 1000, 9.0, 9.0, 9.5, 9.5)
	tacContexte(t, pdb, "i2", tacXUIDMoi, 1000, 1.0)

	q := domain.TacticalQuery{PlayerXUID: tacXUIDMoi, MapID: tacCarteA, Matchs: domain.RestreindreAux([]string{"i1"})}
	got, err := NewTacticalRepo(pdb).MortsAvecContexte(context.Background(), q)
	if err != nil {
		t.Fatalf("MortsAvecContexte: %v", err)
	}
	if len(got.Morts) != 1 {
		t.Fatalf("morts = %+v, want la seule mort de i1 a t=1000", got.Morts)
	}
	m := got.Morts[0]
	if m.MatchID != "i1" || m.TimeMs != 1000 || m.VictimXUID != tacXUIDMoi {
		t.Errorf("mort servie = %+v, want i1 / t=1000 / victime moi", m)
	}
	if m.PlusProcheM != nil {
		t.Errorf("voisinage = %v, want nil (aucun coequipier visible, jamais zero)", *m.PlusProcheM)
	}
	if m.X != 2.0 || m.Y != 2.0 {
		t.Errorf("position = (%v, %v), want la position de la VICTIME (2, 2)", m.X, m.Y)
	}
}
