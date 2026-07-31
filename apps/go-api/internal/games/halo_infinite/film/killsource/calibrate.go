package killsource

// calibrate.go — LA CALIBRATION MAP-DEPENDANTE, AUTOMATIQUE.
//
// Deux parametres du decodeur de replication sont installes AU CHARGEMENT DE LA MAP et ne se
// lisent nulle part dans le film : la largeur d axe de position (`axisW`) et la largeur d index
// (`indexW`). Tant qu ils devaient etre fournis a la main, le decodeur etait MONO-FILM. C est la
// seule chose qui separait << un decodeur >> de << deux resultats >>.
//
// LE PARAMETRE EST REELLEMENT MAP-DEPENDANT : les quatre films de reference rendent QUATRE
// couples DISTINCTS (14/1, 17/2, 16/2, 17/1). Il n y a donc rien a coder en dur, et les tables
// de precision dumpees du jeu ne remplacent PAS le balayage : injectees, elles effondrent trois
// films sur quatre (leur contenu est un instantane installe PAR MAP, pas une specification
// portable).
//
// CRITERE, ET IL EST INTERNE : nombre de records d archetype biped decodes sans desynchronisation
// sur un echantillon FIGE de paquets sans event. Il ne regarde ni le kill-feed, ni une arme, ni
// aucun oracle. GARDE-FOU : la configuration retenue doit dominer la MEDIANE d un facteur >= 2 ;
// un profil plat signifie que le parametre reel n est pas dans l espace balaye, et alors on ne
// devine pas — on garde les valeurs par defaut et on le DIT (champ `Flat`).
//
// LE MONDE DOIT ETRE CHRONOLOGIQUE PENDANT LE BALAYAGE, et c est un correctif paye cher : avancer
// le monde a la FIN du film puis echantillonner des paquets du DEBUT est anodin a 8 joueurs, mais
// en BTB les slots de biped sont RECYCLES AVEC UNE NOUVELLE GENERATION — 387 paquets sur 400
// meurent alors au premier record, et la couverture tombe a 4.4 %. Avec le monde chronologique
// elle remonte a 60.2 %, et le critere choisit la MEME calibration sur les quatre films de
// reference : le correctif est SANS REGRESSION.
//
// TROISIEME PARAMETRE : `recordStateParam`, qui fait varier la largeur de trois composants et
// n est pas dans le flux. Le critere ci-dessus en est TOTALEMENT AVEUGLE (score identique pour
// toutes ses valeurs) parce qu une largeur fausse produit souvent un << faux-propre >> : le record
// se referme, mais au mauvais bit. Il faut donc un critere qui punisse le DECALAGE et non la
// desynchronisation — la CROISSANCE DES SLOTS. Ce choix pese ~8 morts.

import (
	"fmt"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// calibration : ce que le balayage a retenu.
type calibration struct {
	AxisW, IndexW uint
	Score, Median int
	Flat          bool // le profil est plat : les valeurs par defaut ont ete conservees
	RSP           uint32
	RSPRatio      float64
}

func (c calibration) String() string {
	src := fmt.Sprintf("score %d, mediane %d", c.Score, c.Median)
	if c.Flat {
		src = fmt.Sprintf("PROFIL PLAT (score %d, mediane %d) : valeurs par defaut conservees",
			c.Score, c.Median)
	}
	return fmt.Sprintf("axisW=%d indexW=%d [%s] | recordStateParam=%d [croissance x%.3f]",
		c.AxisW, c.IndexW, src, c.RSP, c.RSPRatio)
}

// bornes de l espace balaye : 21 largeurs d axe x 3 largeurs d index = 63 configurations.
const (
	axisWMin, axisWMax   = uint(6), uint(26)
	indexWMin, indexWMax = uint(1), uint(3)
	calibSampleSize      = 400
	rspMax               = uint32(5)
	rspStride            = 6
	flatRatio            = 2.0
)

// calibrate : applique la calibration map-dependante et rend ce qui a ete retenu. `tl` doit etre
// une timeline REMBOBINEE : le balayage la parcourt chronologiquement.
func calibrate(f *film, tl *timeline, views int) calibration {
	sample := calibSample(f, calibSampleSize)
	cfg := filmdec.DefaultFrameConfig()
	saved := filmdec.TraversalPrecision

	type cand struct {
		aw, iw uint
		score  int
	}
	out := make([]cand, 0, (axisWMax-axisWMin+1)*(indexWMax-indexWMin+1))
	for iw := indexWMin; iw <= indexWMax; iw++ {
		for aw := axisWMin; aw <= axisWMax; aw++ {
			filmdec.SetAbsoluteAxisW(aw)
			filmdec.TraversalPrecision = filmdec.PrecisionDescriptor{IndexW: iw, AxisW: saved.AxisW}
			out = append(out, cand{aw, iw, countBipedRecords(sample, tl, cfg, views)})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].score > out[j].score })
	med := out[len(out)/2].score
	res := calibration{AxisW: out[0].aw, IndexW: out[0].iw, Score: out[0].score, Median: med}
	if float64(out[0].score) < flatRatio*float64(max1(med)) {
		res.Flat = true
		res.AxisW, res.IndexW = 14, 1
	}
	filmdec.SetAbsoluteAxisW(res.AxisW)
	filmdec.TraversalPrecision = filmdec.PrecisionDescriptor{IndexW: res.IndexW, AxisW: saved.AxisW}
	calibrateRSP(f, tl, views, &res)
	return res
}

// calibSample : echantillon FIGE de paquets type-0 SANS event, de taille utile. Leur boucle de
// records demarre au bit 2, connu : aucun localisateur n intervient dans le critere.
func calibSample(f *film, n int) []*packet {
	var sm []*packet
	for i := range f.packets {
		p := &f.packets[i]
		if p.typ != packetType0 || hasEvents(p) || len(p.payload) < 400 {
			continue
		}
		sm = append(sm, p)
		if len(sm) >= n {
			break
		}
	}
	return sm
}

// countBipedRecords : le critere de calibration. Le monde est restaure apres chaque paquet : la
// calibration ne doit RIEN laisser derriere elle.
func countBipedRecords(sample []*packet, tl *timeline, cfg filmdec.FrameConfig, views int) int {
	n := 0
	for _, p := range sample {
		snap := tl.w.Snapshot()
		recs := walkFrom(p.payload, tl.w, cfg, 2, views)
		tl.w.Restore(snap)
		for k := range recs {
			if recs[k].DesyncAt == -1 && recs[k].TypeIndex == bipedArchetype {
				n++
			}
		}
	}
	return n
}

// calibrateRSP : calibration de `recordStateParam` au critere de CROISSANCE DES SLOTS, sur un
// sous-echantillon regulier des paquets type-0.
func calibrateRSP(f *film, tl *timeline, views int, res *calibration) {
	var sm []packet
	for i := 0; i < len(f.t0); i += rspStride {
		sm = append(sm, f.t0[i])
	}
	cfg := filmdec.DefaultFrameConfig()
	best, bestN, worstN := uint32(0), -1, 1<<62
	for r := uint32(0); r <= rspMax; r++ {
		filmdec.SetRecordStateParam(r)
		n := monotonicScore(sm, tl, cfg, views)
		if n > bestN {
			best, bestN = r, n
		}
		if n < worstN {
			worstN = n
		}
	}
	filmdec.SetRecordStateParam(best)
	res.RSP = best
	res.RSPRatio = float64(bestN) / float64(max1(worstN))
}

// monotonicScore : sur les paquets localises, nombre de records lus avant la premiere violation
// STRUCTURELLE — desynchronisation, ou slot qui n augmente pas.
//
// DEUX CONTRAINTES INTERNES AU FLUX, aucune source externe :
//
//	FERMETURE   la boucle de records se termine par un marqueur de fin dans le dernier octet ;
//	            un curseur decale n y tombe quasiment jamais.
//	CROISSANCE  les slots d une meme boucle sont ordonnes CROISSANT ; un slot qui recule est la
//	            signature d une lecture de bits de bourrage.
func monotonicScore(t0 []packet, tl *timeline, cfg filmdec.FrameConfig, views int) int {
	records := 0
	tl.rewind()
	for i := range t0 {
		p := &t0[i]
		w := tl.advanceTo(p.ts)
		start := 2
		if hasEvents(p) {
			s := locateRecords(p.payload, w, cfg)
			if s < 0 {
				continue
			}
			start = s
		}
		snap := w.Snapshot()
		recs := walkFrom(p.payload, w, cfg, start, views)
		w.Restore(snap)
		for k := range recs {
			if recs[k].DesyncAt != -1 || (k > 0 && recs[k].Slot <= recs[k-1].Slot) {
				break
			}
			records++
		}
	}
	tl.rewind()
	return records
}

// max1 : denominateur jamais nul.
func max1(n int) int {
	if n < 1 {
		return 1
	}
	return n
}
