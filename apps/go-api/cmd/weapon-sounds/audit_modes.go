package main

// audit_modes.go — le mode `audit-modes` : COMBIEN de conteneurs de type 5 sont des
// SEQUENCES, et lesquels.
//
// ORDRE IMPOSE DE LA SORTIE, comme pour `banks-noms` : d'abord la PLAUSIBILITE de la lecture
// (quelle part des conteneurs rend un `eMode` dans {0, 1} et un drapeau dans ses cinq bits),
// ensuite seulement les resultats. Une lecture au mauvais offset produirait du bruit sur les
// deux, et le taux le dirait avant qu'on ne conclue quoi que ce soit.
//
// LE TEMOIN, ecrit avant la mesure : les banques d'ARMES doivent etre massivement ALEATOIRES.
// Leurs sons ont ete reconstruits « une variante par couche » et VALIDES A L'OREILLE par
// l'utilisateur en aout ; si la lecture disait « sequence » sur les armes, c'est la lecture
// qui serait fausse, pas huit mois d'ecoute.

import (
	"fmt"
	"sort"

	"levelup/go-api/internal/himodule"
)

// auditModesConteneurs balaie les banques d'un module et statue sur leur mode de lecture.
// `cibles` vide = toutes les banques du module.
func auditModesConteneurs(cheminModule string, cibles map[uint32]bool) error {
	m, err := himodule.Open(cheminModule)
	if err != nil {
		return err
	}
	rapporterMemoire("module charge")

	var total, plausibles, sequences, sequencesContinues int
	type trouve struct {
		bank      string
		conteneur string
		enfants   int
		continu   bool
	}
	var liste []trouve

	for _, f := range m.Files("sbnk") {
		if len(cibles) > 0 && !cibles[f.GlobalID] {
			continue
		}
		data, err := m.Extract(f)
		if err != nil {
			continue
		}
		debut := indexBKHD(data)
		if debut < 0 {
			continue
		}
		b, err := parserBank(data[debut:], func(uint32) bool { return false })
		if err != nil {
			continue
		}
		connu := func(id uint32) bool { _, ok := b.Objets[id]; return ok }
		for id, o := range b.Objets {
			if o.Type != typeRandomSeq {
				continue
			}
			total++
			md := lireModeRanSeq(o.Data, connu)
			if !md.Lu {
				continue
			}
			plausibles++
			if !md.Sequence {
				continue
			}
			sequences++
			if md.Continu {
				sequencesContinues++
			}
			if len(liste) < 60 {
				liste = append(liste, trouve{
					bank:      fmt.Sprintf("%08x", f.GlobalID),
					conteneur: fmt.Sprintf("%08x", id),
					enfants:   md.Enfants,
					continu:   md.Continu,
				})
			}
		}
	}

	fmt.Println()
	fmt.Println("=== 1. PLAUSIBILITE DE LA LECTURE (avant tout resultat) ===")
	fmt.Printf("  conteneurs de type 5 balayes        : %d\n", total)
	fmt.Printf("  dont eMode dans {0,1} et bits connus : %d (%.1f %%)\n",
		plausibles, 100*float64(plausibles)/float64(max(total, 1)))
	if total > 0 && float64(plausibles)/float64(total) < 0.90 {
		fmt.Println("  ATTENTION : taux trop bas — l'offset est probablement FAUX, ne rien conclure.")
	}

	fmt.Println()
	fmt.Println("=== 2. MODE DE LECTURE ===")
	fmt.Printf("  ALEATOIRE (eMode = 0) : %d (%.2f %%)\n",
		plausibles-sequences, 100*float64(plausibles-sequences)/float64(max(plausibles, 1)))
	fmt.Printf("  SEQUENCE  (eMode = 1) : %d (%.2f %%), dont CONTINUES : %d\n",
		sequences, 100*float64(sequences)/float64(max(plausibles, 1)), sequencesContinues)

	fmt.Println()
	fmt.Println("=== 3. LES SEQUENCES, UNE PAR UNE (60 au plus) ===")
	sort.Slice(liste, func(i, j int) bool {
		if liste[i].bank != liste[j].bank {
			return liste[i].bank < liste[j].bank
		}
		return liste[i].conteneur < liste[j].conteneur
	})
	for _, t := range liste {
		suite := "pas a pas"
		if t.continu {
			suite = "CONTINUE (phases enchainees)"
		}
		fmt.Printf("  sbnk %s  conteneur %s  %d enfants  %s\n", t.bank, t.conteneur, t.enfants, suite)
	}
	if len(liste) == 0 {
		fmt.Println("  (aucune)")
	}
	return nil
}
