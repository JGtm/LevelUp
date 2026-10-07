package service

// tactical_service_cellule_enrichir_test.go — LA MINI-TUILE « REJEU » DU DÉTAIL D'UNE ZONE : mode,
// score (mon camp d'abord, manches sur une variante qui se décide aux manches), arme ou catégorie,
// placement d'une mort, présence du rejeu. Chaque source est lue UNE fois par requête, chacune peut
// manquer ou échouer sans retirer une contribution, et chaque dégradation est journalisée.

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

const (
	tagBR   uint32 = 11 // source classée : la clé de registre « br75 »
	tagTete uint32 = 22 // source hors registre : sa catégorie seule (« Headshot »)
)

// fakeCelluleMatchs double la lecture canonique et compte ses appels.
type fakeCelluleMatchs struct {
	rows   []canonical.PlayerMatchRow
	err    error
	appels int
	vu     port.PlayerMatchFilters
}

func (f *fakeCelluleMatchs) LoadPlayerMatches(_ context.Context, _, _ string, fl port.PlayerMatchFilters) ([]canonical.PlayerMatchRow, error) {
	f.appels++
	f.vu = fl
	return f.rows, f.err
}

func (f *fakeCelluleMatchs) InvalidatePlayer(string, string) {}

// fakeClassifieur traduit une source de dégât en clé de registre.
type fakeClassifieur map[uint32]string

func (f fakeClassifieur) KillSourceRegistryKey(tag uint32) (string, bool) {
	k, ok := f[tag]
	return k, ok
}

// fakeLibellesArmes double le registre des noms d'armes et compte ses appels.
type fakeLibellesArmes struct {
	labels map[string]port.WeaponLabel
	err    error
	appels int
	cles   []string
}

func (f *fakeLibellesArmes) ResolveWeaponLabels(_ context.Context, keys []string) (map[string]port.WeaponLabel, error) {
	f.appels++
	f.cles = append([]string(nil), keys...)
	if f.err != nil {
		return nil, f.err
	}
	out := make(map[string]port.WeaponLabel, len(keys))
	for _, k := range keys {
		if l, ok := f.labels[k]; ok {
			out[k] = l
		}
	}
	return out, nil
}

func entier(v int) *int          { return &v }
func etiquette(v uint32) *uint32 { return &v }

// ligneCanonique : un match de la carte, mon camp et le score de chaque camp.
func ligneCanonique(id, variante, modeFR, modeEN string, monCamp int, equipes ...canonical.TeamSnapshot) canonical.PlayerMatchRow {
	r := canonical.PlayerMatchRow{Self: canonical.MatchParticipant{TeamID: entier(monCamp)}}
	r.Summary.MatchID = id
	r.Summary.GameVariant = &canonical.AssetReference{DefaultLabel: variante}
	r.Summary.PairMode = &canonical.AssetReference{DefaultLabel: modeEN, Labels: map[string]string{"fr": modeFR, "en": modeEN}}
	r.Summary.Teams = equipes
	return r
}

// corpusDetail : la cellule (4, 4) de deux matchs.
//
//	m1 (Arène, 18 m ; camp 0, 50 - 42)    je meurs à 1 000 ms (Rival, BR75 ; aucun coéquipier
//	                                      visible), je tue à 2 000 ms (Cible, tir à la tête non classé) ;
//	m2 (Oddball aux manches, 24 m)        je suis au camp 1, qui gagne 2 manches à 1 en marquant moins
//	                                      de points ; je meurs à 3 000 ms (coéquipier à 10 m) et à
//	                                      4 000 ms (contexte à 4 500 ms, coéquipier à 30 m), sans source.
//
// m1 est le plus récent : il sort en premier. Seul m1 a un rejeu.
func corpusDetail() (*mockTacticalRepo, *fakeCelluleMatchs) {
	repo := &mockTacticalRepo{}
	repo.pos.Univers = universVariantes(map[string]string{"m1": "Slayer:Arena", "m2": "Oddball:Rounds"})
	repo.pos.Points = []domain.TacticalKillPosition{
		{MatchID: "m1", KillerXUID: tsAdv, VictimXUID: tsMoi, KillerGamertag: "Rival", KillerX: 9, KillerY: 9,
			VictimX: 2.1, VictimY: 2.1, TimeMs: 1000, SourceTag: etiquette(tagBR), SourceCategory: "Bullet"},
		{MatchID: "m1", KillerXUID: tsMoi, VictimXUID: tsAdv, VictimGamertag: "Cible", KillerX: 2.2, KillerY: 2.2,
			VictimX: 9, VictimY: 9, TimeMs: 2000, SourceTag: etiquette(tagTete), SourceCategory: "Headshot"},
		{MatchID: "m2", KillerXUID: tsAdv, VictimXUID: tsMoi, KillerX: 9, KillerY: 9, VictimX: 2.3, VictimY: 2.3, TimeMs: 3000},
		{MatchID: "m2", KillerXUID: tsAdv2, VictimXUID: tsMoi, KillerX: 9, KillerY: 9, VictimX: 2.4, VictimY: 2.4, TimeMs: 4000},
	}
	repo.contextes = []domain.ContexteDeMort{
		{MatchID: "m1", VictimXUID: tsMoi, TimeMs: 1000, Visibles: 0, HorsDeVue: 2},
		{MatchID: "m2", VictimXUID: tsMoi, TimeMs: 3000, PlusProcheM: m(10), Visibles: 1},
		{MatchID: "m2", VictimXUID: tsMoi, TimeMs: 4500, PlusProcheM: m(30), Visibles: 1},
	}
	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	repo.ouvrables = map[string]time.Time{"m1": base.Add(time.Hour), "m2": base}
	matchs := &fakeCelluleMatchs{rows: []canonical.PlayerMatchRow{
		ligneCanonique("m1", "Slayer:Arena", "Assassin", "Slayer", 0,
			canonical.TeamSnapshot{TeamID: 0, Score: entier(50)}, canonical.TeamSnapshot{TeamID: 1, Score: entier(42)}),
		ligneCanonique("m2", "Oddball:Rounds", "Oddball", "Oddball", 1,
			canonical.TeamSnapshot{TeamID: 0, Score: entier(150), RoundsWon: entier(1)},
			canonical.TeamSnapshot{TeamID: 1, Score: entier(120), RoundsWon: entier(2)}),
	}}
	matchs.rows[1].Summary.RoundsTotal = entier(3)
	return repo, matchs
}

// sourcesDuDetailTest : les dépendances du détail, et le journal du service.
type sourcesDuDetailTest struct {
	matchs *fakeCelluleMatchs
	armes  *fakeLibellesArmes
	rejeu  *stubReplayService
	class  port.KillSourceClassifier
	trace  *bytes.Buffer
}

func sourcesCompletes(matchs *fakeCelluleMatchs) sourcesDuDetailTest {
	return sourcesDuDetailTest{
		matchs: matchs,
		armes:  &fakeLibellesArmes{labels: map[string]port.WeaponLabel{"br75": {Label: "BR75 (fr)", LabelEN: "BR75"}}},
		rejeu:  &stubReplayService{shortIDs: []string{"m1"}},
		class:  fakeClassifieur{tagBR: "br75"},
		trace:  &bytes.Buffer{},
	}
}

// service monte le TacticalService avec les sources présentes (une source nil n'est pas injectée).
func (src sourcesDuDetailTest) service(repo *mockTacticalRepo, caps games.CapabilityMap) *TacticalService {
	svc := NewTacticalService(repo, caps, tsMoi).
		WithLogger(slog.New(slog.NewTextHandler(src.trace, &slog.HandlerOptions{Level: slog.LevelDebug}))).
		WithRadarRange(map[string]int{"Slayer:Arena": 18, "Oddball:Rounds": 24}).
		WithRoundsDecide(map[string]bool{"Oddball:Rounds": true})
	if src.class != nil {
		svc = svc.WithKillSourceClassifier(src.class)
	}
	if src.matchs != nil {
		svc = svc.WithPlayerMatches(src.matchs, "slug_test", "Moi")
	}
	if src.armes != nil {
		svc = svc.WithWeaponLabels(src.armes)
	}
	if src.rejeu != nil {
		svc = svc.WithReplay(src.rejeu)
	}
	return svc
}

// attendu : ce qu'une contribution publie.
type attendu struct {
	mode, score, kind, arme, armeEN, categorie string
	placement                                  *domain.TacticalPlacement
	rejeu                                      bool
}

func verifier(t *testing.T, i int, c domain.TacticalContribution, w attendu) {
	t.Helper()
	if c.ModeLabel != w.mode || c.ScoreLabel != w.score || c.ScoreKind != w.kind || c.ArmeLabel != w.arme ||
		c.ArmeLabelEN != w.armeEN || c.CategorieSource != w.categorie || c.ReplayAvailable != w.rejeu {
		t.Errorf("contribution %d (%s, %d ms) = mode %q, score %q (%s), arme %q / %q, catégorie %q, rejeu %v ; want %+v",
			i, c.MatchID, c.InstantMs, c.ModeLabel, c.ScoreLabel, c.ScoreKind, c.ArmeLabel, c.ArmeLabelEN,
			c.CategorieSource, c.ReplayAvailable, w)
	}
	if !memePlacement(c.Placement, w.placement) {
		t.Errorf("contribution %d : placement = %+v, want %+v", i, c.Placement, w.placement)
	}
}

func memePlacement(a, b *domain.TacticalPlacement) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.Seul != b.Seul || (a.DistanceM == nil) != (b.DistanceM == nil) {
		return false
	}
	return a.DistanceM == nil || *a.DistanceM == *b.DistanceM
}

// TestDetail_ChaqueChamp : chaque champ de la mini-tuile, sur ses quatre contributions — score en
// points et en manches, mon camp d'abord au camp 1, arme par la clé, catégorie à défaut, rien à
// défaut ; badge seul sans distance, près, seul à distance, aucun sur un frag ; rejeu vrai ou faux.
func TestDetail_ChaqueChamp(t *testing.T) {
	repo, matchs := corpusDetail()
	got := celluleLue(t, sourcesCompletes(matchs).service(repo, capsCompletes()),
		celluleDemande(repo, domain.TacticalQuestionGagne, domain.TacticalQuiMoi, 4, 4))
	want := []attendu{
		{mode: "Assassin", score: "50 - 42", kind: "points", arme: "BR75 (fr)", armeEN: "BR75",
			placement: &domain.TacticalPlacement{Seul: true}, rejeu: true},
		{mode: "Assassin", score: "50 - 42", kind: "points", categorie: "Headshot", rejeu: true},
		{mode: "Oddball", score: "2 - 1", kind: "rounds", placement: &domain.TacticalPlacement{DistanceM: m(10)}},
		{mode: "Oddball", score: "2 - 1", kind: "rounds", placement: &domain.TacticalPlacement{Seul: true, DistanceM: m(30)}},
	}
	if len(got.Contributions) != len(want) {
		t.Fatalf("contributions = %+v, want %d", got.Contributions, len(want))
	}
	for i, w := range want {
		verifier(t, i, got.Contributions[i], w)
	}
}

// TestDetail_ModeDansLaLangueDeLaRequete : le mode suit la locale de la requête.
func TestDetail_ModeDansLaLangueDeLaRequete(t *testing.T) {
	repo, matchs := corpusDetail()
	svc := sourcesCompletes(matchs).service(repo, capsCompletes())
	got, err := svc.Cellule(ctxkeys.WithLocale(context.Background(), "en"),
		celluleDemande(repo, domain.TacticalQuestionMorts, domain.TacticalQuiMoi, 4, 4))
	if err != nil {
		t.Fatalf("Cellule: %v", err)
	}
	if len(got.Contributions) == 0 || got.Contributions[0].ModeLabel != "Slayer" {
		t.Fatalf("contributions = %+v, want le mode « Slayer » en anglais", got.Contributions)
	}
}

// TestDetail_UneLectureParSource : quatre contributions sur deux matchs, une lecture par source —
// canonique filtrée sur la carte, noms d'armes pour la seule clé classée, contextes bornés aux
// matchs des contributions, présence du rejeu.
func TestDetail_UneLectureParSource(t *testing.T) {
	repo, matchs := corpusDetail()
	src := sourcesCompletes(matchs)
	celluleLue(t, src.service(repo, capsCompletes()), celluleDemande(repo, domain.TacticalQuestionGagne, domain.TacticalQuiMoi, 4, 4))
	if matchs.appels != 1 || len(matchs.vu.MapIDs) != 1 || matchs.vu.MapIDs[0] != tsCarte {
		t.Errorf("lecture canonique : %d appel(s), filtre %v ; want 1 appel sur la carte %s", matchs.appels, matchs.vu.MapIDs, tsCarte)
	}
	if src.armes.appels != 1 || len(src.armes.cles) != 1 || src.armes.cles[0] != "br75" {
		t.Errorf("noms d'armes : %d appel(s), clés %v ; want 1 appel pour [br75]", src.armes.appels, src.armes.cles)
	}
	if len(repo.vuContextes) != 1 || strings.Join(repo.vuContextes[0].Matchs.IDs(), ",") != "m1,m2" {
		t.Errorf("contextes : %+v ; want 1 lecture bornée à [m1 m2]", repo.vuContextes)
	}
	if src.rejeu.calls != 1 {
		t.Errorf("présence du rejeu : %d listing(s), want 1", src.rejeu.calls)
	}
}

// TestDetail_OwnershipInchange : un match non ouvrable sort des contributions ET des lectures
// d'enrichissement (son compte au journal : TestCellule_Ownership_MatchEtrangerCompteSansApparaitre).
func TestDetail_OwnershipInchange(t *testing.T) {
	repo, matchs := corpusDetail()
	repo.ouvrables = map[string]time.Time{"m1": time.Date(2026, 9, 1, 13, 0, 0, 0, time.UTC)}
	got := celluleLue(t, sourcesCompletes(matchs).service(repo, capsCompletes()),
		celluleDemande(repo, domain.TacticalQuestionGagne, domain.TacticalQuiMoi, 4, 4))
	if len(got.Contributions) != 2 {
		t.Fatalf("contributions = %d ; want 2", len(got.Contributions))
	}
	if len(repo.vuContextes) != 1 || strings.Join(repo.vuContextes[0].Matchs.IDs(), ",") != "m1" {
		t.Errorf("contextes lus pour %+v, want [m1] seulement", repo.vuContextes)
	}
}

// TestDetail_TitreSansClassificateurNiCatalogue : ni arme, ni catégorie, ni nom de zone ; le
// registre des noms n'est pas interrogé ; les contributions restent servies.
func TestDetail_TitreSansClassificateurNiCatalogue(t *testing.T) {
	repo, matchs := corpusDetail()
	src := sourcesCompletes(matchs)
	src.class = nil
	got := celluleLue(t, src.service(repo, capsPositionsSeules()),
		celluleDemande(repo, domain.TacticalQuestionGagne, domain.TacticalQuiMoi, 4, 4))
	if len(got.Contributions) != 4 || got.Zone != nil {
		t.Fatalf("contributions = %d, zone = %+v ; want 4 et aucune", len(got.Contributions), got.Zone)
	}
	for i, c := range got.Contributions {
		if c.ArmeLabel != "" || c.ArmeLabelEN != "" || c.CategorieSource != "" {
			t.Errorf("contribution %d : arme %q / %q, catégorie %q ; want rien", i, c.ArmeLabel, c.ArmeLabelEN, c.CategorieSource)
		}
	}
	if src.armes.appels != 0 {
		t.Errorf("registre des noms interrogé %d fois sans classificateur", src.armes.appels)
	}
}

// TestDetail_ZoneSansArtefactDeRejeu : un titre sans artefact de rejeu (Halo 5, positions natives)
// ne nomme pas les zones — aucun appel au magasin de callouts, aucun WARN, la source dite absente
// au DEBUG ; les contributions restent servies.
func TestDetail_ZoneSansArtefactDeRejeu(t *testing.T) {
	repo, matchs := corpusDetail()
	src := sourcesCompletes(matchs)
	magasin := zonesEmpilees()
	halo5 := games.CapabilityMap{games.CapMatchEventsSpatial: games.CapSupported}
	got := celluleLue(t, src.service(repo, halo5).WithCalloutsStore(magasin),
		celluleDemande(repo, domain.TacticalQuestionGagne, domain.TacticalQuiMoi, 4, 4))
	if len(got.Contributions) != 4 || got.Zone != nil {
		t.Fatalf("contributions = %d, zone = %+v ; want 4 et aucune", len(got.Contributions), got.Zone)
	}
	if magasin.appels != 0 {
		t.Errorf("magasin de callouts appelé %d fois sans artefact de rejeu", magasin.appels)
	}
	if strings.Contains(src.trace.String(), "level=WARN") {
		t.Errorf("WARN sur un état nominal :\n%s", src.trace.String())
	}
	if !strings.Contains(src.trace.String(), "level=DEBUG msg=\"tactique: detail de zone, source zone absente\"") {
		t.Errorf("journal sans la source zone absente :\n%s", src.trace.String())
	}
}

// casDeSource : une source cassée, le champ qu'elle nourrit, le niveau de journal attendu.
type casDeSource struct {
	nom, source, niveau string
	casser              func(*sourcesDuDetailTest, *mockTacticalRepo)
	absent              func(domain.TacticalContribution) bool
}

func casDesSources() []casDeSource {
	panne := errors.New("panne de test")
	sansCanonique := func(c domain.TacticalContribution) bool {
		return c.ModeLabel == "" && c.ScoreLabel == "" && c.ScoreKind == ""
	}
	sansArme := func(c domain.TacticalContribution) bool { return c.ArmeLabel == "" && c.ArmeLabelEN == "" }
	sansPlacement := func(c domain.TacticalContribution) bool { return c.Placement == nil }
	sansRejeu := func(c domain.TacticalContribution) bool { return !c.ReplayAvailable }
	return []casDeSource{
		{"canonique non câblée", "canonique", "DEBUG", func(s *sourcesDuDetailTest, _ *mockTacticalRepo) { s.matchs = nil }, sansCanonique},
		{"canonique non supportée", "canonique", "DEBUG", func(s *sourcesDuDetailTest, _ *mockTacticalRepo) {
			s.matchs.err = games.ErrCapabilityNotSupported
		}, sansCanonique},
		{"canonique en échec", "canonique", "WARN", func(s *sourcesDuDetailTest, _ *mockTacticalRepo) { s.matchs.err = panne }, sansCanonique},
		{"classificateur absent", "armes", "DEBUG", func(s *sourcesDuDetailTest, _ *mockTacticalRepo) { s.class = nil }, sansArme},
		{"armes non câblées", "armes", "DEBUG", func(s *sourcesDuDetailTest, _ *mockTacticalRepo) { s.armes = nil }, sansArme},
		{"armes en échec", "armes", "WARN", func(s *sourcesDuDetailTest, _ *mockTacticalRepo) { s.armes.err = panne }, sansArme},
		{"contextes non supportés", "contextes", "DEBUG", func(_ *sourcesDuDetailTest, r *mockTacticalRepo) {
			r.errContextes = games.ErrCapabilityNotSupported
		}, sansPlacement},
		{"contextes en échec", "contextes", "WARN", func(_ *sourcesDuDetailTest, r *mockTacticalRepo) { r.errContextes = panne }, sansPlacement},
		{"rejeu non câblé", "rejeu", "DEBUG", func(s *sourcesDuDetailTest, _ *mockTacticalRepo) { s.rejeu = nil }, sansRejeu},
		{"rejeu en échec", "rejeu", "WARN", func(s *sourcesDuDetailTest, _ *mockTacticalRepo) { s.rejeu.err = panne }, sansRejeu},
	}
}

// TestDetail_SourceAbsenteOuEnEchec : chaque source peut manquer (non câblée, titre sans la donnée)
// ou échouer — le champ qu'elle nourrit disparaît, les contributions restent, et le journal le dit
// (DEBUG pour une absence, WARN pour un échec), en nommant la source.
func TestDetail_SourceAbsenteOuEnEchec(t *testing.T) {
	for _, tc := range casDesSources() {
		t.Run(tc.nom, func(t *testing.T) {
			repo, matchs := corpusDetail()
			src := sourcesCompletes(matchs)
			tc.casser(&src, repo)
			got := celluleLue(t, src.service(repo, capsCompletes()),
				celluleDemande(repo, domain.TacticalQuestionGagne, domain.TacticalQuiMoi, 4, 4))
			if len(got.Contributions) != 4 {
				t.Fatalf("contributions = %d, want 4 (la liste est toujours servie)", len(got.Contributions))
			}
			for i, c := range got.Contributions {
				if !tc.absent(c) {
					t.Errorf("contribution %d : le champ de la source %s est servi : %+v", i, tc.source, c)
				}
			}
			ligne := "level=" + tc.niveau + " msg=\"tactique: detail de zone, source " + tc.source
			if !strings.Contains(src.trace.String(), ligne) {
				t.Errorf("journal sans la ligne %q :\n%s", ligne, src.trace.String())
			}
		})
	}
}
