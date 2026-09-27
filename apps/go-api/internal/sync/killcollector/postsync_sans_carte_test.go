package killcollector

// postsync_sans_carte_test.go — UN MATCH DEJA CONSTATE SANS CARTE NE SE RELIT PAS A CHAQUE CYCLE
// (revue du correctif J7, 2026-09-27).
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : ne plus filtrer par le registre dans `travailDuCycle`
// (cout) ; ne plus vider le registre quand l empreinte change dans `accorder` (invalidation) ;
// ne plus elaguer a la fin du backlog (jauge).

import (
	"context"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/port"
)

// cartesComptees : le resolveur de carte par match, qui COMPTE ses lectures. Un match absent de
// la table n a aucun nom en base.
type cartesComptees struct {
	mu       sync.Mutex
	noms     map[string][]string
	lectures map[string]int
}

func (c *cartesComptees) MapKeysForMatch(_ context.Context, id string) (port.MatchMapKeys, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lectures[id]++
	return port.MatchMapKeys{Names: c.noms[id]}, nil
}

func (c *cartesComptees) MapKeysForMap(context.Context, string) (port.MatchMapKeys, error) {
	return port.MatchMapKeys{}, nil
}

// cycleSansCarte : trois matchs sans carte (a1, a2 sans nom ; a3 hors catalogue) en tete, deux
// avec carte ensuite ; une seule page.
func cycleSansCarte(t *testing.T) (*PostSyncHook, *KillSourceCollector, *cartesComptees, sourceDuBacklog) {
	t.Helper()
	cartes := &cartesComptees{
		noms:     map[string][]string{"a3": {"Forge Personnalisee"}, "b1": {"Bazaar"}, "b2": {"Bazaar"}},
		lectures: map[string]int{},
	}
	col := collecteurDeBobine(t)
	col.WithPositionCapture(cartes, catalogueDuDepot(t))
	src := sourceDuBacklog{
		premiere: []string{"a1", "a2", "a3", "b1", "b2"},
		lire: func(int) ([]string, bool) {
			t.Fatal("une seule page : la suivante ne doit pas etre lue")
			return nil, false
		},
	}
	return &PostSyncHook{perCycle: 2, horizon: 64}, col, cartes, src
}

func verifierTravail(t *testing.T, cycle int, travail []string) {
	t.Helper()
	if len(travail) != 2 || travail[0] != "b1" || travail[1] != "b2" {
		t.Fatalf("cycle %d : travail = %v, attendu [b1 b2]", cycle, travail)
	}
}

func verifierLectures(t *testing.T, cartes *cartesComptees, attendu int, pourquoi string) {
	t.Helper()
	for _, id := range []string{"a1", "a2", "a3"} {
		if n := cartes.lectures[id]; n != attendu {
			t.Errorf("%s : carte lue %d fois, attendu %d — %s", id, n, attendu, pourquoi)
		}
	}
}

func TestTravailDuCycleNeRelitPasLesCartesDejaConstatees(t *testing.T) {
	h, col, cartes, src := cycleSansCarte(t)
	reg := nouveauRegistreSansCarte()
	reg.accorder(context.Background(), "catalogue-1")

	verifierTravail(t, 1, h.travailDuCycle(context.Background(), col, reg, src, nil))
	evictions := observability.LoadCounter(metricCarteAvantTelechargement)
	sautes := observability.LoadCounter(metricSansCarteDejaConstates)

	reg.accorder(context.Background(), "catalogue-1") // meme catalogue au cycle suivant
	verifierTravail(t, 2, h.travailDuCycle(context.Background(), col, reg, src, nil))
	verifierLectures(t, cartes, 1, "un match deja constate sans carte est relu (et rejournalise) a chaque cycle")
	if n := observability.LoadCounter(metricCarteAvantTelechargement) - evictions; n != 0 {
		t.Errorf("%s : +%d au second cycle, attendu 0 (rien de nouveau n est constate)",
			metricCarteAvantTelechargement, n)
	}
	if n := observability.LoadCounter(metricSansCarteDejaConstates) - sautes; n != 3 {
		t.Errorf("%s : +%d, attendu +3 — les sauts sont invisibles", metricSansCarteDejaConstates, n)
	}
	if n := reg.taille(); n != 3 {
		t.Errorf("registre : %d matchs, attendu 3", n)
	}
}

// TestTravailDuCycleRelitQuandLeCatalogueChange : un catalogue de bornes qui change peut connaitre
// la carte d un match constate sans : chaque match se relit UNE fois sous le nouveau.
func TestTravailDuCycleRelitQuandLeCatalogueChange(t *testing.T) {
	h, col, cartes, src := cycleSansCarte(t)
	reg := nouveauRegistreSansCarte()
	reg.accorder(context.Background(), "catalogue-1")
	verifierTravail(t, 1, h.travailDuCycle(context.Background(), col, reg, src, nil))

	reg.accorder(context.Background(), "catalogue-2")
	verifierTravail(t, 2, h.travailDuCycle(context.Background(), col, reg, src, nil))
	verifierLectures(t, cartes, 2, "le catalogue a change et le match n a pas ete relu")
}

// TestRegistreElagueCeQuiAQuitteLeBacklog : un cycle qui lit le backlog jusqu a sa fin retire du
// registre les matchs qui n y sont plus (la jauge ne compte pas des fantomes).
func TestRegistreElagueCeQuiAQuitteLeBacklog(t *testing.T) {
	h, col, _, src := cycleSansCarte(t)
	reg := nouveauRegistreSansCarte()
	reg.empreinte = "catalogue-1"
	reg.ids["parti"] = reg.heure()
	verifierTravail(t, 1, h.travailDuCycle(context.Background(), col, reg, src, nil))
	if _, reste := reg.ids["parti"]; reste {
		t.Error("un match qui a quitte le backlog reste au registre (et compte dans la jauge)")
	}
	if n := reg.taille(); n != 3 {
		t.Errorf("registre : %d matchs, attendu 3 (a1, a2, a3)", n)
	}
}

// TestEmpreinteDuCatalogue : stable pour un meme contenu, differente des qu une entree change.
func TestEmpreinteDuCatalogue(t *testing.T) {
	ctx := context.Background()
	a, b := catalogueDuDepot(t), catalogueDuDepot(t)
	if empreinteDuCatalogue(ctx, a) == "" || empreinteDuCatalogue(ctx, a) != empreinteDuCatalogue(ctx, b) {
		t.Fatal("deux chargements du meme catalogue n ont pas la meme empreinte")
	}
	for cle, e := range b.Maps {
		e.AxisWidths[0]++
		b.Maps[cle] = e
		break
	}
	if empreinteDuCatalogue(ctx, a) == empreinteDuCatalogue(ctx, b) {
		t.Error("une entree du catalogue a change et l empreinte non : le registre ne s invaliderait pas")
	}
}

// TestRegistreDuCycleSansCarteCablee : sans catalogue, aucun registre (tout est ecarte, rien n est
// un constat) et la jauge vaut tout le backlog restant.
func TestRegistreDuCycleSansCarteCablee(t *testing.T) {
	if reg := registreDuCycle(context.Background(), "titre-de-test", DepsCapture{}); reg != nil {
		t.Fatal("un registre est tenu sans catalogue de bornes")
	}
	publierSansCarte(nil, 42)
	if v := observability.LoadCounter(CompteurPostSyncSansCarte); v != 42 {
		t.Errorf("%s = %d, attendu 42", CompteurPostSyncSansCarte, v)
	}
}

// TestTravailDuCycleRelitUnConstatExpire : la cause d un constat peut disparaitre sans changement
// de catalogue (backfill des noms du registre, traduction arrivee) — passe sa duree de vie, le
// match est relu.
func TestTravailDuCycleRelitUnConstatExpire(t *testing.T) {
	h, col, cartes, src := cycleSansCarte(t)
	horloge := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	reg := nouveauRegistreSansCarte()
	reg.maintenant = func() time.Time { return horloge }
	reg.accorder(context.Background(), "catalogue-1")

	verifierTravail(t, 1, h.travailDuCycle(context.Background(), col, reg, src, nil))
	horloge = horloge.Add(DureeDeVieDesConstatsSansCarte - time.Minute)
	verifierTravail(t, 2, h.travailDuCycle(context.Background(), col, reg, src, nil))
	verifierLectures(t, cartes, 1, "un constat encore valide a ete relu")

	horloge = horloge.Add(2 * time.Minute) // le constat du cycle 1 a depasse sa duree de vie
	verifierTravail(t, 3, h.travailDuCycle(context.Background(), col, reg, src, nil))
	verifierLectures(t, cartes, 2, "un constat EXPIRE n a pas ete relu : un nom de carte reecrit en "+
		"base (backfill des noms, traduction arrivee) ne serait jamais vu sans changement de catalogue")
}
