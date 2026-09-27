//go:build integration

package duckdb_test

// squad_emprise_temoins_integration_test.go — LES CHIFFRES TÉMOINS DE L'ONGLET EMPRISE (lot L4.6
// du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26), DE BOUT EN BOUT : persisters réels
// (résumé d'usage, niveaux de socle), feuille de match, vues `_latest`, lecteurs réels, puis le
// calcul pur (analysis/squademprise).
//
// La fixture reproduit la soirée du 22/09 telle que la maquette de l'onglet la relève en base
// (JGtm, Chocoboflor, Madina97294, sept matchs dont Detachment sans film) : prises objet par objet
// et joueur par joueur (données ITEMS / ROLES / MATCHES de la maquette), temps et frags d'effet,
// frags aux armes spéciales de la feuille. Aucune base réelle n'est lue.

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/squademprise"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/persist"
	ddb "levelup/go-api/internal/platform/duckdb"
)

const (
	tJ, tC, tM, tR = "J", "C", "M", "R" // notre camp : JGtm, Chocoboflor, Madina97294, reste du camp
	camoT, osT     = "powerup_camo", "powerup_overshield"
)

// Familles d'arme (huit hexadécimaux) de la fixture.
const (
	spnkr, s7, epee, needler, sentinelle, electrique, empaleur = "0a000001", "0a000002", "0a000003", "0a000004", "0a000005", "0a000006", "0a000007"
	dechiq, br75, vk78, impulsion, vestige, mk50, ravageur     = "0b000001", "0b000002", "0b000003", "0b000004", "0b000005", "0b000006", "0b000007"
	calcineur, bulldog, bandit, disrupteur, traqueur, ma5k     = "0b000008", "0b000009", "0b00000a", "0b00000b", "0b00000c", "0b00000d"
)

// prises : (joueur, arme, compte).
type prise struct {
	xuid, arme string
	n          int
}

// matchTemoin — un match de la soirée.
type matchTemoin struct {
	id         string
	debut      time.Time
	film       bool
	bonus      []replay.UsagePlayerSummary
	socles     map[string]int // socles de bonus vidés
	puissance  []prise
	ratelier   []prise
	fragsArmes map[string]int // power_weapon_kills par joueur
}

func surb(xuid string, pris, ep int, ms int64, frags, garde, lache int) replay.UsagePlayerSummary {
	return bonusDe(xuid, osT, pris, ep, ms, frags, garde, lache)
}

func camouf(xuid string, pris, ep int, ms int64, frags, garde, lache int) replay.UsagePlayerSummary {
	return bonusDe(xuid, camoT, pris, ep, ms, frags, garde, lache)
}

func bonusDe(xuid, fam string, pris, ep int, ms int64, frags, garde, lache int) replay.UsagePlayerSummary {
	s := replay.UsagePlayerSummary{
		XUID: xuid, TakenByFamily: map[string]int{fam: pris},
		KeptByFamily: map[string]int{fam: garde}, DroppedByFamily: map[string]int{fam: lache},
		DroppedObjects: lache,
	}
	if fam == camoT {
		s.CamoEpisodes, s.CamoMS, s.CamoKills = ep, ms, frags
	} else {
		s.OvershieldEpisodes, s.OvershieldMS, s.OvershieldKills = ep, ms, frags
	}
	return s
}

// soiree2209 — la soirée du 22/09 (heures UTC ; 21:23 à Paris = 19:23 UTC).
func soiree2209() []matchTemoin {
	h := func(hh, mm int) time.Time { return time.Date(2026, 9, 22, hh, mm, 0, 0, time.UTC) }
	return []matchTemoin{
		{id: "starboard", debut: h(19, 23), film: true,
			bonus: []replay.UsagePlayerSummary{
				surb(tJ, 2, 2, 20_000, 2, 0, 0), surb(tC, 2, 2, 18_000, 1, 0, 0), surb(tM, 1, 1, 9_000, 0, 0, 0),
				surb("O1", 2, 1, 20_000, 1, 1, 0)},
			socles:     map[string]int{osT: 10},
			puissance:  []prise{{"O2", epee, 2}, {tR, electrique, 2}, {"O2", electrique, 2}},
			ratelier:   []prise{{tJ, vk78, 1}, {tR, vk78, 1}, {"O1", bulldog, 1}},
			fragsArmes: map[string]int{"O1": 4}},
		{id: "curfew", debut: h(19, 36), film: true,
			bonus:  []replay.UsagePlayerSummary{camouf(tM, 3, 3, 75_000, 3, 0, 0), camouf(tJ, 1, 1, 12_000, 1, 0, 0)},
			socles: map[string]int{camoT: 7},
			puissance: []prise{{tC, spnkr, 1}, {tJ, spnkr, 1}, {tR, spnkr, 1}, {tR, sentinelle, 1},
				{"O1", sentinelle, 2}, {tR, empaleur, 2}},
			ratelier: []prise{{tM, dechiq, 2}, {"O1", dechiq, 1}, {tR, vk78, 3}, {tR, impulsion, 1},
				{"O2", impulsion, 3}, {tJ, vestige, 1}, {"O3", vestige, 1}},
			fragsArmes: map[string]int{tJ: 5, tC: 4, tM: 2, tR: 1, "O1": 4}},
		{id: "origin", debut: h(19, 46), film: true,
			puissance:  []prise{{tJ, spnkr, 2}, {"O1", spnkr, 6}, {tJ, s7, 2}, {tM, s7, 1}, {"O2", s7, 1}},
			ratelier:   []prise{{"O1", br75, 7}, {tJ, mk50, 2}},
			fragsArmes: map[string]int{tJ: 8, "O1": 11}},
		{id: "solution", debut: h(19, 59), film: true,
			puissance: []prise{{tM, s7, 2}},
			ratelier: []prise{{tC, dechiq, 2}, {tM, dechiq, 2}, {"O1", dechiq, 1}, {"O2", ravageur, 1},
				{tR, bandit, 1}, {"O3", disrupteur, 1}},
			fragsArmes: map[string]int{tM: 5}},
		{id: "detachment", debut: h(20, 8), film: false, fragsArmes: map[string]int{tJ: 9, "O1": 13}},
		{id: "shogun", debut: h(20, 19), film: true,
			bonus: []replay.UsagePlayerSummary{camouf(tC, 1, 0, 0, 0, 1, 0), camouf(tM, 1, 0, 0, 0, 0, 1),
				camouf("O2", 3, 3, 60_000, 3, 0, 0)},
			socles: map[string]int{camoT: 8},
			puissance: []prise{{tR, spnkr, 1}, {"O1", spnkr, 3}, {tR, epee, 1}, {"O2", epee, 2},
				{tC, sentinelle, 1}},
			ratelier: []prise{{tC, dechiq, 1}, {"O1", vk78, 1}, {tR, impulsion, 2}, {"O2", ravageur, 1},
				{tR, calcineur, 1}, {"O3", calcineur, 1}, {"O4", traqueur, 1}, {"O4", ma5k, 1}},
			fragsArmes: map[string]int{tC: 4, "O2": 10}},
		{id: "catalyst", debut: h(20, 29), film: true,
			bonus:  []replay.UsagePlayerSummary{surb(tR, 1, 1, 25_000, 1, 0, 0), surb("O3", 3, 2, 33_000, 1, 0, 1)},
			socles: map[string]int{osT: 8},
			puissance: []prise{{tJ, spnkr, 2}, {tC, spnkr, 1}, {"O1", spnkr, 4}, {"O2", s7, 2}, {tJ, epee, 1},
				{"O3", epee, 1}, {tJ, needler, 1}, {"O4", needler, 4}},
			ratelier:   []prise{{"O1", vestige, 1}, {tM, mk50, 1}},
			fragsArmes: map[string]int{tJ: 9, "O1": 12}},
	}
}

// ecrireSoiree persiste la soirée par les VRAIS persisters et la feuille de match.
func ecrireSoiree(t *testing.T, db *ddb.DB, soiree []matchTemoin) {
	t.Helper()
	ctx := context.Background()
	usage := persist.NewUsageSummaryPersister(db.SQLDb())
	niveaux := persist.NewPadTiersPersister(db.SQLDb())
	equipes := map[string]int{tJ: 0, tC: 0, tM: 0, tR: 0, "O1": 1, "O2": 1, "O3": 1, "O4": 1}
	for _, m := range soiree {
		for xuid, team := range equipes {
			if _, err := db.SQLDb().Exec(`
				INSERT INTO match_participants (match_id, xuid, gamertag, team_id, present_at_completion, power_weapon_kills)
				VALUES (?, ?, ?, ?, TRUE, ?)`, m.id, xuid, xuid, team, m.fragsArmes[xuid]); err != nil {
				t.Fatalf("participant %s/%s: %v", m.id, xuid, err)
			}
		}
		if !m.film {
			continue
		}
		resume := &replay.UsageSummary{
			Match:   replay.UsageMatchSummary{DurationMS: 600_000, PowerupPadPickups: m.socles},
			Players: m.bonus,
		}
		if err := usage.PersistPass(ctx, m.id, resume); err != nil {
			t.Fatalf("résumé d'usage %s: %v", m.id, err)
		}
		batch := persist.PadTiersBatch{MatchID: m.id, PadsTotal: 14, PadsConfirmed: 14}
		for _, p := range m.puissance {
			batch.Rows = append(batch.Rows, persist.PadTierRow{XUID: p.xuid, Tier: persist.PadTierPower, WeaponFamily: p.arme, Pickups: p.n})
		}
		for _, p := range m.ratelier {
			batch.Rows = append(batch.Rows, persist.PadTierRow{XUID: p.xuid, Tier: persist.PadTierGround, WeaponFamily: p.arme, Pickups: p.n})
		}
		if err := niveaux.PersistPass(ctx, batch); err != nil {
			t.Fatalf("niveaux %s: %v", m.id, err)
		}
	}
}

// lireEmprise lit la soirée par les lecteurs réels et rend le bloc calculé.
func lireEmprise(t *testing.T, pdb *ddb.PlayerDB, soiree []matchTemoin) domain.SquadEmpriseBlock {
	t.Helper()
	ctx := context.Background()
	ids := make([]string, 0, len(soiree))
	current := make([]squademprise.Match, 0, len(soiree))
	for _, m := range soiree {
		ids = append(ids, m.id)
		current = append(current, squademprise.Match{MatchID: m.id, StartTime: m.debut, SessionLabel: "22/09"})
	}
	usage := ddb.NewSessionUsageRepo(pdb)
	films, err := usage.LoadUsageFilms(ctx, ids)
	must(t, err)
	players, err := usage.LoadUsagePlayers(ctx, ids)
	must(t, err)
	participants, err := usage.LoadParticipants(ctx, ids)
	must(t, err)
	tiers, err := usage.LoadPadTiers(ctx, ids)
	must(t, err)
	pwk, err := ddb.NewSquadEmpriseRepo(pdb).LoadPowerWeaponKills(ctx, ids)
	must(t, err)
	return squademprise.Build(squademprise.Input{
		PlayerXUID: tJ,
		Players:    []domain.SessionUsageSquadPlayer{{XUID: tJ, Gamertag: "JGtm"}, {XUID: tC, Gamertag: "Chocoboflor"}, {XUID: tM, Gamertag: "Madina97294"}},
		Current:    current,
		Film:       &squademprise.FilmData{Films: films, Players: players, Participants: participants, PadTiers: tiers},
		PowerKills: pwk,
	})
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("lecture: %v", err)
	}
}

func TestSquadEmprise_TemoinsDu2209(t *testing.T) {
	sharedDB, pdb := setupUsageSharedDB(t)
	soiree := soiree2209()
	ecrireSoiree(t, sharedDB, soiree)
	b := lireEmprise(t, pdb, soiree)

	if b.MatchesTotal != 7 || b.MatchesMeasured != 6 {
		t.Fatalf("couverture %d/%d, attendu 6 matchs filmés sur 7 (Detachment sans film)", b.MatchesMeasured, b.MatchesTotal)
	}
	// Bilan : bonus 12 / 8 ; armes spéciales 23 / 29 ; bonus perdus 2 sur 12, 2 sur 8.
	bonus, armes := b.Resources[0], b.Resources[1]
	if bonus.Taken != (domain.SquadEmpriseCount{Us: 12, Them: 8}) {
		t.Errorf("bonus = %+v, attendu 12 / 8", bonus.Taken)
	}
	if o := bonus.Outcomes; o == nil || o.Us.Kept+o.Us.Dropped != 2 || o.Us.Taken != 12 ||
		o.Them.Kept+o.Them.Dropped != 2 || o.Them.Taken != 8 {
		t.Errorf("bonus perdus = %+v, attendu nous 2 sur 12, eux 2 sur 8", bonus.Outcomes)
	}
	if armes.Resource != domain.EmpriseResourcePowerWeapon || armes.Taken != (domain.SquadEmpriseCount{Us: 23, Them: 29}) {
		t.Errorf("armes spéciales = %+v, attendu 23 / 29", armes)
	}
	verifierProduction(t, b.Production)
	verifierFiches(t, b)
	verifierMatchs(t, b.Matches)
}

// verifierProduction : frags d'effet 8 / 5, temps d'effet 2 min 39 / 1 min 53, frags aux armes
// spéciales 47 / 54 (tous les matchs), exposition 23 / 29 prises.
func verifierProduction(t *testing.T, prod []domain.SquadEmpriseProduction) {
	t.Helper()
	if len(prod) != 2 {
		t.Fatalf("production = %+v", prod)
	}
	bonus, armes := prod[0], prod[1]
	if bonus.Kills != (domain.SquadEmpriseCount{Us: 8, Them: 5}) {
		t.Errorf("frags pendant l'effet = %+v, attendu 8 / 5", bonus.Kills)
	}
	if bonus.Exposure.Value != (domain.SquadEmpriseCount{Us: 159_000, Them: 113_000}) {
		t.Errorf("temps d'effet = %+v ms, attendu 159 s / 113 s", bonus.Exposure.Value)
	}
	if armes.Kills != (domain.SquadEmpriseCount{Us: 47, Them: 54}) {
		t.Errorf("frags aux armes spéciales = %+v, attendu 47 / 54", armes.Kills)
	}
	if armes.Exposure.Value != (domain.SquadEmpriseCount{Us: 23, Them: 29}) {
		t.Errorf("prises = %+v, attendu 23 / 29", armes.Exposure.Value)
	}
	// Le rendement se lit sur les matchs dont les niveaux sont mesurés : Detachment (9 / 13,
	// sans film) en sort — 38 / 41 frags pour 23 / 29 prises.
	if armes.Exposure.Kills != (domain.SquadEmpriseCount{Us: 38, Them: 41}) {
		t.Errorf("frags sur le périmètre des prises = %+v, attendu 38 / 41", armes.Exposure.Kills)
	}
	proche := func(v *float64, want float64) bool { return v != nil && math.Abs(*v-want) < 1e-9 }
	if !proche(bonus.RelativeGap, (8/(159.0/60))/(5/(113.0/60))-1) {
		t.Errorf("écart de rendement des bonus = %v", bonus.RelativeGap)
	}
	if !proche(armes.RelativeGap, (38.0/23)/(41.0/29)-1) {
		t.Errorf("écart de rendement des armes spéciales = %v", armes.RelativeGap)
	}
}

// verifierFiches : ROLES de la maquette — bonus 3 / 3 / 5 / 1, armes spéciales 9 / 3 / 3 / 8 ;
// camouflage perdu par Chocoboflor (gardé) et Madina97294 (lâché) ; 33 socles de bonus vidés.
func verifierFiches(t *testing.T, b domain.SquadEmpriseBlock) {
	t.Helper()
	parJoueur := map[string][4]int{}
	socles, perdus := 0, map[string]int{}
	for _, o := range b.Objects {
		v := parJoueur[o.Resource]
		for i, part := range o.Squad {
			v[i] += part.Taken
			if part.Kept != nil {
				perdus[fmt.Sprintf("%s/%d/garde", o.Key, i)] += *part.Kept
				perdus[fmt.Sprintf("%s/%d/lache", o.Key, i)] += *part.Dropped
			}
		}
		parJoueur[o.Resource] = v
		if o.PadsEmptied != nil {
			socles += *o.PadsEmptied
		}
	}
	if got := parJoueur[domain.EmpriseResourcePowerup]; got != [4]int{3, 3, 5, 1} {
		t.Errorf("bonus par joueur = %v, attendu JGtm 3, Chocoboflor 3, Madina97294 5, reste 1", got)
	}
	if got := parJoueur[domain.EmpriseResourcePowerWeapon]; got != [4]int{9, 3, 3, 8} {
		t.Errorf("armes spéciales par joueur = %v, attendu 9 / 3 / 3 / 8", got)
	}
	if perdus[camoT+"/1/garde"] != 1 || perdus[camoT+"/2/lache"] != 1 {
		t.Errorf("camouflages perdus = %v, attendu Chocoboflor 1 gardé, Madina97294 1 lâché", perdus)
	}
	if socles != 33 {
		t.Errorf("socles de bonus vidés = %d, attendu 33", socles)
	}
}

// verifierMatchs : FIL_SERIES de la maquette, match par match.
func verifierMatchs(t *testing.T, matches []domain.SquadEmpriseMatch) {
	t.Helper()
	attendu := []struct{ bonus, armes, frags *domain.SquadEmpriseCount }{
		{&domain.SquadEmpriseCount{Us: 5, Them: 2}, &domain.SquadEmpriseCount{Us: 2, Them: 4}, &domain.SquadEmpriseCount{Us: 0, Them: 4}},
		{&domain.SquadEmpriseCount{Us: 4, Them: 0}, &domain.SquadEmpriseCount{Us: 6, Them: 2}, &domain.SquadEmpriseCount{Us: 12, Them: 4}},
		{nil, &domain.SquadEmpriseCount{Us: 5, Them: 7}, &domain.SquadEmpriseCount{Us: 8, Them: 11}},
		{nil, &domain.SquadEmpriseCount{Us: 2, Them: 0}, &domain.SquadEmpriseCount{Us: 5, Them: 0}},
		{nil, nil, &domain.SquadEmpriseCount{Us: 9, Them: 13}},
		{&domain.SquadEmpriseCount{Us: 2, Them: 3}, &domain.SquadEmpriseCount{Us: 3, Them: 5}, &domain.SquadEmpriseCount{Us: 4, Them: 10}},
		{&domain.SquadEmpriseCount{Us: 1, Them: 3}, &domain.SquadEmpriseCount{Us: 5, Them: 11}, &domain.SquadEmpriseCount{Us: 9, Them: 12}},
	}
	for i, a := range attendu {
		m := matches[i]
		for res, want := range map[string]*domain.SquadEmpriseCount{
			domain.EmpriseResourcePowerup: a.bonus, domain.EmpriseResourcePowerWeapon: a.armes,
		} {
			got := compteDe(m, res)
			if (want == nil) != (got == nil) || (want != nil && *got != *want) {
				t.Errorf("%s / %s = %v, attendu %v", m.MatchID, res, got, want)
			}
		}
		if m.PowerWeaponKills == nil || *m.PowerWeaponKills != *a.frags {
			t.Errorf("%s : frags aux armes spéciales = %v, attendu %v", m.MatchID, m.PowerWeaponKills, *a.frags)
		}
	}
	if matches[4].HasFilm || len(matches[4].Resources) != 0 {
		t.Errorf("Detachment = %+v, attendu « sans film »", matches[4])
	}
}

func compteDe(m domain.SquadEmpriseMatch, res string) *domain.SquadEmpriseCount {
	for _, r := range m.Resources {
		if r.Resource == res {
			c := r.Taken
			return &c
		}
	}
	return nil
}

// TestSquadEmpriseRepo_FeuilleMuetteResteNil — une feuille qui ne dit rien (NULL) n'est pas un
// zéro : le lecteur rend nil, et le calcul ne publie pas de frags pour ce match.
func TestSquadEmpriseRepo_FeuilleMuetteResteNil(t *testing.T) {
	sharedDB, pdb := setupUsageSharedDB(t)
	if _, err := sharedDB.SQLDb().Exec(`
		INSERT INTO match_participants (match_id, xuid, gamertag, team_id, present_at_completion, power_weapon_kills)
		VALUES ('muet', 'J', 'J', 0, TRUE, NULL), ('muet', 'O1', 'O1', 1, TRUE, NULL)`); err != nil {
		t.Fatalf("participants: %v", err)
	}
	rows, err := ddb.NewSquadEmpriseRepo(pdb).LoadPowerWeaponKills(context.Background(), []string{"muet"})
	must(t, err)
	if len(rows) != 2 || rows[0].Kills != nil || rows[0].TeamID == nil {
		t.Fatalf("lignes = %+v, attendu deux lignes à camp connu, sans frags", rows)
	}
	b := squademprise.Build(squademprise.Input{
		PlayerXUID: "J", Current: []squademprise.Match{{MatchID: "muet"}}, PowerKills: rows,
	})
	if b.Matches[0].PowerWeaponKills != nil || len(b.Production) != 0 {
		t.Errorf("frags publiés sur une feuille muette : %+v", b)
	}
}
