package ops

// seed_demo_weapon_kills_test.go — `weapon_kills` dans le seed de la démo (lot L8-6 du plan
// sécurité du 2026-10-15).
//
// La table n'existe plus que dans le fichier des titres SANS décodeur de film (migration
// title-owned shared_drop_weapon_kills_v1 : Halo Infinite l'a perdue, Halo 5 la garde avec
// ses lignes natives). Le seed :
//   - ne tente d'extraire la table que pour un titre qui la porte (capability CapReplay =
//     décodeur de film, même critère que l'export de snapshot) — plus de ligne WARN « table
//     source absente » à chaque seed d'un titre à film ;
//   - ne crée v_weapon_kills que si la table existe, et RENVOIE l'échec de création quand elle
//     existe (l'erreur était avalée : `_, _ = db.ExecContext(...)`, anti-pattern « swallowed
//     error »).

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
)

func hasExtract(tables []extractTable, name string) bool {
	for _, t := range tables {
		if t.name == name {
			return true
		}
	}
	return false
}

func TestTitleOwnsWeaponKills_SuitLeDecodeurDeFilm(t *testing.T) {
	avecFilm := &titlePkg.TitleDescriptor{Slug: "avec_film", Capabilities: []titlePkg.Capability{titlePkg.CapReplay}}
	sansFilm := &titlePkg.TitleDescriptor{Slug: "sans_film", Capabilities: []titlePkg.Capability{titlePkg.CapAchievements}}
	if titleOwnsWeaponKills(avecFilm) {
		t.Error("titre à décodeur de film : weapon_kills n'existe plus dans son fichier")
	}
	if !titleOwnsWeaponKills(sansFilm) {
		t.Error("titre sans décodeur de film : il garde weapon_kills")
	}
	if !titleOwnsWeaponKills(nil) {
		t.Error("titre inconnu du registre : la table est tentée (l'absence source reste tolérée)")
	}
}

func TestSharedTablesFor_WeaponKillsParTitre(t *testing.T) {
	// Registre par défaut (built-in) : Halo Infinite porte CapReplay, donc plus de weapon_kills.
	if hasExtract(sharedTablesFor(titlePkg.DefaultSlug), "weapon_kills") {
		t.Errorf("%s : weapon_kills n'existe plus dans son fichier, elle ne doit pas être extraite", titlePkg.DefaultSlug)
	}
	if !hasExtract(sharedTablesFor("titre_inconnu"), "weapon_kills") {
		t.Error("titre inconnu : weapon_kills est tentée")
	}
	// Le socle commun reste complet quel que soit le titre.
	for _, slug := range []string{titlePkg.DefaultSlug, "titre_inconnu"} {
		for _, base := range sharedTablesWhere {
			if !hasExtract(sharedTablesFor(slug), base.name) {
				t.Errorf("%s : %s manque à la liste d'extraction", slug, base.name)
			}
		}
	}
	// La liste partagée n'est jamais modifiée par l'appel.
	n := len(sharedTablesWhere)
	_ = sharedTablesFor("titre_inconnu")
	if len(sharedTablesWhere) != n || hasExtract(sharedTablesWhere, "weapon_kills") {
		t.Error("sharedTablesFor a modifié sharedTablesWhere")
	}
}

// vueSharedPrerequis crée les relations que lisent v_gamertag_lookup et v_match_full.
func vueSharedPrerequis(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, q := range []string{
		`CREATE TABLE match_registry (match_id VARCHAR)`,
		`CREATE TABLE xuid_aliases (xuid VARCHAR, gamertag VARCHAR)`,
		`CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR, gamertag VARCHAR)`,
		`CREATE TABLE match_kill_events_latest (feed_killer_xuid VARCHAR, feed_killer_gamertag VARCHAR,
			victim_xuid VARCHAR, victim_gamertag VARCHAR)`,
		`CREATE TABLE killer_victim_pairs (killer_xuid VARCHAR, killer_gamertag VARCHAR,
			victim_xuid VARCHAR, victim_gamertag VARCHAR)`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("prérequis %q : %v", q, err)
		}
	}
}

func vueExiste(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM duckdb_views() WHERE view_name = ?`, name).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n > 0
}

func TestRecreateSharedViews_WeaponKills(t *testing.T) {
	ctx := context.Background()

	t.Run("table presente: la vue est creee", func(t *testing.T) {
		db, err := sql.Open("duckdb", "")
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		vueSharedPrerequis(t, db)
		if _, err := db.Exec(`CREATE TABLE weapon_kills (match_id VARCHAR, xuid VARCHAR, weapon_id UBIGINT,
			reconciled_as UBIGINT, generation_id BIGINT)`); err != nil {
			t.Fatal(err)
		}
		if err := recreateSharedViews(ctx, db); err != nil {
			t.Fatalf("recreateSharedViews : %v", err)
		}
		if !vueExiste(t, db, "v_weapon_kills") {
			t.Error("v_weapon_kills absente alors que la table existe")
		}
	})

	t.Run("table absente: pas de vue, pas d'erreur", func(t *testing.T) {
		db, err := sql.Open("duckdb", "")
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		vueSharedPrerequis(t, db)
		if err := recreateSharedViews(ctx, db); err != nil {
			t.Fatalf("recreateSharedViews sans weapon_kills : %v", err)
		}
		if vueExiste(t, db, "v_weapon_kills") {
			t.Error("v_weapon_kills créée sans table source")
		}
	})

	t.Run("table presente mais vue impossible: l'erreur remonte", func(t *testing.T) {
		db, err := sql.Open("duckdb", "")
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		vueSharedPrerequis(t, db)
		// Sans generation_id ni reconciled_as : la vue ne peut pas être définie.
		if _, err := db.Exec(`CREATE TABLE weapon_kills (match_id VARCHAR, xuid VARCHAR, weapon_id UBIGINT)`); err != nil {
			t.Fatal(err)
		}
		err = recreateSharedViews(ctx, db)
		if err == nil || !strings.Contains(err.Error(), "v_weapon_kills") {
			t.Fatalf("erreur = %v, attendu un échec nommant v_weapon_kills (jusqu'ici avalé)", err)
		}
	})
}
