package main

// options.go — LA LIGNE DE COMMANDE : quatre gestes, les options de chacun, et le SEUL endroit qui
// valide leurs combinaisons.
//
// Les quatre gestes de l'outil. Une option se rattache à l'un d'eux, sinon elle n'existe pas
// (`TestChaqueOptionSeRattacheAUnGeste`) :
//
//	cuire des cartes                  --maps --title --out-dir --natives --forge --style --mpp --avis
//	refaire le plan sans recuire      --plan-seul, avec --maps --title --out-dir --natives --forge --avis
//	ajouter le relief sans recuire    --relief-seul, avec --maps --title --out-dir --natives --forge --avis
//	produire le tableau avant / après --rapport-tsv [--avant-dir], avec une cuisson, un plan seul ou un relief seul

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"levelup/go-api/internal/domain/title"

	"github.com/JGtm/halo-infinite-map-images/himap"
)

// geste groupe les options d'un des quatre gestes.
type geste struct {
	titre   string
	options []string
}

// gestes : la table qui range chaque option sous son geste. L'aide s'écrit depuis elle.
var gestes = []geste{
	{"Cuire des cartes (geste par défaut)",
		[]string{"maps", "title", "out-dir", "natives", "forge", "style", "mpp", "avis"}},
	{"Refaire le plan des murs et l'image plan sans recuire (s'ajoute à la sélection de la cuisson)",
		[]string{"plan-seul"}},
	{"Ajouter le relief aux fonds publiés sans les recuire (s'ajoute à la sélection de la cuisson ; exclut -plan-seul)",
		[]string{"relief-seul"}},
	{"Produire le tableau avant / après (s'ajoute à une cuisson, à un plan seul ou à un relief seul)",
		[]string{"rapport-tsv", "avant-dir"}},
}

// rapportsDemandes : où écrire le rapport en TSV, et où lire les sidecars d'avant.
type rapportsDemandes struct {
	tsv, avantDir string
}

// options est tout ce que la ligne de commande décide.
type options struct {
	// Sélection et sortie, communes à la cuisson et au plan seul.
	titleSlug, maps, outDir string
	natives, forge          bool
	avis                    string
	// Cuisson seulement.
	style string
	mpp   float64
	// Plan seul et relief seul : deux gestes qui ne recuisent pas le fond, et qui s'excluent.
	planSeul, reliefSeul bool
	// Tableau avant / après.
	rapports rapportsDemandes
	// poses : les options écrites sur la ligne de commande, pour refuser celles qu'un geste ignore.
	poses map[string]bool
}

// declareOptions déclare toutes les options de l'outil sur `fs`, liées à `o`.
func declareOptions(fs *flag.FlagSet, o *options) {
	fs.StringVar(&o.titleSlug, "title", title.DefaultSlug, "slug du titre")
	fs.StringVar(&o.maps, "maps", "", "clés de carte séparées par des virgules — dossier installé (native) "+
		"ou map_id UGC (Forge) ; vide = tout ce qui est installé")
	fs.StringVar(&o.outDir, "out-dir", "", "répertoire de sortie (défaut : PathResolver.MapBackgroundDir) ; "+
		"avec -plan-seul, le dossier des sidecars à compléter")
	fs.BoolVar(&o.natives, "natives", true, "traiter les cartes natives (cuites dans un module)")
	fs.BoolVar(&o.forge, "forge", true, "traiter les cartes Forge (posées depuis leur .mvar)")
	fs.StringVar(&o.style, "style", string(himap.StyleFondParDefaut), "habillage : "+stylesConnus())
	fs.Float64Var(&o.mpp, "mpp", 0, "cote du pixel en metres (0 = "+fmt.Sprintf("%.4f", himap.EchelleFondCarte)+", celle de production)")
	fs.StringVar(&o.avis, "avis", "", "fichier(s) d'avis du user, separes par des virgules : le premier donne la regle "+
		"(forme du rognage au maillage par carte), les suivants ignorer / reprendre ; vide = comportement d'avant")
	fs.BoolVar(&o.planSeul, "plan-seul", false, "refait le plan des murs et l'image plan des sidecars publies, depuis leur maillage de navigation, sans recuire")
	fs.BoolVar(&o.reliefSeul, "relief-seul", false, "ajoute le relief (image, donnees, deux champs du sidecar) aux fonds publies : la carte est cuite en memoire, "+
		"le relief se projette dans le calage du sidecar publie, l'image du fond, le plan et le calage ne bougent pas")
	fs.StringVar(&o.rapports.tsv, "rapport-tsv", "", "ecrit le rapport en TSV a ce chemin, une ligne par carte traitee")
	fs.StringVar(&o.rapports.avantDir, "avant-dir", "", "dossier des sidecars d'avant (colonne « ancres avant » du rapport TSV) ; exige -rapport-tsv")
}

// litOptions lit la ligne de commande. Une erreur de syntaxe est écrite sur `sortie` par le paquet
// flag ; la cohérence des options entre elles se vérifie dans `valide`.
func litOptions(args []string, sortie io.Writer) (options, error) {
	var o options
	fs := flag.NewFlagSet("mapfond-build", flag.ContinueOnError)
	fs.SetOutput(sortie)
	declareOptions(fs, &o)
	fs.Usage = func() { ecritAide(fs, sortie) }
	if err := fs.Parse(args); err != nil {
		return options{}, err
	}
	o.poses = map[string]bool{}
	fs.Visit(func(f *flag.Flag) { o.poses[f.Name] = true })
	return o, nil
}

// ecritAide écrit l'aide, les options rangées par geste.
func ecritAide(fs *flag.FlagSet, sortie io.Writer) {
	fmt.Fprintln(sortie, "Usage : mapfond-build [options]")
	for _, g := range gestes {
		fmt.Fprintf(sortie, "\n%s\n", g.titre)
		for _, nom := range g.options {
			f := fs.Lookup(nom)
			if f == nil {
				continue
			}
			fmt.Fprintf(sortie, "  -%s (défaut %q)\n      %s\n", f.Name, f.DefValue, f.Usage)
		}
	}
}

// valide est le SEUL endroit qui refuse une combinaison d'options : une option qu'un geste ignorerait
// en silence est une erreur, pas un réglage perdu.
func (o options) valide() error {
	if !himap.StyleFondValide(himap.StyleFond(o.style)) {
		return fmt.Errorf("habillage inconnu %q (connus : %s)", o.style, stylesConnus())
	}
	if o.planSeul && o.reliefSeul {
		return errors.New("-plan-seul et -relief-seul s'excluent : un geste à la fois")
	}
	if geste := o.gesteSeul(); geste != "" {
		var ignorees []string
		for _, nom := range []string{"style", "mpp"} {
			if o.poses[nom] {
				ignorees = append(ignorees, "-"+nom)
			}
		}
		if len(ignorees) > 0 {
			return fmt.Errorf("%s ne s'applique qu'à la cuisson, pas à %s", strings.Join(ignorees, " ni "), geste)
		}
	}
	if !o.natives && !o.forge {
		return errors.New("-natives=false et -forge=false : aucune carte à traiter")
	}
	if o.rapports.avantDir != "" && o.rapports.tsv == "" {
		return errors.New("-avant-dir exige -rapport-tsv")
	}
	return nil
}

// gesteSeul nomme le geste qui ne recuit pas le fond (`-plan-seul`, `-relief-seul`), vide pour la cuisson.
func (o options) gesteSeul() string {
	switch {
	case o.planSeul:
		return "-plan-seul"
	case o.reliefSeul:
		return "-relief-seul"
	}
	return ""
}
