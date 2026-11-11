package replay

// document_equipement_porte_identite.go — QUEL EQUIPEMENT EST CET OBJET PORTE : la famille et le cout par usage
// d un objet, lus sur l IDENTITE QUE LE FILM ECRIT A SA CREATION (le GlobalID de son tag `eqip`,
// `types.CarriedEquipmentRead.GlobalID`), jamais deduits d une annonce du porteur.
//
// POURQUOI L OBJET ET NON L ANNONCE. Le rang de l annonce i48 du porteur nomme sa capacite, mais l objet porte
// n est joignable a cette annonce que lorsqu il est cree DANS le paquet de la prise, seul. A la reapparition le jeu
// cree deux objets attaches dans le meme paquet, et l equipement de depart d un match est cree avant l origine du
// film : aucune annonce ne le designe. L identite de l objet, elle, est ecrite pour tous, quelle que soit la facon
// dont le joueur l a obtenu.
//
// DEUX SOURCES, QUI NE SE CONTREDISENT PAS :
//   - le catalogue de l energie des equipements (`equipenergy`) relie l objet PORTE (`carried`) aux objets
//     RAMASSABLES qui le donnent (`pickup`) ; la famille de l objet est celle de ses ramassables que le manifeste
//     du titre connait. Plusieurs ramassables donnent un seul objet porte (le mur en a deux) : ils doivent
//     s accorder, sinon l objet n a pas de famille ;
//   - le manifeste (`[[equipment_objects]]`, `LabelCatalog.EquipmentFamilies`) nomme directement certains objets
//     portes (le grappin, le bonus de camouflage) : c est la famille de l identifiant, quand le catalogue n en dit
//     rien. Les deux sources qui parlent doivent s accorder.
//
// CE QUI NE SE PUBLIE JAMAIS : une grenade attachee (famille nommee par `usageGrenadeFamilies` : ce n est pas un
// equipement), et un identifiant que ni le catalogue ni le manifeste ne resolvent. Il se compte en couverture
// (`grenades`, `unresolved`, `conflicting`) ; il n est jamais devine, et aucun repli sur le rang d une annonce ne
// le sauve.
//
// LE COUT PAR USAGE vient du MEME objet porte : la fiche du catalogue dont `carried` est son identifiant. Plusieurs
// fiches pour un meme objet porte (un mur a deux ramassables) portent le meme cout ; si elles n en portaient pas
// le meme, l objet n aurait pas de compte (`ambiguousCost`).

import (
	"slices"

	"github.com/JGtm/halo-infinite-film-decoder/equipenergy"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// rangInconnu est le rang publie d un objet dont la famille n a aucun rang dans la palette du film (palette non
// classee, ou famille que cette palette ne contient pas) : l objet est publie, sa famille se lit dans `family`.
const rangInconnu = -1

// coutDObjetPorte : le cout par usage d un objet porte et son energie initiale, lus dans sa fiche du catalogue.
type coutDObjetPorte struct {
	seuil, variation, energiePorte float32
}

// ficheDObjetPorte : ce que le catalogue dit d un objet porte — les objets ramassables qui le donnent, et ses couts
// distincts.
type ficheDObjetPorte struct {
	ramassables []uint32
	couts       []coutDObjetPorte
}

// referenceDesObjetsPortes joint un identifiant d objet porte a sa famille et a son cout.
type referenceDesObjetsPortes struct {
	// familles : le manifeste, identifiant d objet -> famille.
	familles map[uint32]string
	// fiches : le catalogue de l energie, par identifiant d objet PORTE.
	fiches map[uint32]*ficheDObjetPorte
}

// nouvelleReferenceDesObjetsPortes indexe le catalogue par objet porte. Un catalogue nil donne une reference sans
// fiche : seules les familles du manifeste se resolvent, et aucun cout n est connu.
func nouvelleReferenceDesObjetsPortes(cat *equipenergy.Catalog, familles map[uint32]string) *referenceDesObjetsPortes {
	r := &referenceDesObjetsPortes{familles: familles, fiches: map[uint32]*ficheDObjetPorte{}}
	if cat == nil {
		return r
	}
	for _, e := range cat.Entries {
		id, ok := e.CarriedID()
		if !ok {
			continue
		}
		f := r.fiches[id]
		if f == nil {
			f = &ficheDObjetPorte{}
			r.fiches[id] = f
		}
		if p, ok := e.PickupID(); ok && !slices.Contains(f.ramassables, p) {
			f.ramassables = append(f.ramassables, p)
		}
		c := coutDObjetPorte{seuil: e.Threshold, variation: e.Delta, energiePorte: e.CarriedEnergy}
		if !slices.Contains(f.couts, c) {
			f.couts = append(f.couts, c)
		}
	}
	return r
}

// issueDIdentite : ce que la reference dit d un identifiant.
type issueDIdentite int

const (
	// identiteResolue : une famille, publiable.
	identiteResolue issueDIdentite = iota
	// identiteGrenade : une grenade attachee, jamais publiee.
	identiteGrenade
	// identiteInconnue : ni le catalogue ni le manifeste ne la resolvent.
	identiteInconnue
	// identiteEnDesaccord : les sources qui parlent nomment des familles differentes.
	identiteEnDesaccord
)

// famille rend la famille de l objet porte `id` : l union des familles de ses ramassables connus du manifeste et
// de celle que le manifeste donne a l identifiant lui-meme. Une seule famille = resolue ; aucune = inconnue ;
// plusieurs = desaccord, rien n est publie.
func (r *referenceDesObjetsPortes) famille(id uint32) (string, issueDIdentite) {
	if r == nil {
		return "", identiteInconnue
	}
	var vues []string
	ajouter := func(f string) {
		if !slices.Contains(vues, f) {
			vues = append(vues, f)
		}
	}
	if fiche := r.fiches[id]; fiche != nil {
		for _, p := range fiche.ramassables {
			if f, ok := r.familles[p]; ok {
				ajouter(f)
			}
		}
	}
	if f, ok := r.familles[id]; ok {
		ajouter(f)
	}
	switch {
	case len(vues) == 0:
		return "", identiteInconnue
	case len(vues) > 1:
		return "", identiteEnDesaccord
	case usageGrenadeFamilies[vues[0]]:
		return vues[0], identiteGrenade
	}
	return vues[0], identiteResolue
}

// statutDuCout : ce que le catalogue dit du cout d un objet porte.
type statutDuCout int

const (
	// coutEtabli : une fiche, ou des fiches de meme cout.
	coutEtabli statutDuCout = iota
	// coutSansFiche : l objet porte n est pas au catalogue.
	coutSansFiche
	// coutAmbigu : des fiches d un meme objet porte en desaccord sur le cout.
	coutAmbigu
)

// cout rend le cout par usage de l objet porte `id`.
func (r *referenceDesObjetsPortes) cout(id uint32) (coutDObjetPorte, statutDuCout) {
	var fiche *ficheDObjetPorte
	if r != nil {
		fiche = r.fiches[id]
	}
	switch {
	case fiche == nil || len(fiche.couts) == 0:
		return coutDObjetPorte{}, coutSansFiche
	case len(fiche.couts) > 1:
		return coutDObjetPorte{}, coutAmbigu
	}
	return fiche.couts[0], coutEtabli
}

// identifier decide si l objet se publie et pose sa famille, son rang de palette et son cout ; chaque refus se
// compte. L ordre des gardes est celui de la nature du refus : d abord ce que l objet EST (identifiant, famille),
// puis ce qui empeche de le montrer (porteur sans piste publiee, objet fini avant l origine).
func (o *objetLu) identifier(in entreeDEquipementPorte, publiees map[uint32]bool, cov *CarriedEquipmentCoverage) bool {
	creation := o.lectures[0]
	if !creation.HasGlobalID {
		cov.NoGlobalID++
		return false
	}
	famille, issue := in.refs.famille(creation.GlobalID)
	switch issue {
	case identiteGrenade:
		cov.Grenades++
	case identiteInconnue:
		cov.Unresolved++
	case identiteEnDesaccord:
		cov.Conflicting++
	}
	switch {
	case issue != identiteResolue:
		return false
	case !publiees[creation.Carrier]:
		cov.NoIdentity++
		return false
	case o.finiAvantLOrigine(in):
		cov.EndedBeforeOrigin++
		return false
	}
	o.famille = famille
	o.cout, o.coutStatut = in.refs.cout(creation.GlobalID)
	switch o.coutStatut {
	case coutSansFiche:
		cov.NoCatalog++
	case coutAmbigu:
		cov.AmbiguousCost++
	default:
		cov.WithCost++
	}
	rang, ok := in.palette.RankOf(famille)
	if !ok {
		rang = rangInconnu
	}
	if in.avantLOrigine(creation.TimestampUS) {
		cov.BeforeOrigin++
	}
	o.pub = CarriedEquipment{Slot: creation.Carrier, R: rang, Family: famille, T0: in.frame(creation.TimestampUS)}
	return true
}

// finiAvantLOrigine dit si le film date la suppression de l objet AVANT la premiere frame : il n a alors aucune
// existence sur l axe publie.
func (o *objetLu) finiAvantLOrigine(in entreeDEquipementPorte) bool {
	for _, l := range o.lectures {
		if l.Kind == types.CarriedDel {
			return in.avantLOrigine(l.TimestampUS)
		}
	}
	return false
}

// frame rend la frame d un instant du film. Un instant anterieur a l origine vaut la frame 0 : l axe publie
// commence a l origine, et la soustraction non signee d un instant plus ancien deborderait.
func (in entreeDEquipementPorte) frame(ts uint64) int {
	if ts < in.origin {
		return 0
	}
	return int((ts - in.origin) / in.step) //nolint:gosec // instants posterieurs a l origine
}

// avantLOrigine dit si un instant precede la premiere frame publiee.
func (in entreeDEquipementPorte) avantLOrigine(ts uint64) bool { return ts < in.origin }
