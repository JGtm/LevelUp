package duckdb

// tactical_repo_coequipiers_test.go — MatesDesMorts et JournalDesKills : les deux sources du detail
// d'une zone ajoutees par le lot « coequipiers a la mort ». Memes exigences que ContextesDeMort : liste
// blanche exigee, fenetre de chaque `_latest` bornee aux matchs demandes (ADR 0036 I2), derniere passe
// entiere, table absente = ErrCapabilityNotSupported.

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	titlepkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/migration"
)

const matesParMort = 3

// tacMate pose un coequipier d'une mort (match_death_mates), sous la passe donnee.
func tacMate(t *testing.T, pdb *PlayerDB, passe, ecrit, matchID, victime string, timeMS int, mate, etat string) {
	t.Helper()
	var x, y, d any
	if etat == "visible" {
		x, y, d = 3.0, 4.0, 5.0
	}
	tacExec(t, pdb, `INSERT INTO match_death_mates
		(match_id, decode_pass, decoder_rev, written_at, victim_xuid, time_ms, mate_xuid, state, x, y, distance_m)
		VALUES (?, ?, 'rev_test', ?, ?, ?, ?, ?, ?, ?, ?)`,
		matchID, passe, ecrit, victime, timeMS, mate, etat, x, y, d)
}

// seedMates : pour chacun des `n` matchs du corpus, la mort de MOI a 2 000 ms a trois coequipiers
// (un visible, un hors de vue, un en attente) ; une autre mort (9 000 ms) en a un, que personne ne
// demande.
func seedMates(t *testing.T, pdb *PlayerDB, ids []string) {
	t.Helper()
	for _, id := range ids {
		tacMate(t, pdb, "p", "2026-08-02 10:00:00", id, tacXUIDMoi, 2000, "m1", "visible")
		tacMate(t, pdb, "p", "2026-08-02 10:00:00", id, tacXUIDMoi, 2000, "m2", "out_of_sight")
		tacMate(t, pdb, "p", "2026-08-02 10:00:00", id, tacXUIDMoi, 2000, "m3", "waiting")
		tacMate(t, pdb, "p", "2026-08-02 10:00:00", id, tacXUIDMoi, 9000, "m1", "visible")
	}
}

func TestTacticalRepo_MatesDesMorts_BorneEtCleExacte(t *testing.T) {
	b := newBaseNotee(t)
	ids := seedFenetresTactiques(t, b.pdb, 10)
	seedMates(t, b.pdb, ids)
	repo := NewTacticalRepo(b.pdb)
	q := domain.TacticalQuery{PlayerXUID: tacXUIDMoi, Matchs: domain.RestreindreAux(ids[:2])}
	morts := []domain.CleDeMort{{MatchID: ids[0], VictimXUID: tacXUIDMoi, TimeMs: 2000},
		{MatchID: ids[1], VictimXUID: tacXUIDMoi, TimeMs: 2000}}
	b.carnet.vider()

	got, err := repo.MatesDesMorts(context.Background(), q, morts)
	if err != nil {
		t.Fatalf("MatesDesMorts: %v", err)
	}
	if len(got) != 2*matesParMort {
		t.Fatalf("coequipiers = %d, want %d (les trois coequipiers de chacune des deux morts)", len(got), 2*matesParMort)
	}
	visibles := 0
	for _, m := range got {
		if m.TimeMs != 2000 || (m.MatchID != ids[0] && m.MatchID != ids[1]) {
			t.Errorf("ligne hors de la cle demandee : %+v", m)
		}
		if m.Etat == "visible" {
			visibles++
			if m.X == nil || m.Y == nil || m.DistanceM == nil || *m.DistanceM != 5 {
				t.Errorf("visible sans position : %+v", m)
			}
		} else if m.X != nil || m.Y != nil || m.DistanceM != nil {
			t.Errorf("%s avec une position : %+v (NULL reste nil, jamais zero)", m.Etat, m)
		}
	}
	if visibles != 2 {
		t.Errorf("visibles = %d, want 2", visibles)
	}
	// 4 lignes par match dans la vue (trois a 2 000 ms + une a 9 000 ms) : la fenetre ne voit que les
	// deux matchs de la liste.
	exigerFenetresBornees(t, b, "MatesDesMorts", 2*(matesParMort+1), 1)
}

func TestTacticalRepo_MatesDesMorts_ListeExigeeEtVide(t *testing.T) {
	b := newBaseNotee(t)
	ids := seedFenetresTactiques(t, b.pdb, 2)
	seedMates(t, b.pdb, ids)
	repo := NewTacticalRepo(b.pdb)
	cle := []domain.CleDeMort{{MatchID: ids[0], VictimXUID: tacXUIDMoi, TimeMs: 2000}}
	if _, err := repo.MatesDesMorts(context.Background(), domain.TacticalQuery{PlayerXUID: tacXUIDMoi}, cle); err == nil {
		t.Error("sans liste blanche : attendu un refus")
	}
	b.carnet.vider()
	got, err := repo.MatesDesMorts(context.Background(),
		domain.TacticalQuery{PlayerXUID: tacXUIDMoi, Matchs: domain.RestreindreAux(ids)}, nil)
	if err != nil || len(got) != 0 {
		t.Fatalf("aucune mort demandee : %v / %+v, want aucune ligne sans erreur", err, got)
	}
	if n := len(b.carnet.vider()); n != 0 {
		t.Errorf("aucune mort demandee : %d requete(s) envoyee(s), want 0", n)
	}
}

func TestTacticalRepo_MatesDesMorts_DernierePasseEntiere(t *testing.T) {
	b := newBaseNotee(t)
	tacMatch(t, b.pdb, "mp", tacCarteA, time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC))
	tacMate(t, b.pdb, "ancienne", "2026-08-01 10:00:00", "mp", tacXUIDMoi, 2000, "m1", "visible")
	tacMate(t, b.pdb, "ancienne", "2026-08-01 10:00:00", "mp", tacXUIDMoi, 2000, "m2", "waiting")
	tacMate(t, b.pdb, "neuve", "2026-08-02 10:00:00", "mp", tacXUIDMoi, 2000, "m1", "visible")
	got, err := NewTacticalRepo(b.pdb).MatesDesMorts(context.Background(),
		domain.TacticalQuery{PlayerXUID: tacXUIDMoi, Matchs: domain.RestreindreAux([]string{"mp"})},
		[]domain.CleDeMort{{MatchID: "mp", VictimXUID: tacXUIDMoi, TimeMs: 2000}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].MateXUID != "m1" {
		t.Fatalf("coequipiers = %+v, want la seule ligne de la passe neuve (le coequipier de l'ancienne passe ne ressuscite pas)", got)
	}
}

func TestTacticalRepo_MatesDesMorts_TableAbsente(t *testing.T) {
	pdb := pdbSansTable(t, "match_death_mates")
	q := domain.TacticalQuery{PlayerXUID: tacXUIDMoi, Matchs: domain.RestreindreAux([]string{"m1"})}
	_, err := NewTacticalRepo(pdb).MatesDesMorts(context.Background(), q,
		[]domain.CleDeMort{{MatchID: "m1", VictimXUID: tacXUIDMoi, TimeMs: 1}})
	if !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("err = %v, want ErrCapabilityNotSupported", err)
	}
}

// pdbSansTable : un shared migre dont la table du film donnee a ete retiree.
func pdbSansTable(t *testing.T, table string) *PlayerDB {
	t.Helper()
	sharedSQL, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open shared mem: %v", err)
	}
	t.Cleanup(func() { _ = sharedSQL.Close() })
	if err := migration.RunForDB(sharedSQL, migration.TargetShared); err != nil {
		t.Fatalf("migrations shared: %v", err)
	}
	if _, err := sharedSQL.Exec("DROP TABLE IF EXISTS " + table + " CASCADE"); err != nil {
		t.Fatalf("drop: %v", err)
	}
	shared := newTestDB(sharedSQL, ":memory:")
	return &PlayerDB{Shared: shared, SharedReader: LegacySharedReader(shared),
		XUID: tacXUIDMoi, TitleSlug: titlepkg.DefaultSlug}
}

func TestTacticalRepo_JournalDesKills_BorneBotsEtPositions(t *testing.T) {
	b := newBaseNotee(t)
	ids := seedFenetresTactiques(t, b.pdb, 10)
	// Un kill d'un BOT (tueur sans xuid) sans position, et un kill non publiable, dans le 1er match.
	tacKill(t, b.pdb, ids[0], "", tacXUIDMoi, 8000, true)
	tacKill(t, b.pdb, ids[0], tacXUIDAdv, tacXUIDMoi, 8500, false)
	repo := NewTacticalRepo(b.pdb)
	q := domain.TacticalQuery{PlayerXUID: tacXUIDMoi, Matchs: domain.RestreindreAux(ids[:2])}
	b.carnet.vider()

	got, err := repo.JournalDesKills(context.Background(), q)
	if err != nil {
		t.Fatalf("JournalDesKills: %v", err)
	}
	if want := 2*mortsParMatchFenetres + 1; len(got) != want {
		t.Fatalf("kills = %d, want %d (les kills publiables des deux matchs, le non publiable exclu)", len(got), want)
	}
	bots, avecPosition := 0, 0
	for _, k := range got {
		if k.MatchID != ids[0] && k.MatchID != ids[1] {
			t.Errorf("kill d'un match hors liste : %+v", k)
		}
		if k.KillerXUID == "" {
			bots++
			if k.KillerGamertag != "" || k.KillerX != nil {
				t.Errorf("tueur sans xuid : %+v", k)
			}
		}
		if k.KillerX != nil && k.KillerY != nil {
			avecPosition++
		}
	}
	if bots != 1 {
		t.Errorf("kills sans tueur identifie = %d, want 1 : le LEFT JOIN garde un kill sans position", bots)
	}
	if avecPosition != 2*mortsParMatchFenetres {
		t.Errorf("kills avec position = %d, want %d", avecPosition, 2*mortsParMatchFenetres)
	}
	exigerFenetresBornees(t, b, "JournalDesKills", 2*mortsParMatchFenetres+2, 1)
}

func TestTacticalRepo_JournalDesKills_ListeExigeeEtTableAbsente(t *testing.T) {
	b := newBaseNotee(t)
	seedFenetresTactiques(t, b.pdb, 2)
	repo := NewTacticalRepo(b.pdb)
	if _, err := repo.JournalDesKills(context.Background(), domain.TacticalQuery{PlayerXUID: tacXUIDMoi}); err == nil {
		t.Error("sans liste blanche : attendu un refus")
	}
	pdb := pdbSansTable(t, "match_kill_events")
	q := domain.TacticalQuery{PlayerXUID: tacXUIDMoi, Matchs: domain.RestreindreAux([]string{"m1"})}
	if _, err := NewTacticalRepo(pdb).JournalDesKills(context.Background(), q); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("err = %v, want ErrCapabilityNotSupported", err)
	}
}

// Le nom d'un coequipier vient des participants, a defaut des alias : sur le parc reel, la plupart des
// lignes de participants n'ont pas de gamertag et les coequipiers n'y figurent souvent pas.
func TestTacticalRepo_MatesDesMorts_NomParLesAlias(t *testing.T) {
	b := newBaseNotee(t)
	ids := seedFenetresTactiques(t, b.pdb, 1)
	tacMate(t, b.pdb, "p", "2026-08-02 10:00:00", ids[0], tacXUIDMoi, 2000, "alias-seul", "visible")
	tacMate(t, b.pdb, "p", "2026-08-02 10:00:00", ids[0], tacXUIDMoi, 2000, "inconnu", "waiting")
	tacExec(t, b.pdb, `INSERT INTO xuid_aliases (xuid, gamertag) VALUES ('alias-seul', 'Kestrel 77')`)
	got, err := NewTacticalRepo(b.pdb).MatesDesMorts(context.Background(),
		domain.TacticalQuery{PlayerXUID: tacXUIDMoi, Matchs: domain.RestreindreAux(ids)},
		[]domain.CleDeMort{{MatchID: ids[0], VictimXUID: tacXUIDMoi, TimeMs: 2000}})
	if err != nil {
		t.Fatal(err)
	}
	noms := map[string]string{}
	for _, m := range got {
		noms[m.MateXUID] = m.Gamertag
	}
	if noms["alias-seul"] != "Kestrel 77" || noms["inconnu"] != "" || len(got) != 2 {
		t.Fatalf("noms = %v : l'alias nomme le coequipier, un inconnu garde sa ligne sans nom", noms)
	}
}
