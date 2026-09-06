// cmd/replay-corpus-gate — GATE LOCAL DE NON-REGRESSION DES ARTEFACTS DE REJEU.
//
// # POURQUOI CE GATE EXISTE
//
// Trois regressions de donnees (28/08, 30/08, 02/09 — pont d'identite par manche, drapeaux non
// attribues, "une piste = une vie") ont traverse des goldens SYNTHETIQUES verts pendant
// dix-neuf schemas de suite, faute d'un differentiel sur des FILMS REELS. Le balayage
// retroactif du parc local (119 matchs, .ai/V7.5/v2/BALAYAGE_PARC_2026-09-06.md) les a
// finalement trouvees — mais APRES coup, par une mesure manuelle. Ce gate industrialise CETTE
// MEME METHODE sur un corpus TEMOIN restreint et versionne (config/replay_corpus.toml).
//
// # DEUX MODES DE REFERENCE (decision superviseur, 2026-09-06)
//
// --reference=base (DEFAUT) : cuit chaque temoin DEUX FOIS — avec le code du HEAD, et avec
// le code d'une revision de BASE (defaut : origin/feat/v75 si le HEAD en differe, sinon
// HEAD^ — cf. base.go) — puis compare les deux cuissons fraiches. Toute perte = code 1. C'est
// le gate a lancer avant merge : le signal est binaire, une perte est necessairement due au
// diff en cours de revue, jamais a l'age d'un parc jamais a jour.
//
// --reference=parc : compare le HEAD a l'artefact DEJA CUIT dans le parc local (methode
// historique, balayage de release). INFORMATIF par defaut (imprime le tableau, sort en 0) —
// --strict le rend bloquant. Un gate qui rendrait PERTE sur tous les temoins au meilleur etat
// connu ne gaterait rien : le parc n'est jamais a jour, cf. le tableau §1 de
// BALAYAGE_PARC_2026-09-06.md.
//
// Dans les deux modes, aucun schema n'est bumpe — le gate COMPARE.
//
// # CE QU'IL EXIGE
//
// Le PARC LOCAL de developpement (chunks de film ; + artefacts deja cuits en mode parc) ET
// l'acces en lecture a la base partagee du titre (pour les faits du match, via
// `levelup replay-facts-export` en sous-processus — la SEULE etape qui exige CGO/gcc). PAS le
// jeu installe : la cuisson elle-meme (`cmd/replay-build`, compile a la volee pour le HEAD et,
// en mode base, pour la revision de base) ne lit que des catalogues VERSIONNES
// (data/titles/{slug}/reference), jamais l'installation — contrairement au tag `gamefiles`
// dont ce gate ne partage que l'ESPRIT (ressource locale volumineuse, absente en CI,
// degradation propre plutot qu'echec).
//
// # USAGE
//
//	cd apps/go-api && go run ./cmd/replay-corpus-gate \
//	  [--reference=base|parc] [--base REV] [--strict] \
//	  [--manifest config/replay_corpus.toml] [--parc-root DIR] [--lock-root DIR] \
//	  [--source-root DIR] [--work-root DIR] [--keep-work] [--json rapport.json]
//
// Racines (cf. roots.go et base.go pour le detail) : --source-root = le depot ou ce binaire
// tourne (code + config au HEAD teste, defaut title.FindRepoRoot) ; --parc-root = le parc de
// developpement (chunks, artefacts de reference en mode parc, defaut auto-detecte par le
// .git commun) ; --lock-root = ou poser le verrou de decodage PARTAGE avec tout autre outil
// de cuisson de ce depot (defaut CacheRootDir du parc).
//
// Codes de sortie : 0 = aucune perte bloquante, 1 = au moins un temoin porte une perte
// bloquante ou une erreur de cuisson/comparaison, 2 = usage ou manifeste invalide. Un temoin
// ABSENT est un avertissement slog, jamais un echec (cf. codeSortie).
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"levelup/go-api/internal/games"
)

func main() {
	reference := flag.String("reference", "base", "reference de comparaison : base (cuisson a une revision anterieure, defaut) ou parc (artefact deja cuit, informatif sauf --strict)")
	baseFlag := flag.String("base", "", "revision de base explicite (mode --reference=base ; defaut : origin/feat/v75 si HEAD en differe, sinon HEAD^)")
	strict := flag.Bool("strict", false, "en mode --reference=parc, une perte fait sortir en code 1 (sans effet en mode base, deja bloquant)")
	manifestPath := flag.String("manifest", "", "chemin du manifeste (defaut : <source-root>/config/replay_corpus.toml)")
	parcRootFlag := flag.String("parc-root", "", "racine du parc de developpement (defaut : auto-detecte par git)")
	lockRootFlag := flag.String("lock-root", "", "racine du verrou de decodage partage (defaut : CacheRootDir du parc)")
	sourceRootFlag := flag.String("source-root", "", "depot ou lire le code/la config au HEAD (defaut : title.FindRepoRoot)")
	workRootFlag := flag.String("work-root", "", "racine de travail temporaire (defaut : dossier temporaire jetable)")
	keepWork := flag.Bool("keep-work", false, "conserver la racine de travail apres l'execution (debug)")
	sortieJSON := flag.String("json", "", "fichier ou ecrire le rapport JSON complet (vide = aucun)")
	flag.Parse()

	opts := executerOptions{
		Reference: *reference, Base: *baseFlag, Strict: *strict,
		ManifestPath: *manifestPath, ParcRootFlag: *parcRootFlag, LockRootFlag: *lockRootFlag,
		SourceRootFlag: *sourceRootFlag, WorkRootFlag: *workRootFlag, KeepWork: *keepWork,
		SortieJSON: *sortieJSON,
	}
	if err := executer(opts); err != nil {
		slog.Error("replay-corpus-gate", "err", err)
		os.Exit(2)
	}
}

// executerOptions porte les options de la ligne de commande — un struct plutot qu'une
// signature a onze parametres (CLAUDE.md n°5).
type executerOptions struct {
	Reference, Base                                                                    string
	Strict, KeepWork                                                                   bool
	ManifestPath, ParcRootFlag, LockRootFlag, SourceRootFlag, WorkRootFlag, SortieJSON string
}

// executer orchestre le gate de bout en bout : resolution des racines, chargement du
// manifeste, compilation du ou des binaires de cuisson, une cuisson-comparaison par temoin,
// impression et verdict.
func executer(o executerOptions) error {
	if o.Reference != "base" && o.Reference != "parc" {
		return fmt.Errorf("--reference invalide %q : attendu \"base\" ou \"parc\"", o.Reference)
	}

	sourceRoot, err := resolveSourceRoot(o.SourceRootFlag)
	if err != nil {
		return fmt.Errorf("racine source : %w", err)
	}
	manifestPath := o.ManifestPath
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
	parcRoot, err := resolveParcRoot(o.ParcRootFlag, titleSlug)
	if err != nil {
		return fmt.Errorf("racine du parc : %w", err)
	}
	lockRoot := resolveLockRoot(o.LockRootFlag, parcRoot)

	caps, err := games.LoadCapabilityMap(sourceRoot, titleSlug)
	if err != nil {
		return fmt.Errorf("capabilities du titre %s : %w", titleSlug, err)
	}
	if !caps.Has(games.CapFilmReplayArtifact) {
		return fmt.Errorf("le titre %s ne declare pas la capability %s — aucun artefact a comparer",
			titleSlug, games.CapFilmReplayArtifact)
	}

	workRoot, cleanup, err := prepareWorkRoot(o.WorkRootFlag, o.KeepWork)
	if err != nil {
		return fmt.Errorf("racine de travail : %w", err)
	}
	defer cleanup()

	slog.Info("replay-corpus-gate: racines resolues",
		"reference", o.Reference, "source", sourceRoot, "parc", parcRoot, "verrou", lockRoot,
		"travail", workRoot, "manifeste", manifestPath, "temoins", len(manifest.Temoins))

	goAPIDirHead := filepath.Join(sourceRoot, "apps", "go-api")
	if err := stageReferenceOnce(sourceRoot, workRoot, titleSlug); err != nil {
		return fmt.Errorf("catalogues de reference (HEAD) : %w", err)
	}
	binHead := filepath.Join(workRoot, "bin", "replay-build-head"+exeSuffix())
	if err := compilerReplayBuild(goAPIDirHead, filepath.Join(workRoot, "gocache-head"), binHead); err != nil {
		return fmt.Errorf("compilation replay-build (HEAD) : %w", err)
	}

	ctx := temoinContexte{
		ParcRoot: parcRoot, WorkRoot: workRoot, BinHead: binHead,
		LockRoot: lockRoot, TitleSlug: titleSlug, Reference: o.Reference,
	}
	refLabel := "parc"

	if o.Reference == "base" {
		refLabel, err = preparerReferenceBase(sourceRoot, workRoot, titleSlug, o.Base, &ctx, &cleanup)
		if err != nil {
			return err
		}
	}

	factsDir := filepath.Join(workRoot, "facts")
	ctx.FactsDir = factsDir
	ids := make([]string, len(manifest.Temoins))
	for i, t := range manifest.Temoins {
		ids[i] = t.ID
	}
	if err := exportFacts(goAPIDirHead, parcRoot, titleSlug, factsDir, ids); err != nil {
		return fmt.Errorf("export des faits : %w", err)
	}

	lignes := make([]ligneRapport, 0, len(manifest.Temoins))
	for _, t := range manifest.Temoins {
		lignes = append(lignes, traiterTemoin(t, ctx))
	}

	imprimerTableau(os.Stdout, lignes, refLabel)
	imprimerDetailPertes(os.Stdout, lignes)
	if o.SortieJSON != "" {
		if err := ecrireRapportJSON(o.SortieJSON, lignes); err != nil {
			return err
		}
	}

	pertesBloquent := o.Reference == "base" || o.Strict
	if code := codeSortie(lignes, pertesBloquent); code != 0 {
		os.Exit(code)
	}
	return nil
}

// preparerReferenceBase resout la revision de base, cree son worktree detache (nettoye a la
// fin de `executer` via le `cleanup` compose dans `*previousCleanup`), y compile replay-build,
// et peuple les champs base de `ctx`. Rend le libelle de colonne a afficher.
func preparerReferenceBase(sourceRoot, workRoot, titleSlug, baseFlag string, ctx *temoinContexte, previousCleanup *func()) (string, error) {
	baseRev, err := resolveBaseRevision(baseFlag, sourceRoot)
	if err != nil {
		return "", fmt.Errorf("resolution de la base : %w", err)
	}
	wtBase, cleanupWt, err := creerWorktreeBase(sourceRoot, workRoot, baseRev)
	if err != nil {
		return "", fmt.Errorf("worktree de base (%s) : %w", baseRev, err)
	}
	dejaLa := *previousCleanup
	*previousCleanup = func() { cleanupWt(); dejaLa() }

	workRootBase := filepath.Join(workRoot, "cuisson-base")
	if err := stageReferenceOnce(wtBase.Chemin, workRootBase, titleSlug); err != nil {
		return "", fmt.Errorf("catalogues de reference (base) : %w", err)
	}
	binBase := filepath.Join(workRoot, "bin", "replay-build-base"+exeSuffix())
	if err := compilerReplayBuild(wtBase.GoAPIDir, filepath.Join(workRoot, "gocache-base"), binBase); err != nil {
		return "", fmt.Errorf("compilation replay-build (base %s) : %w", baseRev, err)
	}

	ctx.WorkRootBase, ctx.BinBase = workRootBase, binBase
	slog.Info("replay-corpus-gate: base resolue", "revision", baseRev, "worktree", wtBase.Chemin)
	return "base(" + baseRev + ")", nil
}

// exeSuffix rend ".exe" sur Windows, "" ailleurs — le seul endroit qui le sait, pour ne pas
// le repeter aux deux sites qui nomment un binaire compile a la volee.
func exeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
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
		ID              string       `json:"id"`
		Famille         string       `json:"famille"`
		Absent          bool         `json:"absent,omitempty"`
		Erreur          string       `json:"erreur,omitempty"`
		SchemaReference int          `json:"schemaReference,omitempty"`
		SchemaHEAD      int          `json:"schemaHead,omitempty"`
		Gains           int          `json:"gains"`
		Pertes          int          `json:"pertes"`
		DureeMS         int64        `json:"dureeMs"`
		Detail          []detailJSON `json:"pertesDetail,omitempty"`
	}
	out := make([]ligneJSON, len(lignes))
	for i, l := range lignes {
		lj := ligneJSON{
			ID: l.Temoin.ID, Famille: l.Temoin.Famille, Absent: l.Absent,
			SchemaReference: l.SchemaReference, SchemaHEAD: l.SchemaHEAD,
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
