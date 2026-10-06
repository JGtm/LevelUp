package grammar

// ecrivain_invariants.go — LES REGLES DE L ECRIVAIN DU FILM QU UNE LECTURE FERMEE NE PEUT PAS
// CONTREDIRE.
//
// Un paquet delta est FERME quand sa vue C se lit jusqu a son terminateur et que le reste du
// payload fait 0 a 7 bits nuls ([vueCFermee], la propriete de `FUN_14299d2c8`). Cette condition est
// necessaire, pas suffisante : la vue C se resynchronise d elle-meme depuis une position fausse. Un
// paquet n est donc ferme que si, EN PLUS, aucune regle de l ecrivain n est contredite par ce que
// la marche a lu. Chaque regle ci-dessous cite la fonction de l ecrivain qui la fonde ; une regle
// sans ecrivain n entre pas ici.
//
//	sortie de vue B par rejet   `FUN_142f2e174` n ecrit un DELTA que pour une entite a l etat 3, que
//	                            seul un NEW ecrit et acquitte pose (`FUN_142f2cee0`, `FUN_142f2f8f0`) ;
//	                            la vue du film s acquitte elle-meme a chaque paquet (`FUN_142f2cc78`).
//	                            Un en-tete DELTA rejete (slot sans datum, ou datum d une autre vue)
//	                            n est donc jamais un record lu a sa place.
//	ordre de la vue B           `FUN_14076b9c8` concatene les NEW, puis les DELTA, puis les DEL ;
//	                            `FUN_142f2e174` parcourt les slots par index croissant, un genre par
//	                            entite.
//	masque                      `FUN_142e2da44` : aucun bit au-dela du nombre de composants du
//	                            descripteur ; au plus sept composants en epars, index croissants ;
//	                            plus de sept en dense.
//	vue C                       `FUN_142f2c3b0` concatene les tampons des 32 joueurs par index
//	                            croissant ; `FUN_14076b0e8` n y ecrit que des entrees kind 0, une par
//	                            joueur, bit d en-tete `FUN_1406cdc04` a 0 ; `FUN_1406d5bf4` n ecrit
//	                            jamais le code analogique 63.
//
// Le mot de 32 bits d un DEL (`FUN_142f304a8` : non nul pour l archetype 0x10 seul) n est pas une
// regle d ici : il se juge contre l archetype du DATUM, que le paquet ne porte pas et que le monde
// hors ligne suppose.

import "math/bits"

// InvariantEcrivain nomme la premiere regle de l ecrivain qu une lecture contredit.
type InvariantEcrivain uint8

// Les regles, dans l ordre ou la marche les rencontre.
const (
	// InvariantAucun : aucune regle contredite.
	InvariantAucun InvariantEcrivain = iota
	// InvariantSortieParRejet : la vue B s est arretee sur un en-tete DELTA rejete.
	InvariantSortieParRejet
	// InvariantOrdreVueB : un record de la vue B hors de l ordre NEW*, DELTA*, DEL*, ou un slot
	// non strictement croissant dans son groupe.
	InvariantOrdreVueB
	// InvariantMasqueHorsArchetype : un bit de masque au-dela du dernier composant de l archetype.
	InvariantMasqueHorsArchetype
	// InvariantMasqueDenseCourt : un masque dense qui annonce au plus sept composants.
	InvariantMasqueDenseCourt
	// InvariantMasqueEparsNonCroissant : un masque epars dont les index ne croissent pas.
	InvariantMasqueEparsNonCroissant
	// InvariantVueCKind : une entree de vue C d un kind autre que 0.
	InvariantVueCKind
	// InvariantVueCTropDEntrees : plus d entrees de vue C que de joueurs.
	InvariantVueCTropDEntrees
	// InvariantVueCIndex : deux entrees de vue C d index non strictement croissants.
	InvariantVueCIndex
	// InvariantVueCEnTete : une entree de vue C dont le bit d en-tete `FUN_1406cdc04` vaut 1.
	InvariantVueCEnTete
	// InvariantVueCCodeAnalogique : un scalaire analogique au code 63.
	InvariantVueCCodeAnalogique
	// NombreDInvariants est le nombre de valeurs.
	NombreDInvariants = 11
)

// String rend le nom de la regle, tel que la carte de fermeture l ecrit.
func (v InvariantEcrivain) String() string {
	switch v {
	case InvariantAucun:
		return "aucun"
	case InvariantSortieParRejet:
		return "vue B : sortie par rejet"
	case InvariantOrdreVueB:
		return "ecrivain : ordre de la vue B"
	case InvariantMasqueHorsArchetype:
		return "ecrivain : masque au-dela de l archetype"
	case InvariantMasqueDenseCourt:
		return "ecrivain : masque dense de sept composants au plus"
	case InvariantMasqueEparsNonCroissant:
		return "ecrivain : masque epars non croissant"
	case InvariantVueCKind:
		return "ecrivain : vue C kind non nul"
	case InvariantVueCTropDEntrees:
		return "ecrivain : vue C au-dela de 32 entrees"
	case InvariantVueCIndex:
		return "ecrivain : vue C index non croissants"
	case InvariantVueCEnTete:
		return "ecrivain : vue C en-tete cdc04 pose"
	case InvariantVueCCodeAnalogique:
		return "ecrivain : vue C code analogique 63"
	}
	return "ecrivain : regle inconnue"
}

// composantsEparsMax : `FUN_142e2da44` ecrit le masque en epars tant qu il annonce au plus sept
// composants, en dense au-dela.
const composantsEparsMax = 7

// entreesVueCMax : `FUN_142f2c3b0` concatene les tampons de 32 joueurs, une entree au plus chacun.
const entreesVueCMax = 32

// codeAnalogiqueInterdit : `FUN_1406d5bf4` borne le scalaire a [0, 0x3e] ; 0x3f n est jamais ecrit.
const codeAnalogiqueInterdit = 0x3f

// lireMasque lit le masque de composants d un record (`FUN_1406d7610`, reciproque de
// `FUN_142e2da44`) et dit si l ecrivain peut l ecrire : R(1) ; 0 -> R(3) puis autant de R(6),
// 1 -> R(64). Le bit au-dela de l archetype se juge a la traversee, qui connait l archetype
// ([traverseComponentLoopFrom]).
func lireMasque(br *Lecteur) (uint64, InvariantEcrivain) {
	if br.ReadBit() {
		m := br.ReadBits(64)
		if bits.OnesCount64(m) <= composantsEparsMax {
			return m, InvariantMasqueDenseCourt
		}
		return m, InvariantAucun
	}
	count := br.ReadBits(3)
	var mask uint64
	viole, precedent := InvariantAucun, -1
	for range count {
		idx := int(br.ReadBits(6)) //nolint:gosec // R(6) : 0..63
		if idx <= precedent {
			viole = InvariantMasqueEparsNonCroissant
		}
		precedent = idx
		mask |= uint64(1) << uint(idx)
	}
	return mask, viole
}

// masqueHorsArchetype dit si un masque pose un bit au-dela des `n` composants de l archetype.
func masqueHorsArchetype(mask uint64, n int) bool {
	return n < 64 && mask>>uint(n) != 0 //nolint:gosec // n >= 0
}

// jugeEcrivain releve les regles qu un paquet contredit. `tous` faux : il s arrete a la premiere,
// comme la marche ; vrai : il les releve toutes (la carte de fermeture).
type jugeEcrivain struct {
	tous bool
	// premiere : la premiere regle contredite, dans l ordre de [jugerLePaquet].
	premiere InvariantEcrivain
	// ensemble : un bit par regle contredite (bit `v` pour [InvariantEcrivain] `v`).
	ensemble uint32
}

// noter releve une regle (rien pour [InvariantAucun]) et dit si le juge doit continuer.
func (j *jugeEcrivain) noter(v InvariantEcrivain) bool {
	if v == InvariantAucun {
		return true
	}
	if j.premiere == InvariantAucun {
		j.premiere = v
	}
	j.ensemble |= uint32(1) << uint(v)
	return j.tous
}

// jugerLePaquet juge la lecture d un paquet : la sortie de la vue B, puis ses records dans l ordre
// du flux, puis la vue C.
func jugerLePaquet(recs []FrameRecord, rejet bool, c FluxVueC, tous bool) jugeEcrivain {
	j := jugeEcrivain{tous: tous}
	if rejet && !j.noter(InvariantSortieParRejet) {
		return j
	}
	if !j.jugerLaVueB(recs) {
		return j
	}
	j.jugerLaVueC(c)
	return j
}

// phaseDeRecord rend le rang d un type de record dans l ordre de l ecrivain (NEW, DELTA, DEL).
func phaseDeRecord(typ int) (int, bool) {
	switch typ {
	case recNew:
		return 0, true
	case recDelta:
		return 1, true
	case recDel:
		return 2, true
	}
	return 0, false
}

// ordreDeLaVueB suit la loi d ecriture de la vue B, record par record : `FUN_142f2e174` parcourt
// la table de vue par index croissant et range chaque entree dans le sous-ecrivain de son genre,
// que `FUN_14076b9c8` concatene dans l ordre NEW (+0x1b090), DELTA (+0x1b240), DEL (+0x1b168).
// D ou NEW*, DELTA*, DEL*, slots strictement croissants dans chaque groupe. Le juge de l ecrivain
// ([jugerLaVueB]) et la chaine de tete ([chaineJusqua]) la tiennent par ce seul type.
type ordreDeLaVueB struct {
	phase   int
	dernier [3]int64
}

// nouvelOrdreDeLaVueB rend l ordre d une vue B dont aucun record n est encore lu.
func nouvelOrdreDeLaVueB() ordreDeLaVueB {
	return ordreDeLaVueB{dernier: [3]int64{-1, -1, -1}}
}

// suivre enregistre un record de rang `ph` ([phaseDeRecord]) sur `slot`, et dit s il respecte la
// loi d ecriture apres les records deja suivis.
func (o *ordreDeLaVueB) suivre(ph int, slot uint32) bool {
	respecte := ph >= o.phase && int64(slot) > o.dernier[ph]
	o.phase, o.dernier[ph] = max(o.phase, ph), int64(slot)
	return respecte
}

// jugerLaVueB juge les records de la vue B dans l ordre du flux ; faux : le juge s arrete.
func (j *jugeEcrivain) jugerLaVueB(recs []FrameRecord) bool {
	ordre := nouvelOrdreDeLaVueB()
	for _, r := range recs {
		ph, ok := phaseDeRecord(r.Type)
		if !ok {
			continue
		}
		if !ordre.suivre(ph, r.Slot) && !j.noter(InvariantOrdreVueB) {
			return false
		}
		if !j.noter(r.Trace.MasqueNonEcrit) {
			return false
		}
	}
	return true
}

// jugerLaVueC juge le flux de la vue C.
func (j *jugeEcrivain) jugerLaVueC(c FluxVueC) { //nolint:gocyclo // un test par invariant d ecriture de la vue C (kind de controle, nombre d entrees, index croissants, champ d en-tete absent, code analogique interdit), chacun conjoint a l arret du juge : la complexite compte les invariants
	for _, k := range c.Kinds {
		if k != kindVueCControle && !j.noter(InvariantVueCKind) {
			return
		}
	}
	if len(c.Entrees) > entreesVueCMax && !j.noter(InvariantVueCTropDEntrees) {
		return
	}
	for i, e := range c.Entrees {
		if i > 0 && e.Index <= c.Entrees[i-1].Index && !j.noter(InvariantVueCIndex) {
			return
		}
		if e.Champs.Cdc04 != ChampDeControleAbsent && !j.noter(InvariantVueCEnTete) {
			return
		}
		analogique63 := e.Champs.Analogique[0] == codeAnalogiqueInterdit ||
			e.Champs.Analogique[1] == codeAnalogiqueInterdit
		if e.Bloc && analogique63 && !j.noter(InvariantVueCCodeAnalogique) {
			return
		}
	}
}

// verdictDeVueC rend le verdict d un paquet dont la vue B s est terminee et dont la vue C vient
// d etre lue jusqu a `curseur`. La sortie par rejet se dit toujours ; les autres regles ne se
// jugent que sur un paquet ferme au bit pres, le seul dont la lecture vaut quelque chose.
func verdictDeVueC(pay []byte, curseur int, c FluxVueC, recs []FrameRecord, rejet bool) LectureVueC {
	l := LectureVueC{Atteinte: true, Arret: c.Arret, FermeeAuBit: c.Porte && vueCFermee(pay, curseur)}
	switch {
	case rejet:
		l.Invariant = InvariantSortieParRejet
	case l.FermeeAuBit:
		l.Invariant = jugerLePaquet(recs, false, c, false).premiere
	}
	l.Fermee = l.FermeeAuBit && l.Invariant == InvariantAucun
	if l.Fermee {
		l.Entrees = c.Entrees
	}
	return l
}
