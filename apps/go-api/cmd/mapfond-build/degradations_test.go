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
			nom:  "dossier personnel a point final",
			in:   `lecture C:\Users\Jean D.\Halo\Content\x.module : refusee`,
			want: `lecture x.module : refusee`,
		},
		{
			nom:  "dossier a point et a espaces, barres obliques",
			in:   `open D:/Jeux perso/J. Dupont/Halo Infinite/x.module: absent`,
			want: `open x.module: absent`,
		},
		{
			nom:  "partage reseau a dossiers a espaces",
			in:   `lecture \\hote\partage commun\Halo Infinite\x.module : refusee`,
			want: `lecture x.module : refusee`,
		},
		{
			nom:  "chemin Unix dans un dossier personnel a espaces",
			in:   `open /Users/Jean Dupont/Halo Infinite/x.module: absent`,
			want: `open x.module: absent`,
		},
		{
			nom:  "chemin Unix hors dossier personnel : /opt",
			in:   `open /opt/halo/deploy/x.module: absent`,
			want: `open x.module: absent`,
		},
		{
			nom:  "chemin Unix hors dossier personnel : /mnt",
			in:   `open "/mnt/d/Jeux/Halo/x.module" absent`,
			want: `open "x.module" absent`,
		},
		{
			nom:  "chemin Unix hors dossier personnel : /var",
			in:   `lecture (/var/lib/halo/x.module) absent`,
			want: `lecture (x.module) absent`,
		},
		{
			nom:  "chemin Unix hors dossier personnel : /Volumes",
			in:   `open /Volumes/Jeux Halo/deploy/x.module: absent`,
			want: `open x.module: absent`,
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

// La racine connue de l'outil est retirée avec ce qui la prolonge, même si le motif générique ne la
// reconnaîtrait pas ; les autres chemins restent traités par le motif.
func TestSansCheminAbsolu_RetireLaRacineConnue(t *testing.T) {
	cas := []struct {
		nom    string
		in     string
		racine string
		want   string
	}{
		{
			nom:    "racine Windows, separateurs et casse differents",
			in:     `open c:/users/jean d./halo/deploy/any/globals/x.module: absent`,
			racine: `C:\Users\Jean D.\Halo\deploy`,
			want:   `open x.module: absent`,
		},
		{
			nom:    "racine a point et espaces, separateur final",
			in:     `open C:\Users\Jean D.\Halo Infinite\deploy\any\x.module: absent`,
			racine: `C:\Users\Jean D.\Halo Infinite\deploy\`,
			want:   `open x.module: absent`,
		},
		{
			nom:    "racine Unix a espaces",
			in:     `open /srv/Jean Dupont/halo/deploy/x.module: absent`,
			racine: `/srv/Jean Dupont/halo`,
			want:   `open x.module: absent`,
		},
		{
			nom:    "racine citee seule",
			in:     `installation C:\Jeux\Halo introuvable`,
			racine: `C:\Jeux\Halo`,
			want:   `installation  introuvable`,
		},
		{
			nom:    "dossier voisin dont le nom commence comme la racine : coupe par le motif general",
			in:     `open C:\Jeux\Halo2\x.module: absent`,
			racine: `C:\Jeux\Halo`,
			want:   `open x.module: absent`,
		},
		{
			nom:    "racine relative ignoree",
			in:     `aucun maillage dans deploy`,
			racine: `deploy`,
			want:   `aucun maillage dans deploy`,
		},
		{
			nom:    "racine vide ignoree",
			in:     `aucun maillage de navigation`,
			racine: ``,
			want:   `aucun maillage de navigation`,
		},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			got := sansCheminAbsolu([]string{c.in}, c.racine)
			if len(got) != 1 || got[0] != c.want {
				t.Errorf("sansCheminAbsolu = %q, attendu %q", got, c.want)
			}
		})
	}
}

func TestSansCheminAbsolu_RacineLaPlusLongueDAbord(t *testing.T) {
	got := sansCheminAbsolu([]string{`open C:\Jeux\Halo\deploy\x.module: absent`}, `C:\Jeux`, `C:\Jeux\Halo`)
	if len(got) != 1 || got[0] != `open x.module: absent` {
		t.Errorf("sansCheminAbsolu = %q", got)
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

// ecritSidecar donne au rempart la racine d'installation et celle du dépôt que l'outil possède.
func TestEcritSidecar_RetireLesRacinesDeLOutil(t *testing.T) {
	dir := t.TempDir()
	e := &environnement{
		sortieDir: dir, reglages: &reglagesFond{},
		racineJeu: `C:\Users\Jean D.\Halo\deploy`, repoRoot: `/srv/Jean Dupont/levelup`,
	}
	b := himap.BilanCuisson{Degradations: []string{
		`ouverture C:\Users\Jean D.\Halo\deploy\any\globals\a.module: absent`,
		`copie /srv/Jean Dupont/levelup/data/b.module: refusee`,
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
	want := []string{`ouverture a.module: absent`, `copie b.module: refusee`}
	if !reflect.DeepEqual(s.Degradations, want) {
		t.Errorf("degradations publiees = %q, attendu %q", s.Degradations, want)
	}
}
