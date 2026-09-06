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
		tsDemande(carte, domain.TacticalQuestionMorts, domain.TacticalQuiMoi))
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
	svc := NewTacticalService(&mockTacticalRepo{}, capsPositionsSeules(), tsMoi)

	_, err := svc.Raster(context.Background(),
		tsDemande("", domain.TacticalQuestionMorts, domain.TacticalQuiMoi))
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
	svc := NewTacticalService(&mockTacticalRepo{}, capsPositionsSeules(), tsMoi)

	// « temps » A CESSE D'ETRE UN EXEMPLE DE VALEUR INCONNUE le 2026-09-06 (phase 6) :
	// c'est desormais la quatrieme question servie, l'occupation. La fixture prend une
	// valeur qui n'a aucune chance d'entrer au vocabulaire.
	_, err := svc.Raster(context.Background(), tsDemande(tsCarte, "tout-sauf-ca", domain.TacticalQuiMoi))
	if !errors.Is(err, domain.ErrTacticalQuestionInconnue) || !strings.Contains(err.Error(), "tout-sauf-ca") {
		t.Errorf("question : err = %v, attendue la sentinelle NOMMANT « tout-sauf-ca »", err)
	}

	_, err = svc.Raster(context.Background(), tsDemande(tsCarte, domain.TacticalQuestionMorts, "tout-le-monde"))
	if !errors.Is(err, domain.ErrTacticalQuiInconnu) || !strings.Contains(err.Error(), "tout-le-monde") {
		t.Errorf("axe : err = %v, attendue la sentinelle NOMMANT « tout-le-monde »", err)
	}
}
