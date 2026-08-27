package main

// audit_boucles.go — le mode `audit-boucles` : UN CONTENEUR SE REPETE-T-IL, ET COMBIEN DE FOIS ?
//
// POURQUOI. Un rendu qui joue un fichier UNE fois ne peut pas restituer une boucle : il en
// donne un fragment. C'est l'hypothese H1 du handoff du 2026-08-27 — « une capture en cours »
// et « une montee en charge » sont des boucles par nature, et le rendu actuel ne boucle pas.
// Le jeu porte deja des noms d'evenement qui le disent : `assault_bomb_planted_loop`,
// `assault_bomb_disarm_loop`.
//
// OU VIT LA REPONSE. Dans `AkRanSeqCntrInitialValues`, le nombre de repetitions precede les
// champs deja localises par `conteneurs_mode.go`, qui s'ancrent tous sur la liste d'enfants :
//
//	u16 sLoopCount | u16 sLoopModMin | u16 sLoopModMax          <- LAYOUT A (recent)
//	f32 fTransitionTime | f32 ...ModMin | f32 ...ModMax
//	u16 wAvoidRepeatCount | u8 eTransitionMode | u8 eRandomMode | u8 eMode | u8 byBitVector
//	u32 ulNumChilds   <- `off`, deja valide
//
// LAYOUT B (variantes anciennes) : pas de `sLoopMod*`, `sLoopCount` colle aux transitions.
// Les deux ne different QUE par la position du compteur ; les trois flottants de transition
// sont au meme endroit dans les deux cas et ne les discriminent donc pas.
//
// LA SORTIE MONTRE LES OCTETS (lecon 3 de `RECETTE_SONS_ARMES` : quand un decodage est
// incertain, montrer les octets plutot qu'essayer un offset de plus), puis les deux lectures
// cote a cote, puis les taux de plausibilite de chacune. Le lecteur tranche sur pieces.
//
// CONVENTION WWISE : `sLoopCount` = 0 signifie BOUCLE INFINIE, 1 = joue une fois.

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"

	"levelup/go-api/internal/himodule"
)

// Decalages depuis la liste d'enfants, pour les deux layouts candidats.
const (
	decalageBoucleA = 24 // u16 sLoopCount | u16 ModMin | u16 ModMax | 3 x f32 | 6 octets
	decalageBoucleB = 20 // u16 sLoopCount | 3 x f32 | 6 octets
	decalageTransit = 18 // les trois flottants de transition, communs aux deux layouts
)

// Modes d'enchainement d'un conteneur (`AkTransitionMode`). LE MODE CHANGE LE RENDU AUTANT
// QUE LE COMPTEUR : trois lectures bout a bout et trois lectures declenchees toutes les
// 850 ms ne durent pas la meme chose et ne sonnent pas pareil.
const (
	transitionAucune       = 0 // les lectures s'enchainent bout a bout
	transitionFonduAmp     = 1 // fondu enchaine (amplitude) sur la duree de transition
	transitionFonduPuiss   = 2 // fondu enchaine (puissance)
	transitionDelai        = 3 // silence de la duree de transition entre deux lectures
	transitionEchantillon  = 4 // bout a bout, a l'echantillon pres
	transitionCadence      = 5 // CADENCE : une lecture demarre toutes les N ms, elles se
	transitionCadenceLabel = "cadence de declenchement"
)

// boucleLue : ce qu'un conteneur de type 5 dit de ses repetitions ET de leur enchainement.
type boucleLue struct {
	Repetitions int     // 0 = infini
	Mode        int     // AkTransitionMode
	TransitionS float32 // duree de transition, en secondes
	Lu          bool    // le layout retenu a rendu une lecture plausible
}

// Chevauche dit si les lectures successives se recouvrent au lieu de se suivre.
func (b boucleLue) Chevauche() bool {
	return b.Mode == transitionCadence || b.Mode == transitionFonduAmp || b.Mode == transitionFonduPuiss
}

// lireBoucleRanSeq rend le nombre de repetitions d'un conteneur de type 5.
//
// Le layout A est retenu s'il est plausible, le B en repli. Plausibilite : les trois
// flottants de transition doivent etre finis et dans [0 ; 1000] secondes, le compteur dans
// [0 ; 1000], et les deux modulations nulles ou petites.
func lireBoucleRanSeq(d []byte, connu func(uint32) bool) boucleLue {
	off, n := positionEnfants(d, connu)
	if off < decalageBoucleA || n < 1 || !transitionsPlausibles(d, off) {
		return boucleLue{}
	}
	mode := int(d[off-4])
	transit := math.Float32frombits(binary.LittleEndian.Uint32(d[off-decalageTransit:])) / 1000
	if mode > transitionCadence {
		mode = transitionAucune
	}
	if c, mn, mx := u16A(d, off-decalageBoucleA), u16A(d, off-22), u16A(d, off-20); c <= 1000 && mn <= 1000 && mx <= 1000 {
		return boucleLue{Repetitions: int(c), Mode: mode, TransitionS: transit, Lu: true}
	}
	if c := u16A(d, off-decalageBoucleB); c <= 1000 {
		return boucleLue{Repetitions: int(c), Mode: mode, TransitionS: transit, Lu: true}
	}
	return boucleLue{}
}

// transitionsPlausibles controle les trois flottants communs aux deux layouts.
func transitionsPlausibles(d []byte, off int) bool {
	if off < decalageTransit {
		return false
	}
	for _, dec := range []int{18, 14, 10} {
		v := math.Float32frombits(binary.LittleEndian.Uint32(d[off-dec:]))
		f := float64(v)
		if math.IsNaN(f) || math.IsInf(f, 0) || f < 0 || f > 1000 {
			return false
		}
	}
	return true
}

func u16A(d []byte, off int) uint16 {
	if off < 0 || off+2 > len(d) {
		return 0xffff
	}
	return binary.LittleEndian.Uint16(d[off:])
}

// auditBoucles balaie les banques d'un module et statue sur les repetitions des conteneurs.
func auditBoucles(cheminModule string, cibles map[uint32]bool) error {
	m, err := himodule.Open(cheminModule)
	if err != nil {
		return err
	}
	rapporterMemoire("module charge")

	var total, transitOK, plausA, plausB int
	repartition := map[int]int{}
	// TEMOIN DE COHERENCE INTERNE, ecrit avant la mesure. Les deux layouts sont tous deux
	// « plausibles » sur les bornes ; ce qui les separe est le SENS. Une boucle infinie et
	// le drapeau `bIsContinuous` decrivent la meme chose (enchainer sans s'arreter) : si le
	// compteur lu a l'offset A est le bon, les deux doivent aller ensemble bien plus souvent
	// que le hasard. Croisement imprime ci-dessous ; un tableau sans structure refute A.
	croix := map[[2]bool]int{}
	// Repartition des modes d enchainement, sur les seuls conteneurs qui se repetent.
	modes := map[int]int{}
	type echantillon struct{ bank, cont, octets, lectures string }
	var echs []echantillon

	for _, f := range m.Files("sbnk") {
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
			off, n := positionEnfants(o.Data, connu)
			if off < decalageBoucleA || n < 1 {
				continue
			}
			if !transitionsPlausibles(o.Data, off) {
				continue
			}
			transitOK++
			a, ma, xa := u16A(o.Data, off-24), u16A(o.Data, off-22), u16A(o.Data, off-20)
			bb := u16A(o.Data, off-decalageBoucleB)
			if a <= 1000 && ma <= 1000 && xa <= 1000 {
				plausA++
			}
			if bb <= 1000 {
				plausB++
			}
			if bl := lireBoucleRanSeq(o.Data, connu); bl.Lu {
				repartition[bl.Repetitions]++
				md := lireModeRanSeq(o.Data, connu)
				croix[[2]bool{bl.Repetitions == 0, md.Lu && md.Continu}]++
				if bl.Repetitions != 1 {
					modes[bl.Mode]++
				}
			}
			if cibles[f.GlobalID] && len(echs) < 24 {
				echs = append(echs, echantillon{
					bank:     fmt.Sprintf("%08x", f.GlobalID),
					cont:     fmt.Sprintf("%08x", id),
					octets:   hexa(o.Data[off-24 : off+4]),
					lectures: fmt.Sprintf("A{n=%d mod=%d/%d}  B{n=%d}", a, ma, xa, bb),
				})
			}
		}
	}

	fmt.Println()
	fmt.Println("=== 1. LES OCTETS, AVANT TOUTE LECTURE (24 octets avant la liste d'enfants) ===")
	sort.Slice(echs, func(i, j int) bool {
		if echs[i].bank != echs[j].bank {
			return echs[i].bank < echs[j].bank
		}
		return echs[i].cont < echs[j].cont
	})
	for _, e := range echs {
		fmt.Printf("  %s/%s  %s   %s\n", e.bank, e.cont, e.octets, e.lectures)
	}
	if len(echs) == 0 {
		fmt.Println("  (aucune banque ciblee — passer -banks)")
	}

	fmt.Println()
	fmt.Println("=== 2. PLAUSIBILITE DES DEUX LAYOUTS ===")
	fmt.Printf("  conteneurs de type 5 balayes            : %d\n", total)
	fmt.Printf("  dont trois transitions plausibles       : %d (%.1f %%)\n",
		transitOK, 100*float64(transitOK)/float64(max(total, 1)))
	fmt.Printf("  layout A (avec sLoopMod*) plausible      : %d (%.1f %%)\n",
		plausA, 100*float64(plausA)/float64(max(transitOK, 1)))
	fmt.Printf("  layout B (sans sLoopMod*) plausible      : %d (%.1f %%)\n",
		plausB, 100*float64(plausB)/float64(max(transitOK, 1)))

	fmt.Println()
	fmt.Println("=== 3. TEMOIN DE COHERENCE : compteur a 0 face au drapeau bIsContinuous ===")
	fmt.Println("  (les deux disent « enchainer sans s'arreter » ; ils doivent aller ensemble)")
	fmt.Printf("  %-24s %10s %10s\n", "", "continu", "pas a pas")
	for _, inf := range []bool{true, false} {
		etiq := "compteur = 0 (infini)"
		if !inf {
			etiq = "compteur >= 1"
		}
		fmt.Printf("  %-24s %10d %10d\n", etiq, croix[[2]bool{inf, true}], croix[[2]bool{inf, false}])
	}

	fmt.Println()
	fmt.Println("=== 4. COMBIEN DE REPETITIONS (0 = BOUCLE INFINIE) ===")
	defer imprimerModes(modes)
	cles := make([]int, 0, len(repartition))
	for k := range repartition {
		cles = append(cles, k)
	}
	sort.Ints(cles)
	for _, k := range cles {
		etiq := ""
		if k == 0 {
			etiq = "  <- BOUCLE INFINIE"
		}
		fmt.Printf("  %4d repetition(s) : %6d%s\n", k, repartition[k], etiq)
	}
	return nil
}

func hexa(b []byte) string {
	const chiffres = "0123456789abcdef"
	out := make([]byte, 0, len(b)*3)
	for i, c := range b {
		if i > 0 && i%4 == 0 {
			out = append(out, ' ')
		}
		out = append(out, chiffres[c>>4], chiffres[c&0x0f])
	}
	return string(out)
}

// imprimerModes rend la repartition des modes d'enchainement des conteneurs qui se repetent.
// Sans elle, « x3 » ne dit pas si les trois lectures se suivent ou se chevauchent.
func imprimerModes(modes map[int]int) {
	fmt.Println()
	fmt.Println("=== 5. COMMENT LES LECTURES SE SUIVENT (conteneurs qui se repetent) ===")
	noms := map[int]string{
		transitionAucune:      "bout a bout",
		transitionFonduAmp:    "fondu enchaine (amplitude)",
		transitionFonduPuiss:  "fondu enchaine (puissance)",
		transitionDelai:       "silence entre deux lectures",
		transitionEchantillon: "bout a bout, a l'echantillon",
		transitionCadence:     transitionCadenceLabel,
	}
	cles := make([]int, 0, len(modes))
	for k := range modes {
		cles = append(cles, k)
	}
	sort.Ints(cles)
	for _, k := range cles {
		fmt.Printf("  mode %d %-32s : %6d\n", k, noms[k], modes[k])
	}
	if len(cles) == 0 {
		fmt.Println("  (aucun)")
	}
}
