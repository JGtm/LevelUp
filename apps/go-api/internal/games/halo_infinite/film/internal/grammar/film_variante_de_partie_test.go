package grammar

// film_variante_de_partie_test.go — les vecteurs du lot VA, etape V3 : la marche du corps de
// `chunk_00` jusqu a la table des joueurs et ce que sa variante de partie declare. Les vecteurs sont
// ecrits d apres les ECRIVAINS du jeu (`FUN_1407ec560`, et pour Bond `FUN_140ac755c`,
// `FUN_140ac7668`, `FUN_140ac75e8`, `FUN_140ac7430`, `FUN_1411b3740`, `FUN_1424d6668`,
// `FUN_140d1a268`), avec leurs immediats, jamais les constantes du portage.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// bondEcrit est un message Bond CompactBinary v2, en octets, tel que les ecrivains du jeu le forment.
type bondEcrit []byte

// entier ecrit un entier variable (`FUN_140ac7668`).
func (b *bondEcrit) entier(v uint64) {
	for v >= 0x80 {
		*b = append(*b, byte(v)|0x80)
		v >>= 7
	}
	*b = append(*b, byte(v))
}

// enTete ecrit un en-tete de champ (`FUN_140ac75e8`).
func (b *bondEcrit) enTete(id, ty int) {
	switch {
	case id < 6:
		*b = append(*b, byte(ty|id<<5))
	case id < 0x100:
		*b = append(*b, byte(ty|0xC0), byte(id))
	default:
		*b = append(*b, byte(ty|0xE0), byte(id), byte(id>>8))
	}
}

// structure ecrit une structure : sa longueur (`FUN_140ac755c`), ses champs, son octet de fin 0.
func (b *bondEcrit) structure(champs func(*bondEcrit)) {
	var c bondEcrit
	if champs != nil {
		champs(&c)
	}
	c = append(c, 0)
	b.entier(uint64(len(c)))
	*b = append(*b, c...)
}

func (b *bondEcrit) champStructure(id int, champs func(*bondEcrit)) {
	b.enTete(id, 0xA)
	b.structure(champs)
}

// finDeBase ecrit la fin d une structure de base (`FUN_1411b3740`, drapeau de base pose : l octet 1).
// Une structure de base n a pas de longueur (`FUN_140ac755c` ne l ecrit que hors base) : ses champs
// precedent cet octet dans la structure qui en derive.
func (b *bondEcrit) finDeBase() { *b = append(*b, 1) }

func (b *bondEcrit) booleen(id int, v bool) {
	b.enTete(id, 2)
	if v {
		*b = append(*b, 1)
	} else {
		*b = append(*b, 0)
	}
}

func (b *bondEcrit) entier32(id int, v int32) {
	b.enTete(id, 0x10)
	b.entier(uint64(uint32(v<<1) ^ uint32(v>>31))) //nolint:gosec // zigzag de FUN_140d1a268
}

// liste ecrit une liste de structures (`FUN_140ac7430`, n < 7).
func (b *bondEcrit) liste(id int, elements ...func(*bondEcrit)) {
	b.enTete(id, 0xB)
	*b = append(*b, byte(0xA|(len(elements)+1)<<5))
	for _, e := range elements {
		b.structure(e)
	}
}

// varianteEcrite decrit la variante d un vecteur.
type varianteEcrite struct {
	moteur        int32
	killcam, potg bool
}

// bondDeLaVariante ecrit la variante comme `FUN_140b85504` : la structure de tete, dont le champ 1 est
// une liste d une structure de variante (`FUN_141086fec`), avec son en-tete, ses reglages generaux
// (`FUN_141025dd0`) et des champs que la marche enjambe.
func bondDeLaVariante(v varianteEcrite) bondEcrit {
	var b bondEcrit
	b.structure(func(t *bondEcrit) {
		t.liste(0) // champ 0 : liste vide
		t.liste(1, func(s *bondEcrit) {
			s.champStructure(0, func(e *bondEcrit) {
				if v.moteur != 0 { // optionnel : omis a son defaut
					e.entier32(0, v.moteur)
				}
				e.entier32(1, 7) // m_statBucketGameType
				e.champStructure(3, nil)
			})
			s.champStructure(1, func(r *bondEcrit) {
				r.champStructure(0, func(g *bondEcrit) {
					for id := range 5 {
						g.champStructure(id, func(x *bondEcrit) { x.entier32(0, int32(id)) }) //nolint:gosec // id < 5
					}
					g.champStructure(5, func(p *bondEcrit) {
						p.finDeBase() // une base vide : la fin de base precede les champs
						if v.killcam {
							p.booleen(0, true)
						}
						p.entier32(1, 3)
						if v.potg {
							p.booleen(2, true)
						}
						p.booleen(3, true)
						p.booleen(0x200, true) // en-tete 0xE0 : l id 0x200 sur deux octets, poids faible d abord
					})
					g.champStructure(15, nil) // en-tete a id echappe
				})
				r.champStructure(7, nil)
			})
			s.champStructure(2, func(x *bondEcrit) { x.booleen(0, true) })
			s.champStructure(6, nil)
		})
	})
	return b
}

// ecrireOctets ecrit des octets dans le flux de bits, huit bits chacun (`FUN_1406d5f18`).
func (w *bitWriter) ecrireOctets(o []byte) {
	for _, x := range o {
		w.bits(uint64(x), 8)
	}
}

// corpsEcrit decrit le corps d un vecteur.
type corpsEcrit struct {
	variante    *varianteEcrite
	blocOptionn bool
	bondDeTete  bondEcrit
	chaine      []byte
	casser      func(b bondEcrit) bondEcrit
}

// ecrireCorps ecrit le corps comme `FUN_1407ec560`, a partir du bit courant, et rend le bit du premier
// enregistrement de joueur.
func (w *bitWriter) ecrireCorps(c corpsEcrit) int {
	for _, n := range []int{3, 3, 2, 7, 0x40, 0x20, 3, 0x20, 0x20, 0x20} {
		w.bits(1, n)
	}
	w.ecrireOctets(c.bondDeTete)
	w.ecrireOctets(append(append([]byte{}, c.chaine...), 0)) // FUN_1407ebe7c, R9D = 0x80
	w.bits(0, 0x20)
	w.bits(0, 0x20)
	if c.blocOptionn { // FUN_1410bc140 : R(1), puis 0x700 bits
		w.bit(1)
		w.bits(0, 0x40)
		for range (0x700 - 0x40) / 0x40 {
			w.bits(0, 0x40)
		}
	} else {
		w.bit(0)
	}
	for _, n := range []int{1, 1, 2, 1, 0x20, 0x20, 1} {
		w.bits(0, n)
	}
	if c.variante == nil {
		w.bit(0)
	} else {
		w.bit(1)
		b := bondDeLaVariante(*c.variante)
		if c.casser != nil {
			b = c.casser(b)
		}
		w.ecrireOctets(b)
	}
	w.ecrireOctets([]byte("Nom\x00")) // R9D = 0x100
	w.ecrireOctets([]byte{0})         // R9D = 0x100
	w.bits(0, 0x20)                   // + 0xEAA10
	for range 0x6C0 / 0x40 {          // + 0xEAA18, R9D = 0x6C0
		w.bits(0, 0x40)
	}
	return w.n
}

// bondDeTeteDeTest est un premier message Bond a champs de plusieurs types (`FUN_140b857d8`).
func bondDeTeteDeTest() bondEcrit {
	var b bondEcrit
	b.structure(func(s *bondEcrit) {
		s.champStructure(0, func(x *bondEcrit) { x.entier32(0, -5) })
		s.entier32(1, 1234)
		s.booleen(2, true)
	})
	return b
}

// TestLeCorpsSeLitJusquALaTableDesJoueurs : la marche du corps s arrete au premier bit de la table des
// joueurs et rend ce que la variante declare ; un champ optionnel absent vaut son defaut ; une variante
// absente du film n est pas lue ; une fin de structure de base et un en-tete a id sur deux octets (forme
// 0xE0) se lisent. MUTATIONS — killcamEnabled pris au champ 1, playOfTheGameEnabled au champ 3, le
// drapeau inverse, la longueur d une structure ignoree, la fin de base refusee, l id 0xE0 lu poids fort
// d abord : ROUGE.
func TestLeCorpsSeLitJusquALaTableDesJoueurs(t *testing.T) {
	for _, c := range []struct {
		nom    string
		corps  corpsEcrit
		attend profile.VarianteDePartie
	}{
		{"moteur 2, playOfTheGame", corpsEcrit{variante: &varianteEcrite{moteur: 2, potg: true}},
			profile.VarianteDePartie{Lue: true, Presente: true, TypeDeMoteur: 2, PlayOfTheGameEnabled: true}},
		{"moteur 1, killcam, bloc optionnel, chaine", corpsEcrit{variante: &varianteEcrite{moteur: 1, killcam: true},
			blocOptionn: true, chaine: []byte("Slayer")},
			profile.VarianteDePartie{Lue: true, Presente: true, TypeDeMoteur: 1, KillcamEnabled: true}},
		{"moteur au defaut", corpsEcrit{variante: &varianteEcrite{}},
			profile.VarianteDePartie{Lue: true, Presente: true}},
		{"variante absente", corpsEcrit{}, profile.VarianteDePartie{Lue: true}},
	} {
		var w bitWriter
		w.bits(0x15, 5)
		c.corps.bondDeTete = bondDeTeteDeTest()
		fin := w.ecrireCorps(c.corps)
		w.bits(0xffff, 16)
		v, bit := marcherLeCorps(w.buf, 5)
		if v != c.attend || bit != fin {
			t.Errorf("%s : %+v au bit %d, attendu %+v au bit %d", c.nom, v, bit, c.attend, fin)
		}
	}
}

// TestUnCorpsQueLaGrammaireNeLitPasNEstPasLu : une longueur de structure qui ne tombe pas sur son
// octet de fin, un entier variable plus long que ce que l ecrivain forme, une chaine sans octet nul, un
// corps coupe — rien n est rendu. MUTATION — entier variable lu sur dix octets : ROUGE.
func TestUnCorpsQueLaGrammaireNeLitPasNEstPasLu(t *testing.T) {
	for _, c := range []struct {
		nom    string
		corps  corpsEcrit
		couper int
	}{
		{"longueur de la variante + 1", corpsEcrit{variante: &varianteEcrite{moteur: 2},
			casser: func(b bondEcrit) bondEcrit { b[0]++; return append(b, 0) }}, 0},
		{"longueur de la variante sur six octets", corpsEcrit{variante: &varianteEcrite{moteur: 2},
			casser: entierDeTeteSurSixOctets}, 0},
		{"chaine de 0x80 octets sans nul", corpsEcrit{chaine: make([]byte, 0x80)}, 0},
		{"corps coupe avant la table", corpsEcrit{variante: &varianteEcrite{moteur: 2}}, 64},
	} {
		for i := range c.corps.chaine {
			c.corps.chaine[i] = 'A'
		}
		var w bitWriter
		w.bits(0x15, 5)
		c.corps.bondDeTete = bondDeTeteDeTest()
		w.ecrireCorps(c.corps)
		buf := w.buf[:len(w.buf)-c.couper]
		if v, _ := marcherLeCorps(buf, 5); v != (profile.VarianteDePartie{}) {
			t.Errorf("%s : %+v, attendu rien de lu", c.nom, v)
		}
	}
}

// TestLaVarianteDesBobines : sur les sept bobines par build, la marche du corps tombe sur le premier
// enregistrement de la table des joueurs que [ReadPlayerTable] trouve par son propre balayage, et la
// variante declare m_gameEngineType 2, killcamEnabled faux, playOfTheGameEnabled vrai sauf sur
// `a521164d` (HI_1_4_1). Mesure de la sonde V3 sur les 1 657 films du cache a section
// d identification : 1 656 marches fermees ainsi (le dernier, HI_1_5_1, n a pas de table des joueurs
// lisible), `TypeDeMoteur` 2 partout.
func TestLaVarianteDesBobines(t *testing.T) {
	for _, b := range bobinesIdentite() {
		reg := bobineChunk00(t, b.film)
		id, err := ReadFilmIdentity(reg)
		if err != nil {
			t.Fatalf("%s : %v", b.film, err)
		}
		_, rep, err := ReadPlayerTable(reg, id)
		if err != nil {
			t.Fatalf("%s : table des joueurs : %v", b.film, err)
		}
		v, fin := marcherLeCorps(reg, id.BodyBit)
		attendu := profile.VarianteDePartie{Lue: true, Presente: true, TypeDeMoteur: 2,
			PlayOfTheGameEnabled: b.film != "a521164d"}
		if v != attendu || id.Variante != v || fin != rep.FirstRecordBit {
			t.Errorf("%s (%s) : %+v (identite %+v), fin %d ; attendu %+v, fin %d", b.film, b.build, v,
				id.Variante, fin, attendu, rep.FirstRecordBit)
		}
	}
}

// entierDeTeteSurSixOctets reecrit l entier variable de tete de b (la longueur de la structure) sur
// six octets, de meme valeur : une forme que `FUN_140ac7668` n ecrit pas, son entier etant un `uint`
// de 32 bits (cinq octets au plus).
func entierDeTeteSurSixOctets(b bondEcrit) bondEcrit {
	var n uint64
	k := 0
	for ; b[k]&0x80 != 0; k++ {
		n |= uint64(b[k]&0x7F) << (7 * k)
	}
	n |= uint64(b[k]) << (7 * k)
	r := bondEcrit{}
	for i := range 6 {
		o := byte(n>>(7*i)) & 0x7F
		if i < 5 {
			o |= 0x80
		}
		r = append(r, o)
	}
	return append(r, b[k+1:]...)
}
