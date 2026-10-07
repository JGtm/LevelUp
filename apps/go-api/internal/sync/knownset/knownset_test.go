package knownset

// knownset_test.go — la règle « connu » sur DuckDB réel (en mémoire) : régime normal identique
// à l'ancienne union, enrichissement orphelin inconnu et compté, base partagée illisible fatale.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"slices"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/observability"
)

const (
	xuidJoueur = "2533274823110022"
	xuidAutre  = "2535469190789936"
)

func openMem(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func exec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

// newPlayerDB : base joueur avec la vraie vue player_match_enrichment_latest (migration
// append-only) et une ligne d'enrichissement par match_id.
func newPlayerDB(t *testing.T, enriched ...string) *sql.DB {
	t.Helper()
	db := openMem(t)
	exec(t, db, `CREATE TABLE player_match_enrichment (
		match_id VARCHAR PRIMARY KEY, performance_score FLOAT, session_id VARCHAR,
		session_label VARCHAR, is_with_friends BOOLEAN DEFAULT FALSE, teammates_signature VARCHAR)`)
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(db); err != nil {
		t.Fatalf("EnsurePlayerMatchEnrichmentAppendOnly: %v", err)
	}
	for _, id := range enriched {
		exec(t, db, `INSERT INTO player_match_enrichment (match_id) VALUES (?)`, id)
	}
	return db
}

// sharedFixture : contenu de la base partagée — registre et participants par xuid.
type sharedFixture struct {
	registry     []string
	participants map[string][]string // xuid → match_id
}

func newSharedDB(t *testing.T, f sharedFixture) *sql.DB {
	t.Helper()
	db := openMem(t)
	exec(t, db, `CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY)`)
	exec(t, db, `CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR)`)
	for _, id := range f.registry {
		exec(t, db, `INSERT INTO match_registry VALUES (?)`, id)
	}
	for xuid, ids := range f.participants {
		for _, id := range ids {
			exec(t, db, `INSERT INTO match_participants VALUES (?, ?)`, id, xuid)
		}
	}
	return db
}

func keys(m map[string]bool) []string {
	return slices.Sorted(maps.Keys(m))
}

func assertKnown(t *testing.T, got map[string]bool, want ...string) {
	t.Helper()
	slices.Sort(want)
	if !slices.Equal(keys(got), want) {
		t.Errorf("connus = %v, attendu %v", keys(got), want)
	}
}

func orphanCounter() int64 {
	return observability.LoadCounterT(ctxkeys.TitleSlug(context.Background()), OrphanEnrichmentsCounter)
}

// TestLoad_RegimeNormalIdentiqueALAncienneUnion : quand chaque enrichissement a son match au
// registre (régime normal), l'ensemble connu est EXACTEMENT l'ancienne union
// enrichissements ∪ participants du xuid, et aucun orphelin n'est compté.
func TestLoad_RegimeNormalIdentiqueALAncienneUnion(t *testing.T) {
	enriched := []string{"m1", "m2", "m3"}
	participants := []string{"m1", "m2", "m3", "m4"} // m4 : inséré par un coéquipier, pas encore enrichi
	playerDB := newPlayerDB(t, enriched...)
	sharedDB := newSharedDB(t, sharedFixture{
		registry:     []string{"m1", "m2", "m3", "m4", "x1"},
		participants: map[string][]string{xuidJoueur: participants, xuidAutre: {"x1", "m1"}},
	})
	avant := orphanCounter()

	known, err := Load(context.Background(), playerDB, sharedDB, xuidJoueur)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	ancienneUnion := slices.Compact(slices.Sorted(slices.Values(append(slices.Clone(enriched), participants...))))
	assertKnown(t, known, ancienneUnion...)
	if d := orphanCounter() - avant; d != 0 {
		t.Errorf("orphelins comptés = %d, attendu 0 en régime normal", d)
	}
}

// TestLoad_EnrichissementSansRegistreEstInconnu : un enrichissement dont le match manque au
// registre partagé (base partagée restaurée plus ancienne) n'est PAS connu — il sera
// re-récupéré — et il est compté.
func TestLoad_EnrichissementSansRegistreEstInconnu(t *testing.T) {
	playerDB := newPlayerDB(t, "m1", "orphelin-1", "orphelin-2")
	sharedDB := newSharedDB(t, sharedFixture{
		registry:     []string{"m1"},
		participants: map[string][]string{xuidJoueur: {"m1"}},
	})
	avant := orphanCounter()

	known, err := Load(context.Background(), playerDB, sharedDB, xuidJoueur)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	assertKnown(t, known, "m1")
	if d := orphanCounter() - avant; d != 2 {
		t.Errorf("orphelins comptés = %d, attendu 2", d)
	}
}

// TestLoad_RegistreSansParticipantDuXuidResteConnu : un match enrichi présent au registre mais
// sans ligne de participant pour ce xuid reste connu — la persistance le sauterait de toute
// façon (idempotence sur match_registry), le re-récupérer à chaque cycle ne réparerait rien.
func TestLoad_RegistreSansParticipantDuXuidResteConnu(t *testing.T) {
	playerDB := newPlayerDB(t, "m1", "r1")
	sharedDB := newSharedDB(t, sharedFixture{
		registry:     []string{"m1", "r1"},
		participants: map[string][]string{xuidJoueur: {"m1"}, xuidAutre: {"r1"}},
	})
	avant := orphanCounter()

	known, err := Load(context.Background(), playerDB, sharedDB, xuidJoueur)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	assertKnown(t, known, "m1", "r1")
	if d := orphanCounter() - avant; d != 0 {
		t.Errorf("orphelins comptés = %d, attendu 0 (r1 est au registre)", d)
	}
}

// TestLoad_ParticipantSansRegistreEstInconnu : une ligne de participant dont le match manque
// au registre n'est pas « dans la base partagée » au sens de la persistance.
func TestLoad_ParticipantSansRegistreEstInconnu(t *testing.T) {
	sharedDB := newSharedDB(t, sharedFixture{
		registry:     []string{"m1"},
		participants: map[string][]string{xuidJoueur: {"m1", "p1"}},
	})
	known, err := Load(context.Background(), newPlayerDB(t), sharedDB, xuidJoueur)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	assertKnown(t, known, "m1")
}

// TestLoad_IsolationEntreXUID : les participants d'un autre joueur ne sont jamais connus.
func TestLoad_IsolationEntreXUID(t *testing.T) {
	sharedDB := newSharedDB(t, sharedFixture{
		registry:     []string{"m1", "x1"},
		participants: map[string][]string{xuidJoueur: {"m1"}, xuidAutre: {"x1"}},
	})
	known, err := Load(context.Background(), nil, sharedDB, xuidJoueur)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	assertKnown(t, known, "m1")
}

// TestLoad_VerificationDuRegistreParPaquets : plus de registryChunk enrichissements hors
// participants, dont la moitié au registre — le découpage des requêtes IN ne perd ni n'ajoute
// aucun match.
func TestLoad_VerificationDuRegistreParPaquets(t *testing.T) {
	const n = 2*registryChunk + 37
	var enriched, registry []string
	for i := range n {
		id := fmt.Sprintf("e-%04d", i)
		enriched = append(enriched, id)
		if i%2 == 0 {
			registry = append(registry, id)
		}
	}
	playerDB := newPlayerDB(t, enriched...)
	sharedDB := newSharedDB(t, sharedFixture{registry: registry})
	avant := orphanCounter()

	known, err := Load(context.Background(), playerDB, sharedDB, xuidJoueur)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	assertKnown(t, known, registry...)
	if d := orphanCounter() - avant; d != int64(n-len(registry)) {
		t.Errorf("orphelins comptés = %d, attendu %d", d, n-len(registry))
	}
}

// TestLoad_EnrichissementsIllisiblesTolere : base joueur neuve (vue absente) — la règle ne
// dépend que de la base partagée, le chargement aboutit.
func TestLoad_EnrichissementsIllisiblesTolere(t *testing.T) {
	sharedDB := newSharedDB(t, sharedFixture{
		registry:     []string{"m1"},
		participants: map[string][]string{xuidJoueur: {"m1"}},
	})
	known, err := Load(context.Background(), openMem(t), sharedDB, xuidJoueur)
	if err != nil {
		t.Fatalf("Load: %v (base joueur sans vue tolérée)", err)
	}
	assertKnown(t, known, "m1")
}

// TestLoad_BasePartageeIllisibleEstFatale : connexion absente, tables absentes ou registre
// absent → ErrSharedUnreadable et aucun ensemble (ni vide, ni « enrichissements seuls »).
func TestLoad_BasePartageeIllisibleEstFatale(t *testing.T) {
	playerDB := newPlayerDB(t, "m1", "m2")
	sansRegistre := openMem(t)
	exec(t, sansRegistre, `CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR)`)
	exec(t, sansRegistre, `INSERT INTO match_participants VALUES ('m1', ?)`, xuidJoueur)
	fermee := openMem(t)
	_ = fermee.Close()

	cas := map[string]*sql.DB{
		"connexion absente": nil,
		"base vide":         openMem(t),
		"registre absent":   sansRegistre,
		"connexion fermée":  fermee,
	}
	for nom, sharedDB := range cas {
		t.Run(nom, func(t *testing.T) {
			known, err := Load(context.Background(), playerDB, sharedDB, xuidJoueur)
			if !errors.Is(err, ErrSharedUnreadable) {
				t.Fatalf("err = %v, attendu ErrSharedUnreadable", err)
			}
			if known != nil {
				t.Errorf("connus = %v, attendu nil (aucun ensemble partiel)", keys(known))
			}
		})
	}
}

// TestLoad_XUIDVideEstFatal : sans xuid, les participants ne peuvent pas être bornés au joueur.
func TestLoad_XUIDVideEstFatal(t *testing.T) {
	sharedDB := newSharedDB(t, sharedFixture{registry: []string{"m1"}})
	for _, xuid := range []string{"", "   "} {
		if _, err := Load(context.Background(), newPlayerDB(t, "m1"), sharedDB, xuid); !errors.Is(err, ErrNoXUID) {
			t.Errorf("xuid %q : err = %v, attendu ErrNoXUID", xuid, err)
		}
	}
}
