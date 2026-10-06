package duckdb

// solo_lives_repo_test.go — LES VIES D'UN JOUEUR SE LISENT BORNÉES, SUR LES VUES `_latest` (plan
// `.ai/PLAN_TIMESERIES_USAGES_EMPRISE_2026-10-05.md`, lot L3.3 ; ADR 0036 I2).
//
// Base `:memory:` migrée (les vraies migrations shared, donc les vraies vues). Ce que chaque test
// verrouille :
//   - seule la DERNIÈRE PASSE ENTIÈRE de chaque match est rendue (une passe plus ancienne, plus
//     longue, ne laisse survivre aucune vie) ;
//   - seuls les matchs ET le joueur demandés ; la fenêtre de CHAQUE vue ne voit que les lignes des
//     matchs demandés (EXPLAIN ANALYZE, exigerFenetresBornees) ;
//   - une distance NULL reste nil ; les frags d'une passe non publiable et ceux d'un autre tueur
//     sortent, et le match au journal non publiable est nommé ; les camps viennent de
//     `match_participants` (NULL pour une victime sans ligne) ;
//   - table absente : games.ErrCapabilityNotSupported.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	duckdbdrv "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/games"
)

// poserVieDuFilm : une ligne de `match_lives`.
func poserVieDuFilm(t *testing.T, pdb *PlayerDB, match, pass, written, xuid string, start int64, cause string) {
	t.Helper()
	tacExec(t, pdb, `INSERT INTO match_lives
		(match_id, decode_pass, decoder_rev, written_at, xuid, start_ms, end_ms, end_cause, named_by)
		VALUES (?, ?, 'vies-test', CAST(? AS TIMESTAMP), ?, ?, ?, ?, 'death')`,
		match, pass, written, xuid, start, start+20000, cause)
}

// corpusVies : dix matchs ; dans chacun, P (camp 0) et O (camp 1) ont deux vies, chacun une mort
// au contexte, et P deux frags (sur O, sur son coéquipier A) ; O tue P. Le drapeau `publishable`
// vaut pour une PASSE ENTIÈRE (jamais mêlé dans une passe) : la passe de `m9` n'est pas publiable,
// ses vies et ses morts au contexte existent quand même (portes indépendantes).
// `m1` porte en plus une passe ANCIENNE à trois vies pour P.
func corpusVies(t *testing.T, pdb *PlayerDB) {
	t.Helper()
	for i := 0; i < 10; i++ {
		m := fmt.Sprintf("m%d", i)
		poserVariante(t, pdb, m, fmt.Sprintf("Variante %d", i))
		tacParticipant(t, pdb, m, "P", 0, 2)
		tacParticipant(t, pdb, m, "A", 0, 2)
		tacParticipant(t, pdb, m, "O", 1, 3)
		for _, x := range []string{"P", "O"} {
			poserVieDuFilm(t, pdb, m, "p2", "2026-09-29 10:00:00", x, 1000, "death")
			poserVieDuFilm(t, pdb, m, "p2", "2026-09-29 10:00:00", x, 60000, "film_end")
		}
		tacContexte(t, pdb, m, "P", 40000, 12.5)
		tacContexte(t, pdb, m, "O", 30000, nil)
		publiable := m != "m9"
		tacKill(t, pdb, m, "P", "O", 30000, publiable)
		tacKill(t, pdb, m, "P", "A", 35000, publiable)
		tacKill(t, pdb, m, "O", "P", 40000, publiable)
	}
	for _, start := range []int64{1000, 20000, 60000} {
		poserVieDuFilm(t, pdb, "m1", "p1", "2026-09-28 10:00:00", "P", start, "cut")
	}
}

func TestSoloLivesRepo_BorneEtDernierePasse(t *testing.T) {
	b := newBaseNotee(t)
	corpusVies(t, b.pdb)
	b.carnet.vider()

	got, err := NewSoloLivesRepo(b.pdb).LoadLivesNearTeammate(context.Background(), []string{"m1", "m2"}, "P")
	if err != nil {
		t.Fatalf("LoadLivesNearTeammate : %v", err)
	}
	// Fenêtres : au plus les lignes des deux matchs demandés, toutes passes et tous joueurs
	// confondus (vies : m1 4 + 3 anciennes, m2 4 = 11) ; quatre lectures de vues `_latest`
	// (vies, morts, frags, publiabilité du journal).
	exigerFenetresBornees(t, b, "LoadLivesNearTeammate", 11, 4)

	var vies []string
	for _, v := range got.Vies {
		vies = append(vies, fmt.Sprintf("%s/%d/%s", v.MatchID, v.StartMS, v.EndCause))
	}
	if want := "[m1/1000/death m1/60000/film_end m2/1000/death m2/60000/film_end]"; fmt.Sprint(vies) != want {
		t.Fatalf("vies lues = %v\nattendu     %s (dernière passe, joueur P seul)", vies, want)
	}
	if len(got.Morts) != 2 || got.Morts[0].MatchID != "m1" || got.Morts[0].TimeMS != 40000 ||
		got.Morts[0].PlusProcheM == nil || *got.Morts[0].PlusProcheM != 12.5 {
		t.Errorf("morts = %+v, attendu les deux morts de P (40 s, 12,5 m)", got.Morts)
	}
	if len(got.Frags) != 4 {
		t.Fatalf("frags = %+v, attendu 4 (deux de P par match, pas celui de O)", got.Frags)
	}
	if len(got.JournalNonPubliable) != 0 {
		t.Errorf("journaux non publiables = %v, attendu aucun (m1 et m2 publiables)", got.JournalNonPubliable)
	}
	for _, f := range got.Frags {
		if f.CampTueur == nil || *f.CampTueur != 0 || f.CampVictime == nil {
			t.Errorf("frag %+v : camps attendus (tueur 0, victime lue)", f)
		}
	}
	if got.Variantes["m1"] != "Variante 1" || got.Variantes["m2"] != "Variante 2" || len(got.Variantes) != 2 {
		t.Errorf("variantes = %v, attendu m1 et m2 seulement", got.Variantes)
	}
}

// Un match dont la dernière passe du journal n'est pas publiable (m9) : aucun de ses frags n'est
// rendu, mais ses vies et ses morts le sont, et le match est NOMMÉ comme non publiable — c'est ce
// qui permet au calcul d'écarter et de compter ses vies au lieu de les ranger avec zéro frag. La
// lecture de la publiabilité est bornée comme les autres.
func TestSoloLivesRepo_JournalNonPubliable(t *testing.T) {
	b := newBaseNotee(t)
	corpusVies(t, b.pdb)
	b.carnet.vider()

	got, err := NewSoloLivesRepo(b.pdb).LoadLivesNearTeammate(context.Background(), []string{"m1", "m9"}, "P")
	if err != nil {
		t.Fatalf("LoadLivesNearTeammate : %v", err)
	}
	exigerFenetresBornees(t, b, "LoadLivesNearTeammate", 11, 4)
	if fmt.Sprint(got.JournalNonPubliable) != "map[m9:true]" {
		t.Errorf("journaux non publiables = %v, attendu m9 seul", got.JournalNonPubliable)
	}
	for _, f := range got.Frags {
		if f.MatchID == "m9" {
			t.Errorf("frag %+v rendu alors que la passe de m9 n'est pas publiable", f)
		}
	}
	viesM9, mortsM9 := 0, 0
	for _, v := range got.Vies {
		if v.MatchID == "m9" {
			viesM9++
		}
	}
	for _, m := range got.Morts {
		if m.MatchID == "m9" {
			mortsM9++
		}
	}
	if viesM9 != 2 || mortsM9 != 1 {
		t.Errorf("m9 : %d vie(s), %d mort(s), attendu 2 et 1 (portes indépendantes du journal)", viesM9, mortsM9)
	}
}

// Une distance NULL reste nil ; une victime sans ligne de participant a un camp nil.
func TestSoloLivesRepo_NullConserves(t *testing.T) {
	b := newBaseNotee(t)
	corpusVies(t, b.pdb)
	tacKill(t, b.pdb, "m3", "P", "BOT", 50000, true) // aucune ligne de participant pour BOT
	got, err := NewSoloLivesRepo(b.pdb).LoadLivesNearTeammate(context.Background(), []string{"m3"}, "O")
	if err != nil {
		t.Fatalf("LoadLivesNearTeammate : %v", err)
	}
	if len(got.Morts) != 1 || got.Morts[0].PlusProcheM != nil {
		t.Errorf("morts de O = %+v, attendu une mort à distance nil", got.Morts)
	}
	gotP, err := NewSoloLivesRepo(b.pdb).LoadLivesNearTeammate(context.Background(), []string{"m3"}, "P")
	if err != nil {
		t.Fatalf("LoadLivesNearTeammate : %v", err)
	}
	sansCamp := 0
	for _, f := range gotP.Frags {
		if f.CampVictime == nil {
			sansCamp++
		}
	}
	if sansCamp != 1 {
		t.Errorf("frags de P = %+v, attendu un frag à victime sans camp", gotP.Frags)
	}
}

// Liste vide ou joueur vide : aucune requête.
func TestSoloLivesRepo_ListesVides(t *testing.T) {
	b := newBaseNotee(t)
	corpusVies(t, b.pdb)
	b.carnet.vider()
	repo := NewSoloLivesRepo(b.pdb)
	for _, c := range []struct {
		ids  []string
		xuid string
	}{{nil, "P"}, {[]string{"m1"}, ""}} {
		got, err := repo.LoadLivesNearTeammate(context.Background(), c.ids, c.xuid)
		if err != nil || len(got.Vies)+len(got.Morts)+len(got.Frags) != 0 {
			t.Errorf("%+v : %+v, err %v", c, got, err)
		}
	}
	if n := len(b.carnet.vider()); n != 0 {
		t.Errorf("%d requête(s) envoyée(s) pour des listes vides", n)
	}
}

func TestSoloLivesRepo_TableAbsente(t *testing.T) {
	connector, err := duckdbdrv.NewConnector(":memory:", nil)
	if err != nil {
		t.Fatalf("NewConnector: %v", err)
	}
	raw := sql.OpenDB(connector)
	t.Cleanup(func() { _ = raw.Close(); _ = connector.Close() })
	shared := newTestDB(raw, ":memory:")
	pdb := &PlayerDB{Shared: shared, SharedReader: LegacySharedReader(shared)}
	_, err = NewSoloLivesRepo(pdb).LoadLivesNearTeammate(context.Background(), []string{"m1"}, "P")
	if !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Fatalf("err = %v, attendu games.ErrCapabilityNotSupported", err)
	}
}
