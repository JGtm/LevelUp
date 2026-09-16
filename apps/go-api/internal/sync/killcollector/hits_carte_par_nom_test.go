package killcollector

// hits_carte_par_nom_test.go — LOT 1.9.4 : LA CARTE DES DISTANCES DE TOUCHE VIENT DU NOM DE
// MATCH, ET FAUSSER LE NOM SE VOIT.
//
// # CE QUE CE FICHIER TIENT
//
// La passe des touches appelait `grammar.DetectFilmMapEntry(dir, catalogue, "")` : elle
// IDENTIFIAIT la carte par la SIGNATURE de ses largeurs d'axe, lues dans le film. Elle lit
// desormais le NOM DE MATCH que le collecteur resout deja pour les positions. Ce fichier
// verrouille les quatre sorties de cette resolution — la bonne carte, et les trois refus.
//
// # POURQUOI LES JUMELLES SONT LE TEMOIN, ET PAS UNE CARTE QUELCONQUE
//
// `behemoth`, `fragmentation` et `launch site` partagent EXACTEMENT la signature `17/17/15` dans
// le catalogue versionne, et leurs AABB n'ont rien a voir : l'etendue en X vaut 1 132 m pour
// Behemoth contre 1 680 m pour Fragmentation, soit une echelle fausse de pres de moitie sur toute
// distance publiee. Une signature ne peut pas les departager — la mesure du lot compte 68 cartes
// sur 79 dans ce cas, reparties en cinq classes dont une de 59. Un nom, lui, tranche.
//
// # LES MUTATIONS, JOUEES ET RESTAUREES PAR NOM (2026-09-15, sorties collees au §5 du plan)
//
//	(A) la carte temoin remplacee par sa jumelle dans LES DEUX ROLES
//	    (`carteTemoinDesJumelles = "Behemoth"`) : DEUX tests rouges —
//	    [TestJumellesDuCatalogueSontIndistinguablesParSignature] d'abord (« fausser le nom serait
//	    sans effet, donc intestable »), ce qui est exactement son role de garde du presuppose ;
//	(B) LA FIXTURE seule faussee — le collecteur recoit `jumelleDeLaCarteTemoin` quand l'attente
//	    reste la carte temoin : [TestCarteDesTouchesVientDuNomDeMatch] rouge sur les bornes
//	    (`[-621,99 510,37]` rendu contre `[-1136,27 543,44]` attendu). C'est la mutation que le
//	    brief du lot demande : une carte jumelle A donnee pour B.
//
// AVANT CE LOT, LA MUTATION (B) N'AVAIT MEME PAS DE PRISE : la passe des touches ne recevait
// AUCUN nom de carte — `DetectFilmMapEntry(dir, catalogue, "")` ignorait le parametre qui
// existait pour cela. Il n'y avait pas de fixture a fausser, et c'est le defaut que le lot ferme.

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/profile"
	"levelup/go-api/internal/port"
)

// carteTemoinDesJumelles / jumelleDeLaCarteTemoin : deux cartes du catalogue VERSIONNE de meme
// signature d'axe et de bornes differentes. Le couple est le materiau de la mutation.
const (
	carteTemoinDesJumelles  = "Fragmentation"
	jumelleDeLaCarteTemoin  = "Behemoth"
	signatureDesJumelles    = "17/17/15"
	carteAbsenteDuCatalogue = "Carte Forge Inconnue"
)

// collecteurAvecCarte cable un collecteur avec le catalogue VERSIONNE et les noms candidats
// donnes — le meme cablage que `WithPositionCapture` en production.
func collecteurAvecCarte(t *testing.T, noms ...string) *KillSourceCollector {
	t.Helper()
	c := &KillSourceCollector{}
	c.WithPositionCapture(
		fakeMapNames{keys: port.MatchMapKeys{Names: noms}},
		catalogueDeBornesVersionne(t),
	)
	return c
}

// TestJumellesDuCatalogueSontIndistinguablesParSignature : le PRESUPPOSE du lot, verifie sur la
// donnee versionnee et non sur un souvenir. Si le catalogue changeait au point de rendre ces deux
// cartes distinguables, le temoin de mutation ci-dessous ne prouverait plus rien et ce test le
// dirait AVANT lui.
func TestJumellesDuCatalogueSontIndistinguablesParSignature(t *testing.T) {
	cat := catalogueDeBornesVersionne(t)
	a, err := cat.Lookup(carteTemoinDesJumelles)
	if err != nil {
		t.Fatalf("%s absente du catalogue versionne : %v", carteTemoinDesJumelles, err)
	}
	b, err := cat.Lookup(jumelleDeLaCarteTemoin)
	if err != nil {
		t.Fatalf("%s absente du catalogue versionne : %v", jumelleDeLaCarteTemoin, err)
	}
	if a.AxisWidths != b.AxisWidths {
		t.Fatalf("%s %v et %s %v ne partagent plus leur signature (%s attendue) : le temoin de "+
			"mutation de ce fichier ne prouve plus rien, en choisir un autre",
			carteTemoinDesJumelles, a.AxisWidths, jumelleDeLaCarteTemoin, b.AxisWidths, signatureDesJumelles)
	}
	if a.Range() == b.Range() {
		t.Fatalf("%s et %s ont desormais les MEMES bornes : fausser le nom serait sans effet, "+
			"donc intestable", carteTemoinDesJumelles, jumelleDeLaCarteTemoin)
	}
}

// TestCarteDesTouchesVientDuNomDeMatch — LE TEMOIN DE MUTATION. Le nom de match decide, et il
// designe SA carte, pas sa jumelle d'echelle.
func TestCarteDesTouchesVientDuNomDeMatch(t *testing.T) {
	c := collecteurAvecCarte(t, carteTemoinDesJumelles)
	entry, ok := c.entreeDeCarteDesTouches(context.Background(), "m-1")
	if !ok {
		t.Fatal("la carte du match ne se resout pas alors que son nom est connu et au catalogue")
	}
	attendu, err := catalogueDeBornesVersionne(t).Lookup(carteTemoinDesJumelles)
	if err != nil {
		t.Fatalf("catalogue: %v", err)
	}
	if entry.Range() != attendu.Range() {
		t.Errorf("bornes %v, attendues celles de %s %v", entry.Range(), carteTemoinDesJumelles, attendu.Range())
	}
	jumelle, err := catalogueDeBornesVersionne(t).Lookup(jumelleDeLaCarteTemoin)
	if err != nil {
		t.Fatalf("catalogue: %v", err)
	}
	if entry.Range() == jumelle.Range() {
		t.Errorf("les bornes rendues sont celles de %s, la JUMELLE d echelle de %s — une signature "+
			"de largeurs ne les distingue pas, le nom si", jumelleDeLaCarteTemoin, carteTemoinDesJumelles)
	}
}

// TestCarteDesTouchesRefuseUnNomHorsCatalogue : le nom est connu, la carte n'est pas au
// catalogue de bornes. C'est une ERREUR TYPEE COMPTEE (D-4 d'ADR 0034) — jamais une carte « au
// plus proche » choisie par signature.
func TestCarteDesTouchesRefuseUnNomHorsCatalogue(t *testing.T) {
	c := collecteurAvecCarte(t, carteAbsenteDuCatalogue)
	if _, ok := c.entreeDeCarteDesTouches(context.Background(), "m-1"); ok {
		t.Fatal("une carte hors catalogue a produit une entree : la signature a-t-elle repris la main ?")
	}
	_, err := c.entreeDeCatalogueParNom([]string{carteAbsenteDuCatalogue})
	if !errors.Is(err, profile.ErrUnknownMapBounds) {
		t.Errorf("erreur %v, attendue ErrUnknownMapBounds", err)
	}
}

// TestCarteDesTouchesRefuseUnMatchSansNom : la base ne nomme aucune carte pour ce match. Erreur
// typee DISTINCTE de la precedente — les deux causes appellent deux gestes differents.
func TestCarteDesTouchesRefuseUnMatchSansNom(t *testing.T) {
	c := collecteurAvecCarte(t, "", "")
	if _, ok := c.entreeDeCarteDesTouches(context.Background(), "m-1"); ok {
		t.Fatal("un match sans nom de carte a produit une entree")
	}
	_, err := c.nomsDeCarteDuMatch(context.Background(), "m-1")
	if !errors.Is(err, ErrSansNomDeCarte) {
		t.Errorf("erreur %v, attendue ErrSansNomDeCarte", err)
	}
}

// TestCarteDesTouchesRefuseSansCablage : le collecteur n'a pas recu `WithPositionCapture`. Ni
// panique (l'interface est nil), ni repli par signature : les distances sont desactivees et le
// cablage manquant est compte a part.
func TestCarteDesTouchesRefuseSansCablage(t *testing.T) {
	c := &KillSourceCollector{}
	if _, ok := c.entreeDeCarteDesTouches(context.Background(), "m-1"); ok {
		t.Fatal("un collecteur sans resolution de carte a produit une entree")
	}
}

// TestLesDeuxPassesPartagentLaMemeResolutionDeCarte : positions et touches rendent la MEME entree
// pour le meme match. C'est l'invariant que le lot installe — deux producteurs du meme fait
// divergeraient, et c'est exactement ce qui se passait quand les touches devinaient la carte.
func TestLesDeuxPassesPartagentLaMemeResolutionDeCarte(t *testing.T) {
	c := collecteurAvecCarte(t, carteAbsenteDuCatalogue, carteTemoinDesJumelles)
	positions, err := c.resolveMapBounds(context.Background(), "m-1")
	if err != nil {
		t.Fatalf("resolution des positions: %v", err)
	}
	touches, ok := c.entreeDeCarteDesTouches(context.Background(), "m-1")
	if !ok {
		t.Fatal("resolution des touches refusee la ou celle des positions a abouti")
	}
	if positions != touches {
		t.Errorf("les deux passes resolvent des cartes differentes :\n  positions %+v\n  touches   %+v",
			positions, touches)
	}
}
