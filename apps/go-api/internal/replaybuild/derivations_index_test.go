package replaybuild

// derivations_index_test.go — LE PRÉDICAT DE FRAÎCHEUR DES DÉRIVÉS, cas par cas.
//
// Ce qui est éprouvé : `DerivationsUpToDate` répond FAUX dans les quatre situations qui
// appellent toutes la même conduite — rejouer les dérivations — et VRAI dans la seule qui
// autorise à passer. Chacun des quatre faux est le trou du constat A2 sous un angle différent.

import (
	"os"
	"path/filepath"
	"testing"
)

// artefactDe pose un artefact de contenu donné et rend son chemin.
func artefactDe(t *testing.T, contenu string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "abcd1234.json")
	if err := os.WriteFile(p, []byte(contenu), 0o600); err != nil {
		t.Fatalf("write artefact: %v", err)
	}
	return p
}

func TestDerivationsUpToDate(t *testing.T) {
	t.Run("artefact absent", func(t *testing.T) {
		if DerivationsUpToDate(filepath.Join(t.TempDir(), "jamais.json")) {
			t.Error("un artefact absent n'a rien de dérivé — la cuisson doit passer d'abord")
		}
	})

	t.Run("artefact vide", func(t *testing.T) {
		if DerivationsUpToDate(artefactDe(t, "")) {
			t.Error("un artefact de taille nulle n'est pas un artefact")
		}
	})

	t.Run("marque absente", func(t *testing.T) {
		if DerivationsUpToDate(artefactDe(t, `{"schemaVersion":39}`)) {
			t.Error("sans marque, les dérivés n'ont jamais été écrits — c'est LE trou du " +
				"constat A2 (le prédicat d'origine tenait la présence du fichier pour suffisante)")
		}
	})

	t.Run("marque illisible", func(t *testing.T) {
		p := artefactDe(t, `{"schemaVersion":39}`)
		if err := os.WriteFile(DerivationsMarkPath(p), []byte("{pas du json"), 0o600); err != nil {
			t.Fatal(err)
		}
		if DerivationsUpToDate(p) {
			t.Error("une marque illisible doit se lire « pas dérivé », jamais « erreur »")
		}
	})

	t.Run("marque a la revision courante", func(t *testing.T) {
		p := artefactDe(t, `{"schemaVersion":39}`)
		st, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		if err := WriteDerivationsMark(p, 39, int(st.Size())); err != nil {
			t.Fatalf("WriteDerivationsMark: %v", err)
		}
		if !DerivationsUpToDate(p) {
			t.Error("un artefact fraîchement dérivé doit être à jour — sinon le rattrapage " +
				"ne converge jamais et rejoue les mêmes cinq artefacts à chaque cycle")
		}
	})

	t.Run("artefact re-cuit depuis la marque", func(t *testing.T) {
		p := artefactDe(t, `{"schemaVersion":39}`)
		st, _ := os.Stat(p)
		if err := WriteDerivationsMark(p, 39, int(st.Size())); err != nil {
			t.Fatal(err)
		}
		// MÊME chemin, contenu DIFFÉRENT (donc taille différente) : la marque ne colle plus.
		if err := os.WriteFile(p, []byte(`{"schemaVersion":40,"tracks":[]}`), 0o600); err != nil {
			t.Fatal(err)
		}
		if DerivationsUpToDate(p) {
			t.Error("un artefact re-cuit doit redemander ses dérivations")
		}
	})

	t.Run("revision anterieure", func(t *testing.T) {
		p := artefactDe(t, `{"schemaVersion":39}`)
		st, _ := os.Stat(p)
		if err := ecrireMarqueRev(p, "derivations-1970-01-01", int(st.Size())); err != nil {
			t.Fatal(err)
		}
		if DerivationsUpToDate(p) {
			t.Error("une marque d'une révision antérieure doit faire rejouer les dérivations — " +
				"c'est ce qui permet de réparer le parc après une correction de projection")
		}
	})
}

// TestRemoveDerivationsMark : le geste « redemande les dérivations de ce match ».
func TestRemoveDerivationsMark(t *testing.T) {
	p := artefactDe(t, `{"schemaVersion":39}`)
	st, _ := os.Stat(p)
	if err := WriteDerivationsMark(p, 39, int(st.Size())); err != nil {
		t.Fatal(err)
	}
	if err := RemoveDerivationsMark(p); err != nil {
		t.Fatalf("RemoveDerivationsMark: %v", err)
	}
	if DerivationsUpToDate(p) {
		t.Error("la marque effacée, l'artefact doit redemander ses dérivations")
	}
	// Effacer deux fois n'est pas une erreur.
	if err := RemoveDerivationsMark(p); err != nil {
		t.Errorf("second RemoveDerivationsMark = %v, attendu nil", err)
	}
}

// TestDerivationsMarkPath : la marque vit À CÔTÉ de l'artefact, pas ailleurs.
func TestDerivationsMarkPath(t *testing.T) {
	got := DerivationsMarkPath(filepath.Join("data", "cache", "replays", "halo_infinite", "abcd1234.json"))
	want := filepath.Join("data", "cache", "replays", "halo_infinite", "abcd1234.derived.json")
	if got != want {
		t.Errorf("DerivationsMarkPath = %q, attendu %q", got, want)
	}
}

// ecrireMarqueRev pose une marque avec une révision ARBITRAIRE (le chemin de production n'écrit
// que la révision courante — ce test a besoin d'une autre pour prouver que la comparaison mord).
func ecrireMarqueRev(artifactPath, rev string, octets int) error {
	blob := []byte(`{"rev":"` + rev + `","artifactBytes":` + itoa(octets) + `,"artifactSchema":39}`)
	return os.WriteFile(DerivationsMarkPath(artifactPath), blob, 0o600)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
