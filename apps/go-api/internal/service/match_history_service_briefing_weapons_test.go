package service

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// briefingWeaponsRepo est un lecteur d'armes factice qui MÉMORISE les filtres reçus :
// le contrat du bloc porte autant sur ce qu'il demande (xuid, pas gamertag) que sur ce
// qu'il rend.
type briefingWeaponsRepo struct {
	rows []port.WeaponKillRow
	err  error
	got  port.WeaponKillFilters
	// calls compte les appels : le bloc ne doit interroger la base qu'une fois.
	calls int
}

func (f *briefingWeaponsRepo) LoadWeaponKillsAggregated(
	_ context.Context, _ string, filters port.WeaponKillFilters,
) ([]port.WeaponKillRow, error) {
	f.calls++
	f.got = filters
	return f.rows, f.err
}

// briefingWeaponsScope construit un scope de n matchs portant chacun kills frags.
func briefingWeaponsScope(n, kills int) []domain.MatchHistoryRawRow {
	rows := make([]domain.MatchHistoryRawRow, 0, n)
	for i := 0; i < n; i++ {
		rows = append(rows, briefingRaw("m"+string(rune('a'+i)), i, domain.OutcomeWin, kills, 5, 2, 60, "map1", "Aquarius", "Slayer", "Arène classée"))
	}
	return rows
}

func TestBuildBriefingWeapons_TopTwoAndMeasuredDenominator(t *testing.T) {
	repo := &briefingWeaponsRepo{rows: []port.WeaponKillRow{
		{WeaponID: 1, Label: "Fusil de combat", Kills: 40, Class: "shoulder"},
		{WeaponID: 2, Label: "Pistolet", Kills: 25, Class: "sidearm"},
		{WeaponID: 3, Label: "Fusil à pompe", Kills: 5, Class: "shoulder"},
		// Libellé non résolu : compte pour zéro, ni au classement ni au dénominateur.
		{WeaponID: 4, Label: "", Kills: 100},
		// Sentinelle grenade/mêlée : même traitement (ce n'est pas une arme du classement).
		{WeaponID: 5, Label: "Grenade à fragmentation", Kills: 30, IsGrenadeMelee: true},
	}}
	// 12 matchs à 10 frags = 120 frags au scope.
	got := buildBriefingWeapons(context.Background(), repo, "halo_infinite", "2533", briefingWeaponsScope(12, 10))
	if got == nil {
		t.Fatal("bloc nil alors que des armes sont mesurées")
	}
	if len(got.Entries) != 2 {
		t.Fatalf("entries = %d, want 2 (troncature au top 2)", len(got.Entries))
	}
	if got.Entries[0].Label != "Fusil de combat" || got.Entries[0].Kills != 40 {
		t.Errorf("premiere entree = %+v, want Fusil de combat/40", got.Entries[0])
	}
	if got.Entries[1].Label != "Pistolet" || got.Entries[1].Kills != 25 {
		t.Errorf("seconde entree = %+v, want Pistolet/25", got.Entries[1])
	}
	if got.Entries[0].Class != "shoulder" {
		t.Errorf("Class doit remonter jusqu'au front (couleur de barre), got %q", got.Entries[0].Class)
	}
	// 40 + 25 + 5 : la troisième arme entre au dénominateur sans être affichée ; ni la
	// ligne sans libellé ni la sentinelle grenade n'y entrent.
	if got.MeasuredKills != 70 {
		t.Errorf("MeasuredKills = %d, want 70", got.MeasuredKills)
	}
	if got.ScopeKills != 120 {
		t.Errorf("ScopeKills = %d, want 120", got.ScopeKills)
	}
	if repo.calls != 1 {
		t.Errorf("calls = %d, want 1 (une seule lecture par briefing)", repo.calls)
	}
}

func TestBuildBriefingWeapons_FiltersPassedToRepo(t *testing.T) {
	repo := &briefingWeaponsRepo{rows: []port.WeaponKillRow{
		{WeaponID: 1, Label: "Fusil de combat", Kills: 10},
	}}
	scope := briefingWeaponsScope(3, 4)
	if got := buildBriefingWeapons(context.Background(), repo, "halo_5", "2533274823110022", scope); got == nil {
		t.Fatal("bloc nil alors qu'une arme est mesurée")
	}
	f := repo.got
	if len(f.XUIDs) != 1 || f.XUIDs[0] != "2533274823110022" {
		t.Errorf("XUIDs = %v, want [2533274823110022]", f.XUIDs)
	}
	if f.Gamertag != "" {
		t.Errorf("Gamertag = %q, want vide : le filtre gamertag passe par une jointure d'alias", f.Gamertag)
	}
	if f.IncludeGrenadeMelee {
		t.Error("IncludeGrenadeMelee doit rester faux (sentinelles grenade/melee hors classement)")
	}
	if !f.ResolveRoles {
		t.Error("ResolveRoles doit etre vrai (Class alimente la couleur de la barre)")
	}
	if len(f.MatchIDs) != len(scope) {
		t.Errorf("MatchIDs = %d, want %d (les matchs du scope filtre)", len(f.MatchIDs), len(scope))
	}
	if err := f.Validate(); err != nil {
		t.Errorf("filtres rejetes par le port : %v", err)
	}
}

func TestBuildBriefingWeapons_MeasuredZeroYieldsNil(t *testing.T) {
	scope := briefingWeaponsScope(12, 10)
	cases := map[string][]port.WeaponKillRow{
		"aucune ligne": nil,
		"que des libelles non resolus": {
			{WeaponID: 4, Label: "", Kills: 100},
			{WeaponID: 5, Label: "", Kills: 3},
		},
		"que des sentinelles grenade/melee": {
			{WeaponID: 6, Label: "Grenade à fragmentation", Kills: 30, IsGrenadeMelee: true},
		},
	}
	for name, rows := range cases {
		t.Run(name, func(t *testing.T) {
			repo := &briefingWeaponsRepo{rows: rows}
			if got := buildBriefingWeapons(context.Background(), repo, "halo_infinite", "2533", scope); got != nil {
				t.Errorf("bloc = %+v, want nil (aucun frag mesure)", got)
			}
		})
	}
}

func TestBuildBriefingWeapons_DegradesWithoutPropagating(t *testing.T) {
	scope := briefingWeaponsScope(12, 10)
	rows := []port.WeaponKillRow{{WeaponID: 1, Label: "Fusil de combat", Kills: 40}}
	t.Run("repo absent", func(t *testing.T) {
		if got := buildBriefingWeapons(context.Background(), nil, "halo_infinite", "2533", scope); got != nil {
			t.Errorf("bloc = %+v, want nil", got)
		}
	})
	t.Run("xuid vide", func(t *testing.T) {
		repo := &briefingWeaponsRepo{rows: rows}
		if got := buildBriefingWeapons(context.Background(), repo, "halo_infinite", "", scope); got != nil {
			t.Errorf("bloc = %+v, want nil", got)
		}
		if repo.calls != 0 {
			t.Error("sans xuid, aucune lecture ne doit partir")
		}
	})
	t.Run("scope vide", func(t *testing.T) {
		repo := &briefingWeaponsRepo{rows: rows}
		if got := buildBriefingWeapons(context.Background(), repo, "halo_infinite", "2533", nil); got != nil {
			t.Errorf("bloc = %+v, want nil", got)
		}
		if repo.calls != 0 {
			t.Error("sans scope, aucune lecture ne doit partir")
		}
	})
	t.Run("capability absente", func(t *testing.T) {
		repo := &briefingWeaponsRepo{err: games.ErrCapabilityNotSupported}
		if got := buildBriefingWeapons(context.Background(), repo, "halo_infinite", "2533", scope); got != nil {
			t.Errorf("bloc = %+v, want nil", got)
		}
	})
	t.Run("erreur inattendue", func(t *testing.T) {
		repo := &briefingWeaponsRepo{err: errors.New("duckdb: connexion perdue")}
		if got := buildBriefingWeapons(context.Background(), repo, "halo_infinite", "2533", scope); got != nil {
			t.Errorf("bloc = %+v, want nil", got)
		}
	})
}

func TestBuildExplorerBriefing_WeaponsModuleGating(t *testing.T) {
	rows := []port.WeaponKillRow{{WeaponID: 1, Label: "Fusil de combat", Kills: 40}}
	t.Run("scope nominal", func(t *testing.T) {
		svc := &MatchHistoryService{weaponKillsRepo: &briefingWeaponsRepo{rows: rows}, weaponKillsXUID: "2533"}
		scope := briefingWeaponsScope(12, 10)
		b := svc.buildExplorerBriefing(context.Background(), scope, scope)
		if b == nil || b.Weapons == nil {
			t.Fatalf("bloc arme favorite attendu, got %+v", b)
		}
		if len(b.Weapons.Entries) != 1 || b.Weapons.Entries[0].Label != "Fusil de combat" {
			t.Errorf("entries = %+v", b.Weapons.Entries)
		}
	})
	t.Run("low sample", func(t *testing.T) {
		repo := &briefingWeaponsRepo{rows: rows}
		svc := &MatchHistoryService{weaponKillsRepo: repo, weaponKillsXUID: "2533"}
		scope := briefingWeaponsScope(8, 10)
		b := svc.buildExplorerBriefing(context.Background(), scope, scope)
		if b == nil || !b.LowSample {
			t.Fatalf("low sample attendu, got %+v", b)
		}
		if b.Weapons != nil {
			t.Errorf("low sample : le bloc doit etre omis comme les autres modules, got %+v", b.Weapons)
		}
		if repo.calls != 0 {
			t.Error("low sample : aucune lecture ne doit partir (sortie avant les modules)")
		}
	})
}
