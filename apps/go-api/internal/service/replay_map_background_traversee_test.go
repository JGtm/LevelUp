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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/replay"
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
// La cle vient alors du CONTENU d'un repertoire : des fichiers, donc des donnees, donc
// quelque chose qui se remplace. Une seule regle pour les deux branches.
//
// LA PREMIERE VERSION DE CE TEST NE MORDAIT PAS SOUS WINDOWS (revue R2, P2), et c'est
// instructif : elle posait la cle hostile en passant `..\evade` a `MapBackgroundMetaPath`,
// dont le `filepath.Join` NETTOYAIT le `..\` — le sidecar atterrissait hors de
// `map_backgrounds/`, qui n'etait donc jamais cree, `MapBackgroundIndexFor` echouait sur
// `os.ReadDir` et la fonction sortait AVANT la garde d'index. Le test passait pour la
// mauvaise raison, et retirer la garde le laissait vert sur ce poste.
//
// LA VERSION QUI MORD, sur les deux plates-formes : le repertoire des fonds EXISTE et
// contient deux sidecars ecrits a des noms de fichier CHOISIS (`os.WriteFile` sur un chemin
// que ce test construit lui-meme, jamais `filepath.Join` de la cle) —
//
//   - `ridgeline.json`, legitime, pour que l'index soit constructible ;
//   - `..evade.json`, dont le STEM (`..evade`, la cle rendue par l'index) contient `..`.
//     C'est un nom de fichier parfaitement legal : il vit BIEN dans `map_backgrounds/`, et
//     `MapBackgroundMetaPath` le retrouve. Rien d'autre que la garde ne peut donc le
//     refuser — sans elle, la resolution ABOUTIT.
//
// Une cle d'index ne peut pas, aujourd'hui, porter un separateur : elle vient d'un nom de
// fichier rendu par `os.ReadDir`. La garde est donc CONSERVATRICE sur cette branche, et
// c'est voulu — la regle porte sur la CLE, pas sur sa provenance, et une provenance ne
// reste sure que tant que personne ne la change.
func TestResolveBackgroundKey_RefuseUneCleDIndexHostile(t *testing.T) {
	root := t.TempDir()
	dir := title.NewPathResolver(root).MapBackgroundDir(title.DefaultSlug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	// Sidecar legitime : sans lui l'index serait vide et le test ne prouverait rien de la
	// branche de repli.
	ecrire(t, filepath.Join(dir, "ridgeline.json"), sidecarNomme("ridgeline", "cliffhanger"))
	// Sidecar HOSTILE, pose par son NOM DE FICHIER : le stem porte `..`.
	ecrire(t, filepath.Join(dir, "..evade.json"), sidecarNomme("..evade", "piege"))
	ecrire(t, filepath.Join(dir, "..evade.png"), "\x89PNG\r\n\x1a\nfaux")

	// Le decor est bien celui qu'on croit : l'index resout « piege » vers la cle hostile,
	// et le sidecar de cette cle est LISIBLE. Sans ces deux verifications, un test vert ne
	// dirait pas si c'est la garde qui a refuse ou le decor qui a manque.
	idx, err := replay.MapBackgroundIndexFor(dir)
	if err != nil {
		t.Fatalf("MapBackgroundIndexFor: %v", err)
	}
	cle, ok := idx.Lookup("piege")
	if !ok || cle != "..evade" {
		t.Fatalf("index : Lookup(piege) = %q, %v — attendu la cle hostile", cle, ok)
	}
	if _, err := replay.LoadMapBackground(
		title.NewPathResolver(root).MapBackgroundMetaPath(title.DefaultSlug, cle)); err != nil {
		t.Fatalf("le sidecar de la cle hostile doit etre lisible, sinon le refus est trivial : %v", err)
	}

	repo := &mapNamesStub{names: []string{"piege"}}
	svc := NewReplayService(title.DefaultSlug, root, repo)
	if _, err := svc.MapBackgroundForMap(context.Background(), "asset-x"); !errors.Is(err, port.ErrMapBackgroundNotAvailable) {
		t.Fatalf("err = %v, attendu ErrMapBackgroundNotAvailable", err)
	}
	if _, err := svc.MapBackgroundImageForMap(context.Background(), "asset-x"); !errors.Is(err, port.ErrMapBackgroundNotAvailable) {
		t.Fatalf("err image = %v, attendu ErrMapBackgroundNotAvailable", err)
	}
}

// sidecarNomme rend un sidecar de calage minimal mais de la MEME forme que la cuisson :
// `module` et `mapNames` sont les deux champs dont l'index tire ses identites.
func sidecarNomme(module, nom string) string {
	return `{"schemaVersion":1,"module":"` + strings.ReplaceAll(module, `\`, `\`) + `",` +
		`"mapNames":["` + nom + `"],"image":"` + nom + `.png",` +
		`"source":"test","generatedAt":"2026-09-06T10:00:00Z","style":"jeu",` +
		`"calibration":{"metersPerPixel":0.092,"originX":0,"originY":0,` +
		`"widthPx":100,"heightPx":100,"convention":"x = originX + (px+0.5)*mpp"},` +
		`"stats":{"anchors":4,"anchorsInFrame":4,"anchorsWithGround":4}}`
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
