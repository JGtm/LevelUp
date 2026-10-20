package service

// tactical_service_cellule_icone_test.go — l'icône d'arme et le mode normalisé de la mini-tuile
// « Rejeu » du détail d'une zone.

import (
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
)

// fakeIcones double l'adapter d'assets : seule KillSourceIcon répond (les autres méthodes
// paniquent par l'interface nil embarquée, ce qui prouve qu'elles ne sont pas appelées).
type fakeIcones struct {
	games.TitleAssetURLAdapter
	icones map[uint32]canonical.KillSourceIcon
}

func (f fakeIcones) KillSourceIcon(tag uint32) (canonical.KillSourceIcon, bool) {
	ic, ok := f.icones[tag]
	return ic, ok
}

// TestDetail_IconeDArme : l'icône est publiée pour la source qui en a une (même sans nom de
// registre), absente pour celle que l'adapter ne connaît pas ; sans adapter, aucune icône.
func TestDetail_IconeDArme(t *testing.T) {
	repo, matchs := corpusDetail()
	src := sourcesCompletes(matchs)
	adapter := fakeIcones{icones: map[uint32]canonical.KillSourceIcon{
		tagBR: {WeaponKey: "br75", ImageURL: "/static/weapons/br75.png", Tinted: true},
	}}
	req := celluleDemande(repo, domain.TacticalQuestionGagne, domain.TacticalQuiMoi, 4, 4)

	got := celluleLue(t, src.service(repo, capsCompletes()).WithAssetURL(adapter), req)
	c := got.Contributions
	if c[0].ArmeImageURL != "/static/weapons/br75.png" || !c[0].ArmeImageTinted {
		t.Errorf("BR75 : icône %q tinted %v, want l'URL de l'adapter, teintée", c[0].ArmeImageURL, c[0].ArmeImageTinted)
	}
	for i := 1; i < len(c); i++ {
		if c[i].ArmeImageURL != "" || c[i].ArmeImageTinted {
			t.Errorf("contribution %d : icône %q tinted %v, want aucune (source inconnue de l'adapter ou sans source)",
				i, c[i].ArmeImageURL, c[i].ArmeImageTinted)
		}
	}

	repo2, matchs2 := corpusDetail()
	sans := celluleLue(t, sourcesCompletes(matchs2).service(repo2, capsCompletes()),
		celluleDemande(repo2, domain.TacticalQuestionGagne, domain.TacticalQuiMoi, 4, 4))
	for i, x := range sans.Contributions {
		if x.ArmeImageURL != "" {
			t.Errorf("sans adapter, contribution %d : icône %q, want aucune", i, x.ArmeImageURL)
		}
	}
}

// TestDetail_ModeNormalise : le libellé de mode du match passe par la normalisation du mode avec la
// carte du match (« Arena:Slayer sur Ruelles » → « Slayer »).
func TestDetail_ModeNormalise(t *testing.T) {
	repo, matchs := corpusDetail()
	matchs.rows[0].Summary.PairMode = &canonical.AssetReference{DefaultLabel: "Arena:Slayer sur Ruelles"}
	matchs.rows[0].Summary.Map = &canonical.AssetReference{DefaultLabel: "Ruelles"}
	got := celluleLue(t, sourcesCompletes(matchs).service(repo, capsCompletes()),
		celluleDemande(repo, domain.TacticalQuestionGagne, domain.TacticalQuiMoi, 4, 4))
	if m := got.Contributions[0].ModeLabel; m != "Slayer" {
		t.Errorf("mode = %q, want « Slayer » (conteneur et carte retirés)", m)
	}
}
