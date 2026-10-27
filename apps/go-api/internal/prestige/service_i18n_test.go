package prestige

import (
	"context"
	"testing"

	"levelup/go-api/internal/i18n"
)

// Tests du contrat i18n du module Prestige (ADR 0040 D-3, plan i18n étape 6, lot B) : le service rend
// libellé et description d'UNE langue, celle du contexte de la requête (posée par le middleware depuis
// X-LevelUp-Locale), plus de champs `label_fr` / `label_en`. Une langue absente du contexte (ou
// inconnue du middleware) donne la langue par défaut ; un texte sans traduction se replie sur `en`.

func ctxLangue(l i18n.Locale) context.Context {
	return i18n.WithContext(context.Background(), l)
}

func templateDeuxLangues(id string) Template {
	return Template{
		ID:           id,
		Cadence:      CadenceWeekly,
		Labels:       i18n.FromColumns("Stay sharp", "Reste affûté"),
		Descriptions: i18n.FromColumns("Keep your aim", "Garde ta visée"),
	}
}

func TestService_SuggestTemplates_LibelleParLangue(t *testing.T) {
	svc, _, _, _, _, tplRepo := buildFullService()
	tplRepo.templates = []Template{templateDeuxLangues("t1")}

	for _, c := range []struct {
		nom        string
		ctx        context.Context
		label, des string
	}{
		{"fr", ctxLangue(i18n.LocaleFR), "Reste affûté", "Garde ta visée"},
		{"en", ctxLangue(i18n.LocaleEN), "Stay sharp", "Keep your aim"},
		// Aucune langue dans le contexte (tâche de fond) : langue par défaut.
		{"défaut", context.Background(), "Reste affûté", "Garde ta visée"},
	} {
		out, err := svc.SuggestTemplates(c.ctx, "u1", "halo_infinite", 3)
		if err != nil || len(out) != 1 {
			t.Fatalf("%s : err=%v len=%d", c.nom, err, len(out))
		}
		if out[0].Label != c.label || out[0].Description != c.des {
			t.Errorf("%s : label=%q description=%q, attendu %q / %q", c.nom, out[0].Label, out[0].Description, c.label, c.des)
		}
	}
	// Le dépôt n'est pas muté : une autre requête dans une autre langue repart des textes par langue.
	if tplRepo.templates[0].Label != "" {
		t.Errorf("la tranche du dépôt a été localisée en place : %q", tplRepo.templates[0].Label)
	}
}

func TestService_SuggestTemplates_RepliSurEnglish(t *testing.T) {
	svc, _, _, _, _, tplRepo := buildFullService()
	tplRepo.templates = []Template{{ID: "t1", Cadence: CadenceWeekly, Labels: i18n.FromColumns("Only English", "")}}
	out, err := svc.SuggestTemplates(ctxLangue(i18n.LocaleFR), "u1", "halo_infinite", 3)
	if err != nil || len(out) != 1 {
		t.Fatalf("err=%v len=%d", err, len(out))
	}
	if out[0].Label != "Only English" || out[0].Description != "" {
		t.Errorf("repli en attendu, description vide : %+v", out[0])
	}
}

func TestService_SuggestNext_LibelleParLangue(t *testing.T) {
	svc, chRepo, _, _, _, _ := buildCoverageService()
	chRepo.stored["ch1"] = Challenge{ID: "ch1", TitleSlug: "halo_infinite", TemplateID: "t1"}
	svc.deps.Templates.(*fakeTemplateRepo).templates = []Template{templateDeuxLangues("t2")}

	fr, err := svc.SuggestNext(ctxLangue(i18n.LocaleFR), "ch1")
	if err != nil || len(fr) != 1 {
		t.Fatalf("fr : err=%v len=%d", err, len(fr))
	}
	en, err := svc.SuggestNext(ctxLangue(i18n.LocaleEN), "ch1")
	if err != nil || len(en) != 1 {
		t.Fatalf("en : err=%v len=%d", err, len(en))
	}
	if fr[0].Label != "Reste affûté" || en[0].Label != "Stay sharp" {
		t.Errorf("fr=%q en=%q", fr[0].Label, en[0].Label)
	}
}

func TestService_RefreshSquadPool_LibelleParLangue(t *testing.T) {
	svc, _, _, _, sqRepo, tplRepo := buildFullService()
	sqRepo.members = []SquadMember{{UserID: "u1"}}
	tplRepo.templates = []Template{templateDeuxLangues("t1"), templateDeuxLangues("t2"), templateDeuxLangues("t3"),
		templateDeuxLangues("t4"), templateDeuxLangues("t5"), templateDeuxLangues("t6")}

	pool, err := svc.RefreshSquadPool(ctxLangue(i18n.LocaleEN), "sq1", "halo_infinite", "u1")
	if err != nil || len(pool) == 0 {
		t.Fatalf("err=%v len=%d", err, len(pool))
	}
	for _, tpl := range pool {
		if tpl.Label != "Stay sharp" {
			t.Errorf("template %s : label=%q, attendu Stay sharp", tpl.ID, tpl.Label)
		}
	}
}

func TestService_EnablePilotMode_ChoixHebdomadairesParLangue(t *testing.T) {
	svc, chRepo, _, _, _, tplRepo := buildFullService()
	tplRepo.templates = []Template{
		{ID: "t1", Cadence: CadenceDaily, WindowType: WindowSession, EvalType: EvalThreshold, Metric: "FieldKDA",
			Labels: i18n.FromColumns("Daily", "Quotidien"), NormalTarget: 1.1, HeroicTarget: 1.35, LegendaryTarget: 1.6, MythicTarget: 2},
		templateDeuxLangues("t2"),
	}
	out, err := svc.EnablePilotMode(ctxLangue(i18n.LocaleEN), "u1", "halo_infinite")
	if err != nil {
		t.Fatal(err)
	}
	if len(out.WeeklyChoices) != 1 || out.WeeklyChoices[0].Label != "Stay sharp" {
		t.Errorf("choix hebdomadaires : %+v", out.WeeklyChoices)
	}
	// Le défi créé par le mode pilote garde un libellé STOCKÉ dans la langue par défaut, quelle que soit
	// la langue de la requête ; la réponse le sert dans la langue de la requête (service_localize.go,
	// ADR 0040 D-3) tant qu'on le retrouve par son modèle.
	if out.Daily == nil || out.Daily.Label != "Daily" {
		t.Errorf("libellé servi du défi quotidien : %+v", out.Daily)
	}
	if len(chRepo.createdChallenges) == 0 || chRepo.createdChallenges[0].Label != "Quotidien" {
		t.Errorf("libellé stocké du défi quotidien : %+v", chRepo.createdChallenges)
	}
}

func TestService_ListArcPresets_TitreParLangue(t *testing.T) {
	preset := &stubPresetArcRepo{
		listResult: []PresetArc{{ID: "p1", TitleSlug: "halo_infinite",
			Titles: i18n.FromColumns("Spartan Ascension", "Ascension du Spartan"), Descriptions: i18n.FromColumns("", "Seulement en français")}},
	}
	svc, _, _, _ := buildPresetService(preset)

	fr, err := svc.ListArcPresets(ctxLangue(i18n.LocaleFR), "u1", "halo_infinite")
	if err != nil || len(fr) != 1 {
		t.Fatalf("fr : err=%v len=%d", err, len(fr))
	}
	if fr[0].Title != "Ascension du Spartan" || fr[0].Description != "Seulement en français" {
		t.Errorf("fr : %+v", fr[0])
	}
	// Liste relue : `listResult` est servi tel quel par le dépôt factice ; on repart des textes par langue.
	preset.listResult = []PresetArc{{ID: "p1", TitleSlug: "halo_infinite",
		Titles: i18n.FromColumns("Spartan Ascension", "Ascension du Spartan"), Descriptions: i18n.FromColumns("", "Seulement en français")}}
	en, err := svc.ListArcPresets(ctxLangue(i18n.LocaleEN), "u1", "halo_infinite")
	if err != nil || len(en) != 1 {
		t.Fatalf("en : err=%v len=%d", err, len(en))
	}
	// Description sans traduction anglaise : « en » est le repli, vide ici : la description reste vide.
	if en[0].Title != "Spartan Ascension" || en[0].Description != "" {
		t.Errorf("en : %+v", en[0])
	}
}
