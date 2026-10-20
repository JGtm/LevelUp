package main

// provision_title_errors_test.go — la boucle de provisioning d'un titre additionnel
// TRAITE TOUTES SES BASES. Tests purs (aucune DuckDB) : ils exercent la mécanique de
// collecte d'erreurs, pas les migrations.
//
// Régression couverte : entre le 2026-09-12 et le 2026-09-20, l'échec de la metadata de
// halo_5 faisait retourner la boucle immédiatement, donc les migrations shared et social
// de ce titre n'étaient plus jouées à aucun boot — en silence, sous une seule ligne ERROR.

import (
	"errors"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
)

func testProvisionTargets() []provisionTarget {
	return []provisionTarget{
		{"/data/meta.duckdb", migration.TargetMetadata},
		{"/data/shared.duckdb", migration.TargetShared},
		{"/data/social.duckdb", migration.TargetSharedSocial},
	}
}

// TestProvisionAllTargets_EchecDeLaPremiereNEmpechePasLesSuivantes : c'est LE défaut
// corrigé — la première base en échec ne doit pas emporter les autres.
func TestProvisionAllTargets_EchecDeLaPremiereNEmpechePasLesSuivantes(t *testing.T) {
	var vues []string
	err := provisionAllTargets(t.Context(), "titre_test", testProvisionTargets(),
		func(tg provisionTarget) error {
			vues = append(vues, string(tg.kind))
			if tg.kind == migration.TargetMetadata {
				return errors.New("name_en NOT NULL violé par le seed du registre d'armes")
			}
			return nil
		})

	if err == nil {
		t.Fatal("erreur attendue (la metadata a échoué)")
	}
	if len(vues) != 3 {
		t.Errorf("bases tentées = %v, want les 3 (metadata, shared, shared_social)", vues)
	}
	for _, want := range []string{string(migration.TargetShared), string(migration.TargetSharedSocial)} {
		found := false
		for _, v := range vues {
			if v == want {
				found = true
			}
		}
		if !found {
			t.Errorf("base %s jamais tentée alors que seule la metadata a échoué", want)
		}
	}
}

// TestProvisionAllTargets_ErreurJointeNommeChaqueBase : l'erreur retournée nomme CHAQUE
// base en échec (message + failedProvisionTargets), sinon le log de boot ne dit pas quoi
// réparer.
func TestProvisionAllTargets_ErreurJointeNommeChaqueBase(t *testing.T) {
	err := provisionAllTargets(t.Context(), "titre_test", testProvisionTargets(),
		func(tg provisionTarget) error {
			if tg.kind == migration.TargetShared {
				return nil
			}
			return errors.New("boom " + string(tg.kind))
		})
	if err == nil {
		t.Fatal("erreur attendue (metadata + social ont échoué)")
	}
	for _, want := range []string{string(migration.TargetMetadata), string(migration.TargetSharedSocial)} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("l'erreur jointe ne nomme pas %s : %v", want, err)
		}
	}
	if strings.Contains(err.Error(), "/data/shared.duckdb") {
		t.Errorf("l'erreur jointe nomme une base qui a RÉUSSI : %v", err)
	}
	got := failedProvisionTargets(err)
	want := []string{string(migration.TargetMetadata), string(migration.TargetSharedSocial)}
	if len(got) != len(want) {
		t.Fatalf("failedProvisionTargets = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("failedProvisionTargets[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	// Le chemin de la base fautive reste lisible dans le message (diagnostic de boot).
	if !strings.Contains(err.Error(), "/data/meta.duckdb") {
		t.Errorf("l'erreur jointe ne porte pas le chemin de la base fautive : %v", err)
	}
}

// TestProvisionAllTargets_ToutReussi : aucune erreur, aucune cible oubliée.
func TestProvisionAllTargets_ToutReussi(t *testing.T) {
	n := 0
	err := provisionAllTargets(t.Context(), "titre_test", testProvisionTargets(),
		func(provisionTarget) error { n++; return nil })
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	if n != 3 {
		t.Errorf("bases tentées = %d, want 3", n)
	}
	if got := failedProvisionTargets(nil); got != nil {
		t.Errorf("failedProvisionTargets(nil) = %v, want nil", got)
	}
}

// TestProvisionTargetsFor_PvEGateeParCapability : la DB PvE n'entre dans la liste que si
// le titre déclare Firefight (gating par capability, jamais par comparaison de slug).
func TestProvisionTargetsFor_PvEGateeParCapability(t *testing.T) {
	pr := title.NewPathResolver(t.TempDir())
	sans := &title.TitleDescriptor{Slug: "t_sans", Capabilities: []title.Capability{title.CapMatchmaking}}
	avec := &title.TitleDescriptor{Slug: "t_avec", Capabilities: []title.Capability{title.CapMatchmaking, title.CapFirefight}}

	if got := len(provisionTargetsFor(pr, sans)); got != 3 {
		t.Errorf("cibles sans Firefight = %d, want 3 (metadata, shared, shared_social)", got)
	}
	kinds := map[migration.TargetDB]bool{}
	for _, tg := range provisionTargetsFor(pr, avec) {
		kinds[tg.kind] = true
	}
	if !kinds[migration.TargetSharedPvE] {
		t.Error("cible PvE absente alors que le titre déclare CapFirefight")
	}
	for _, k := range []migration.TargetDB{migration.TargetMetadata, migration.TargetShared, migration.TargetSharedSocial} {
		if !kinds[k] {
			t.Errorf("cible %s absente de la liste", k)
		}
	}
}
