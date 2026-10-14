package service

// tactical_service_equipe_poids_test.go — LE POIDS DE LA REPONSE D'UNE ZONE CHARGEE (lot L2.9) et la
// degradation d'un titre sans les sources de l'equipe (Halo 5 : positions natives, pas de film).

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
)

// zoneChargee : `n` matchs, une mort de MOI dans la cellule (4, 4) de chacun, avec `mates` coequipiers
// visibles (position, distances) et un tueur abattu par un coequipier dans la fenetre.
func zoneChargee(n, mates int) *mockTacticalRepo {
	repo := &mockTacticalRepo{ouvrables: map[string]time.Time{}}
	variantes := map[string]string{}
	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("zone%04d", i)
		variantes[id] = "Slayer:Arena"
		repo.ouvrables[id] = base.Add(time.Duration(i) * time.Hour)
		repo.pos.Points = append(repo.pos.Points, domain.TacticalKillPosition{
			MatchID: id, KillerXUID: tsAdv, VictimXUID: tsMoi, KillerGamertag: "Rival",
			KillerX: 9, KillerY: 9, VictimX: 2.1, VictimY: 2.1, TimeMs: 100_000,
		})
		repo.contextes = append(repo.contextes, domain.ContexteDeMort{
			MatchID: id, VictimXUID: tsMoi, TimeMs: 100_000, PlusProcheM: m(4), Visibles: mates, Total: mates,
		})
		for k := 0; k < mates; k++ {
			repo.mates = append(repo.mates, domain.MateDeMort{
				MatchID: id, VictimXUID: tsMoi, TimeMs: 100_000, MateXUID: fmt.Sprintf("25332740000%05d", 100+k),
				Gamertag: fmt.Sprintf("Coequipier%d", k), Etat: domain.TacticalMateVisible,
				X: m(4.5), Y: m(6.25), DistanceM: m(4 + float64(k)),
			})
		}
		repo.journal = append(repo.journal,
			domain.KillDuJournal{MatchID: id, TimeMs: 100_000, VictimXUID: tsMoi, VictimGamertag: "Moi", KillerXUID: tsAdv, KillerGamertag: "Rival"},
			domain.KillDuJournal{MatchID: id, TimeMs: 101_000, VictimXUID: tsAdv, VictimGamertag: "Rival",
				KillerXUID: fmt.Sprintf("25332740000%05d", 100), KillerGamertag: "Coequipier0", KillerX: m(5), KillerY: m(5)})
	}
	repo.pos.Univers = universVariantes(variantes)
	for _, id := range triees(variantes) {
		repo.pos.Univers.Equipes[id][fmt.Sprintf("25332740000%05d", 100)] = 0
		for k := 1; k < mates; k++ {
			repo.pos.Univers.Equipes[id][fmt.Sprintf("25332740000%05d", 100+k)] = 0
		}
	}
	return repo
}

// TestCellule_PoidsDeLaReponseDUneZoneChargee : 100 morts dans une zone, en Arene (3 coequipiers) puis
// en BTB (11) ; on mesure l'octet servi avec et sans l'equipe, la riposte et le bilan, et on tient un
// plafond (marge ~30 % sur la mesure) pour qu'un champ ajoute sans y penser fasse rougir ce test.
func TestCellule_PoidsDeLaReponseDUneZoneChargee(t *testing.T) {
	cas := []struct {
		nom     string
		mates   int
		plafond int
	}{
		{"arene (3 coequipiers)", 3, 130_000},
		{"btb (11 coequipiers)", 11, 300_000},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			repo := zoneChargee(100, c.mates)
			got := celluleLue(t, NewTacticalService(repo, capsCompletes(), tsMoi).
				WithRadarRange(map[string]int{"Slayer:Arena": 18}),
				celluleDemande(repo, domain.TacticalQuestionMorts, domain.TacticalQuiMoi, 4, 4))
			if len(got.Contributions) != 100 || got.Bilan == nil || got.Bilan.Morts.TueurAbattu != (domain.TacticalNSurM{N: 100, M: 100}) {
				t.Fatalf("contributions = %d, bilan = %+v", len(got.Contributions), got.Bilan)
			}
			avec, _ := json.Marshal(got)
			sans := got
			sans.Bilan = nil
			sans.Contributions = append([]domain.TacticalContribution(nil), got.Contributions...)
			for i := range sans.Contributions {
				sans.Contributions[i].Equipe, sans.Contributions[i].Riposte = nil, nil
			}
			base, _ := json.Marshal(sans)
			t.Logf("%s : %d octets avec equipe/riposte/bilan, %d sans (+%d, soit %d octets par mort)",
				c.nom, len(avec), len(base), len(avec)-len(base), (len(avec)-len(base))/100)
			if len(avec) > c.plafond {
				t.Errorf("reponse = %d octets, plafond %d : un champ de l'equipe a grossi la reponse", len(avec), c.plafond)
			}
		})
	}
}

// TestEquipe_TitreSansLesSources_ServiSansErreur : un titre dont le shared n'a pas les tables du film
// (Halo 5, positions natives) : ErrCapabilityNotSupported des deux sources, aucune equipe, aucune
// riposte, aucun bilan, aucune erreur.
func TestEquipe_TitreSansLesSources_ServiSansErreur(t *testing.T) {
	repo := corpusEquipe()
	repo.errMates, repo.errJournal = games.ErrCapabilityNotSupported, games.ErrCapabilityNotSupported
	_, matchs := corpusDetail()
	got := celluleLue(t, sourcesCompletes(matchs).service(repo, capsPositionsSeules()),
		celluleDemande(repo, domain.TacticalQuestionGagne, domain.TacticalQuiMoi, 4, 4))
	if len(got.Contributions) != 4 {
		t.Fatalf("contributions = %d, want 4", len(got.Contributions))
	}
	for _, c := range got.Contributions {
		if c.Equipe != nil || c.Riposte != nil {
			t.Errorf("contribution %+v : ni equipe ni riposte sans les sources", c)
		}
	}
	if got.Bilan == nil || got.Bilan.Frags != nil || got.Bilan.Morts == nil || got.Bilan.Morts.Examinees != 0 {
		t.Errorf("bilan = %+v : les morts se comptent, rien n'est examine", got.Bilan)
	}
}
