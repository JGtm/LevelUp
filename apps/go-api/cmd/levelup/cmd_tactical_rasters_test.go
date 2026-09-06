package main

// cmd_tactical_rasters_test.go — LE RATTRAPAGE DES SIDECARS, SUR UN REPERTOIRE TEMPORAIRE.
//
// Aucune base, aucun film, aucune cuisson : la commande LIT des artefacts et depose de
// petits JSON. Ce qui est eprouve ici :
//
//	l'enumeration     elle passe par LE lecteur d'artefacts du depot, donc elle ne compte
//	                  QUE les `{short}.json` de premier niveau — le sous-dossier `rasters/`
//	                  qu'elle vient de remplir ne se compte pas lui-meme (sans quoi la
//	                  seconde passe examinerait des « matchs » nommes d'apres ses sidecars) ;
//	l'idempotence     une seconde passe immediate ecrit ZERO fichier ;
//	la fraicheur      un sidecar projete d'un AUTRE schema d'artefact est refait ;
//	le --dry-run      il COMPTE ce qu'il ecrirait, et n'ecrit rien ;
//	le refus nu       sans --backfill, la commande ne parcourt rien.

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
)

// trArtefact : un artefact minimal, deux joueurs immobiles pendant 2 s (100 ms par frame).
const trArtefact = `{"schemaVersion":39,"matchId":"%ID%","frameCount":21,"frameIntervalMs":100,
  "tracks":[
    {"slot":1,"team":-1,"xuid":"111","startFrame":0,"endFrame":20,
     "points":[{"t":0,"x":0.25,"y":0.25},{"t":20,"x":0.25,"y":0.25}]},
    {"slot":2,"team":-1,"xuid":"222","startFrame":0,"endFrame":20,
     "points":[{"t":0,"x":4.25,"y":0.25},{"t":20,"x":4.25,"y":0.25}]}]}`

// trPoserArtefacts depose des artefacts a l'endroit EXACT ou la commande ira les chercher.
func trPoserArtefacts(t *testing.T, root string, shorts ...string) {
	t.Helper()
	pr := titlePkg.NewPathResolver(root)
	for _, short := range shorts {
		path := pr.ReplayArtifactPath(titlePkg.DefaultSlug, short)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("creer le dossier des artefacts: %v", err)
		}
		corps := strings.ReplaceAll(trArtefact, "%ID%", short)
		if err := os.WriteFile(path, []byte(corps), 0o644); err != nil {
			t.Fatalf("ecrire artefact: %v", err)
		}
	}
}

// trOptions : la passe nominale sur un titre qui declare bien `film.replay_artifact`.
// On appelle `projeterCorpusRasters` directement plutot que `runTacticalRasters` : le gate
// de capability et le parsing de flags ont leurs propres tests, et charger un TOML de
// titre ferait dependre ces cas-ci d'une configuration de depot.
func trOptions(dryRun bool) tacticalRastersOptions {
	return tacticalRastersOptions{titleSlug: titlePkg.DefaultSlug, backfill: true, dryRun: dryRun}
}

// TestTacticalRasters_DeposeEtEstIdempotent — le coeur du rattrapage.
func TestTacticalRasters_DeposeEtEstIdempotent(t *testing.T) {
	root := t.TempDir()
	trPoserArtefacts(t, root, "aaaaaaaa", "bbbbbbbb")
	cfg := &config.AppConfig{RepoRoot: root}
	ctx := context.Background()

	shorts, err := artefactsDuTitre(ctx, cfg, titlePkg.DefaultSlug)
	if err != nil {
		t.Fatalf("enumeration: %v", err)
	}
	if len(shorts) != 2 || shorts[0] != "aaaaaaaa" || shorts[1] != "bbbbbbbb" {
		t.Fatalf("artefacts enumeres = %v, attendu [aaaaaaaa bbbbbbbb] dans l'ordre", shorts)
	}

	b := projeterCorpusRasters(ctx, cfg, trOptions(false), shorts)
	if b.lus != 2 || b.ecrits != 2 || b.sautes != 0 || b.echecs != 0 {
		t.Fatalf("premiere passe = %+v, attendu 2 lus / 2 ecrits", b)
	}
	pr := titlePkg.NewPathResolver(root)
	for _, short := range shorts {
		if _, err := os.Stat(pr.TacticalRasterPath(titlePkg.DefaultSlug, short)); err != nil {
			t.Fatalf("sidecar de %s absent: %v", short, err)
		}
	}

	// L'ENUMERATION NE SE COMPTE PAS ELLE-MEME : les sidecars viennent d'etre deposes dans
	// `rasters/`, et le lecteur d'artefacts saute les repertoires. Sans cela, la seconde
	// passe irait chercher des artefacts nommes d'apres les sidecars.
	shorts2, err := artefactsDuTitre(ctx, cfg, titlePkg.DefaultSlug)
	if err != nil {
		t.Fatalf("seconde enumeration: %v", err)
	}
	if len(shorts2) != 2 {
		t.Fatalf("seconde enumeration = %v, attendu les 2 memes artefacts", shorts2)
	}

	// IDEMPOTENCE : zero ecriture.
	b2 := projeterCorpusRasters(ctx, cfg, trOptions(false), shorts2)
	if b2.ecrits != 0 || b2.sautes != 2 || b2.echecs != 0 {
		t.Fatalf("seconde passe = %+v, attendu 0 ecrit / 2 deja a jour", b2)
	}
}

// TestTacticalRasters_SidecarPerime — un sidecar projete d'un AUTRE schema d'artefact est
// refait. C'est la cle de fraicheur apres une re-cuisson du parc : sans elle, le
// rattrapage laisserait en place des rasters tires d'un decodage perime.
func TestTacticalRasters_SidecarPerime(t *testing.T) {
	root := t.TempDir()
	trPoserArtefacts(t, root, "aaaaaaaa")
	cfg := &config.AppConfig{RepoRoot: root}
	ctx := context.Background()
	pr := titlePkg.NewPathResolver(root)
	cible := pr.TacticalRasterPath(titlePkg.DefaultSlug, "aaaaaaaa")
	if err := os.MkdirAll(filepath.Dir(cible), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Un sidecar au BON format, mais projete d'un artefact de schema 20.
	perime := `{"schema_version":1,"match_id":"aaaaaaaa","short_id":"aaaaaaaa",
	  "artifact_schema_version":20,"pas_m":0.5,"frame_interval_ms":100,
	  "pas_echantillon_ms":250,"joueurs":[]}`
	if err := os.WriteFile(cible, []byte(perime), 0o644); err != nil {
		t.Fatalf("ecrire sidecar perime: %v", err)
	}
	b := projeterCorpusRasters(ctx, cfg, trOptions(false), []string{"aaaaaaaa"})
	if b.ecrits != 1 || b.sautes != 0 {
		t.Fatalf("passe = %+v, attendu 1 ecrit (le sidecar etait projete du schema 20)", b)
	}
	// Et il porte desormais des joueurs, ce que le sidecar perime n'avait pas.
	s, ok := lireSidecarRaster(cible)
	if !ok || len(s.Joueurs) != 2 || s.ArtifactSchemaVersion != 39 {
		t.Fatalf("sidecar refait = %+v (ok=%v)", s, ok)
	}
}

// TestTacticalRasters_SidecarDUnAutreFormat — un sidecar dont le `schema_version` n'est
// plus le courant n'est PAS relu comme a jour : on ne fait pas confiance a un fichier
// qu'on ne sait pas interpreter.
func TestTacticalRasters_SidecarDUnAutreFormat(t *testing.T) {
	root := t.TempDir()
	trPoserArtefacts(t, root, "aaaaaaaa")
	pr := titlePkg.NewPathResolver(root)
	cible := pr.TacticalRasterPath(titlePkg.DefaultSlug, "aaaaaaaa")
	if err := os.MkdirAll(filepath.Dir(cible), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(cible, []byte(`{"schema_version":99,"artifact_schema_version":39}`), 0o644); err != nil {
		t.Fatalf("ecrire: %v", err)
	}
	if sidecarAJour(cible) {
		t.Fatal("un sidecar d'un autre format a ete pris pour a jour")
	}
	b := projeterCorpusRasters(context.Background(), &config.AppConfig{RepoRoot: root},
		trOptions(false), []string{"aaaaaaaa"})
	if b.ecrits != 1 {
		t.Fatalf("passe = %+v, attendu 1 ecrit", b)
	}
}

// TestTacticalRasters_DryRunNEcritRien — il COMPTE ce qu'il ecrirait, et le disque ne
// bouge pas.
func TestTacticalRasters_DryRunNEcritRien(t *testing.T) {
	root := t.TempDir()
	trPoserArtefacts(t, root, "aaaaaaaa", "bbbbbbbb")
	b := projeterCorpusRasters(context.Background(), &config.AppConfig{RepoRoot: root},
		trOptions(true), []string{"aaaaaaaa", "bbbbbbbb"})
	if b.ecrits != 2 || b.echecs != 0 {
		t.Fatalf("dry-run = %+v, attendu 2 « a ecrire »", b)
	}
	dir := titlePkg.NewPathResolver(root).TacticalRasterDir(titlePkg.DefaultSlug)
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("--dry-run a cree le dossier des rasters (err = %v)", err)
	}
}

// TestTacticalRasters_ArtefactIllisible — l'echec d'un artefact n'arrete pas la passe, et
// il se voit dans le bilan.
func TestTacticalRasters_ArtefactIllisible(t *testing.T) {
	root := t.TempDir()
	trPoserArtefacts(t, root, "bbbbbbbb")
	pr := titlePkg.NewPathResolver(root)
	casse := pr.ReplayArtifactPath(titlePkg.DefaultSlug, "aaaaaaaa")
	if err := os.WriteFile(casse, []byte(`{"schemaVersion":`), 0o644); err != nil {
		t.Fatalf("ecrire: %v", err)
	}
	b := projeterCorpusRasters(context.Background(), &config.AppConfig{RepoRoot: root},
		trOptions(false), []string{"aaaaaaaa", "bbbbbbbb"})
	if b.lus != 2 || b.ecrits != 1 || b.echecs != 1 {
		t.Fatalf("passe = %+v, attendu 2 lus / 1 ecrit / 1 echec", b)
	}
}

// TestTacticalRasters_SansBackfill — la commande n'a pas de mode par defaut : une
// invocation nue ne parcourt rien.
func TestTacticalRasters_SansBackfill(t *testing.T) {
	root := t.TempDir()
	trPoserArtefacts(t, root, "aaaaaaaa")
	if err := runTacticalRasters(&config.AppConfig{RepoRoot: root}, nil); err == nil {
		t.Fatal("sans --backfill : attendu un refus explicite")
	}
	dir := titlePkg.NewPathResolver(root).TacticalRasterDir(titlePkg.DefaultSlug)
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("une invocation nue a ecrit des sidecars (err = %v)", err)
	}
}

// TestTacticalRasters_DossierAbsent — un titre dont aucun artefact n'a jamais ete cuit est
// un etat NOMINAL : ensemble vide, pas d'erreur.
func TestTacticalRasters_DossierAbsent(t *testing.T) {
	shorts, err := artefactsDuTitre(context.Background(), &config.AppConfig{RepoRoot: t.TempDir()}, titlePkg.DefaultSlug)
	if err != nil {
		t.Fatalf("dossier absent : attendu un ensemble vide, got %v", err)
	}
	if len(shorts) != 0 {
		t.Fatalf("artefacts = %v, attendu aucun", shorts)
	}
}
