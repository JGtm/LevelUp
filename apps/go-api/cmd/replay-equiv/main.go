// cmd/replay-equiv — LE HARNAIS D'EQUIVALENCE DE LA CUISSON : UN FILM, UN PROCESSUS BORNE.
//
// # CE QU'IL PROUVE
//
// Qu'un refacto du decodeur de film rend des sorties IDENTIQUES A L'OCTET. Il ne compare pas
// seulement l'artefact final : il hache la sortie de CHAQUE balayage (l'observateur vit dans le
// code de production, cf. `replay/observe.go`), ce qui LOCALISE une divergence au balayage pres
// au lieu de dire « quelque chose a change ». Les digests de reference sont figes dans
// `internal/analysis/replay/testdata/equivalence/<short8>.tsv`, avec LES FAITS DU MATCH a cote
// (`<short8>.facts.json`) : sans eux, zones, actions d'objectif, VIP/crane/bombe, socles et
// points d'apparition sont court-circuites et l'equivalence serait VACUANTE.
//
// Chaque fichier de digests s'ouvre par sa VERSION DE GRAMMAIRE (`# digest-grammar: N`, cf.
// digest.GrammarVersion). Le parent la lit AVANT de comparer : une reference figee sous une
// autre grammaire est une panne d'INFRASTRUCTURE (« re-figer par -update »), jamais un ecart de
// decodage — sans ce marqueur, un changement du RENDU se lit comme une regression.
//
// # POURQUOI PARENT ET ENFANT DANS LE MEME BINAIRE
//
// La construction d'artefact est un AMPLIFICATEUR memoire : quatre sinistres (2026-08-20,
// 08-24, 08-26, 08-31) ont le meme motif — une boucle qui enchaine des films dans UN SEUL
// processus. Ce harnais n'enchaine JAMAIS deux films dans un processus : le parent planifie et
// ne decode rien, chaque film nait dans un enfant borne (plafond dur, priorite basse, verrou
// solo en attente bornee) et meurt avec sa RAM. L'ENFANT ECRIT LUI-MEME son fichier de digests
// (`-out`) : le tube du lanceur fusionne stdout et stderr et ne transporte qu'un JOURNAL, jamais
// une donnee de mesure.
//
// # LES DEUX MODES
//
//	replay-equiv [-corpus F] [-films a,b] [-update]   equivalence : digests par etape + artefact
//	replay-equiv -walkers [-films a,b]                divergence des grammaires de decoupage (D3)
//
// `-corpus` ne nomme QUE la liste des films : les digests de reference et les faits vivent a
// l'emplacement canonique (cf. dossierEquivalence) — un fichier de corpus ailleurs ne deplace
// pas la reference.
//
// Exemple (depuis apps/go-api) :
//
//	LEVELUP_REPO_ROOT=<repo> go run ./cmd/replay-equiv -films 000d5950 -update
//	LEVELUP_REPO_ROOT=<repo> go run ./cmd/replay-equiv          # les 11 films du corpus
//
// PLAN_CUISSON_PERF §3 D4b et D3, §4 items 0.4, 0.5 et 0.7.
package main

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/observability/logging"
)

// outilNom : le nom que ce binaire porte dans le journal des protections (verrou, sentinelle,
// priorite). Il apparait tel quel dans le message de refus lu par le prochain operateur.
const outilNom = "replay-equiv"

// options porte les drapeaux des DEUX roles : le parent planifie, l'enfant traite un film.
type options struct {
	repoRoot   string
	titleSlug  string
	corpus     string
	films      string
	update     bool
	memGiB     int
	walkers    bool
	walkersOut string
	// child, film et out ne sont poses que par le PARENT, pour son enfant.
	child bool
	film  string
	out   string
}

func main() {
	o, err := lireDrapeaux()
	if err != nil {
		// Avant la resolution de la racine, rien n'est journalisable proprement : le message
		// part sur stderr et le code 2 signale une erreur d'usage, hors protocole parent/enfant.
		_, _ = os.Stderr.WriteString("replay-equiv: " + err.Error() + "\n")
		os.Exit(2)
	}
	os.Exit(executer(o))
}

// executer aiguille vers l'un des quatre roles et rend le code de sortie.
//
// LE JOURNAL EST INSTALLE ICI, POUR LES DEUX ROLES, ET IL NE L'ETAIT PAS DU TOUT. Sans handler,
// ce binaire gardait le defaut de la bibliotheque : les Debug etaient perdus et — bien plus
// grave pour un harnais d'equivalence — les DEGRADATIONS de `replaybuild` (catalogue de zones
// absent, socles introuvables, faits vides) passaient en silence. `-update` pouvait figer des
// digests de vide sans qu'une seule ligne ne le dise. Le tube du lanceur fusionne stdout et
// stderr de l'enfant : le journal de l'enfant remonte donc dans celui du parent.
func executer(o options) int {
	defer logging.InstallCLILevel(o.repoRoot, logging.ConsoleLevelFromEnv())()
	switch {
	case o.child && o.walkers:
		return enfantWalkers(o)
	case o.child:
		return enfantEquivalence(o)
	case o.walkers:
		return parentWalkers(o)
	default:
		return parentEquivalence(o)
	}
}

// lireDrapeaux analyse la ligne de commande et resout la racine du depot.
func lireDrapeaux() (options, error) {
	var o options
	flag.StringVar(&o.repoRoot, "repo-root", os.Getenv(filmproc.EnvRepoRoot),
		"racine du depot (defaut : "+filmproc.EnvRepoRoot+", sinon detection depuis l'executable)")
	flag.StringVar(&o.titleSlug, "title", title.DefaultSlug, "slug du titre")
	flag.StringVar(&o.corpus, "corpus", "",
		"fichier listant les films a comparer (defaut : CORPUS.txt du dossier d'equivalence)")
	flag.StringVar(&o.films, "films", "",
		"liste de films short8 separes par des virgules — REMPLACE le corpus")
	flag.BoolVar(&o.update, "update", false,
		"ecrit les digests de reference au lieu de les comparer (lots 0, 3 et 4b UNIQUEMENT)")
	flag.IntVar(&o.memGiB, "mem-gib", filmproc.DefaultLimitGiB,
		"plafond memoire de chaque enfant, en gibioctets (0 = desarme)")
	flag.BoolVar(&o.walkers, "walkers", false,
		"mesure la divergence des grammaires de decoupage sur les films du cache (D3, item 0.7)")
	flag.StringVar(&o.walkersOut, "walkers-out", "",
		"fichier TSV de la mesure -walkers (defaut : tmp/walkers.tsv sous la racine)")
	flag.BoolVar(&o.child, "child", false, "INTERNE : role d'enfant, un seul film")
	flag.StringVar(&o.film, "film", "", "INTERNE : le film short8 de l'enfant")
	flag.StringVar(&o.out, "out", "", "INTERNE : fichier de sortie de l'enfant")
	flag.Parse()

	if o.repoRoot == "" {
		root, err := title.FindRepoRoot()
		if err != nil {
			return o, err
		}
		o.repoRoot = root
	}
	if o.corpus == "" {
		o.corpus = filepath.Join(dossierEquivalence(o.repoRoot), "CORPUS.txt")
	}
	if o.walkersOut == "" {
		o.walkersOut = filepath.Join(o.repoRoot, "tmp", "walkers.tsv")
	}
	if o.child && o.film == "" {
		return o, errors.New("role d'enfant sans -film : rien a traiter")
	}
	return o, nil
}

// dossierEquivalence rend l'emplacement CANONIQUE des digests de reference et des faits figes.
//
// Chemin de SOURCE, pas de donnee : il ne passe donc pas par PathResolver (qui resout `data/`).
func dossierEquivalence(repoRoot string) string {
	return filepath.Join(repoRoot, "apps", "go-api",
		"internal", "analysis", "replay", "testdata", "equivalence")
}

// filmsDemandes rend la liste short8 de `-films`, vide si le drapeau ne l'est pas.
func filmsDemandes(liste string) []string {
	var out []string
	for _, part := range strings.Split(liste, ",") {
		if s := strings.TrimSpace(part); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// gio rend un nombre d'octets en gibioctets, pour le recap de l'operateur.
func gio(octets uint64) float64 { return float64(octets) / (1 << 30) }
