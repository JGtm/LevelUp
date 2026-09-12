package filmdec

// r7_charges_lot2_research_test.go — suite de `r7_charges_research_test.go` : la famille des
// SACS DE PROPRIETES NOMMEES (types 82, 15, 85), derivee de l'executable pendant ce lot.
//
// TROIS DECOUVERTES QUI CHANGENT LA DONNE :
//
//  1. `0x1406d676c` N'EST PAS un `R(64)` fixe (la grammaire de 2026-08-30 l'affirmait) :
//     c'est un `R(n)` GENERIQUE dont n, en bits, est le 4e argument
//     (`for (; 0x3f < n; n -= 0x40) +0x2c += 0x40 ; puis +0x2c += n`).
//  2. Du coup le type 15 `Script` est AUTO-DELIMITE : il porte `R(10)` = la LONGUEUR EN BITS
//     de sa charge brute, passee telle quelle a `0x1406d676c`
//     (`SHR R9,0x36 ; ... ; CALL 0x1406d676c` avec R9 = la valeur de 10 bits).
//  3. Le type 82 se termine par 32 `R(1)` INCONDITIONNELS (`FUN_14080ae28`, boucle
//     `CMP EBX,0x20 ; JL`) que la description anterieure du record ignorait.
//
// VARIANTES DE BUILD. Trois champs sont gardes par des globales a zero dans l'image mais
// ecrites au demarrage — non lisibles statiquement. Elles sont ici des VARIABLES calibrees
// sur pieces par `TestR7Variantes` (oracle de trame), jamais devinees.

// Variantes de build, CALIBREES SUR PIECES par TestR7Variantes (oracle de trame, seuil
// facteur 2 ecrit d'avance). Quand la mesure ne tranche pas, la valeur reste l'hypothese la
// plus faible (« globale a zero dans l'image ») et le type porte sa reserve.
var (
	// r7Var15Prefixe : type 15, prefixe R(15) lu si FUN_1404f25f4() != 0.
	// MESURE : profondeur de trame 1,664 AVEC prefixe contre 0,672 SANS (facteur 2,48,
	// 195-256 trames, 3 films) -> le prefixe EST present sur ce build.
	r7Var15Prefixe = true
	// r7Var85Queue : type 85, queue R(32)+R(32)+R(4) lue sous garde de session.
	// MESURE : aucune liste contenant un type 85 n'a ete marchee (2 tetes sur le parc) —
	// NON CONCLUANT, la reserve tient.
	r7Var85Queue = false
	// r7Var82Tag7Brut : type 82 tag 7, R(96) brut si FUN_14076f91c() != 0, sinon vecteur
	// quantifie (meme primitive 0x14076e524 que le type 117).
	// MESURE : profondeurs 1,921 et 1,936 — INDISCERNABLES (facteur 1,01). Explication
	// mesuree par TestR7Tag7 : le tag 7 n'apparait PAS dans les sacs de proprietes du parc,
	// donc la branche n'est jamais empruntee et le choix est sans effet.
	r7Var82Tag7Brut = false
)

// r7Porte5 consomme `R(1) g ; si g==0 : R(5)` (primitive 0x1407f2058, POLARITE INVERSEE).
func r7Porte5(br *BitReader) {
	if !br.ReadBit() {
		br.Skip(5)
	}
}

// r7TagsA / r7TagsB : recensement des etiquettes d'union reellement rencontrees dans le
// parc (rempli quand r7CompteTags est vrai). C'est ce qui permet de dire si la branche
// « tag 7 », la seule non fermee statiquement, est empruntee ou non.
var (
	r7CompteTags bool
	r7TagsA      = map[uint64]int{}
	r7TagsB      = map[uint64]int{}
)

// r7ValeurTagA consomme la valeur d'une propriete du sac principal du type 82
// (union etiquetee 0x14080eff0, tag = R(3) deja lu).
func r7ValeurTagA(br *BitReader, tag uint64, ctx r7Ctx) bool {
	if r7CompteTags {
		r7TagsA[tag]++
	}
	switch tag {
	case 0:
		return true // aucun handler
	case 1, 2, 3, 6:
		br.Skip(32)
		return true
	case 4:
		br.Skip(1)
		return true
	case 5: // chaine C auto-delimitee : R(8) jusqu'au premier octet nul, plafond 16 octets
		for i := 0; i < 16; i++ {
			if br.ReadBits(8) == 0 {
				return true
			}
		}
		return true
	case 7:
		// FUN_140f04f18 : R(96) brut si FUN_14076f91c(), sinon vecteur quantifie classe 0x10 —
		// exactement la primitive du type 117. r7VecteurQuantifie porte deja cette bascule.
		return r7VecteurQuantifie(br, ctx, 16)
	}
	return false
}

// r7ValeurTagB consomme la valeur d'un element du sous-sac du type 82 (union 0x1407f0ebc).
func r7ValeurTagB(br *BitReader, tag uint64) {
	if r7CompteTags {
		r7TagsB[tag]++
	}
	switch tag {
	case 0:
		return
	case 1:
		r7Porte5(br)
	case 2: // 0x142c70cd0 : R(1) g ; si g==0 : R(32) ; sinon R(24)
		if !br.ReadBit() {
			br.Skip(32)
		} else {
			br.Skip(24)
		}
	default:
		br.Skip(32)
	}
}

// r7SkipChargeLot2 est la suite de r7SkipCharge. Rend false pour tout type non ferme.
func r7SkipChargeLot2(br *BitReader, typ int, ctx r7Ctx) bool {
	switch typ {
	// --- 82 PlayerGameEventSmall (lecteur 0x14080add8) ---
	case 82:
		br.Skip(32 + 8) // event-id + R(8)  (FUN_14080ae70)
		n := br.ReadBits(3)
		for i := uint64(0); i < n; i++ {
			br.Skip(32) // nom de propriete
			if !r7ValeurTagA(br, br.ReadBits(3), ctx) {
				return false
			}
		}
		if br.ReadBit() { // sous-sac optionnel (FUN_14080b034)
			br.Skip(32)
			m := br.ReadBits(3)
			for i := uint64(0); i < m; i++ {
				r7ValeurTagB(br, br.ReadBits(3))
			}
		}
		br.Skip(32) // masque de 32 R(1) inconditionnels (FUN_14080ae28)
		return true

	// --- 15 Script (lecteur 0x14080bb4c) : AUTO-DELIMITE ---
	case 15:
		if r7Var15Prefixe {
			br.Skip(15)
		}
		br.Skip(13) // script-id
		n := br.ReadBits(10)
		br.Skip(int(n)) // charge brute, longueur en BITS
		return true

	// --- 85 PlayerKilledEvent (lecteur 0x14104bd08) ---
	case 85:
		r7Porte5(br)
		r7Porte5(br)
		br.Skip(32)
		br.Skip(1)
		r7Porte5(br)
		br.Skip(32)
		if r7Var85Queue {
			br.Skip(32 + 32 + 4)
		}
		return true
	}
	return r7SkipChargeLot3(br, typ, ctx)
}
