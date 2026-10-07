package squademprise

// unclassified_test.go — LES PRISES SUR UN EMPLACEMENT NON IDENTIFIÉ, au grain du match (Vue match,
// ligne sous « Contrôle des ressources, par match ») : les lignes `non_classe` des niveaux de socle,
// équipe contre adversaire, sur un match filmé au camp connu, quel que soit l'état des niveaux ; nil
// sans ligne non classée, sans film ou au camp inconnu. Témoin : le BTB du 24/07 (MESURES §3 :
// 19 pour l'équipe, 12 pour l'adversaire), réduit à quatre lignes.

import (
	"testing"

	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/domain"
)

func avecNonClassees(in Input, rows ...sessionusage.PadTierRow) Input {
	in.Film.PadTiers = append(append([]sessionusage.PadTierRow(nil), in.Film.PadTiers...), rows...)
	return in
}

func TestBuild_PrisesNonClassees(t *testing.T) {
	in := avecNonClassees(entreeUnMatch(),
		tier("m1", "P", domain.PadTierUnclassified, "0d000001", 7),
		tier("m1", "R", domain.PadTierUnclassified, "0d000002", 12),
		tier("m1", "E1", domain.PadTierUnclassified, "0d000001", 10),
		tier("m1", "E2", domain.PadTierUnclassified, "0d000003", 2),
	)
	got := Build(in).Matches[0].UnclassifiedPickups
	if got == nil || *got != (domain.SquadEmpriseCount{Us: 19, Them: 12}) {
		t.Fatalf("prises non classées = %+v, attendu 19 / 12", got)
	}
}

func TestBuild_PrisesNonClassees_NiveauxNonEtablis(t *testing.T) {
	in := entreeUnMatch()
	in.Film.PadTiers = []sessionusage.PadTierRow{
		{MatchID: "m1", XUID: "P", Tier: domain.PadTierUnclassified, WeaponFamily: "0d000001", Pickups: 3, PadsTotal: 8, PadsConfirmed: 0},
	}
	if got := Build(in).Matches[0].UnclassifiedPickups; got == nil || *got != (domain.SquadEmpriseCount{Us: 3}) {
		t.Fatalf("niveaux non établis : %+v, attendu 3 / 0 (les prises restent comptées)", got)
	}
}

func TestBuild_PrisesNonClassees_Absentes(t *testing.T) {
	if got := Build(entreeUnMatch()).Matches[0].UnclassifiedPickups; got != nil {
		t.Errorf("aucune ligne non classée : %+v, attendu nil", got)
	}
	sansCamp := avecNonClassees(entreeUnMatch(), tier("m1", "P", domain.PadTierUnclassified, "0d000001", 2))
	sansCamp.Film.Participants = nil
	if got := Build(sansCamp).Matches[0].UnclassifiedPickups; got != nil {
		t.Errorf("camp inconnu : %+v, attendu nil", got)
	}
	sansFilm := avecNonClassees(entreeUnMatch(), tier("m1", "P", domain.PadTierUnclassified, "0d000001", 2))
	sansFilm.Film.Films = map[string]sessionusage.FilmRow{}
	if got := Build(sansFilm).Matches[0].UnclassifiedPickups; got != nil {
		t.Errorf("sans film : %+v, attendu nil", got)
	}
}
