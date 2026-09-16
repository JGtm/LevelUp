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
// HEAD^ — cf. base.go) — puis compare les deux cuissons fraiches. Toute perte = codePerte. C'est
// le gate a lancer avant merge : le signal est binaire, une perte est necessairement due au
// diff en cours de revue, jamais a l'age d'un parc jamais a jour.
//
// --reference=parc : compare le HEAD a l'artefact DEJA CUIT dans le parc local (methode
// historique, balayage de release). INFORMATIF par defaut (imprime le tableau, sort en 0) —
// --strict le rend bloquant. Un gate qui rendrait PERTE sur tous les temoins au meilleur etat
// connu ne gaterait rien : le parc n'est jamais a jour, cf. le tableau paragraphe 1 de
// BALAYAGE_PARC_2026-09-06.md.
//
// Dans les deux modes, aucun schema n'est bumpe — le gate COMPARE.
//
// # CE QU'IL EXIGE
//
// Le PARC LOCAL de developpement (chunks de film ; + artefacts deja cuits en mode parc) ET
// l'acces en lecture a la base partagee du titre (pour les faits du match, via
// levelup replay-facts-export en sous-processus — la SEULE etape qui exige CGO/gcc). PAS le
// jeu installe : la cuisson elle-meme (cmd/replay-build, compile a la volee pour le HEAD et,
// en mode base, pour la revision de base) ne lit que des catalogues VERSIONNES
// (data/titles/{slug}/reference), jamais l'installation — contrairement au tag gamefiles
// dont ce gate ne partage que l'ESPRIT (ressource locale volumineuse, absente en CI,
// degradation propre plutot qu'echec).
//
// # USAGE
//
//	cd apps/go-api && go run ./cmd/replay-corpus-gate \
//	  [--reference=base|parc] [--base REV] [--strict] [--allow-missing] \
//	  [--manifest config/replay_corpus.toml] [--temoins a,b] [--parc-root DIR] \
//	  [--lock-root DIR] [--source-root DIR] [--work-root DIR] [--keep-work] \
//	  [--json rapport.json]
//
// --temoins rejoue les SEULS temoins nommes, sans fabriquer de manifeste reduit (D2 (cloture
// M1)) : le corpus versionne reste le seul corpus. Un id inconnu est une erreur (codeUsage),
// jamais un manifeste ampute en silence.
//
// Racines (cf. roots.go et base.go pour le detail) : --source-root = le depot ou ce binaire
// tourne (code + config au HEAD teste, defaut : git rev-parse --show-toplevel) ; --parc-root =
// le parc de developpement (chunks, artefacts de reference en mode parc, defaut : sourceRoot
// s'il porte deja la base du titre, sinon auto-detecte par le .git commun) ; --lock-root = ou
// poser le verrou de decodage PARTAGE avec tout autre outil de cuisson de ce depot (defaut
// CacheRootDir du parc).
//
// # CODES DE SORTIE (constantes nommees dans report.go, 2026-09-17)
//
// Ils se lisent sans parser le tableau, et chacun dit UNE chose :
//
//	0  codeOK                   tout le manifeste a ete compare, aucun temoin bloquant.
//	1  codePerte                au moins un temoin compare porte une PERTE ou un CHANGEMENT
//	                            bloquant. LE verdict de ce gate.
//	2  codeUsage                le gate n'a pas DEMARRE : drapeau invalide, manifeste illisible,
//	                            racine introuvable, capability absente, worktree de base
//	                            impossible. Rien n'a ete mesure du diff sous revue.
//	3  codeErreurCuisson        le gate a demarre, mais un temoin CUIT a echoue a la cuisson ou
//	                            a la comparaison. Distinct du 1 : la question n'a pas pu etre
//	                            posee sur ce temoin, la reponse n'est pas « il a perdu ».
//	4  codeCouvertureIncomplete au moins un temoin est ABSENT (film, faits ou artefact de
//	                            reference manquants) sans --allow-missing — CORPUS-R1 C3 : un
//	                            cache de film purge ne doit jamais faire sortir ce gate en 0
//	                            sans rien comparer. Distinct du 1 ET du 2 (D2 (cloture M1)) :
//	                            le manifeste est valide, aucun temoin n'a perdu, il en manque.
//	                            --allow-missing restaure l'ancien comportement (un temoin
//	                            absent est un avertissement slog, jamais un echec).
//
// AVANT LE 2026-09-17, le `2` disait a la fois « manifeste invalide » et « temoin absent » : un
// appelant ne pouvait pas distinguer « ce gate n'a pas demarre » de « ce gate a demarre mais
// n'a pas tout compare », et une erreur de cuisson se confondait avec une perte sous le `1`.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/replaydiff"
)

// referenceBase / referenceParc : les deux valeurs valides de --reference — centralisees ici
// (goconst : "base" apparaissait 4 fois en litteral dans ce paquet).
const (
	referenceBase = "base"
	referenceParc = "parc"
)

func main() {
	reference := flag.String("reference", referenceBase, "reference de comparaison : base (cuisson a une revision anterieure, defaut) ou parc (artefact deja cuit, informatif sauf --strict)")
	baseFlag := flag.String("base", "", "revision de base explicite (mode --reference=base ; defaut : origin/feat/v75 si HEAD en differe, sinon HEAD^)")
	strict := flag.Bool("strict", false, "en mode --reference=parc, une perte ou un changement fait sortir en code 1 (sans effet en mode base, deja bloquant)")
	manifestPath := flag.String("manifest", "", "chemin du manifeste (defaut : <source-root>/config/replay_corpus.toml)")
	parcRootFlag := flag.String("parc-root", "", "racine du parc de developpement (defaut : source-root s'il porte la base du titre, sinon auto-detecte par git)")
	lockRootFlag := flag.String("lock-root", "", "racine du verrou de decodage partage (defaut : CacheRootDir du parc)")
	sourceRootFlag := flag.String("source-root", "", "depot ou lire le code/la config au HEAD (defaut : git rev-parse --show-toplevel)")
	workRootFlag := flag.String("work-root", "", "racine de travail temporaire (defaut : dossier temporaire jetable)")
	keepWork := flag.Bool("keep-work", false, "conserver la racine de travail apres l'execution (debug)")
	sortieJSON := flag.String("json", "", "fichier ou ecrire le rapport JSON complet (vide = aucun)")
	allowMissing := flag.Bool("allow-missing", false, "tolerer un temoin ABSENT (avertissement seul, code 0 possible) au lieu de refuser la couverture incomplete (code 4, defaut)")
	temoinsFlag := flag.String("temoins", "", "rejouer les SEULS temoins nommes (ids separes par des virgules) — le manifeste versionne reste le corpus, aucun manifeste reduit a ecrire ; un id inconnu est une erreur")
	flag.Parse()

	opts := executerOptions{
		Reference: *reference, Base: *baseFlag, Strict: *strict, AllowMissing: *allowMissing,
		ManifestPath: *manifestPath, ParcRootFlag: *parcRootFlag, LockRootFlag: *lockRootFlag,
		SourceRootFlag: *sourceRootFlag, WorkRootFlag: *workRootFlag, KeepWork: *keepWork,
		SortieJSON: *sortieJSON, Temoins: *temoinsFlag,
	}

	// signal.NotifyContext, PAS un handler qui appellerait os.Exit lui-meme : ce gate dure 13 a
	// 25 min, une interruption manuelle (Ctrl-C) doit annuler ctx (interrompt l'attente
	// bornee du verrou partage, bake.go) et laisser executer() RETOURNER normalement — c'est ce
	// qui joue le nettoyage compose (CORPUS-R1 C1/C2 : un os.Exit dans un handler de signal
	// sauterait les memes defers qu'un os.Exit au milieu de executer()).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	code, err := executer(ctx, opts)
	if err != nil {
		slog.Error("replay-corpus-gate", "err", err)
	}
	os.Exit(code)
}

// executerOptions porte les options de la ligne de commande — un struct plutot qu'une
// signature a onze parametres (CLAUDE.md n°5).
type executerOptions struct {
	Reference, Base                                                                    string
	Strict, KeepWork, AllowMissing                                                     bool
	ManifestPath, ParcRootFlag, LockRootFlag, SourceRootFlag, WorkRootFlag, SortieJSON string
	// Temoins : la valeur brute de --temoins (ids separes par des virgules, vide = tout le
	// manifeste). Decoupee par NomsDemandes, appliquee par FiltrerTemoins.
	Temoins string
}

// environnementGate regroupe la resolution des racines et du manifeste — un struct plutot
// qu'une signature a six valeurs de retour.
type environnementGate struct {
	SourceRoot, ParcRoot, LockRoot, ManifestPath, TitleSlug string
	Manifest                                                Manifest
}

// executer orchestre le gate de bout en bout. Ne quitte JAMAIS le processus lui-meme (pas
// d'os.Exit ici) : le code rendu est le SEUL canal de sortie, pour que le nettoyage compose
// (defer juste apres sa creation) joue toujours en entier avant que main() n'appelle os.Exit
// (CORPUS-R1 C1/C2).
func executer(ctx context.Context, o executerOptions) (int, error) {
	if o.Reference != referenceBase && o.Reference != referenceParc {
		return codeUsage, fmt.Errorf("--reference invalide %q : attendu \"base\" ou \"parc\"", o.Reference)
	}

	env, err := chargerEnvironnement(ctx, o)
	if err != nil {
		return codeUsage, err
	}

	nettoyeur := &nettoyeurCompose{}
	defer func() { nettoyeur.Executer() }()

	workRoot, cleanupWorkRoot, err := prepareWorkRoot(o.WorkRootFlag, o.KeepWork)
	if err != nil {
		return codeUsage, fmt.Errorf("racine de travail : %w", err)
	}
	nettoyeur.Ajouter(cleanupWorkRoot)

	slog.Info("replay-corpus-gate: racines resolues",
		"reference", o.Reference, "source", env.SourceRoot, "parc", env.ParcRoot,
		"verrou", env.LockRoot, "travail", workRoot, "manifeste", env.ManifestPath,
		"temoins", len(env.Manifest.Temoins))

	binHead, err := preparerCuissonHead(ctx, env.SourceRoot, workRoot, env.TitleSlug)
	if err != nil {
		return codeUsage, err
	}

	tc := temoinContexte{
		ParcRoot: env.ParcRoot, WorkRoot: workRoot, BinHead: binHead,
		LockRoot: env.LockRoot, TitleSlug: env.TitleSlug, Reference: o.Reference,
	}
	refLabel := referenceParc
	if o.Reference == referenceBase {
		bp := basePrepParams{SourceRoot: env.SourceRoot, WorkRoot: workRoot, BaseFlag: o.Base}
		refLabel, err = preparerReferenceBase(ctx, bp, &tc, nettoyeur)
		if err != nil {
			return codeUsage, err
		}
	}

	tc.FactsDir = filepath.Join(workRoot, "facts")
	ep := exportParams{
		GoAPIDir: filepath.Join(env.SourceRoot, "apps", "go-api"),
		ParcRoot: env.ParcRoot, TitleSlug: env.TitleSlug, FactsDir: tc.FactsDir,
	}
	if err := exportFacts(ctx, ep, idsDuManifeste(env.Manifest)); err != nil {
		return codeUsage, fmt.Errorf("export des faits : %w", err)
	}

	lignes := cuireEtComparerTousLesTemoins(ctx, env.Manifest, tc)
	return finaliser(lignes, refLabel, o)
}

// chargerEnvironnement resout les trois racines, charge le manifeste et verifie la capability
// du titre. Aucune ressource jetable creee ici (rien a nettoyer).
func chargerEnvironnement(ctx context.Context, o executerOptions) (environnementGate, error) {
	sourceRoot, err := resolveSourceRoot(ctx, o.SourceRootFlag)
	if err != nil {
		return environnementGate{}, fmt.Errorf("racine source : %w", err)
	}
	manifestPath := o.ManifestPath
	if manifestPath == "" {
		manifestPath = filepath.Join(sourceRoot, "config", "replay_corpus.toml")
	}
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return environnementGate{}, err
	}
	manifest, err = FiltrerTemoins(manifest, NomsDemandes(o.Temoins))
	if err != nil {
		return environnementGate{}, err
	}
	titleSlug := manifest.Meta.TitleSlug

	// La resolution du parc a besoin du titre pour VALIDER l'auto-detection (presence de sa
	// base partagee) — d'ou l'ordre : manifeste avant parc.
	parcRoot, err := resolveParcRoot(ctx, o.ParcRootFlag, sourceRoot, titleSlug)
	if err != nil {
		return environnementGate{}, fmt.Errorf("racine du parc : %w", err)
	}
	lockRoot := resolveLockRoot(o.LockRootFlag, parcRoot)

	caps, err := games.LoadCapabilityMap(sourceRoot, titleSlug)
	if err != nil {
		return environnementGate{}, fmt.Errorf("capabilities du titre %s : %w", titleSlug, err)
	}
	if !caps.Has(games.CapFilmReplayArtifact) {
		return environnementGate{}, fmt.Errorf(
			"le titre %s ne declare pas la capability %s — aucun artefact a comparer",
			titleSlug, games.CapFilmReplayArtifact)
	}

	return environnementGate{
		SourceRoot: sourceRoot, ParcRoot: parcRoot, LockRoot: lockRoot,
		ManifestPath: manifestPath, TitleSlug: titleSlug, Manifest: manifest,
	}, nil
}

// preparerCuissonHead copie les catalogues de reference VERSIONNES au HEAD (en avertissant si
// l'arbre de travail les modifie localement, CORPUS-R1 C11) et compile cmd/replay-build depuis
// le code au HEAD.
func preparerCuissonHead(ctx context.Context, sourceRoot, workRoot, titleSlug string) (binHead string, err error) {
	avertirSiCatalogueModifie(ctx, sourceRoot, titleSlug)
	if err := stageReferenceOnce(sourceRoot, workRoot, titleSlug); err != nil {
		return "", fmt.Errorf("catalogues de reference (HEAD) : %w", err)
	}
	binHead = filepath.Join(workRoot, "bin", "replay-build-head"+exeSuffix())
	goAPIDirHead := filepath.Join(sourceRoot, "apps", "go-api")
	if err := compilerReplayBuild(ctx, goAPIDirHead, filepath.Join(workRoot, "gocache-head"), binHead); err != nil {
		return "", fmt.Errorf("compilation replay-build (HEAD) : %w", err)
	}
	return binHead, nil
}

// basePrepParams regroupe les chemins fixes de la preparation de la reference base — un struct
// plutot qu'une signature a plus de 5 parametres une fois `ctx` ajoute (CLAUDE.md n°5).
type basePrepParams struct {
	SourceRoot, WorkRoot, BaseFlag string
}

// preparerReferenceBase resout la revision de base, cree son worktree detache (nettoye via
// nettoyeur, compose sans jamais reassigner une closure sous un defer deja arme —
// CORPUS-R1 C1), y compile replay-build, et peuple les champs base de tc. Rend le libelle de
// colonne a afficher.
func preparerReferenceBase(ctx context.Context, p basePrepParams, tc *temoinContexte, nettoyeur *nettoyeurCompose) (string, error) {
	baseRev, err := resolveBaseRevision(ctx, p.BaseFlag, p.SourceRoot)
	if err != nil {
		return "", fmt.Errorf("resolution de la base : %w", err)
	}
	wtBase, cleanupWt, err := creerWorktreeBase(ctx, p.SourceRoot, p.WorkRoot, baseRev)
	if err != nil {
		return "", fmt.Errorf("worktree de base (%s) : %w", baseRev, err)
	}
	nettoyeur.Ajouter(cleanupWt)

	workRootBase := filepath.Join(p.WorkRoot, "cuisson-base")
	if err := stageReferenceOnce(wtBase.Chemin, workRootBase, tc.TitleSlug); err != nil {
		return "", fmt.Errorf("catalogues de reference (base) : %w", err)
	}
	binBase := filepath.Join(p.WorkRoot, "bin", "replay-build-base"+exeSuffix())
	if err := compilerReplayBuild(ctx, wtBase.GoAPIDir, filepath.Join(p.WorkRoot, "gocache-base"), binBase); err != nil {
		return "", fmt.Errorf("compilation replay-build (base %s) : %w", baseRev, err)
	}

	tc.WorkRootBase, tc.BinBase = workRootBase, binBase
	slog.Info("replay-corpus-gate: base resolue", "revision", baseRev, "worktree", wtBase.Chemin)
	return "base(" + baseRev + ")", nil
}

// idsDuManifeste extrait les ids, dans l'ordre du manifeste — l'entree de exportFacts.
func idsDuManifeste(m Manifest) []string {
	ids := make([]string, len(m.Temoins))
	for i, t := range m.Temoins {
		ids[i] = t.ID
	}
	return ids
}

// cuireEtComparerTousLesTemoins cuit et compare chaque temoin du manifeste, DANS L'ORDRE,
// SEQUENTIELLEMENT — jamais deux cuissons en parallele DANS ce processus (cf.
// internal/archlint/no_unbounded_film_loop_test.go). S'arrete tot si ctx est deja annule
// (interruption) : les temoins restants ne sont alors pas meme tentes.
func cuireEtComparerTousLesTemoins(ctx context.Context, manifest Manifest, tc temoinContexte) []ligneRapport {
	lignes := make([]ligneRapport, 0, len(manifest.Temoins))
	for _, t := range manifest.Temoins {
		if err := ctx.Err(); err != nil {
			slog.Warn("replay-corpus-gate: interruption — temoins restants non tentes",
				"temoin", t.ID, "err", err)
			break
		}
		lignes = append(lignes, traiterTemoin(ctx, t, tc))
	}
	return lignes
}

// finaliser imprime le tableau et LES DEUX sections de detail, ecrit le rapport JSON optionnel,
// verifie la couverture (CORPUS-R1 C3) puis rend le code de sortie.
//
// L'ECRITURE DU JSON PRECEDE LA VERIFICATION DE COUVERTURE, et c'est voulu : un run a temoins
// absents rend `codeCouvertureIncomplete`, et son rapport doit exister quand meme — c'est lui
// qui dit LESQUELS relancer.
func finaliser(lignes []ligneRapport, refLabel string, o executerOptions) (int, error) {
	imprimerTableau(os.Stdout, lignes, refLabel)
	imprimerDetailPertes(os.Stdout, lignes)
	imprimerDetailChangements(os.Stdout, lignes)
	if o.SortieJSON != "" {
		if err := ecrireRapportJSON(o.SortieJSON, lignes); err != nil {
			return codeUsage, err
		}
	}
	if err := verifierCouverture(lignes, o.AllowMissing); err != nil {
		return codeCouvertureIncomplete, err
	}
	pertesBloquent := o.Reference == referenceBase || o.Strict
	return codeSortie(lignes, pertesBloquent), nil
}

// exeSuffix rend ".exe" sur Windows, "" ailleurs — le seul endroit qui le sait, pour ne pas
// le repeter aux deux sites qui nomment un binaire compile a la volee.
func exeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

// detailJSON est UN ecart nomme du rapport JSON — la forme partagee par `pertesDetail` et
// `changementsDetail`, pour que les deux sections restent identiques champ pour champ.
type detailJSON struct {
	Axe      string `json:"axe"`
	Metrique string `json:"metrique"`
	Sens     string `json:"sens"`
	Ancien   string `json:"ancien,omitempty"`
	Nouveau  string `json:"nouveau,omitempty"`
}

// ligneJSON est UN temoin du rapport JSON.
//
// `statut` et `absentCause` sont apparus le 2026-09-17 (D5 (1.9.9)) : un lecteur qui n'avait
// que le JSON lisait `{"gains":0,"pertes":0,"changements":0}` pour un temoin en ERREUR comme
// pour un temoin propre, et comptait donc des temoins conclus qui ne l'etaient pas. `statut`
// porte desormais le MEME verdict que la derniere colonne du tableau, toujours renseigne :
// « 0 difference » et « pas compare » ne se confondent plus.
//
// `changementsDetail` (meme date, meme decouverte) est le symetrique de `pertesDetail` : sans
// lui, le JSON disait « 2 changements » sans jamais dire LESQUELS, et la cloture M1 a du
// relancer `replay-diff` a la main sur les artefacts conserves pour les nommer (plan §5).
type ligneJSON struct {
	ID                string       `json:"id"`
	Famille           string       `json:"famille"`
	Statut            string       `json:"statut"`
	Absent            bool         `json:"absent,omitempty"`
	AbsentCause       string       `json:"absentCause,omitempty"`
	Erreur            string       `json:"erreur,omitempty"`
	SchemaReference   int          `json:"schemaReference,omitempty"`
	SchemaHEAD        int          `json:"schemaHead,omitempty"`
	Gains             int          `json:"gains"`
	Pertes            int          `json:"pertes"`
	Changements       int          `json:"changements"`
	DureeMS           int64        `json:"dureeMs"`
	PertesDetail      []detailJSON `json:"pertesDetail,omitempty"`
	ChangementsDetail []detailJSON `json:"changementsDetail,omitempty"`
}

// ligneVersJSON traduit UNE ligne de rapport en sa forme serialisable.
func ligneVersJSON(l ligneRapport) ligneJSON {
	lj := ligneJSON{
		ID: l.Temoin.ID, Famille: l.Temoin.Famille, Statut: l.statut(),
		Absent: l.Absent, AbsentCause: l.AbsentCause,
		SchemaReference: l.SchemaReference, SchemaHEAD: l.SchemaHEAD,
		Gains: l.Gains, Pertes: l.Pertes, Changements: l.Changements,
		DureeMS: l.Duree.Milliseconds(),
	}
	if l.Erreur != nil {
		lj.Erreur = l.Erreur.Error()
	}
	lj.PertesDetail = detailsVersJSON(l.PertesDetail)
	lj.ChangementsDetail = detailsVersJSON(l.ChangementsDetail)
	return lj
}

// detailsVersJSON traduit une liste d'ecarts — un seul endroit, pour les deux sections.
func detailsVersJSON(diffs []replaydiff.Difference) []detailJSON {
	if len(diffs) == 0 {
		return nil
	}
	out := make([]detailJSON, len(diffs))
	for i, d := range diffs {
		out[i] = detailJSON{
			Axe: d.Axe, Metrique: d.Metrique, Sens: d.Sens, Ancien: d.Ancien, Nouveau: d.Nouveau,
		}
	}
	return out
}

// ecrireRapportJSON depose le detail des lignes, pour un consommateur automatique (CI, un
// futur tableau de bord) — le tableau texte reste la sortie lisible par un operateur.
func ecrireRapportJSON(path string, lignes []ligneRapport) error {
	out := make([]ligneJSON, len(lignes))
	for i, l := range lignes {
		out[i] = ligneVersJSON(l)
	}
	return ecrireJSONGenerique(path, out)
}
