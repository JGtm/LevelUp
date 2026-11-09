package service

// tactical_service_fonds_test.go — LE CALAGE DU FOND PORTÉ PAR LA CARTE DE LA GRILLE (plan
// backlog 2026-10-16, E7.2, D-10).
//
// Ce que ces tests attrapent : un calage qui ne serait pas servi (la vignette retomberait sur
// une requête par carte), un calage posé sur une carte SOUS le plancher (qui n'a pas de
// vignette), une carte sans fond qui recevrait un calage vide au lieu de rien, et une panne de
// lecture du fond qui arrêterait la grille au lieu de la laisser sans fond.

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/replaydoc"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/port"
)

// rejeuAvecFonds sert un calage par carte ; une carte absente de `fonds` n'a pas de fond figé,
// une carte de `pannes` échoue autrement que par l'absence.
type rejeuAvecFonds struct {
	stubReplayService
	fonds  map[string]replaydoc.MapBackgroundCalibration
	pannes map[string]error
	lus    []string
}

func (r *rejeuAvecFonds) MapBackgroundForMap(_ context.Context, mapID string) (*replaydoc.MapBackground, error) {
	r.lus = append(r.lus, mapID)
	if err, ok := r.pannes[mapID]; ok {
		return nil, err
	}
	cal, ok := r.fonds[mapID]
	if !ok {
		return nil, port.ErrMapBackgroundNotAvailable
	}
	return &replaydoc.MapBackground{Calibration: cal}, nil
}

func repoDeTroisCartes() *mockTacticalRepo {
	return &mockTacticalRepo{maps: []domain.TacticalMapRow{
		{MapID: "a", MapName: i18n.FromColumns("Aquarius", ""), Matchs: domain.PlancherMatchsParCarte, Victoires: 6, Defaites: 4},
		{MapID: "b", MapName: i18n.FromColumns("Bazaar", ""), Matchs: domain.PlancherMatchsParCarte + 2, Victoires: 5, Defaites: 5},
		{MapID: "petite", MapName: i18n.FromColumns("Petite", ""), Matchs: domain.PlancherMatchsParCarte - 1, Victoires: 3, Defaites: 3},
	}}
}

func TestMapsPlayed_CalageDuFondPorteParLaCarte(t *testing.T) {
	cal := replaydoc.MapBackgroundCalibration{MetersPerPixel: 0.5, OriginX: -10, OriginY: 20, WidthPx: 800, HeightPx: 600}
	rejeu := &rejeuAvecFonds{fonds: map[string]replaydoc.MapBackgroundCalibration{
		"a": cal,
		// La carte sous le plancher a un fond, mais n'a pas de vignette : il n'est pas lu.
		"petite": cal,
	}}
	svc := NewTacticalService(repoDeTroisCartes(), capsCompletes(), tsMoi).WithReplay(rejeu)

	page, err := svc.MapsPlayed(context.Background(), domain.TacticalScope{})
	if err != nil {
		t.Fatalf("MapsPlayed: %v", err)
	}
	par := map[string]domain.TacticalMapCard{}
	for _, c := range page.Cartes {
		par[c.MapID] = c
	}
	if got := par["a"].CalageFond; got == nil || *got != cal {
		t.Errorf("carte a : calage = %+v, attendu %+v", got, cal)
	}
	if par["b"].CalageFond != nil {
		t.Errorf("carte b sans fond figé : calage = %+v, attendu nil", par["b"].CalageFond)
	}
	if par["petite"].CalageFond != nil {
		t.Errorf("carte sous le plancher : calage = %+v, attendu nil (aucune vignette)", par["petite"].CalageFond)
	}
	for _, id := range rejeu.lus {
		if id == "petite" {
			t.Errorf("le fond d'une carte sous le plancher a été lu (%v)", rejeu.lus)
		}
	}
}

func TestMapsPlayed_FondEnPanneNArretePasLaGrille(t *testing.T) {
	var trace bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&trace, nil))
	cal := replaydoc.MapBackgroundCalibration{MetersPerPixel: 1, WidthPx: 10, HeightPx: 10}
	rejeu := &rejeuAvecFonds{
		fonds:  map[string]replaydoc.MapBackgroundCalibration{"b": cal},
		pannes: map[string]error{"a": errors.New("registre illisible")},
	}
	svc := NewTacticalService(repoDeTroisCartes(), capsCompletes(), tsMoi).WithReplay(rejeu).WithLogger(logger)

	page, err := svc.MapsPlayed(context.Background(), domain.TacticalScope{})
	if err != nil {
		t.Fatalf("MapsPlayed: %v (un fond illisible n'est pas une panne de grille)", err)
	}
	if page.Cartes[0].CalageFond != nil {
		t.Errorf("carte a en panne : calage = %+v, attendu nil", page.Cartes[0].CalageFond)
	}
	if page.Cartes[1].CalageFond == nil {
		t.Error("carte b : la panne de a a privé b de son calage")
	}
	if !strings.Contains(trace.String(), "registre illisible") {
		t.Errorf("la panne n'est pas journalisée : %q", trace.String())
	}
}

func TestMapsPlayed_SansServiceDeRejeuAucuneCarteNePorteDeCalage(t *testing.T) {
	svc := NewTacticalService(repoDeTroisCartes(), capsCompletes(), tsMoi)
	page, err := svc.MapsPlayed(context.Background(), domain.TacticalScope{})
	if err != nil {
		t.Fatalf("MapsPlayed: %v", err)
	}
	for _, c := range page.Cartes {
		if c.CalageFond != nil {
			t.Errorf("carte %s : calage %+v sans service de rejeu", c.MapID, c.CalageFond)
		}
	}
}
