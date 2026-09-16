package filmdec

import "levelup/go-api/internal/analysis/filmsource"

// selectorBits is the width of the leading selector in a signed variable-width
// integer; the selector value sel gives the field width w = 8 << sel.
const selectorBits = 2

// signExtendMaxWidth is the exclusive upper bound below which a variable-width
// field is sign-extended (only widths 8 and 16; width 32 is a full int32 and
// width 64 is truncated to its low 32 bits). Mirrors the `iVar4 < 0x20` test in
// FUN_140c18a1c.
const signExtendMaxWidth = 32

// lecteur.go — LE LECTEUR DE LA GRAMMAIRE (`bitreader.go` jusqu au lot 2.4.2).

// Lecteur est LE LECTEUR DE LA GRAMMAIRE : le lecteur de bits canonique de la couche
// `source` ([filmsource.Bits]), plus ce qui appartient au decodeur — un profil de largeurs, un
// etat de capture, un observateur.
//
// # IL NE LIT PLUS AUCUN OCTET LUI-MEME (lot 2.4, ADR 0034 D-2)
//
// Jusqu au lot 2.4 il s appelait `BitReader`, portait son propre `buf []byte` et sa propre
// position en bits, et c etait le PREMIER des sept lecteurs de bits du depot. Le nom a change
// avec la nature : il ne lit plus, il DECORE. `BitReader` et `NewBitReader` restent nommes dans
// `archlint/no_raw_film_bytes_outside_source_test.go`, en ratchet anti-resurrection — les
// reintroduire, ici ou ailleurs, rougit. Les methodes de lecture — `ReadBits`, `ReadBit`, `Skip`,
// `BitPos`, `SetBitPos`, `Remaining` — sont desormais celles de [filmsource.Bits], PROMUES par
// l embarquement : une seule implantation, une seule convention de bourrage de queue. Ce qui
// reste ici est la grammaire : [Lecteur.ReadSignedVarWidth], le codec du moteur, et les trois
// champs prives ci-dessous.
type Lecteur struct {
	// Bits : le lecteur canonique. Embarque, donc ses methodes sont celles de ce type.
	*filmsource.Bits
	// p est le PROFIL DE BALAYAGE que ce lecteur porte (lot 2.3 du PLAN_DECODEUR_FILM ;
	// les lots 2.2.a a 2.2.e l avaient assemble valeur par valeur). Il decide des largeurs :
	// descripteur de traversee et largeur d axe absolue, cadre d image-cle, decoupage MPP,
	// `param_4` force. Cf. [ProfilDeBalayage] pour ce qu il porte et [Lecteur.poserProfil]
	// pour qui l installe.
	p ProfilDeBalayage
	// cap est l ETAT DE CAPTURE de ce lecteur (lot 2.3) : ou le composant i0 courant a
	// commence, a quel slot il appartient, et le monde sur lequel les positions
	// s accumulent. Cf. [captureDePosition] — six variables de paquet jusqu au lot 2.3.
	cap captureDePosition
	// obs est l OBSERVATEUR de ce lecteur (lot 2.3), ou nil — le cas de la PRODUCTION,
	// qui n observe rien. Il ne change AUCUNE consommation de bits : c est la propriete
	// qui le distingue du profil (cf. l en-tete de `observateur.go`).
	obs *Observation
}

// LecteurSur rend un lecteur de grammaire positionne sur le premier bit de `buf`. C est la
// SEULE porte de construction du paquet, et elle passe par la couche source
// ([filmsource.NewBits]) : `filmdec` ne fabrique plus de lecteur de bits.
//
// LE LECTEUR NAIT AVEC L INVARIANT DU PROFIL ([ProfilDeBalayageParDefaut]), jamais avec un
// etat de processus : depuis le lot 2.3 il n en existe plus. Un balayage qui tient son propre
// profil l installe EN TETE ([Lecteur.poserProfil]) — c est ce que font les portes a
// [FrameConfig], le contexte du film ([FilmContext.NouveauLecteur]) et les marches qui
// recoivent leur profil de leur appelant.
func LecteurSur(buf []byte) *Lecteur {
	return &Lecteur{Bits: filmsource.NewBits(buf), p: ProfilDeBalayageParDefaut()}
}

// PoserProfil installe le profil de balayage de ce lecteur et rend le precedent. C est la
// SEULE porte : un lecteur ne prend ses largeurs nulle part ailleurs.
func (b *Lecteur) PoserProfil(p ProfilDeBalayage) ProfilDeBalayage {
	prev := b.p
	b.p = p
	return prev
}

// Profil rend le profil de balayage que ce lecteur porte.
func (b *Lecteur) Profil() ProfilDeBalayage { return b.p }

// PoserObservation installe l observateur de ce lecteur et rend le precedent. `nil` = personne
// n observe, et c est le cas de la production.
func (b *Lecteur) PoserObservation(o *Observation) *Observation {
	prev := b.obs
	b.obs = o
	return prev
}

// PoserContexte installe SUR CE LECTEUR le profil et l observateur d un balayage, d un seul
// geste — aucune des deux moities ne peut etre oubliee.
func (b *Lecteur) PoserContexte(c ContexteDeLecture) {
	b.p = c.Profil
	b.obs = c.Obs
}

// poserCadre installe SUR CE LECTEUR tout ce que le cadre d un balayage porte : le profil (qui
// DECIDE des largeurs) et l observateur (qui ne fait que RECEVOIR). Les portes de balayage
// passent par la — un lecteur construit au milieu d une marche hérite ainsi des deux d un seul
// geste, et aucune des deux moities ne peut etre oubliee.
func (b *Lecteur) poserCadre(cfg FrameConfig) {
	b.p = cfg.Profil
	b.obs = cfg.Obs
}

// cadre rend le CADRE d image-cle d etat complet que ce lecteur porte : l en-tete par entite,
// la largeur d un mot de taille, et la regle `172 + etat(ti)` ([KeyframeProfile.CadreBits]).
func (b *Lecteur) cadre() KeyframeProfile { return b.p.Cadre }

// poserMouvement installe le profil de MOUVEMENT du balayage, EN TETE de celui-ci, sans
// toucher au reste du profil que le lecteur porte deja.
//
// # CE QU IL REMPLACE
//
// Jusqu au lot 2.2.a, les cinq valeurs du chemin de position (`TraversalPrecision`,
// `absoluteAxisW`, `PositionFullPrecision`, `PositionDeltaHasHandleTail`,
// `PositionCalibratedSkip`) etaient des VARIABLES DE PAQUET : le seul ecrivain de production
// (la calibration de `killsource`) les posait pour tout le processus, ce qui obligeait tout
// decodage a passer sous un verrou de paquet. Elles voyagent avec le lecteur.
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
// `cfg.Profil`. Partout ailleurs, l invariant pose par [LecteurSur] fait foi.
func (b *Lecteur) poserMouvement(m MovementProfile) { b.p.Mouvement = m }

// ReadSignedVarWidth decodes the engine's signed variable-width integer: a 2-bit
// selector sel sets the field width w = 8 << sel (8, 16, 32 or 64); w bits follow
// MSB-first. The result is sign-extended only for w in {8, 16}; w = 32 is a full
// int32 and w = 64 is truncated to its low 32 bits. Mirrors FUN_140c18a1c.
func (b *Lecteur) ReadSignedVarWidth() int32 {
	sel := uint(b.ReadBits(selectorBits))
	w := uint(8) << sel
	v := uint32(b.ReadBits(w)) // low 32 bits: w=64 discards the high half, w=32 is a full dword
	if w < signExtendMaxWidth && v>>(w-1)&1 != 0 {
		v |= ^uint32(0) << w // fill the bits above the sign bit
	}
	return int32(v)
}
