package grammar

// keyframe_etat_complet_admission.go — LA REGLE D ADMISSION D UNE LECTURE D ETAT COMPLET DU BIPEDE
// ET SA PUBLICATION (plan `.ai/PLAN_RI_ETAT_COMPLET_IMAGES_CLES_2026-10-08.md`, D1.1.2 et D1.1.3).
//
// # LA REGLE (U-1 AMENDEE, DECISION DE L UTILISATEUR DU 2026-10-09)
//
// Une valeur d image-cle lue par la grammaire n est publiee que si son record est ADMIS :
//
//	(record ferme OU n(i22) = 4) ET T1 ET T2
//
//	T1 : la marche a depasse le dernier emplacement d arme, au moins un emplacement est non vide, et
//	     la famille (moitie haute de l identifiant) de chaque emplacement non vide est au catalogue ;
//	T2 : le masque d i47 (R(6)) est EGAL a la bitmap des compteurs non nuls d i22.
//
// Pourquoi deux temoins APRES i43 : n(i22) = 4 ne valide le curseur que jusqu a i22, et dans les
// formats 20 et 21 meme un record ferme rend des armes et des munitions fausses (D1.0.6 : A f20-21
// armes 0/16). Mesure du 2026-10-09 sur les 28 films du corpus : 6 507 records admis sur 10 710 ;
// 0 record a armes fausses admis en f20-21 (49 admis) ; sur la marche decalee d un bit, 1 record
// admis sur 10 407 ; T1 seul en admettait un faux en f20-21, T2 seul 86.
//
// Un record NON ADMIS ne rend rien ici : la fenetre le rend, derriere la lecture, sous un repli
// nomme et compte (D1.2).

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/weaponv3"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// LA CONFIGURATION DE LA MARQUE DE PORTAGE (decision U-2 (b), 2026-10-08, confirmee le 2026-10-09).
// La fenetre `0x00010005` (keyframe_carrier_mark.go) n est pas un champ : elle recouvre la fin de
// l etat de mort i11 sous sa forme par defaut de 42 bits, le bit d `object-scale` i12 a 1, et la
// tete du R(5) de drapeaux d i13 `object-maximum-vitalities` (`FUN_1407eef08`, octet +0x39f de
// l objet) a 01111. La grammaire lit la MEME configuration dans ses occurrences ; le SENS des
// drapeaux 0x4 et 0x8 n est pas relu (D1.0.4 : 120 records les portent hors de la fenetre, tous des
// joueurs morts), d ou la configuration exacte et non le seul drapeau. Mesure : egale a la fenetre
// sur les 5 401 records fermes du corpus (92/92).
const (
	// largeurDeLEtatDeMortParDefaut et etatDeMortParDefaut : la forme par defaut d i11,
	// `000000000011000000000000000010000000000000`.
	largeurDeLEtatDeMortParDefaut = 42
	etatDeMortParDefaut           = 0xC0002000
	// largeurDesDrapeauxDeVitalites et vitalitesDeLaMarque : le R(5) de tete d i13 (`FUN_1407f0354`).
	largeurDesDrapeauxDeVitalites = 5
	vitalitesDeLaMarque           = 0b01111
)

// bitsDeLOccurrence rend les `n` premiers bits de l occurrence `co`, MSB d abord.
func bitsDeLOccurrence(pay []byte, co lecture.Composant, n int) uint64 {
	return source.BitsBourres(pay, int(co.Debut), n)
}

// raisonDeRefus dit pourquoi un record bipede n est pas admis ; [refusAucun] pour un record admis.
type raisonDeRefus uint8

// Les raisons de refus, dans l ordre ou la regle les juge.
const (
	refusAucun raisonDeRefus = iota
	// refusNiFermeNiI22 : record non ferme dont i22 n est pas lu a quatre compteurs.
	refusNiFermeNiI22
	// refusT1 : temoin des armes refuse.
	refusT1
	// refusT2 : temoin du jeu de grenades refuse.
	refusT2
	// refusDebordement : une occurrence du record ne se relit pas a l etendue que la marche lui a donnee
	// ([relireLOccurrence] rend faux) ; ses crochets ont pu ecrire des valeurs lues a une autre
	// largeur, aucune n est publiee.
	refusDebordement
)

// admettre juge un record bipede lu ; `dernierEmplacement` est l index du dernier composant
// d identite d arme de l archetype (-1 : aucun).
func admettre(r *lecture.Record, l *lectureDEtatComplet, dernierEmplacement int) raisonDeRefus {
	if l.debordements > 0 {
		return refusDebordement
	}
	quatre := l.grenadesLues && l.compte == 4 && len(l.compteurs) == emplacementsDArme
	switch {
	case r.Preuve != lecture.PreuveFerme && !quatre:
		return refusNiFermeNiI22
	case !l.temoinDesArmes(int(r.Desync), dernierEmplacement):
		return refusT1
	case !quatre || !l.jeuDeGrenadesLu || l.masque != bitmapDesCompteurs(l.compteurs):
		return refusT2
	}
	return refusAucun
}

// temoinDesArmes est T1 : la marche (arretee a `desync`, -1 au bout) a depasse le dernier
// emplacement, au moins un emplacement est non vide, chaque famille non vide est au catalogue.
func (l *lectureDEtatComplet) temoinDesArmes(desync, dernierEmplacement int) bool {
	if dernierEmplacement < 0 || (desync >= 0 && desync <= dernierEmplacement) {
		return false
	}
	nonVides := 0
	for k := range emplacementsDArme {
		if !l.armeLue[k] || l.armeHaute[k] == noVariant {
			continue
		}
		if _, connue := weaponv3.KnownWeaponHigh32Lookup(l.armeHaute[k]); !connue {
			return false
		}
		nonVides++
	}
	return nonVides > 0
}

// bitmapDesCompteurs rend la bitmap des compteurs non nuls (bit r = rang r), la forme du masque
// d i47.
func bitmapDesCompteurs(compteurs []uint64) uint32 {
	var m uint32
	for r, v := range compteurs {
		if v > 0 {
			m |= 1 << uint(r) //nolint:gosec // quatre rangs
		}
	}
	return m
}

// rangDeCapaciteMin et rangDeCapaciteMax bornent le domaine des rangs de capacite que la publication
// d image-cle porte : celui de la fenetre (16..23, [invAbilityRankHigh]). Un rang lu hors du domaine
// est compte et non publie (decision U-3) ; le rang complet vient d i48 dans les trames delta.
const (
	rangDeCapaciteMin = int(invAbilityRankHigh << 3)
	rangDeCapaciteMax = rangDeCapaciteMin + 7
)

// armesDe rend la dotation lue d un record admis : la famille de chaque emplacement non vide, dans
// l ordre des emplacements (l ordre des bits du record).
func (l *lectureDEtatComplet) armesDe(slot uint32) types.KeyframeLoadout {
	out := types.KeyframeLoadout{Slot: slot}
	for k := range emplacementsDArme {
		if l.armeLue[k] && l.armeHaute[k] != noVariant {
			out.Families = append(out.Families, l.armeHaute[k])
		}
	}
	return out
}

// refusDInventaire dit ce que la publication d un inventaire admis a tu : un rang de capacite hors du
// domaine publie, une selection de grenade hors du masque d i47 ou de son domaine.
type refusDInventaire struct{ capaciteHorsDomaine, selectionHorsMasque bool }

// inventaireDe rend l inventaire lu d un record admis, et ce que sa publication a tu.
func (l *lectureDEtatComplet) inventaireDe(slot uint32) (types.KeyframeInventory, refusDInventaire) {
	inv := types.KeyframeInventory{Slot: slot, AbilityRank: -1, DrawnSlot: -1, SelectedGrenadeRank: -1}
	for k := range emplacementsDArme {
		inv.Grenades[k] = uint32(l.compteurs[k]) //nolint:gosec // R(8)
	}
	inv.GrenadesRead = true
	var tu refusDInventaire
	if l.capaciteLue && l.rang != AbilitySetNoRank {
		if l.rang >= rangDeCapaciteMin && l.rang <= rangDeCapaciteMax {
			inv.AbilityRank = l.rang
		} else {
			tu.capaciteHorsDomaine = true
		}
	}
	// LA SELECTION SUIT LE CONTRAT DU CANAL DELTA (decision du superviseur, revue D1.4.6, constat 5) :
	// une selection non nulle n est publiee que si elle designe un type du masque d i47 ; sinon la
	// lecture est publiee SANS selection, et le refus se compte.
	if l.jeuDeGrenadesLu && l.selection != GrenadeSetNoSelection {
		if selectionDansLeMasque(l.masque, l.selection) {
			inv.SelectedGrenadeRank = l.selection - 1
		} else {
			tu.selectionHorsMasque = true
		}
	}
	if l.jeuLu {
		inv.DrawnSlot = l.jeu.Principal
	}
	inv.Ammo, inv.AmmoRead = l.munitions()
	if inv.AmmoRead {
		inv.AmmoCandidates = 1
	}
	return inv, tu
}

// munitions rend l etat des quatre emplacements lus, et vrai quand les quatre chargeurs et les
// quatre reserves l ont ete.
func (l *lectureDEtatComplet) munitions() ([types.InventorySlotCount]types.SlotAmmo, bool) {
	var st [types.InventorySlotCount]types.SlotAmmo
	for k := range emplacementsDArme {
		if !l.chargeurLu[k] || !l.reserveLue[k] {
			return [types.InventorySlotCount]types.SlotAmmo{}, false
		}
		if l.aChargeur[k] {
			c := l.chargeur[k]
			st[k].Mag = &c
		}
		if l.aJauge[k] {
			j := float64(l.jauge[k]) / quantumDeJauge
			st[k].Gauge = &j
		}
		res := l.reserve[k]
		st[k].Res = &res
		st[k].Overheat, st[k].Flags = l.surchauffe[k], l.drapeaux[k]
	}
	return st, true
}

// quantumDeJauge est le plus grand quantum du R(12) de la jauge (`FUN_1406d84b4`, [0, 1]).
const quantumDeJauge = 4095.0

// porteLaMarque dit si un record admis porte la configuration de la marque de portage.
func (l *lectureDEtatComplet) porteLaMarque() bool {
	return l.mortParDefaut && l.echelleAUn && l.vitalites == vitalitesDeLaMarque
}
