// cmd/mapplafond-mesure — MESURE ce qu'un plafond deduit des hauteurs jouees couperait sur
// chaque carte. Phase C0 du lot « plafonds » (`.ai/V7.5/PLAN_REPLAY2D_NOTION_2026-08-25.md`,
// point 10 de l'encadre Notion REPLAY 2D).
//
// LA QUESTION PRODUIT : « pour les cartes non validees, peut-on dataminer les hauteurs ou le
// joueur ne se rend jamais et virer les toits ou plafonds au-dessus de cette hauteur ».
//
// CE BINAIRE NE COUPE RIEN. Il mesure, et c'est deliberé : la coupe se decide sur des
// chiffres, pas sur une intuition. Ce qu'il rend, carte par carte :
//
//	hauteur frequentee    max, 99e centile et mediane des positions de joueur du corpus cuit
//	geometrie dessinee    distribution d'altitude des pixels de la carte cuite en production
//	ce que la coupe fait  part de l'image qui changerait, volumes supprimes, volumes decapites
//	faux positifs         zones NOMMEES du jeu situees au-dessus du plafond propose, et
//	                      stabilite du maximum quand on retire un film du corpus
//
// LICENCE — CONTRAINTE STRUCTURANTE, la meme que `cmd/mapfond-build` : la mesure de la
// geometrie passe par internal/himap -> internal/himodule -> internal/ooz, **GPLv3**. C'est de
// l'outillage HORS LIGNE ; l'application ne linke jamais cette chaine. Le drapeau
// `--sans-jeu` rend la passe corpus seule, sans aucun fichier du jeu.
//
// Usage :
//
//	CGO_ENABLED=1 go run ./cmd/mapplafond-mesure [--maps "ctf_aquarius,catalyst"]
//	    [--rejeux DIR] [--marge 5] [--rapport FILE] [--sans-jeu]
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"math"
	"os"
	"runtime/debug"
	"sort"
	"strings"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/himap"
)

func main() {
	titleSlug := flag.String("title", title.DefaultSlug, "slug du titre")
	maps := flag.String("maps", "", "modules a mesurer, separes par des virgules ; vide = tous")
	rejeux := flag.String("rejeux", "", "repertoire des documents de rejeu cuits "+
		"(defaut : PathResolver.ReplayArtifactsDir)")
	marge := flag.Float64("marge", margeParDefaut, "marge, en metres, au-dessus de la hauteur frequentee")
	rapport := flag.String("rapport", "", "ecrit le rapport Markdown a ce chemin")
	planches := flag.String("planches", "", "ecrit dans ce repertoire, par carte, la planche "+
		"« ou la coupe mord » (carte cuite, pixels au-dessus du seuil teintes)")
	sansJeu := flag.Bool("sans-jeu", false, "passe corpus seule : ne touche aucun fichier du jeu")
	flag.Parse()

	ctx := context.Background()
	code, err := execute(ctx, options{slug: *titleSlug, maps: *maps, rejeux: *rejeux,
		marge: *marge, rapport: *rapport, planches: *planches, sansJeu: *sansJeu})
	if err != nil {
		slog.ErrorContext(ctx, "mesure des plafonds", "err", err)
		os.Exit(1)
	}
	os.Exit(code)
}

// margeParDefaut : la marge, en metres, entre la plus haute position jouee et le plafond
// propose. Valeur du point 10 de l'encadre Notion — c'est une ENTREE de la mesure, pas un
// resultat : l'instrument rend aussi ce que d'autres marges donneraient.
const margeParDefaut = 5.0

type options struct {
	slug     string
	maps     string
	rejeux   string
	marge    float64
	rapport  string
	planches string
	sansJeu  bool
}

// ligne est ce que le rapport dit d'une carte.
type ligne struct {
	c     carte
	freq  *histogramme
	films []filmMesure
	geom  *mesureGeom
	// erreur : la carte n'a pas pu etre cuite. Elle reste au tableau avec son motif — une
	// carte disparue sans raison ecrite serait indiscernable d'un trou de l'instrument.
	erreur string
	// seuil est le plafond propose : hauteur maximale frequentee + marge.
	seuil float64
	coupe coupe
	zones []string
	// seuilP99 est la VARIANTE : 99e centile + marge. Elle n'est pas une proposition — elle
	// est la pour chiffrer ce que gagne (ou pas) un seuil qui ignore le dernier pour cent des
	// positions, et ce qu'il coute en zones nommees sacrifiees.
	seuilP99 float64
	coupeP99 coupe
	zonesP99 []string
	// perteMax : de combien le maximum descendrait si UN film manquait au corpus.
	perteMax float64
}

func execute(ctx context.Context, o options) (int, error) {
	racine, err := title.FindRepoRoot()
	if err != nil {
		return 1, fmt.Errorf("racine du depot : %w", err)
	}
	cartes, ecartees, err := chargeCartes(racine, o.slug)
	if err != nil {
		return 1, err
	}
	slog.InfoContext(ctx, "cartes mesurables", "n", len(cartes), "ecartees", len(ecartees))
	dir := o.rejeux
	if dir == "" {
		dir = title.NewPathResolver(racine).ReplayArtifactsDir(o.slug)
	}
	freq, err := mesureFrequentation(ctx, dir, cartes)
	if err != nil {
		return 1, err
	}
	lignes, echecs := construitLignes(ctx, o, cartes, freq)
	if o.rapport != "" {
		if err := ecritRapport(o, lignes, freq, ecartees, dir); err != nil {
			return 1, err
		}
		slog.InfoContext(ctx, "rapport ecrit", "path", o.rapport, "cartes", len(lignes))
	}
	return echecs, nil
}

// construitLignes mesure chaque carte retenue, UNE A LA FOIS, et rend le nombre d'echecs.
func construitLignes(ctx context.Context, o options, cartes []carte, freq *frequentation) ([]ligne, int) {
	filtre := filtreDe(o.maps)
	racineJeu := ""
	if !o.sansJeu {
		r, err := himap.DeployRoot()
		if err != nil {
			slog.ErrorContext(ctx, "installation du jeu introuvable — passe corpus seule", "err", err)
		}
		racineJeu = r
	}
	var out []ligne
	echecs := 0
	for i := range cartes {
		c := cartes[i]
		if filtre != nil && !filtre[c.module] {
			continue
		}
		l := ligne{c: c, freq: freq.parCarte[c.module], films: freq.filmsDe(c.module),
			seuil: math.NaN(), seuilP99: math.NaN()}
		l.seuil, l.perteMax = seuilEtStabilite(freq, c.module, l.freq, o.marge)
		if l.freq != nil && l.freq.taille() > 0 {
			l.seuilP99 = l.freq.centile(0.99) + o.marge
		}
		if racineJeu != "" {
			g, rendu, err := mesureGeometrie(ctx, racineJeu, c)
			if err != nil {
				slog.ErrorContext(ctx, "mesure de carte", "err", err, "module", c.module)
				l.erreur = err.Error()
				echecs++
			} else {
				l.geom = &g
				poseplanche(ctx, o, &l, rendu)
			}
			libereCarte()
		}
		if l.geom != nil && !math.IsNaN(l.seuil) {
			l.coupe, l.zones = l.geom.evalueCoupe(l.seuil), c.zonesAuDessus(l.seuil)
			l.coupeP99, l.zonesP99 = l.geom.evalueCoupe(l.seuilP99), c.zonesAuDessus(l.seuilP99)
		}
		out = append(out, l)
	}
	trieLignes(out)
	return out, echecs
}

// libereCarte rend au systeme la memoire de la carte qui vient d'etre mesuree.
//
// MESURE QUI L'IMPOSE (2026-08-25) : sans elle, le balayage des 19 cartes atteint **15,5 Go
// de resident** a la septieme — alors que la meme carte, mesuree SEULE dans son processus,
// tient en quelques centaines de mega-octets et s'acheve en secondes. Ce n'est pas une fuite :
// c'est un tas qui double a chaque cycle et que le ramasse-miettes ne rend jamais, sur des
// cartes qui allouent chacune plusieurs millions de pixels et des centaines de maillages
// decompresses. Le balayage entier dans un processus est la « bombe RAM » du registre, et
// c'est ici qu'elle se desamorce — une carte a la fois, memoire rendue entre chaque.
func libereCarte() {
	debug.FreeOSMemory()
}

// poseplanche ecrit, si l'appelant l'a demandee, la planche « ou la coupe mord » d'une carte.
// Une carte sans seuil (pas de corpus) n'en a pas : il n'y a rien a teinter.
func poseplanche(ctx context.Context, o options, l *ligne, rendu *himap.Rendu) {
	if o.planches == "" || rendu == nil || math.IsNaN(l.seuil) {
		return
	}
	if err := ecritPlanche(o.planches, l.c.module, rendu, l.seuil); err != nil {
		// Journalise, jamais avale : une planche manquante ne doit pas passer pour une carte
		// que la coupe n'entame pas.
		slog.ErrorContext(ctx, "planche de coupe", "err", err, "module", l.c.module)
		return
	}
	slog.InfoContext(ctx, "planche de coupe ecrite", "module", l.c.module,
		"seuil", fmt.Sprintf("%.1f", l.seuil), "dir", o.planches)
}

// seuilEtStabilite rend le plafond propose et la PERTE que subirait ce plafond si un seul film
// manquait au corpus — le controle qui dit si le maximum est une borne de la CARTE ou seulement
// de ce que le corpus a visite.
//
// SOUS DEUX FILMS, LA PERTE EST NaN, PAS ZERO. Retirer l'unique film d'une carte ne laisse rien
// a comparer : le controle n'est pas « stable », il est IMPOSSIBLE. Publier 0,0 la ou rien n'a
// ete mesure serait la pire des sorties — une carte a un seul film passerait pour la mieux
// bornee du tableau.
func seuilEtStabilite(freq *frequentation, module string, h *histogramme, marge float64) (float64, float64) {
	if h == nil || h.taille() == 0 {
		return math.NaN(), math.NaN()
	}
	films := freq.filmsDe(module)
	hMax := h.maximum()
	if len(films) < 2 {
		return hMax + marge, math.NaN()
	}
	pire := 0.0
	for _, f := range films {
		sans := freq.hauteurMaxSansFilm(module, f.id)
		if math.IsNaN(sans) {
			continue
		}
		if d := hMax - sans; d > pire {
			pire = d
		}
	}
	return hMax + marge, pire
}

// filtreDe rend l'ensemble des modules demandes, ou nil pour « tous ».
func filtreDe(list string) map[string]bool {
	if strings.TrimSpace(list) == "" {
		return nil
	}
	out := map[string]bool{}
	for _, raw := range strings.Split(list, ",") {
		if k := strings.TrimSpace(raw); k != "" {
			out[k] = true
		}
	}
	return out
}

// trieLignes ordonne le tableau par module — l'ordre d'une map Go n'est pas un ordre.
func trieLignes(l []ligne) {
	sort.Slice(l, func(i, j int) bool { return l[i].c.module < l[j].c.module })
}
