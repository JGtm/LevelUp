package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-map-images/himap"
)

func TestSansCheminAbsolu(t *testing.T) {
	cas := []struct {
		nom  string
		in   string
		want string
	}{
		{
			nom: "chemin Windows avec espaces, message du systeme",
			in: `sddt : module any/globals introuvable : GetFileAttributesEx C:\Jeux\Halo Infinite\Content\deploy\any\levels\multi\globals\common-rtx-new.module: ` +
				`Le chemin d’accès spécifié est introuvable.`,
			want: `sddt : module any/globals introuvable : GetFileAttributesEx common-rtx-new.module: Le chemin d’accès spécifié est introuvable.`,
		},
		{
			nom:  "chemin Windows aux barres obliques",
			in:   `eau : open D:/Games/Halo Infinite/deploy/any/globals/common.module: introuvable`,
			want: `eau : open common.module: introuvable`,
		},
		{
			nom:  "chemin Unix",
			in:   `sddt : open /home/joueur/halo/deploy/any/globals/common.module: no such file or directory`,
			want: `sddt : open common.module: no such file or directory`,
		},
		{
			nom:  "chemin Unix en debut de texte",
			in:   `/Users/joueur/halo/common.module introuvable`,
			want: `common.module introuvable`,
		},
		{
			nom:  "partage reseau",
			in:   `lecture \\poste\partage\halo\common.module : refusee`,
			want: `lecture common.module : refusee`,
		},
		{
			nom:  "deux chemins dans une meme degradation",
			in:   `copie C:\a\b\src.module vers /tmp/work/dst.module impossible`,
			want: `copie src.module vers dst.module impossible`,
		},
		{
			nom:  "sans chemin : intacte",
			in:   `aucun maillage de navigation dans le module any`,
			want: `aucun maillage de navigation dans le module any`,
		},
		{
			nom:  "chemin relatif de module : intact",
			in:   `module any/globals introuvable : any/levels/multi/globals`,
			want: `module any/globals introuvable : any/levels/multi/globals`,
		},
		{
			nom:  "adresse web : intacte",
			in:   `telechargement https://example.org/a/b.module refuse`,
			want: `telechargement https://example.org/a/b.module refuse`,
		},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			got := sansCheminAbsolu([]string{c.in})
			if len(got) != 1 || got[0] != c.want {
				t.Errorf("sansCheminAbsolu = %q, attendu %q", got, c.want)
			}
		})
	}
}

func TestSansCheminAbsolu_NeModifiePasLEntreeEtRendNilSansDegradation(t *testing.T) {
	in := []string{`open C:\a\b.module`, `aucun maillage`}
	garde := append([]string(nil), in...)
	got := sansCheminAbsolu(in)
	if !reflect.DeepEqual(in, garde) {
		t.Errorf("l'entree a ete modifiee : %q", in)
	}
	if !reflect.DeepEqual(got, []string{`open b.module`, `aucun maillage`}) {
		t.Errorf("sansCheminAbsolu = %q", got)
	}
	if sansCheminAbsolu(nil) != nil || sansCheminAbsolu([]string{}) != nil {
		t.Error("une liste vide doit rendre nil")
	}
}

// Le fichier publié ne porte aucun chemin absolu : ecritSidecar passe par le rempart.
func TestEcritSidecar_DegradationSansCheminAbsolu(t *testing.T) {
	dir := t.TempDir()
	e := &environnement{sortieDir: dir, reglages: &reglagesFond{}}
	b := himap.BilanCuisson{Degradations: []string{
		`sddt : module any/globals introuvable : GetFileAttributesEx C:\Jeux\Halo Infinite\any\globals\common.module: absent`,
		`aucun maillage de navigation dans le module any`,
	}}
	cal := himap.CalageFond{MetresParPixel: 1, Largeur: 10, Hauteur: 10}
	if err := e.ecritSidecar(context.Background(), cible{cle: "carte_test"}, b, cal, nil); err != nil {
		t.Fatalf("ecritSidecar : %v", err)
	}
	brut, err := os.ReadFile(filepath.Join(dir, "carte_test.json"))
	if err != nil {
		t.Fatal(err)
	}
	var s struct {
		Degradations []string `json:"degradations"`
	}
	if err := json.Unmarshal(brut, &s); err != nil {
		t.Fatal(err)
	}
	want := []string{
		`sddt : module any/globals introuvable : GetFileAttributesEx common.module: absent`,
		`aucun maillage de navigation dans le module any`,
	}
	if !reflect.DeepEqual(s.Degradations, want) {
		t.Errorf("degradations publiees = %q, attendu %q", s.Degradations, want)
	}
}
