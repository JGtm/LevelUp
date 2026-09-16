package filmdec

// selectorBits is the width of the leading selector in a signed variable-width
// integer; the selector value sel gives the field width w = 8 << sel.
const selectorBits = 2

// signExtendMaxWidth is the exclusive upper bound below which a variable-width
// field is sign-extended (only widths 8 and 16; width 32 is a full int32 and
// width 64 is truncated to its low 32 bits). Mirrors the `iVar4 < 0x20` test in
// FUN_140c18a1c.
const signExtendMaxWidth = 32

// BitReader is a sequential MSB-first big-endian bit reader over a byte buffer.
// It is the behavioral equivalent of the Halo replication reader: the i-th bit
// consumed is bit (7-(i mod 8)) of byte (i div 8).
type BitReader struct {
	buf []byte
	pos int // bit offset of the next bit to read
	// mv est le PROFIL DE MOUVEMENT que ce lecteur porte (lot 2.2.a du PLAN_DECODEUR_FILM).
	// Cf. l en-tete de [BitReader.poserMouvement] pour ce qu il remplace et pourquoi il vit
	// ici plutot qu en variable de paquet.
	mv MovementProfile
	// kf est le CADRE d image-cle que ce lecteur porte (lot 2.2.c). Il ne se pose jamais : ses
	// deux largeurs sont des INVARIANTS du format, relus chez l ecrivain, que rien n installe
	// par film. Il voyage quand meme avec le lecteur pour que les marches d etat complet
	// prennent leur cadre AU PROFIL et non a une constante du paquet — c est la seule forme
	// sous laquelle « fausser la valeur dans le profil » rougit leur lecture.
	kf KeyframeProfile
	// mpp est le decoupage du bloc `object-multiplayer-properties` que ce lecteur porte
	// (lot 2.2.e). Il vient de la VERSION DE FORMAT du film, posee par `InstallFilmFormatMPP`.
	mpp MPPWidths
	// rsp / rspImpose : le `param_4` du moteur qu un harnais de balayage a force, et le
	// drapeau qui dit qu il l a force (lot 2.2.e). Hors balayage, la table par composant
	// (`paramByComponent`) decide seule et ces deux champs ne sont jamais consultes.
	rsp       uint32
	rspImpose bool
}

// NewBitReader returns a reader positioned at the first bit of buf.
//
// LE LECTEUR NAIT AVEC LE PROFIL DE MOUVEMENT HERITE, dont la valeur AU REPOS est l invariant
// de [mouvementDuProfil] — la MEME fonction que [ResolveProfile] emploie, jamais une seconde
// table de valeurs. L heritage (`mouvement_herite.go`) reproduit a l identique ce que les cinq
// variables de paquet du chemin de position portaient avant le lot 2.2.a. Un balayage qui tient
// son propre profil l installe EN TETE ([BitReader.poserMouvement]) et n en depend plus.
func NewBitReader(buf []byte) *BitReader {
	return &BitReader{buf: buf, mv: herite.mouvement, kf: cadreDuProfil(),
		mpp: herite.mpp, rsp: herite.rsp, rspImpose: herite.rspImpose}
}

// cadre rend le CADRE d image-cle d etat complet que ce lecteur porte : l en-tete par entite,
// la largeur d un mot de taille, et la regle `172 + etat(ti)` ([KeyframeProfile.CadreBits]).
func (b *BitReader) cadre() KeyframeProfile { return b.kf }

// poserMouvement installe le profil de mouvement du balayage, EN TETE de celui-ci.
//
// # CE QU IL REMPLACE
//
// Jusqu au lot 2.2.a, les cinq valeurs du chemin de position (`TraversalPrecision`,
// `absoluteAxisW`, `PositionFullPrecision`, `PositionDeltaHasHandleTail`,
// `PositionCalibratedSkip`) etaient des VARIABLES DE PAQUET : le seul ecrivain de production
// (la calibration de `killsource`) les posait pour tout le processus, ce qui obligeait tout
// decodage a passer sous `LockProcessDecode`. Elles voyagent desormais avec le lecteur.
//
// # POURQUOI SUR LE LECTEUR, ET PAS EN PARAMETRE
//
// Le budget 0.A.5 interdit de lire un champ de profil PAR BIT LU. Le lecteur de bits est le
// seul objet deja passe a TOUS les deserialiseurs : y poser le profil coute une copie par
// lecteur construit (jamais par bit), et rend la valeur a portee de chaque feuille sans
// ajouter un parametre a la centaine de `consume*`. C est aussi la direction du lot 2.4
// (« une seule porte aux octets »).
//
// # QUI L APPELLE
//
// Les portes de balayage qui tiennent un [FrameConfig] (`DecodeFrameRecords`, `TryDeltaAt`,
// `DecodeFrameViews`, `DecodeFrameResync`, `DecodeFrameInfer`, `ScanFrameTargets`), avec
// `cfg.Mouvement`. Partout ailleurs, le profil herite pose par [NewBitReader] est EXACTEMENT
// la valeur que les variables de paquet portaient au meme instant.
func (b *BitReader) poserMouvement(m MovementProfile) { b.mv = m }

// BitPos returns the bit offset of the next bit to read.
func (b *BitReader) BitPos() int { return b.pos }

// Remaining returns the number of unread bits in the buffer.
func (b *BitReader) Remaining() int { return len(b.buf)*8 - b.pos }

// ReadBits reads n bits (0..64) MSB-first and returns them right-aligned in the
// low n bits. Bits past the end of the buffer read as zero, matching the engine's
// tail padding.
// Lecture par mot de 64 bits (cf. bits_word.go) sur le domaine ou elle coincide avec la
// boucle d'origine : position courante non negative et largeur <= 64. Hors de ce domaine
// — position negative (l'indexation panique, comme avant) ou largeur > 64 (le resultat ne
// garde que les 64 DERNIERS bits lus, et le curseur avance quand meme de n) — la boucle
// d'origine reste seule maitresse.
func (b *BitReader) ReadBits(n uint) uint64 {
	if b.pos >= 0 && n <= 64 {
		r := wordBitsAt(b.buf, b.pos, n)
		b.pos += int(n)
		return r
	}
	return b.readBitsLoop(n)
}

// readBitsLoop est la lecture bit a bit d'origine : elle porte les conventions de bord que
// le chemin par mot ne couvre pas.
func (b *BitReader) readBitsLoop(n uint) uint64 {
	var r uint64
	for i := uint(0); i < n; i++ {
		var bit uint64
		if idx := b.pos >> 3; idx < len(b.buf) {
			bit = uint64(b.buf[idx]>>(7-(uint(b.pos)&7))) & 1
		}
		r = r<<1 | bit
		b.pos++
	}
	return r
}

// ReadBit reads a single bit MSB-first.
func (b *BitReader) ReadBit() bool { return b.ReadBits(1) != 0 }

// Skip advances the read position by n bits without decoding them.
func (b *BitReader) Skip(n int) { b.pos += n }

// SetBitPos moves the read position to an absolute bit offset (used to resync a
// shared reader across replication views without allocating a new BitReader).
func (b *BitReader) SetBitPos(p int) { b.pos = p }

// ReadSignedVarWidth decodes the engine's signed variable-width integer: a 2-bit
// selector sel sets the field width w = 8 << sel (8, 16, 32 or 64); w bits follow
// MSB-first. The result is sign-extended only for w in {8, 16}; w = 32 is a full
// int32 and w = 64 is truncated to its low 32 bits. Mirrors FUN_140c18a1c.
func (b *BitReader) ReadSignedVarWidth() int32 {
	sel := uint(b.ReadBits(selectorBits))
	w := uint(8) << sel
	v := uint32(b.ReadBits(w)) // low 32 bits: w=64 discards the high half, w=32 is a full dword
	if w < signExtendMaxWidth && v>>(w-1)&1 != 0 {
		v |= ^uint32(0) << w // fill the bits above the sign bit
	}
	return int32(v)
}
