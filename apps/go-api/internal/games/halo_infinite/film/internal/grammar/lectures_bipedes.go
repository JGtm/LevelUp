package grammar

// lectures_bipedes.go — LES LECTURES DE COMPOSANT DES RECORDS BIPEDES DELTA : LA MARCHE DES TRAMES
// D ABORD, L ANCRAGE ENSUITE (plan de l etape 2 de la representation intermediaire, 2.7.b ; ADR 0037
// IR-6 ; option A de l utilisateur du 2026-10-04).
//
// # LES HUIT LECTEURS
//
// Les charges, les impulsions, les rangs, le camouflage, le grappin, l arme portee, les deltas
// d inventaire et l equipement lisent chacun quelques composants des records bipedes delta. Ils
// lisent [lecturesBipedes] : ce que la marche des trames a lu de ces records, puis ce que l ancrage
// d en-tete bipede a recupere DERRIERE elle.
//
// # D OU VIENT CHAQUE RECORD
//
// LA MARCHE : ses deserialiseurs publient pendant la marche ; le canal des lectures bipedes
// ([canalDesLecturesBipedes]) recueille chaque publication avec la position du lecteur de la
// marche, et l attribue, a la cloture de la trame, au composant du record retenu dont l etendue la
// porte. Une publication qu aucun record retenu ne porte (le corps d un NEW qui desynchronise, que
// la marche ne garde pas) n est attribuee a personne, et se compte.
//
// L ANCRAGE, DERRIERE : il ne lit qu un record dont la marche n a lu aucun record du meme slot
// dans le paquet, et seulement dans une trame qu elle n a pas fermee — une trame fermee prouve sa
// liste entiere, et un record ancre que la marche n y a pas lu est une fausse ancre. Un tel record
// est marque recupere ([recordBipedeLu.Recupere]) et compte au rapport des replis du contexte
// (`repli_ancrage_bipede_apres_la_marche`, ordre « apres la lecture »).
//
// # CE QU UN LECTEUR EN FAIT
//
// Il rejoue les publications d un record sur ses propres crochets, composant par composant
// ([recordBipedeLu.parcourir], [recordBipedeLu.parcourirJusqua]) : le contrat de la marche de
// record ([walkRecordComponents]), sans relire un bit. Sa logique ne change pas, sa source si.

import (
	"cmp"
	"math/bits"
	"slices"
)

// recordBipedeLu est UN record bipede delta lu : par la marche des trames, ou recupere par
// l ancrage derriere elle.
type recordBipedeLu struct {
	// Slot et Gen nomment la vie du corps : le slot et la generation du handle du record.
	Slot, Gen uint32
	// Chunk et Packet situent le record dans le film. Le payload ne s y relit plus.
	Chunk  int
	Packet FilmPacket
	// I0 est le bit du composant i0 quand le record l annonce, sinon celui de son en-tete : la cle
	// qui ordonne les records d un paquet.
	I0 int
	// masque : les index que le masque annonce ; atteints : ceux dont le composant a ete traverse,
	// i0 excepte. Un masque de record ne depasse pas 64 index.
	masque, atteints uint64
	// arret : le composant sur lequel la lecture s est arretee APRES qu il a publie (un corps que
	// son deserialiseur ne porte pas jusqu au bout), -1 sans arret. Ses publications se rejouent ;
	// il n est pas traverse.
	arret int
	// appels : les publications du record, dans l ordre ou ses composants ont ete lus.
	appels []appelDeComposant
	// Recupere : le record vient de l ancrage, derriere la marche.
	Recupere bool
}

// appelDeComposant est UNE publication d un deserialiseur pendant la lecture d un record : le
// composant qui l a faite, et de quoi la rejouer sur une observation.
type appelDeComposant struct {
	composant int
	rejouer   func(*Observation)
}

// annonce dit si le masque du record annonce le composant `id`.
func (r *recordBipedeLu) annonce(id int) bool {
	return id >= 0 && id < 64 && r.masque>>uint(id)&1 == 1
}

// parcourir rejoue sur `obs`, composant par composant, ce que la lecture du record a publie, et
// appelle `visit` APRES chaque composant traverse, dans l ordre des index ; `visit` rend faux
// pour arreter. Le composant d arret publie sans etre visite. C est le contrat de
// [walkRecordComponents] : un deserialiseur publie pendant qu il lit, et la marche ne visite que
// ce qu il a porte jusqu au bout.
func (r *recordBipedeLu) parcourir(obs *Observation, visit func(id int) bool) {
	k := 0
	for m := r.atteints; m != 0; m &= m - 1 {
		id := bits.TrailingZeros64(m)
		k = r.rejouer(obs, k, id)
		if !visit(id) {
			return
		}
	}
	if r.arret >= 0 {
		r.rejouer(obs, k, r.arret)
	}
}

// rejouer rejoue sur `obs` les publications du composant `id`, a partir du rang `k`, et rend le
// rang qui suit.
func (r *recordBipedeLu) rejouer(obs *Observation, k, id int) int {
	for ; k < len(r.appels) && r.appels[k].composant < id; k++ {
	}
	for ; k < len(r.appels) && r.appels[k].composant == id; k++ {
		r.appels[k].rejouer(obs)
	}
	return k
}

// parcourirJusqua rejoue jusqu au composant `cible` compris et dit s il a ete traverse : le
// contrat de la marche de record arretee sur sa cible.
func (r *recordBipedeLu) parcourirJusqua(obs *Observation, cible int) bool {
	atteint := false
	r.parcourir(obs, func(id int) bool {
		atteint = id == cible
		return !atteint
	})
	return atteint
}

// lecturesBipedes est ce que la marche des trames, puis l ancrage derriere elle, ont lu des records
// bipedes delta d un film : ceux qui annoncent un composant des huit lecteurs, dans l ordre du flux.
type lecturesBipedes struct {
	records []recordBipedeLu
	// examines : records bipedes delta lus, meme ceux qui n annoncent rien des huit lecteurs ;
	// recuperes : ceux que l ancrage a rendus ; horsRecord : publications de la marche qu aucun
	// record retenu ne porte.
	examines, recuperes, horsRecord int
}

// lecturesBipedes rend les lectures bipedes du film, faites une fois par contexte : par la
// distribution de la cuisson, qui les recueille avec ses autres canaux, ou par une distribution
// faite ici pour elles seules.
func (c *FilmContext) lecturesBipedes() (*lecturesBipedes, error) {
	if c.recup.lectures == nil {
		if err := Distribuer(c, nouveauCanalDesLecturesBipedes(c)); err != nil {
			return nil, err
		}
	}
	return c.recup.lectures, nil
}

// composantsDesLecteursBipedes rend le masque des index de l archetype bipede que les huit lecteurs
// lisent, resolus par leurs propres regles : un record qui n en annonce aucun ne concerne
// personne.
func composantsDesLecteursBipedes(arch Archetype) uint64 {
	var m uint64
	poser := func(id int) {
		if id >= 0 && id < 64 {
			m |= 1 << uint(id)
		}
	}
	poser(componentIndexOfAny(arch, abilityEnergyName, abilityEnergyNameAlt))
	poser(componentIndexOfAny(arch, abilityPredictedName, abilityPredictedNameAlt))
	poser(componentIndexOfAny(arch, grappleComponentName, grappleComponentNameAlt))
	poser(componentIndexOfAny(arch, camoComponentName))
	poser(indexDeLEquipementDUnite(arch))
	poser(i48Index)
	for id := range weaponEmplacements(arch) {
		poser(id)
	}
	for id := range invDeltaRoles(arch) {
		poser(id)
	}
	return m
}

// masqueDesIndex rend le masque des index `idx` (moins de 64).
func masqueDesIndex(idx []int) uint64 {
	var m uint64
	for _, id := range idx {
		if id >= 0 && id < 64 {
			m |= 1 << uint(id)
		}
	}
	return m
}

// rangerDansLeFlux trie les records dans l ordre du flux : rang du chunk dans le film, rang du
// paquet, puis position du record.
func rangerDansLeFlux(recs []recordBipedeLu, chunks []int) {
	rang := make(map[int]int, len(chunks))
	for i, c := range chunks {
		rang[c] = i
	}
	slices.SortStableFunc(recs, func(a, b recordBipedeLu) int {
		return cmp.Or(cmp.Compare(rang[a.Chunk], rang[b.Chunk]), cmp.Compare(a.Packet.Index, b.Packet.Index),
			cmp.Compare(a.I0, b.I0))
	})
}
