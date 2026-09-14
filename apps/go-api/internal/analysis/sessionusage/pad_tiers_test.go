package sessionusage

// Tests — L'AGREGAT DES NIVEAUX D'ARMES.
//
// CE QU'ILS VERROUILLENT :
//   - aucune ligne ⇒ bloc NIL, jamais un bloc a zero (« pas encore mesure » ≠ « aucune prise ») ;
//   - les QUATRE denominateurs disent quatre choses differentes, et un match sans socle n'est
//     pas un match non mesure ;
//   - le ZERO MESURE (`aucune_prise`) compte le match et ne cree AUCUN niveau ;
//   - les parts se calculent sur le perimetre a camp connu, numerateur ET denominateur ;
//   - l'ordre des niveaux est celui, ECRIT, de `domain.PadTierOrder` — jamais le volume.

import (
	"testing"

	"levelup/go-api/internal/domain"
)

const (
	hydra  = "b619d84a"
	sniper = "9d6aaed2"
)

// lignes du temoin : deux matchs, deux camps, le joueur suivi est « moi ».
func temoinPadTiers() PadTiersInput {
	r := func(m, x, tier, arme string, n, conf, tot int) PadTierRow {
		return PadTierRow{
			MatchID: m, XUID: x, Tier: tier, WeaponFamily: arme, Pickups: n,
			PadsConfirmed: conf, PadsTotal: tot,
		}
	}
	return PadTiersInput{
		PlayerXUID: "moi",
		Rows: []PadTierRow{
			r("m1", "moi", "puissance", sniper, 3, 5, 6),
			r("m1", "moi", "terrain", hydra, 2, 5, 6),
			r("m1", "allie", "puissance", sniper, 1, 5, 6),
			r("m1", "ennemi", "puissance", sniper, 4, 5, 6),
			r("m2", "moi", "terrain", hydra, 1, 3, 3),
			{MatchID: "m2", XUID: "allie", Tier: "aucune_prise", PadsConfirmed: 3, PadsTotal: 3},
		},
		PlayerTeam: map[string]int{"m1": 0, "m2": 0},
		TeamOf: map[string]map[string]int{
			"m1": {"moi": 0, "allie": 0, "ennemi": 1},
			"m2": {"moi": 0, "allie": 0},
		},
	}
}

func niveau(b *domain.SessionUsagePadTiersBlock, tier string) *domain.SessionUsagePadTier {
	for i := range b.Tiers {
		if b.Tiers[i].Tier == tier {
			return &b.Tiers[i]
		}
	}
	return nil
}

func TestComputePadTiers_NilSansLigne(t *testing.T) {
	if got := ComputePadTiers(PadTiersInput{PlayerXUID: "moi"}); got != nil {
		t.Errorf("bloc servi sans aucune ligne : %+v — « pas encore mesure » n'est pas « aucune prise »", got)
	}
}

func TestComputePadTiers_LesQuatreDenominateurs(t *testing.T) {
	in := temoinPadTiers()
	// Un troisieme match MESURE mais dont le film n'a vu AUCUN socle (le cas Fiesta), et dont
	// le mode est a departs aleatoires.
	in.Rows = append(in.Rows, PadTierRow{
		MatchID: "m3", XUID: "moi", Tier: "aucune_prise",
		PadsConfirmed: 0, PadsTotal: 0, RandomStarts: true,
	})
	// Un quatrieme match avec des socles mais SANS carte a la reference.
	in.Rows = append(in.Rows, PadTierRow{
		MatchID: "m4", XUID: "moi", Tier: "non_classe", WeaponFamily: sniper, Pickups: 2,
		PadsConfirmed: 0, PadsTotal: 4,
	})
	b := ComputePadTiers(in)
	if b.MatchesMeasured != 4 {
		t.Errorf("matchs mesures = %d, attendu 4", b.MatchesMeasured)
	}
	if b.MatchesWithPads != 3 {
		t.Errorf("matchs avec socles = %d, attendu 3 (m3 n'en a aucun)", b.MatchesWithPads)
	}
	if b.MatchesTiersEstablished != 2 {
		t.Errorf("matchs a niveaux etablis = %d, attendu 2 (m4 est hors reference)", b.MatchesTiersEstablished)
	}
	if b.MatchesRandomStarts != 1 {
		t.Errorf("matchs a departs aleatoires = %d, attendu 1", b.MatchesRandomStarts)
	}
}

func TestComputePadTiers_LeZeroMesureNeCreeAucunNiveau(t *testing.T) {
	b := ComputePadTiers(temoinPadTiers())
	for _, tier := range b.Tiers {
		if tier.Tier == "aucune_prise" {
			t.Errorf("le zero mesure est devenu un niveau : %+v", tier)
		}
	}
	// Il compte le match, en revanche.
	if b.MatchesMeasured != 2 {
		t.Errorf("matchs mesures = %d, attendu 2", b.MatchesMeasured)
	}
}

func TestComputePadTiers_PartsSurLePerimetreACampConnu(t *testing.T) {
	b := ComputePadTiers(temoinPadTiers())
	puissance := niveau(b, "puissance")
	if puissance == nil {
		t.Fatalf("niveau « puissance » absent : %+v", b.Tiers)
	}
	if puissance.PlayerTotal != 3 || puissance.LobbyTotal != 8 {
		t.Errorf("puissance : joueur=%v lobby=%v, attendu 3 et 8", puissance.PlayerTotal, puissance.LobbyTotal)
	}
	// Le camp du joueur (moi + allie) = 4 ; sa part = 3/4.
	if puissance.TeamTotal == nil || *puissance.TeamTotal != 4 {
		t.Fatalf("total du camp = %v, attendu 4", puissance.TeamTotal)
	}
	if puissance.PlayerShareOfTeamPct == nil || *puissance.PlayerShareOfTeamPct != 75 {
		t.Errorf("part du camp = %v, attendu 75", puissance.PlayerShareOfTeamPct)
	}
	if puissance.PlayerShareOfLobbyPct == nil || *puissance.PlayerShareOfLobbyPct != 37.5 {
		t.Errorf("part du lobby = %v, attendu 37,5", puissance.PlayerShareOfLobbyPct)
	}
	// Cadence : 3 prises sur 2 matchs mesures.
	if puissance.PlayerPerMatch == nil || *puissance.PlayerPerMatch != 1.5 {
		t.Errorf("cadence = %v, attendu 1,5", puissance.PlayerPerMatch)
	}
}

func TestComputePadTiers_OrdreEcritEtDetailParArme(t *testing.T) {
	b := ComputePadTiers(temoinPadTiers())
	// L'ORDRE EST CELUI DE `PadTierOrder` : terrain AVANT puissance, meme si la puissance
	// pese plus lourd.
	if len(b.Tiers) != 2 || b.Tiers[0].Tier != "terrain" || b.Tiers[1].Tier != "puissance" {
		t.Fatalf("ordre des niveaux inattendu : %+v", b.Tiers)
	}
	terrain := niveau(b, "terrain")
	if len(terrain.Weapons) != 1 || terrain.Weapons[0].FamilyKey != hydra {
		t.Fatalf("detail par arme du terrain : %+v", terrain.Weapons)
	}
	if terrain.Weapons[0].PlayerPickups != 3 || terrain.Weapons[0].LobbyPickups != 3 {
		t.Errorf("detail par arme = joueur %v / lobby %v, attendu 3 et 3",
			terrain.Weapons[0].PlayerPickups, terrain.Weapons[0].LobbyPickups)
	}
}

// TestComputePadTiers_PasDeZeroPourCent — un camp qui n'a rien pris ne donne PAS de part.
func TestComputePadTiers_PasDeZeroPourCent(t *testing.T) {
	b := ComputePadTiers(PadTiersInput{
		PlayerXUID: "moi",
		Rows: []PadTierRow{{
			MatchID: "m1", XUID: "ennemi", Tier: "puissance", WeaponFamily: sniper,
			Pickups: 2, PadsConfirmed: 2, PadsTotal: 2,
		}},
		PlayerTeam: map[string]int{"m1": 0},
		TeamOf:     map[string]map[string]int{"m1": {"ennemi": 1}},
	})
	p := niveau(b, "puissance")
	if p == nil {
		t.Fatal("niveau absent")
	}
	if p.TeamTotal != nil || p.PlayerShareOfTeamPct != nil {
		t.Errorf("part d'equipe servie sur un camp a zero : total=%v part=%v",
			p.TeamTotal, p.PlayerShareOfTeamPct)
	}
}

// TestComputePadTiers_UnXuidAbsentNestPasLEquipe0 — le piege de la valeur zero d'une map.
func TestComputePadTiers_UnXuidAbsentNestPasLEquipe0(t *testing.T) {
	b := ComputePadTiers(PadTiersInput{
		PlayerXUID: "moi",
		Rows: []PadTierRow{{
			MatchID: "m1", XUID: "inconnu", Tier: "puissance", WeaponFamily: sniper,
			Pickups: 5, PadsConfirmed: 1, PadsTotal: 1,
		}},
		PlayerTeam: map[string]int{"m1": 0},
		// `inconnu` n'est PAS dans la table : il ne doit pas passer pour un joueur du camp 0.
		TeamOf: map[string]map[string]int{"m1": {"moi": 0}},
	})
	if p := niveau(b, "puissance"); p.TeamTotal != nil {
		t.Errorf("un xuid absent de la table a ete compte dans le camp : %v", *p.TeamTotal)
	}
}
