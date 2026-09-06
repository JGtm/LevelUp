package handlers_test

// tactical_mapid_test.go — LE map_id EST LA PREMIERE CLE DE FOND CONTROLEE PAR L'APPELANT.
//
// Sur le chemin par match, la cle venait de `match_registry` : la base la fournissait. Ici
// elle arrive de l'URL et finit, apres le service, dans
// `filepath.Join(map_backgrounds, cle + ".json")`. Sous Windows, `..\..\x` traverse chi
// comme UN SEUL segment et `filepath.Join` resout l'antislash comme separateur : le
// `os.Stat` sortait alors du repertoire des fonds.
//
// Ce que ces tests cadenassent (revue R1, constat G1) :
//   - la liste blanche accepte ce qui EST un map_id (uuid d'asset, module Forge) ;
//   - elle refuse tout le reste, y compris les formes qui ne traversent PAS aujourd'hui
//     (`%2e%2e%2f` litteral, espace) — la frontiere ne doit pas dependre du fait que chi ne
//     de-echappe pas ;
//   - le refus est un 404 portant le code d'absence habituel de chaque route, JAMAIS un 400
//     ni un 500 : un code distinct ferait oracle sur la validation ;
//   - AUCUN service n'est appele quand l'entree est refusee.

import (
	"net/http"
	"strings"
	"testing"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// mapIDsHostiles : tout ce qui doit etre refuse par le PREDICAT, quelle que soit la
// plate-forme.
var mapIDsHostiles = []string{
	"..",
	"../x",
	`..\x`,
	"x/y",
	`x\y`,
	"%2e%2e%2f",
	"",
	"   ",
	".",
	".hidden",
	"-flag",
	"carte.json",
	"carte%20",
	"c:/windows",
	strings.Repeat("a", 129),
}

// mapIDsHostilesURL : les memes, moins ceux qui n'atteignent pas notre validation.
//
//   - ESPACE BRUTE dans la cible d'une requete : requete malformee, `httptest` refuse meme
//     de la construire. Sa forme reelle sur le fil est `%20`, deja dans la table.
//   - SEGMENT VIDE : refuse en amont par le transport (chi ne trouve pas de route pour
//     l'image ; la liaison de parametre de Huma rend 422 pour le calage). Le refus a bien
//     lieu, simplement pas chez nous — l'assertion « code du contrat » n'aurait aucun sens.
//
// Les deux restent dans la table du PREDICAT, ou ils sont a leur place.
var mapIDsHostilesURL = filtrerEmettables(mapIDsHostiles)

func filtrerEmettables(entrees []string) []string {
	var out []string
	for _, e := range entrees {
		if e == "" || strings.ContainsAny(e, " 	") {
			continue
		}
		out = append(out, e)
	}
	return out
}

// mapIDsValides : les deux formes reelles d'un identifiant de carte.
var mapIDsValides = []string{
	"105f5d84-8de1-4908-af3a-1c4f3bf9d642", // asset UGC (uuid)
	"ridgeline",                            // module natif
	"fo08_wetland",                         // canevas Forge (souligne)
	"A0",
	strings.Repeat("a", 128),
}

// verifieCodeSiJSON controle le code d'erreur du contrat QUAND la reponse vient de nos
// handlers.
//
// UNE PARTIE DES ENTREES HOSTILES N'ATTEINT JAMAIS LE HANDLER, et c'est une bonne nouvelle
// : `../x`, `x/y` ou `c:/windows` portent un `/`, donc DEUX segments — chi ne trouve plus de
// route et rend son 404 en texte brut. Le refus a lieu un cran plus tot, la reponse reste un
// 404, et aucun service n'est appele. Les entrees a UN SEUL segment (antislash, point,
// tiret, espace echappe, longueur) sont, elles, celles que la liste blanche doit arreter :
// ce sont elles qui portent le test.
func verifieCodeSiJSON(t *testing.T, entree, corps, codeAttendu string) {
	t.Helper()
	if !strings.HasPrefix(strings.TrimSpace(corps), "{") {
		return // 404 du routeur, en texte brut : le handler n'a pas ete atteint
	}
	if !strings.Contains(corps, codeAttendu) {
		t.Errorf("map_id %q : code d'erreur inattendu — %s", entree, corps)
	}
}

// TestMapIDValide_ListeBlanche — le predicat seul, aux deux bords.
func TestMapIDValide_ListeBlanche(t *testing.T) {
	for _, hostile := range mapIDsHostiles {
		if _, ok := handlers.MapIDValide(hostile); ok {
			t.Errorf("map_id %q accepte alors qu'il doit etre refuse", hostile)
		}
	}
	for _, valide := range mapIDsValides {
		got, ok := handlers.MapIDValide(valide)
		if !ok {
			t.Errorf("map_id %q refuse alors qu'il est legitime", valide)
			continue
		}
		if got != valide {
			t.Errorf("map_id %q rendu comme %q", valide, got)
		}
	}
	// L'espacement est REFUSE, pas nettoye : une frontiere qui repare son entree
	// accepterait `carte%20` comme `carte` — deux URL pour une meme ressource, et une
	// normalisation de plus a raisonner le jour ou le motif change.
	if _, ok := handlers.MapIDValide("  ridgeline  "); ok {
		t.Error("espacement nettoye au lieu d'etre refuse")
	}
}

// TestTacticalMapID_ImageRefuseLesChemins — l'image : 404 map_background_not_available,
// et le service n'est jamais appele.
func TestTacticalMapID_ImageRefuseLesChemins(t *testing.T) {
	for _, hostile := range mapIDsHostilesURL {
		// Le double SERVIRAIT une image si on l'appelait : c'est ce qui rend le test
		// concluant — un 404 ici ne peut venir que du refus, pas d'une absence de donnee.
		mock := &mockReplayService{imageMap: []byte{1, 2, 3}}
		w := appel(t, routeurFond(mock), "/players/JGtm/tactical/"+hostile+"/background.png")
		if w.Code != http.StatusNotFound {
			t.Errorf("map_id %q : status=%d, attendu 404 — body=%s", hostile, w.Code, w.Body.String())
		}
		verifieCodeSiJSON(t, hostile, w.Body.String(), "map_background_not_available")
		if mock.vuMapID != "" {
			t.Errorf("map_id %q : transmis au service (%q)", hostile, mock.vuMapID)
		}
	}
}

// TestTacticalMapID_CalageRefuseLesChemins — le calage : meme refus, meme code.
func TestTacticalMapID_CalageRefuseLesChemins(t *testing.T) {
	for _, hostile := range mapIDsHostilesURL {
		mock := &mockReplayService{}
		w := appel(t, routeurFond(mock), "/players/JGtm/tactical/"+hostile+"/background")
		if w.Code != http.StatusNotFound {
			t.Errorf("map_id %q : status=%d, attendu 404 — body=%s", hostile, w.Code, w.Body.String())
		}
		if mock.vuMapID != "" {
			t.Errorf("map_id %q : transmis au service (%q)", hostile, mock.vuMapID)
		}
	}
}

// TestTacticalMapID_RasterRefuseLesChemins — la lecture de placement rend SON code
// d'absence (`tactical_map_unknown`), pas celui du fond : un code de validation distinct
// dirait a l'appelant que son entree a franchi le routeur mais pas le filtre.
func TestTacticalMapID_RasterRefuseLesChemins(t *testing.T) {
	for _, hostile := range mapIDsHostilesURL {
		svc := &fakeTacticalSvc{}
		w := appel(t, newTacticalRouter(tacticalFactory(svc, nil)),
			"/players/JGtm/tactical/"+hostile+"/raster")
		if w.Code != http.StatusNotFound {
			t.Errorf("map_id %q : status=%d, attendu 404 — body=%s", hostile, w.Code, w.Body.String())
		}
		verifieCodeSiJSON(t, hostile, w.Body.String(), "tactical_map_unknown")
		if svc.vuCarte != "" {
			t.Errorf("map_id %q : transmis au service (%q)", hostile, svc.vuCarte)
		}
	}
}

// TestTacticalMapID_IndiscernableDUneCarteInconnue — un map_id HOSTILE et un map_id
// LEGITIME mais inconnu rendent EXACTEMENT la meme reponse. C'est la propriete qui empeche
// l'oracle : rien ne dit a l'appelant laquelle des deux frontieres il a heurtee.
func TestTacticalMapID_IndiscernableDUneCarteInconnue(t *testing.T) {
	svcInconnue := &fakeTacticalSvc{errRast: domain.ErrTacticalCarteInconnue}
	legitime := appel(t, newTacticalRouter(tacticalFactory(svcInconnue, nil)),
		"/players/JGtm/tactical/carte-jamais-jouee/raster")
	hostile := appel(t, newTacticalRouter(tacticalFactory(&fakeTacticalSvc{}, nil)),
		`/players/JGtm/tactical/..\..\x/raster`)
	if legitime.Code != hostile.Code {
		t.Fatalf("statuts distincts : legitime=%d hostile=%d", legitime.Code, hostile.Code)
	}
	if legitime.Body.String() != hostile.Body.String() {
		t.Errorf("corps distincts :\n  legitime = %s\n  hostile  = %s",
			legitime.Body.String(), hostile.Body.String())
	}

	mockSansFond := &mockReplayService{bgMapErr: port.ErrMapBackgroundNotAvailable}
	legitimeFond := appel(t, routeurFond(mockSansFond),
		"/players/JGtm/tactical/carte-sans-fond/background.png")
	hostileFond := appel(t, routeurFond(&mockReplayService{}),
		`/players/JGtm/tactical/..\..\x/background.png`)
	if legitimeFond.Code != hostileFond.Code || legitimeFond.Body.String() != hostileFond.Body.String() {
		t.Errorf("fond : reponses distinctes (%d %s) vs (%d %s)",
			legitimeFond.Code, legitimeFond.Body.String(),
			hostileFond.Code, hostileFond.Body.String())
	}
}
