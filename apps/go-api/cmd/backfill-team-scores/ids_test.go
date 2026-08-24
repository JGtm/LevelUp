package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "liste.tsv")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("écriture du fichier de test : %v", err)
	}
	return path
}

func TestLoadMatchIDs(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    []string
	}{
		{
			// La forme réelle du TSV de la phase 1 : en-tête nommé, `match_id` en tête.
			name: "TSV a en-tete, colonne match_id en premier",
			content: "match_id\tgame_variant_name\tdb_t0\tdb_t1\tapi_t0\tapi_t1\n" +
				"7344d24f-0154-4949-80ad-e2b781c122f1\tStrongholds:Arena\t193\t112\t200\t126\n" +
				"606d9844-4f22-42c1-8fb6-e9d541e5ff4c\tKOTH:Arena\t105\t8\t3\t0\n",
			want: []string{"7344d24f-0154-4949-80ad-e2b781c122f1", "606d9844-4f22-42c1-8fb6-e9d541e5ff4c"},
		},
		{
			// La colonne est retrouvée par son NOM : l'ordre du TSV peut changer sans
			// que l'outil se mette à lire des noms de mode comme des match_id.
			name: "colonne match_id ailleurs qu'en premier",
			content: "game_variant_name\tmatch_id\tdb_t0\n" +
				"Strongholds:Arena\tabc-1\t193\n",
			want: []string{"abc-1"},
		},
		{
			name:    "liste nue sans en-tete",
			content: "abc-1\nabc-2\nabc-3\n",
			want:    []string{"abc-1", "abc-2", "abc-3"},
		},
		{
			name:    "lignes vides et commentaires ignores",
			content: "# liste du jour J\n\nabc-1\n\n# fin\nabc-2\n",
			want:    []string{"abc-1", "abc-2"},
		},
		{
			name:    "doublons ecartes en conservant l'ordre",
			content: "match_id\nabc-2\nabc-1\nabc-2\nabc-1\n",
			want:    []string{"abc-2", "abc-1"},
		},
		{
			name:    "fins de ligne CRLF",
			content: "match_id\r\nabc-1\r\nabc-2\r\n",
			want:    []string{"abc-1", "abc-2"},
		},
		{
			name:    "espaces autour de l'identifiant",
			content: "match_id\n  abc-1  \n",
			want:    []string{"abc-1"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := LoadMatchIDs(writeTemp(t, tc.content))
			if err != nil {
				t.Fatalf("LoadMatchIDs : %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("%d ids, attendu %d : %v", len(got), len(tc.want), got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("id[%d] = %q, attendu %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestLoadMatchIDs_Erreurs(t *testing.T) {
	t.Run("fichier absent", func(t *testing.T) {
		if _, err := LoadMatchIDs(filepath.Join(t.TempDir(), "nexiste-pas.tsv")); err == nil {
			t.Fatal("un fichier absent doit être une erreur, pas une liste vide")
		}
	})
	t.Run("fichier sans aucun id", func(t *testing.T) {
		_, err := LoadMatchIDs(writeTemp(t, "match_id\n\n# rien\n"))
		if err == nil {
			t.Fatal("une liste vide doit être une erreur : rien à traiter n'est pas un succès")
		}
		if !strings.Contains(err.Error(), "aucun match_id") {
			t.Errorf("message d'erreur peu explicite : %v", err)
		}
	})
}

// TestLoadMatchIDs_ValeursIgnorees fige l'invariant : le fichier ne sert QUE de liste.
// Si un jour une colonne de valeurs devenait exploitée, ce test le dirait — l'outil doit
// re-télécharger les scores, jamais réécrire une mesure vieille de plusieurs semaines.
func TestLoadMatchIDs_ValeursIgnorees(t *testing.T) {
	content := "match_id\tapi_t0\tapi_t1\n" + "abc-1\t200\t126\n"
	got, err := LoadMatchIDs(writeTemp(t, content))
	if err != nil {
		t.Fatalf("LoadMatchIDs : %v", err)
	}
	if len(got) != 1 || got[0] != "abc-1" {
		t.Fatalf("ids = %v, attendu [abc-1] — seule la colonne match_id doit sortir du fichier", got)
	}
}

// TestLoadMatchIDs_FichierVersionneDuLot lit le TSV réellement livré par la phase 1 :
// le défaut de --ids-file doit rester chargeable, en-tête compris.
func TestLoadMatchIDs_FichierVersionneDuLot(t *testing.T) {
	// Le test tourne depuis apps/go-api/cmd/backfill-team-scores/ ; la racine du dépôt
	// est cinq niveaux au-dessus.
	path := filepath.Join("..", "..", "..", "..", defaultIDsFile)
	if _, err := os.Stat(path); err != nil {
		t.Skipf("TSV du lot introuvable depuis ce répertoire (%v) — test sans objet hors arborescence", err)
	}
	got, err := LoadMatchIDs(path)
	if err != nil {
		t.Fatalf("le TSV versionné du lot doit être chargeable : %v", err)
	}
	if len(got) != 80 {
		t.Errorf("%d ids chargés, 80 attendus (les 80 écarts mesurés le 2026-08-24)", len(got))
	}
	for _, id := range got {
		if strings.EqualFold(id, matchIDColumn) {
			t.Fatal("la ligne d'en-tête a été prise pour une donnée")
		}
	}
}
