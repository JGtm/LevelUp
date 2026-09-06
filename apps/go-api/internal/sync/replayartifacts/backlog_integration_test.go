//go:build integration

package replayartifacts

// backlog_integration_test.go — LE RATTRAPAGE, EXECUTE POUR DE VRAI.
//
// POURQUOI CE FICHIER EST UN TEST D INTEGRATION. La requete de retard est construite par
// concatenation (fenetre de retention optionnelle, fragment de temps canonique) : un `?` de
// trop, une virgule manquante ou un renommage de colonne gardent toutes les sous-chaines
// presentes, donc un test `strings.Contains` resterait vert pendant qu a l execution
// `QueryContext` echoue -> WARN -> retard vide -> rattrapage no-op permanent. C est-a-dire
// exactement le mode de panne silencieuse que ce lot corrige, reproduit dans son propre
// garde-rail (lecon de killcollector/postsync_backlog_integration_test.go).
//
// Ici la requete tourne sur une vraie base migree. Elle ne peut plus mentir sur sa syntaxe, ni
// sur l ensemble qu elle selectionne, ni sur son ordre.
//
// AUCUN ARTEFACT N EST CONSTRUIT ICI (decision D8 du plan) : les tests forgent des FICHIERS
// d artefact, ils ne decodent aucun film.

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/analysis/replay"
	titlePkg "levelup/go-api/internal/domain/title"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/replaybuild"
	"levelup/go-api/internal/sync/haloclient"
	"levelup/go-api/internal/sync/matchflags"
)

func baseRegistre(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "retard.duckdb"))
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	if err := migration.RunForDB(db, migration.TargetShared); err != nil {
		t.Fatalf("migrate shared: %v", err)
	}
	return db
}

func inscrireAuRegistre(t *testing.T, db *sql.DB, id string, quand time.Time, bits int64) {
	t.Helper()
	_, err := db.ExecContext(context.Background(),
		`INSERT INTO match_registry (match_id, start_time, start_time_utc, backfill_completed, map_name)
		 VALUES (?, ?, ?, ?, 'Recharge')`, id, quand, quand, bits)
	if err != nil {
		t.Fatalf("insert registre %s: %v", id, err)
	}
}

func poserArtefact(t *testing.T, repoRoot, matchID string) {
	t.Helper()
	path := titlePkg.NewPathResolver(repoRoot).ReplayArtifactPath(titlePkg.DefaultSlug, matchID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	blob, err := json.Marshal(replay.ReplayDocument{SchemaVersion: replay.SchemaVersion, MatchID: matchID})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, blob, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func depsRattrapage(repoRoot string, months int) Deps {
	return Deps{RepoRoot: repoRoot, TitleSlug: titlePkg.DefaultSlug, RetentionMonths: months}
}

func ids(work []buildWork) []string {
	out := make([]string, 0, len(work))
	for _, w := range work {
		out = append(out, w.matchID)
	}
	return out
}

// TestCandidatsARattraper_SelectionOrdreEtJauge : les quatre proprietes du rattrapage, sur une
// base reelle.
//
//  1. le marqueur terminal EXCLUT   un match dont le film est declare absent ne revient jamais
//     (~29 % du corpus, sinon il occuperait le lot a vie).
//  2. l artefact present EXCLUT     c est le travail deja fait.
//  3. les matchs deja pris par le chemin des inseres EXCLUENT — pas de double traitement.
//  4. l ordre est du PLUS RECENT au plus vieux, et le plafond borne le LOT sans effacer la
//     jauge : ce qui n est pas pris ce cycle reste compte comme du retard.
func TestCandidatsARattraper_SelectionOrdreEtJauge(t *testing.T) {
	db := baseRegistre(t)
	repoRoot := t.TempDir()
	t0 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	inscrireAuRegistre(t, db, "vieux-sans-film", t0, int64(matchflags.MBitFilmAbsent)) // (1)
	inscrireAuRegistre(t, db, "deja-cuit", t0.AddDate(0, 1, 0), 0)                     // (2)
	poserArtefact(t, repoRoot, "deja-cuit")
	inscrireAuRegistre(t, db, "pris-par-les-inseres", t0.AddDate(0, 2, 0), 0) // (3)
	inscrireAuRegistre(t, db, "en-retard-ancien", t0.AddDate(0, 3, 0), 0)
	inscrireAuRegistre(t, db, "en-retard-recent", t0.AddDate(0, 4, 0), 0)

	deja := map[string]bool{"pris-par-les-inseres": true}
	work, restant := candidatsARattraper(context.Background(), db, nil, depsRattrapage(repoRoot, 0), deja, 5)

	attendu := []string{"en-retard-recent", "en-retard-ancien"}
	got := ids(work)
	if len(got) != len(attendu) {
		t.Fatalf("travail = %v, attendu %v", got, attendu)
	}
	for i := range attendu {
		if got[i] != attendu[i] {
			t.Errorf("travail[%d] = %q, attendu %q (ordre : du plus recent au plus vieux)", i, got[i], attendu[i])
		}
	}
	if restant != 0 {
		t.Errorf("retard restant = %d, attendu 0 (tout tenait dans le plafond)", restant)
	}
}

// TestCandidatsARattraper_PlafondLaisseDuRetard : le plafond borne le LOT, pas la mesure. Une
// jauge remise a zero parce que le lot etait plein decrirait le lot, pas le retard.
func TestCandidatsARattraper_PlafondLaisseDuRetard(t *testing.T) {
	db := baseRegistre(t)
	repoRoot := t.TempDir()
	t0 := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	for i, id := range []string{"m1", "m2", "m3", "m4"} {
		inscrireAuRegistre(t, db, id, t0.Add(time.Duration(i)*time.Hour), 0)
	}

	work, restant := candidatsARattraper(context.Background(), db, nil, depsRattrapage(repoRoot, 0), nil, 1)
	if len(work) != 1 || work[0].matchID != "m4" {
		t.Fatalf("travail = %v, attendu [m4] (le plus recent d abord)", ids(work))
	}
	if restant != 3 {
		t.Errorf("retard restant = %d, attendu 3", restant)
	}
}

// TestCandidatsARattraper_PlafondNulMesureQuandMeme : un lot deja rempli par les matchs
// inseres ne doit pas rendre la jauge muette — sinon elle ne se rafraichit que les bons jours.
func TestCandidatsARattraper_PlafondNulMesureQuandMeme(t *testing.T) {
	db := baseRegistre(t)
	t0 := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	inscrireAuRegistre(t, db, "en-retard", t0, 0)

	work, restant := candidatsARattraper(context.Background(), db, nil, depsRattrapage(t.TempDir(), 0), nil, 0)
	if len(work) != 0 {
		t.Errorf("travail = %v, attendu aucun (plafond nul)", ids(work))
	}
	if restant != 1 {
		t.Errorf("retard restant = %d, attendu 1", restant)
	}
}

// TestCandidatsARattraper_FenetreDeRetention : on ne rattrape pas ce que la purge effacera.
func TestCandidatsARattraper_FenetreDeRetention(t *testing.T) {
	db := baseRegistre(t)
	maintenant := time.Now().UTC()
	inscrireAuRegistre(t, db, "dans-la-fenetre", maintenant.AddDate(0, 0, -10), 0)
	inscrireAuRegistre(t, db, "hors-fenetre", maintenant.AddDate(0, -8, 0), 0)

	work, _ := candidatsARattraper(context.Background(), db, nil, depsRattrapage(t.TempDir(), 3), nil, 10)
	if got := ids(work); len(got) != 1 || got[0] != "dans-la-fenetre" {
		t.Fatalf("travail = %v, attendu [dans-la-fenetre] : la fenetre de retention doit s appliquer AVANT le lot", got)
	}
}

// TestRun_RattrapeSansAucuneInsertion — LE DEFAUT MESURE LE 2026-09-01, FERME.
//
// Un cycle qui n insere rien devait tout de meme rattraper : le film Theater se publie APRES la
// partie. Avant ce lot, `runReplayArtifacts` sortait sur `len(insertedIDs) == 0` et `Run`
// aussi ; 221 des 222 matchs des 90 derniers jours n avaient donc aucun artefact.
//
// Le placement est « worker » : la mise en file est le SEUL chemin qui prouve la selection sans
// decoder un film (decision D8 — aucun artefact n est construit par ce lot).
func TestRun_RattrapeSansAucuneInsertion(t *testing.T) {
	db := baseRegistre(t)
	repoRoot := t.TempDir()
	t0 := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	inscrireAuRegistre(t, db, "aaaaaaaa-1111-4000-8000-000000000000", t0, 0)
	inscrireAuRegistre(t, db, "bbbbbbbb-2222-4000-8000-000000000000", t0.Add(time.Hour), 0)

	var enfiles []string
	Run(context.Background(), Deps{
		Placement: replaybuild.PlacementWorker,
		RepoRoot:  repoRoot,
		TitleSlug: titlePkg.DefaultSlug,
		WithRead:  func(ctx context.Context, _ string, fn func(*sql.DB)) { fn(db) },
		Enqueue: func(_ context.Context, _, matchID string) error {
			enfiles = append(enfiles, matchID)
			return nil
		},
	}, nil) // AUCUN match insere

	if len(enfiles) != 2 {
		t.Fatalf("mis en file = %v, attendu les 2 matchs du retard : sans insertion, le cycle "+
			"ne rattrapait rien et c est tout le defaut", enfiles)
	}
}

// TestRun_LaJaugeDecroitSurDeuxCycles — LE CRITERE DE RECETTE DU RATTRAPAGE.
//
// Un rattrapage qui ne converge pas n en est pas un. Deux cycles consecutifs sont joues sur la
// meme base : le premier prend son lot, l ouvrier simule depose les artefacts, le second
// constate un retard STRICTEMENT plus petit.
//
// AUCUN FILM N EST DECODE (decision D8) : le placement est « worker », donc le cycle met en
// file ; c est la fonction d enfilage du test qui pose le fichier d artefact, comme le ferait
// un ouvrier.
func TestRun_LaJaugeDecroitSurDeuxCycles(t *testing.T) {
	db := baseRegistre(t)
	repoRoot := t.TempDir()
	t0 := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 9; i++ {
		inscrireAuRegistre(t, db, fmt.Sprintf("%08d-1111-4000-8000-000000000000", i),
			t0.Add(time.Duration(i)*time.Hour), 0)
	}
	d := Deps{
		Placement: replaybuild.PlacementWorker,
		RepoRoot:  repoRoot,
		TitleSlug: titlePkg.DefaultSlug,
		WithRead:  func(ctx context.Context, _ string, fn func(*sql.DB)) { fn(db) },
		Enqueue: func(_ context.Context, _, matchID string) error {
			poserArtefact(t, repoRoot, matchID) // l ouvrier a fait son travail
			return nil
		},
	}

	Run(context.Background(), d, nil)
	premier := observability.LoadCounterT("", CompteurRetard)
	Run(context.Background(), d, nil)
	second := observability.LoadCounterT("", CompteurRetard)

	if premier <= 0 {
		t.Fatalf("retard apres le 1er cycle = %d : le lot devait laisser du solde (9 matchs, plafond %d)",
			premier, maxPerCycle)
	}
	if second >= premier {
		t.Errorf("retard : %d puis %d — la jauge ne decroit pas, le rattrapage ne converge pas",
			premier, second)
	}
}

// TestBuildAll_BudgetEpuiseArreteEntreDeuxMatchs : la borne `maxPerCycle` compte des matchs,
// pas des secondes. Un budget a zero doit arreter la passe AVANT le premier match — la preuve
// que la garde existe et qu elle s applique entre deux matchs, sans decoder quoi que ce soit.
func TestBuildAll_BudgetEpuiseArreteEntreDeuxMatchs(t *testing.T) {
	appels := 0
	d := Deps{
		RepoRoot:  t.TempDir(),
		TitleSlug: titlePkg.DefaultSlug,
		Budget:    -1, // deja epuise a la premiere iteration
		Fetcher:   fetcherComptant{n: &appels},
	}
	b := buildAll(context.Background(), d, []buildWork{{matchID: "m1"}, {matchID: "m2"}})
	if !b.budgetEpuise {
		t.Error("le bilan ne signale pas le budget epuise : le cycle mentirait sur ce qu il a fait")
	}
	if appels != 0 {
		t.Errorf("appels au client film = %d, attendu 0 : le budget doit arreter AVANT le pont disque", appels)
	}
}

// fetcherComptant : un ChunksFetcher qui compte ses appels et ne rend jamais de film.
type fetcherComptant struct{ n *int }

func (f fetcherComptant) GetFilmChunks(context.Context, string) ([]haloclient.FilmChunk, bool, error) {
	*f.n++
	return nil, false, nil
}

// ─── LE RATTRAPAGE DES DÉRIVÉS (constat A2) ───────────────────────────────────────────────────
//
// Même raison d'être que les tests ci-dessus : la requête d'horizon est partagée avec le
// rattrapage de cuisson, mais la SÉLECTION est différente (artefact présent ET dérivés absents).
// Elle ne peut être éprouvée que sur une vraie base migrée.

// TestCandidatsDerivations_SelectionneLesRangesSansDerives : la sélection, cas par cas.
func TestCandidatsDerivations_SelectionneLesRangesSansDerives(t *testing.T) {
	db := baseRegistre(t)
	repoRoot := t.TempDir()
	maintenant := time.Now().UTC()

	// m-sans-artefact : rien sur disque -> AFFAIRE DE LA CUISSON, pas de la dérivation.
	inscrireAuRegistre(t, db, "sansarte", maintenant.Add(-1*time.Hour), 0)
	// m-a-deriver : artefact rangé, aucune marque -> candidat.
	inscrireAuRegistre(t, db, "aderiver", maintenant.Add(-2*time.Hour), 0)
	poserArtefact(t, repoRoot, "aderiver")
	// m-deja-derive : artefact rangé ET marqué à la révision courante -> écarté.
	inscrireAuRegistre(t, db, "dejaderi", maintenant.Add(-3*time.Hour), 0)
	poserArtefact(t, repoRoot, "dejaderi")
	marquerArtefactCommeDerive(t, repoRoot, "dejaderi")

	work, restant := candidatsDerivations(context.Background(), db, depsRattrapage(repoRoot, 0))

	if restant != 0 {
		t.Errorf("restant = %d, attendu 0 (trois matchs, un seul candidat)", restant)
	}
	if len(work) != 1 || work[0].MatchID != "aderiver" {
		t.Fatalf("candidats = %+v, attendu le seul artefact rangé sans marque", work)
	}
	attendu := titlePkg.NewPathResolver(repoRoot).ReplayArtifactPath(titlePkg.DefaultSlug, "aderiver")
	if work[0].Path != attendu {
		t.Errorf("chemin = %q, attendu la place canonique %q", work[0].Path, attendu)
	}
}

// TestCandidatsDerivations_ConvergeApresDerivation : LA PROPRIÉTÉ QUI FAIT TOUT TENIR. Sans
// marque, les mêmes cinq artefacts reviendraient à chaque cycle, indéfiniment.
func TestCandidatsDerivations_ConvergeApresDerivation(t *testing.T) {
	db := baseRegistre(t)
	repoRoot := t.TempDir()
	inscrireAuRegistre(t, db, "unmatch1", time.Now().UTC().Add(-time.Hour), 0)
	poserArtefact(t, repoRoot, "unmatch1")
	d := depsRattrapage(repoRoot, 0)

	work, _ := candidatsDerivations(context.Background(), db, d)
	if len(work) != 1 {
		t.Fatalf("cycle 1 : %d candidat(s), attendu 1", len(work))
	}
	// La dérivation pose la marque (marquerDerivations, appelé par Deriver). Bilan VIERGE :
	// aucune famille n'a échoué, donc la marque se pose — c'est la seule condition depuis le
	// constat C1 de la revue A-R1.
	marquerDerivations(context.Background(), &bilanDerivations{},
		lireArtefacts(context.Background(), d, work))

	work2, restant := candidatsDerivations(context.Background(), db, d)
	if len(work2) != 0 || restant != 0 {
		t.Fatalf("cycle 2 : %d candidat(s) et %d restant(s), attendu 0 et 0 — le rattrapage "+
			"ne converge pas, il rejouerait les mêmes artefacts à chaque cycle", len(work2), restant)
	}
}

// TestCandidatsDerivations_PlafondEtRestant : le lot est borné par maxPerCycle, et ce qui
// dépasse est COMPTÉ (une jauge qui reste à zéro pendant qu'il reste du travail ne décrit rien).
func TestCandidatsDerivations_PlafondEtRestant(t *testing.T) {
	db := baseRegistre(t)
	repoRoot := t.TempDir()
	const total = maxPerCycle + 3
	for i := 0; i < total; i++ {
		id := fmt.Sprintf("mderiv%02d", i)
		inscrireAuRegistre(t, db, id, time.Now().UTC().Add(-time.Duration(i)*time.Hour), 0)
		poserArtefact(t, repoRoot, id)
	}

	work, restant := candidatsDerivations(context.Background(), db, depsRattrapage(repoRoot, 0))

	if len(work) != maxPerCycle {
		t.Errorf("lot = %d, attendu le plafond %d", len(work), maxPerCycle)
	}
	if restant != total-maxPerCycle {
		t.Errorf("restant = %d, attendu %d", restant, total-maxPerCycle)
	}
}

// TestCandidatsDerivations_ArtefactPerimeEstDeriveTelQuel : la re-cuisson du corpus est un
// ARBITRAGE UTILISATEUR DATÉ (registre des reports l. 17) que ce rattrapage ne renverse pas —
// il dérive l'artefact TEL QU'IL EST, ce qui vaut mieux que rien.
func TestCandidatsDerivations_ArtefactPerimeEstDeriveTelQuel(t *testing.T) {
	db := baseRegistre(t)
	repoRoot := t.TempDir()
	inscrireAuRegistre(t, db, "perimeee", time.Now().UTC().Add(-time.Hour), 0)
	poserArtefactPerime(t, repoRoot, "perimeee")

	work, _ := candidatsDerivations(context.Background(), db, depsRattrapage(repoRoot, 0))
	if len(work) != 1 || work[0].MatchID != "perimeee" {
		t.Fatalf("candidats = %+v, attendu l'artefact périmé (dérivé tel quel, JAMAIS re-cuit ici)", work)
	}
}

// TestRattraperDerivations_SansSegmentDeLectureNeFaitRien : un chemin de sync sans lecture
// câblée ne panique pas et ne prétend rien rattraper.
func TestRattraperDerivations_SansSegmentDeLectureNeFaitRien(t *testing.T) {
	rattraperDerivations(context.Background(), Deps{RepoRoot: t.TempDir(), TitleSlug: titlePkg.DefaultSlug})
}

// marquerArtefactCommeDerive pose la marque de dérivation d'un artefact déjà sur disque.
func marquerArtefactCommeDerive(t *testing.T, repoRoot, matchID string) {
	t.Helper()
	path := titlePkg.NewPathResolver(repoRoot).ReplayArtifactPath(titlePkg.DefaultSlug, matchID)
	st, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat artefact %s: %v", matchID, err)
	}
	if err := replaybuild.WriteDerivationsMark(path, replay.SchemaVersion, int(st.Size())); err != nil {
		t.Fatalf("WriteDerivationsMark %s: %v", matchID, err)
	}
}

// poserArtefactPerime pose un artefact d'une version de schéma ANTÉRIEURE — la situation des
// 106 artefacts du cache local (9 versions, aucune à la courante).
func poserArtefactPerime(t *testing.T, repoRoot, matchID string) {
	t.Helper()
	path := titlePkg.NewPathResolver(repoRoot).ReplayArtifactPath(titlePkg.DefaultSlug, matchID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	blob, err := json.Marshal(replay.ReplayDocument{SchemaVersion: replay.SchemaVersion - 1, MatchID: matchID})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, blob, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}
