package handlers_test

// tactical_oracle_test.go — LES DEUX FRONTIERES DE LA ROUTE RASTER DOIVENT ETRE
// INDISCERNABLES, ET AUCUNE BASE NE DOIT S'OUVRIR POUR RIEN.
//
// Revue R2, deux constats sur `/tactical/{map_id}/raster` :
//
//  1. P1 — ORACLE PAR LE LIBELLE. Le refus de `MapIDValide` publiait la sentinelle nue,
//     tandis qu'une carte legitime jamais jouee passait par le service, qui enrobait son
//     erreur avec la carte demandee (`fmt.Errorf("%w (%q)", ...)`), publiee telle quelle par
//     `mapTacticalError`. Meme statut, meme code, mais DEUX CORPS : la presence de
//     l'identifiant cite disait a l'appelant s'il avait franchi la validation. Le code avait
//     ete rendu indiscernable a la ronde 1 ; le message ne l'etait pas.
//  2. P2 — ORDRE. La fabrique de service (qui RESOUT le joueur et OUVRE sa base) s'executait
//     avant la validation, contrairement aux deux routes du fond.
//
// Ces tests couvrent les deux couches du correctif, chacune par son propre cas :
// le service rend desormais la sentinelle NUE (verrouille cote service par
// `tactical_service_carte_test.go`), et le handler publie le message CANONIQUE quoi que le
// service lui donne — la ceinture, eprouvee ici avec un double qui enrobe.

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// fabriqueComptee enveloppe un service et COMPTE les appels a la fabrique.
func fabriqueComptee(
	svc port.TacticalService, compteur *int,
) handlers.ServiceFactory[port.TacticalService] {
	return func(context.Context, string) (port.TacticalService, error) {
		*compteur++
		return svc, nil
	}
}

// comparerEntetes exige des en-tetes IDENTIQUES entre deux reponses. Le corps ne suffit pas :
// un `Content-Type`, un `Cache-Control` ou une longueur qui differeraient rendraient les deux
// refus distinguables sans qu'aucun octet du corps ne bouge.
func comparerEntetes(t *testing.T, a, b *httptest.ResponseRecorder) {
	t.Helper()
	for nom, valeurs := range a.Header() {
		if got := b.Header().Values(nom); !memesValeurs(valeurs, got) {
			t.Errorf("en-tete %q : %v vs %v", nom, valeurs, got)
		}
	}
	for nom := range b.Header() {
		if len(a.Header().Values(nom)) == 0 {
			t.Errorf("en-tete %q present d'un seul cote", nom)
		}
	}
}

func memesValeurs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// carteHostile : un map_id a UN SEUL segment que la liste blanche doit refuser. Les
// antislashs sont le cas qui compte — c'est celui que chi laisse passer entier.
const carteHostile = `..\..\x`

// TestRaster_AucuneOuvertureDeBaseSurEntreeRefusee — la fabrique n'est PAS appelee.
//
// Elle resout le joueur et ouvre sa base : une entree hors vocabulaire ne doit rien faire
// ouvrir. Avant la revue R2 elle s'executait la premiere, et seul `vuCarte` etait verifie —
// un cran trop tard.
func TestRaster_AucuneOuvertureDeBaseSurEntreeRefusee(t *testing.T) {
	for _, hostile := range mapIDsHostilesURL {
		svc := &fakeTacticalSvc{}
		appels := 0
		w := appel(t, newTacticalRouter(fabriqueComptee(svc, &appels)),
			"/players/JGtm/tactical/"+hostile+"/raster")
		if w.Code != http.StatusNotFound {
			t.Errorf("map_id %q : status=%d, attendu 404", hostile, w.Code)
		}
		if appels != 0 {
			t.Errorf("map_id %q : fabrique appelee %d fois — elle ouvre la base du joueur",
				hostile, appels)
		}
	}
}

// TestRaster_FabriqueAppeleeSurEntreeValide — sentinelle anti-vacuite du test precedent :
// sur une entree LEGITIME la fabrique est bien appelee. Sans elle, un handler qui ne
// l'appellerait plus jamais passerait pour exemplaire.
func TestRaster_FabriqueAppeleeSurEntreeValide(t *testing.T) {
	appels := 0
	w := appel(t, newTacticalRouter(fabriqueComptee(&fakeTacticalSvc{}, &appels)),
		"/players/JGtm/tactical/streets/raster")
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if appels != 1 {
		t.Errorf("fabrique appelee %d fois, attendu 1", appels)
	}
}

// TestRaster_OctetPourOctet_FormeReelleDuService — la comparaison demandee par la revue,
// avec la forme d'erreur que le service produit AUJOURD'HUI : la sentinelle nue.
//
// C'est ce test que fait tomber le retour de `fmt.Errorf("%w (%q)", ...)` cote service.
func TestRaster_OctetPourOctet_FormeReelleDuService(t *testing.T) {
	inconnue := appel(t,
		newTacticalRouter(tacticalFactory(&fakeTacticalSvc{errRast: formeReelleCarteInconnue()}, nil)),
		"/players/JGtm/tactical/carte-jamais-jouee/raster")
	hostile := appel(t, newTacticalRouter(tacticalFactory(&fakeTacticalSvc{}, nil)),
		"/players/JGtm/tactical/"+carteHostile+"/raster")

	if inconnue.Code != hostile.Code {
		t.Fatalf("statuts distincts : inconnue=%d hostile=%d", inconnue.Code, hostile.Code)
	}
	if strings.Contains(inconnue.Body.String(), "carte-jamais-jouee") {
		t.Errorf("le corps CITE la carte demandee : %s", inconnue.Body.String())
	}
	if inconnue.Body.String() != hostile.Body.String() {
		t.Errorf("corps distincts :\n  inconnue = %s\n  hostile  = %s",
			inconnue.Body.String(), hostile.Body.String())
	}
	comparerEntetes(t, inconnue, hostile)
}

// TestRaster_OctetPourOctet_MemeSiLeServiceEnrobe — LA CEINTURE, cote handler.
//
// On suppose ici le contraire du test precedent : un service qui enrobe son erreur avec la
// carte demandee. Le corps publie ne doit PAS en dependre — `mapTacticalError` sert le
// message canonique, jamais `err.Error()`. Sans cette ceinture, il suffirait qu'un futur
// enrobage revienne cote service pour rouvrir l'oracle par le libelle.
func TestRaster_OctetPourOctet_MemeSiLeServiceEnrobe(t *testing.T) {
	enrobe := &fakeTacticalSvc{
		errRast: fmt.Errorf("%w (%q)", domain.ErrTacticalCarteInconnue, "zzz"),
	}
	inconnue := appel(t, newTacticalRouter(tacticalFactory(enrobe, nil)),
		"/players/JGtm/tactical/zzz/raster")
	hostile := appel(t, newTacticalRouter(tacticalFactory(&fakeTacticalSvc{}, nil)),
		"/players/JGtm/tactical/"+carteHostile+"/raster")

	if strings.Contains(inconnue.Body.String(), "zzz") {
		t.Errorf("le corps publie CITE la carte demandee : %s", inconnue.Body.String())
	}
	if inconnue.Code != hostile.Code || inconnue.Body.String() != hostile.Body.String() {
		t.Errorf("reponses distinctes : (%d %s) vs (%d %s)",
			inconnue.Code, inconnue.Body.String(), hostile.Code, hostile.Body.String())
	}
	comparerEntetes(t, inconnue, hostile)
}

// formeReelleCarteInconnue : la sentinelle NUE, telle que le service la rend depuis la revue
// R2. Le verrou de cette forme est cote service (`tactical_service_carte_test.go`) : ici on
// ne fait que la reproduire.
func formeReelleCarteInconnue() error { return domain.ErrTacticalCarteInconnue }
