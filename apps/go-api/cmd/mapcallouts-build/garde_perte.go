package main

// garde_perte.go — LE CATALOGUE NE MAIGRIT PAS PAR ACCIDENT.
//
// LE DEFAUT, PAYE DEUX FOIS. La passe native relit la geometrie de l installation du jeu, et son
// resultat DEPEND de l installation : le 2026-09-03, rejouee sur ce poste pour la seule raison
// qu une carte venait de recevoir un fond, elle a rendu un catalogue de 2,97 Mo la ou HEAD en
// portait 3,71 — **32 716 nombres de moins, soit environ 16 400 sommets**, repartis sur 16 cartes
// (btb_fragmentation -9 372, btb_highpower -4 786, btb_exiled -4 299, va_behemoth -4 120...). Le
// meme piege est consigne dans `.ai/V7.5/cartes/HANDOFF_FONDS_CARTE_2026-09-03.md` (section 5)
// pour une autre machine, avec un ecart de 645 Ko.
//
// CE QUI L A RENDU INVISIBLE, ET C EST LE POINT. Les invariants deja poses par la passe comptent
// les CARTES (22), les ZONES (816) et les LIBELLES (816/816). Aucun des trois n a bouge : la
// perte est sous la zone, dans ses SOMMETS. Un compte de zones ne voit pas un polygone qui passe
// de 400 sommets a 40 — la carte est toujours la, la zone aussi, et le contour est devenu faux.
// C est exactement la lecon deja ecrite au journal le 2026-08-26 : « un garde-rail ne protege que
// le champ qu il compte ».
//
// LA REGLE : on compare la sortie a ce qui est DEJA sur le disque, carte par carte, et on refuse
// d ecrire des qu une carte perd des sommets. Le seuil est ZERO — une passe qui produit la meme
// donnee produit le meme compte, et toute baisse est soit une regression d entree, soit un
// changement qui merite d etre nomme.
//
// L ECHAPPATOIRE EST EXPLICITE ET BRUYANTE : `--accepte-perte` la leve, apres avoir journalise
// chaque carte perdante avec ses deux comptes. Elle existe pour le cas legitime — un decoupage
// qui simplifie volontairement des contours — et elle oblige a le declarer.

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/replay"
)

// perteCarte est le constat pour UNE carte : combien de sommets avant, combien apres.
type perteCarte struct {
	Cle   string
	Avant int
	Apres int
}

// sommetsZone totalise les sommets publies d une zone : son contour, ses parties et ses trous.
// Les trois comptent — un decoupage range une partie de la forme dans `Parts`, et n additionner
// que `Polygon` ferait passer un decoupage reussi pour une perte.
func sommetsZone(z replay.CalloutZone) int {
	n := len(z.Polygon)
	for _, p := range z.Parts {
		n += len(p)
	}
	for _, h := range z.Holes {
		n += len(h)
	}
	return n
}

// sommetsParCarte rend le nombre de sommets de chaque carte du catalogue, sections native et
// Forge confondues. Les deux espaces de cles ne se croisent jamais (module installe contre
// map_id), les fusionner dans une seule table est donc sans ambiguite.
func sommetsParCarte(cat replay.MapCalloutsCatalog) map[string]int {
	out := map[string]int{}
	ajoute := func(cle string, e replay.MapCalloutsEntry) {
		n := 0
		for _, z := range e.Zones {
			n += sommetsZone(z)
		}
		out[cle] = n
	}
	for cle, e := range cat.Maps {
		ajoute(cle, e)
	}
	for cle, e := range cat.MapsByID {
		ajoute(cle, e)
	}
	return out
}

// pertes rend les cartes qui perdent des sommets, triees de la perte la plus lourde a la plus
// legere. Une carte ABSENTE de la sortie compte comme une perte totale : la faire disparaitre est
// pire que l amaigrir.
func pertes(avant, apres replay.MapCalloutsCatalog) []perteCarte {
	sa, sb := sommetsParCarte(avant), sommetsParCarte(apres)
	var out []perteCarte
	for cle, n := range sa {
		if m := sb[cle]; m < n {
			out = append(out, perteCarte{Cle: cle, Avant: n, Apres: m})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		di, dj := out[i].Avant-out[i].Apres, out[j].Avant-out[j].Apres
		if di != dj {
			return di > dj
		}
		return out[i].Cle < out[j].Cle
	})
	return out
}

// verifiePasDePerte compare le catalogue a ecrire a celui deja sur disque et rend une erreur si
// une carte y perd des sommets.
//
// ABSENCE DE FICHIER = PAS DE COMPARAISON POSSIBLE, et ce n est pas une faute : c est le premier
// run d un titre. Un fichier ILLISIBLE, en revanche, est signale — on ne veut pas qu un catalogue
// corrompu desarme le garde en silence.
func verifiePasDePerte(outPath string, cat replay.MapCalloutsCatalog, accepte bool) error {
	ancien, err := replay.LoadMapCallouts(outPath)
	if errors.Is(err, os.ErrNotExist) {
		slog.Info("garde de perte : aucun catalogue existant — premier ecrit", "path", outPath)
		return nil
	}
	if err != nil {
		slog.Warn("garde de perte : catalogue existant illisible — comparaison impossible",
			"err", err, "path", outPath)
		return nil
	}
	perdantes := pertes(*ancien, cat)
	if len(perdantes) == 0 {
		return nil
	}
	total := 0
	for _, p := range perdantes {
		total += p.Avant - p.Apres
		slog.Warn("garde de perte : carte amaigrie", "carte", p.Cle,
			"sommets_avant", p.Avant, "sommets_apres", p.Apres, "perte", p.Avant-p.Apres)
	}
	if accepte {
		slog.Warn("garde de perte : perte ACCEPTEE sur demande explicite",
			"cartes", len(perdantes), "sommets_perdus", total, "flag", "--accepte-perte")
		return nil
	}
	return fmt.Errorf("garde de perte : %d carte(s) perdent %d sommets — rien ecrit. "+
		"La passe native depend de l installation du jeu : une relecture depuis une autre "+
		"racine, ou une installation partielle, rend moins de geometrie a comptes de cartes et "+
		"de zones INCHANGES. Restaurer le catalogue depuis HEAD et ne rejouer que la carte "+
		"produite, ou relancer avec --accepte-perte si la simplification est voulue",
		len(perdantes), total)
}
