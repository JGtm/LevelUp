// cmd/mappower-build — MESURE les signaux dont se derivent les POSITIONS DE FORCE d'une
// carte, et rend de quoi les juger a l'oeil : un CSV par carte (une ligne par cellule
// peuplee) et des PNG de controle (fond de carte + cellules coloriees + contours des zones
// nommees).
//
// A QUOI CA SERT. Une position de force est un lieu qu'une equipe cherche a tenir. Aucune
// source du jeu ne la declare : elle se DERIVE. Ce programme est l'etape de MESURE du
// chantier (.ai/PLAN_POSITIONS_DE_FORCE_2026-09-20.md, etape 1) — il n'ecrit AUCUN
// catalogue de reference, il produit les distributions sur lesquelles la formule de score
// est choisie, puis figee. Le mode production viendra apres le verdict contre l'oracle.
//
// CE QU'IL LIT, ET DANS QUEL ETAT IL LAISSE LA BASE. Rien n'est ecrit en base, nulle part.
// La lecture passe par `duckdb.OpenReadForQuery` — JAMAIS un `OpenReadOnly` force : le
// serveur de developpement peut tenir la base partagee en ecriture, et deux configurations
// concurrentes sur un meme fichier echouent (modele mono-process, ADR 0013/0016). Les
// tables sont lues par leurs VUES `_latest` uniquement (regle ART n°2, ADR 0026) : une
// lecture brute d'une table append-only sert des lignes perimees.
//
// # DEUX MESURES QUI NE SE DEVINENT PAS (2026-09-20, corpus Halo Infinite)
//
//   - LA VARIANTE DE PLAYLIST N'EST PAS UNE CARTE. « Live Fire » et « Live Fire - Ranked »
//     sont deux assets, la meme geometrie et le meme repere monde (meme `level_id`, cf.
//     profile.variantSuffixes). Les compter separement coupe le corpus en deux : 50 + 30
//     matchs au lieu de 80 sur Live Fire, 47 + 31 sur Recharge. L'agregation se fait donc
//     sur `decfilm.NormalizeMapName`, qui rabote « - Ranked » et « Heavies ».
//   - UNE MORT SANS TUEUR N'EST PAS UN DUEL. 6,5 % des 138 382 lignes de
//     `kill_positions_latest` n'ont pas de position de tueur (chute, suicide). Les compter
//     en « morts ici » peindrait les fosses en positions faibles alors que personne ne les
//     tient. Le paquet `analysis/powerpos` les ecarte et les compte.
//
// Usage :
//
//	mappower-build --recensement [--data-root DIR] [--title slug] [--sortie FICHIER]
//	mappower-build --mesure [--data-root DIR] [--title slug] [--sortie DIR]
//	               [--cartes "live fire,recharge"] [--top N] [--sans-png]
//	mappower-build --fusion --mesures DIR --geometrie DIR --sortie DIR
//	               [--data-root DIR] [--title slug] [--cartes ...] [--sans-png]
//
// `--fusion` (item 2bis.D, 2026-09-20) n'ouvre AUCUNE base : il relit les CSV et les
// fichiers de positions des deux passes (`--mesure --reglage v2` et `cmd/mapgeo-build`) et
// ne lit sous `data/` que les fonds et les zones nommees, pour les planches.
//
// `--data-root` est la racine qui CONTIENT `data/` (le `repoRoot` du PathResolver), et non
// le dossier `data` lui-meme : aucun chemin n'est construit a la main ici.
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"levelup/go-api/internal/domain/title"
)

// options rassemble les drapeaux (une struct plutot que six parametres — seuil du depot).
type options struct {
	dataRoot    string
	titleSlug   string
	sortie      string
	cartes      []string
	top         int
	sansPNG     bool
	recensement bool
	mesure      bool
	// fusion : combiner la passe empirique (`mesures`) et la cuisson geometrique
	// (`geometrie`) — cf. fusion.go.
	fusion    bool
	mesures   string
	geometrie string
	// reglage : « v1 » (fige le 2026-09-20, preuve du verdict) ou « v2 » (item 2bis.B).
	reglage string
	// exclure : noms AFFICHES de variantes dont les positions de kill sont ecartees de la
	// mesure, en minuscules. Cf. lireOptions pour la valeur par defaut et sa date.
	exclure map[string]bool
}

// variantesExcluesParDefaut : « Live Fire - Ranked », dont les positions de kill sont
// FAUSSES en base (decouverte n°1 du plan, 2026-09-20 : x_faux = x_juste / 2 + 23,26 m sur
// les lignes decodees avant le 2026-09-15 ; barycentre a 9,88 m de la variante de base).
// Un correctif est en cours dans un chantier voisin (`wt/livefire-killpos`) et la
// recuisson revient a l'utilisateur. EXCLURE plutot que corriger a la lecture : une
// correction ici ne saurait pas distinguer une ligne deja recuite d'une ligne perimee, et
// la doublerait. Critere de retrait de ce defaut (a verifier a chaque passe) :
// `controle_variantes` sous 2 m sur live fire apres recuisson — alors passer `--exclure-variantes ""`.
const variantesExcluesParDefaut = "Live Fire - Ranked"

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	opts, err := lireOptions()
	if err != nil {
		slog.Error("mappower: options invalides", "err", err)
		os.Exit(2)
	}
	res := title.NewPathResolver(opts.dataRoot)

	if opts.fusion {
		// Avant tout chargement de corpus : la fusion n'a pas besoin de la base, et ne
		// doit pas l'ouvrir (le serveur de developpement peut la tenir en ecriture).
		if err := LanceFusion(res, opts); err != nil {
			slog.Error("mappower: fusion impossible", "err", err)
			os.Exit(1)
		}
		return
	}

	corpus, err := ChargeCorpus(res, opts.titleSlug)
	if err != nil {
		slog.Error("mappower: recensement du corpus impossible", "err", err)
		os.Exit(1)
	}

	if opts.recensement {
		if err := EcrisRecensement(corpus, opts.sortie); err != nil {
			slog.Error("mappower: ecriture du recensement impossible", "err", err, "sortie", opts.sortie)
			os.Exit(1)
		}
	}
	if opts.mesure {
		if err := LanceMesure(res, opts, corpus); err != nil {
			slog.Error("mappower: mesure impossible", "err", err)
			os.Exit(1)
		}
	}
}

// lireOptions lit et valide les drapeaux.
func lireOptions() (options, error) {
	var opts options
	var cartes, exclure string
	flag.StringVar(&opts.dataRoot, "data-root", "", "racine qui CONTIENT data/ (defaut : racine du depot)")
	flag.StringVar(&opts.reglage, "reglage", "v1", "reglage du score et de la selection : v1 (fige) ou v2")
	flag.StringVar(&exclure, "exclure-variantes", variantesExcluesParDefaut,
		"noms affiches de variantes dont les kills sont ecartes, separes par des virgules")
	flag.StringVar(&opts.titleSlug, "title", "halo_infinite", "slug du titre")
	flag.StringVar(&opts.sortie, "sortie", "", "fichier (recensement) ou dossier (mesure) de sortie")
	flag.StringVar(&cartes, "cartes", "", "cartes a mesurer, noms normalises separes par des virgules")
	flag.IntVar(&opts.top, "top", 0, "mesurer les N cartes les plus fournies du corpus")
	flag.BoolVar(&opts.sansPNG, "sans-png", false, "ne produire que les CSV (mesure)")
	flag.BoolVar(&opts.recensement, "recensement", false, "ecrire le recensement du corpus par carte")
	flag.BoolVar(&opts.mesure, "mesure", false, "mesurer les accumulateurs par cellule")
	flag.BoolVar(&opts.fusion, "fusion", false, "fusionner la passe empirique v2 et la geometrie (sans base)")
	flag.StringVar(&opts.mesures, "mesures", "", "dossier de la passe --mesure --reglage v2 (fusion)")
	flag.StringVar(&opts.geometrie, "geometrie", "", "dossier de la cuisson cmd/mapgeo-build (fusion)")
	flag.Parse()

	if !opts.recensement && !opts.mesure && !opts.fusion {
		return opts, fmt.Errorf("choisir au moins --recensement, --mesure ou --fusion")
	}
	if opts.fusion && (opts.mesures == "" || opts.geometrie == "" || opts.sortie == "") {
		return opts, fmt.Errorf("--fusion exige --mesures, --geometrie et --sortie")
	}
	if opts.mesure && opts.sortie == "" {
		return opts, fmt.Errorf("--mesure exige --sortie (dossier des CSV et des PNG)")
	}
	if opts.mesure && cartes == "" && opts.top <= 0 {
		return opts, fmt.Errorf("--mesure exige --cartes ou --top")
	}
	if opts.reglage != "v1" && opts.reglage != "v2" {
		return opts, fmt.Errorf("--reglage %q inconnu (v1 ou v2)", opts.reglage)
	}
	for _, c := range strings.Split(cartes, ",") {
		if nom := strings.TrimSpace(c); nom != "" {
			opts.cartes = append(opts.cartes, nom)
		}
	}
	opts.exclure = map[string]bool{}
	for _, v := range strings.Split(exclure, ",") {
		if nom := strings.ToLower(strings.TrimSpace(v)); nom != "" {
			opts.exclure[nom] = true
		}
	}
	if opts.dataRoot == "" {
		racine, err := title.FindRepoRoot()
		if err != nil {
			return opts, fmt.Errorf("racine du depot introuvable et --data-root absent : %w", err)
		}
		opts.dataRoot = racine
	}
	return opts, nil
}
