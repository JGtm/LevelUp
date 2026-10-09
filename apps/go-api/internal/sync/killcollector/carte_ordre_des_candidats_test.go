package killcollector

// carte_ordre_des_candidats_test.go — L'ORDRE DES NOMS CANDIDATS EST LE MECANISME DU REPLI
// `repli_carte_premier_nom_resolu` (revue de jalon M1, lentille L6).
//
// # LE TROU QUE CE FICHIER FERME, ET IL A ETE MESURE
//
// [KillSourceCollector.entreeDeCatalogueParNom] EST le repli `repli_carte_premier_nom_resolu`
// (registre des replis, `facts/fallback/registre_killsource_carte.go`) : « les identites
// candidates sont essayees dans l'ordre et la PREMIERE qui resout gagne, sans arbitrage ».
// Mutation jouee le 2026-09-15 sur la base `34fa53da5` — la boucle parcourue A L'ENVERS — et TOUT
// restait vert : `killcollector`, `film/...` et `archlint` compris. Deux raisons, et il faut les
// dire ensemble : aucun temoin de `hits_carte_par_nom_test.go` ne donne DEUX noms qui resolvent
// (le seul a deux candidats commence par une carte absente du catalogue), et l'empreinte de
// `TestKillsourceRevSuitLaSortie` hache le perimetre de `film/internal/facts/killsource`,
// jamais `killcollector`.
//
// Le mecanisme d'un repli non teste est un repli dont personne ne verra changer la reponse.
//
// # POURQUOI LES DEUX SENS SONT VERIFIES
//
// Un seul temoin (« Fragmentation gagne ») serait satisfait par un lecteur qui prefererait
// Fragmentation pour n'importe quelle raison. Les deux sens ensemble ne peuvent etre satisfaits
// que par l'ORDRE : la meme paire, permutee, doit rendre l'autre carte.
//
// # CE QUI N'EST PAS TESTE ICI, ET POURQUOI
//
// Le declenchement de ce repli N'EST PAS COMPTE en production — le registre le dit
// (`CompteurBranche: false`, `CibleComptage` = comptage de la famille 1.9). Il n'y a donc aucun
// compteur a asserter ; ce fichier tient le MECANISME, et le cablage du compteur reste au lot qui
// portera `fallback.Compteur` jusqu'a cette passe. La MESURE que le critere de retrait du
// registre reclame (« 0 match du parc ou deux noms candidats resolvent DES ENTREES DIFFERENTES »)
// est consignee au §4 du plan, entree « Revue M1 — L6 ».

import (
	"context"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
)

// bornesDeLaCarte rend les bornes du catalogue VERSIONNE pour un nom, ou echoue.
func bornesDeLaCarte(t *testing.T, nom string) decfilm.MapQuantEntry {
	t.Helper()
	e, err := catalogueDeBornesVersionne(t).Lookup(nom)
	if err != nil {
		t.Fatalf("%s absente du catalogue versionne : %v", nom, err)
	}
	return e
}

// TestLePremierNomQuiResoutGagne — LE TEMOIN. Deux candidats qui resolvent tous les deux, et
// vers des bornes DIFFERENTES : c'est le premier de la liste qui decide, dans les deux sens.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : parcourir `noms` a l'envers dans
// [KillSourceCollector.entreeDeCatalogueParNom]. Les deux sous-cas basculent ensemble. Jouee et
// restauree par nom le 2026-09-15 (sorties collees au §5 du plan).
func TestLePremierNomQuiResoutGagne(t *testing.T) {
	temoin := bornesDeLaCarte(t, carteTemoinDesJumelles)
	jumelle := bornesDeLaCarte(t, jumelleDeLaCarteTemoin)

	cas := []struct {
		nom     string
		noms    []string
		attendu decfilm.MapQuantEntry
		perdant string
	}{
		{nom: "temoin d'abord", noms: []string{carteTemoinDesJumelles, jumelleDeLaCarteTemoin},
			attendu: temoin, perdant: jumelleDeLaCarteTemoin},
		{nom: "jumelle d'abord", noms: []string{jumelleDeLaCarteTemoin, carteTemoinDesJumelles},
			attendu: jumelle, perdant: carteTemoinDesJumelles},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			got, err := collecteurAvecCarte(t, c.noms...).entreeDeCatalogueParNom(c.noms)
			if err != nil {
				t.Fatalf("candidats %v : %v — les deux resolvent pourtant", c.noms, err)
			}
			if got.Range() != c.attendu.Range() {
				t.Errorf("bornes %v pour les candidats %v, attendues celles de %q %v : c'est le "+
					"PREMIER nom qui resout qui decide, pas %q",
					got.Range(), c.noms, c.noms[0], c.attendu.Range(), c.perdant)
			}
		})
	}
}

// TestLesDeuxPassesSuiventLeMemeOrdreDeCandidats : les deux consommateurs du repli — la passe des
// positions et celle des touches — tirent la MEME carte d'une liste a deux candidats resolvants.
// C'est l'invariant du site unique pose au lot 1.9.4, verifie la ou il peut se rompre : quand les
// candidats ne s'accordent pas.
func TestLesDeuxPassesSuiventLeMemeOrdreDeCandidats(t *testing.T) {
	c := collecteurAvecCarte(t, jumelleDeLaCarteTemoin, carteTemoinDesJumelles)

	positions, err := c.resolveMapBounds(context.Background(), "m-1")
	if err != nil {
		t.Fatalf("resolution des positions : %v", err)
	}
	touches, ok := c.entreeDeCarteDesTouches(context.Background(), "m-1")
	if !ok {
		t.Fatal("resolution des touches refusee la ou celle des positions a abouti")
	}
	attendu := bornesDeLaCarte(t, jumelleDeLaCarteTemoin)
	if positions.Range() != attendu.Range() || touches.Range() != attendu.Range() {
		t.Errorf("positions %v et touches %v, attendues toutes deux celles de %q %v (le PREMIER "+
			"candidat)", positions.Range(), touches.Range(), jumelleDeLaCarteTemoin, attendu.Range())
	}
}
