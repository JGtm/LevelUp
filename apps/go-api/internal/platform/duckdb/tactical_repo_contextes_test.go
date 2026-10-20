package duckdb

// tactical_repo_contextes_test.go — ContextesDeMort : le voisinage de chaque mort des matchs de la
// liste blanche, dernière passe entière, borné à la liste sous la fenêtre `_latest` (ADR 0036 I2).

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
	titlepkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/migration"
)

// TestTacticalRepo_ContextesDeMort_BorneEtNull : deux matchs demandés sur dix ; seules leurs morts
// sont rendues, la fenêtre de la vue ne voit qu'elles, une distance NULL reste nil.
func TestTacticalRepo_ContextesDeMort_BorneEtNull(t *testing.T) {
	b := newBaseNotee(t)
	ids := seedFenetresTactiques(t, b.pdb, 10)
	// Une mort sans coéquipier visible dans le premier match du périmètre.
	tacContexte(t, b.pdb, ids[0], tacXUIDMoi, 9000, nil)
	repo := NewTacticalRepo(b.pdb)
	q := domain.TacticalQuery{PlayerXUID: tacXUIDMoi, Matchs: domain.RestreindreAux(ids[:2])}
	b.carnet.vider()

	got, err := repo.ContextesDeMort(context.Background(), q)
	if err != nil {
		t.Fatalf("ContextesDeMort: %v", err)
	}
	borne := 2*mortsParMatchFenetres + 1
	if len(got) != borne {
		t.Fatalf("contextes = %d, want %d (les morts des deux matchs demandés)", len(got), borne)
	}
	nuls := 0
	for _, c := range got {
		if c.MatchID != ids[0] && c.MatchID != ids[1] {
			t.Errorf("contexte d'un match hors liste : %+v", c)
		}
		if c.PlusProcheM == nil {
			nuls++
			if c.TimeMs != 9000 || c.VictimXUID != tacXUIDMoi {
				t.Errorf("distance nil sur une autre mort : %+v", c)
			}
		} else if *c.PlusProcheM != 4.0 {
			t.Errorf("distance = %v, want 4", *c.PlusProcheM)
		}
		if c.Visibles != 1 || c.HorsDeVue != 2 {
			t.Errorf("coéquipiers = %d visibles / %d hors de vue, want 1 / 2", c.Visibles, c.HorsDeVue)
		}
	}
	if nuls != 1 {
		t.Errorf("distances nil = %d, want 1 (jamais un zéro)", nuls)
	}
	exigerFenetresBornees(t, b, "ContextesDeMort", borne, 1)
}

// TestTacticalRepo_ContextesDeMort_ListeExigee : une liste non posée est REFUSÉE (jamais l'historique
// entier) ; une liste vide ne lance aucune requête.
func TestTacticalRepo_ContextesDeMort_ListeExigee(t *testing.T) {
	b := newBaseNotee(t)
	seedFenetresTactiques(t, b.pdb, 2)
	repo := NewTacticalRepo(b.pdb)
	if _, err := repo.ContextesDeMort(context.Background(), domain.TacticalQuery{PlayerXUID: tacXUIDMoi}); err == nil {
		t.Error("sans liste blanche : attendu un refus")
	}
	b.carnet.vider()
	got, err := repo.ContextesDeMort(context.Background(),
		domain.TacticalQuery{PlayerXUID: tacXUIDMoi, Matchs: domain.RestreindreAux(nil)})
	if err != nil || len(got) != 0 {
		t.Fatalf("liste vide : %v / %+v, want aucune ligne sans erreur", err, got)
	}
	if n := len(b.carnet.vider()); n != 0 {
		t.Errorf("liste vide : %d requête(s) envoyée(s), want 0", n)
	}
}

// TestTacticalRepo_ContextesDeMort_TableAbsente : un shared sans la table du film rend
// ErrCapabilityNotSupported.
func TestTacticalRepo_ContextesDeMort_TableAbsente(t *testing.T) {
	sharedSQL, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open shared mem: %v", err)
	}
	t.Cleanup(func() { _ = sharedSQL.Close() })
	if err := migration.RunForDB(sharedSQL, migration.TargetShared); err != nil {
		t.Fatalf("migrations shared: %v", err)
	}
	if _, err := sharedSQL.Exec("DROP TABLE IF EXISTS match_death_context CASCADE"); err != nil {
		t.Fatalf("drop: %v", err)
	}
	shared := newTestDB(sharedSQL, ":memory:")
	pdb := &PlayerDB{Shared: shared, SharedReader: LegacySharedReader(shared),
		XUID: tacXUIDMoi, TitleSlug: titlepkg.DefaultSlug}
	q := domain.TacticalQuery{PlayerXUID: tacXUIDMoi, Matchs: domain.RestreindreAux([]string{"m1"})}
	if _, err := NewTacticalRepo(pdb).ContextesDeMort(context.Background(), q); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("err = %v, want ErrCapabilityNotSupported", err)
	}
}

// TestTacticalRepo_ContextesDeMort_DernierePasseEntiere : deux passes d'un même match, sur des morts
// DIFFÉRENTES — seule la passe la plus récente sort, en entier ; la mort que seule l'ancienne passe
// portait n'est pas ressuscitée.
func TestTacticalRepo_ContextesDeMort_DernierePasseEntiere(t *testing.T) {
	b := newBaseNotee(t)
	const ins = `INSERT INTO match_death_context
		(match_id, decode_pass, decoder_rev, written_at, victim_xuid, time_ms, nearest_teammate_m,
		 teammates_visible, teammates_waiting, teammates_out_of_sight, teammates_left, teammates_total)
		VALUES ('mp', ?, 'rev', ?, ?, ?, 4.0, 1, 0, 2, 0, 3)`
	tacExec(t, b.pdb, ins, "ancienne", "2026-08-01 10:00:00", tacXUIDMoi, 5000)
	tacExec(t, b.pdb, ins, "neuve", "2026-08-02 10:00:00", tacXUIDMoi, 7000)
	tacExec(t, b.pdb, ins, "neuve", "2026-08-02 10:00:00", tacXUIDMoi, 8000)

	got, err := NewTacticalRepo(b.pdb).ContextesDeMort(context.Background(),
		domain.TacticalQuery{PlayerXUID: tacXUIDMoi, Matchs: domain.RestreindreAux([]string{"mp"})})
	if err != nil {
		t.Fatalf("ContextesDeMort: %v", err)
	}
	if len(got) != 2 || got[0].TimeMs != 7000 || got[1].TimeMs != 8000 {
		t.Fatalf("contextes = %+v, want les deux morts de la passe neuve (7000, 8000)", got)
	}
}
