package replay

// document_equipement_porte.go — L ÉQUIPEMENT PORTÉ : son énergie, ses usages restants et la pose de
// la faille du translocateur, lus dans le film (schéma 95).
//
// CE QUE LE FILM ÉCRIT (grammar/equipement_porte.go). Quand un joueur prend un équipement, le jeu
// crée un objet ti=37 attaché à lui, dans le MÊME paquet que l'annonce i48 du porteur (`taken` ou la
// réapparition) : le rang de cette annonce nomme la famille de l'objet, par la palette du film. Tant
// qu'il est porté, l'objet écrit son ÉNERGIE (i24) quand elle change, une ACTIVATION (i21) à chaque
// usage, et sa suppression clôt le port.
//
// LES USAGES RESTANTS SONT LA RÈGLE DU JEU (`FUN_1407e9384`), appliquée à l'énergie lue :
// trunc((énergie + 1e-4) / coût), le coût étant la capacité de l'objet porté, lue dans les tags du
// jeu par le catalogue de l'énergie des équipements (`equipenergy`). Une famille dont les fiches du
// catalogue ne s'accordent pas sur le coût — ou absente du catalogue — ne reçoit aucun compte : elle
// se compte dans la couverture, jamais devinée.
//
// LA POSE DE LA FAILLE est l'instant où l'énergie de l'objet porté du translocateur tombe à zéro : le
// jeu consomme son unique charge (coût 1). Elle suit toujours une activation (~1,63 s plus tôt, mesure
// du parc), et elle précède le premier échange (`translocations[]`). Sa position n'est lue que par ce
// premier échange : la faille est là où le joueur arrive. Une pose jamais suivie d'un échange est
// publiée sans position.

import (
	"cmp"
	"context"
	"log/slog"
	"math"
	"slices"

	"levelup/go-api/internal/games/halo_infinite/equipenergy"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// familleDuTranslocateur est la famille de la palette dont la pose se publie.
const familleDuTranslocateur = "translocator_beacon"

// CarriedEquipment est UN objet d'équipement porté, de sa prise à sa fin.
type CarriedEquipment struct {
	// Slot désigne la Track du porteur ; R le rang de palette de l'équipement (AbilityLabels).
	Slot uint32 `json:"slot"`
	R    int    `json:"r"`
	// T0 est la frame de la prise ; T1 celle de la fin de l'objet, absente quand le film ne la dit pas.
	T0 int  `json:"t0"`
	T1 *int `json:"t1,omitempty"`
	// Energy : l'énergie de l'objet, à chaque lecture, et les usages restants quand le coût est connu.
	Energy []CarriedEnergy `json:"energy,omitempty"`
	// Uses : les frames des activations (i21).
	Uses []int `json:"uses,omitempty"`
}

// CarriedEnergy est une lecture de l'énergie d'un objet porté.
type CarriedEnergy struct {
	T int     `json:"t"`
	E float32 `json:"e"`
	// N : les usages restants (règle du jeu), absent quand le coût de la famille n'est pas établi.
	N *int `json:"n,omitempty"`
	// Tag : l'énergie n'est pas écrite par le film à la prise — elle vaut l'énergie initiale de
	// l'objet porté, lue dans le catalogue.
	Tag bool `json:"tag,omitempty"`
}

// RiftPose est la POSE d'une faille du translocateur.
type RiftPose struct {
	// T est la frame de la pose ; A celle de l'activation qui la précède, quand elle est lue.
	T int  `json:"t"`
	A *int `json:"a,omitempty"`
	// Slot désigne la Track du porteur.
	Slot uint32 `json:"slot"`
	// X/Y/Z : la position de la faille, celle de l'arrivée du premier échange qui suit la pose ;
	// SOLIDAIRES, absents quand aucun échange ne suit.
	X *float32 `json:"x,omitempty"`
	Y *float32 `json:"y,omitempty"`
	Z *float32 `json:"z,omitempty"`
}

// CarriedEquipmentCoverage dit ce que le calque a lu et ce qu'il a écarté.
type CarriedEquipmentCoverage struct {
	// Created, Attached : objets ti=37 créés, et ceux créés attachés à un bipède (lus).
	Created  int `json:"created"`
	Attached int `json:"attached"`
	// Published : objets publiés. NoIdentity : porteur sans piste publiée ; BeforeOrigin : pris
	// avant la première frame ; WithoutRank : aucune annonce i48 du porteur dans le paquet ;
	// AmbiguousRank : plusieurs objets créés pour le même porteur dans le paquet ; Unnamed : rang
	// sans famille dans la palette du film.
	Published     int `json:"published"`
	NoIdentity    int `json:"noIdentity"`
	BeforeOrigin  int `json:"beforeOrigin"`
	WithoutRank   int `json:"withoutRank"`
	AmbiguousRank int `json:"ambiguousRank"`
	Unnamed       int `json:"unnamed"`
	// WithCost : objets publiés dont le coût par usage est établi ; NoCatalog : famille absente du
	// catalogue ; AmbiguousCost : fiches de la famille en désaccord sur le coût.
	WithCost      int `json:"withCost"`
	NoCatalog     int `json:"noCatalog"`
	AmbiguousCost int `json:"ambiguousCost"`
	// EnergyReads, Uses : lectures d'énergie et activations publiées.
	EnergyReads int `json:"energyReads"`
	Uses        int `json:"uses"`
	// RiftPoses : poses de faille publiées ; Positioned : celles qui portent une position ;
	// TeleportsWithoutPose : échanges (i22) d'un translocateur dont aucune pose n'a été lue.
	RiftPoses            int `json:"riftPoses"`
	Positioned           int `json:"positioned"`
	TeleportsWithoutPose int `json:"teleportsWithoutPose"`
}

// coutDeFamille : le coût par usage d'une fiche du catalogue, l'énergie initiale de son objet porté
// et celle de son objet ramassable.
type coutDeFamille struct {
	seuil, variation, energiePorte, energieRamassable float32
}

// coutsParFamille indexe les fiches DISTINCTES du catalogue par famille (par l'objet ramassable que
// le manifeste nomme).
func coutsParFamille(cat *equipenergy.Catalog, familles map[uint32]string) map[string][]coutDeFamille {
	out := map[string][]coutDeFamille{}
	if cat == nil {
		return out
	}
	for _, e := range cat.Entries {
		id, ok := e.PickupID()
		if !ok {
			continue
		}
		f, ok := familles[id]
		if !ok {
			continue
		}
		c := coutDeFamille{seuil: e.Threshold, variation: e.Delta, energiePorte: e.CarriedEnergy,
			energieRamassable: e.PickupEnergy}
		if !slices.Contains(out[f], c) {
			out[f] = append(out[f], c)
		}
	}
	return out
}

// pasDEnergie est le pas de la quantification de l'énergie (i24, 14 bits aux extrémités exactes).
const pasDEnergie = 1.0 / 16382

// choisirLeCout rend la fiche d'une famille : la seule, ou, quand la famille en a plusieurs, celle
// dont l'énergie de l'objet ramassable est l'énergie écrite à la prise (au pas de quantification près).
func choisirLeCout(cands []coutDeFamille, creation types.CarriedEquipmentRead) (coutDeFamille, bool) {
	if len(cands) == 1 {
		return cands[0], true
	}
	if !creation.HasEnergy {
		return coutDeFamille{}, false
	}
	e := float64(grammar.EnergieDEquipement(creation.Energy))
	var pris []coutDeFamille
	for _, c := range cands {
		if math.Abs(float64(c.energieRamassable)-e) <= pasDEnergie {
			pris = append(pris, c)
		}
	}
	if len(pris) != 1 {
		return coutDeFamille{}, false
	}
	return pris[0], true
}

// entreeDEquipementPorte : ce que l'assemblage du calque reçoit.
type entreeDEquipementPorte struct {
	reads          []types.CarriedEquipmentRead
	st             types.CarriedEquipmentStats
	changes        []types.EquipmentChange
	palette        *AbilityPalette
	couts          map[string][]coutDeFamille
	tracks         []Track
	translocations []Translocation
	origin, step   uint64
}

// objetLu : un objet porté pendant l'assemblage.
type objetLu struct {
	pub      CarriedEquipment
	famille  string
	cout     coutDeFamille
	coutOK   bool
	lectures []types.CarriedEquipmentRead
}

// buildCarriedEquipment assemble le calque de l'équipement porté et les poses de faille.
func buildCarriedEquipment(in entreeDEquipementPorte) ([]CarriedEquipment, []RiftPose, CarriedEquipmentCoverage) {
	cov := CarriedEquipmentCoverage{Created: in.st.Created, Attached: in.st.Attached}
	if in.step == 0 {
		return nil, nil, cov
	}
	idx := indexDesRangs{rangs: rangsParPaquet(in.changes), nouveaux: nouveauxParPaquet(in.reads),
		published: publishedSlots(in.tracks)}
	var out []CarriedEquipment
	var poses []RiftPose
	for _, o := range regrouperParObjet(in.reads) {
		creation := o.lectures[0]
		rang, ok := idx.rangDe(creation, in.origin, in.palette, &cov)
		if !ok {
			continue
		}
		o.famille = in.palette.FamilyOf(rang)
		cands := in.couts[o.famille]
		o.cout, o.coutOK = choisirLeCout(cands, creation)
		switch {
		case len(cands) == 0:
			cov.NoCatalog++
		case !o.coutOK:
			cov.AmbiguousCost++
		default:
			cov.WithCost++
		}
		o.pub = CarriedEquipment{Slot: creation.Carrier, R: rang, T0: in.frame(creation.TimestampUS)}
		poses = append(poses, o.projeter(in, &cov)...)
		out = append(out, o.pub)
		cov.Published++
	}
	trierLesPoses(poses)
	cov.RiftPoses = len(poses)
	for _, p := range poses {
		if p.X != nil {
			cov.Positioned++
		}
	}
	return out, poses, cov
}

// indexDesRangs : les annonces i48 par paquet et porteur, les créations par paquet et porteur, et les
// pistes publiées.
type indexDesRangs struct {
	rangs     map[paquetDuPorteur]int
	nouveaux  map[paquetDuPorteur]int
	published map[uint32]bool
}

// rangDe rend le rang de l'objet créé par `creation` : celui de l'annonce i48 de son porteur dans le
// même paquet, quand l'objet y est seul et que la palette du film le nomme. Un écart se compte.
func (x indexDesRangs) rangDe(creation types.CarriedEquipmentRead, origin uint64, palette *AbilityPalette,
	cov *CarriedEquipmentCoverage) (int, bool) {
	cle := paquetDuPorteur{creation.TimestampUS, creation.Carrier}
	rang, annonce := x.rangs[cle]
	switch {
	case creation.TimestampUS < origin:
		cov.BeforeOrigin++
	case !x.published[creation.Carrier]:
		cov.NoIdentity++
	case !annonce:
		cov.WithoutRank++
	case x.nouveaux[cle] > 1:
		cov.AmbiguousRank++
	case palette.FamilyOf(rang) == "":
		cov.Unnamed++
	default:
		return rang, true
	}
	return 0, false
}

// frame rend la frame d'un instant du film.
func (in entreeDEquipementPorte) frame(ts uint64) int {
	return int((ts - in.origin) / in.step) //nolint:gosec // instants posterieurs a l origine
}

// projeter publie les lectures d'un objet sur l'axe de frames, et rend ses poses de faille.
func (o *objetLu) projeter(in entreeDEquipementPorte, cov *CarriedEquipmentCoverage) []RiftPose {
	translocateur := o.famille == familleDuTranslocateur
	var poses []RiftPose
	var activation *int
	poseLue := false
	for i, l := range o.lectures {
		t := in.frame(l.TimestampUS)
		if i == 0 && !l.HasEnergy && o.coutOK {
			o.ajouterEnergie(t, o.cout.energiePorte, true, cov)
		}
		if l.Kind == types.CarriedDel {
			o.pub.T1 = &t
		}
		if l.HasActivation {
			o.pub.Uses = append(o.pub.Uses, t)
			cov.Uses++
			a := t
			activation = &a
		}
		if l.HasEnergy {
			e := grammar.EnergieDEquipement(l.Energy)
			o.ajouterEnergie(t, e, false, cov)
			if translocateur && l.Energy == 0 && i > 0 {
				poses = append(poses, RiftPose{T: t, A: activation, Slot: o.pub.Slot})
				poseLue = true
			}
		}
		if translocateur && l.HasSignal && !poseLue {
			cov.TeleportsWithoutPose++
		}
	}
	for k := range poses {
		positionnerLaPose(&poses[k], o.pub.T1, in.translocations)
	}
	return poses
}

// ajouterEnergie publie une lecture d'énergie, avec les usages restants quand le coût est établi.
func (o *objetLu) ajouterEnergie(t int, e float32, tag bool, cov *CarriedEquipmentCoverage) {
	r := CarriedEnergy{T: t, E: float32(math.Round(float64(e)*1e4) / 1e4), Tag: tag}
	if o.coutOK {
		if n, ok := equipenergy.UsagesRestants(e, o.cout.seuil, o.cout.variation); ok {
			r.N = &n
		}
	}
	o.pub.Energy = append(o.pub.Energy, r)
	if !tag {
		cov.EnergyReads++
	}
}

// positionnerLaPose pose la position de la faille : l'arrivée du premier échange du porteur qui suit
// la pose, avant la fin de l'objet.
func positionnerLaPose(p *RiftPose, fin *int, translocations []Translocation) {
	for _, tr := range translocations {
		if tr.Slot != p.Slot || tr.T < p.T || (fin != nil && tr.T > *fin) {
			continue
		}
		if tr.TX != nil && tr.TY != nil && tr.TZ != nil {
			p.X, p.Y, p.Z = tr.TX, tr.TY, tr.TZ
		}
		return
	}
}

// regrouperParObjet range les lectures par vie d'objet, chaque vie ouverte par sa création, dans
// l'ordre de création. Une lecture sans création lue n'est pas publiée.
func regrouperParObjet(reads []types.CarriedEquipmentRead) []*objetLu {
	var out []*objetLu
	courants := map[types.LifeKey]*objetLu{}
	for _, l := range reads {
		if l.Kind == types.CarriedNew {
			o := &objetLu{lectures: []types.CarriedEquipmentRead{l}}
			courants[l.Object] = o
			out = append(out, o)
			continue
		}
		if o, ok := courants[l.Object]; ok {
			o.lectures = append(o.lectures, l)
			if l.Kind == types.CarriedDel {
				delete(courants, l.Object)
			}
		}
	}
	return out
}

// paquetDuPorteur : un paquet (son instant) et un porteur.
type paquetDuPorteur struct {
	ts   uint64
	slot uint32
}

// rangsParPaquet indexe le rang des annonces i48 qui donnent un équipement (prise ou réapparition).
func rangsParPaquet(changes []types.EquipmentChange) map[paquetDuPorteur]int {
	out := map[paquetDuPorteur]int{}
	for _, c := range changes {
		if c.Kind == types.EquipmentTaken || c.Kind == types.EquipmentSpawned {
			out[paquetDuPorteur{c.TimestampUS, c.Slot}] = c.Rank
		}
	}
	return out
}

// nouveauxParPaquet compte les objets créés attachés par paquet et par porteur.
func nouveauxParPaquet(reads []types.CarriedEquipmentRead) map[paquetDuPorteur]int {
	out := map[paquetDuPorteur]int{}
	for _, l := range reads {
		if l.Kind == types.CarriedNew {
			out[paquetDuPorteur{l.TimestampUS, l.Carrier}]++
		}
	}
	return out
}

// trierLesPoses ordonne les poses par frame puis par porteur.
func trierLesPoses(p []RiftPose) {
	slices.SortStableFunc(p, func(a, b RiftPose) int {
		return cmp.Or(cmp.Compare(a.T, b.T), cmp.Compare(a.Slot, b.Slot))
	})
}

// logCarriedEquipmentCoverage sort la couverture du calque.
func logCarriedEquipmentCoverage(ctx context.Context, cov CarriedEquipmentCoverage) {
	slog.InfoContext(ctx, "rejeu : equipement porte",
		"crees", cov.Created, "attaches", cov.Attached, "publies", cov.Published, "sansIdentite", cov.NoIdentity,
		"avantOrigine", cov.BeforeOrigin, "sansRang", cov.WithoutRank, "rangAmbigu", cov.AmbiguousRank,
		"nonNommes", cov.Unnamed, "avecCout", cov.WithCost, "horsCatalogue", cov.NoCatalog,
		"coutAmbigu", cov.AmbiguousCost, "lecturesEnergie", cov.EnergyReads, "usages", cov.Uses,
		"posesDeFaille", cov.RiftPoses, "posesSituees", cov.Positioned, "echangesSansPose", cov.TeleportsWithoutPose)
}

// poserLEquipementPorte publie l'équipement porté et les poses de faille, nommés par la palette classée
// et comptés par le catalogue de l'énergie des équipements.
func (a *assemblage) poserLEquipementPorte() {
	carried, poses, cov := buildCarriedEquipment(entreeDEquipementPorte{
		reads: a.opt.CarriedEquipment, st: a.opt.CarriedEquipmentStats, changes: a.opt.EquipmentChanges,
		palette: a.palette, couts: coutsParFamille(a.opt.EquipmentEnergy, a.opt.Labels.EquipmentFamilies),
		tracks: a.doc.Tracks, translocations: a.doc.Translocations, origin: a.origin, step: a.step,
	})
	a.doc.CarriedEquipment, a.doc.RiftPoses = carried, poses
	a.doc.Coverage.CarriedEquipment = &cov
	logCarriedEquipmentCoverage(a.ctx, cov)
}
