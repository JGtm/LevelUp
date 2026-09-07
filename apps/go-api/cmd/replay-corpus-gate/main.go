// cmd/replay-corpus-gate — GATE LOCAL DE NON-REGRESSION DES ARTEFACTS DE REJEU.
//
// # POURQUOI CE GATE EXISTE
//
// Trois regressions de donnees (28/08, 30/08, 02/09 — pont d'identite par manche, drapeaux non
// attribues, "une piste = une vie") ont traverse des goldens SYNTHETIQUES verts pendant
// dix-neuf schemas de suite, faute d'un differentiel sur des FILMS REELS. Le balayage
// retroactif du parc local (119 matchs, .ai/V7.5/v2/BALAYAGE_PARC_2026-09-06.md) les a
// finalement trouvees — mais APRES coup, par une mesure manuelle. Ce gate industrialise CETTE
// MEME METHODE sur un corpus TEMOIN restreint et versionne (config/replay_corpus.toml) : cuire
// chaque temoin au HEAD et le confronter a l'artefact deja cuit dans le parc local, AVANT tout
// merge qui toucherait au decodeur ou au constructeur de rejeu.
//
// # CE QUE CE GATE NE FAIT PAS
//
// Il ne bumpe AUCUN schema — il COMPARE. Si le parc local est a un schema plus ancien que le
// HEAD teste, les ecarts attendus sont des GAINS (nouveaux calques, corrections documentees) ;
// toute PERTE est un fait a rapporter au journal, jamais a masquer en resserrant le manifeste
// ou en filtrant le rapport.
//
// # CE QU'IL EXIGE
//
// Le PARC LOCAL de developpement (chunks de film + artefacts deja cuits sous
// data/cache/film_chunks|film_manifests|replays) ET l'acces en lecture a la base partagee du
// titre (pour les faits du match, via `levelup replay-facts-export` en sous-processus — la
// SEULE etape qui exige CGO/gcc). PAS le jeu installe : la cuisson elle-meme (`replaybuild`)
// ne lit que des catalogues VERSIONNES (data/titles/{slug}/reference), jamais l'installation —
// contrairement au tag `gamefiles` dont ce gate ne partage que l'ESPRIT (ressource locale
// volumineuse, absente en CI, degradation propre plutot qu'echec).
//
// # USAGE
//
//	cd apps/go-api && go run ./cmd/replay-corpus-gate \
//	  [--manifest config/replay_corpus.toml] [--parc-root DIR] [--lock-root DIR] \
//	  [--source-root DIR] [--work-root DIR] [--keep-work] [--json rapport.json]
//
// Racines (cf. roots.go pour le detail et les variables d'environnement equivalentes) :
// `--source-root` = le depot ou ce binaire tourne (code + config au HEAD teste, defaut
// title.FindRepoRoot) ; `--parc-root` = le parc de developpement (chunks, artefacts de
// reference, defaut auto-detecte par le `.git` commun) ; `--lock-root` = ou poser le verrou de
// decodage PARTAGE avec tout autre outil de cuisson de ce depot (defaut CacheRootDir du parc).
//
// Codes de sortie : 0 = aucune perte, 1 = au moins un temoin porte une perte ou une erreur de
// cuisson/comparaison, 2 = usage ou manifeste invalide. Un temoin ABSENT du parc local est un
// avertissement `slog`, jamais un echec (cf. codeSortie).
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/replaybuild"
)

func main() {
	manifestPath := flag.String("manifest", "", "chemin du manifeste (defaut : <source-root>/config/replay_corpus.toml)")
	parcRootFlag := flag.String("parc-root", "", "racine du parc de developpement (defaut : auto-detecte par git)")
	lockRootFlag := flag.String("lock-root", "", "racine du verrou de decodage partage (defaut : CacheRootDir du parc)")
	sourceRootFlag := flag.String("source-root", "", "depot ou lire le code/la config au HEAD (defaut : title.FindRepoRoot)")
	workRootFlag := flag.String("work-root", "", "racine de travail temporaire (defaut : dossier temporaire jetable)")
	keepWork := flag.Bool("keep-work", false, "conserver la racine de travail apres l'execution (debug)")
	sortieJSON := flag.String("json", "", "fichier ou ecrire le rapport JSON complet (vide = aucun)")
	flag.Parse()

	if err := executer(*manifestPath, *parcRootFlag, *lockRootFlag, *sourceRootFlag, *workRootFlag, *keepWork, *sortieJSON); err != nil {
		slog.Error("replay-corpus-gate", "err", err)
		os.Exit(2)
	}
}

// executer orchestre le gate de bout en bout : resolution des racines, chargement du
// manifeste, preparation de la reference, une cuisson-comparaison par temoin, impression et
// verdict.
func executer(manifestPath, parcRootFlag, lockRootFlag, sourceRootFlag, workRootFlag string, keepWork bool, sortieJSON string) error {
	sourceRoot, err := resolveSourceRoot(sourceRootFlag)
	if err != nil {
		return fmt.Errorf("racine source : %w", err)
	}
	if manifestPath == "" {
		manifestPath = filepath.Join(sourceRoot, "config", "replay_corpus.toml")
	}
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return err
	}
	titleSlug := manifest.Meta.TitleSlug

	// La resolution du parc a besoin du titre pour VALIDER l'auto-detection (presence de sa
	// base partagee) — d'ou l'ordre : manifeste avant parc.
	parcRoot, err := resolveParcRoot(parcRootFlag, titleSlug)
	if err != nil {
		return fmt.Errorf("racine du parc : %w", err)
	}
	lockRoot := resolveLockRoot(lockRootFlag, parcRoot)

	caps, err := games.LoadCapabilityMap(sourceRoot, titleSlug)
	if err != nil {
		return fmt.Errorf("capabilities du titre %s : %w", titleSlug, err)
	}
	if !caps.Has(games.CapFilmReplayArtifact) {
		return fmt.Errorf("le titre %s ne declare pas la capability %s — aucun artefact a comparer",
			titleSlug, games.CapFilmReplayArtifact)
	}

	workRoot, cleanup, err := prepareWorkRoot(workRootFlag, keepWork)
	if err != nil {
		return fmt.Errorf("racine de travail : %w", err)
	}
	defer cleanup()

	slog.Info("replay-corpus-gate: racines resolues",
		"source", sourceRoot, "parc", parcRoot, "verrou", lockRoot, "travail", workRoot,
		"manifeste", manifestPath, "temoins", len(manifest.Temoins))

	if err := stageReferenceOnce(sourceRoot, workRoot, titleSlug); err != nil {
		return fmt.Errorf("catalogues de reference : %w", err)
	}

	factsDir := filepath.Join(workRoot, "facts")
	ids := make([]string, len(manifest.Temoins))
	for i, t := range manifest.Temoins {
		ids[i] = t.ID
	}
	goAPIDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("repertoire courant : %w", err)
	}
	if err := exportFacts(goAPIDir, parcRoot, titleSlug, factsDir, ids); err != nil {
		return fmt.Errorf("export des faits : %w", err)
	}

	lignes := make([]ligneRapport, 0, len(manifest.Temoins))
	for _, t := range manifest.Temoins {
		lignes = append(lignes, traiterTemoin(t, parcRoot, workRoot, lockRoot, titleSlug, factsDir))
	}

	imprimerTableau(os.Stdout, lignes)
	imprimerDetailPertes(os.Stdout, lignes)
	if sortieJSON != "" {
		if err := ecrireRapportJSON(sortieJSON, lignes); err != nil {
			return err
		}
	}

	code := codeSortie(lignes)
	if code != 0 {
		os.Exit(code)
	}
	return nil
}

// traiterTemoin cuit et compare UN temoin ; ne rend JAMAIS d'erreur — un temoin absent ou en
// echec produit une ligne qui le dit, pour que les autres temoins du manifeste soient traites
// quand meme (CLAUDE.md : un rapport partiel muet vaut moins qu'un rapport complet nomme).
func traiterTemoin(t Temoin, parcRoot, workRoot, lockRoot, titleSlug, factsDir string) ligneRapport {
	base := ligneRapport{Temoin: t}

	filmDir, err := stageFilm(parcRoot, workRoot, t.ID)
	if err != nil {
		slog.Warn("replay-corpus-gate: temoin absent du parc local — ignore, pas un echec",
			"temoin", t.ID, "famille", t.Famille, "err", err)
		base.Absent, base.AbsentCause = true, err.Error()
		return base
	}

	factsPath := filepath.Join(factsDir, t.ID+".facts.json")
	facts, err := replaybuild.ReadFactsFile(factsPath)
	if err != nil {
		base.Erreur = fmt.Errorf("faits du match : %w", err)
		return base
	}

	refPath := referenceArtifactPath(parcRoot, titleSlug, facts.MatchID)
	if _, statErr := os.Stat(refPath); statErr != nil {
		slog.Warn("replay-corpus-gate: aucun artefact de reference au parc — temoin ignore",
			"temoin", t.ID, "chemin", refPath, "err", statErr)
		base.Absent, base.AbsentCause = true, "aucun artefact de reference au parc : "+statErr.Error()
		return base
	}

	cuisson, err := bakeTemoin(workRoot, lockRoot, titleSlug, facts, filmDir)
	if err != nil {
		base.Erreur = fmt.Errorf("cuisson : %w", err)
		return base
	}
	base.Duree = cuisson.Duree

	rap, err := compareTemoin(refPath, cuisson.Sortie.ArtifactPath)
	if err != nil {
		base.Erreur = fmt.Errorf("comparaison : %w", err)
		return base
	}
	base.SchemaParc, base.SchemaHEAD, base.Gains, base.Pertes, base.PertesDetail = bilanDepuisRapport(rap)
	return base
}

// ecrireRapportJSON depose le detail des lignes, pour un consommateur automatique (CI, un
// futur tableau de bord) — le tableau texte reste la sortie lisible par un operateur.
func ecrireRapportJSON(path string, lignes []ligneRapport) error {
	type detailJSON struct {
		Axe      string `json:"axe"`
		Metrique string `json:"metrique"`
		Sens     string `json:"sens"`
		Ancien   string `json:"ancien,omitempty"`
		Nouveau  string `json:"nouveau,omitempty"`
	}
	type ligneJSON struct {
		ID         string       `json:"id"`
		Famille    string       `json:"famille"`
		Absent     bool         `json:"absent,omitempty"`
		Erreur     string       `json:"erreur,omitempty"`
		SchemaParc int          `json:"schemaParc,omitempty"`
		SchemaHEAD int          `json:"schemaHead,omitempty"`
		Gains      int          `json:"gains"`
		Pertes     int          `json:"pertes"`
		DureeMS    int64        `json:"dureeMs"`
		Detail     []detailJSON `json:"pertesDetail,omitempty"`
	}
	out := make([]ligneJSON, len(lignes))
	for i, l := range lignes {
		lj := ligneJSON{
			ID: l.Temoin.ID, Famille: l.Temoin.Famille, Absent: l.Absent,
			SchemaParc: l.SchemaParc, SchemaHEAD: l.SchemaHEAD,
			Gains: l.Gains, Pertes: l.Pertes, DureeMS: l.Duree.Milliseconds(),
		}
		if l.Erreur != nil {
			lj.Erreur = l.Erreur.Error()
		}
		for _, d := range l.PertesDetail {
			lj.Detail = append(lj.Detail, detailJSON{
				Axe: d.Axe, Metrique: d.Metrique, Sens: d.Sens, Ancien: d.Ancien, Nouveau: d.Nouveau,
			})
		}
		out[i] = lj
	}
	return ecrireJSONGenerique(path, out)
}
