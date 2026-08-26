package replay

// inventory_position_i22_test.go — MESURER LA POSITION DU MOTIF i22 (grenades) RELATIVEMENT A
// DES REPERES INDEPENDANTS DE L'ANCRE DE CAPACITE.
//
// LA QUESTION. La mesure du 2026-08-24 (MESURE_TROUS_INVENTAIRE_2026-08-24.md) etablit que R2
// (grenades) demarre a la position de l'ancre R1 (capacite), absente de 80,9 % des records :
// 4 278 records sur 6 721 sont ARMES (R3+R4 passent) mais sans grenade. Or R3 et R4 trouvent
// leur position SANS R1 — le record porte donc des reperes qui marchent. Si i22 occupe une
// position STRUCTURELLE mesurable relativement a l'un d'eux, R2 se decouple de R1.
//
// CE QUI EST MESURE, sur les seuls records ou R1 ET R2 ET R4 reussissent (le corpus
// d'entrainement : la position VRAIE d'i22 y est connue) :
//
//	l'offset en bits entre le debut du motif i22 retenu par la production et chacun des
//	reperes candidats — debut du record, ancre de capacite (temoin : le repere actuel),
//	debut du bloc de munitions, fin du bloc (bit de porte d'i43), premiere famille d'arme.
//
// UN REPERE EST EXPLOITABLE si la distribution de son offset CONCENTRE : un petit ensemble de
// decalages couvre l'essentiel des records. Une distribution etalee dit que le repere ne borne
// rien.
//
// LE TEMOIN DE HASARD EST MESURE EN MEME TEMPS : combien de candidats i22 (R(3)=4 puis quatre
// R(8) bornes) le record porte-t-il AILLEURS ? Si le record en porte des dizaines, une regle de
// position doit etre etroite ou elle ramassera du bruit. C'est la lecon de la mesure du 24/08,
// ou le temoin rang+1 (443) battait le signal (328).
//
// LECTURE SEULE. Gate par variable d'environnement, saute en CI. Meme corpus que
// TestInventaireTrousMesure (INV_FILMS / INV_CACHE / INV_SAMPLE).
//
// USAGE (depuis apps/go-api) :
//
//	INV_CACHE=<repo>/data/cache/film_chunks INV_SAMPLE=24 \
//	  go test ./internal/analysis/replay/ -run '^TestPositionI22$' -timeout 180m -v

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// invPosCand est un candidat i22 : un motif R(3)=4 suivi de quatre R(8) tous bornes.
type invPosCand struct {
	bit int
	sum uint32
}

// invPosCands releve TOUS les candidats i22 d'un intervalle de bits. C'est la meme grammaire
// que invGrenadesAfter, SANS le critere `somme > 0` : la somme est relevee a cote, pour que
// « zero grenade » reste distinguable de « pas de motif ».
func invPosCands(pay []byte, from, to int, maxVal uint32) []invPosCand {
	var out []invPosCand
	for b := from; b+35 <= to; b++ {
		if invBits(pay, b, 3) != 4 {
			continue
		}
		sum, ok := uint32(0), true
		for i := 0; i < invGrenadeSlots && ok; i++ {
			v := invBits(pay, b+3+8*i, 8)
			if v > maxVal {
				ok = false
			}
			sum += v
		}
		if ok {
			out = append(out, invPosCand{bit: b, sum: sum})
		}
	}
	return out
}

// invPosRepere est un repere candidat : un nom et la fonction qui rend sa position bit.
type invPosRepere struct {
	nom string
	pos int
	ok  bool
}

// invPosObs est une observation : un record du corpus d'entrainement.
type invPosObs struct {
	film       string
	slot       uint32
	chunk      int
	i22        int // position VRAIE du motif i22 (celle que la production retient)
	i22Sum     uint32
	reperes    []invPosRepere
	nbCands    int // candidats i22 dans TOUT le record
	nbCandsAmt int // candidats i22 dans les 400 bits precedant le bloc de munitions
	largeur    int
	// winCands est la liste des candidats i22 dont le debut tombe dans la FENETRE LARGE
	// [debutAmmo-invPosWinLarge, debutAmmo], leur position exprimee RELATIVEMENT a debutAmmo
	// (donc negative). C'est la matiere premiere des regles de selection evaluees au rapport.
	winCands []invPosCand
	i22Rel   int // position vraie d'i22, relative a debutAmmo
	// r2bGren / r2bGrenOk : SEULEMENT pour la population invPosAncreSommeNulle. Ce que la voie
	// de PRODUCTION (invGrenadesNearAmmo, meme fonction que le decodeur) rendrait sur ce record,
	// calcule ici pendant qu'on tient encore le payload — pas une redecouverte separee. Sert a
	// verifier que la somme nulle mesuree par l'ancre tient aussi par la position (cf.
	// invPosVerifieAncreSommeNulle).
	r2bGren   [invGrenadeSlots]uint32
	r2bGrenOk bool
}

// invPosWinLarge est la profondeur de la fenetre exploratoire avant le bloc de munitions. Elle
// est VOLONTAIREMENT plus large que la dispersion attendue : c'est la mesure qui doit resserrer
// la fenetre, pas la fenetre qui doit confirmer la mesure.
const invPosWinLarge = 400

// invPosWinLo / invPosWinHi sont la FENETRE DE LA LOI, en bits relativement au debut du bloc de
// munitions. La loi MESUREE sur 1 167 records d'entrainement (24 films) est [-204,-139] ; la
// fenetre l'elargit de 12 bits de chaque cote. Le candidat parasite le plus proche EN DESSOUS
// d'une position vraie est a 105 bits (minimum sur 1 167 records) : la marge basse reste donc
// large de 93 bits apres elargissement.
const (
	invPosWinLo = -216
	invPosWinHi = -127
)

// invPosHisto est un histogramme d'entiers, rendu trie par effectif decroissant.
type invPosHisto map[int]int

func (h invPosHisto) top(n int) string {
	type kv struct{ k, v int }
	var xs []kv
	for k, v := range h {
		xs = append(xs, kv{k, v})
	}
	sort.Slice(xs, func(i, j int) bool {
		if xs[i].v != xs[j].v {
			return xs[i].v > xs[j].v
		}
		return xs[i].k < xs[j].k
	})
	var sb strings.Builder
	total := 0
	for _, x := range xs {
		total += x.v
	}
	for i, x := range xs {
		if i >= n {
			fmt.Fprintf(&sb, " ... (%d valeurs distinctes)", len(xs))
			break
		}
		fmt.Fprintf(&sb, " %d:%d(%.1f%%)", x.k, x.v, 100*float64(x.v)/float64(total))
	}
	return sb.String()
}

func TestPositionI22(t *testing.T) {
	films := invTrousCorpus(t)
	if len(films) == 0 {
		t.Skipf("%s ou %s est requis — mesure sautee", invTrousFilmsEnv, invTrousCacheEnv)
	}
	var obs, sansAncre, ancreZero []invPosObs
	for _, dir := range films {
		a, b, c := invPosFilm(t, dir)
		obs = append(obs, a...)
		sansAncre = append(sansAncre, b...)
		ancreZero = append(ancreZero, c...)
	}
	if len(obs) == 0 {
		t.Fatalf("aucun record d'entrainement (R1+R2+R4) sur %d films", len(films))
	}
	invPosRapport(t, obs)
	invPosTransfert(t, obs, sansAncre)
	invPosVerifieAncreSommeNulle(t, films, ancreZero)
}

// invPosFilm collecte les observations d'UN film : le corpus d'ENTRAINEMENT (R1+R2+R4 vrais,
// position d'i22 connue), la population CIBLE (R1 echoue, R4 passe — les records armes sans
// grenade que ce lot vise), et la population ANCRE SOMME NULLE (R1 reussit mais AUCUN motif
// i22 de somme non nulle ne suit — le cas « 104/104 » du 24/08 §4.4, ni entrainement ni cible :
// avant cette instrumentation, invPosObserve l'ecartait en invPosHorsSujet et elle n'etait
// controlee nulle part).
func invPosFilm(t *testing.T, dir string) (entr, cible, ancreZero []invPosObs) {
	t.Helper()
	release := filmdec.LockProcessDecode()
	defer release()
	known := loadoutFamilies()
	nom := invPosBase(dir)
	n := filmdec.CountFilmChunks(dir)
	for ch := 1; ch <= n; ch++ {
		chunk, err := filmdec.ReadFilmChunk(dir, ch)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			pay := p.Payload(chunk)
			for _, sp := range invRecordSpans(pay) {
				if sp.ti != invBipedTI {
					continue
				}
				o, classe := invPosObserve(pay, sp, known)
				if classe == invPosHorsSujet {
					continue
				}
				o.film, o.chunk = nom, ch
				switch classe {
				case invPosEntrainement:
					entr = append(entr, o)
				case invPosAncreSommeNulle:
					ancreZero = append(ancreZero, o)
				default:
					cible = append(cible, o)
				}
			}
		}
	}
	t.Logf("%s : %d records d'entrainement, %d records cibles (sans ancre, armes), "+
		"%d records ancre-somme-nulle",
		nom, len(entr), len(cible), len(ancreZero))
	return entr, cible, ancreZero
}

// invPosClasse dit a quelle population un record appartient.
type invPosClasse int

const (
	invPosHorsSujet invPosClasse = iota
	invPosEntrainement
	invPosCible
	// invPosAncreSommeNulle : R1 rend une ancre UNIQUE mais AUCUN motif i22 apres elle n'a une
	// somme non nulle — le cas « 104/104 » du 24/08 §4.4 (un Spartan ayant lance toutes ses
	// grenades). Ni entrainement (R2a echoue, la position VRAIE n'est pas etablie par cette
	// voie) ni cible (l'ancre existe), cette population etait ecartee en invPosHorsSujet avant
	// cette instrumentation — donc invisible du corpus d'entrainement ET des trois controles.
	invPosAncreSommeNulle
)

func invPosBase(dir string) string {
	dir = strings.ReplaceAll(dir, "\\", "/")
	if i := strings.LastIndex(dir, "/"); i >= 0 {
		return dir[i+1:]
	}
	return dir
}

// invPosObserve classe UN record et rend son observation.
//
// ENTRAINEMENT : R1 rend une ancre unique, R2 trouve son motif apres elle, R4 resout le bloc de
// munitions — la position VRAIE d'i22 y est connue, et c'est ce qui en fait une reference.
// CIBLE : R1 ne rend RIEN (aucune ancre unique) mais R4 resout le bloc — les 63,7 % de records
// armes sans grenade que ce lot vise.
func invPosObserve(pay []byte, sp invRecordSpan, known map[uint32]bool) (invPosObs, invPosClasse) {
	// R4 d'abord : sans bloc de munitions, aucun repere independant, donc aucune des deux
	// populations n'est instruite.
	first, ok := invFirstFamily(pay, sp.from, sp.to, known)
	if !ok {
		return invPosObs{}, invPosHorsSujet
	}
	end := first - 1
	lo := end - invAmmoSearchSpan
	if lo < sp.from {
		lo = sp.from
	}
	sols := invSolveAmmoBlock(pay, end, lo)
	if len(sols) == 0 {
		return invPosObs{}, invPosHorsSujet
	}
	hits := invAbilityIn(pay, sp.from, sp.to)
	i22, sum, classe := -1, uint32(0), invPosCible
	var r2bGren [invGrenadeSlots]uint32
	var r2bGrenOk bool
	if len(hits) == 1 {
		// La position VRAIE d'i22 : le premier candidat de somme non nulle apres l'ancre — la
		// regle de production, mot pour mot.
		for _, c := range invPosCands(pay, hits[0].anchorBit, sp.to, DefaultGrenadeMax) {
			if c.sum > 0 {
				i22, sum, classe = c.bit, c.sum, invPosEntrainement
				break
			}
		}
		if classe != invPosEntrainement {
			// L'ancre est la mais AUCUN candidat de somme non nulle ne suit : classe a part
			// (invPosAncreSommeNulle), PAS ecartee. On y garde le premier candidat rencontre
			// (quel que soit sa somme — toujours nulle par construction de cette branche) comme
			// repere i22, et on y calcule DEJA ce que R2b (la voie de production,
			// invGrenadesNearAmmo) y lirait : la verification (invPosVerifieAncreSommeNulle)
			// n'a plus qu'a la consommer, sans redecoder.
			classe = invPosAncreSommeNulle
			if cands := invPosCands(pay, hits[0].anchorBit, sp.to, DefaultGrenadeMax); len(cands) > 0 {
				i22, sum = cands[0].bit, cands[0].sum
			}
			r2bGren, r2bGrenOk = invGrenadesNearAmmo(pay, sols[0], sp.from, DefaultGrenadeMax)
		}
	}
	last, hasLast := invLastFamily(pay, sp.from, sp.to, known)
	ancre := invPosRepere{nom: "ancreR1", ok: len(hits) == 1}
	if ancre.ok {
		ancre.pos = hits[0].anchorBit
	}
	o := invPosObs{
		slot: uint32(sp.slot), i22: i22, i22Sum: sum, largeur: sp.to - sp.from,
		r2bGren: r2bGren, r2bGrenOk: r2bGrenOk,
		reperes: []invPosRepere{
			{nom: "debutRecord", pos: sp.from, ok: true},
			ancre,
			{nom: "debutAmmo", pos: sols[0], ok: true},
			{nom: "finAmmo", pos: end, ok: true},
			{nom: "1ereFamille", pos: first, ok: true},
			{nom: "derniereFamille", pos: last, ok: hasLast},
			{nom: "finRecord", pos: sp.to, ok: true},
		},
		nbCands: len(invPosCands(pay, sp.from, sp.to, DefaultGrenadeMax)),
	}
	amtLo := sols[0] - invPosWinLarge
	if amtLo < sp.from {
		amtLo = sp.from
	}
	win := invPosCands(pay, amtLo, sols[0], DefaultGrenadeMax)
	o.nbCandsAmt = len(win)
	o.i22Rel = i22 - sols[0]
	if i22 < 0 {
		o.i22Rel = 1 // hors de toute fenetre : aucune position vraie a comparer
	}
	for _, c := range win {
		o.winCands = append(o.winCands, invPosCand{bit: c.bit - sols[0], sum: c.sum})
	}
	return o, classe
}

// invPosTransfert est LE TEST REFUTABLE du lot : la loi de position, etablie sur les records
// PORTANT l'ancre, tient-elle sur ceux qui ne la portent pas ?
//
// TROIS FAITS SONT CONFRONTES, et un seul d'entre eux suffit a refuter :
//  1. le TAUX DE LECTURE — combien de records cibles rendent un candidat dans la fenetre ;
//  2. la FORME de la distribution des offsets — si la loi est reelle, les offsets trouves se
//     concentrent sur les MEMES modes que le corpus d'entrainement ; s'ils s'etalent
//     uniformement sur la fenetre, c'est du bruit ramasse par une fenetre trop large ;
//  3. le TEMOIN — la meme regle sur une fenetre DECALEE hors de la loi. S'il rend autant de
//     candidats que le signal, la fenetre ne borne rien (lecon de la mesure du 24/08, ou le
//     temoin rang+1 battait le signal).
func invPosTransfert(t *testing.T, entr, cible []invPosObs) {
	t.Helper()
	t.Logf("=== TRANSFERT : %d records cibles (R1 echoue, R4 passe) ===", len(cible))
	if len(cible) == 0 {
		return
	}
	sig := invPosRegle{lo: invPosWinLo, hi: invPosWinHi, strat: "premier"}
	dec := invPosWinHi - invPosWinLo + 1
	tem := invPosRegle{lo: sig.lo - 2*dec, hi: sig.hi - 2*dec, strat: "premier"}
	for _, cas := range []struct {
		nom string
		r   invPosRegle
	}{{"signal", sig}, {"temoin decale", tem}} {
		hOff, hSum := invPosHisto{}, invPosHisto{}
		rendus := 0
		for _, o := range cible {
			c, ok := invPosApplique(cas.r, o.winCands)
			if !ok {
				continue
			}
			rendus++
			hOff[c.bit]++
			hSum[int(c.sum)]++
		}
		t.Logf("%-14s fenetre [%d,%d] : %d/%d rendus (%.1f%%)",
			cas.nom, cas.r.lo, cas.r.hi, rendus, len(cible),
			100*float64(rendus)/float64(len(cible)))
		t.Logf("    offsets :%s", hOff.top(12))
		t.Logf("    sommes  :%s", hSum.top(8))
	}
	// La distribution de reference, pour comparer les formes a l'oeil nu.
	hRef := invPosHisto{}
	for _, o := range entr {
		hRef[o.i22Rel]++
	}
	t.Logf("reference (entrainement) offsets :%s", hRef.top(12))
}

// invPosVerifieAncreSommeNulle est le CONTROLE MINIMAL sur la population ANCRE SOMME NULLE
// (invPosAncreSommeNulle) : l'ancre existe, mais R2a n'y trouve aucun motif i22 de somme non
// nulle — le cas « 104/104 » du 24/08 §4.4. Avant cette instrumentation, invPosObserve les
// ecartait en invPosHorsSujet : ni le corpus d'entrainement ni les trois controles ne les
// voyaient.
//
// CE QUI EST VERIFIE, pour chaque record de cette population : la lecture que R2b (la voie de
// PRODUCTION, invGrenadesNearAmmo — deja calculee dans o.r2bGren par invPosObserve, pas
// redecodee ici) en tire est-elle COHERENTE ? Deux issues acceptables :
//  1. R2b rend une somme NULLE lui aussi — la mesure par l'ancre (aucune somme non nulle nulle
//     part dans le record) et la mesure par la position s'accordent ;
//  2. R2b rend une somme NON nulle, mais chaque type qu'il dit porte a ete VU LANCE dans ce
//     film par l'oracle independant (memes canaux disjoints que TestOracleTypesPortesEtLances,
//     invPosTypes) — l'oracle « parle » (throws decodables) et ne contredit pas la lecture.
//
// Un record n'est PAS verifie (ni succes ni echec) si R2b lui-meme ne rend rien, ou si l'oracle
// est MUET sur ce film (throws illisibles) : on ne peut alors ni confirmer ni contredire, et il
// serait faux de compter cela comme un accord.
func invPosVerifieAncreSommeNulle(t *testing.T, films []string, ancreZero []invPosObs) {
	t.Helper()
	if len(ancreZero) == 0 {
		t.Logf("ANCRE SOMME NULLE : 0 record sur ce corpus — rien a verifier")
		return
	}
	lance := map[string][invGrenadeSlots]bool{}
	oracleMuetFilms := 0
	for _, dir := range films {
		_, l, err := invPosTypes(dir)
		if err != nil {
			oracleMuetFilms++
			continue
		}
		lance[invPosBase(dir)] = l
	}
	sommeNulle, oracleAccord, r2bMuet, oracleMuetRecord := 0, 0, 0, 0
	var contredits []string
	for _, o := range ancreZero {
		if !o.r2bGrenOk {
			r2bMuet++
			continue
		}
		var sum uint32
		for _, v := range o.r2bGren {
			sum += v
		}
		if sum == 0 {
			sommeNulle++
			continue
		}
		l, known := lance[o.film]
		if !known {
			oracleMuetRecord++
			continue
		}
		accord := true
		for r, v := range o.r2bGren {
			if v > 0 && !l[r] {
				accord = false
			}
		}
		if accord {
			oracleAccord++
		} else {
			contredits = append(contredits,
				fmt.Sprintf("%s/chunk%d/slot%d", o.film, o.chunk, o.slot))
		}
	}
	t.Logf("ANCRE SOMME NULLE : %d records — R2b somme nulle %d, R2b non-nulle accorde a "+
		"l'oracle %d, R2b muet %d, oracle muet (film) %d, oracle muet (record) %d",
		len(ancreZero), sommeNulle, oracleAccord, r2bMuet, oracleMuetFilms, oracleMuetRecord)
	if len(contredits) > 0 {
		t.Errorf("%d record(s) « ancre somme nulle » ou R2b lit un type que l'oracle des "+
			"lancers n'a jamais vu porte dans son film — le repli positionnel dererait sur "+
			"cette population sans que rien ne le signale : %v", len(contredits), contredits)
	}
}

// invPosRapport rend les histogrammes d'offset par repere, et le denombrement des candidats.
func invPosRapport(t *testing.T, obs []invPosObs) {
	t.Helper()
	t.Logf("=== corpus d'entrainement : %d records (R1+R2+R4 tous vrais) ===", len(obs))
	noms := []string{}
	for _, r := range obs[0].reperes {
		noms = append(noms, r.nom)
	}
	for i, nom := range noms {
		h := invPosHisto{}
		manquants := 0
		for _, o := range obs {
			r := o.reperes[i]
			if !r.ok {
				manquants++
				continue
			}
			h[o.i22-r.pos]++
		}
		t.Logf("repere %-16s (manquant %d) offsets i22-repere :%s", nom, manquants, h.top(12))
		t.Logf("    -> %d valeurs distinctes, couverture du top-3 : %.1f%%",
			len(h), invPosCouverture(h, 3))
	}
	// Densite des candidats : le denominateur du faux positif.
	hc, ha := invPosHisto{}, invPosHisto{}
	for _, o := range obs {
		hc[o.nbCands]++
		ha[o.nbCandsAmt]++
	}
	t.Logf("candidats i22 par record (tout le record) :%s", hc.top(10))
	t.Logf("candidats i22 dans les 400 bits avant le bloc ammo :%s", ha.top(10))
	// Somme nulle : combien de records d'entrainement portent AUSSI un candidat nul avant le
	// candidat retenu ? (une regle positionnelle acceptant la somme nulle doit le savoir)
	hs := invPosHisto{}
	for _, o := range obs {
		hs[int(o.i22Sum)]++
	}
	t.Logf("somme des compteurs du motif retenu :%s", hs.top(10))
	hl := invPosHisto{}
	for _, o := range obs {
		hl[o.largeur/1000]++
	}
	t.Logf("largeur des records d'entrainement (milliers de bits) :%s", hl.top(10))
	invPosRegles(t, obs)
}

// invPosRegle est une regle de selection positionnelle : une fenetre [lo,hi] relative au debut
// du bloc de munitions, et une strategie de departage entre les candidats qui y tombent.
type invPosRegle struct {
	lo, hi int
	strat  string // "dernier" (le plus proche du bloc ammo), "premier", "unique"
}

// invPosApplique rend le candidat que la regle retient, et si elle en retient un.
func invPosApplique(r invPosRegle, cands []invPosCand) (invPosCand, bool) {
	var in []invPosCand
	for _, c := range cands {
		if c.bit >= r.lo && c.bit <= r.hi {
			in = append(in, c)
		}
	}
	switch {
	case len(in) == 0:
		return invPosCand{}, false
	case r.strat == "unique":
		if len(in) != 1 {
			return invPosCand{}, false
		}
		return in[0], true
	case r.strat == "premier":
		return in[0], true
	default: // dernier
		return in[len(in)-1], true
	}
}

// invPosEval mesure une regle sur le corpus d'entrainement : combien de fois elle rend un
// candidat, et parmi eux combien tombent SUR la position vraie.
func invPosEval(r invPosRegle, obs []invPosObs) (rendus, justes int) {
	for _, o := range obs {
		c, ok := invPosApplique(r, o.winCands)
		if !ok {
			continue
		}
		rendus++
		if c.bit == o.i22Rel {
			justes++
		}
	}
	return
}

// invPosRegles cherche la meilleure fenetre par balayage, pour les trois strategies. Le score
// maximise `justes - faux` : une regle qui rend beaucoup de mauvaises lectures est PIRE qu'une
// regle qui se tait (la mesure du 24/08 l'a paye assez cher).
func invPosRegles(t *testing.T, obs []invPosObs) {
	t.Helper()
	for _, strat := range []string{"dernier", "premier", "unique"} {
		best, bestScore := invPosRegle{strat: strat}, -1<<30
		for lo := -invPosWinLarge; lo <= -40; lo += 5 {
			for hi := lo + 5; hi <= 0; hi += 5 {
				r := invPosRegle{lo: lo, hi: hi, strat: strat}
				rendus, justes := invPosEval(r, obs)
				if s := 2*justes - rendus; s > bestScore {
					best, bestScore = r, s
				}
			}
		}
		rendus, justes := invPosEval(best, obs)
		t.Logf("regle %-8s meilleure fenetre [%d,%d] : rendues %d/%d (%.1f%%), JUSTES %d (%.2f%% des rendues)",
			strat, best.lo, best.hi, rendus, len(obs),
			100*float64(rendus)/float64(len(obs)), justes,
			100*float64(justes)/float64(max(rendus, 1)))
	}
	invPosMarges(t, obs)
	// TEMOIN DE HASARD : la meme regle appliquee a une fenetre DECALEE de sa largeur. Si le
	// temoin rend autant de candidats que le signal, la fenetre ne borne rien.
	r := invPosRegle{lo: -240, hi: -120, strat: "dernier"}
	rendus, justes := invPosEval(r, obs)
	tem := invPosRegle{lo: r.lo - 120, hi: r.hi - 120, strat: "dernier"}
	tRendus, tJustes := invPosEval(tem, obs)
	t.Logf("temoin : fenetre [%d,%d] rend %d dont %d justes ; fenetre decalee [%d,%d] rend %d dont %d justes",
		r.lo, r.hi, rendus, justes, tem.lo, tem.hi, tRendus, tJustes)
}

// invPosMarges donne les bornes EXACTES de la loi de position et la marge qui la separe du
// premier candidat parasite. Une fenetre choisie sans connaitre sa marge est un reglage, pas
// une mesure.
func invPosMarges(t *testing.T, obs []invPosObs) {
	t.Helper()
	minV, maxV := 1<<30, -(1 << 30)
	// margeBas : distance entre la position vraie et le candidat parasite le plus proche EN
	// DESSOUS d'elle (le seul qui puisse tromper la strategie « premier »).
	margeBas := 1 << 30
	var pireFilm string
	sansParasiteBas := 0
	for _, o := range obs {
		if o.i22Rel < minV {
			minV = o.i22Rel
		}
		if o.i22Rel > maxV {
			maxV = o.i22Rel
		}
		best := -(1 << 30)
		for _, c := range o.winCands {
			if c.bit < o.i22Rel && c.bit > best {
				best = c.bit
			}
		}
		if best == -(1 << 30) {
			sansParasiteBas++
			continue
		}
		if d := o.i22Rel - best; d < margeBas {
			margeBas, pireFilm = d, fmt.Sprintf("%s/chunk%d/slot%d", o.film, o.chunk, o.slot)
		}
	}
	t.Logf("loi de position : offset i22-debutAmmo dans [%d,%d] (largeur %d bits)",
		minV, maxV, maxV-minV+1)
	t.Logf("marge basse minimale (position vraie - candidat parasite en dessous) : %d bits (%s) ; %d records sans aucun parasite en dessous dans la fenetre large",
		margeBas, pireFilm, sansParasiteBas)
}

func invPosCouverture(h invPosHisto, n int) float64 {
	var vals []int
	total := 0
	for _, v := range h {
		vals = append(vals, v)
		total += v
	}
	sort.Sort(sort.Reverse(sort.IntSlice(vals)))
	s := 0
	for i := 0; i < n && i < len(vals); i++ {
		s += vals[i]
	}
	if total == 0 {
		return 0
	}
	return 100 * float64(s) / float64(total)
}

// TestInventaireApresR2b — LA TAXONOMIE DU 2026-08-24, REJOUEE PAR LE DECODEUR DE PRODUCTION.
//
// Le tableau de MESURE_TROUS_INVENTAIRE_2026-08-24.md tenait le compte des records ARMES SANS
// GRENADE : 4 278 sur 6 721, soit 63,7 %. Ce test-ci rend le MEME chiffre APRES R2b, et il le
// rend en passant par `keyframeInventories`, pas par une copie des regles : une mesure qui
// reimplemente ce qu'elle mesure ne mesure rien.
//
// MEME CORPUS, MEMES VARIABLES D'ENVIRONNEMENT que TestPositionI22.
func TestInventaireApresR2b(t *testing.T) {
	films := invTrousCorpus(t)
	if len(films) == 0 {
		t.Skipf("%s ou %s est requis — mesure sautee", invTrousFilmsEnv, invTrousCacheEnv)
	}
	var tot invPosBilan
	for _, dir := range films {
		b := invPosBilanFilm(dir)
		b.nom = invPosBase(dir)
		b.log(t)
		tot.fusion(b)
	}
	tot.nom = "TOTAL"
	tot.log(t)
}

// invPosBilan compte, par film, ce que le decodeur de production rend.
type invPosBilan struct {
	nom                          string
	records, armes, vides        int
	grenAncre, grenPos, sansGren int
	// armesSansGren est LE chiffre du rapport du 24/08 : munitions lues, aucune grenade.
	armesSansGren int
	// grenNulles compte les lectures dont les quatre compteurs sont a zero — « aucune grenade
	// portee », une MESURE que R2a rejetait.
	grenNulles int
}

func (b *invPosBilan) fusion(o invPosBilan) {
	b.records += o.records
	b.armes += o.armes
	b.vides += o.vides
	b.grenAncre += o.grenAncre
	b.grenPos += o.grenPos
	b.sansGren += o.sansGren
	b.armesSansGren += o.armesSansGren
	b.grenNulles += o.grenNulles
}

func (b *invPosBilan) log(t *testing.T) {
	t.Helper()
	pc := func(n int) float64 {
		if b.records == 0 {
			return 0
		}
		return 100 * float64(n) / float64(b.records)
	}
	t.Logf("%-9s records %5d | armes %5d | grenades: ancre %5d + position %5d = %5d (%.1f%%) | "+
		"ARMES SANS GRENADE %4d (%.1f%%) | vides %4d | compteurs tous nuls %4d",
		b.nom, b.records, b.armes, b.grenAncre, b.grenPos, b.grenAncre+b.grenPos,
		pc(b.grenAncre+b.grenPos), b.armesSansGren, pc(b.armesSansGren), b.vides, b.grenNulles)
}

func invPosBilanFilm(dir string) invPosBilan {
	release := filmdec.LockProcessDecode()
	defer release()
	known := loadoutFamilies()
	var b invPosBilan
	n := filmdec.CountFilmChunks(dir)
	for ch := 1; ch <= n; ch++ {
		chunk, err := filmdec.ReadFilmChunk(dir, ch)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			for _, inv := range keyframeInventories(p.Payload(chunk), known, DefaultGrenadeMax) {
				b.compter(inv)
			}
		}
	}
	return b
}

func (b *invPosBilan) compter(inv KeyframeInventory) {
	b.records++
	if inv.AmmoRead {
		b.armes++
	}
	switch {
	case !inv.GrenadesRead:
		b.sansGren++
		if inv.AmmoRead {
			b.armesSansGren++
		}
	case inv.GrenadesByPosition:
		b.grenPos++
	default:
		b.grenAncre++
	}
	if inv.GrenadesRead && inv.Grenades == [invGrenadeSlots]uint32{} {
		b.grenNulles++
	}
	if invReadingIsEmpty(inv) {
		b.vides++
	}
}

// TestOracleTypesPortesEtLances — L'ORACLE INDEPENDANT DE R2b.
//
// LE PRINCIPE. Les compteurs i22 sont lus aux IMAGES-CLES ; les lancers de grenade sont decodes
// dans les PAQUETS DELTA, par un tout autre chemin (filmdec.ScanFilmGrenadeThrows), et ils
// portent le TYPE lance. Les deux canaux ne partagent aucun bit. Si R2b lisait du bruit, la
// repartition des types PORTES n'aurait aucune raison de suivre celle des types LANCES.
//
// CE QUI EST COMPTE, par couple (film, rang de grenade) : le type est-il LANCE au moins une
// fois dans le film, est-il PORTE au moins une fois (un compteur non nul) ? Quatre cases. Le
// couple « lance mais jamais porte » est le seul qui accuse R2b : un joueur qui lance une
// Dynamo en portait une.
//
// LE TEMOIN est la MEME table calculee en decalant le rang porte de 1 (modulo 4) : il mesure ce
// que rendrait un appariement au hasard sur la meme repartition marginale.
func TestOracleTypesPortesEtLances(t *testing.T) {
	films := invTrousCorpus(t)
	if len(films) == 0 {
		t.Skipf("%s ou %s est requis — mesure sautee", invTrousFilmsEnv, invTrousCacheEnv)
	}
	var lanceEtPorte, lanceNonPorte, porteNonLance, niUniNi int
	var temLanceEtPorte, temLanceNonPorte int
	for _, dir := range films {
		porte, lance, err := invPosTypes(dir)
		if err != nil {
			t.Logf("%s : lancers illisibles (%v) — film ecarte de l'oracle", invPosBase(dir), err)
			continue
		}
		for r := 0; r < invGrenadeSlots; r++ {
			switch {
			case lance[r] && porte[r]:
				lanceEtPorte++
			case lance[r] && !porte[r]:
				lanceNonPorte++
			case !lance[r] && porte[r]:
				porteNonLance++
			default:
				niUniNi++
			}
			if lance[r] && porte[(r+1)%invGrenadeSlots] {
				temLanceEtPorte++
			} else if lance[r] {
				temLanceNonPorte++
			}
		}
		t.Logf("%-9s portes %v · lances %v", invPosBase(dir), porte, lance)
	}
	t.Logf("SIGNAL : lance ET porte %d · lance mais JAMAIS porte %d · porte sans lancer %d · ni l'un ni l'autre %d",
		lanceEtPorte, lanceNonPorte, porteNonLance, niUniNi)
	t.Logf("TEMOIN (rang porte decale de 1) : lance ET porte %d · lance mais jamais porte %d",
		temLanceEtPorte, temLanceNonPorte)
}

// invPosTypes rend, pour un film, les rangs de grenade PORTES (compteur i22 non nul, decodeur de
// production) et les rangs LANCES (canal delta, totalement disjoint).
func invPosTypes(dir string) (porte, lance [invGrenadeSlots]bool, err error) {
	release := filmdec.LockProcessDecode()
	known := loadoutFamilies()
	n := filmdec.CountFilmChunks(dir)
	for ch := 1; ch <= n; ch++ {
		chunk, e := filmdec.ReadFilmChunk(dir, ch)
		if e != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			for _, inv := range keyframeInventories(p.Payload(chunk), known, DefaultGrenadeMax) {
				if !inv.GrenadesRead {
					continue
				}
				for r, v := range inv.Grenades {
					if v > 0 {
						porte[r] = true
					}
				}
			}
		}
	}
	release()
	throws, err := filmdec.ScanFilmGrenadeThrows(dir)
	if err != nil {
		return porte, lance, err
	}
	for _, g := range throws {
		if r, ok := g.Rank(); ok && r >= 0 && r < invGrenadeSlots {
			lance[r] = true
		}
	}
	return porte, lance, nil
}
