package service

// replay_map_background_traversee_test.go — LE SERVICE NE FAIT CONFIANCE A AUCUN APPELANT.
//
// Le handler valide deja le map_id (liste blanche, `handlers.MapIDValide`). Ces tests
// portent sur la SECONDE porte : `resolveBackgroundKeyDepuis` est le dernier point avant
// `PathResolver` et `os.Stat` / `os.ReadFile`, et un futur appelant (CLI, tache de fond)
// pourrait ne pas passer par le handler.
//
// Le troisieme test est le plus important : il montre CE QUI SE PASSERAIT sans la garde —
// le chemin construit par `PathResolver` sort reellement du repertoire des fonds — puis
// verifie que la garde le refuse. Sans cette moitie-la, on testerait un refus sans jamais
// avoir montre le danger.

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"
)

// clesHostiles : ce qu'aucune cle de fond ne doit jamais etre.
var clesHostiles = []string{
	"..",
	"../x",
	`..\x`,
	"x/y",
	`x\y`,
	"a/../../b",
	".",
	`..\..\..\Windows\win`,
}

// TestResolveBackgroundKey_RefuseUnMapIDHostile — la cle map_id, celle qui vient de
// l'appelant. Le sidecar EXISTE sous le nom nominal : un refus ici ne peut donc pas venir
// d'une absence de fichier.
func TestResolveBackgroundKey_RefuseUnMapIDHostile(t *testing.T) {
	root := fondDeCarte(t, title.DefaultSlug, "cliffhanger", "ridgeline", true)
	for _, hostile := range clesHostiles {
		repo := &mapNamesStub{mapID: hostile, names: []string{"Cliffhanger"}}
		svc := NewReplayService(title.DefaultSlug, root, repo)

		if _, err := svc.MapBackgroundForMap(context.Background(), hostile); !errors.Is(err, port.ErrMapBackgroundNotAvailable) {
			t.Errorf("cle %q : err = %v, attendu ErrMapBackgroundNotAvailable", hostile, err)
		}
		if _, err := svc.MapBackgroundImageForMap(context.Background(), hostile); !errors.Is(err, port.ErrMapBackgroundNotAvailable) {
			t.Errorf("cle %q (image) : err = %v, attendu ErrMapBackgroundNotAvailable", hostile, err)
		}
	}
}

// TestResolveBackgroundKey_RefuseUneCleDIndexHostile — meme garde sur la branche de repli.
// La cle vient alors du CONTENU d'un sidecar : un fichier, donc une donnee, donc quelque
// chose qui se remplace. Une seule regle pour les deux branches.
func TestResolveBackgroundKey_RefuseUneCleDIndexHostile(t *testing.T) {
	// `fondDeCarte` ecrit un sidecar dont `module` (= la cle) est `ridgeline` et dont
	// `mapNames` porte « cliffhanger ». On rejoue la meme forme avec un module hostile.
	root := fondDeCarte(t, title.DefaultSlug, "cliffhanger", `..\evade`, true)
	repo := &mapNamesStub{names: []string{"cliffhanger"}}
	svc := NewReplayService(title.DefaultSlug, root, repo)

	if _, err := svc.MapBackgroundForMap(context.Background(), "asset-x"); !errors.Is(err, port.ErrMapBackgroundNotAvailable) {
		t.Fatalf("err = %v, attendu ErrMapBackgroundNotAvailable", err)
	}
}

// TestResolveBackgroundKey_AucuneLectureHorsDuRepertoire — la preuve par le chemin.
//
// Premiere moitie : pour chaque cle hostile, le chemin que `PathResolver` construirait SORT
// du repertoire des fonds une fois `filepath.Clean` applique. C'est le danger, montre et
// non suppose.
// Seconde moitie : toute cle ACCEPTEE par la garde produit un chemin qui reste sous ce
// repertoire.
func TestResolveBackgroundKey_AucuneLectureHorsDuRepertoire(t *testing.T) {
	res := title.NewPathResolver(t.TempDir())
	dir := filepath.Clean(res.MapBackgroundDir(title.DefaultSlug))

	dangereuses := 0
	for _, hostile := range clesHostiles {
		if cleDeFondSure(hostile) {
			t.Errorf("cle %q acceptee par cleDeFondSure", hostile)
			continue
		}
		chemin := filepath.Clean(res.MapBackgroundMetaPath(title.DefaultSlug, hostile))
		if !sousLeRepertoire(chemin, dir) {
			dangereuses++
		}
	}
	// Sentinelle anti-vacuite : si AUCUNE cle de la table ne sortait du repertoire, ce test
	// ne dirait rien du danger qu'il pretend couvrir.
	if dangereuses == 0 {
		t.Fatal("aucune cle de la table ne fait sortir le chemin du repertoire des fonds — " +
			"la table ne decrit plus la menace")
	}

	for _, sure := range []string{
		"ridgeline",
		"fo08_wetland",
		"105f5d84-8de1-4908-af3a-1c4f3bf9d642",
	} {
		if !cleDeFondSure(sure) {
			t.Errorf("cle legitime %q refusee", sure)
			continue
		}
		for _, chemin := range []string{
			filepath.Clean(res.MapBackgroundMetaPath(title.DefaultSlug, sure)),
			filepath.Clean(res.MapBackgroundPath(title.DefaultSlug, sure)),
		} {
			if !sousLeRepertoire(chemin, dir) {
				t.Errorf("cle %q : chemin %q hors de %q", sure, chemin, dir)
			}
		}
	}
}

// sousLeRepertoire dit si `chemin` est contenu dans `dir` (les deux deja nettoyes).
func sousLeRepertoire(chemin, dir string) bool {
	return strings.HasPrefix(chemin, dir+string(filepath.Separator))
}
