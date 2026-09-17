package service

// compare_weapons_test.go — LE PROFIL D'ARMES DU FACE-À-FACE, sans DuckDB
// (plan .ai/PLAN_COMPARE_PROFIL_ARMES_2026-09-17.md, lot 3, item 3.7).
//
// # CE QUE CES TESTS VERROUILLENT, DANS L'ORDRE D'IMPORTANCE
//
//  1. LE SCOPE DE B N'EST PAS CELUI DE A. Un joueur non suivi n'existe localement que par les
//     matchs joués AVEC le joueur courant : son profil décrit un ÉCHANTILLON, et la réponse
//     doit le dire (`is_sample`). Servir sa carrière là où on n'a vu que trois matchs est
//     l'erreur la plus coûteuse de cette section, et la plus invisible.
//  2. CHAQUE BLOC TOMBE SEUL (D9). Un titre sans positions par kill garde ses classes et son
//     top 3 ; seul le bloc de portée disparaît. Faire tomber le profil entier cacherait ce
//     qui est pourtant mesuré.
//  3. LES PARTS SOMMENT À 100. Le résidu « non attribué » est CONSERVÉ : l'écarter ferait des
//     parts qui ne bouclent pas, et le lecteur mettrait la différence sur un arrondi alors
//     qu'elle mesure un trou de registre.
//  4. UNE CLÉ D'ARME SANS RÔLE EST ÉCARTÉE (D8), jamais rangée sous un seau fourre-tout qui
//     mélangerait une épée et un fusil de précision dans une même « portée ».

import (
	"context"
	"errors"
	"math"
	"testing"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

const epsPart = 0.01

// ─── fakes ───────────────────────────────────────────────────────────────────

// fakeCompareWeaponRepo — un port.CompareRepository dont les DEUX scopes sont scriptables par
// xuid. Fake dédié plutôt que `mockCompareRepoAB` étendu : les tests de métriques ont besoin
// d'un scope NUL par défaut, ceux-ci d'un scope par joueur, et un mock qui sert les deux
// besoins finit par ne servir aucun des deux lisiblement.
type fakeCompareWeaponRepo struct {
	statsA, statsB *domain.NormalizedPlayerStats
	xuidB          string
	// scopes : xuid -> scope lifetime. Absent = joueur non suivi localement.
	scopes map[string]*domain.CompareWeaponScope
	// cross : scope croisé rendu pour n'importe quel couple. nil = jamais croisé.
	cross *domain.CompareWeaponScope
	// scopeErr / crossErr : pannes de lecture (best-effort attendu côté service).
	scopeErr, crossErr error

	scopeCalls, crossCalls int
}

func (f *fakeCompareWeaponRepo) GetLocalStats(_ context.Context, xuid, _ string) (*domain.NormalizedPlayerStats, error) {
	if xuid == "xuid-a" {
		return f.statsA, nil
	}
	if f.statsB == nil {
		return nil, errors.New("joueur B non local")
	}
	return f.statsB, nil
}

func (f *fakeCompareWeaponRepo) ResolveXUID(_ context.Context, _ string) (string, error) {
	return f.xuidB, nil
}

func (f *fakeCompareWeaponRepo) GetPlayerATH(_ context.Context) (*domain.PlayerATH, error) {
	return &domain.PlayerATH{}, nil
}

func (f *fakeCompareWeaponRepo) GetPlayerATHFor(_ context.Context, _, _ string) (*domain.PlayerATH, error) {
	return &domain.PlayerATH{}, nil
}

func (f *fakeCompareWeaponRepo) GetEncounterStats(_ context.Context, _, _ string) (*domain.CompareEncounterStats, error) {
	return nil, nil
}

func (f *fakeCompareWeaponRepo) GetCrossMatchSample(_ context.Context, _, _ string) (*domain.CrossMatchSample, error) {
	return nil, nil
}

func (f *fakeCompareWeaponRepo) GetWeaponScope(_ context.Context, xuid, _ string) (*domain.CompareWeaponScope, error) {
	f.scopeCalls++
	if f.scopeErr != nil {
		return nil, f.scopeErr
	}
	return f.scopes[xuid], nil
}

func (f *fakeCompareWeaponRepo) GetCrossWeaponScope(_ context.Context, _, _, _ string) (*domain.CompareWeaponScope, error) {
	f.crossCalls++
	if f.crossErr != nil {
		return nil, f.crossErr
	}
	return f.cross, nil
}

// fakeCompareWeaponKills — un port.WeaponKillsRepository scriptable par xuid.
type fakeCompareWeaponKills struct {
	rows map[string][]port.WeaponKillRow
	err  error
	// lastFilters : mémorisé pour vérifier que le scope voyage bien jusqu'au repo.
	lastFilters port.WeaponKillFilters
}

func (f *fakeCompareWeaponKills) LoadWeaponKillsAggregated(
	_ context.Context, _ string, filters port.WeaponKillFilters,
) ([]port.WeaponKillRow, error) {
	f.lastFilters = filters
	if f.err != nil {
		return nil, f.err
	}
	if len(filters.XUIDs) == 0 {
		return nil, nil
	}
	return f.rows[filters.XUIDs[0]], nil
}

// fakeCompareWeaponRange — un port.WeaponRangeRepository scriptable par xuid.
type fakeCompareWeaponRange struct {
	kills map[string][]analysis.MeasuredKill
	dims  map[string]port.WeaponDimensions
	err   error
}

func (f *fakeCompareWeaponRange) LoadWeaponRange(
	_ context.Context, _ string, filters port.WeaponRangeFilters,
) ([]analysis.MeasuredKill, error) {
	if f.err != nil {
		return nil, f.err
	}
	if len(filters.XUIDs) == 0 {
		return nil, nil
	}
	return f.kills[filters.XUIDs[0]], nil
}

func (f *fakeCompareWeaponRange) LoadWeaponOpening(
	_ context.Context, _ string, _ port.WeaponRangeFilters,
) ([]analysis.MeasuredKill, error) {
	// L'entame est hors périmètre de la page : le service ne doit jamais l'appeler, et
	// une erreur ici le dirait tout de suite si cela changeait.
	return nil, errors.New("LoadWeaponOpening ne doit pas etre appele par le compare")
}

func (f *fakeCompareWeaponRange) ResolveWeaponDimensions(
	_ context.Context, _ string, _ []string,
) (map[string]port.WeaponDimensions, error) {
	return f.dims, nil
}

func (f *fakeCompareWeaponRange) ResolveWeaponLabels(
	_ context.Context, _ []string,
) (map[string]port.WeaponLabel, error) {
	return nil, nil
}

// ─── fixtures ────────────────────────────────────────────────────────────────

// cwScope : un scope de n matchs, avec ses totaux.
func cwScope(n int, kills, deaths, melee, grenade int) *domain.CompareWeaponScope {
	ids := make([]string, 0, n)
	for i := 0; i < n; i++ {
		ids = append(ids, "m"+string(rune('a'+i)))
	}
	return &domain.CompareWeaponScope{
		MatchIDs: ids, Matches: n,
		Kills: kills, Deaths: deaths, MeleeKills: melee, GrenadeKills: grenade,
	}
}

// cwRows : trois armes résolues (épaule/poing/lourde) aux frags décroissants.
func cwRows() []port.WeaponKillRow {
	return []port.WeaponKillRow{
		{WeaponID: 1, WeaponKey: "br75", Label: "BR75", LabelEN: "BR75",
			Class: domain.FragClassShoulder, Role: "precision", Kills: 10},
		{WeaponID: 2, WeaponKey: "sidekick", Label: "Sidekick",
			Class: domain.FragClassSidearm, Role: "sidearm", Kills: 5},
		{WeaponID: 3, WeaponKey: "sniper", Label: "Sniper",
			Class: domain.FragClassHeavy, Role: "sniper", Kills: 3},
	}
}

// cwMeasured : n frags mesurés d'une arme et d'un côté, à distance fixe.
func cwMeasured(weaponKey string, side analysis.Side, n int, dist float64) []analysis.MeasuredKill {
	out := make([]analysis.MeasuredKill, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, analysis.MeasuredKill{
			MatchID: "ma", TimeMS: int64(i), WeaponKey: weaponKey, Side: side, DistanceM: dist,
		})
	}
	return out
}

// cwService monte un CompareService câblé sur les trois fakes.
func cwService(
	repo *fakeCompareWeaponRepo, kills *fakeCompareWeaponKills, rng *fakeCompareWeaponRange,
) *CompareService {
	svc := NewCompareService(repo, &mockStatsProvider{statsErr: errors.New("non utilise")},
		"xuid-a", "hi")
	return svc.WithWeaponProfile(kills, rng, func(id int64) (string, bool) {
		if id == 1 {
			return "https://cdn/br75.png", true
		}
		return "", false
	})
}

// sommeDesParts additionne les parts publiées d'un côté.
func sommeDesParts(side domain.CompareWeaponSide) float64 {
	total := 0.0
	for _, c := range side.FragClasses {
		total += c.SharePct
	}
	return total
}

// ─── tests ───────────────────────────────────────────────────────────────────

// TestBuildWeaponProfile_DeuxJoueursLocaux : deux scopes LIFETIME, aucun échantillon, les
// trois blocs présents des deux côtés.
func TestBuildWeaponProfile_DeuxJoueursLocaux(t *testing.T) {
	repo := &fakeCompareWeaponRepo{scopes: map[string]*domain.CompareWeaponScope{
		"xuid-a": cwScope(3, 25, 12, 4, 3),
		"xuid-b": cwScope(2, 18, 9, 2, 1),
	}}
	kills := &fakeCompareWeaponKills{rows: map[string][]port.WeaponKillRow{
		"xuid-a": cwRows(), "xuid-b": cwRows(),
	}}
	rng := &fakeCompareWeaponRange{
		kills: map[string][]analysis.MeasuredKill{
			"xuid-a": cwMeasured("br75", analysis.SideKiller, 9, 12),
		},
		dims: map[string]port.WeaponDimensions{
			"br75": {Class: domain.FragClassShoulder, Role: "precision"},
		},
	}

	profile := cwService(repo, kills, rng).buildWeaponProfile(context.Background(), "xuid-b")
	if profile == nil {
		t.Fatal("profil nil : les deux joueurs ont un scope")
	}
	if profile.PlayerA.Matches != 3 || profile.PlayerB.Matches != 2 {
		t.Errorf("matchs = A%d B%d, attendu A3 B2", profile.PlayerA.Matches, profile.PlayerB.Matches)
	}
	if profile.PlayerA.IsSample || profile.PlayerB.IsSample {
		t.Errorf("aucun échantillon attendu (les deux joueurs sont locaux) : A=%v B=%v",
			profile.PlayerA.IsSample, profile.PlayerB.IsSample)
	}
	if repo.crossCalls != 0 {
		t.Errorf("scope croisé appelé %d fois : B est local, son scope est sa carrière",
			repo.crossCalls)
	}
	if profile.PlayerA.Range == nil {
		t.Error("bloc de portée A attendu : 9 frags mesurés au-dessus du seuil")
	}
	if profile.PlayerB.Range != nil {
		t.Error("bloc de portée B attendu nil : aucun frag mesuré pour B")
	}
	if len(profile.PlayerA.TopWeapons) != 3 {
		t.Errorf("top armes A = %d, attendu 3", len(profile.PlayerA.TopWeapons))
	}
}

// TestBuildWeaponProfile_BNonLocalEstUnEchantillon : B n'a pas de scope lifetime ; son profil
// vient du scope CROISÉ et se déclare échantillon. C'est la distinction que la page doit
// annoncer — sans elle, une poignée de matchs se lit comme une carrière.
func TestBuildWeaponProfile_BNonLocalEstUnEchantillon(t *testing.T) {
	repo := &fakeCompareWeaponRepo{
		scopes: map[string]*domain.CompareWeaponScope{"xuid-a": cwScope(5, 40, 20, 5, 5)},
		cross:  cwScope(2, 7, 11, 1, 0),
	}
	kills := &fakeCompareWeaponKills{rows: map[string][]port.WeaponKillRow{
		"xuid-a": cwRows(), "xuid-b": cwRows(),
	}}
	rng := &fakeCompareWeaponRange{}

	profile := cwService(repo, kills, rng).buildWeaponProfile(context.Background(), "xuid-b")
	if profile == nil {
		t.Fatal("profil nil : A a un scope et B est croisé")
	}
	if !profile.PlayerB.IsSample {
		t.Error("is_sample attendu vrai : B n'est pas suivi localement")
	}
	if profile.PlayerB.Matches != 2 {
		t.Errorf("matchs B = %d, attendu 2 (ceux du scope croisé, pas la carrière)",
			profile.PlayerB.Matches)
	}
	if profile.PlayerB.TotalKills != 7 {
		t.Errorf("frags B = %d, attendu 7 (ceux du scope croisé)", profile.PlayerB.TotalKills)
	}
	if profile.PlayerA.IsSample {
		t.Error("A est local : son profil n'est jamais un échantillon")
	}
}

// TestBuildWeaponProfile_BJamaisCroise_ProfilAbsent : ni scope lifetime ni scope croisé —
// la section entière disparaît, proprement.
func TestBuildWeaponProfile_BJamaisCroise_ProfilAbsent(t *testing.T) {
	repo := &fakeCompareWeaponRepo{
		scopes: map[string]*domain.CompareWeaponScope{"xuid-a": cwScope(5, 40, 20, 5, 5)},
	}
	svc := cwService(repo, &fakeCompareWeaponKills{}, &fakeCompareWeaponRange{})

	if p := svc.buildWeaponProfile(context.Background(), "xuid-b"); p != nil {
		t.Fatalf("profil = %+v, attendu nil : B n'a ni carrière locale ni match commun", p)
	}
}

// TestBuildWeaponProfile_BSansXUID_ProfilAbsent : un B que rien n'a résolu (jamais croisé,
// résolution live en échec) n'a aucun scope possible — et le scope croisé n'est même pas tenté.
func TestBuildWeaponProfile_BSansXUID_ProfilAbsent(t *testing.T) {
	repo := &fakeCompareWeaponRepo{
		scopes: map[string]*domain.CompareWeaponScope{"xuid-a": cwScope(5, 40, 20, 5, 5)},
		cross:  cwScope(2, 7, 11, 1, 0),
	}
	svc := cwService(repo, &fakeCompareWeaponKills{}, &fakeCompareWeaponRange{})

	if p := svc.buildWeaponProfile(context.Background(), ""); p != nil {
		t.Fatalf("profil = %+v, attendu nil : aucun xuid pour B", p)
	}
	if repo.crossCalls != 0 {
		t.Errorf("scope croisé appelé %d fois sans xuid B : la jointure n'aurait aucun sens",
			repo.crossCalls)
	}
}

// TestBuildWeaponProfile_CapabilityAbsente_PorteeSeuleTombe : le repo de portée rend
// games.ErrCapabilityNotSupported (titre sans positions par kill, cas Halo 5). Classes et top
// 3 restent publiés — c'est exactement la dégradation par BLOC que D9 exige.
func TestBuildWeaponProfile_CapabilityAbsente_PorteeSeuleTombe(t *testing.T) {
	repo := &fakeCompareWeaponRepo{scopes: map[string]*domain.CompareWeaponScope{
		"xuid-a": cwScope(3, 25, 12, 4, 3),
		"xuid-b": cwScope(3, 25, 12, 4, 3),
	}}
	kills := &fakeCompareWeaponKills{rows: map[string][]port.WeaponKillRow{
		"xuid-a": cwRows(), "xuid-b": cwRows(),
	}}
	rng := &fakeCompareWeaponRange{err: games.ErrCapabilityNotSupported}

	profile := cwService(repo, kills, rng).buildWeaponProfile(context.Background(), "xuid-b")
	if profile == nil {
		t.Fatal("profil nil : l'absence de portée ne doit pas emporter la section")
	}
	if profile.PlayerA.Range != nil || profile.PlayerB.Range != nil {
		t.Error("blocs de portée attendus nil (capability absente)")
	}
	if len(profile.PlayerA.FragClasses) == 0 || len(profile.PlayerA.TopWeapons) == 0 {
		t.Errorf("classes et top 3 doivent survivre : %+v", profile.PlayerA)
	}
}

// TestBuildWeaponProfile_PartsSommentACent : le résidu « non attribué » est CONSERVÉ, et
// c'est lui qui fait boucler les parts. La fixture laisse 5 frags non attribuables (25 frags
// au scope, 20 attribués) pour que son absence se voie.
func TestBuildWeaponProfile_PartsSommentACent(t *testing.T) {
	repo := &fakeCompareWeaponRepo{scopes: map[string]*domain.CompareWeaponScope{
		"xuid-a": cwScope(3, 25, 12, 1, 1), // 10+5+3 (armes) + 1 + 1 = 20 -> 5 non attribués
		"xuid-b": cwScope(3, 18, 9, 0, 0),
	}}
	kills := &fakeCompareWeaponKills{rows: map[string][]port.WeaponKillRow{
		"xuid-a": cwRows(), "xuid-b": cwRows(),
	}}

	profile := cwService(repo, kills, &fakeCompareWeaponRange{}).
		buildWeaponProfile(context.Background(), "xuid-b")
	if profile == nil {
		t.Fatal("profil nil")
	}
	if got := sommeDesParts(profile.PlayerA); math.Abs(got-100) > epsPart {
		t.Fatalf("somme des parts A = %v, attendu 100 ± %v : %+v",
			got, epsPart, profile.PlayerA.FragClasses)
	}
	var residu *domain.CompareFragClass
	for i := range profile.PlayerA.FragClasses {
		if profile.PlayerA.FragClasses[i].Class == domain.FragClassUnattributed {
			residu = &profile.PlayerA.FragClasses[i]
		}
	}
	if residu == nil || residu.Kills != 5 {
		t.Fatalf("classe « non attribué » attendue à 5 frags, obtenu %+v", residu)
	}
	if profile.PlayerA.TotalKills != 25 {
		t.Errorf("dénominateur = %d, attendu 25 (le total du scope, pas la somme des classes)",
			profile.PlayerA.TotalKills)
	}
}

// TestBuildWeaponProfile_PorteeParRoleEtCleSansRoleEcartee : deux ARMES distinctes fusionnent
// en UNE ligne de rôle, et une clé que le registre ne connaît pas sort du corpus (D8) au lieu
// de fabriquer une ligne qui mélangerait des portées sans rapport.
func TestBuildWeaponProfile_PorteeParRoleEtCleSansRoleEcartee(t *testing.T) {
	mesures := append(cwMeasured("br75", analysis.SideKiller, 5, 12),
		cwMeasured("commando", analysis.SideKiller, 5, 14)...)
	mesures = append(mesures, cwMeasured("arme_hors_registre", analysis.SideKiller, 9, 40)...)

	repo := &fakeCompareWeaponRepo{scopes: map[string]*domain.CompareWeaponScope{
		"xuid-a": cwScope(3, 25, 12, 4, 3),
		"xuid-b": cwScope(3, 25, 12, 4, 3),
	}}
	kills := &fakeCompareWeaponKills{rows: map[string][]port.WeaponKillRow{
		"xuid-a": cwRows(), "xuid-b": cwRows(),
	}}
	rng := &fakeCompareWeaponRange{
		kills: map[string][]analysis.MeasuredKill{"xuid-a": mesures},
		dims: map[string]port.WeaponDimensions{
			"br75":     {Class: domain.FragClassShoulder, Role: "precision"},
			"commando": {Class: domain.FragClassShoulder, Role: "precision"},
			// « arme_hors_registre » : absente — le registre ne lui donne aucun rôle.
		},
	}

	profile := cwService(repo, kills, rng).buildWeaponProfile(context.Background(), "xuid-b")
	if profile == nil || profile.PlayerA.Range == nil {
		t.Fatalf("bloc de portée A attendu : %+v", profile)
	}
	rows := profile.PlayerA.Range.Weapons
	if len(rows) != 1 {
		t.Fatalf("une seule ligne de rôle attendue, obtenu %d : %+v", len(rows), rows)
	}
	if rows[0].WeaponKey != "precision" {
		t.Errorf("clé de ligne = %q, attendu « precision » : le grain publié est le RÔLE (D1)",
			rows[0].WeaponKey)
	}
	if rows[0].Label != "" || rows[0].LabelEN != "" {
		t.Errorf("libellés attendus VIDES (D8 : le front résout frags.role.<clé>), obtenu %q/%q",
			rows[0].Label, rows[0].LabelEN)
	}
	if rows[0].Kills == nil || rows[0].Kills.Measured != 10 {
		t.Fatalf("10 frags mesurés attendus sur la ligne « precision » (5 BR75 + 5 Commando), "+
			"obtenu %+v", rows[0].Kills)
	}
	if profile.PlayerA.Range.Opening != nil {
		t.Error("bloc d'entame attendu nil : l'entame est hors périmètre de cette page")
	}
}

// TestBuildWeaponProfile_TopArmesBorneEtTrie : le top est plafonné à 3, trié par frags
// décroissants, et les compteurs natifs (mêlée, grenade) n'y entrent pas — ils ne sont pas des
// armes de l'arsenal, et les y ranger ferait concourir « Mêlée » contre un fusil.
func TestBuildWeaponProfile_TopArmesBorneEtTrie(t *testing.T) {
	rows := append(cwRows(), port.WeaponKillRow{
		WeaponID: 4, WeaponKey: "hydra", Label: "Hydra",
		Class: domain.FragClassHeavy, Role: "power", Kills: 2,
	}, port.WeaponKillRow{
		WeaponID: 5, Label: "Mêlée", Kills: 99, IsGrenadeMelee: true,
	}, port.WeaponKillRow{
		WeaponID: 6, WeaponKey: "sans_nom", Class: domain.FragClassShoulder, Role: "automatic", Kills: 50,
	})

	repo := &fakeCompareWeaponRepo{scopes: map[string]*domain.CompareWeaponScope{
		"xuid-a": cwScope(3, 25, 12, 4, 3),
		"xuid-b": cwScope(3, 25, 12, 4, 3),
	}}
	kills := &fakeCompareWeaponKills{rows: map[string][]port.WeaponKillRow{
		"xuid-a": rows, "xuid-b": rows,
	}}

	profile := cwService(repo, kills, &fakeCompareWeaponRange{}).
		buildWeaponProfile(context.Background(), "xuid-b")
	if profile == nil {
		t.Fatal("profil nil")
	}
	top := profile.PlayerA.TopWeapons
	if len(top) != 3 {
		t.Fatalf("top = %d entrées, attendu 3 : %+v", len(top), top)
	}
	attendu := []string{"BR75", "Sidekick", "Sniper"}
	for i := range attendu {
		if top[i].Label != attendu[i] {
			t.Fatalf("top = %v, attendu %v (Mêlée est un compteur natif, « sans_nom » n'a pas "+
				"de libellé résolu)", []string{top[0].Label, top[1].Label, top[2].Label}, attendu)
		}
	}
	if top[0].ImageURL != "https://cdn/br75.png" || !top[0].ImageTinted {
		t.Errorf("icône du BR75 = %q (masque %v), attendu l'URL injectée et un masque",
			top[0].ImageURL, top[0].ImageTinted)
	}
	if top[1].ImageURL != "" || top[1].ImageTinted {
		t.Errorf("arme sans icône : URL vide et masque faux attendus, obtenu %q/%v",
			top[1].ImageURL, top[1].ImageTinted)
	}
}

// TestBuildWeaponProfile_ScopeVoyageJusquAuRepo : le repo d'armes est borné par les match_id
// du scope ET par le xuid du côté lu. Sans ce bornage, il balaierait une table PARTAGÉE, tous
// joueurs confondus — ce que son Validate() refuse.
func TestBuildWeaponProfile_ScopeVoyageJusquAuRepo(t *testing.T) {
	repo := &fakeCompareWeaponRepo{scopes: map[string]*domain.CompareWeaponScope{
		"xuid-a": cwScope(3, 25, 12, 4, 3),
		"xuid-b": cwScope(2, 18, 9, 2, 1),
	}}
	kills := &fakeCompareWeaponKills{rows: map[string][]port.WeaponKillRow{
		"xuid-a": cwRows(), "xuid-b": cwRows(),
	}}

	cwService(repo, kills, &fakeCompareWeaponRange{}).
		buildWeaponProfile(context.Background(), "xuid-b")

	f := kills.lastFilters // dernier appel = côté B
	if len(f.MatchIDs) != 2 || len(f.XUIDs) != 1 || f.XUIDs[0] != "xuid-b" {
		t.Fatalf("filtres du dernier appel = %+v, attendu 2 matchs et le xuid de B", f)
	}
	if !f.ResolveRoles {
		t.Error("ResolveRoles attendu vrai : classe et rôle doivent arriver dans la MÊME passe")
	}
	if err := f.Validate(); err != nil {
		t.Errorf("les filtres produits doivent passer Validate() : %v", err)
	}
}

// TestBuildWeaponProfile_PanneDeScope_SectionAbsenteSansErreur : une lecture en échec ne fait
// pas tomber la page — elle retire la section. Le profil est ADDITIF à une page qui existait
// avant lui.
func TestBuildWeaponProfile_PanneDeScope_SectionAbsenteSansErreur(t *testing.T) {
	repo := &fakeCompareWeaponRepo{scopeErr: errors.New("boum")}
	svc := cwService(repo, &fakeCompareWeaponKills{}, &fakeCompareWeaponRange{})

	if p := svc.buildWeaponProfile(context.Background(), "xuid-b"); p != nil {
		t.Fatalf("profil = %+v, attendu nil sur panne de lecture", p)
	}
}

// TestBuildWeaponProfile_SansCablage_ProfilAbsent : sans les deux repos, `Weapons` reste
// absent et la page sert ses métriques comme avant. C'est l'état de tout titre non câblé.
func TestBuildWeaponProfile_SansCablage_ProfilAbsent(t *testing.T) {
	repo := &fakeCompareWeaponRepo{scopes: map[string]*domain.CompareWeaponScope{
		"xuid-a": cwScope(3, 25, 12, 4, 3),
	}}
	svc := NewCompareService(repo, &mockStatsProvider{}, "xuid-a", "hi")
	if p := svc.buildWeaponProfile(context.Background(), "xuid-b"); p != nil {
		t.Fatalf("profil = %+v, attendu nil sans câblage", p)
	}
	// Un seul repo ne suffit pas : un demi-profil se lirait comme une page cassée.
	svc = svc.WithWeaponProfile(&fakeCompareWeaponKills{}, nil, nil)
	if p := svc.buildWeaponProfile(context.Background(), "xuid-b"); p != nil {
		t.Fatalf("profil = %+v, attendu nil avec un seul repo", p)
	}
}

// TestCompareService_GetPage_PorteLeProfilDArmes : le câblage jusqu'à la réponse. Les tests
// ci-dessus portent sur le constructeur du profil ; celui-ci vérifie qu'il atteint bien
// `CompareResponse.Weapons`.
func TestCompareService_GetPage_PorteLeProfilDArmes(t *testing.T) {
	repo := &fakeCompareWeaponRepo{
		statsA: &domain.NormalizedPlayerStats{Gamertag: "PlayerA", Matches: 100},
		statsB: &domain.NormalizedPlayerStats{Gamertag: "PlayerB", Matches: 80},
		xuidB:  "xuid-b",
		scopes: map[string]*domain.CompareWeaponScope{
			"xuid-a": cwScope(3, 25, 12, 4, 3),
			"xuid-b": cwScope(2, 18, 9, 2, 1),
		},
	}
	kills := &fakeCompareWeaponKills{rows: map[string][]port.WeaponKillRow{
		"xuid-a": cwRows(), "xuid-b": cwRows(),
	}}
	svc := cwService(repo, kills, &fakeCompareWeaponRange{})

	resp, err := svc.GetPage(context.Background(), domain.CompareRequest{TargetGamertag: "PlayerB"})
	if err != nil {
		t.Fatalf("GetPage: %v", err)
	}
	if resp.Weapons == nil {
		t.Fatal("resp.Weapons nil : le profil n'atteint pas la réponse")
	}
	if resp.Weapons.PlayerA.Matches != 3 || resp.Weapons.PlayerB.Matches != 2 {
		t.Errorf("matchs = A%d B%d, attendu A3 B2",
			resp.Weapons.PlayerA.Matches, resp.Weapons.PlayerB.Matches)
	}
	if len(resp.Metrics) == 0 {
		t.Error("les métriques de la page doivent rester intactes")
	}
}
