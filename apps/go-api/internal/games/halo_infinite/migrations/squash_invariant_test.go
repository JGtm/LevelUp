//go:build integration

package migrations

// squash_invariant_test.go — VERROU CENTRAL du chantier squash (plan M2). Prouve que
// le schéma produit par (baseline + steps restants) est BIT-IDENTIQUE au schéma
// produit par (historique complet des migrations), cible par cible.
//
// EMPLACEMENT (déviation documentée du plan M2a qui citait
// internal/migration/squash_invariant_test.go) : le provisioning COMPLET d'une cible
// exige les steps title-owned (StepsFor) — que le package migration ne peut pas
// importer (cycle). Comme order_audit_test.go, l'invariant vit donc dans le package
// titre, seul endroit voyant à la fois migration.* ET StepsFor.
//
// MODE ACTUEL (aucun squash landé) : provisionFullHistory == provisionCandidate →
// A == B trivialement (« harnais prêt », M2a). Le SEAM est en place : quand le 1er
// squash landera (M3), provisionFullHistory rejouera le FIXTURE des steps squashés +
// le reste (oracle), tandis que provisionCandidate rejouera la baseline + le reste ;
// l'égalité stricte des snapshots deviendra alors la preuve zéro-perte (M3d).
//
// La morsure de la comparaison (qu'elle détecte réellement un écart) est prouvée par
// TestSquashInvariant_BiteProof — sans quoi un A==B trivial ne prouverait rien.

import (
	"database/sql"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

// invariantTargets : toutes les cibles du titre par défaut provisionnées au boot /
// à la demande. Player inclus (provisionné par joueur).
var invariantTargets = []migration.TargetDB{
	migration.TargetMetadata,
	migration.TargetShared,
	migration.TargetSharedPvE,
	migration.TargetSharedSocial,
	migration.TargetPlayer,
}

// freshProvision provisionne une DB :memory: vierge pour target via le chemin réel
// du runner (registre actif + steps title-owned).
func freshProvision(t *testing.T, target migration.TargetDB) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	migration.SetTitleStepsProvider(StepsFor)
	if err := migration.RunForDB(db, target); err != nil {
		t.Fatalf("RunForDB(%s): %v", target, err)
	}
	return db
}

// provisionFullHistory = ORACLE : le schéma cumulé de TOUT l'historique de migrations
// de la cible. AUJOURD'HUI identique à provisionCandidate (aucun squash actif). SEAM
// M3 : y préfixer le fixture des steps squashés quand une baseline landera.
func provisionFullHistory(t *testing.T, target migration.TargetDB) *sql.DB {
	return freshProvision(t, target)
}

// provisionCandidate = chemin CANDIDAT : ce que le runner ACTIF produit. AUJOURD'HUI
// == full history ; POST-M3 == baseline + steps restants.
func provisionCandidate(t *testing.T, target migration.TargetDB) *sql.DB {
	return freshProvision(t, target)
}

func snapshotOrFatal(t *testing.T, db *sql.DB) string {
	t.Helper()
	s, err := migration.SchemaSnapshot(db)
	if err != nil {
		t.Fatalf("SchemaSnapshot: %v", err)
	}
	return s
}

// TestSquashInvariant_SchemaBitIdentical : pour chaque cible, snapshot(historique
// complet) == snapshot(candidat), octet pour octet. VERT en mode A=B (harnais prêt) ;
// devient la preuve zéro-perte quand un squash sépare les deux chemins (M3d).
func TestSquashInvariant_SchemaBitIdentical(t *testing.T) {
	for _, target := range invariantTargets {
		target := target
		t.Run(string(target), func(t *testing.T) {
			full := snapshotOrFatal(t, provisionFullHistory(t, target))
			cand := snapshotOrFatal(t, provisionCandidate(t, target))
			if full != cand {
				t.Errorf("cible %s : schéma (historique) != schéma (candidat)\n%s",
					target, firstDiff(full, cand))
			}
		})
	}
}

// TestSquashInvariant_BiteProof : preuve que la comparaison MORD. Deux provisionings
// candidats identiques, puis une colonne ajoutée à l'un → les snapshots DOIVENT
// diverger. Sans ce test, un A==B trivial (mode harnais) ne garantirait pas que
// l'invariant détecte un vrai écart de schéma.
func TestSquashInvariant_BiteProof(t *testing.T) {
	target := migration.TargetPlayer
	ref := snapshotOrFatal(t, provisionCandidate(t, target))

	mutated := provisionCandidate(t, target)
	if _, err := mutated.Exec(
		`ALTER TABLE player_match_enrichment ADD COLUMN squash_bite_probe INTEGER`,
	); err != nil {
		t.Fatalf("mutation sonde: %v", err)
	}
	if ref == snapshotOrFatal(t, mutated) {
		t.Errorf("morsure absente : l'invariant ne détecte pas une colonne ajoutée")
	}
}

// firstDiff retourne un extrait lisible du 1er écart entre deux snapshots
// (numéro de ligne + lignes divergentes), pour un diagnostic de section rapide.
func firstDiff(a, b string) string {
	la, lb := strings.Split(a, "\n"), strings.Split(b, "\n")
	n := len(la)
	if len(lb) < n {
		n = len(lb)
	}
	for i := 0; i < n; i++ {
		if la[i] != lb[i] {
			return "1er écart ligne " + itoa(i+1) + ":\n  historique: " + la[i] + "\n  candidat  : " + lb[i]
		}
	}
	if len(la) != len(lb) {
		return "longueurs différentes: historique=" + itoa(len(la)) + " candidat=" + itoa(len(lb))
	}
	return "(aucun écart ligne à ligne — divergence hors zone comparée)"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
