package main

// comparer.go — DEUX FILMS, COTE A COTE.
//
// A QUOI CA SERT. Le decodeur se calibre PAR FILM (la precision de position est une propriete de
// la CARTE) et se comporte differemment selon le mode. Comparer deux films est donc le moyen le
// plus court de voir si une bascule a change quelque chose : si un chiffre bouge sur un film et
// pas sur l autre, ce n est pas le decodeur qui a bouge, c est le film qui est different.
//
// CONTRAINTE D EXECUTION RESPECTEE ICI : les deux films sont decodes L UN APRES L AUTRE, jamais en
// parallele. Les parametres de replication du lecteur de bits sont des globaux de paquet ; deux
// decodages simultanes se contamineraient. Le paquet serialise deja les passes par un verrou, mais
// cette commande n essaie meme pas — un parallelisme qui serait de toute facon serialise n achete
// rien et se lirait comme une intention.

import (
	"errors"
	"fmt"
	"os"
	"text/tabwriter"

	"levelup/go-api/internal/games/halo_infinite/film/killsource"
)

func comparer(args []string, o options) error {
	if len(args) != 2 {
		return errors.New("comparer prend exactement deux films")
	}
	a, err := decoder(args[0], o.cache)
	if err != nil {
		return err
	}
	b, err := decoder(args[1], o.cache)
	if err != nil {
		return err
	}
	fmt.Printf("COMPARAISON  %s  contre  %s\n\n", a.film, b.film)
	// La calibration sort du tableau : elle est longue, et une cellule longue elargit toute la
	// colonne. Elle DOIT differer d un film a l autre — c est une propriete de la CARTE.
	fmt.Printf("calibration %s : %s\n", a.film, a.result.Calibration)
	fmt.Printf("calibration %s : %s\n", b.film, b.result.Calibration)
	fmt.Println("(deux calibrations differentes ne sont PAS une anomalie : le parametre de precision")
	fmt.Println(" de position est installe par la CARTE, il se resout automatiquement par film.)")
	fmt.Println()
	tableComparaison(a, b)
	sourcesComparees(a, b)
	fmt.Println("\nCE QUI DOIT ETRE IDENTIQUE D UN FILM A L AUTRE : le DESACCORD entre voies (0), les")
	fmt.Println("morts a plusieurs candidats (0), et la coherence divergence/origine. Tout le reste")
	fmt.Println("depend legitimement de la carte et du mode — a commencer par la calibration.")
	return nil
}

// tableComparaison : les quantites porteuses, une ligne chacune.
func tableComparaison(a, b *rapport) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "\t%s\t%s\t\n", a.film, b.film)
	ligneCmp(w, "morts publiees", len(a.result.Kills), len(b.result.Kills))
	ligneCmp(w, "couples REELS (denominateur de reference)", a.result.Coverage.RealPairs, b.result.Coverage.RealPairs)
	ligneCmp(w, "couples reconstruits", a.result.Coverage.ReconstructedPairs, b.result.Coverage.ReconstructedPairs)
	ligneCmp(w, "morts du kill-feed", a.result.Coverage.FeedDeaths, b.result.Coverage.FeedDeaths)
	ligneCmp(w, "couples FABRIQUES retires", a.result.Coverage.GhostPairs, b.result.Coverage.GhostPairs)
	ligneCmp(w, "morts de BOT (population neuve)", a.result.Coverage.BotDeaths, b.result.Coverage.BotDeaths)
	fmt.Fprintln(w, "\t\t\t")
	fmt.Fprintf(w, "couverture des couples REELS\t%s\t%s\t\n",
		pct(a.result.Coverage.Covered, a.result.Coverage.RealPairs),
		pct(b.result.Coverage.Covered, b.result.Coverage.RealPairs))
	fmt.Fprintf(w, "appariement de la voie sequentielle\t%s\t%s\t\n",
		pct(a.result.Stats.Walk.Matched, a.result.Stats.Walk.Population),
		pct(b.result.Stats.Walk.Matched, b.result.Stats.Walk.Population))
	fmt.Fprintf(w, "appariement du balayage (rattrapage)\t%s\t%s\t\n",
		pct(a.result.Stats.Scan.Matched, a.result.Stats.Scan.Population),
		pct(b.result.Stats.Scan.Matched, b.result.Stats.Scan.Population))
	fmt.Fprintln(w, "\t\t\t")
	ligneCmp(w, "DESACCORD entre voies (doit valoir 0)", a.result.Stats.Disagree, b.result.Stats.Disagree)
	ligneCmp(w, "morts a plusieurs candidats (doit valoir 0)", a.result.Stats.MultiCandidate, b.result.Stats.MultiCandidate)
	ligneCmp(w, "divergences credit / source", divergences(a), divergences(b))
	ligneCmp(w, "sources sans nom propre (<< Autres >>)", sansNom(a), sansNom(b))
	fmt.Fprintln(w, "\t\t\t")
	fmt.Fprintf(w, "verdict de sante\t%s\t%s\t\n", a.result.Health.Verdict(), b.result.Health.Verdict())
	fmt.Fprintf(w, "taux d inexpliques\t%.1f %%\t%.1f %%\t\n",
		100*a.result.Health.UnexplainedRatio(), 100*b.result.Health.UnexplainedRatio())
	ligneCmp(w, "marge de bijection (0 = lignes non publiables)", a.result.BijectionMargin, b.result.BijectionMargin)
	fmt.Fprintf(w, "publication ligne par ligne\t%s\t%s\t\n",
		ouiNon(a.result.LineByLinePublishable()), ouiNon(b.result.LineByLinePublishable()))
	_ = w.Flush()
}

func ligneCmp(w *tabwriter.Writer, nom string, a, b int) {
	fmt.Fprintf(w, "%s\t%d\t%d\t\n", nom, a, b)
}

func ouiNon(b bool) string {
	if b {
		return "autorisee"
	}
	return "REFUSEE (agregat seulement)"
}

func divergences(r *rapport) int {
	return compte(r.result.Kills, func(k killsource.Kill) bool { return k.Diverges })
}

func sansNom(r *rapport) int {
	return compte(r.result.Kills, func(k killsource.Kill) bool { return !k.Source.Named })
}

// sourcesComparees : les sources les plus frequentes de chaque film. C est le controle le plus
// parlant pour un humain : un arsenal doit ressembler a son mode de jeu.
//
// A NE PAS SURINTERPRETER : la dispersion de l arsenal ne caracterise PAS le mode. Une carte Forge
// est aussi dispersee qu un Fiesta — la piste << signature d arsenal par mode >> a ete mesuree et
// refutee. Ce bloc sert a reconnaitre, pas a classer.
func sourcesComparees(a, b *rapport) {
	fmt.Println("\nLES CINQ SOURCES LES PLUS FREQUENTES DE CHAQUE FILM")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fa, fb := top5(a), top5(b)
	fmt.Fprintf(w, "  %s\t\t%s\t\t\n", a.film, b.film)
	for i := 0; i < 5; i++ {
		fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t\n", nomDe(fa, i), cntDe(fa, i), nomDe(fb, i), cntDe(fb, i))
	}
	_ = w.Flush()
	fmt.Println("  La dispersion ne caracterise PAS le mode : une carte Forge est aussi dispersee")
	fmt.Println("  qu un Fiesta. Ce bloc sert a RECONNAITRE un film, pas a le classer.")
}

// entree : une source et son compte.
type entree struct {
	nom string
	n   int
}

func top5(r *rapport) []entree {
	par := map[string]int{}
	for _, k := range r.result.Kills {
		par[k.Source.Display]++
	}
	out := make([]entree, 0, len(par))
	for nom, n := range par {
		out = append(out, entree{nom, n})
	}
	// Tri decroissant par compte, puis par nom : deux passes doivent rendre le meme ordre.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && plusHaut(out[j], out[j-1]); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	if len(out) > 5 {
		out = out[:5]
	}
	return out
}

func plusHaut(a, b entree) bool {
	if a.n != b.n {
		return a.n > b.n
	}
	return a.nom < b.nom
}

func nomDe(es []entree, i int) string {
	if i >= len(es) {
		return ""
	}
	return es[i].nom
}

func cntDe(es []entree, i int) string {
	if i >= len(es) {
		return ""
	}
	return fmt.Sprintf("%d", es[i].n)
}
