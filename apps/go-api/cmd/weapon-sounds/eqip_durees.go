package main

// eqip_durees.go — mode `eqip-durees` : TRIAGE des candidats "pose de balise" du balayage
// structurel (`eqip-arbre -banks all`) par duree.
//
// Lot L6, S3 (`.ai/V7.5/replay2d/PLAN_BALISE_MIX_WWISE.md` phase 5.2). Deux filtres, dans
// l'ordre de leur cout :
//
//  1. STRUCTUREL, gratuit (deja dans le JSON du balayage) : ecarte les banques que le graphe
//     `eqip -> effe -> snd! -> sbnk` atteint deja (elles ont ete ECOUTEES, cf. phase 4 du plan
//     sœur — ce sont des vehicules) et les evenements dont au moins une couche est de famille
//     MUSICALE (MusicTrack/MusicSegment/MusicSwitch/MusicRanSeq).
//  2. DUREE, cout d'un rechargement du module : pour les survivants, lit l'en-tete RIFF de
//     leurs `.wem` EMBARQUES (`wemduree.go` — pas de decodage audio) et retient les
//     evenements dont TOUTES les variantes tombent dans 0,3-2 s.
//
// CE QUE CE MODE N'AFFIRME PAS : l'absence de boucle. Le parseur ne decode aucun drapeau de
// bouclage Wwise (hors de portee sans reference pour le valider, meme discipline que le
// "trou de preuve" du delai d'action documente dans `RECETTE_SONS_ARMES.md` §4) — la colonne
// "nature" et la duree sont les seuls elements publies, jamais un "non-boucle" invente.

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/himodule"
)

// candidatDuree : un evenement survivant des deux filtres, avec ses durees et sa recette.
type candidatDuree struct {
	Bank        string             `json:"bank"`
	Event       string             `json:"event"`
	Nature      string             `json:"nature"`
	NbCouches   int                `json:"nb_couches"`
	Wems        []uint32           `json:"wems"`
	DureesS     map[string]float64 `json:"durees_s"`
	DureeMinS   float64            `json:"duree_min_s"`
	DureeMaxS   float64            `json:"duree_max_s"`
	WemsResolus int                `json:"wems_resolus"`
	GainsDB     []float32          `json:"gains_db,omitempty"`
}

// triageDurees est le mode `eqip-durees`.
//
// `entree` : sortie de `eqip-arbre -banks all` (schema `RapportStructure`). `exclues` :
// banques a ecarter (le graphe eqip deja ecoute, `-exclure`). `sortie` : table complete
// (JSON), triee par duree minimale croissante.
func triageDurees(cheminModule, entree string, exclues map[uint32]bool, sortie string) error {
	if entree == "" {
		return fmt.Errorf("le mode eqip-durees exige -json (sortie de eqip-arbre -banks all)")
	}
	blob, err := os.ReadFile(entree)
	if err != nil {
		return err
	}
	var rap RapportStructure
	if err := json.Unmarshal(blob, &rap); err != nil {
		return err
	}

	type survivant struct {
		bankHex string
		ev      evenementStructure
	}
	var survivants []survivant
	bankExclues, evMusique := 0, 0
	besoin := map[uint32]bool{}
	for _, b := range rap.Banks {
		gid, errHex := strconv.ParseUint(b.Bank, 16, 32)
		if errHex == nil && exclues[uint32(gid)] {
			bankExclues++
			continue
		}
		for _, ev := range b.Evenements {
			if evenementEstMusical(ev) {
				evMusique++
				continue
			}
			survivants = append(survivants, survivant{bankHex: b.Bank, ev: ev})
			if errHex == nil {
				besoin[uint32(gid)] = true
			}
		}
	}
	fmt.Printf("filtre structurel : %d evenements survivants (banques exclues : %d, evenements musicaux ecartes : %d, banques a rouvrir : %d)\n",
		len(survivants), bankExclues, evMusique, len(besoin))

	m, err := himodule.Open(cheminModule)
	if err != nil {
		return err
	}
	rapporterMemoire("module charge")

	dureesParBanque := map[string]map[uint32]float64{}
	for gid := range besoin {
		hex := fmt.Sprintf("%08x", gid)
		_, brut, errB := bankParIdentifiant(m, gid)
		if errB != nil {
			continue
		}
		ch := chunks(brut)
		dureesParBanque[hex] = dureesEmbarquees(mediasEmbarques(ch), ch["DATA"])
	}

	var out []candidatDuree
	resolus, horsBucket := 0, 0
	for _, s := range survivants {
		c := candidatDuree{
			Bank: s.bankHex, Event: s.ev.Event, Nature: s.ev.Nature,
			NbCouches: len(s.ev.Couches), Wems: s.ev.Wems, DureesS: map[string]float64{},
		}
		durees := dureesParBanque[s.bankHex]
		premier := true
		for _, w := range s.ev.Wems {
			d, ok := durees[w]
			if !ok {
				continue
			}
			c.DureesS[fmt.Sprintf("%d", w)] = arrondi3(d)
			c.WemsResolus++
			if premier || d < c.DureeMinS {
				c.DureeMinS = d
			}
			if premier || d > c.DureeMaxS {
				c.DureeMaxS = d
			}
			premier = false
		}
		for _, co := range s.ev.Couches {
			for _, g := range co.Gains {
				c.GainsDB = append(c.GainsDB, g)
			}
		}
		sort.Slice(c.GainsDB, func(i, j int) bool { return c.GainsDB[i] > c.GainsDB[j] })
		if c.WemsResolus == 0 {
			continue // duree inconnue (aucun wem embarque lisible) : ni retenu ni rejete
		}
		resolus++
		c.DureeMinS, c.DureeMaxS = arrondi3(c.DureeMinS), arrondi3(c.DureeMaxS)
		if c.DureeMinS < 0.3 || c.DureeMaxS > 2.0 {
			horsBucket++
			continue
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].DureeMinS != out[j].DureeMinS {
			return out[i].DureeMinS < out[j].DureeMinS
		}
		return out[i].Bank < out[j].Bank
	})

	fmt.Printf("duree lue sur au moins un wem : %d evenements (%d sans aucun wem embarque lisible) ; dans 0,3-2s : %d ; hors bucket : %d\n",
		resolus, len(survivants)-resolus, len(out), horsBucket)
	afficherTop(out, 40)

	if sortie == "" {
		return nil
	}
	j, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	fmt.Printf("\ntable complete ecrite : %s (%d candidats)\n", sortie, len(out))
	return os.WriteFile(sortie, j, 0o644)
}

// evenementEstMusical dit si au moins une couche de l'evenement est une famille MUSIQUE —
// hierarchie distincte, sans objet pour une pose d'equipement (cf. audit du chantier armes,
// `PLAN_EXTRACTION_SONS_ARMES.md` etape 12, verdict `MusicTrack`/`MusicSegment`/
// `MusicRanSeq`/`MusicSwitch` : "IGNORE AVEC RAISON").
func evenementEstMusical(ev evenementStructure) bool {
	for _, c := range ev.Couches {
		if strings.HasPrefix(c.TypeNoeud, "Music") {
			return true
		}
	}
	return false
}

// arrondi3 evite d'afficher des durees a 15 decimales de bruit float64.
func arrondi3(f float64) float64 { return math.Round(f*1000) / 1000 }

// afficherTop imprime les N candidats de plus courte duree — c'est la source du tableau du
// compte rendu, dans le meme ordre que la table complete ecrite en JSON.
func afficherTop(out []candidatDuree, n int) {
	if n > len(out) {
		n = len(out)
	}
	fmt.Printf("\n--- top %d candidats (duree min croissante, 0,3-2s, hors banques deja ecoutees) ---\n", n)
	for i := 0; i < n; i++ {
		c := out[i]
		fmt.Printf("%2d. sbnk %s  event %s  %.2f-%.2fs  %d couche(s)  %d/%d wem resolus  gains=%v  [%s]\n",
			i+1, c.Bank, c.Event, c.DureeMinS, c.DureeMaxS, c.NbCouches, c.WemsResolus, len(c.Wems), c.GainsDB, c.Nature)
	}
}
