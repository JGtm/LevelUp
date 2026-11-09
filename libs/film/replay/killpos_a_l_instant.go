package replay

import (
	"cmp"
	"slices"
	"sort"

	"github.com/JGtm/LevelUp/libs/film/internal/facts/fallback"
	"github.com/JGtm/LevelUp/libs/film/internal/grammar"
	"github.com/JGtm/LevelUp/libs/film/types"
)

// killpos_a_l_instant.go — OU EST UN JOUEUR A L'INSTANT D'UNE MORT, QUAND LE FILM SE TAIT (decision
// utilisateur D-16 du 2026-10-17).
//
// # LA LECTURE, ET SES TROIS VOIES
//
// Elle sert les deux cotes d'un frag (`kill_positions`), le lieu d'un tueur deja mort (D-17) et le
// lieu de la victime du contexte de mort (`ContextesDesMorts`) : un seul lieu par joueur et par
// instant. A l'instant `t`, sur les sieges que le registre nomme pour le joueur :
//
//	ECHANTILLON  un seul siege porte un echantillon a `killPosToleranceUS` : c'est lui (`positionOf`).
//	             Deux sieges : la lecture se tait (deux corps pour un joueur).
//	TENUE        aucun echantillon a la tolerance. Le film n'ecrit la position d'un bipede que quand
//	             elle change : le DERNIER echantillon d'avant `t` est la position du joueur tant que le
//	             film n'en ecrit pas d'autre, a trois conditions lues — un seul siege porte ce dernier
//	             echantillon, le journal ne porte aucune mort du joueur entre lui et `t` (meme vie), et
//	             aucune montee a bord (`object-parent-state`, i10) ne court sur le silence.
//	SIEGE        une montee a bord court sur le silence (lue depuis `vehicleSeatTol` avant le dernier
//	             echantillon, ni descente ni retour du bipede au flux depuis) : un occupant ne replique
//	             plus son bipede, sa position est celle du VEHICULE qu'il occupe, l'echantillon du
//	             vehicule a la tolerance — la meme lecture que celle qui pose les tirs embarques.
//
// Le reste se tait et se compte ([KillPosReport]) : a bord d'un vehicule sans echantillon a la
// tolerance, debarque pendant le silence (montee puis descente, sans retour au flux).
//
// AUCUN CRITERE DE VITESSE, AUCUNE DUREE : la tenue s'appuie sur ce que le film ecrit (il ecrit les
// changements), pas sur une allure presumee. Sans lecture d'occupation (`LectureDuBord.Lue` faux), la
// condition « hors embarquement » n'est pas verifiable : seule la voie ECHANTILLON s'applique.

// LectureDuBord porte ce que le film dit de l'occupation des vehicules, deja lu par l'appelant.
type LectureDuBord struct {
	// Lue : l'occupation a ete lue (la marche des trames a tourne). Faux : ni tenue ni siege.
	Lue bool
	// Occupation : les lectures d'`object-parent-state` (i10) de la bande bipede — montees, descentes
	// et vehicule parent ([types.VehicleOccupancy]).
	Occupation []types.VehicleOccupancy
	// Vehicules : le nuage de positions des vehicules (`ti=40`), lu comme la cuisson le lit
	// ([PositionsDesVehicules]).
	Vehicules []grammar.BipedPosition
	// indexe : l'index de la lecture, partage par les copies de [LectureDuBord.Partagee].
	indexe *bordIndexe
}

// bordIndexe est l'index d'une lecture partagee, pose a la premiere lecture.
type bordIndexe struct {
	fait bool
	bord bordDuFilm
}

// Partagee rend la lecture dotee d'un index commun a toutes ses copies : la passe qui place les morts
// (`kill_positions`) et celle qui en date le contexte lisent le MEME bord contre les MEMES bipedes, et
// ne l'indexent qu'une fois — le repli `position_hors_emprise_ecartee` se compte alors une seule fois
// par passe, au lieu d'une fois par lecteur. Les copies ne se lisent que contre les positions du
// premier lecteur.
func (l LectureDuBord) Partagee() LectureDuBord {
	l.indexe = &bordIndexe{}
	return l
}

// voieDuLieu dit comment un lieu a ete lu, ou pourquoi il ne l'a pas ete.
type voieDuLieu int

const (
	voieMuette voieDuLieu = iota
	voieEchantillon
	voieTenue
	voieSiege
	// voieABordSansVehicule / voieDebarque : silences qui se taisent et se comptent.
	voieABordSansVehicule
	voieDebarque
)

// bordDuFilm est la lecture du bord indexee pour la recherche par instant.
type bordDuFilm struct {
	lu         bool
	occupation map[uint32][]types.VehicleOccupancy // par slot d'occupant, triees par instant
	vehicules  map[uint32]slotTrack
}

// lireLeBord indexe la lecture du bord. Les echantillons de vehicule hors de l'emprise jouee du film
// sont ecartes (F-1, la porte du calque des vehicules), l'emprise etant mesuree sur les bipedes.
//
// Une lecture [LectureDuBord.Partagee] n'est indexee, et ses replis comptes, qu'a la premiere lecture.
func lireLeBord(l LectureDuBord, bipedes []grammar.BipedPosition, fb *fallback.Compteur) bordDuFilm {
	if !l.Lue {
		return bordDuFilm{}
	}
	if l.indexe != nil && l.indexe.fait {
		return l.indexe.bord
	}
	b := indexerLeBord(l, bipedes, fb)
	if l.indexe != nil {
		l.indexe.fait, l.indexe.bord = true, b
	}
	return b
}

// indexerLeBord construit l'index d'une lecture lue.
func indexerLeBord(l LectureDuBord, bipedes []grammar.BipedPosition, fb *fallback.Compteur) bordDuFilm {
	b := bordDuFilm{lu: true, occupation: map[uint32][]types.VehicleOccupancy{}}
	for _, o := range l.Occupation {
		b.occupation[o.Slot] = append(b.occupation[o.Slot], o)
	}
	for s := range b.occupation {
		slices.SortStableFunc(b.occupation[s], func(x, y types.VehicleOccupancy) int {
			return cmp.Compare(x.TimestampUS, y.TimestampUS)
		})
	}
	if len(l.Vehicules) > 0 {
		gardes, _, _ := ecarterVehiculesHorsEmprise(VehicleScan{Scanned: true, Positions: l.Vehicules},
			empriseDesAxes(axesDesPositions(bipedes)), fb)
		b.vehicules = indexBySlot(gardes.Positions)
	}
	return b
}

// lectureALInstant est LA lecture du lieu d'un joueur a un instant (cf. l'en-tete).
type lectureALInstant struct {
	tracks   map[uint32]slotTrack
	bord     bordDuFilm
	morts    *mortsDuTueur
	offsetUS int64
}

// placer rend le lieu du joueur `xuid` a l'instant film `tUS` sur ses sieges `slots`, le slot qui le
// porte (0 hors des voies ECHANTILLON et TENUE) et la voie.
func (l lectureALInstant) placer(slots []uint32, xuid uint64, tUS uint64) (*Vec3, uint32, voieDuLieu) {
	proches := 0
	for _, s := range slots {
		if p, d := l.tracks[s].at(tUS); d <= killPosToleranceUS && p.HasWorld {
			proches++
		}
	}
	switch {
	case proches == 1:
		v, s := positionOf(l.tracks, slots, tUS)
		return v, s, voieEchantillon
	case proches > 1 || !l.bord.lu:
		return nil, 0, voieMuette
	}
	dernier, slot, ok := dernierEchantillonAvant(l.tracks, slots, tUS)
	if !ok || l.mortEntre(xuid, dernier.TimestampUS, tUS) {
		return nil, 0, voieMuette
	}
	switch parent, etat := l.bord.etatPendantLeSilence(l.tracks[slot], slot, dernier.TimestampUS, tUS); etat {
	case etatABord:
		v, ok := l.bord.positionDuVehicule(parent, tUS)
		if !ok {
			return nil, 0, voieABordSansVehicule
		}
		return v, 0, voieSiege
	case etatDebarque:
		return nil, 0, voieDebarque
	}
	return &Vec3{X: float64(dernier.X), Y: float64(dernier.Y), Z: float64(dernier.Z)}, slot, voieTenue
}

// dernierEchantillonAvant rend le dernier echantillon monde d'avant `tUS` des sieges `slots`, et son
// slot. Faux quand aucun n'en porte, ou quand deux sieges le portent au meme instant.
func dernierEchantillonAvant(tracks map[uint32]slotTrack, slots []uint32, tUS uint64,
) (grammar.BipedPosition, uint32, bool) {
	var best grammar.BipedPosition
	var slot uint32
	trouve, exAequo := false, false
	for _, s := range slots {
		pts := tracks[s].pts
		i := sort.Search(len(pts), func(k int) bool { return pts[k].TimestampUS > tUS })
		for i > 0 && !pts[i-1].HasWorld {
			i--
		}
		if i == 0 {
			continue
		}
		p := pts[i-1]
		switch {
		case trouve && p.TimestampUS == best.TimestampUS:
			exAequo = true
		case !trouve || p.TimestampUS > best.TimestampUS:
			best, slot, trouve, exAequo = p, s, true, false
		}
	}
	return best, slot, trouve && !exAequo
}

// mortEntre dit que le journal porte une mort du joueur entre le dernier echantillon et l'instant
// (exclu) : le silence n'est plus la meme vie. La borne basse recule de la tolerance des positions :
// le corps d'un mort replique encore quelques dizaines de millisecondes apres sa mort, et son dernier
// echantillon peut suivre l'instant de la mort.
func (l lectureALInstant) mortEntre(xuid uint64, depuisUS, tUS uint64) bool {
	if l.morts == nil {
		return false
	}
	for _, mMS := range l.morts.morts[xuid] {
		m := mMS*1000 + l.offsetUS
		if m+killPosToleranceUS >= int64(depuisUS) && m < int64(tUS) { //nolint:gosec // horloge du film, bien sous 2^63
			return true
		}
	}
	return false
}

// etatDuSilence : ce que l'occupation dit du silence d'un siege.
type etatDuSilence int

const (
	etatLibre etatDuSilence = iota
	etatABord
	etatDebarque
)

// etatPendantLeSilence lit les lectures d'occupation du siege `slot` depuis `vehicleSeatTol` avant
// son dernier echantillon jusqu'a `tUS`. La DERNIERE decide : une montee dont le bipede n'est pas
// revenu au flux (aucun echantillon apres `vehicleSeatTol` passe la montee) met le joueur a bord, et
// rend le vehicule parent ; une descente apres une montee de la fenetre le dit debarque ; une descente
// seule ou aucune lecture le laisse libre.
func (b bordDuFilm) etatPendantLeSilence(piste slotTrack, slot uint32, dernierUS, tUS uint64,
) (uint32, etatDuSilence) {
	tol := uint64(vehicleSeatTol.Microseconds()) //nolint:gosec // duree positive
	debut := uint64(0)
	if dernierUS > tol {
		debut = dernierUS - tol
	}
	monte, descend := false, false
	var montee types.VehicleOccupancy
	for _, o := range b.occupation[slot] {
		if o.TimestampUS < debut || o.TimestampUS > tUS {
			continue
		}
		if o.Attached {
			monte, descend, montee = true, false, o
			continue
		}
		descend = descend || monte
	}
	switch {
	case monte && !descend && !piste.echantillonEntre(montee.TimestampUS+tol, tUS):
		return montee.ParentSlot, etatABord
	case monte && descend:
		return 0, etatDebarque
	}
	return 0, etatLibre
}

// echantillonEntre dit que la piste porte un echantillon dans ]de, a].
func (s slotTrack) echantillonEntre(de, a uint64) bool {
	i := sort.Search(len(s.pts), func(k int) bool { return s.pts[k].TimestampUS > de })
	return i < len(s.pts) && s.pts[i].TimestampUS <= a
}

// positionDuVehicule rend la position du vehicule `slot` a `tUS`, a la tolerance des positions.
func (b bordDuFilm) positionDuVehicule(slot uint32, tUS uint64) (*Vec3, bool) {
	p, d := b.vehicules[slot].at(tUS)
	if d > killPosToleranceUS || !p.HasWorld {
		return nil, false
	}
	return &Vec3{X: float64(p.X), Y: float64(p.Y), Z: float64(p.Z)}, true
}
