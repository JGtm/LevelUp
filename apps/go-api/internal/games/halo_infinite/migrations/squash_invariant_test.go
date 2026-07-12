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
// MODE 5-CIBLES (metadata/shared/pve/social/player) : provisionFullHistory ==
// provisionCandidate → A == B via le runner ACTIF. Test de fumée : le provisioning des
// 5 cibles reste sans erreur et self-consistent. La morsure est prouvée par
// TestSquashInvariant_BiteProof.
//
// SQUASH v1 LANDÉ (chantier N4, 2026-07-12) — cible PLAYER : le bloc contigu title-owned
// create_base_player_schema..player_append_only_csr_snapshots_v1 (33 steps, bornes M3a)
// est remplacé par create_baseline_player_v1. La PREUVE zéro-perte est
// TestSquashInvariant_PlayerBaselineEquivalent : SchemaSnapshot(baseline) == GOLDEN
// (testdata/squash/player_block_golden.snapshot), le golden ayant été capturé de
// l'HISTORIQUE réel (les 33 steps) AVANT leur retrait — oracle indépendant de la
// baseline (« fixture des steps squashés » branché comme historique complet). Comme les
// steps player POSTÉRIEURS à la borne sont inchangés (code byte-identique) et que la
// substitution baseline↔bloc opère sur un schéma d'entrée identique, l'égalité au niveau
// bloc IMPLIQUE l'égalité du provisioning player complet (preuve compositionnelle).
// Déviation SEAM documentée (comme M2a) : le bloc étant ARCHIVÉ/retiré du code actif, son
// rejeu live n'existe plus — le golden figé EST l'oracle historique.

import (
	"database/sql"
	"os"
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

// goldenPlayerBlock lit le snapshot golden du bloc squashé (historique réel, figé).
func goldenPlayerBlock(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("testdata/squash/player_block_golden.snapshot")
	if err != nil {
		t.Fatalf("lecture golden bloc player: %v", err)
	}
	return string(b)
}

// TestSquashInvariant_PlayerBaselineEquivalent — PREUVE ZÉRO-PERTE du squash player v1.
// SchemaSnapshot(create_baseline_player_v1 seule) == golden(33 steps historiques),
// octet pour octet. Le golden est l'oracle indépendant (capturé de l'historique avant
// retrait). VERT = le schéma « à plat » de la baseline reproduit EXACTEMENT le schéma
// cumulé du bloc → aucune table/colonne/contrainte/index/séquence perdue.
func TestSquashInvariant_PlayerBaselineEquivalent(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migration.RunSteps(db, migration.TargetPlayer, playerBaselineSteps()); err != nil {
		t.Fatalf("RunSteps(baseline): %v", err)
	}
	got := snapshotOrFatal(t, db)
	if want := goldenPlayerBlock(t); got != want {
		t.Errorf("baseline player v1 != historique (33 steps) — PERTE DE SCHÉMA\n%s", firstDiff(want, got))
	}
}

// TestSquashInvariant_PlayerBaselineBiteProof — la comparaison au golden MORD : une
// colonne ajoutée au schéma de la baseline fait diverger du golden. Sans ça, un golden
// accidentellement égal ne prouverait rien.
func TestSquashInvariant_PlayerBaselineBiteProof(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migration.RunSteps(db, migration.TargetPlayer, playerBaselineSteps()); err != nil {
		t.Fatalf("RunSteps(baseline): %v", err)
	}
	if _, err := db.Exec(`ALTER TABLE player_match_enrichment ADD COLUMN squash_bite_probe INTEGER`); err != nil {
		t.Fatalf("mutation sonde: %v", err)
	}
	if snapshotOrFatal(t, db) == goldenPlayerBlock(t) {
		t.Errorf("morsure absente : la comparaison baseline↔golden ne détecte pas une colonne ajoutée")
	}
}

// TestSquashInvariant_PlayerSquashedStepsRemoved — GARDE-FOU anti-régression : aucun
// des 33 steps squashés ne doit réapparaître dans le runner ACTIF (StepsFor player ni
// canonicalOrder) ; leur schéma vit exclusivement dans la baseline. Empêche une
// ré-introduction accidentelle (qui rejouerait un DDL déjà couvert et fausserait le boot).
func TestSquashInvariant_PlayerSquashedStepsRemoved(t *testing.T) {
	squashed := make(map[string]bool, len(squashedPlayerV1Steps))
	for _, n := range squashedPlayerV1Steps {
		squashed[n] = true
	}
	for _, m := range StepsFor(migration.TargetPlayer) {
		if squashed[m.Name] {
			t.Errorf("step squashé %q toujours présent dans StepsFor(player) — doit vivre dans la baseline", m.Name)
		}
	}
	for _, n := range migration.CanonicalOrder() {
		if squashed[n] {
			t.Errorf("step squashé %q toujours dans canonicalOrder — doit être retiré (baseline le remplace)", n)
		}
	}
	// La sentinelle DM-5 (dernier step squashé) doit être le dernier de la liste.
	if got := squashedPlayerV1Steps[len(squashedPlayerV1Steps)-1]; got != "player_append_only_csr_snapshots_v1" {
		t.Errorf("sentinelle DM-5 = %q, attendu player_append_only_csr_snapshots_v1", got)
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
