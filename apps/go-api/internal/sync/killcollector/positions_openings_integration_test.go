//go:build integration

package killcollector

// positions_openings_integration_test.go — CE QUE LA PASSE D'ENTAMES ÉCRIT VRAIMENT, et ce
// qu'elle fait quand l'écriture échoue.
//
// LES TROIS DÉFAUTS QUE CE FICHIER FERME (revue adversariale du 2026-09-06) :
//
//	B2 — supprimer l'appel `c.persistOpenings(...)` de l'orchestrateur laissait TOUTE la suite
//	     verte : la passe d'entames n'était exercée par aucun test. Elle est décodée, projetée,
//	     validée par le persister… et jamais écrite, sans qu'un seul test rougisse.
//	C7 — un échec d'écriture des POSITIONS annulait silencieusement la passe d'entames, alors
//	     que `writeOpenings` promet en doc un lease séparé « pour qu'un échec de l'une ne fasse
//	     pas retomber l'autre ». Le code disait le contraire de sa doc.
//	C6 — faire retourner `nil` sur erreur au persister laissait tout vert, et
//	     `publishOpeningsPass` aurait alors compté des matchs « couverts » à ZÉRO ligne.
//
// LA BASE EST MONTÉE PAR LES MIGRATIONS RÉELLES (`openSharedTestDB`, collector_test.go) — une
// DDL recopiée dans un test diverge de la production sans que rien ne rougisse, c'est le piège
// le plus cher du dépôt.

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
)

// f64ptr : les coordonnées d'une ligne sont nullables (un seul côté peut être localisé).
func f64ptr(v float64) *float64 { return &v }

// passeDeuxLignes : une passe minimale mais COMPLÈTE — une position de coup fatal et une
// entame pour la même mort, l'une et l'autre localisées des deux côtés.
func passeDeuxLignes(matchID string) passePositions {
	return passePositions{
		rep: replay.KillPosReport{Kills: 1, Both: 1},
		rows: []persist.KillPositionInsert{{
			MatchID: matchID, KillerXUID: "111", TimeMS: 5000,
			KillerX: f64ptr(5), KillerY: f64ptr(0), KillerZ: f64ptr(0),
			VictimX: f64ptr(20), VictimY: f64ptr(0), VictimZ: f64ptr(0),
		}},
		openRep: replay.KillPosReport{Kills: 1, Both: 1},
		openRows: []persist.KillOpeningInsert{{
			MatchID: matchID, KillerXUID: "111", TimeMS: 5000,
			KillerX: f64ptr(3.5), KillerY: f64ptr(0), KillerZ: f64ptr(0),
			VictimX: f64ptr(20), VictimY: f64ptr(0), VictimZ: f64ptr(0),
		}},
	}
}

// compterVuePositions : le nombre de lignes servies par une vue `_latest` pour un match.
// On lit la VUE, jamais la table brute (ADR 0026).
func compterVuePositions(t *testing.T, db *sql.DB, vue, matchID string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM `+vue+` WHERE match_id = ?`, matchID).Scan(&n); err != nil {
		t.Fatalf("select %s: %v", vue, err)
	}
	return n
}

// supprimerTable retire une table ET sa vue `_latest` : c'est ainsi qu'on fabrique un échec
// d'écriture RÉEL (l'INSERT ne trouve plus sa cible) sans mentir sur le reste du schéma.
func supprimerTable(t *testing.T, db *sql.DB, table string) {
	t.Helper()
	for _, stmt := range []string{`DROP VIEW IF EXISTS ` + table + `_latest`, `DROP TABLE ` + table} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
}

// journalCapture redirige le journal par défaut vers un tampon pour la durée du test. Il sert
// à prouver une ABSENCE de trace, ce qu'aucun compteur ne peut dire.
func journalCapture(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	precedent := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(precedent) })
	return &buf
}

// TestEcrireLesDeuxPasses_EcritLesDeuxTables — B2. L'orchestrateur écrit LES DEUX passes.
// Retirer l'appel à `persistOpenings` fait rougir ce test, et lui seul.
func TestEcrireLesDeuxPasses_EcritLesDeuxTables(t *testing.T) {
	db := openSharedTestDB(t)
	c := &KillSourceCollector{acquireShared: sharedWriter(db)}

	c.ecrireLesDeuxPasses(context.Background(), "m-deux-passes", passeDeuxLignes("m-deux-passes"),
		materiauDIsolement{}, MatchIdentities{}, nil)

	if n := compterVuePositions(t, db, "kill_positions_latest", "m-deux-passes"); n != 1 {
		t.Errorf("kill_positions_latest = %d ligne(s), attendu 1", n)
	}
	if n := compterVuePositions(t, db, "kill_openings_latest", "m-deux-passes"); n != 1 {
		t.Fatalf("kill_openings_latest = %d ligne(s), attendu 1 — la passe d entames n a pas ete "+
			"ecrite (appel a persistOpenings retire ?)", n)
	}
	// La ligne d'entame porte le time_ms DU KILL et les coordonnées d'AVANT : c'est ce couple
	// qui rend la jointure du lecteur possible.
	var timeMS int
	var killerX float64
	if err := db.QueryRow(
		`SELECT time_ms, killer_x FROM kill_openings_latest WHERE match_id = 'm-deux-passes'`).
		Scan(&timeMS, &killerX); err != nil {
		t.Fatalf("select entame: %v", err)
	}
	if timeMS != 5000 || killerX != 3.5 {
		t.Errorf("entame ecrite = (time_ms %d, killer_x %v), attendu (5000, 3.5)", timeMS, killerX)
	}
}

// TestEcrireLesDeuxPasses_EchecDesPositionsNAnnulePasLEntame — C7. La table `kill_positions`
// a disparu : son écriture échoue, elle se compte et se journalise. L'entame, elle, s'écrit
// quand même — c'est ce que la doc de `writeOpenings` promet depuis le premier jour.
func TestEcrireLesDeuxPasses_EchecDesPositionsNAnnulePasLEntame(t *testing.T) {
	db := openSharedTestDB(t)
	supprimerTable(t, db, "kill_positions")
	c := &KillSourceCollector{acquireShared: sharedWriter(db)}

	avantEchec := observability.LoadCounter(metricPositionsWriteFail)
	c.ecrireLesDeuxPasses(context.Background(), "m-c7", passeDeuxLignes("m-c7"),
		materiauDIsolement{}, MatchIdentities{}, nil)

	if got := observability.LoadCounter(metricPositionsWriteFail) - avantEchec; got != 1 {
		t.Errorf("%s a bougé de %d, attendu 1", metricPositionsWriteFail, got)
	}
	if n := compterVuePositions(t, db, "kill_openings_latest", "m-c7"); n != 1 {
		t.Errorf("kill_openings_latest = %d ligne(s), attendu 1 — l echec des positions a fait "+
			"retomber la passe d entames", n)
	}
}

// TestPersistOpenings_EchecDEcriture_CompteEtNePublieRien — C6. L'INSERT échoue (table
// absente) : le compteur d'erreurs bouge, celui des matchs COUVERTS ne bouge pas, et aucune
// trace « entames decodees » n'est émise. Sans ces trois assertions ensemble, faire retourner
// `nil` au persister sur erreur laisserait tout vert ET publierait un match « couvert » à zéro
// ligne — un chiffre faux, pas une absence.
func TestPersistOpenings_EchecDEcriture_CompteEtNePublieRien(t *testing.T) {
	db := openSharedTestDB(t)
	supprimerTable(t, db, "kill_openings")
	c := &KillSourceCollector{acquireShared: sharedWriter(db)}
	journal := journalCapture(t)

	avantErreurs := observability.LoadCounter(metricOpeningsWriteErrors)
	avantMatchs := observability.LoadCounter(metricOpeningsMatches)
	c.persistOpenings(context.Background(), "m-c6", passeDeuxLignes("m-c6"))

	if got := observability.LoadCounter(metricOpeningsWriteErrors) - avantErreurs; got != 1 {
		t.Errorf("%s a bougé de %d, attendu 1", metricOpeningsWriteErrors, got)
	}
	if got := observability.LoadCounter(metricOpeningsMatches) - avantMatchs; got != 0 {
		t.Errorf("%s a bougé de %d, attendu 0 (un match dont l ecriture a echoue n est pas couvert)",
			metricOpeningsMatches, got)
	}
	if trace := journal.String(); strings.Contains(trace, "entames decodees") {
		t.Errorf("la trace « entames decodees » a ete emise malgre l echec d ecriture :\n%s", trace)
	}
}

// TestPersistOpenings_EchecDEcriture_CompteQuandMemeLaLecture — résidu 4.0d. Les deux pertes
// de LECTURE (morts sans position, côtés hors vie) décrivent ce que le décodeur a VU : elles
// sont acquises que l'écriture réussisse ou non. Avant la correction, le `return` d'échec
// précédait leur publication — la doc et le test frère promettaient pourtant « il compte MÊME
// quand rien n'est écrit », et un incident d'écriture devenait indiscernable d'une passe qui
// n'avait rien trouvé. Ce qui reste conditionné au succès (couverture, lignes écrites) est
// couvert par le test précédent.
func TestPersistOpenings_EchecDEcriture_CompteQuandMemeLaLecture(t *testing.T) {
	db := openSharedTestDB(t)
	supprimerTable(t, db, "kill_openings")
	c := &KillSourceCollector{acquireShared: sharedWriter(db)}

	pass := passeDeuxLignes("m-4-0-d")
	pass.openRep = replay.KillPosReport{Kills: 4, Both: 1, Dropped: 3, OpeningOutOfLife: 5}

	avantSansPos := observability.LoadCounter(metricOpeningsKillsNoPos)
	avantHorsVie := observability.LoadCounter(metricOpeningsOutOfLife)
	c.persistOpenings(context.Background(), "m-4-0-d", pass)

	if got := observability.LoadCounter(metricOpeningsKillsNoPos) - avantSansPos; got != 3 {
		t.Errorf("%s a bougé de %d, attendu 3 (la lecture a eu lieu, l'écriture a échoué)",
			metricOpeningsKillsNoPos, got)
	}
	if got := observability.LoadCounter(metricOpeningsOutOfLife) - avantHorsVie; got != 5 {
		t.Errorf("%s a bougé de %d, attendu 5 (la lecture a eu lieu, l'écriture a échoué)",
			metricOpeningsOutOfLife, got)
	}
}

// TestWriteOpenings_LeaseIndisponible_RemonteLErreur — le second chemin d'échec de la passe :
// le lease RW n'est pas obtenu. Il ne doit pas se confondre avec un succès silencieux.
func TestWriteOpenings_LeaseIndisponible_RemonteLErreur(t *testing.T) {
	c := &KillSourceCollector{
		acquireShared: func(context.Context) (*sql.DB, func(), error) {
			return nil, nil, errors.New("lease indisponible")
		},
	}
	err := c.writeOpenings(context.Background(), "m-lease",
		passeDeuxLignes("m-lease").openRows)
	if err == nil {
		t.Fatal("attendu une erreur (lease indisponible), obtenu nil")
	}
}
