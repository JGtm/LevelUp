package squademprise

import (
	"math"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/analysis/squadformes"
	"levelup/go-api/internal/domain"
)

const (
	camo = "powerup_camo"
	os   = "powerup_overshield"
	lr   = "0a000001" // arme de socle de puissance
	br   = "0b000001" // arme de râtelier
)

func equipe(v int) *int { return &v }

func entier(v int) *int { return &v }

var t0 = time.Date(2026, 9, 22, 19, 0, 0, 0, time.UTC)

// participants : moi (P) et mon coéquipier (A) en équipe 0, le reste du camp (R) aussi, deux
// adversaires (E1, E2) en équipe 1.
func participants(matchID string) []sessionusage.ParticipantRow {
	out := []sessionusage.ParticipantRow{}
	for xuid, team := range map[string]int{"P": 0, "A": 0, "R": 0, "E1": 1, "E2": 1} {
		out = append(out, sessionusage.ParticipantRow{MatchID: matchID, XUID: xuid, TeamID: equipe(team), PresentAtCompletion: true})
	}
	return out
}

func tier(matchID, xuid, t, fam string, n int) sessionusage.PadTierRow {
	return sessionusage.PadTierRow{MatchID: matchID, XUID: xuid, Tier: t, WeaponFamily: fam, Pickups: n, PadsTotal: 8, PadsConfirmed: 8}
}

// entreeUnMatch — un match filmé, niveaux mesurés, bonus et armes des deux côtés.
func entreeUnMatch() Input {
	return Input{
		PlayerXUID: "P",
		Players:    []domain.SessionUsageSquadPlayer{{XUID: "P", Gamertag: "Moi"}, {XUID: "A", Gamertag: "Alpha"}},
		Current:    []Match{{MatchID: "m1", StartTime: t0, SessionLabel: "s", Family: "Assassin"}},
		Film: &FilmData{
			Films: map[string]sessionusage.FilmRow{"m1": {MatchID: "m1", DurationMS: 600_000, PowerupPickups: map[string]int{camo: 5}}},
			Players: []sessionusage.PlayerRow{
				{MatchID: "m1", XUID: "P", CamoEpisodes: 1, CamoMS: 30_000, CamoKills: 2,
					TakenByFamily: map[string]int{camo: 2}, KeptByFamily: map[string]int{camo: 1}},
				{MatchID: "m1", XUID: "R", OvershieldEpisodes: 1, OvershieldMS: 30_000, OvershieldKills: 1,
					TakenByFamily: map[string]int{os: 1}},
				{MatchID: "m1", XUID: "E1", CamoEpisodes: 0, TakenByFamily: map[string]int{camo: 1},
					DroppedByFamily: map[string]int{camo: 1}},
			},
			Participants: participants("m1"),
			PadTiers: []sessionusage.PadTierRow{
				tier("m1", "A", domain.PadTierPower, lr, 2), tier("m1", "E2", domain.PadTierPower, lr, 1),
				tier("m1", "P", domain.PadTierGround, br, 1), tier("m1", "E1", domain.PadTierBase, "0c000001", 3),
			},
		},
		PowerKills: []PowerKillRow{
			{MatchID: "m1", XUID: "P", TeamID: equipe(0), Kills: entier(3)},
			{MatchID: "m1", XUID: "E1", TeamID: equipe(1), Kills: entier(2)},
		},
		Weapons: map[string]squadformes.WeaponInfo{lr: {WeaponKey: "rocket_launcher", Label: "M41 SPNKr"}},
	}
}

func ressource(b domain.SquadEmpriseBlock, res string) *domain.SquadEmpriseResource {
	for i := range b.Resources {
		if b.Resources[i].Resource == res {
			return &b.Resources[i]
		}
	}
	return nil
}

func TestBuild_BilanObjetsEtBonusPerdus(t *testing.T) {
	b := Build(entreeUnMatch())
	if b.MatchesTotal != 1 || b.MatchesMeasured != 1 {
		t.Fatalf("couverture %d/%d, attendu 1/1", b.MatchesMeasured, b.MatchesTotal)
	}
	bonus := ressource(b, domain.EmpriseResourcePowerup)
	if bonus == nil || bonus.Taken != (domain.SquadEmpriseCount{Us: 3, Them: 1}) {
		t.Fatalf("bonus = %+v, attendu 3 / 1", bonus)
	}
	if bonus.Outcomes == nil || bonus.Outcomes.Us.Kept+bonus.Outcomes.Us.Dropped != 1 ||
		bonus.Outcomes.Them.Kept+bonus.Outcomes.Them.Dropped != 1 {
		t.Errorf("bonus perdus = %+v, attendu 1 chez nous (gardé), 1 chez eux (lâché)", bonus.Outcomes)
	}
	armes := ressource(b, domain.EmpriseResourcePowerWeapon)
	if armes == nil || armes.Taken != (domain.SquadEmpriseCount{Us: 2, Them: 1}) {
		t.Fatalf("armes spéciales = %+v, attendu 2 / 1 (le niveau base n'en est pas)", armes)
	}
	if ressource(b, domain.EmpriseResourceRack) != nil {
		t.Error("les râteliers n'entrent pas au bilan")
	}
	// Objets de la soirée : bonus d'abord, puis armes ; le camouflage porte ses socles vidés.
	if len(b.Objects) != 4 || b.Objects[0].Key != camo || b.Objects[0].PadsEmptied == nil || *b.Objects[0].PadsEmptied != 5 {
		t.Fatalf("objets = %+v", b.Objects)
	}
	moi := b.Objects[0].Squad[0]
	if moi.XUID != "P" || moi.Taken != 2 || moi.Kept == nil || *moi.Kept != 1 {
		t.Errorf("ma part du camouflage = %+v, attendu 2 pris dont 1 gardé", moi)
	}
	lance := b.Objects[2]
	if lance.Key != lr || lance.Label != "M41 SPNKr" || lance.Squad[1].Taken != 2 || lance.Squad[1].Kept != nil {
		t.Errorf("lance-roquettes = %+v, attendu nommé, 2 prises d'Alpha, sans issue", lance)
	}
	surb := b.Objects[1]
	if surb.Key != os || surb.Squad[2].XUID != "" || surb.Squad[2].Taken != 1 {
		t.Errorf("surbouclier = %+v, attendu 1 prise au reste du camp", surb)
	}
}

func TestBuild_MatchParMatchEtProduction(t *testing.T) {
	b := Build(entreeUnMatch())
	m := b.Matches[0]
	if !m.HasFilm || !m.TeamKnown || m.Tiers != domain.EmpriseTiersMeasured {
		t.Fatalf("match = %+v", m)
	}
	if len(m.Resources) != 3 || m.Resources[2].Resource != domain.EmpriseResourceRack ||
		m.Resources[2].Taken != (domain.SquadEmpriseCount{Us: 1}) {
		t.Errorf("ressources du match = %+v, attendu bonus, armes spéciales, râtelier 1/0", m.Resources)
	}
	if m.PowerWeaponKills == nil || *m.PowerWeaponKills != (domain.SquadEmpriseCount{Us: 3, Them: 2}) {
		t.Errorf("frags aux armes spéciales = %+v, attendu 3 / 2", m.PowerWeaponKills)
	}
	if len(b.Production) != 2 {
		t.Fatalf("production = %+v", b.Production)
	}
	bonus := b.Production[0]
	if bonus.Kills != (domain.SquadEmpriseCount{Us: 3}) || bonus.Exposure.Value != (domain.SquadEmpriseCount{Us: 60_000}) {
		t.Errorf("bonus = %+v / %+v", bonus.Kills, bonus.Exposure)
	}
	if bonus.YieldUs == nil || math.Abs(*bonus.YieldUs-3) > 1e-9 || bonus.YieldThem != nil || bonus.RelativeGap != nil {
		t.Errorf("rendement bonus = %v / %v / %v, attendu 3 par minute, rien chez eux", bonus.YieldUs, bonus.YieldThem, bonus.RelativeGap)
	}
	armes := b.Production[1]
	if armes.RelativeGap == nil || math.Abs(*armes.RelativeGap-(1.5/2-1)) > 1e-9 {
		t.Errorf("écart relatif armes = %v, attendu 1,5 / 2 − 1", armes.RelativeGap)
	}
}

// TestBuild_SansFilm_SeuleLaFeuille — D10 : sans film, seuls les frags aux armes spéciales.
func TestBuild_SansFilm_SeuleLaFeuille(t *testing.T) {
	in := entreeUnMatch()
	in.Film, in.FilmUnavailable = nil, domain.EmpriseFilmUnsupported
	in.Timeline = in.Current
	b := Build(in)
	if b.FilmUnavailable != domain.EmpriseFilmUnsupported || b.MatchesMeasured != 0 || b.Habit != nil {
		t.Fatalf("bloc = %+v", b)
	}
	if len(b.Resources) != 0 || len(b.Objects) != 0 || len(b.Matches[0].Resources) != 0 {
		t.Errorf("grandeurs du film publiées sans film : %+v", b)
	}
	if len(b.Production) != 1 || b.Production[0].Resource != domain.EmpriseResourcePowerWeapon ||
		b.Production[0].Exposure != nil || b.Production[0].Kills != (domain.SquadEmpriseCount{Us: 3, Them: 2}) {
		t.Errorf("production = %+v, attendu frags aux armes spéciales seuls, sans exposition", b.Production)
	}
	if !b.Matches[0].TeamKnown || b.Matches[0].Tiers != "" {
		t.Errorf("match = %+v, camp connu par la feuille, pas d'état de niveaux sans film", b.Matches[0])
	}
}

// TestBuild_NiveauxNonMesures — sans passe des niveaux, ni armes spéciales ni râteliers ; carte
// hors référence : « non établis ».
func TestBuild_NiveauxNonMesures(t *testing.T) {
	in := entreeUnMatch()
	in.Film.PadTiers = nil
	b := Build(in)
	if b.Matches[0].Tiers != domain.EmpriseTiersNotMeasured || ressource(b, domain.EmpriseResourcePowerWeapon) != nil {
		t.Errorf("niveaux non mesurés : %+v", b.Matches[0])
	}
	if p := b.Production[1]; p.Exposure != nil || p.Kills != (domain.SquadEmpriseCount{Us: 3, Them: 2}) {
		t.Errorf("armes spéciales sans niveaux = %+v, attendu les frags seuls", p)
	}
	in = entreeUnMatch()
	for i := range in.Film.PadTiers {
		in.Film.PadTiers[i].PadsConfirmed = 0
	}
	if b := Build(in); b.Matches[0].Tiers != domain.EmpriseTiersUnestablished {
		t.Errorf("carte hors référence : Tiers = %q", b.Matches[0].Tiers)
	}
}

// TestBuild_CampInconnu — sans camp du joueur, aucun compte camp contre camp.
func TestBuild_CampInconnu(t *testing.T) {
	in := entreeUnMatch()
	in.Film.Participants = nil
	in.PowerKills = nil
	b := Build(in)
	if b.Matches[0].TeamKnown || len(b.Matches[0].Resources) != 0 || len(b.Resources) != 0 {
		t.Errorf("camp inconnu : %+v", b)
	}
	// Constat R2 de la revue L6.1 : filmé mais sans camp, le match n'apporte rien : il n'est pas
	// mesuré (le web l'écrit « camp inconnu », pas « rien à prendre »).
	if !b.Matches[0].HasFilm || b.MatchesMeasured != 0 {
		t.Errorf("camp inconnu : has_film = %v, matchs mesurés = %d ; attendu filmé, 0 mesuré",
			b.Matches[0].HasFilm, b.MatchesMeasured)
	}
}

// TestBuild_SansEchelleDeTemps_HorsDuRendementDesBonus — constat R1 de la revue L6.1 : un match
// filmé sans échelle de temps (durée nulle, temps d'effet à 0 que le schéma interdit de croire)
// sort du rendement des bonus, frags d'effet ET temps d'effet, comme il sort des cadences de
// Sessions. Sans cette règle, ses frags d'effet s'ajoutaient à un temps d'effet nul : rendement
// gonflé, et la barre épaisse de « Frags obtenus » ne portait plus la population de la fine.
func TestBuild_SansEchelleDeTemps_HorsDuRendementDesBonus(t *testing.T) {
	in := entreeUnMatch()
	m2 := Match{MatchID: "m2", StartTime: t0.Add(15 * time.Minute), SessionLabel: "s", Family: "Assassin"}
	in.Current = append(in.Current, m2)
	in.Film.Films["m2"] = sessionusage.FilmRow{MatchID: "m2"} // DurationMS = 0
	in.Film.Participants = append(in.Film.Participants, participants("m2")...)
	in.Film.Players = append(in.Film.Players, sessionusage.PlayerRow{
		MatchID: "m2", XUID: "P", CamoEpisodes: 1, CamoKills: 5, TakenByFamily: map[string]int{camo: 1},
	})
	if n := WithoutTimeScale(&in); n != 1 {
		t.Errorf("matchs sans échelle de temps = %d, attendu 1 (m2)", n)
	}
	b := Build(in)
	bonus := b.Production[0]
	if bonus.Kills != (domain.SquadEmpriseCount{Us: 3}) || bonus.Exposure.Kills != bonus.Kills ||
		bonus.Exposure.Value != (domain.SquadEmpriseCount{Us: 60_000}) {
		t.Errorf("bonus = %+v / %+v, attendu les 3 frags et 60 s de m1 seul (m2 sans échelle de temps)",
			bonus.Kills, bonus.Exposure)
	}
	if bonus.YieldUs == nil || math.Abs(*bonus.YieldUs-3) > 1e-9 {
		t.Errorf("rendement = %v, attendu 3 frags par minute d'effet (celui de m1)", bonus.YieldUs)
	}
	// Ses prises restent comptées : seul le rendement l'écarte.
	if r := ressource(b, domain.EmpriseResourcePowerup); r == nil || r.Taken.Us != 4 {
		t.Errorf("bonus pris = %+v, attendu 4 chez nous (3 de m1 + 1 de m2)", r)
	}
}

// TestBuild_ParticipantSansCamp_EstAdversaire — constat R11 de la revue L6.1 : dans un match à
// camp connu, un participant dont le camp est inconnu compte pour l'ADVERSAIRE (même règle que
// computeOutcomes), jamais pour nous — côté film (prises) comme côté feuille de match.
func TestBuild_ParticipantSansCamp_EstAdversaire(t *testing.T) {
	in := entreeUnMatch()
	// « X » a pris deux camouflages mais n'est pas dans les participants (camp inconnu).
	in.Film.Players = append(in.Film.Players, sessionusage.PlayerRow{
		MatchID: "m1", XUID: "X", TakenByFamily: map[string]int{camo: 2},
	})
	// « Y » est dans la feuille sans camp : ses quatre frags aux armes spéciales sont les leurs.
	in.PowerKills = append(in.PowerKills, PowerKillRow{MatchID: "m1", XUID: "Y", Kills: entier(4)})
	b := Build(in)
	if r := ressource(b, domain.EmpriseResourcePowerup); r == nil || r.Taken != (domain.SquadEmpriseCount{Us: 3, Them: 3}) {
		t.Errorf("bonus pris = %+v, attendu 3 / 3 (les deux camouflages de X chez l'adversaire)", r)
	}
	if pwk := b.Matches[0].PowerWeaponKills; pwk == nil || *pwk != (domain.SquadEmpriseCount{Us: 3, Them: 6}) {
		t.Errorf("frags aux armes spéciales = %+v, attendu 3 / 6 (les quatre de Y chez l'adversaire)", pwk)
	}
}
