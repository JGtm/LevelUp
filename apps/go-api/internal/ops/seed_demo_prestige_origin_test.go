package ops

import (
	"context"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/prestige"
	"levelup/go-api/internal/testutil"
)

// testDemoCatalog charge le catalogue RÉEL du titre par défaut (config/titles versionné), par le
// chargeur que la génération de démo emploie.
func testDemoCatalog(t *testing.T) demoPrestigeCatalog {
	t.Helper()
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du dépôt: %v", err)
	}
	cat, err := loadDemoPrestigeCatalog(context.Background(), root, titlePkg.DefaultSlug)
	if err != nil {
		t.Fatalf("catalogue du titre %s: %v", titlePkg.DefaultSlug, err)
	}
	if cat.empty() {
		t.Fatalf("catalogue du titre %s vide", titlePkg.DefaultSlug)
	}
	return cat
}

// TestDemoPrestigePlan_OriginsServeEachLanguage : chaque objectif et chaque arc de la démo a une
// ORIGINE du catalogue réel, son texte stocké est la copie de départ du modèle / preset, et la
// résolution à la lecture (Template.ServedLabel, PresetArc.ServedTitle / ServedDescription : les
// fonctions que le service applique à chaque défi et chaque arc, localizeChallenges et
// localizeArcs) sert le texte de la langue demandée. Sans origine, une instance de démonstration en
// anglais affichait les libellés français tels que stockés.
func TestDemoPrestigePlan_OriginsServeEachLanguage(t *testing.T) {
	cat := testDemoCatalog(t)
	if gaps := demoCatalogGaps(cat); len(gaps) > 0 {
		t.Fatalf("origines attendues par la démo absentes du catalogue : %v", gaps)
	}
	st, matches := testDemoCorpus()
	plan := buildDemoPrestigePlan(titlePkg.DefaultSlug, cat, st, matches)

	if len(plan.Challenges) == 0 || len(plan.Arcs) == 0 {
		t.Fatalf("plan vide : %d objectifs, %d arcs", len(plan.Challenges), len(plan.Arcs))
	}
	for _, c := range plan.Challenges {
		tmpl, ok := cat.Templates[c.TemplateID]
		if c.TemplateID == "" || !ok {
			t.Errorf("objectif %s (%q) sans origine du catalogue (template_id=%q)", c.ID, c.Label, c.TemplateID)
			continue
		}
		if c.Label != tmpl.StoredLabel() {
			t.Errorf("objectif %s : libellé stocké %q, copie de départ du modèle %q", c.ID, c.Label, tmpl.StoredLabel())
		}
		if c.Metric != tmpl.Metric || c.Cadence != tmpl.Cadence || c.EvalType != tmpl.EvalType ||
			c.WindowType != tmpl.WindowType || c.WindowValue != tmpl.WindowValue {
			t.Errorf("objectif %s : mesure/fenêtre/cadence/évaluation différentes du modèle %s", c.ID, tmpl.ID)
		}
		for _, loc := range []i18n.Locale{i18n.LocaleEN, i18n.LocaleFR} {
			if got, want := tmpl.ServedLabel(c.Label, loc), tmpl.Labels.Pick(loc); got != want {
				t.Errorf("objectif %s en %s : servi %q, attendu %q", c.ID, loc, got, want)
			}
		}
	}
	for _, a := range plan.Arcs {
		preset, ok := cat.Presets[a.PresetID]
		if !a.IsPreset || a.PresetID == "" || !ok {
			t.Errorf("arc %s (%q) sans preset du catalogue (preset_id=%q)", a.ID, a.Title, a.PresetID)
			continue
		}
		if a.Title != preset.StoredTitle() || a.Description != preset.StoredDescription() {
			t.Errorf("arc %s : titre/description stockés ne sont pas la copie de départ du preset %s", a.ID, preset.ID)
		}
		for _, loc := range []i18n.Locale{i18n.LocaleEN, i18n.LocaleFR} {
			if got, want := preset.ServedTitle(a.Title, loc), preset.Titles.Pick(loc); got != want {
				t.Errorf("arc %s en %s : titre servi %q, attendu %q", a.ID, loc, got, want)
			}
			if got, want := preset.ServedDescription(a.Description, loc), preset.Descriptions.PickOptional(loc); got != want {
				t.Errorf("arc %s en %s : description servie %q, attendue %q", a.ID, loc, got, want)
			}
		}
	}
}

// TestDemoPrestigePlan_EnglishIsNotStoredFrench : preuve de morsure sur la langue — en anglais, au
// moins un objectif et un arc sont servis dans un texte DIFFÉRENT du texte stocké (le français).
func TestDemoPrestigePlan_EnglishIsNotStoredFrench(t *testing.T) {
	cat := testDemoCatalog(t)
	st, matches := testDemoCorpus()
	plan := buildDemoPrestigePlan(titlePkg.DefaultSlug, cat, st, matches)

	translated := 0
	for _, c := range plan.Challenges {
		if tmpl, ok := cat.Templates[c.TemplateID]; ok && tmpl.ServedLabel(c.Label, i18n.LocaleEN) != c.Label {
			translated++
		}
	}
	if translated == 0 {
		t.Error("aucun objectif de démo servi en anglais autrement que par son texte stocké")
	}
	arcsTranslated := 0
	for _, a := range plan.Arcs {
		if p, ok := cat.Presets[a.PresetID]; ok && p.ServedTitle(a.Title, i18n.LocaleEN) != a.Title {
			arcsTranslated++
		}
	}
	if arcsTranslated != len(plan.Arcs) {
		t.Errorf("%d arc(s) sur %d servi(s) en anglais autrement que par leur texte stocké",
			arcsTranslated, len(plan.Arcs))
	}
}

// TestDemoPrestigePlan_NoCatalogFallsBackWithoutPhantomOrigin : un titre sans catalogue (Halo 5)
// écrit le même cycle de vie sans origine — ni template_id, ni preset_id inventé (l'ancien preset
// "ascension" n'existait pas) — et le plan reste cohérent.
func TestDemoPrestigePlan_NoCatalogFallsBackWithoutPhantomOrigin(t *testing.T) {
	st, matches := testDemoCorpus()
	plan := buildDemoPrestigePlan("halo_5", demoPrestigeCatalog{Slug: "halo_5"}, st, matches)

	seen := map[prestige.ChallengeStatus]int{}
	for _, c := range plan.Challenges {
		if c.TemplateID != "" {
			t.Errorf("objectif %s : template_id %q sans catalogue", c.ID, c.TemplateID)
		}
		if c.Label == "" || c.Target <= 0 {
			t.Errorf("objectif %s : libellé %q / cible %v inutilisables", c.ID, c.Label, c.Target)
		}
		seen[c.Status]++
	}
	for _, st := range []prestige.ChallengeStatus{
		prestige.StatusActive, prestige.StatusCompleted, prestige.StatusExpired, prestige.StatusAbandoned,
	} {
		if seen[st] == 0 {
			t.Errorf("repli : aucun objectif au statut %s", st)
		}
	}
	for _, a := range plan.Arcs {
		if a.IsPreset || a.PresetID != "" || a.Title == "" {
			t.Errorf("arc %s : repli attendu sans preset (is_preset=%v preset_id=%q title=%q)",
				a.ID, a.IsPreset, a.PresetID, a.Title)
		}
	}
}

// TestLoadDemoPrestigeCatalog_TitleWithoutCatalog : Halo 5 n'a pas de fichier de catalogue de
// défis — catalogue vide, sans erreur ; une racine sans dossier de titre est, elle, une erreur.
func TestLoadDemoPrestigeCatalog_TitleWithoutCatalog(t *testing.T) {
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du dépôt: %v", err)
	}
	cat, err := loadDemoPrestigeCatalog(context.Background(), root, "halo_5")
	if err != nil || !cat.empty() {
		t.Errorf("halo_5 : catalogue vide sans erreur attendu, obtenu empty=%v err=%v", cat.empty(), err)
	}
	if _, err := loadDemoPrestigeCatalog(context.Background(), t.TempDir(), titlePkg.DefaultSlug); err == nil {
		t.Error("racine sans config/titles : erreur attendue")
	}
}
