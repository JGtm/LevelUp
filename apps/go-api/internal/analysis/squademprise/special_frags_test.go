package squademprise

import (
	"math"
	"testing"

	"levelup/go-api/internal/analysis/sessionusage"
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

// armesDuRendement rend les frags et l'exposition (prises) du rendement des armes spéciales.
func armesDuRendement(t *testing.T, b domain.SquadEmpriseBlock) (frags, prises domain.SquadEmpriseCount) {
	t.Helper()
	for _, p := range b.Production {
		if p.Resource == domain.EmpriseResourcePowerWeapon && p.Exposure != nil {
			return p.Exposure.Kills, p.Exposure.Value
		}
	}
	t.Fatalf("production sans rendement des armes spéciales : %+v", b.Production)
	return
}

// TestBuild_FragsAuxArmesSpeciales_FamilleHorsPuissanceEcartee — une famille du socle de puissance
// qu'un autre emplacement du match porte aussi (râtelier, départ, emplacement non identifié), ou
// sans clé de registre, sort du rendement DES DEUX CÔTÉS : ni ses frags ni ses prises. Une ligne à
// zéro prise ne qualifie ni n'écarte une famille.
func TestBuild_FragsAuxArmesSpeciales_FamilleHorsPuissanceEcartee(t *testing.T) {
	cas := map[string]func(*Input){
		"râtelier": func(in *Input) {
			in.Film.PadTiers = append(in.Film.PadTiers, tier("m1", "E1", domain.PadTierGround, lr, 1))
		},
		"départ": func(in *Input) {
			in.Film.PadTiers = append(in.Film.PadTiers, tier("m1", "E1", domain.PadTierBase, lr, 1))
		},
		"non identifié": func(in *Input) {
			in.Film.PadTiers = append(in.Film.PadTiers, tier("m1", "E1", domain.PadTierUnclassified, lr, 1))
		},
		"sans clé registre": func(in *Input) { in.Weapons[lr] = squadformes.WeaponInfo{Label: "M41 SPNKr"} },
	}
	for nom, muter := range cas {
		in := entreeAvecJournal()
		muter(&in)
		frags, prises := armesDuRendement(t, Build(in))
		if frags != (domain.SquadEmpriseCount{}) || prises != (domain.SquadEmpriseCount{}) {
			t.Errorf("%s : rendement frags %+v / prises %+v, attendu 0 / 0 des deux côtés", nom, frags, prises)
		}
	}
	// Zéro prise : une ligne de râtelier vide n'écarte pas la famille, une ligne de puissance vide
	// ne qualifie pas une autre famille.
	in := entreeAvecJournal()
	in.Weapons["0a000009"] = squadformes.WeaponInfo{WeaponKey: "hammer"}
	in.Film.PadTiers = append(in.Film.PadTiers,
		tier("m1", "E1", domain.PadTierGround, lr, 0), tier("m1", "E1", domain.PadTierPower, "0a000009", 0))
	in.Journal.Rows = append(in.Journal.Rows, JournalKillRow{MatchID: "m1", XUID: "P", WeaponKey: "hammer", Kills: 9})
	frags, prises := armesDuRendement(t, Build(in))
	if frags != (domain.SquadEmpriseCount{Us: 5, Them: 2}) || prises != (domain.SquadEmpriseCount{Us: 2, Them: 1}) {
		t.Errorf("lignes à zéro prise : rendement frags %+v / prises %+v, attendu 5 / 2 et 2 / 1", frags, prises)
	}
}

// TestBuild_FragsAuxArmesSpeciales_SuicideTrahisonBot — un suicide et une trahison ne sont pas des
// frags ; un tueur bot se range d'après sa victime : adversaire d'une victime de notre camp, nôtre
// pour une victime de l'autre camp dans un match à deux camps, non compté au-delà.
func TestBuild_FragsAuxArmesSpeciales_SuicideTrahisonBot(t *testing.T) {
	in := entreeAvecJournal()
	in.Journal.Rows = []JournalKillRow{
		{MatchID: "m1", XUID: "A", VictimXUID: "A", WeaponKey: keyLance, Kills: 1},  // suicide
		{MatchID: "m1", XUID: "X", VictimXUID: "X", WeaponKey: keyLance, Kills: 1},  // suicide, camp inconnu
		{MatchID: "m1", XUID: "A", VictimXUID: "P", WeaponKey: keyLance, Kills: 1},  // trahison
		{MatchID: "m1", XUID: "A", VictimXUID: "E1", WeaponKey: keyLance, Kills: 1}, // frag
		{MatchID: "m1", VictimXUID: "P", WeaponKey: keyLance, Kills: 2},             // bot adverse
		{MatchID: "m1", VictimXUID: "E2", WeaponKey: keyLance, Kills: 3},            // bot de notre camp
		{MatchID: "m1", VictimXUID: "inconnu", WeaponKey: keyLance, Kills: 4},       // camp indéductible
	}
	if pwk := Build(in).Matches[0].PowerWeaponKills; pwk == nil || *pwk != (domain.SquadEmpriseCount{Us: 4, Them: 2}) {
		t.Errorf("frags aux armes spéciales = %+v, attendu 4 / 2", pwk)
	}
	// Trois camps : le bot qui tue un adversaire n'est plus forcément des nôtres.
	in.Film.Participants = append(in.Film.Participants, sessionusage.ParticipantRow{MatchID: "m1", XUID: "T", TeamID: equipe(2)})
	if pwk := Build(in).Matches[0].PowerWeaponKills; pwk == nil || *pwk != (domain.SquadEmpriseCount{Us: 1, Them: 2}) {
		t.Errorf("trois camps : frags aux armes spéciales = %+v, attendu 1 / 2", pwk)
	}
}

// TestBuild_FragsAuxArmesSpeciales_CatalogueVide — sans catalogue d'armes, aucune famille ne se
// traduit : la feuille de match, jamais un 0 / 0.
func TestBuild_FragsAuxArmesSpeciales_CatalogueVide(t *testing.T) {
	in := entreeAvecJournal()
	in.Weapons = map[string]squadformes.WeaponInfo{}
	if pwk := Build(in).Matches[0].PowerWeaponKills; pwk == nil || *pwk != (domain.SquadEmpriseCount{Us: 3, Them: 2}) {
		t.Errorf("catalogue vide : frags aux armes spéciales = %+v, attendu la feuille 3 / 2", pwk)
	}
}
