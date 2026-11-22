package killcollector

// carte_par_nom_test.go — LOT 1.9.4 : LA CARTE DU MATCH VIENT DE SON NOM, ET FAUSSER LE NOM SE VOIT.
//
// # CE QUE CE FICHIER TIENT
//
// La carte du match se resout par le NOM DE MATCH que la base donne au collecteur, jamais par la
// SIGNATURE de ses largeurs d'axe lues dans le film. Ce fichier verrouille les trois sorties de cette
// resolution — la bonne carte, et les deux refus.
//
// # POURQUOI LES JUMELLES SONT LE TEMOIN, ET PAS UNE CARTE QUELCONQUE
//
// `behemoth`, `fragmentation` et `launch site` partagent EXACTEMENT la signature `17/17/15` dans
// le catalogue versionne, et leurs AABB n'ont rien a voir : l'etendue en X vaut 1 132 m pour
// Behemoth contre 1 680 m pour Fragmentation, soit une echelle fausse de pres de moitie sur toute
// position publiee. Une signature ne peut pas les departager : 68 cartes sur 79 sont dans ce cas,
// reparties en cinq classes dont une de 59. Un nom, lui, tranche.
//
// # LES MUTATIONS QUI FONT ROUGIR CE FICHIER
//
//	(A) la carte temoin remplacee par sa jumelle dans LES DEUX ROLES
//	    (`carteTemoinDesJumelles = "Behemoth"`) :
//	    [TestJumellesDuCatalogueSontIndistinguablesParSignature] rougit d'abord (« fausser le nom
//	    serait sans effet, donc intestable »), role de garde du presuppose ;
//	(B) LA FIXTURE seule faussee — le collecteur recoit `jumelleDeLaCarteTemoin` quand l'attente
//	    reste la carte temoin : [TestCarteVientDuNomDeMatch] rougit sur les bornes.

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/port"

	"github.com/JGtm/halo-infinite-film-decoder/decfilm"
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

// TestCarteVientDuNomDeMatch — LE TEMOIN DE MUTATION. Le nom de match decide, et il designe SA carte,
// pas sa jumelle d'echelle.
func TestCarteVientDuNomDeMatch(t *testing.T) {
	c := collecteurAvecCarte(t, carteTemoinDesJumelles)
	entry, err := c.resolveMapBounds(context.Background(), "m-1")
	if err != nil {
		t.Fatalf("la carte du match ne se resout pas alors que son nom est connu et au catalogue : %v", err)
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

// TestCarteRefuseUnNomHorsCatalogue : le nom est connu, la carte n'est pas au catalogue de bornes.
// C'est une ERREUR TYPEE COMPTEE (D-4 d'ADR 0034) — jamais une carte « au plus proche » choisie par
// signature.
func TestCarteRefuseUnNomHorsCatalogue(t *testing.T) {
	c := collecteurAvecCarte(t, carteAbsenteDuCatalogue)
	if _, err := c.resolveMapBounds(context.Background(), "m-1"); !errors.Is(err, decfilm.ErrUnknownMapBounds) {
		t.Errorf("erreur %v, attendue ErrUnknownMapBounds : la signature a-t-elle repris la main ?", err)
	}
}

// TestCarteRefuseUnMatchSansNom : la base ne nomme aucune carte pour ce match. Erreur typee DISTINCTE
// de la precedente — les deux causes appellent deux gestes differents.
func TestCarteRefuseUnMatchSansNom(t *testing.T) {
	c := collecteurAvecCarte(t, "", "")
	_, err := c.nomsDeCarteDuMatch(context.Background(), "m-1")
	if !errors.Is(err, ErrSansNomDeCarte) {
		t.Errorf("erreur %v, attendue ErrSansNomDeCarte", err)
	}
}
