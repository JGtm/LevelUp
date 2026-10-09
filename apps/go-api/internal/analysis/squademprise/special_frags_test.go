package squademprise

import (
	"math"
	"testing"

	"levelup/go-api/internal/analysis/squadformes"
	"levelup/go-api/internal/domain"
)

const (
	keyLance  = "rocket_launcher" // clé de registre de `lr` (entreeUnMatch)
	keyFusil  = "battle_rifle"    // clé de registre de `br`, arme de râtelier
	keyHorsSo = "sniper"          // arme spéciale du jeu, absente des socles du match
)

// entreeAvecJournal — le match d'entreeUnMatch, journal des morts publiable : chez nous 5 frags à
// l'arme du socle de puissance (Alpha 4, le reste du camp 1), chez eux 2 ; des frags à une arme de
// râtelier et à une arme absente des socles, qui ne sont pas des frags aux armes spéciales. La
// feuille de match (PowerKills : 3 / 2) compte, elle, sa propre liste d'armes.
func entreeAvecJournal() Input {
	in := entreeUnMatch()
	in.Weapons = map[string]squadformes.WeaponInfo{
		lr: {WeaponKey: keyLance, Label: "M41 SPNKr"},
		br: {WeaponKey: keyFusil, Label: "BR75"},
	}
	in.Journal = &JournalRead{
		Read: map[string]bool{"m1": true},
		Rows: []JournalKillRow{
			{MatchID: "m1", XUID: "A", WeaponKey: keyLance, Kills: 4},
			{MatchID: "m1", XUID: "R", WeaponKey: keyLance, Kills: 1},
			{MatchID: "m1", XUID: "E1", WeaponKey: keyLance, Kills: 2},
			{MatchID: "m1", XUID: "P", WeaponKey: keyFusil, Kills: 6},
			{MatchID: "m1", XUID: "E2", WeaponKey: keyHorsSo, Kills: 3},
		},
	}
	return in
}

// TestBuild_FragsAuxArmesSpeciales_JournalDuFilm — sur un match aux niveaux mesurés et au journal
// publiable, les frags aux armes spéciales sont ceux des armes des socles de puissance DU MATCH,
// pas la liste d'armes de puissance de la feuille de match (cas du Needler d'un socle de puissance,
// que la feuille ne compte pas). Le rendement divise ces frags par les prises des mêmes armes.
func TestBuild_FragsAuxArmesSpeciales_JournalDuFilm(t *testing.T) {
	b := Build(entreeAvecJournal())
	want := domain.SquadEmpriseCount{Us: 5, Them: 2}
	if pwk := b.Matches[0].PowerWeaponKills; pwk == nil || *pwk != want {
		t.Fatalf("frags aux armes spéciales = %+v, attendu %+v (journal, armes du socle de puissance)", pwk, want)
	}
	armes := b.Production[1]
	if armes.Resource != domain.EmpriseResourcePowerWeapon || armes.Kills != want || armes.Exposure == nil ||
		armes.Exposure.Kills != want || armes.Exposure.Value != (domain.SquadEmpriseCount{Us: 2, Them: 1}) {
		t.Fatalf("production armes = %+v / %+v", armes.Kills, armes.Exposure)
	}
	// 5 frags pour 2 prises contre 2 pour 1 : 2,5 / 2 − 1.
	if armes.RelativeGap == nil || math.Abs(*armes.RelativeGap-(2.5/2-1)) > 1e-9 {
		t.Errorf("écart relatif = %v, attendu 2,5 / 2 − 1", armes.RelativeGap)
	}
}

// TestBuild_FragsAuxArmesSpeciales_RepliFeuille — journal non publiable, niveaux non mesurés, ou
// journal non lu : la feuille de match.
func TestBuild_FragsAuxArmesSpeciales_RepliFeuille(t *testing.T) {
	feuille := domain.SquadEmpriseCount{Us: 3, Them: 2}
	cas := map[string]func(*Input){
		"journal non publiable": func(in *Input) { in.Journal.Read = map[string]bool{} },
		"journal non lu":        func(in *Input) { in.Journal = nil },
		"niveaux non mesurés":   func(in *Input) { in.Film.PadTiers = nil },
	}
	for nom, muter := range cas {
		in := entreeAvecJournal()
		muter(&in)
		if pwk := Build(in).Matches[0].PowerWeaponKills; pwk == nil || *pwk != feuille {
			t.Errorf("%s : frags aux armes spéciales = %+v, attendu la feuille %+v", nom, pwk, feuille)
		}
	}
}

// TestBuild_FragsAuxArmesSpeciales_FamillePartagee — une famille posée sur un socle de puissance ET
// sur un râtelier du même match ne compte pas : ses frags ne se séparent pas entre les deux.
func TestBuild_FragsAuxArmesSpeciales_FamillePartagee(t *testing.T) {
	in := entreeAvecJournal()
	in.Film.PadTiers = append(in.Film.PadTiers, tier("m1", "E1", domain.PadTierGround, lr, 1))
	if pwk := Build(in).Matches[0].PowerWeaponKills; pwk == nil || *pwk != (domain.SquadEmpriseCount{}) {
		t.Errorf("frags aux armes spéciales = %+v, attendu 0 / 0 (famille aussi au râtelier)", pwk)
	}
}
