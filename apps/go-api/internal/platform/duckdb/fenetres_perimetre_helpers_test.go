package duckdb

// fenetres_perimetre_helpers_test.go — OUTIL DE TEST (lot L5a du plan perf, 2026-09-23) :
// NOTER les requetes qu'un lecteur execute, puis les REJOUER sous
// `EXPLAIN (ANALYZE, FORMAT JSON)` pour lire le nombre de lignes que chaque FENETRE a vues.
//
// # POURQUOI LES FENETRES
//
// Les vues `_latest` du film (journal des morts, positions, entames, contexte des morts)
// portent une fenetre `QUALIFY ... OVER (PARTITION BY match_id ...)`. DuckDB n'y pousse qu'un
// filtre CONSTANT sur la cle de partition : une semi-jointure sur une sous-requete, ou un
// filtre pose sur une AUTRE vue de la jointure, laisse la fenetre se calculer sur la table
// ENTIERE. La page paie alors l'historique complet pour afficher six matchs, sans qu'aucun
// chiffre ne change — donc sans qu'aucun test de valeur ne le voie (constat du plan perf,
// 2026-09-23 : 1,7 s pour 6 matchs comme pour 1 158).
//
// Le nombre de lignes entrees dans chaque operateur WINDOW est LA mesure de « cette lecture
// est bornee a son perimetre ». Contrairement a un chronometre, il ne depend ni de la machine
// ni de la charge : un test peut l'exiger.
//
// # POURQUOI NOTER LES REQUETES PLUTOT QUE LES RECONSTRUIRE
//
// Un test qui recomposerait le SQL a partir des constantes verifierait le constructeur, pas le
// lecteur : le jour ou une methode enverrait une autre requete, il resterait vert. Ici, c'est
// ce que la methode a REELLEMENT envoye qui est rejoue.

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	duckdbdrv "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

// requeteNotee : une requete de lecture telle que le lecteur l'a envoyee.
type requeteNotee struct {
	sql  string
	args []driver.NamedValue
}

// carnetRequetes : les lectures notees, dans l'ordre.
type carnetRequetes struct {
	mu       sync.Mutex
	requetes []requeteNotee
}

func (c *carnetRequetes) noter(q string, args []driver.NamedValue) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.requetes = append(c.requetes, requeteNotee{sql: q, args: append([]driver.NamedValue(nil), args...)})
}

// vider rend les requetes notees depuis le dernier appel, et repart de zero.
func (c *carnetRequetes) vider() []requeteNotee {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := c.requetes
	c.requetes = nil
	return out
}

type connecteurNoteur struct {
	inner  driver.Connector
	carnet *carnetRequetes
}

func (c connecteurNoteur) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.inner.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &connNoteuse{Conn: conn, carnet: c.carnet}, nil
}

func (c connecteurNoteur) Driver() driver.Driver { return c.inner.Driver() }

// connNoteuse note chaque LECTURE (QueryContext) ; les ecritures du corpus passent sans trace.
type connNoteuse struct {
	driver.Conn
	carnet *carnetRequetes
}

func (c *connNoteuse) QueryContext(ctx context.Context, q string, args []driver.NamedValue) (driver.Rows, error) {
	c.carnet.noter(q, args)
	return c.Conn.(driver.QueryerContext).QueryContext(ctx, q, args)
}

func (c *connNoteuse) ExecContext(ctx context.Context, q string, args []driver.NamedValue) (driver.Result, error) {
	return c.Conn.(driver.ExecerContext).ExecContext(ctx, q, args)
}

func (c *connNoteuse) CheckNamedValue(nv *driver.NamedValue) error {
	if ch, ok := c.Conn.(driver.NamedValueChecker); ok {
		return ch.CheckNamedValue(nv)
	}
	return driver.ErrSkip
}

// baseNotee : une base shared EN MEMOIRE, migree, dont les lectures du PlayerDB sont notees,
// et un acces BRUT (non note) a la meme base pour rejouer les requetes.
type baseNotee struct {
	pdb    *PlayerDB
	carnet *carnetRequetes
	brute  *sql.DB
}

// newBaseNotee monte la base. Le PlayerDB porte la metadata de test du lecteur tactique
// (traductions de cartes), inoffensive pour les autres lecteurs.
func newBaseNotee(t *testing.T) baseNotee {
	t.Helper()
	connector, err := duckdbdrv.NewConnector(":memory:", nil)
	if err != nil {
		t.Fatalf("NewConnector: %v", err)
	}
	carnet := &carnetRequetes{}
	notee := sql.OpenDB(connecteurNoteur{inner: connector, carnet: carnet})
	brute := sql.OpenDB(connector)
	t.Cleanup(func() {
		_ = notee.Close()
		_ = brute.Close()
		_ = connector.Close()
	})
	_ = migration.All()
	if err := migration.RunForDB(brute, migration.TargetShared); err != nil {
		t.Fatalf("RunForDB(Shared): %v", err)
	}
	shared := newTestDB(notee, ":memory:")
	pdb := &PlayerDB{
		Shared:       shared,
		SharedReader: LegacySharedReader(shared),
		Metadata:     newTacticalTestMetadata(t),
		XUID:         tacXUIDMoi,
		TitleSlug:    "halo_infinite",
	}
	carnet.vider()
	return baseNotee{pdb: pdb, carnet: carnet, brute: brute}
}

// fenetresVues rejoue `r` sous EXPLAIN ANALYZE et rend le nombre de lignes vues par CHAQUE
// operateur de fenetre du plan (vide si la requete n'en porte aucune).
func fenetresVues(t *testing.T, db *sql.DB, r requeteNotee) []int {
	t.Helper()
	args := make([]any, 0, len(r.args))
	for _, a := range r.args {
		args = append(args, a.Value)
	}
	rows, err := db.QueryContext(context.Background(), "EXPLAIN (ANALYZE, FORMAT JSON) "+r.sql, args...)
	if err != nil {
		t.Fatalf("EXPLAIN ANALYZE %.80s... : %v", r.sql, err)
	}
	defer func() { _ = rows.Close() }()
	var plan string
	for rows.Next() {
		var cle, valeur sql.NullString
		if err := rows.Scan(&cle, &valeur); err != nil {
			t.Fatalf("scan du plan : %v", err)
		}
		plan += valeur.String
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("plan : %v", err)
	}
	var racine map[string]any
	if err := json.Unmarshal([]byte(plan), &racine); err != nil {
		t.Fatalf("plan JSON illisible : %v\n%.400s", err, plan)
	}
	var out []int
	collecterFenetres(racine, &out)
	return out
}

// collecterFenetres parcourt l'arbre du profil et note la cardinalite de chaque WINDOW.
func collecterFenetres(noeud map[string]any, out *[]int) {
	if typ, _ := noeud["operator_type"].(string); strings.Contains(typ, "WINDOW") {
		if n, ok := noeud["operator_cardinality"].(float64); ok {
			*out = append(*out, int(n))
		}
	}
	enfants, _ := noeud["children"].([]any)
	for _, e := range enfants {
		if m, ok := e.(map[string]any); ok {
			collecterFenetres(m, out)
		}
	}
}

// exigerFenetresBornees rejoue chaque requete notee et echoue si UNE fenetre a vu plus de
// `borne` lignes. `min` : au moins ce nombre de requetes doivent porter une fenetre — sans
// quoi le test ne prouverait rien (une requete qui ne lit plus aucune vue `_latest`).
func exigerFenetresBornees(t *testing.T, b baseNotee, lecture string, borne, minAvecFenetre int) {
	t.Helper()
	requetes := b.carnet.vider()
	avecFenetre := 0
	for _, r := range requetes {
		fenetres := fenetresVues(t, b.brute, r)
		if len(fenetres) > 0 {
			avecFenetre++
		}
		for _, n := range fenetres {
			if n > borne {
				t.Errorf("%s : une fenetre `_latest` a vu %d lignes, borne %d — la lecture n'est pas "+
					"restreinte a son perimetre :\n%.300s", lecture, n, borne, r.sql)
			}
		}
	}
	if avecFenetre < minAvecFenetre {
		t.Fatalf("%s : %d requete(s) avec fenetre sur %d notees, want >= %d — le test ne mesure rien",
			lecture, avecFenetre, len(requetes), minAvecFenetre)
	}
}
