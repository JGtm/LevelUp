package squadagg

// emprise_lectures_test.go — la lecture du FILM de l'Emprise, partagée par l'Escouade et les Séries
// temporelles : l'habitude ajoute SES matchs aux lectures du périmètre, et son échec la dégrade
// SEULE ; les niveaux de socle portent sur le périmètre ET l'habitude.

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/analysis/squademprise"
	"levelup/go-api/internal/analysis/squadformes"
	"levelup/go-api/internal/domain"
)

// usageEspion — un résumé d'usage en mémoire qui note les matchs dont il lit les niveaux.
type usageEspion struct {
	echecSur string // un match dont la lecture des joueurs échoue (les films, eux, se lisent)
	tiers    [][]string
}

func (*usageEspion) LoadUsageFilms(_ context.Context, ids []string) (map[string]sessionusage.FilmRow, error) {
	out := map[string]sessionusage.FilmRow{}
	for _, id := range ids {
		out[id] = sessionusage.FilmRow{MatchID: id}
	}
	return out, nil
}

func (u *usageEspion) LoadUsagePlayers(_ context.Context, ids []string) ([]sessionusage.PlayerRow, error) {
	if slices.Contains(ids, u.echecSur) {
		return nil, errors.New("base indisponible")
	}
	return nil, nil
}

func (u *usageEspion) LoadPadTiers(_ context.Context, ids []string) ([]sessionusage.PadTierRow, error) {
	u.tiers = append(u.tiers, ids)
	return nil, nil
}

func (u *usageEspion) LoadParticipants(context.Context, []string) ([]sessionusage.ParticipantRow, error) {
	return nil, nil
}

// soireeDe — un match d'une soirée et d'une famille de mode donnée.
func soireeDe(id, soiree string, jour int) squademprise.Match {
	return squademprise.Match{
		MatchID: id, SessionLabel: soiree, Family: "slayer",
		StartTime: time.Date(2026, 9, jour, 20, 0, 0, 0, time.UTC),
	}
}

func TestEmpriseLecteurFilm_LHabitudeEstLueEnPlusDuPerimetre(t *testing.T) {
	current := []squademprise.Match{soireeDe("ce-soir", "s2", 22)}
	timeline := append([]squademprise.Match{soireeDe("hier", "s1", 21)}, current...)
	repo := &usageEspion{}
	film, raison := EmpriseLecteur{Page: "test"}.Film(context.Background(), repo, current, timeline, nil)
	if raison != "" || film == nil {
		t.Fatalf("film = %v, raison %q ; attendu une lecture réussie", film, raison)
	}
	if _, ok := film.Films["hier"]; !ok {
		t.Errorf("films lus = %v, attendu le match de l'habitude (hier) en plus du périmètre", film.Films)
	}
	if len(repo.tiers) != 1 || !slices.Equal(repo.tiers[0], []string{"ce-soir", "hier"}) {
		t.Errorf("niveaux lus sur %v, attendu le périmètre puis l'habitude", repo.tiers)
	}
}

func TestEmpriseLecteurFilm_LHabitudeEnEchecDegradeSeule(t *testing.T) {
	current := []squademprise.Match{soireeDe("ce-soir", "s2", 22)}
	timeline := append([]squademprise.Match{soireeDe("hier", "s1", 21)}, current...)
	repo := &usageEspion{echecSur: "hier"}
	film, raison := EmpriseLecteur{Page: "test"}.Film(context.Background(), repo, current, timeline, nil)
	if raison != "" || film == nil {
		t.Fatalf("film = %v, raison %q ; l'échec de l'habitude ne doit pas retirer le film", film, raison)
	}
	if _, ok := film.Films["hier"]; ok || len(film.Films) != 1 {
		t.Errorf("films = %v, attendu le seul périmètre", film.Films)
	}
}

func TestEmpriseLecteurFilm_SansRepoLeTitreNAPasDeFilm(t *testing.T) {
	film, raison := EmpriseLecteur{Page: "test"}.Film(context.Background(), nil, nil, nil, nil)
	if film != nil || raison != domain.EmpriseFilmUnsupported {
		t.Errorf("(%v, %q), attendu (nil, %q)", film, raison, domain.EmpriseFilmUnsupported)
	}
}

// journalEspion — le journal des morts, qui note s'il a été lu.
type journalEspion struct{ lu bool }

func (*journalEspion) LoadPowerWeaponKills(context.Context, []string) ([]squademprise.PowerKillRow, error) {
	return nil, nil
}

func (j *journalEspion) LoadJournalWeaponKills(context.Context, []string) (squademprise.JournalRead, error) {
	j.lu = true
	return squademprise.JournalRead{Read: map[string]bool{"m": true}}, nil
}

// Catalogue d'armes vide : le journal n'est pas lu, les frags aux armes spéciales restent à la
// feuille de match (jamais un 0 / 0 faute de clé de registre).
func TestEmpriseLecteurJournal_CatalogueVide_RepliFeuille(t *testing.T) {
	repo := &journalEspion{}
	l := EmpriseLecteur{Page: "test", Player: "J"}
	if got := l.Journal(context.Background(), repo, []string{"m"}, nil); got != nil || repo.lu {
		t.Errorf("catalogue vide : journal = %+v, lu = %v ; attendu nil, non lu", got, repo.lu)
	}
	weapons := map[string]squadformes.WeaponInfo{"0a000001": {WeaponKey: "k"}}
	if got := l.Journal(context.Background(), repo, []string{"m"}, weapons); got == nil || !got.Read["m"] {
		t.Errorf("catalogue lu : journal = %+v, attendu la lecture du repo", got)
	}
}
