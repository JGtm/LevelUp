package squademprise

import (
	"fmt"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// matchSurCarte — un match du périmètre solo (joueur P, seul joueur des fiches) sur une carte.
func matchSurCarte(id, carte string, i int, issue canonical.Outcome) Match {
	return Match{MatchID: id, StartTime: t0.Add(time.Duration(i) * time.Minute), MapKey: "k-" + carte, MapLabel: carte, Outcome: issue}
}

// filme ajoute au film un match lu (camp connu), avec un camouflage pris par moi et un par E1.
func filme(f *FilmData, id string, niveaux bool) {
	f.Films[id] = sessionusage.FilmRow{MatchID: id, DurationMS: 600_000}
	f.Participants = append(f.Participants, participants(id)...)
	f.Players = append(f.Players,
		sessionusage.PlayerRow{MatchID: id, XUID: "P", TakenByFamily: map[string]int{camo: 1}},
		sessionusage.PlayerRow{MatchID: id, XUID: "R", TakenByFamily: map[string]int{camo: 1}},
		sessionusage.PlayerRow{MatchID: id, XUID: "E1", TakenByFamily: map[string]int{camo: 1}},
	)
	if niveaux {
		f.PadTiers = append(f.PadTiers, tier(id, "P", domain.PadTierPower, lr, 1), tier(id, "E2", domain.PadTierPower, lr, 2))
	}
}

func entreeSolo() Input {
	return Input{
		PlayerXUID: "P",
		Players:    []domain.SessionUsageSquadPlayer{{XUID: "P", Gamertag: "Moi"}},
		Film:       &FilmData{Films: map[string]sessionusage.FilmRow{}},
	}
}

// TestBuildMaps_TriEtSommesParCarte — la carte la plus jouée d'abord ; une colonne somme les
// prises de ses matchs lisibles, compte ses matchs filmés, mesurés, à niveaux, et ses résultats.
func TestBuildMaps_TriEtSommesParCarte(t *testing.T) {
	in := entreeSolo()
	in.Current = []Match{
		matchSurCarte("b1", "Bazaar", 1, canonical.OutcomeWin),
		matchSurCarte("a1", "Aquarius", 2, canonical.OutcomeLoss),
		matchSurCarte("b2", "Bazaar", 3, canonical.OutcomeLoss),
		matchSurCarte("b3", "Bazaar", 4, canonical.OutcomeDNF),
	}
	filme(in.Film, "b1", true)
	filme(in.Film, "b2", false) // filmé, niveaux non mesurés
	filme(in.Film, "a1", true)
	in.PowerKills = []PowerKillRow{
		{MatchID: "b3", XUID: "P", TeamID: equipe(0), Kills: entier(4)},
		{MatchID: "b3", XUID: "E1", TeamID: equipe(1), Kills: entier(1)},
	}
	cols := BuildMaps(in)
	if len(cols) != 2 || cols[0].MapLabel != "Bazaar" || cols[1].MapLabel != "Aquarius" {
		t.Fatalf("colonnes = %+v, attendu Bazaar (3 matchs) puis Aquarius (1)", cols)
	}
	b := cols[0]
	if b.Matches != 3 || b.MatchesFilmed != 2 || b.MatchesMeasured != 2 || b.MatchesTiers != 1 {
		t.Errorf("Bazaar : %d matchs, %d filmés, %d mesurés, %d à niveaux ; attendu 3, 2, 2, 1",
			b.Matches, b.MatchesFilmed, b.MatchesMeasured, b.MatchesTiers)
	}
	if b.Wins != 1 || b.Losses != 1 || b.Others != 1 || b.MapKey != "k-Bazaar" || b.OtherMaps != 0 {
		t.Errorf("Bazaar : %+v, attendu 1 V, 1 D, 1 autre, clé k-Bazaar", b)
	}
	bonus := ressourceDeColonne(b, domain.EmpriseResourcePowerup)
	if bonus == nil || bonus.Taken != (domain.SquadEmpriseCount{Us: 4, Them: 2}) {
		t.Fatalf("bonus de Bazaar = %+v, attendu 4 / 2 (deux matchs lus, moi + reste contre E1)", bonus)
	}
	if moi := bonus.Objects[0].Squad[0]; moi.XUID != "P" || moi.Taken != 2 {
		t.Errorf("ma part du camouflage = %+v, attendu 2", moi)
	}
	if reste := bonus.Objects[0].Squad[1]; reste.XUID != "" || reste.Taken != 2 {
		t.Errorf("reste du camp = %+v, attendu 2", reste)
	}
	armes := ressourceDeColonne(b, domain.EmpriseResourcePowerWeapon)
	if armes == nil || armes.Taken != (domain.SquadEmpriseCount{Us: 1, Them: 2}) {
		t.Errorf("armes spéciales de Bazaar = %+v, attendu 1 / 2 (le seul match à niveaux)", armes)
	}
	if b.PowerWeaponKills == nil || *b.PowerWeaponKills != (domain.SquadEmpriseCount{Us: 4, Them: 1}) {
		t.Errorf("frags aux armes spéciales = %+v, attendu 4 / 1 (feuille, match sans film compris)", b.PowerWeaponKills)
	}
}

// TestBuildMaps_RepliAuDelaDeTreizeCartes — jusqu'à treize cartes, toutes nommées ; à partir de
// quatorze, les douze plus jouées puis « Autres cartes », qui somme le reste.
func TestBuildMaps_RepliAuDelaDeTreizeCartes(t *testing.T) {
	for _, cas := range []struct {
		cartes, colonnes, repliees int
	}{{13, 13, 0}, {14, 13, 2}, {20, 13, 8}} {
		in := entreeSolo()
		for c := 0; c < cas.cartes; c++ {
			// La carte c a (cartes - c) matchs : l'ordre de jeu est l'ordre attendu.
			for k := 0; k < cas.cartes-c; k++ {
				id := fmt.Sprintf("c%02d-%02d", c, k)
				in.Current = append(in.Current, matchSurCarte(id, fmt.Sprintf("Carte %02d", c), c*30+k, canonical.OutcomeWin))
				filme(in.Film, id, false)
			}
		}
		cols := BuildMaps(in)
		if len(cols) != cas.colonnes {
			t.Fatalf("%d cartes : %d colonnes, attendu %d", cas.cartes, len(cols), cas.colonnes)
		}
		last := cols[len(cols)-1]
		if last.OtherMaps != cas.repliees {
			t.Errorf("%d cartes : dernière colonne %+v, attendu %d cartes repliées", cas.cartes, last, cas.repliees)
		}
		if cas.repliees == 0 {
			continue
		}
		attendu := 0
		for c := domain.EmpriseGridMaxMaps; c < cas.cartes; c++ {
			attendu += cas.cartes - c
		}
		bonus := ressourceDeColonne(last, domain.EmpriseResourcePowerup)
		if last.MapLabel != "" || last.Matches != attendu || last.Wins != attendu || bonus == nil || bonus.Taken.Us != 2*attendu {
			t.Errorf("%d cartes : repli = %+v, attendu %d matchs sommés et %d bonus pour nous", cas.cartes, last, attendu, 2*attendu)
		}
		if cols[0].MapLabel != "Carte 00" || cols[domain.EmpriseGridMaxMaps-1].MapLabel != "Carte 11" {
			t.Errorf("%d cartes : ordre %q … %q", cas.cartes, cols[0].MapLabel, cols[domain.EmpriseGridMaxMaps-1].MapLabel)
		}
	}
}

// TestBuildMaps_CarteSansFilm — une carte jamais filmée n'a aucune prise lue, mais ses matchs et
// ses résultats sont comptés.
func TestBuildMaps_CarteSansFilm(t *testing.T) {
	in := entreeSolo()
	in.Current = []Match{matchSurCarte("x1", "Forest", 1, canonical.OutcomeLoss)}
	cols := BuildMaps(in)
	if len(cols) != 1 || cols[0].MatchesFilmed != 0 || len(cols[0].Resources) != 0 || cols[0].Losses != 1 {
		t.Errorf("Forest = %+v, attendu 1 match sans film, aucune ressource, 1 défaite", cols)
	}
}

func ressourceDeColonne(c domain.EmpriseMapColumn, res string) *domain.SquadEmpriseMatchResource {
	for i := range c.Resources {
		if c.Resources[i].Resource == res {
			return &c.Resources[i]
		}
	}
	return nil
}
