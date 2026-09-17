package killsource

// calibrate.go — L INFERENCE DES LARGEURS, DEVENUE ORACLE, ET LA CALIBRATION DE `param_4`.
//
// # CE QUI A CHANGE AU LOT 3.4.1-b, ET POURQUOI
//
// Deux parametres du decodeur de replication sont installes AU CHARGEMENT DE LA CARTE et ne se
// lisent nulle part dans le film : les largeurs d axe de position (`axisW`) et la largeur
// d index de plage (`indexW`). Ce fichier les INFERAIT, par balayage, faute de les connaitre.
//
// ON LES CONNAIT. Elles sont dans le catalogue des cartes, calculees des bornes par la loi du
// moteur (`profile/loi_largeurs.go`, accord 79 cartes sur 79) et posees au profil par
// [profile.MapQuantEntry.PrecisionAbsolue]. L inference NE DECIDE DONC PLUS : la valeur LUE
// prime sur la valeur mesuree (arbitrage utilisateur V17, M3-Q8 — « 3.4 lira la valeur dans le
// film plutot que de la mesurer »). Le balayage RESTE, comme ORACLE : il est le seul juge
// INTERNE AU FILM dont on dispose pour dire qu une entree de catalogue ment, et un balayage
// qu on retire est une mesure qu on cesse de faire.
//
// CE QUE LE BALAYAGE PROUVAIT, ET CE QU IL PROUVE ENCORE : les quatre films de reference
// rendaient QUATRE couples DISTINCTS (14/1, 17/2, 16/2, 17/1) — c est-a-dire exactement ce
// qu une grandeur PAR CARTE produit. La lecture Ghidra du remplisseur (`FUN_140be9a14`, note
// 3.4 du 2026-09-16) explique enfin POURQUOI, et d ou la valeur vient.
//
// CRITERE, ET IL EST INTERNE : nombre de records d archetype biped decodes sans desynchronisation
// sur un echantillon FIGE de paquets sans event. Il ne regarde ni le kill-feed, ni une arme, ni
// aucun oracle externe. GARDE-FOU : la configuration designee doit dominer la MEDIANE d un
// facteur >= 2 ; un profil plat signifie que le parametre reel n est pas dans l espace balaye,
// et alors l oracle ne designe rien (champ `Flat`) — il ne contredit personne.
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

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// calibration : ce que le decodeur APPLIQUE, et ce que le balayage MESURE a cote.
type calibration struct {
	// LueAxisW / LueIndexW : LES VALEURS QUE LE DECODAGE APPLIQUE, lues au profil — les trois
	// largeurs d axe de la table PAR INDEX de la carte et la largeur d index de plage
	// ([profile.MapQuantEntry.PrecisionAbsolue]). Elles PRIMENT (V17, M3-Q8).
	LueAxisW  [3]uint
	LueIndexW uint
	// AxisW / IndexW : ce que le BALAYAGE designe. Depuis le lot 3.4.1 il ne DECIDE plus : il
	// sert d ORACLE, confronte aux valeurs lues ci-dessus.
	AxisW, IndexW uint
	// CarteLue : les largeurs viennent-elles de l entree de catalogue de la CARTE du match ?
	// FAUX = repli `repli_carte_absente_largeurs_par_defaut` — l invariant conserve, c est-a-dire
	// les largeurs d UNE carte (`cliffhanger`) appliquees a celle-ci. Jamais un zero muet : le
	// rendu lisible le dit, et `Decode` l avertit par film.
	CarteLue bool
	// Desaccords : nombre de grandeurs ou l inference CONTREDIT la valeur lue — la largeur
	// d axe (hors du voisinage du triplet lu) et la largeur d index de plage. Zero = l oracle
	// ne contredit pas la carte. Cf. [infererLargeurs] pour la portee exacte du critere.
	Desaccords    int
	Score, Median int
	Flat          bool // le profil est plat : l inference n a rien designe de net
	RSP           uint32
	RSPRatio      float64
	// Profil est le PROFIL DE BALAYAGE retenu, celui que toutes les passes qui suivent posent
	// sur leur lecteur de bits (lot 2.2.a pour le mouvement, elargi au `param_4` au lot 2.3).
	//
	// AVANT CES LOTS, LE RESULTAT DE LA CALIBRATION N ETAIT NULLE PART : il etait ECRIT dans
	// des variables de paquet de `grammar` (`SetAbsoluteAxisW`, `TraversalPrecision`,
	// `SetRecordStateParam`) que la passe suivante relisait sans le savoir — `runWalk`
	// reconstruisait un `DefaultFrameConfig` et heritait pourtant des largeurs calibrees, par
	// effet de bord du processus, et LA CUISSON DU REJEU aussi (decouverte D1 du lot 2.2.a).
	// Il est desormais RENDU, et passe explicitement par `FrameConfig.Profil` puis par
	// [Result.ProfilCalibre].
	Profil grammar.ProfilDeBalayage
}

func (c calibration) String() string {
	src := fmt.Sprintf("score %d, mediane %d", c.Score, c.Median)
	if c.Flat {
		src = fmt.Sprintf("PROFIL PLAT (score %d, mediane %d)", c.Score, c.Median)
	}
	source := "CARTE"
	if !c.CarteLue {
		source = "DEFAUT (carte absente)"
	}
	return fmt.Sprintf("LU axisW=%v indexW=%d [%s] | ORACLE axisW=%d indexW=%d [%s] "+
		"desaccords=%d | recordStateParam=%d [croissance x%.3f]",
		c.LueAxisW, c.LueIndexW, source, c.AxisW, c.IndexW, src, c.Desaccords, c.RSP, c.RSPRatio)
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

// calibrate : pose le profil du film, MESURE l inference a cote, et calibre `recordStateParam`.
// `tl` doit etre une timeline REMBOBINEE : le balayage la parcourt chronologiquement.
//
// L INFERENCE DES LARGEURS NE DECIDE PLUS (lot 3.4.1-b, V17 M3-Q8 : « la valeur LUE prime sur
// la valeur mesuree »). Le profil porte les largeurs de la table PAR INDEX de la carte et la
// largeur d index de plage ; le balayage reste, il rend un VERDICT qu on confronte — c est le
// seul oracle INTERNE AU FILM dont on dispose pour dire qu une entree de catalogue ment.
func calibrate(f *film, tl *timeline, views int, carte *profile.MapQuantEntry) calibration {
	profil, carteLue := ProfilDeDepartPourCarte(carte)
	abs := profil.LargeursObjetDuMonde()
	res := calibration{Profil: profil, CarteLue: carteLue, LueAxisW: abs.AxisW, LueIndexW: abs.IndexW}
	infererLargeurs(f, tl, views, &res)
	calibrateRSP(f, tl, views, &res)
	return res
}

// infererLargeurs : L ORACLE. Il balaie les 63 configurations, retient celle qui maximise le
// nombre de records de bipede lus sans desynchronisation, et COMPTE les desaccords avec ce que
// le profil a pose. Il n ECRIT rien dans `res.Profil`.
//
// LE GARDE-FOU EST INCHANGE : une configuration qui ne domine pas la MEDIANE d un facteur 2
// signifie que le parametre reel n est pas dans l espace balaye. L inference ne designe alors
// rien (`Flat`), et il n y a pas de desaccord a compter — un oracle qui ne voit rien ne
// contredit personne.
func infererLargeurs(f *film, tl *timeline, views int, res *calibration) {
	sample := calibSample(f, calibSampleSize)
	cfg := grammar.DefaultFrameConfig()
	cfg.Profil = res.Profil
	saved := cfg.Profil.Mouvement.Traversal

	type cand struct {
		aw, iw uint
		score  int
	}
	out := make([]cand, 0, (axisWMax-axisWMin+1)*(indexWMax-indexWMin+1))
	for iw := indexWMin; iw <= indexWMax; iw++ {
		for aw := axisWMin; aw <= axisWMax; aw++ {
			cfg.Profil.Mouvement.WorldObject.AxisW = [3]uint{aw, aw, aw}
			cfg.Profil.Mouvement.WorldObject.IndexW = iw
			cfg.Profil.Mouvement.Traversal = profile.PrecisionDescriptor{IndexW: iw, AxisW: saved.AxisW}
			out = append(out, cand{aw, iw, countBipedRecords(sample, tl, cfg, views)})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].score > out[j].score })
	med := out[len(out)/2].score
	res.AxisW, res.IndexW, res.Score, res.Median = out[0].aw, out[0].iw, out[0].score, med
	if float64(out[0].score) < flatRatio*float64(max1(med)) {
		res.Flat = true
		return
	}
	// LE DESACCORD SE COMPTE A LA PORTEE DE L ORACLE, ET PAS PLUS FINEMENT (lot 3.4.1).
	//
	// L oracle balaie une largeur UNIFORME ; la verite est un TRIPLET par axe (17/17/15 sur
	// Fragmentation). Exiger l egalite ferait donc un desaccord sur toute carte dont les trois
	// axes ne sont pas egaux — c est-a-dire presque toutes — et le compteur ne dirait plus rien.
	// Ce qu une sonde uniforme peut dire, et qu elle dit ici : la valeur lue est-elle DANS son
	// voisinage ? Un oracle a 16 contre `[17 17 15]` ne contredit pas la carte ; un oracle a 16
	// contre `[13 13 14]` — l invariant applique a une carte qui n est pas la sienne — si.
	if res.AxisW < minLargeur(res.LueAxisW) || res.AxisW > maxLargeur(res.LueAxisW) {
		res.Desaccords++
	}
	if res.LueIndexW != res.IndexW {
		res.Desaccords++
	}
}

// minLargeur / maxLargeur : les bornes du triplet de largeurs lu.
func minLargeur(w [3]uint) uint {
	m := w[0]
	for _, v := range w[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func maxLargeur(w [3]uint) uint {
	m := w[0]
	for _, v := range w[1:] {
		if v > m {
			m = v
		}
	}
	return m
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
func countBipedRecords(sample []*packet, tl *timeline, cfg grammar.FrameConfig, views int) int {
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
	// LE CADRE PORTE LE PROFIL DEJA CALIBRE : le balayage de `recordStateParam` doit se juger
	// aux largeurs retenues, pas aux largeurs par defaut. C etait vrai avant le lot 2.2.a
	// parce que les largeurs vivaient dans le processus ; c est desormais ecrit.
	cfg := grammar.DefaultFrameConfig()
	cfg.Profil = res.Profil
	best, bestN, worstN := uint32(0), -1, 1<<62
	for r := uint32(0); r <= rspMax; r++ {
		cfg.Profil.PoserParamEtat(r)
		n := monotonicScore(sm, tl, cfg, views)
		if n > bestN {
			best, bestN = r, n
		}
		if n < worstN {
			worstN = n
		}
	}
	res.Profil.PoserParamEtat(best)
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
func monotonicScore(t0 []packet, tl *timeline, cfg grammar.FrameConfig, views int) int {
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
