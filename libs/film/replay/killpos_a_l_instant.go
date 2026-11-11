package replay

import (
	"cmp"
	"slices"
	"sort"

	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/fallback"
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// killpos_a_l_instant.go — OU EST UN JOUEUR A L'INSTANT D'UNE MORT, QUAND LE FILM SE TAIT (decision
// utilisateur D-16 du 2026-10-17).
//
// # LA LECTURE, ET SES TROIS VOIES
//
// Elle sert les deux cotes d'un frag (`kill_positions`), l'entame (`kill_openings`), le lieu d'un tueur deja mort (D-17) et le
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
//	             echantillon, ni descente anterieure a la tolerance ni retour du bipede au flux depuis) :
//	             un occupant ne replique plus son bipede, sa position est celle du VEHICULE qu'il occupe,
//	             par la meme lecture que celle qui pose les tirs embarques ([bordDuFilm.positionDuVehicule]).
//
// Le reste se tait et se compte ([KillPosReport]) : a bord d'un vehicule qu'aucune lecture ne place,
// debarque pendant le silence (montee puis descente, sans retour au flux).
//
// UN ECHANTILLON N'APPARTIENT A LA LECTURE QUE S'IL EST CELUI DU CORPS QUI TIENT LE SIEGE A `t`
// ([lectureALInstant.sansEchantillonDUnAutreJoueur]) : un siege recycle porte plusieurs joueurs, et le
// premier echantillon d'un corps cree dans la tolerance de `t` ne dit rien du joueur precedent.
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
	// ([LireLesVehicules]).
	Vehicules []grammar.BipedPosition
	// Creations : les records de creation des vehicules (`ti=40`) : la position de naissance et le mot
	// d'identite du chassis. Une piece montee ou un poste fixe sans echantillon se lit par eux
	// ([bordDuFilm.positionDuVehicule]).
	Creations []types.EquipmentCreation
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
	creations  map[uint32][]types.EquipmentCreation // par slot de vehicule, triees par instant
	// fb compte, a la lecture, les porteurs de piece decides par le voisin de slot.
	fb *fallback.Compteur
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
	b := bordDuFilm{lu: true, occupation: map[uint32][]types.VehicleOccupancy{}, fb: fb,
		creations: creationsVehiculesParSlot(l.Creations)}
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
	// registre : le registre d'identite, pour savoir qui tient un siege a l'instant d'un echantillon.
	// Valeur zero : aucun corps connu, tout echantillon de la tolerance vaut pour le siege.
	registre IdentityRegistry
}

// sansEchantillonDUnAutreJoueur retire de `slots` les sieges dont l'echantillon a la tolerance de `tUS`
// est celui d'un AUTRE joueur que `xuid` : sur un siege aux index divergents, le registre nomme autrement
// (ou ne nomme pas) le corps qui tient le siege a l'instant de l'echantillon. Ce siege n'etait pas le
// corps du joueur a `tUS` : son echantillon ne le place pas. La comparaison est par JOUEUR, pas par corps :
// deux index du meme joueur gardent l'echantillon.
func (l lectureALInstant) sansEchantillonDUnAutreJoueur(slots []uint32, xuid uint64, tUS uint64) []uint32 {
	if len(l.registre.corps) == 0 {
		return slots
	}
	out := make([]uint32, 0, len(slots))
	for _, s := range slots {
		if p, d := l.tracks[s].at(tUS); d <= killPosToleranceUS && p.HasWorld && !l.memeJoueur(s, p.TimestampUS, xuid) {
			continue
		}
		out = append(out, s)
	}
	return out
}

// memeJoueur dit que le registre nomme `xuid` sur le siege `slot` a l'instant `tUS`. Vrai sur un siege
// sans divergence d'index : ses corps successifs sont un seul participant, quelle que soit leur generation.
// Vrai aussi sur un siege en COLLISION (deux joueurs dans un corps indiscernable) : aucun corps n'y est
// etabli, le registre s'y tait a tout instant hors d'une vie nommee, et ce silence ne prouve pas qu'un
// autre joueur tient le siege (un echantillon du corps juste apres la fin ecrite de sa vie, par exemple).
func (l lectureALInstant) memeJoueur(slot uint32, tUS uint64, xuid uint64) bool {
	if c, ok := l.registre.corps[slot]; !ok || !c.indexDivergents() || l.registre.collisionDeSiege(slot) {
		return true
	}
	x, _ := l.registre.joueurDuLieuA(slot, tUS)
	return x == xuid
}

// placer rend le lieu du joueur `xuid` a l'instant film `tUS` sur ses sieges `slots`, le slot qui le
// porte (son siege de bipede, sur les voies ECHANTILLON, TENUE et SIEGE) et la voie.
func (l lectureALInstant) placer(slots []uint32, xuid uint64, tUS uint64) (*Vec3, uint32, voieDuLieu) {
	if int64(tUS) < 0 { //nolint:gosec // un instant decale d avant l origine du film (entame) deborde en uint64
		return nil, 0, voieMuette
	}
	slots = l.sansEchantillonDUnAutreJoueur(slots, xuid, tUS)
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
	// LE DERNIER ECHANTILLON DOIT ETRE CELUI DU JOUEUR : sur un siege aux index divergents, il peut etre
	// celui du corps d'un autre occupant (un bot entre deux corps du joueur) ; tenir le joueur la, ou le
	// dire a bord depuis la, serait poser le lieu d'un autre.
	if !ok || !l.memeJoueur(slot, dernier.TimestampUS, xuid) || l.mortEntre(xuid, dernier.TimestampUS, tUS) {
		return nil, 0, voieMuette
	}
	switch parent, etat := l.bord.etatPendantLeSilence(l.tracks[slot], slot, dernier.TimestampUS, tUS); etat {
	case etatABord:
		v, ok := l.bord.positionDuVehicule(parent, tUS)
		if !ok {
			return nil, 0, voieABordSansVehicule
		}
		return v, slot, voieSiege
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
//
// UNE DESCENTE A LA TOLERANCE DE `tUS` N EST PAS UN DEBARQUEMENT : le film ecrit la sortie du siege a
// la mort a bord, dans la meme fraction de seconde. Elle se lit comme le corps replique encore a ± la
// tolerance : le joueur etait a bord a l'instant, et le vehicule le place.
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
		if o.TimestampUS+killPosToleranceUS >= tUS {
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

// positionDuVehicule rend le lieu du vehicule `slot` a `tUS`, par l'echantillon a la tolerance, sinon
// par ce que son record de creation dit de lui : une piece montee ne replique jamais sa position (son
// lieu est celui du chassis qui la porte, [porteurDUnePiece]), un poste de tir fixe reste ou le film
// le cree. Nil sur tout autre vehicule silencieux : le film ne le situe pas a cet instant.
func (b bordDuFilm) positionDuVehicule(slot uint32, tUS uint64) (*Vec3, bool) {
	if v, ok := b.echantillonDeVehicule(slot, tUS); ok {
		return v, true
	}
	c, ok := b.creationA(slot, tUS)
	if !ok {
		return nil, false
	}
	chassis, lu := gwPadsIdentity(c)
	if !lu {
		return nil, false
	}
	if spec, piece := vehicleTurretByChassis[chassis]; piece {
		return b.positionDuPorteur(c, spec, tUS)
	}
	if vehicleFamilyOf(chassis) == familleTourelleFixe {
		return &Vec3{X: float64(c.X), Y: float64(c.Y), Z: float64(c.Z)}, true
	}
	return nil, false
}

// echantillonDeVehicule rend la position du vehicule `slot` a `tUS`, a la tolerance des positions.
func (b bordDuFilm) echantillonDeVehicule(slot uint32, tUS uint64) (*Vec3, bool) {
	p, d := b.vehicules[slot].at(tUS)
	if d > killPosToleranceUS || !p.HasWorld {
		return nil, false
	}
	return &Vec3{X: float64(p.X), Y: float64(p.Y), Z: float64(p.Z)}, true
}

// creationA rend le record de creation du vehicule `slot` qui tient le slot a `tUS` : le dernier date
// au plus tard a cet instant.
func (b bordDuFilm) creationA(slot uint32, tUS uint64) (types.EquipmentCreation, bool) {
	var trouve types.EquipmentCreation
	lu := false
	for _, c := range b.creations[slot] {
		if c.TimestampUS > tUS {
			break
		}
		trouve, lu = c, true
	}
	return trouve, lu
}

// creationsVehiculesParSlot indexe les records de creation des vehicules par slot, tries par instant.
func creationsVehiculesParSlot(creations []types.EquipmentCreation) map[uint32][]types.EquipmentCreation {
	if len(creations) == 0 {
		return nil
	}
	out := map[uint32][]types.EquipmentCreation{}
	for _, c := range creations {
		out[c.Slot] = append(out[c.Slot], c)
	}
	for s := range out {
		slices.SortStableFunc(out[s], func(x, y types.EquipmentCreation) int {
			return cmp.Compare(x.TimestampUS, y.TimestampUS)
		})
	}
	return out
}

// positionDuPorteur rend le lieu du chassis qui porte la piece `piece` a `tUS`, par son echantillon a la
// tolerance : le voisin de slot n est qu un repli, nomme, compte a chaque lieu qu il decide.
func (b bordDuFilm) positionDuPorteur(piece types.EquipmentCreation, spec vehicleTurret, tUS uint64) (*Vec3, bool) {
	creationA := func(slot uint32) (types.EquipmentCreation, bool) { return b.creationA(slot, tUS) }
	porteur, ok := porteurDUnePiece(creationA, piece, spec)
	if !ok {
		return nil, false
	}
	v, ok := b.echantillonDeVehicule(porteur.Slot, tUS)
	if !ok {
		return nil, false
	}
	b.fb.Declenche(fallback.NomTourellePorteurVoisinDeSlot)
	return v, true
}
