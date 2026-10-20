package grammar

import "levelup/go-api/internal/games/halo_infinite/film/internal/profile"

// film_variante_de_partie.go — LE CORPS DE `chunk_00` LU JUSQU A LA TABLE DES JOUEURS, ET CE QUE SA
// VARIANTE DE PARTIE DECLARE (lot VA de la campagne de grammaire, etape V3).
//
// # CE QUE LE JEU LIT
//
// Releve Ghidra `HaloInfinite.exe` HI_1_13_0, base 0x140000000, lecture seule
// (`.ai/V7.5/film_re/campagne_grammaire_2026-10-01/va_ghidra/`). Le lecteur du corps est
// `FUN_1407ee138` (appele par `FUN_14299ac50` sur `film + 0xCE690`), l ecrivain `FUN_1407ec560`.
// Le corps est la structure des OPTIONS DE PARTIE (pas `0x1134F0`), et il se lit dans cet ordre :
//
//	R(3) R(3) R(2) R(7) R(64) R(32) R(3) R(32) R(32) R(32)     options + 0x0 .. + 0xE9518
//	message Bond                                              `FUN_140ee5f40` (+ 0xE951C)
//	chaine de 0x80 octets au plus                             `FUN_1407cbc24` (+ 0xEA694)
//	R(32) R(32)                                               + 0xEA714, + 0xEA718
//	R(1) ; si 1 : R(0x700)                                    `FUN_140ee5ef8` (+ 0xEA72C)
//	R(1) R(1) R(2) R(1) R(32) R(32) R(1)                      + 0xEA719 .. + 0xEA80C
//	R(1) ; si 1 : message Bond de la variante                 `FUN_140b3a118` (+ 0x28)
//	deux chaines de 0x100 octets au plus, R(32), R(0x6C0)     + 0xEA810 .. + 0xEAA18
//	puis les 32 enregistrements de joueur                     (player_table.go)
//
// Une chaine est lue octet par octet jusqu a l octet nul compris, ou jusqu a sa longueur maximale.
//
// # LES MESSAGES BOND
//
// Les deux messages sont du Microsoft Bond CompactBinary de version 2, ecrit octet par octet dans le
// flux de bits (`FUN_140ac78d0` -> `FUN_1406d5f18`, huit bits par octet). Lus chez l ecrivain :
//
//	debut de structure   `FUN_140ac755c` : la longueur en octets du contenu (entier variable)
//	entier variable      `FUN_140ac7668` : sept bits par octet, poids faible d abord, 0x80 = suite
//	en-tete de champ     `FUN_140ac75e8` : `type | id << 5` si id < 6 ; sinon `type | 0xC0` puis
//	                     l id sur un octet (id < 0x100), ou `type | 0xE0` puis deux octets
//	liste                `FUN_140ac7430` : `type | (n + 1) << 5` si n < 7 ; sinon `type` puis n
//	fin de structure     `FUN_1411b3740` : un octet, 0 (1 pour une structure de base)
//	booleen              `FUN_1424d6668` : en-tete de type 2 puis un octet
//	entier signe 32      `FUN_140d1a268` : en-tete de type 0x10 puis l entier en zigzag
//	champ optionnel      ecrit seulement s il differe de son defaut (`FUN_140b23f64` et suivants)
//
// La variante (`FUN_140b85504`) est une structure dont les champs 0 a 3 sont des listes d au plus une
// structure, celle de la variante (`FUN_141086fec`). On y suit le chemin des champs lus :
//
//	variante.0 (en-tete, `FUN_1410301c4`) .0     int32   m_gameEngineType        defaut 0
//	variante.1 (`FUN_141057d14`) .0 (`FUN_141025dd0`) .5   `i343.NetProtocol.GameOptions.PlaybackSettings`
//	    .0 booleen killcamEnabled (`options + 0x260`), .2 booleen playOfTheGameEnabled (`+ 0x268`)
//
// Tout autre champ est enjambe : une structure par sa longueur, un booleen et un entier signe de 32
// bits par la forme que leurs ecrivains leur donnent. La marche s arrete — rien n est alors lu — sur
// un champ d un autre type a l une de ces places ; sur un champ d un type qu elle ne sait pas enjamber
// quand un champ du chemin reste a lire dans la meme structure (il pourrait le suivre, et sa valeur
// par defaut serait rendue comme lue) ; sur une structure dont la longueur ne tombe pas sur son octet
// de fin ; ou au bout du tampon. Un champ du chemin absent d une structure lue jusqu a sa fin vaut son
// defaut : l ecrivain omet un champ optionnel egal a son defaut.

// largeurChaineDuCorps et largeurChaineLongueDuCorps sont les longueurs maximales (`R9D`) des chaines
// du corps : `0x80` (+ 0xEA694), `0x100` (+ 0xEA810, + 0xEA910).
const (
	largeurChaineDuCorps       = 0x80
	largeurChaineLongueDuCorps = 0x100
	largeurBlocOptionnelCorps  = 0x700 // `FUN_140ee5ef8` : R(1) ; si 1 : R(0x700)
	largeurQueueDuCorps        = 0x6C0 // + 0xEAA18
)

// Les types Bond que la marche reconnait (valeurs ecrites par les ecrivains cites plus haut).
const (
	bondFin       = 0x0
	bondFinDeBase = 0x1
	bondBooleen   = 0x2
	bondStructure = 0xA
	bondListe     = 0xB
	bondEntier32  = 0x10
)

// Les champs Bond du chemin de la variante.
const (
	champEnTeteDeVariante   = 0 // variante.0
	champTypeDeMoteur       = 0 // en-tete.0 : m_gameEngineType
	champReglagesDeVariante = 1 // variante.1
	champReglagesGeneraux   = 0 // variante.1.0
	champReglagesDeRejeu    = 5 // variante.1.0.5 : PlaybackSettings
	champKillcam            = 0
	champMeilleureAction    = 2
)

// lecteurDuCorps est le curseur de la marche du corps. ok tombe au premier champ que la grammaire ne
// lit pas ; une lecture au-dela du tampon se constate a la fin ([source.Bits.Deborde]).
type lecteurDuCorps struct {
	br *Lecteur
	ok bool
}

// champsDuChemin est l ensemble des id de champ d une structure que la marche suit (bit `id`), les
// seuls que la grammaire lit ; tout autre champ est enjambe.
type champsDuChemin uint64

// contient dit que le champ `id` est sur le chemin.
func (c champsDuChemin) contient(id int) bool { return id >= 0 && id < 64 && c&(1<<id) != 0 }

// champs rend l ensemble des id `ids`.
func champs(ids ...int) champsDuChemin {
	var c champsDuChemin
	for _, id := range ids {
		c |= 1 << id
	}
	return c
}

// lire lit n bits (n <= 64).
func (l *lecteurDuCorps) lire(n uint) uint64 { return l.br.ReadBits(n) }

// chaine lit une chaine de `FUN_1407cbc24` ([lireChaine]) ; sans octet nul, le flux est en erreur.
func (l *lecteurDuCorps) chaine(max int) { l.ok = lireChaine(l.br, max) && l.ok }

// entier lit un entier variable de Bond (`FUN_140ac7668`), sur au plus cinq octets (32 bits).
func (l *lecteurDuCorps) entier() uint64 {
	var v uint64
	for i := range uint(5) {
		o := l.lire(8)
		v |= (o & 0x7F) << (7 * i)
		if o&0x80 == 0 {
			return v
		}
	}
	l.ok = false
	return 0
}

// enTete lit un en-tete de champ Bond : son id et son type.
func (l *lecteurDuCorps) enTete() (int, int) {
	o := l.lire(8)
	ty, id := int(o&0x1F), int(o>>5)
	switch id {
	case 6:
		id = int(l.lire(8))
	case 7:
		id = int(l.lire(8) | l.lire(8)<<8)
	}
	return id, ty
}

// structure lit une structure Bond : sa longueur, puis ses champs. Un champ du chemin (`chemin`) est
// passe a lireChamp, qui le lit s il a le type que la grammaire lit ; d un autre type, la lecture
// s arrete. Un champ hors du chemin est enjambe ([lecteurDuCorps.enjamber]) ; d un type que la marche
// ne sait pas enjamber, elle saute a la fin de la structure, connue par sa longueur — sauf s il reste
// un champ du chemin non lu, qui pourrait le suivre : la lecture s arrete, rien n est devine. La
// structure doit finir sur son octet de fin.
func (l *lecteurDuCorps) structure(chemin champsDuChemin, lireChamp func(id, ty int) bool) {
	n := int(l.entier())
	fin := l.br.BitPos() + 8*n
	var lus champsDuChemin
	for l.ok && l.br.BitPos() < fin {
		id, ty := l.enTete()
		switch {
		case ty == bondFin:
			l.ok = l.ok && l.br.BitPos() == fin
			return
		case ty == bondFinDeBase:
		case chemin.contient(id):
			lu := lireChamp(id, ty) // d abord : sa lecture peut elle-meme faire tomber ok
			l.ok = l.ok && lu
			lus |= champs(id)
		case l.enjamber(ty):
		case chemin&^lus != 0:
			l.ok = false
		default:
			l.allerA(fin - 8)
			l.ok = l.ok && l.lire(8) == bondFin
			return
		}
	}
	l.ok = false
}

// enjamber passe la valeur d un champ hors du chemin dont la marche connait la forme : une structure
// par sa longueur, un booleen (`FUN_1424d6668`, un octet) et un entier signe de 32 bits
// (`FUN_140d1a268`, entier variable) tels que leurs ecrivains les ecrivent. Rend faux pour tout autre
// type, dont la valeur n est pas lue.
func (l *lecteurDuCorps) enjamber(ty int) bool {
	switch ty {
	case bondStructure:
		l.enjamberLaStructure()
	case bondBooleen:
		l.lire(8)
	case bondEntier32:
		l.entier()
	default:
		return false
	}
	return true
}

// enjamberLaStructure passe une structure Bond par sa longueur ; son dernier octet est sa fin.
func (l *lecteurDuCorps) enjamberLaStructure() {
	n := int(l.entier())
	if n == 0 {
		l.ok = false
		return
	}
	l.allerA(l.br.BitPos() + 8*(n-1))
	l.ok = l.ok && l.lire(8) == bondFin
}

// allerA avance le curseur au bit `bit`.
func (l *lecteurDuCorps) allerA(bit int) {
	if !l.ok || bit < l.br.BitPos() {
		l.ok = false
		return
	}
	l.br.SetBitPos(bit)
}

// lireLaVarianteDePartie lit, depuis le premier bit du corps de `chunk_00`, ce que la variante de partie
// declare. Rien n est rendu (Lue fausse) quand la marche ne va pas jusqu a la table des joueurs.
func lireLaVarianteDePartie(chunk0 []byte, bodyBit int) profile.VarianteDePartie {
	v, _ := marcherLeCorps(chunk0, bodyBit)
	return v
}

// marcherLeCorps lit le corps de `chunk_00` jusqu au premier enregistrement de joueur, et rend ce que
// la variante declare et le bit de cet enregistrement.
func marcherLeCorps(chunk0 []byte, bodyBit int) (profile.VarianteDePartie, int) {
	l := &lecteurDuCorps{br: LecteurSur(chunk0), ok: bodyBit > 0 && bodyBit < len(chunk0)*8}
	if !l.ok {
		return profile.VarianteDePartie{}, 0
	}
	l.br.SetBitPos(bodyBit)
	for _, n := range [...]uint{3, 3, 2, 7, 64, 32, 3, 32, 32, 32} {
		l.lire(n)
	}
	l.structure(0, func(int, int) bool { return false }) // FUN_140ee5f40 : aucun champ suivi
	l.chaine(largeurChaineDuCorps)
	l.lire(32)
	l.lire(32)
	if l.lire(1) == 1 {
		l.br.Skip(largeurBlocOptionnelCorps)
	}
	for _, n := range [...]uint{1, 1, 2, 1, 32, 32, 1} {
		l.lire(n)
	}
	var v profile.VarianteDePartie
	if v.Presente = l.lire(1) == 1; v.Presente {
		l.structure(champs(0, 1, 2, 3), func(_, ty int) bool { return l.listeDeVariantes(ty, &v) })
	}
	l.chaine(largeurChaineLongueDuCorps)
	l.chaine(largeurChaineLongueDuCorps)
	l.lire(32)
	l.br.Skip(largeurQueueDuCorps)
	if !l.ok || l.br.Deborde() {
		return profile.VarianteDePartie{}, 0
	}
	v.Lue = true
	return v, l.br.BitPos()
}

// listeDeVariantes lit un champ de la structure de tete de la variante : une liste d au plus une
// structure de variante.
func (l *lecteurDuCorps) listeDeVariantes(ty int, v *profile.VarianteDePartie) bool {
	if ty != bondListe {
		return false
	}
	o := l.lire(8)
	n := int(o >> 5)
	if o&0x1F != bondStructure || n == 0 {
		return false
	}
	for range n - 1 {
		l.structure(champs(champEnTeteDeVariante, champReglagesDeVariante),
			func(id, ty int) bool { return l.champDeVariante(id, ty, v) })
	}
	return true
}

// champDeVariante lit les deux champs de la variante que la marche suit : l en-tete et les reglages.
func (l *lecteurDuCorps) champDeVariante(id, ty int, v *profile.VarianteDePartie) bool {
	if ty != bondStructure {
		return false
	}
	if id == champEnTeteDeVariante {
		l.structure(champs(champTypeDeMoteur), func(_, ty int) bool {
			if ty != bondEntier32 {
				return false
			}
			v.TypeDeMoteur = zigzag32(l.entier())
			return true
		})
		return true
	}
	l.structure(champs(champReglagesGeneraux), func(_, ty int) bool {
		if ty != bondStructure {
			return false
		}
		l.reglagesGeneraux(v)
		return true
	})
	return true
}

// reglagesGeneraux lit la structure `FUN_141025dd0` jusqu a ses reglages de rejeu.
func (l *lecteurDuCorps) reglagesGeneraux(v *profile.VarianteDePartie) {
	l.structure(champs(champReglagesDeRejeu), func(_, ty int) bool {
		if ty != bondStructure {
			return false
		}
		l.structure(champs(champKillcam, champMeilleureAction),
			func(id, ty int) bool { return l.reglageDeRejeu(id, ty, v) })
		return true
	})
}

// reglageDeRejeu lit un booleen de `PlaybackSettings` que la marche suit.
func (l *lecteurDuCorps) reglageDeRejeu(id, ty int, v *profile.VarianteDePartie) bool {
	if ty != bondBooleen {
		return false
	}
	b := l.lire(8) != 0
	if id == champKillcam {
		v.KillcamEnabled = b
	} else {
		v.PlayOfTheGameEnabled = b
	}
	return true
}

// zigzag32 decode l entier signe que `FUN_140d1a268` ecrit en zigzag.
func zigzag32(u uint64) int32 {
	return int32(uint32(u>>1)) ^ -int32(uint32(u&1))
}
