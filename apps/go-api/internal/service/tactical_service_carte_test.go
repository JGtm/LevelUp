package service

// tactical_service_carte_test.go — LE MESSAGE D'UNE CARTE INCONNUE NE CITE JAMAIS LA CARTE.
//
// Ce message est PUBLIE tel quel par le handler. Y citer la carte demandee — ce que faisait
// `fmt.Errorf("%w (%q)", ...)` — faisait differer le corps d'une carte legitime jamais jouee
// de celui d'un map_id refuse par `MapIDValide`, qui n'a rien a citer. La ronde 1 avait rendu
// le CODE indiscernable ; le LIBELLE ne l'etait pas (revue R2, P1). Le detail vit desormais
// au journal.
//
// Ce fichier verrouille la forme cote SERVICE ; la ceinture cote handler (`mapTacticalError`
// publie le message canonique quoi qu'on lui donne) est eprouvee par
// `api/handlers/tactical_oracle_test.go`.

import (
	"context"
	"errors"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
)

// TestRaster_CarteInconnue_NeCiteJamaisLaCarte — univers vide : sentinelle NUE.
func TestRaster_CarteInconnue_NeCiteJamaisLaCarte(t *testing.T) {
	const carte = "carte-jamais-jouee-zzz"
	repo := &mockTacticalRepo{} // aucun match dans l'univers
	svc := NewTacticalService(repo, capsPositionsSeules(), tsMoi)

	_, err := svc.Raster(context.Background(),
		tsDemande(repo, carte, domain.TacticalQuestionMorts, domain.TacticalQuiMoi))
	if !errors.Is(err, domain.ErrTacticalCarteInconnue) {
		t.Fatalf("err = %v, attendue ErrTacticalCarteInconnue", err)
	}
	if strings.Contains(err.Error(), carte) {
		t.Errorf("le message CITE la carte demandee : %q", err.Error())
	}
	// Egalite STRICTE au message canonique, pas seulement absence de la carte : tout
	// enrobage — meme sans citer la carte — rendrait ce refus distinguable des autres 404
	// de la meme famille.
	if err.Error() != domain.ErrTacticalCarteInconnue.Error() {
		t.Errorf("message = %q, attendu le canonique %q",
			err.Error(), domain.ErrTacticalCarteInconnue.Error())
	}
}

// TestRaster_CarteVide_MemeMessageCanonique — l'autre producteur du meme 404
// (`validerLecture`, carte vide) rend EXACTEMENT le meme message. Les deux refus de la
// famille doivent etre indiscernables entre eux comme du refus de validation.
func TestRaster_CarteVide_MemeMessageCanonique(t *testing.T) {
	repo := &mockTacticalRepo{}
	svc := NewTacticalService(repo, capsPositionsSeules(), tsMoi)

	_, err := svc.Raster(context.Background(),
		tsDemande(repo, "", domain.TacticalQuestionMorts, domain.TacticalQuiMoi))
	if !errors.Is(err, domain.ErrTacticalCarteInconnue) {
		t.Fatalf("err = %v, attendue ErrTacticalCarteInconnue", err)
	}
	if err.Error() != domain.ErrTacticalCarteInconnue.Error() {
		t.Errorf("message = %q, attendu le canonique %q",
			err.Error(), domain.ErrTacticalCarteInconnue.Error())
	}
}

// TestRaster_QuestionEtAxe_NOMMENT la valeur refusee — et c'est VOULU.
//
// Ces deux-la sont des 400 sur des parametres de REQUETE, a validation unique : il n'existe
// aucune seconde frontiere dont il faudrait les rendre indiscernables, et nommer la valeur
// rejetee est ce qui rend le 400 utile. La regle du message canonique ne vaut que pour le
// 404 de carte, qui a DEUX producteurs.
func TestRaster_QuestionEtAxeNommentLaValeurRefusee(t *testing.T) {
	repo := &mockTacticalRepo{}
	svc := NewTacticalService(repo, capsPositionsSeules(), tsMoi)

	// « temps » A CESSE D'ETRE UN EXEMPLE DE VALEUR INCONNUE le 2026-09-06 (phase 6) :
	// c'est desormais la quatrieme question servie, l'occupation. La fixture prend une
	// valeur qui n'a aucune chance d'entrer au vocabulaire.
	_, err := svc.Raster(context.Background(), tsDemande(repo, tsCarte, "tout-sauf-ca", domain.TacticalQuiMoi))
	if !errors.Is(err, domain.ErrTacticalQuestionInconnue) || !strings.Contains(err.Error(), "tout-sauf-ca") {
		t.Errorf("question : err = %v, attendue la sentinelle NOMMANT « tout-sauf-ca »", err)
	}

	_, err = svc.Raster(context.Background(), tsDemande(repo, tsCarte, domain.TacticalQuestionMorts, "tout-le-monde"))
	if !errors.Is(err, domain.ErrTacticalQuiInconnu) || !strings.Contains(err.Error(), "tout-le-monde") {
		t.Errorf("axe : err = %v, attendue la sentinelle NOMMANT « tout-le-monde »", err)
	}
}

// TestMapsPlayed_MiniPlanDesVignettes — LE MINI-PLAN « OU JE MEURS » D'UNE VIGNETTE (lot F,
// 2026-09-13, maquette 034b1915).
//
// Ce que ce test attrape : un mini-plan qui ne serait pas servi, un pas qui redeviendrait
// adaptatif (deux vignettes a des finesses differentes ne se comparent plus a l'oeil), et
// une carte SANS mort mesuree qui recevrait un cadre vide au lieu de rien.
func TestMapsPlayed_MiniPlanDesVignettes(t *testing.T) {
	repo := &mockTacticalRepo{
		maps: []domain.TacticalMapRow{
			{MapID: "a", MapName: "Aquarius", Matchs: domain.PlancherMatchsParCarte, Victoires: 6, Defaites: 4},
			{MapID: "b", MapName: "Bazaar", Matchs: domain.PlancherMatchsParCarte, Victoires: 5, Defaites: 5},
		},
		// Trois matchs DISTINCTS au meme endroit : la cellule passe le plancher.
		mortsParCarte: map[string][]domain.PositionSample{
			"a": {
				{MatchID: "m1", X: 2, Y: 2}, {MatchID: "m2", X: 2.5, Y: 2.5},
				{MatchID: "m3", X: 3, Y: 3},
			},
		},
	}
	// LE PERIMETRE EST EXPLICITE : le double honore la liste blanche comme le vrai lecteur,
	// et une liste VIDE veut dire « aucun match », pas « tous ».
	page, err := NewTacticalService(repo, capsCompletes(), tsMoi).
		MapsPlayed(context.Background(), domain.TacticalScope{MatchIDs: []string{"m1", "m2", "m3"}})
	if err != nil {
		t.Fatalf("MapsPlayed: %v", err)
	}
	if len(page.Cartes) != 2 {
		t.Fatalf("cartes = %d, attendu 2", len(page.Cartes))
	}
	a := page.Cartes[0]
	if len(a.Cellules) == 0 {
		t.Fatal("la vignette de la carte a n'a aucun mini-plan")
	}
	if a.PasM != domain.TacticalTuilePasM {
		t.Errorf("pas = %v m, attendu %v m (FIXE, jamais adaptatif)", a.PasM, domain.TacticalTuilePasM)
	}
	if !a.Bornes.Valide {
		t.Error("les bornes de la vignette ne sont pas valides")
	}
	if a.Cellules[0].Matchs != 3 {
		t.Errorf("matchs distincts de la cellule = %d, attendu 3", a.Cellules[0].Matchs)
	}
	b := page.Cartes[1]
	if len(b.Cellules) != 0 || b.Bornes.Valide {
		t.Errorf("la carte b n'a aucune mort mesuree : elle ne doit porter aucun cadre (%+v)", b)
	}
}

// TestMapsPlayed_MiniPlanEnEchecNArretePasLaGrille : la grille se lit sur le REGISTRE, qui
// ne depend d'aucun film. Une lecture de positions en echec laisse les vignettes avec leur
// seul fond — une degradation, jamais une panne de page.
func TestMapsPlayed_MiniPlanEnEchecNArretePasLaGrille(t *testing.T) {
	repo := &mockTacticalRepo{
		maps: []domain.TacticalMapRow{
			{MapID: "a", MapName: "Aquarius", Matchs: domain.PlancherMatchsParCarte, Victoires: 6, Defaites: 4},
		},
		errMortsParCarte: errors.New("lecture en echec"),
	}
	page, err := NewTacticalService(repo, capsCompletes(), tsMoi).
		MapsPlayed(context.Background(), domain.TacticalScope{})
	if err != nil {
		t.Fatalf("MapsPlayed: %v (une vignette muette n'est pas une panne de grille)", err)
	}
	if len(page.Cartes) != 1 || len(page.Cartes[0].Cellules) != 0 {
		t.Fatalf("cartes = %+v, attendu une carte sans mini-plan", page.Cartes)
	}
}
