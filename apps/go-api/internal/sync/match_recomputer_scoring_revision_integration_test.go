//go:build integration

// match_recomputer_scoring_revision_integration_test.go : le rejeu des KPI après un changement de
// formule (ApplyScoringRevision). Contrat prouvé : la révision n'est rangée qu'après un recalcul
// réussi, une révision à jour ne recalcule rien, une révision différente relance.
package sync

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	"levelup/go-api/internal/sync/skill"
)

const scoringRevisionTestMatches = 12

// scoringRevisionFixture : bases de test, fournisseur partagé et recalculateur du joueur de test.
type scoringRevisionFixture struct {
	playerPath string
	rec        *MatchRecomputer
}

func newScoringRevisionFixture(t *testing.T, provider func(base sharedprovider.Provider) sharedprovider.Provider) *scoringRevisionFixture {
	t.Helper()
	dir := t.TempDir()
	sharedPath := filepath.Join(dir, "shared_matches_v2.duckdb")
	playerPath := filepath.Join(dir, "player", "stats.duckdb")
	seedLUSRShared(t, sharedPath, scoringRevisionTestMatches)
	seedLUSRPlayer(t, playerPath)
	base, err := sharedprovider.New(sharedPath)
	if err != nil {
		t.Fatalf("sharedprovider.New: %v", err)
	}
	t.Cleanup(func() { _ = base.Close() })
	return &scoringRevisionFixture{
		playerPath: playerPath,
		rec: NewMatchRecomputer(playerPath, sharedPath, "", recomputerTestXUID, "Tester", provider(base)).
			WithTitle("halo_infinite"),
	}
}

func identity(p sharedprovider.Provider) sharedprovider.Provider { return p }

// query ouvre la base joueur en lecture le temps de fn (aucun écrivain n'est tenu hors recalcul).
func (f *scoringRevisionFixture) query(t *testing.T, fn func(db *sql.DB)) {
	t.Helper()
	db, err := sql.Open("duckdb", f.playerPath+"?access_mode=read_only")
	if err != nil {
		t.Fatalf("open player: %v", err)
	}
	defer func() { _ = db.Close() }()
	fn(db)
}

// storedRevision rend la révision rangée dans sync_meta ("" si aucune).
func (f *scoringRevisionFixture) storedRevision(t *testing.T) string {
	t.Helper()
	var v sql.NullString
	f.query(t, func(db *sql.DB) {
		err := db.QueryRow(`SELECT value FROM sync_meta WHERE key = ?`, skill.SyncMetaScoringRevision).Scan(&v)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("lecture de la révision: %v", err)
		}
	})
	return v.String
}

// count rend le résultat d'un COUNT(*) sur la base joueur.
func (f *scoringRevisionFixture) count(t *testing.T, query string) int {
	t.Helper()
	var n int
	f.query(t, func(db *sql.DB) {
		if err := db.QueryRow(query).Scan(&n); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
	})
	return n
}

const (
	lusrNotesQuery = `SELECT COUNT(*) FROM match_skill_rank_latest WHERE rating_type = 'LUSR'`
	// lusrWritesQuery compte TOUTES les lignes écrites (la table est append-only) : une passe de
	// plus en ajoute, une passe évitée n'en ajoute aucune.
	lusrWritesQuery = `SELECT COUNT(*) FROM match_skill_rank`
	perfNotesQuery  = `SELECT COUNT(*) FROM player_match_enrichment_latest WHERE performance_score IS NOT NULL`
)

// Premier appel : recalcul complet puis révision rangée ; second appel avec la même révision : rien
// n'est recalculé (aucune ligne écrite de plus) ; une révision différente relance le recalcul.
func TestMatchRecomputer_ApplyScoringRevision_PremierAppelPuisAJourPuisRelance(t *testing.T) {
	f := newScoringRevisionFixture(t, identity)
	ctx := context.Background()

	applied, err := f.rec.ApplyScoringRevision(ctx, "rev-a")
	if err != nil || !applied {
		t.Fatalf("premier appel : applied=%v err=%v, attendu true, nil", applied, err)
	}
	if got := f.storedRevision(t); got != "rev-a" {
		t.Fatalf("révision rangée = %q, attendu rev-a", got)
	}
	if got := f.count(t, lusrNotesQuery); got != scoringRevisionTestMatches {
		t.Fatalf("%d match(s) notés LUSR, attendu %d", got, scoringRevisionTestMatches)
	}
	if got := f.count(t, perfNotesQuery); got == 0 {
		t.Fatal("aucune note de performance après le recalcul : la fixture ne prouve rien sur la passe des notes")
	}
	writes := f.count(t, lusrWritesQuery)

	applied, err = f.rec.ApplyScoringRevision(ctx, "rev-a")
	if err != nil || applied {
		t.Fatalf("second appel : applied=%v err=%v, attendu false, nil", applied, err)
	}
	if got := f.count(t, lusrWritesQuery); got != writes {
		t.Fatalf("lignes de notes %d puis %d : une révision à jour ne doit rien recalculer", writes, got)
	}

	applied, err = f.rec.ApplyScoringRevision(ctx, "rev-b")
	if err != nil || !applied {
		t.Fatalf("révision différente : applied=%v err=%v, attendu true, nil", applied, err)
	}
	if got := f.storedRevision(t); got != "rev-b" {
		t.Fatalf("révision rangée = %q, attendu rev-b", got)
	}
	if got := f.count(t, lusrWritesQuery); got <= writes {
		t.Fatalf("lignes de notes %d puis %d : une révision différente doit relancer le rejeu", writes, got)
	}
}

// Un recalcul en échec (écrivain partagé perdu au milieu du rejeu) ne range aucune révision ; le
// démarrage suivant, écrivain revenu, mène le recalcul au bout et la range.
func TestMatchRecomputer_ApplyScoringRevision_RecalculEnEchecNeNotePas(t *testing.T) {
	var flaky *flakyWriterProvider
	f := newScoringRevisionFixture(t, func(base sharedprovider.Provider) sharedprovider.Provider {
		flaky = &flakyWriterProvider{Provider: base, failFrom: 2}
		return flaky
	})
	ctx := context.Background()

	applied, err := f.rec.ApplyScoringRevision(ctx, "rev-a")
	if !errors.Is(err, errSharedWriterDown) || applied {
		t.Fatalf("recalcul sans écrivain : applied=%v err=%v, attendu false et l'erreur de l'écrivain", applied, err)
	}
	if got := f.storedRevision(t); got != "" {
		t.Fatalf("révision rangée = %q après un recalcul en échec, attendu aucune", got)
	}

	flaky.failFrom = 1 << 30
	applied, err = f.rec.ApplyScoringRevision(ctx, "rev-a")
	if err != nil || !applied {
		t.Fatalf("relance : applied=%v err=%v, attendu true, nil", applied, err)
	}
	if got := f.storedRevision(t); got != "rev-a" {
		t.Fatalf("révision rangée = %q, attendu rev-a", got)
	}
}

// Une écriture des notes de performance refusée ne range aucune révision : sans l'erreur rendue par
// le calcul des notes, les anciennes notes resteraient et le recalcul serait noté réussi.
func TestMatchRecomputer_ApplyScoringRevision_EcritureDesNotesEnEchecNeNotePas(t *testing.T) {
	f := newScoringRevisionFixture(t, identity)
	breakPerfNotesWrite(t, f.playerPath)

	applied, err := f.rec.ApplyScoringRevision(context.Background(), "rev-a")
	if err == nil || applied {
		t.Fatalf("écriture des notes refusée : applied=%v err=%v, attendu false et une erreur", applied, err)
	}
	if got := f.storedRevision(t); got != "" {
		t.Fatalf("révision rangée = %q après un échec d'écriture des notes, attendu aucune", got)
	}
}

// breakPerfNotesWrite rend l'INSERT des notes de performance impossible sans toucher aux lectures
// tolérantes du calcul : la vue est retirée (le calcul en repart sans note existante) et la colonne de
// chaîne devient un entier (l'INSERT d'un nom de chaîne ne se convertit plus). Une colonne renommée ou
// retirée serait recréée par l'ouverture de la base joueur ; un type changé ne l'est pas.
func breakPerfNotesWrite(t *testing.T, playerPath string) {
	t.Helper()
	h, err := OpenPlayerDB(playerPath)
	if err != nil {
		t.Fatalf("OpenPlayerDB: %v", err)
	}
	defer h.Close()
	mustExec(t, h.SQLDb(), `DROP VIEW player_match_enrichment_latest`)
	mustExec(t, h.SQLDb(), `ALTER TABLE player_match_enrichment ALTER COLUMN performance_chain SET DATA TYPE INTEGER`)
}

// Le calcul des notes rend l'erreur d'écriture (après nettoyage et résumé) au lieu de la taire.
func TestScoringRevision_BatchComputePerformanceScoresRendLErreurDEcriture(t *testing.T) {
	dir := t.TempDir()
	sharedPath := filepath.Join(dir, "shared_matches_v2.duckdb")
	playerPath := filepath.Join(dir, "player", "stats.duckdb")
	seedLUSRShared(t, sharedPath, scoringRevisionTestMatches)
	seedLUSRPlayer(t, playerPath)
	breakPerfNotesWrite(t, playerPath)

	shared, err := OpenSharedDB(sharedPath)
	if err != nil {
		t.Fatalf("OpenSharedDB: %v", err)
	}
	defer shared.Close()
	player, err := OpenPlayerDB(playerPath)
	if err != nil {
		t.Fatalf("OpenPlayerDB: %v", err)
	}
	defer player.Close()

	n, err := batchComputePerformanceScores(context.Background(), player.SQLDb(), shared.SQLDb(), recomputerTestXUID, nil, true)
	if err == nil {
		t.Fatalf("écriture des notes refusée : n=%d err=nil, attendu une erreur", n)
	}
	if n != 0 {
		t.Errorf("n = %d avec une erreur, attendu 0 (aucune note écrite)", n)
	}
}

// breakLUSRNoteWrite rend l'INSERT d'une note LUSR impossible : les vues de la table sont retirées et
// rating_type devient un entier (l'INSERT de 'LUSR' ne se convertit plus). Rend la fonction qui remet
// la colonne en VARCHAR et recrée les vues par l'ouverture de la base joueur.
func breakLUSRNoteWrite(t *testing.T, playerPath string) (repair func()) {
	t.Helper()
	h, err := OpenPlayerDB(playerPath)
	if err != nil {
		t.Fatalf("OpenPlayerDB: %v", err)
	}
	for _, v := range []string{"match_skill_rank_latest_by_type", "match_skill_rank_latest"} {
		mustExec(t, h.SQLDb(), `DROP VIEW IF EXISTS `+v)
	}
	mustExec(t, h.SQLDb(), `ALTER TABLE match_skill_rank ALTER COLUMN rating_type SET DATA TYPE INTEGER`)
	if err := h.Close(); err != nil {
		t.Fatalf("close player: %v", err)
	}
	return func() {
		h, err := OpenPlayerDB(playerPath)
		if err != nil {
			t.Fatalf("OpenPlayerDB: %v", err)
		}
		defer h.Close()
		mustExec(t, h.SQLDb(), `ALTER TABLE match_skill_rank ALTER COLUMN rating_type SET DATA TYPE VARCHAR`)
	}
}

// Un match dont la note LUSR ne s'écrit pas ne fait pas échouer le rejeu (il est journalisé) mais le
// recalcul est incomplet : aucune révision n'est rangée, elle l'est au prochain essai réussi.
func TestMatchRecomputer_ApplyScoringRevision_MatchEnEchecNeNotePas(t *testing.T) {
	f := newScoringRevisionFixture(t, identity)
	repair := breakLUSRNoteWrite(t, f.playerPath)

	applied, err := f.rec.ApplyScoringRevision(context.Background(), "rev-a")
	if !errors.Is(err, ErrPartialRecompute) || applied {
		t.Fatalf("match en échec : applied=%v err=%v, attendu false et ErrPartialRecompute", applied, err)
	}
	if got := f.storedRevision(t); got != "" {
		t.Fatalf("révision rangée = %q alors qu'un match a échoué, attendu aucune", got)
	}

	repair()
	applied, err = f.rec.ApplyScoringRevision(context.Background(), "rev-a")
	if err != nil || !applied {
		t.Fatalf("essai suivant : applied=%v err=%v, attendu true, nil", applied, err)
	}
	if got := f.storedRevision(t); got != "rev-a" {
		t.Fatalf("révision rangée = %q, attendu rev-a", got)
	}
	if got := f.count(t, lusrNotesQuery); got != scoringRevisionTestMatches {
		t.Fatalf("%d match(s) notés LUSR, attendu %d", got, scoringRevisionTestMatches)
	}
}

// Après un rejeu abouti, les caches de lectures du joueur sont invalidés (sinon l'écran sert les
// anciennes notes jusqu'à l'expiration de l'entrée) ; une révision à jour ou un recalcul en échec
// n'y touchent pas.
func TestMatchRecomputer_ApplyScoringRevision_InvalideLesCachesDeLecture(t *testing.T) {
	f := newScoringRevisionFixture(t, identity)
	var calls []string
	f.rec.invalidateReadCaches = func(_ context.Context, xuid, titleSlug string) { calls = append(calls, xuid+"/"+titleSlug) }
	ctx := context.Background()

	if _, err := f.rec.ApplyScoringRevision(ctx, "rev-a"); err != nil {
		t.Fatalf("ApplyScoringRevision: %v", err)
	}
	if len(calls) != 1 || calls[0] != recomputerTestXUID+"/halo_infinite" {
		t.Fatalf("invalidations = %v, attendu une seule pour le joueur et son titre", calls)
	}
	if _, err := f.rec.ApplyScoringRevision(ctx, "rev-a"); err != nil || len(calls) != 1 {
		t.Fatalf("révision à jour : err=%v invalidations=%v, attendu aucune de plus", err, calls)
	}

	failing := newScoringRevisionFixture(t, identity)
	var failingCalls int
	failing.rec.invalidateReadCaches = func(context.Context, string, string) { failingCalls++ }
	breakPerfNotesWrite(t, failing.playerPath)
	if _, err := failing.rec.ApplyScoringRevision(ctx, "rev-a"); err == nil {
		t.Fatal("attendu une erreur d'écriture des notes")
	}
	if failingCalls != 0 {
		t.Fatalf("%d invalidation(s) après un recalcul en échec, attendu aucune", failingCalls)
	}
}

// Un match dont la note LUSR ne s'écrit pas ne fait pas échouer l'exclusion d'un match pour
// l'utilisateur (comportement d'avant : l'échec est journalisé), mais le recalcul reste marqué
// échoué : une demande répétée recalcule.
func TestMatchRecomputer_RecomputeAfterExclusion_MatchEnEchecNEchouePasEtResteEnAttente(t *testing.T) {
	f := newScoringRevisionFixture(t, identity)
	repair := breakLUSRNoteWrite(t, f.playerPath)
	ctx := context.Background()
	const matchID = "00000000-0000-0000-0000-000000000001"

	if err := f.rec.RecomputeAfterExclusion(ctx, matchID); err != nil {
		t.Fatalf("RecomputeAfterExclusion avec un match en échec : %v, attendu nil", err)
	}
	if !f.rec.RecomputePending() {
		t.Fatal("RecomputePending = false : une demande répétée doit recalculer")
	}

	repair()
	if err := f.rec.RecomputeAfterExclusion(ctx, matchID); err != nil {
		t.Fatalf("demande répétée : %v", err)
	}
	if f.rec.RecomputePending() {
		t.Error("RecomputePending = true après un recalcul réussi")
	}
}
