package replay

// golden_equipement_origines_test.go — LE GARDE-RAIL DE L EQUIPEMENT PORTE, SUR DONNEES REELLES : chaque
// combinaison (famille, origine) que les builds figes OBSERVENT est publiee, et le reste cesse de pouvoir regresser en
// silence.
//
// # LE TROU QU IL FERME
//
// La famille d un objet porte se joignait a l annonce i48 du porteur dans le paquet de sa creation. Le jeu cree DEUX
// objets attaches dans le paquet de la reapparition (jointure ambigue, rien de publie), et cree l equipement de depart
// d un match AVANT l origine du film (rien de publie) : le detecteur de menaces de reapparition de `bc4bde17`, utilise
// quatre fois, n avait ni dessin, ni son, ni compte d usages. Aucun test ne comparait ce que le film CREE a ce que le
// document PUBLIE, par famille et par facon d obtenir l objet. Celui-ci le fait sur les huit builds, depuis leurs
// fixtures figees (aucun film lu) :
//
//   - OBSERVES : les objets ti=37 crees attaches dont l identite (le GlobalID de leur tag `eqip`) se resout en une
//     famille publiable, hors grenades, portes par un joueur dont la piste est publiee et non supprimes avant l origine
//     — la regle de publication, recalculee ici depuis les lectures, pas depuis le constructeur ;
//   - PUBLIES : parmi eux, ceux que le document porte (meme porteur, meme famille, meme frame de prise) ;
//   - L ORIGINE de l objet : `ramasse` (le paquet de sa creation porte la prise du porteur), `reapparition` (l annonce
//     de reapparition), `depart` (cree avant la premiere frame), `autre` sinon.
//
// Le test echoue si un objet observe n est pas publie, si le golden change, et si une combinaison attendue
// ([combinaisonsAttendues]) n est observee par aucun build : une regeneration ne peut pas retirer en silence ce que le
// parc montre.
//
// # REGENERATION (jamais d edition a la main) — DEPUIS LES FIXTURES FIGEES, aucun film lu :
//
//	go test ./replay/ -run TestGoldenEquipementOriginesRegenerate -update-golden-equipement-origines
//
// Elle echoue TOUJOURS en nommant ce qu elle a ecrit ; on relance sans le drapeau pour verifier.

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// updateGoldenEquipementOrigines : LA PORTE D ECRITURE DE CE GOLDEN, ET D AUCUN AUTRE.
var updateGoldenEquipementOrigines = flag.Bool("update-golden-equipement-origines", false,
	"reecrire testdata/equipement_origines.golden, DEPUIS les fixtures figees")

// Les quatre facons d obtenir un objet porte.
const (
	origineRamasse      = "ramasse"
	origineReapparition = "reapparition"
	origineDepart       = "depart"
	origineAutre        = "autre"
)

// combinaisonAttendue : une famille obtenue d une facon donnee, qu au moins un build fige doit montrer.
type combinaisonAttendue struct{ famille, origine string }

// combinaisonsAttendues : ce que le parc MESURE (cf. le journal du lot) et que les builds et films de garde figes doivent
// continuer a couvrir. Le grappin et le propulseur n y sont pas : leur objet est toujours inactif, il ne se resout pas et ne
// se publie pas. Une combinaison que plus aucun fixture n observe est une couverture perdue : la retirer d ici est une
// decision, pas un effet de bord d une regeneration.
var combinaisonsAttendues = []combinaisonAttendue{
	{"sensor", origineRamasse}, {"sensor", origineReapparition}, {"sensor", origineDepart},
	{"wall", origineRamasse}, {"wall", origineReapparition}, {"wall", origineDepart},
	{"repulsor", origineRamasse}, {"repulsor", origineReapparition},
	{"repair_field", origineRamasse}, {"repair_field", origineReapparition},
	{familleDeLEcranOccultant, origineRamasse}, {familleDeLEcranOccultant, origineReapparition},
	{"translocator_beacon", origineRamasse}, {"translocator_beacon", origineReapparition},
	{"powerup_camo", origineRamasse}, {"powerup_overshield", origineRamasse},
}

// ligneOrigine : une ligne du golden.
type ligneOrigine struct {
	film, famille, origine string
	observes, publies      int
}

func (l ligneOrigine) String() string {
	return fmt.Sprintf("%s %-22s %-13s observes=%d publies=%d", l.film, l.famille, l.origine, l.observes, l.publies)
}

// cleDAnnonce : un paquet et un porteur.
type cleDAnnonce struct {
	ts   uint64
	slot uint32
}

// releve est ce qu un film donne au garde-rail : ce que le film CREE (les lectures, les annonces du porteur, l origine,
// les pistes publiees) et ce que le document PUBLIE.
type releve struct {
	lectures []types.CarriedEquipmentRead
	annonces map[cleDAnnonce]types.EquipmentChangeKind
	origine  uint64
	pas      uint64
	pistes   map[uint32]bool
	publies  []CarriedEquipment
}

// annoncesDuPorteur range les annonces de prise et de reapparition par paquet et par porteur.
func annoncesDuPorteur(changes []types.EquipmentChange) map[cleDAnnonce]types.EquipmentChangeKind {
	out := map[cleDAnnonce]types.EquipmentChangeKind{}
	for _, c := range changes {
		if c.Kind == types.EquipmentTaken || c.Kind == types.EquipmentSpawned {
			out[cleDAnnonce{c.TimestampUS, c.Slot}] = c.Kind
		}
	}
	return out
}

// releveDuBuild : un build fige, assemble de bout en bout depuis ses fixtures.
func releveDuBuild(t *testing.T, b goldenBuild) releve {
	t.Helper()
	g, entry := chargerGoldenBuild(t, b)
	doc := assemblerGoldenBuild(t, b, g, entry)
	c := releve{lectures: g.CarriedEquipment, annonces: annoncesDuPorteur(g.EquipmentChanges),
		origine: g.Positions[0].TimestampUS, pas: uint64(doc.FrameIntervalMS) * 1000, pistes: map[uint32]bool{},
		publies: doc.CarriedEquipment}
	for _, p := range g.Positions {
		c.origine = min(c.origine, p.TimestampUS)
	}
	for _, tr := range doc.Tracks {
		c.pistes[tr.Slot] = true
	}
	// LE RANG PUBLIE DIT LA MEME FAMILLE QUE L OBJET : `r` est le rang de palette de sa famille, ou -1.
	for _, o := range doc.CarriedEquipment {
		if lab, ok := doc.AbilityLabels[strconv.Itoa(o.R)]; o.R != rangInconnu && ok && lab.Family != o.Family {
			t.Errorf("%s : objet du porteur %d de famille %q au rang %d, que la palette nomme %q",
				b.Short8, o.Slot, o.Family, o.R, lab.Family)
		}
	}
	return c
}

// objetObserve est un objet porte que la regle de publication doit montrer.
type objetObserve struct {
	famille, origine string
	slot             uint32
	t0               int
}

// observes recalcule, depuis les lectures et les annonces du film, les objets que le document doit publier.
func (c releve) observes(t *testing.T) []objetObserve {
	t.Helper()
	refs := nouvelleReferenceDesObjetsPortes(goldenEnergyCatalog(t), goldenCatalog(t).EquipmentFamilies)
	supprimes := map[types.LifeKey]uint64{}
	for _, l := range c.lectures {
		if l.Kind == types.CarriedDel {
			supprimes[l.Object] = l.TimestampUS
		}
	}
	var out []objetObserve
	for _, l := range c.lectures {
		if l.Kind != types.CarriedNew || !l.HasGlobalID {
			continue
		}
		famille, issue := refs.famille(l.GlobalID)
		if fin, fini := supprimes[l.Object]; issue != identiteResolue || !c.pistes[l.Carrier] || (fini && fin < c.origine) {
			continue
		}
		o := objetObserve{famille: famille, origine: origineAutre, slot: l.Carrier}
		switch {
		case l.TimestampUS < c.origine:
			o.origine = origineDepart
		case c.annonces[cleDAnnonce{l.TimestampUS, l.Carrier}] == types.EquipmentTaken:
			o.origine = origineRamasse
		case c.annonces[cleDAnnonce{l.TimestampUS, l.Carrier}] == types.EquipmentSpawned:
			o.origine = origineReapparition
		}
		if l.TimestampUS >= c.origine {
			o.t0 = int((l.TimestampUS - c.origine) / c.pas) //nolint:gosec // instants posterieurs a l origine
		}
		out = append(out, o)
	}
	return out
}

// lignes range les objets observes par (famille, origine) et compte ceux que le document publie.
func (c releve) lignes(t *testing.T, film string) []ligneOrigine {
	t.Helper()
	type cle struct {
		famille string
		slot    uint32
		t0      int
	}
	publies := map[cle]int{}
	for _, o := range c.publies {
		publies[cle{o.Family, o.Slot, o.T0}]++
	}
	rangees := map[[2]string]*ligneOrigine{}
	for _, o := range c.observes(t) {
		k := [2]string{o.famille, o.origine}
		if rangees[k] == nil {
			rangees[k] = &ligneOrigine{film: film, famille: o.famille, origine: o.origine}
		}
		rangees[k].observes++
		pub := cle{o.famille, o.slot, o.t0}
		if publies[pub] > 0 {
			publies[pub]--
			rangees[k].publies++
		}
	}
	var out []ligneOrigine
	for _, l := range rangees {
		out = append(out, *l)
	}
	slices.SortFunc(out, func(x, y ligneOrigine) int { return strings.Compare(x.String(), y.String()) })
	return out
}

// lignesDeTousLesFilms : la table complete — les huit builds, puis les films de garde.
func lignesDeTousLesFilms(t *testing.T) []ligneOrigine {
	t.Helper()
	var out []ligneOrigine
	for _, b := range goldenBuilds() {
		out = append(out, releveDuBuild(t, b).lignes(t, b.Short8)...)
	}
	for _, f := range filmsDeGarde() {
		out = append(out, releveDuFilmDeGarde(t, f).lignes(t, f.Short8)...)
	}
	return out
}

func texteDuGoldenOrigines(lignes []ligneOrigine) string {
	var b strings.Builder
	b.WriteString("# EQUIPEMENT PORTE : CE QUE LE FILM CREE ET CE QUE LE DOCUMENT PUBLIE, PAR FAMILLE ET PAR ORIGINE.\n")
	b.WriteString("# Fige par golden_equipement_origines_test.go, depuis les fixtures des huit builds et des films de garde.\n")
	b.WriteString("# Ne s edite JAMAIS a la main.\n")
	for _, l := range lignes {
		b.WriteString(l.String() + "\n")
	}
	return b.String()
}

func goldenEquipementOriginesPath() string {
	return filepath.Join(goldenDir, "equipement_origines.golden")
}

// TestGoldenEquipementOrigines : tout objet observe est publie, et la table est celle qui est figee.
func TestGoldenEquipementOrigines(t *testing.T) {
	lignes := lignesDeTousLesFilms(t)
	for _, l := range lignes {
		if l.publies != l.observes {
			t.Errorf("%s : %d objet(s) cree(s) sur %d ne sont pas publies — une combinaison (famille, origine) observee "+
				"cesse d etre publiee", l, l.observes-l.publies, l.observes)
		}
	}
	want, err := os.ReadFile(goldenEquipementOriginesPath()) //nolint:gosec // chemin fige dans le code
	if err != nil {
		t.Fatalf("golden absent : %v — regenerer avec -update-golden-equipement-origines", err)
	}
	if got := texteDuGoldenOrigines(lignes); string(want) != got {
		t.Errorf("la table famille x origine a change par rapport a %s.\n%s", goldenEquipementOriginesPath(),
			premierEcartAssembly(string(want), got))
	}
}

// TestGoldenEquipementOriginesCouvre : les combinaisons attendues sont observees par au moins un film.
func TestGoldenEquipementOriginesCouvre(t *testing.T) {
	vues := map[combinaisonAttendue]bool{}
	for _, l := range lignesDeTousLesFilms(t) {
		if l.publies > 0 {
			vues[combinaisonAttendue{l.famille, l.origine}] = true
		}
	}
	for _, c := range combinaisonsAttendues {
		if !vues[c] {
			t.Errorf("combinaison (%s, %s) : aucun film fige ne la montre publiee", c.famille, c.origine)
		}
	}
}

// TestGoldenEquipementOriginesTemoinNominal : `bc4bde17`, slot 518 (JGtm, Bazaar, Super Fiesta) — le detecteur de menaces
// de depart de match, cree 22,3 s avant la premiere frame, active quatre fois : il est publie (prise a la frame 0), avec ses
// quatre activations, ses quatre objets engendres, tous situes, et ses usages restants.
func TestGoldenEquipementOriginesTemoinNominal(t *testing.T) {
	c := releveDuFilmDeGarde(t, filmsDeGarde()[0])
	var objets []CarriedEquipment
	for _, o := range c.publies {
		if o.Slot == 518 {
			objets = append(objets, o)
		}
	}
	if len(objets) != 1 {
		t.Fatalf("slot 518 : %d objet(s) publie(s), attendu 1", len(objets))
	}
	o := objets[0]
	situes := 0
	for _, tr := range o.Tracked {
		if tr.Land != nil {
			situes++
		}
	}
	if o.Family != "sensor" || o.T0 != 0 || len(o.Uses) != 4 || len(o.Tracked) != 4 || situes != 4 ||
		len(o.Energy) == 0 || o.Energy[0].N == nil || *o.Energy[0].N != 4 {
		t.Fatalf("objet du slot 518 : famille %q, t0 %d, %d activation(s), %d objet(s) engendre(s) dont %d situe(s), energie %+v",
			o.Family, o.T0, len(o.Uses), len(o.Tracked), situes, o.Energy)
	}
}

// TestGoldenEquipementOriginesRegenerate : LA SEULE PORTE D ECRITURE du golden et des fixtures des films de garde.
// `fixtures` relit les films (REPLAY_FILM_CACHE) ; `table` ne lit que les fixtures figees.
func TestGoldenEquipementOriginesRegenerate(t *testing.T) {
	if !*updateGoldenEquipementOrigines {
		t.Skip("regeneration du golden famille x origine : passer -update-golden-equipement-origines")
	}
	t.Run("fixtures", func(t *testing.T) { ecrireLesFixturesDeGarde(t) })
	t.Run("table", func(t *testing.T) {
		if err := os.WriteFile(goldenEquipementOriginesPath(), []byte(texteDuGoldenOrigines(lignesDeTousLesFilms(t))), 0o600); err != nil {
			t.Fatalf("ecriture de %s : %v", goldenEquipementOriginesPath(), err)
		}
		t.Fatalf("golden reecrit : %s ; relancer sans -update-golden-equipement-origines pour verifier",
			goldenEquipementOriginesPath())
	})
}
