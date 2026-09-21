package replay

// filmfacts_directions.go — LES DIRECTIONS D UNE POSITION : CAP, VELOCITE, SECOND VECTEUR, MASQUE.
//
// Extrait de `filmfacts_codec.go` le 2026-09-18, meme raison de taille. La coupe suit une
// frontiere reelle : la SUITE des positions reste la-bas, les seize champs de `componentDirs`
// viennent ici avec la mesure qui a impose de les porter.

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// Les drapeaux du SECOND octet d une position : les directions que `componentDirs` porte et que
// l octet historique ne nommait pas.
const (
	gdHasAim   byte = 1 << 0
	gdHasVel   byte = 1 << 1
	gdHasAimB  byte = 1 << 2
	gdAimFlag0 byte = 1 << 3
	gdAimFlag1 byte = 1 << 4
	gdAimFlag2 byte = 1 << 5
	gdMaskOver byte = 1 << 6
	// gdHasRoll : L ANGLE DE ROULIS d `i2`, la moitie manquante de l AVANT du chassis (lot 5.4).
	// Pose, il est suivi d un octet de mode (bit 0..1 = le mode, bit 2 = direction par defaut)
	// puis du quantum d angle. Les BIPEDES ne le posent jamais — leur balayage ne capture pas
	// les directions — donc cet octet et ce qui le suit ne coutent qu aux vehicules.
	gdHasRoll byte = 1 << 7
)

// Les deux bits de l octet de mode qui suit `gdHasRoll`.
const (
	gdModeMask   byte = 0x03
	gdAimDefault byte = 1 << 2
)

// encodeDirectionsDePosition / decodeDirectionsDePosition : LES TREIZE CHAMPS DE DIRECTION QUE LE
// CODEC NE PORTAIT PAS.
//
// # CE QUE LEUR ABSENCE A COUTE, ET COMMENT ON L A SU (2026-09-18, gate S8)
//
// `grammar.BipedPosition` embarque `componentDirs`, SEIZE champs. Le codec en portait TROIS —
// `HasYaw`, `YawRaw`, `PitchRaw` — et c etait suffisant pour les BIPEDES, qui sont balayes sans
// capture de directions. Les VEHICULES, eux, sont balayes avec `CaptureDirs` et
// `DynPrecOrientation` (`vehicleScanOptions`) : leur CAP sort de `HasAim` / `AimRaw`.
//
// MESURE SUR `11de8353` (diff structurel des deux artefacts, un seul decodage) :
// `/vehicles/samples/h` ABSENT aux faits sur **4 602 echantillons**, et
// `coverage.vehicles.withHeading` passant de **4 602 a 0**. Soit 43 526 octets d artefact, et
// l explication complete du fait que seuls les films a VEHICULES perdaient.
//
// LE SECOND OCTET DE DRAPEAUX NE COUTE RIEN AUX BIPEDES : aucun de leurs champs de direction n est
// pose, donc l octet vaut zero et rien ne suit.
//
// # `MaskBits` N EST PAS PORTE, ET C EST PROUVE PAR L ORACLE, PAS DECIDE A LA MAIN
//
// `componentDirs.MaskBits` est le MASQUE de composants du record : de la tracabilite de balayage,
// qu aucun calque ne publie. Le porter coutait 7,4 Mio sur les huit fixtures (un uint64 par
// position, 171 826 positions sur le seul film de reference) — le jeu passait de 13,4 a 21,4 Mio.
// IL EST DONC OMIS, et ce n est pas un jugement a la main comme celui qui a produit le defaut de
// 2026-09-18 : c est `TestGoldenInputsFidelite` qui le prouve, en comparant sur huit films reels
// l ARTEFACT SERIALISE des deux cotes. Si un calque venait a lire ce masque, ce gate rougirait.
func encodeDirectionsDePosition(w *gwriter, p grammar.BipedPosition) {
	var fd byte
	for _, c := range []struct {
		pose bool
		bit  byte
	}{
		{p.HasAim, gdHasAim}, {p.HasVel, gdHasVel}, {p.HasAimB, gdHasAimB},
		{p.AimFlag0, gdAimFlag0}, {p.AimFlag1, gdAimFlag1}, {p.AimFlag2, gdAimFlag2},
		{p.MaskOver, gdMaskOver}, {p.HasRoll, gdHasRoll},
	} {
		if c.pose {
			fd |= c.bit
		}
	}
	w.byte8(fd)
	if fd&gdHasRoll != 0 {
		m := p.FwdMode & gdModeMask
		if p.AimDefault {
			m |= gdAimDefault
		}
		w.byte8(m)
		w.u(uint64(p.RollRaw))
	}
	if fd&gdHasAim != 0 {
		w.u(uint64(p.AimRaw))
	}
	if fd&gdHasVel != 0 {
		w.u(uint64(p.VelRaw))
		w.u(uint64(p.VelScale))
	}
	if fd&gdHasAimB != 0 {
		w.u(uint64(p.YawRawB))
		w.u(uint64(p.PitchRawB))
	}
}

func decodeDirectionsDePosition(r *greader, p *grammar.BipedPosition) {
	fd := r.byte8()
	p.HasAim, p.HasVel, p.HasAimB = fd&gdHasAim != 0, fd&gdHasVel != 0, fd&gdHasAimB != 0
	p.AimFlag0, p.AimFlag1 = fd&gdAimFlag0 != 0, fd&gdAimFlag1 != 0
	p.AimFlag2, p.MaskOver = fd&gdAimFlag2 != 0, fd&gdMaskOver != 0
	if p.HasRoll = fd&gdHasRoll != 0; p.HasRoll {
		m := r.byte8()
		p.FwdMode, p.AimDefault = m&gdModeMask, m&gdAimDefault != 0
		p.RollRaw = uint32(r.u())
	}
	if p.HasAim {
		p.AimRaw = uint32(r.u())
	}
	if p.HasVel {
		p.VelRaw, p.VelScale = uint32(r.u()), uint32(r.u())
	}
	if p.HasAimB {
		p.YawRawB, p.PitchRawB = uint32(r.u()), uint32(r.u())
	}
}
