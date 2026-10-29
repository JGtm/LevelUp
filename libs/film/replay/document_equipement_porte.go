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
// LA POSE DE LA FAILLE est l'instant où une lecture de l'objet porté du TRANSLOCATEUR, postérieure à
// sa création, écrit une énergie nulle : le jeu consomme son unique charge (coût 1). La règle ne vaut
// que pour cette famille (une autre famille épuisée ne pose rien). Quand l'activation est lue, elle
// la précède de ~1,63 s (123 sur 123 au parc) et `a` la publie ; elle n'est pas exigée, parce qu'une
// trame que la marche ne ferme pas la perd (Live Fire).
//
// LA POSITION DE LA FAILLE EST CELLE DE L'OBJET QUE LA POSE CRÉE. Le paquet de la pose crée la
// faille (une entité ti=38, que la pile i28 de l'objet porté nomme au paquet suivant) : sa position
// i0 à la création est l'emplacement de la faille (`src` = « objet », 123 sur 123 au parc). Repli,
// nommé et compté : la position du porteur lue dans sa piste à l'image même de la pose (`src` =
// « porteur ») ; sans l'une ni l'autre, la pose est publiée sans position. Aucune position ne vient
// d'un échange : la faille existe, à sa place, dès la pose.
//
// LA FIN DE LA FAILLE EST LA SUPPRESSION LUE DE CET OBJET : la pile i28 de l objet porte qui le suivait
// ecrit sa fin (`tracked-end`), et la pose la publie (`t1`). Sans elle, le client clot la faille par
// son repli (consommation de l equipement ou mort du porteur), compte dans `riftEndsUnread`.

import (
	"cmp"
	"context"
	"log/slog"
	"math"
	"slices"

	"github.com/JGtm/LevelUp/libs/film/equipenergy"
	"github.com/JGtm/LevelUp/libs/film/internal/facts/fallback"
	"github.com/JGtm/LevelUp/libs/film/internal/grammar"
	"github.com/JGtm/LevelUp/libs/film/types"
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
	// Tracked : les objets que l'équipement a engendrés et suivis (pile i28 : le capteur, le traqueur,
	// l'écran, le champ de réparation, la faille...), dans l'ordre de leur apparition.
	Tracked []CarriedTracked `json:"tracked,omitempty"`
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
	// X/Y/Z : la position de la faille, SOLIDAIRES ; Src dit d'où elle vient : « objet » (la faille
	// créée par la pose) ou « porteur » (sa piste à l'image de la pose). Absents ensemble sinon.
	X   *float32 `json:"x,omitempty"`
	Y   *float32 `json:"y,omitempty"`
	Z   *float32 `json:"z,omitempty"`
	Src string   `json:"src,omitempty"`
	// T1 est la frame de la FIN de la faille : la suppression lue de l objet que la pose cree. Absente
	// quand le film ne la dit pas (objet non lu, suppression non lue) : le client clot alors la faille
	// par son repli, la consommation de l equipement ou la mort du porteur.
	T1 *int `json:"t1,omitempty"`
}

// Les deux sources de la position d'une faille.
const (
	sourceFailleObjet   = "objet"
	sourceFaillePorteur = "porteur"
)

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
	// RiftPoses : poses de faille publiées ; PositionedByObject / PositionedByCarrier : situées par la
	// faille créée, par le repli du porteur ; Unpositioned : sans position ; TeleportsWithoutPose :
	// échanges (i22) d'un translocateur dont aucune pose n'a été lue.
	RiftPoses            int `json:"riftPoses"`
	PositionedByObject   int `json:"positionedByObject"`
	PositionedByCarrier  int `json:"positionedByCarrier"`
	Unpositioned         int `json:"unpositioned"`
	TeleportsWithoutPose int `json:"teleportsWithoutPose"`
	// TrackedObjects : objets engendrés publiés ; TrackedEnds : ceux dont la suppression est lue.
	TrackedObjects int `json:"trackedObjects"`
	TrackedEnds    int `json:"trackedEnds"`
	// TrackedPositioned : objets engendrés dont une trajectoire publiée porte la vie (position de
	// repos publiée) ; TrackedUnpositioned : les autres, publiés sans position.
	TrackedPositioned   int `json:"trackedPositioned"`
	TrackedUnpositioned int `json:"trackedUnpositioned"`
	// TrackedEffects : objets engendres de l ecran occultant dont l objet d effet (ti=38) est publie ;
	// TrackedEffectEnds : ceux dont la suppression est lue ; TrackedEffectsAmbiguous : plusieurs candidates
	// retenues, rien n est publie ; TrackedEffectsUnlinked : aucune candidate retenue (cf. `document_effets.go`).
	TrackedEffects          int `json:"trackedEffects"`
	TrackedEffectEnds       int `json:"trackedEffectEnds"`
	TrackedEffectsAmbiguous int `json:"trackedEffectsAmbiguous"`
	TrackedEffectsUnlinked  int `json:"trackedEffectsUnlinked"`
	// RiftEndsRead : poses dont la fin (suppression de l objet cree) est lue ; RiftEndsUnread : poses
	// sans fin lue, que le client clot par son repli (consommation ou mort du porteur).
	RiftEndsRead   int `json:"riftEndsRead"`
	RiftEndsUnread int `json:"riftEndsUnread"`
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
//
// UNE CRÉATION SANS ÉNERGIE ÉCRITE NE CHOISIT RIEN. Elle garde l'énergie par défaut de l'objet PORTÉ,
// pas celle du ramassable : sur `5676a9ba`, les répulseurs (ramassable 0,6) et les champs de
// réparation (ramassable 0,5) créés sans énergie lisent 0,8 et 0,5 à leur premier usage, soit 1 au
// départ. Les deux fiches du mur ont la même énergie portée (1) : rien ne les sépare.
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
	reads        []types.CarriedEquipmentRead
	st           types.CarriedEquipmentStats
	changes      []types.EquipmentChange
	palette      *AbilityPalette
	couts        map[string][]coutDeFamille
	tracks       []Track
	projectiles  []types.ProjectileTrack
	origin, step uint64
}

// objetLu : un objet porté pendant l'assemblage.
type objetLu struct {
	pub      CarriedEquipment
	famille  string
	cout     coutDeFamille
	coutOK   bool
	lectures []types.CarriedEquipmentRead
	// suivis : l'indice, dans pub.Tracked, de chaque objet engendré nommé par la pile i28.
	suivis map[types.LifeKey]int
	// effets : les candidates a l objet d effet, par vie suivie ; finsDEffet : la suppression lue de chaque vie d effet.
	effets     map[types.LifeKey][]effetCandidat
	finsDEffet map[types.LifeKey][]finDEffet
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
	repos := reposParVie(in.projectiles, in.origin, in.step)
	fins := finsDEffetLues(in.reads, in)
	for _, o := range regrouperParObjet(in.reads) {
		o.finsDEffet = fins
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
		o.situerLesObjetsEngendres(repos, &cov)
		o.situerLesEffets(&cov)
		out = append(out, o.pub)
		cov.Published++
	}
	trierLesPoses(poses)
	cov.RiftPoses = len(poses)
	for _, p := range poses {
		switch p.Src {
		case sourceFailleObjet:
			cov.PositionedByObject++
		case sourceFaillePorteur:
			cov.PositionedByCarrier++
		default:
			cov.Unpositioned++
		}
		if p.T1 != nil {
			cov.RiftEndsRead++
		} else {
			cov.RiftEndsUnread++
		}
	}
	return out, poses, cov
}

// projeter publie les lectures d'un objet sur l'axe de frames, et rend ses poses de faille.
func (o *objetLu) projeter(in entreeDEquipementPorte, cov *CarriedEquipmentCoverage) []RiftPose { //nolint:gocyclo // une passe sur les lectures d un objet, un test par sorte de lecture (energie initiale, suppression, effet, activation, energie, pose du translocateur, signal sans pose) : la complexite compte les sortes de lecture, pas des chemins imbriques
	translocateur := o.famille == familleDuTranslocateur
	var poses []RiftPose
	var instants []uint64
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
		if l.Kind == grammar.CarriedEffectBirth || l.Kind == grammar.CarriedEffectEnd {
			o.noterLEffet(l, in)
			continue
		}
		o.suivre(l, t, cov)
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
				instants = append(instants, l.TimestampUS)
				poseLue = true
			}
		}
		if translocateur && l.HasSignal && !poseLue {
			cov.TeleportsWithoutPose++
		}
	}
	o.situerLesPoses(poses, instants, in.tracks)
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

// regrouperParObjet range les lectures par vie d'objet, chaque vie ouverte par sa création, dans
// l'ordre de création. Une lecture sans création lue n'est pas publiée. La fin d'un objet engendré
// (`tracked-end`) revient à la dernière vie de l'objet porté qui le suivait, même supprimée : un
// champ de réparation survit à son porteur.
func regrouperParObjet(reads []types.CarriedEquipmentRead) []*objetLu {
	var out []*objetLu
	courants := map[types.LifeKey]*objetLu{}
	derniers := map[types.LifeKey]*objetLu{}
	for _, l := range reads {
		switch l.Kind {
		case types.CarriedNew:
			o := &objetLu{lectures: []types.CarriedEquipmentRead{l}}
			courants[l.Object], derniers[l.Object] = o, o
			out = append(out, o)
		case types.CarriedTrackedEnd, grammar.CarriedEffectBirth, grammar.CarriedEffectEnd:
			if o, ok := derniers[l.Object]; ok {
				o.lectures = append(o.lectures, l)
			}
		default:
			if o, ok := courants[l.Object]; ok {
				o.lectures = append(o.lectures, l)
				if l.Kind == types.CarriedDel {
					delete(courants, l.Object)
				}
			}
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
		"posesDeFaille", cov.RiftPoses, "situeesParLObjet", cov.PositionedByObject,
		"situeesParLePorteur", cov.PositionedByCarrier, "sansPosition", cov.Unpositioned,
		"echangesSansPose", cov.TeleportsWithoutPose,
		"objetsEngendres", cov.TrackedObjects, "finsLues", cov.TrackedEnds,
		"objetsSitues", cov.TrackedPositioned, "objetsSansPosition", cov.TrackedUnpositioned,
		"finsDeFailleLues", cov.RiftEndsRead, "finsDeFailleNonLues", cov.RiftEndsUnread,
		"effets", cov.TrackedEffects, "finsDEffetLues", cov.TrackedEffectEnds,
		"effetsAmbigus", cov.TrackedEffectsAmbiguous, "effetsSansLien", cov.TrackedEffectsUnlinked)
}

// poserLEquipementPorte publie l'équipement porté et les poses de faille, nommés par la palette classée
// et comptés par le catalogue de l'énergie des équipements.
func (a *assemblage) poserLEquipementPorte() {
	carried, poses, cov := buildCarriedEquipment(entreeDEquipementPorte{
		reads: a.opt.CarriedEquipment, st: a.opt.CarriedEquipmentStats, changes: a.opt.EquipmentChanges,
		palette: a.palette, couts: coutsParFamille(a.opt.EquipmentEnergy, a.opt.Labels.EquipmentFamilies),
		tracks: a.doc.Tracks, projectiles: a.opt.Projectiles, origin: a.origin, step: a.step,
	})
	a.doc.CarriedEquipment, a.doc.RiftPoses = carried, poses
	a.doc.Coverage.CarriedEquipment = &cov
	a.opt.Fallbacks.DeclencheN(fallback.NomFinDeFailleConsommationOuMortDuPorteur, cov.RiftEndsUnread)
	a.opt.Fallbacks.DeclencheN(fallback.NomEffetDEcranParLaFinDeLaGrenade, cov.TrackedEffects)
	logCarriedEquipmentCoverage(a.ctx, cov)
}

// suivre publie les objets engendrés qu'une lecture nomme (pile i28), et date la fin de celui dont
// elle lit la suppression.
func (o *objetLu) suivre(l types.CarriedEquipmentRead, t int, cov *CarriedEquipmentCoverage) {
	if o.suivis == nil {
		o.suivis = map[types.LifeKey]int{}
	}
	if l.Kind == types.CarriedTrackedEnd {
		for _, s := range l.Tracked {
			if k, ok := o.suivis[s]; ok && o.pub.Tracked[k].T1 == nil {
				o.pub.Tracked[k].T1 = &t
				cov.TrackedEnds++
			}
		}
		return
	}
	for _, s := range l.Tracked {
		if _, ok := o.suivis[s]; !ok {
			o.suivis[s] = len(o.pub.Tracked)
			o.pub.Tracked = append(o.pub.Tracked, CarriedTracked{T0: t})
			cov.TrackedObjects++
		}
	}
}
