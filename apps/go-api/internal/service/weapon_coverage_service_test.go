package service

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/port"
)

// fauxWeaponRepo : des frags fixes ; ClassifyWeaponIDs rend les ensembles donnés.
type fauxWeaponRepo struct {
	byID      []port.WeaponKillCount
	bySource  []port.KillSourceCount
	meta      bool
	inReg     map[int64]bool
	inLabel   map[int64]bool
	keys      map[string]bool
	idsErr    error
	classedID []int64
}

func (f *fauxWeaponRepo) KillsByWeaponID(context.Context) ([]port.WeaponKillCount, error) {
	return f.byID, f.idsErr
}

func (f *fauxWeaponRepo) KillsBySourceTag(context.Context) ([]port.KillSourceCount, error) {
	return f.bySource, nil
}

func (f *fauxWeaponRepo) HasMetadata() bool { return f.meta }

func (f *fauxWeaponRepo) ClassifyWeaponIDs(_ context.Context, _ string, ids []int64) (map[int64]bool, map[int64]bool) {
	f.classedID = ids
	return f.inReg, f.inLabel
}

func (f *fauxWeaponRepo) RegistryWeaponKeys(context.Context, string) (map[string]bool, error) {
	return f.keys, nil
}

type fauxClassifier map[uint32]string

func (c fauxClassifier) KillSourceRegistryKey(tag uint32) (string, bool) {
	k, ok := c[tag]
	return k, ok
}

func serviceArmes(repo *fauxWeaponRepo, classifier port.KillSourceClassifier) *WeaponCoverageService {
	return NewWeaponCoverageService(WeaponCoverageDeps{
		Open:       func(context.Context, string) (port.WeaponCoverageRepo, func(), error) { return repo, func() {}, nil },
		Classifier: func(string) port.KillSourceClassifier { return classifier },
	})
}

// TestWeaponCoverage_ParIdentifiantDArme : registre, libellé seul, non résolu ; pourcentages ; tri des non
// résolus par frags décroissants.
func TestWeaponCoverage_ParIdentifiantDArme(t *testing.T) {
	repo := &fauxWeaponRepo{
		meta:    true,
		byID:    []port.WeaponKillCount{{WeaponID: 10, Kills: 5}, {WeaponID: 20, Kills: 7}, {WeaponID: 30, Kills: 1}, {WeaponID: 40, Kills: 9}},
		inReg:   map[int64]bool{10: true},
		inLabel: map[int64]bool{10: true, 20: true},
	}
	resp, err := serviceArmes(repo, nil).WeaponCoverage(context.Background(), "halo_infinite")
	if err != nil {
		t.Fatal(err)
	}
	if resp.DistinctWeapons != 4 || resp.ResolvedRegistry != 1 || resp.ResolvedLabel != 1 || resp.Unresolved != 2 {
		t.Fatalf("couverture = %+v", resp)
	}
	if resp.CoveragePercent != 50 || resp.RegistryPercent != 25 {
		t.Errorf("pourcentages = %v / %v, attendu 50 / 25", resp.CoveragePercent, resp.RegistryPercent)
	}
	if len(resp.TopUnresolved) != 2 || resp.TopUnresolved[0].WeaponID != "40" || resp.TopUnresolved[1].WeaponID != "30" {
		t.Errorf("non résolues = %+v, attendu 40 puis 30", resp.TopUnresolved)
	}
}

// TestWeaponCoverage_SansMetadataToutEstNonResolu : sans metadata, aucune classification n'est demandée.
func TestWeaponCoverage_SansMetadataToutEstNonResolu(t *testing.T) {
	repo := &fauxWeaponRepo{byID: []port.WeaponKillCount{{WeaponID: 10, Kills: 5}}}
	resp, err := serviceArmes(repo, nil).WeaponCoverage(context.Background(), "halo_infinite")
	if err != nil || resp.Unresolved != 1 || repo.classedID != nil {
		t.Fatalf("%+v, %v, classés %v", resp, err, repo.classedID)
	}
}

// TestWeaponCoverage_ParSourceDeDegat : l'unité est le tag ; ResolvedLabel reste à zéro ; une clé absente du
// registre n'empêche pas la page d'être servie.
func TestWeaponCoverage_ParSourceDeDegat(t *testing.T) {
	repo := &fauxWeaponRepo{
		meta:     true,
		bySource: []port.KillSourceCount{{SourceTag: 0x0a000001, Kills: 4}, {SourceTag: 0x0a000002, Kills: 2}, {SourceTag: 0xff, Kills: 6}},
		keys:     map[string]bool{"arme_a": true},
	}
	resp, err := serviceArmes(repo, fauxClassifier{0x0a000001: "arme_a", 0x0a000002: "arme_b"}).
		WeaponCoverage(context.Background(), "halo_infinite")
	if err != nil {
		t.Fatal(err)
	}
	if resp.DistinctWeapons != 3 || resp.ResolvedRegistry != 2 || resp.ResolvedLabel != 0 || resp.Unresolved != 1 {
		t.Fatalf("couverture = %+v", resp)
	}
	if len(resp.TopUnresolved) != 1 || resp.TopUnresolved[0].WeaponID != "0x000000ff" || resp.TopUnresolved[0].Kills != 6 {
		t.Errorf("non résolues = %+v", resp.TopUnresolved)
	}
	if resp.RegistryPercent != resp.CoveragePercent {
		t.Errorf("voie source : registre = couverture, obtenu %v / %v", resp.RegistryPercent, resp.CoveragePercent)
	}
}

// TestWeaponCoverage_Erreurs : l'ouverture et la lecture en échec remontent telles quelles.
func TestWeaponCoverage_Erreurs(t *testing.T) {
	boom := errors.New("boom")
	repo := &fauxWeaponRepo{idsErr: boom}
	if _, err := serviceArmes(repo, nil).WeaponCoverage(context.Background(), "x"); !errors.Is(err, boom) {
		t.Errorf("lecture : %v", err)
	}
	svc := NewWeaponCoverageService(WeaponCoverageDeps{
		Open:       func(context.Context, string) (port.WeaponCoverageRepo, func(), error) { return nil, nil, boom },
		Classifier: func(string) port.KillSourceClassifier { return nil },
	})
	if _, err := svc.WeaponCoverage(context.Background(), "x"); !errors.Is(err, boom) {
		t.Errorf("ouverture : %v", err)
	}
}
