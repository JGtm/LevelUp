package prestige

import (
	"context"
	"testing"

	"levelup/go-api/internal/i18n"
)

// Tests de la résolution à la lecture des textes stockés en une langue (Challenge.Label,
// Arc.Title, Arc.Description) — service_localize.go. Le stockage reste en langue par défaut ;
// l'origine (TemplateID, PresetID) retrouve le texte de la langue de la requête tant que le texte
// stocké est la copie exacte de l'origine.

func TestService_ListChallenges_LibelleIssuDuModele(t *testing.T) {
	tmpl := templateDeuxLangues("t1") // StoredLabel() = « Reste affûté »
	cases := []struct {
		nom    string
		ctx    context.Context
		stored Challenge
		want   string
	}{
		{"modele, fr", ctxLangue(i18n.LocaleFR), Challenge{TemplateID: "t1", Label: tmpl.StoredLabel()}, "Reste affûté"},
		{"modele, en", ctxLangue(i18n.LocaleEN), Challenge{TemplateID: "t1", Label: tmpl.StoredLabel()}, "Stay sharp"},
		{"modele, langue inconnue: langue par defaut", ctxLangue(i18n.Locale("zz")), Challenge{TemplateID: "t1", Label: tmpl.StoredLabel()}, "Reste affûté"},
		{"modele, aucune langue dans le contexte", context.Background(), Challenge{TemplateID: "t1", Label: tmpl.StoredLabel()}, "Reste affûté"},
		{"libelle modifie par le joueur: jamais remplace", ctxLangue(i18n.LocaleEN), Challenge{TemplateID: "t1", Label: "Mon objectif"}, "Mon objectif"},
		{"defi libre: jamais remplace", ctxLangue(i18n.LocaleEN), Challenge{Label: tmpl.StoredLabel()}, tmpl.StoredLabel()},
		{"modele introuvable: libelle stocke", ctxLangue(i18n.LocaleEN), Challenge{TemplateID: "absent", Label: tmpl.StoredLabel()}, tmpl.StoredLabel()},
	}
	for _, c := range cases {
		t.Run(c.nom, func(t *testing.T) {
			svc, chRepo, _, _, _, tplRepo := buildFullService()
			tplRepo.templates = []Template{tmpl}
			c.stored.ID, c.stored.Status = "ch1", StatusCompleted
			chRepo.listResult = []Challenge{c.stored}
			out, err := svc.ListChallenges(c.ctx, "u1", "halo_infinite", []ChallengeStatus{StatusCompleted})
			if err != nil || len(out) != 1 {
				t.Fatalf("err=%v len=%d", err, len(out))
			}
			if out[0].Label != c.want {
				t.Errorf("label = %q, attendu %q", out[0].Label, c.want)
			}
		})
	}
}

func TestService_ListChallenges_ModeleSansTraduction_RepliSurEnglish(t *testing.T) {
	// Un modele qui n'a que l'anglais : StoredLabel() = Pick(Default) replie deja sur `en`.
	tmpl := Template{ID: "t1", Labels: i18n.FromColumns("Only English", "")}
	svc, chRepo, _, _, _, tplRepo := buildFullService()
	tplRepo.templates = []Template{tmpl}
	chRepo.listResult = []Challenge{{ID: "ch1", Status: StatusCompleted, TemplateID: "t1", Label: tmpl.StoredLabel()}}
	for _, l := range []i18n.Locale{i18n.LocaleFR, i18n.LocaleEN} {
		out, err := svc.ListChallenges(ctxLangue(l), "u1", "halo_infinite", []ChallengeStatus{StatusCompleted})
		if err != nil || len(out) != 1 || out[0].Label != "Only English" {
			t.Fatalf("%s: err=%v out=%+v", l, err, out)
		}
	}
}

func TestService_GetArc_TitreIssuDuPreset(t *testing.T) {
	preset := PresetArc{
		ID: "p1", TitleSlug: "halo_infinite",
		Titles:       i18n.FromColumns("Ascension", "Ascension FR"),
		Descriptions: i18n.FromColumns("Climb the ranks", "Grimpe les rangs"),
	}
	cases := []struct {
		nom              string
		ctx              context.Context
		arc              Arc
		wantTitle, wantD string
	}{
		{"preset, fr", ctxLangue(i18n.LocaleFR), Arc{IsPreset: true, PresetID: "p1", Title: preset.StoredTitle(), Description: preset.StoredDescription()}, "Ascension FR", "Grimpe les rangs"},
		{"preset, en", ctxLangue(i18n.LocaleEN), Arc{IsPreset: true, PresetID: "p1", Title: preset.StoredTitle(), Description: preset.StoredDescription()}, "Ascension", "Climb the ranks"},
		{"preset, langue inconnue", ctxLangue(i18n.Locale("zz")), Arc{IsPreset: true, PresetID: "p1", Title: preset.StoredTitle(), Description: preset.StoredDescription()}, "Ascension FR", "Grimpe les rangs"},
		{"titre edite: jamais remplace, description copiee localisee", ctxLangue(i18n.LocaleEN), Arc{IsPreset: true, PresetID: "p1", Title: "Mon arc", Description: preset.StoredDescription()}, "Mon arc", "Climb the ranks"},
		{"arc libre: jamais remplace", ctxLangue(i18n.LocaleEN), Arc{Title: preset.StoredTitle(), Description: preset.StoredDescription()}, preset.StoredTitle(), preset.StoredDescription()},
		{"preset introuvable: texte stocke", ctxLangue(i18n.LocaleEN), Arc{IsPreset: true, PresetID: "absent", Title: preset.StoredTitle()}, preset.StoredTitle(), ""},
	}
	for _, c := range cases {
		t.Run(c.nom, func(t *testing.T) {
			svc, _, arcRepo, _ := buildPresetService(&stubPresetArcRepo{byID: map[string]PresetArc{"p1": preset}})
			c.arc.ID = "a1"
			arcRepo.getResp = c.arc
			got, err := svc.GetArc(c.ctx, "a1")
			if err != nil {
				t.Fatal(err)
			}
			if got.Title != c.wantTitle || got.Description != c.wantD {
				t.Errorf("title=%q description=%q, attendu %q / %q", got.Title, got.Description, c.wantTitle, c.wantD)
			}
		})
	}
}

func TestService_AdoptPresetArc_ServiEnLangueDeLaRequete(t *testing.T) {
	preset := PresetArc{
		ID: "p1", TitleSlug: "halo_infinite",
		Titles: i18n.FromColumns("Ascension", "Ascension FR"),
	}
	svc, _, arcRepo, _ := buildPresetService(&stubPresetArcRepo{byID: map[string]PresetArc{"p1": preset}})
	arc, err := svc.AdoptPresetArc(ctxLangue(i18n.LocaleEN), "u1", "halo_infinite", "p1")
	if err != nil {
		t.Fatal(err)
	}
	if arc.Title != "Ascension" {
		t.Errorf("reponse en anglais attendue : %q", arc.Title)
	}
	// Le stockage reste en langue par defaut.
	if len(arcRepo.created) != 1 || arcRepo.created[0].Title != "Ascension FR" {
		t.Errorf("arc stocke : %+v", arcRepo.created)
	}
}

// Un défi créé depuis le formulaire web porte le `label` du modèle dans la langue de la requête de
// création (le formulaire poste le libellé localisé qu'il a reçu), donc parfois une AUTRE langue que
// la langue par défaut : c'est encore une copie du modèle, servie dans la langue de la requête.
func TestService_ListChallenges_CopieDansUneAutreLangueDuModele(t *testing.T) {
	tmpl := templateDeuxLangues("t1") // fr « Reste affûté », en « Stay sharp »
	cases := []struct {
		nom    string
		ctx    context.Context
		stored string
		want   string
	}{
		{"cree en en, lu en fr", ctxLangue(i18n.LocaleFR), "Stay sharp", "Reste affûté"},
		{"cree en en, lu en en", ctxLangue(i18n.LocaleEN), "Stay sharp", "Stay sharp"},
		{"cree en fr, lu en en", ctxLangue(i18n.LocaleEN), "Reste affûté", "Stay sharp"},
		{"texte edite: jamais remplace, meme en langue par defaut", ctxLangue(i18n.LocaleFR), "Stay sharp !", "Stay sharp !"},
	}
	for _, c := range cases {
		t.Run(c.nom, func(t *testing.T) {
			svc, chRepo, _, _, _, tplRepo := buildFullService()
			tplRepo.templates = []Template{tmpl}
			chRepo.listResult = []Challenge{{ID: "ch1", Status: StatusCompleted, TemplateID: "t1", Label: c.stored}}
			out, err := svc.ListChallenges(c.ctx, "u1", "halo_infinite", []ChallengeStatus{StatusCompleted})
			if err != nil || len(out) != 1 {
				t.Fatalf("err=%v len=%d", err, len(out))
			}
			if out[0].Label != c.want {
				t.Errorf("label = %q, attendu %q", out[0].Label, c.want)
			}
		})
	}
}

// Sans origine enregistrée (défi libre : pas de TemplateID ; arc dont le PresetID est absent du
// catalogue), le texte stocké est servi tel quel dans toutes les langues : c'est le cas des défis et
// de l'arc de la démo (« Précision létale — tirs à la tête », « Ascension du Spartan »), écrits en
// français par le générateur de démo avec un PresetID (« ascension ») que le catalogue ne connaît pas.
func TestService_SansOrigineEnregistree_TexteStockeServiTelQuel(t *testing.T) {
	tmpl := templateDeuxLangues("t1")
	svc, chRepo, _, _, _, tplRepo := buildFullService()
	tplRepo.templates = []Template{tmpl}
	chRepo.listResult = []Challenge{{ID: "ch1", Status: StatusActive, Label: "Précision létale — tirs à la tête"}}
	for _, l := range []i18n.Locale{i18n.LocaleFR, i18n.LocaleEN} {
		out, err := svc.ListChallenges(ctxLangue(l), "u1", "halo_infinite", []ChallengeStatus{StatusActive})
		if err != nil || len(out) != 1 || out[0].Label != "Précision létale — tirs à la tête" {
			t.Fatalf("%s: err=%v out=%+v", l, err, out)
		}
	}

	preset := PresetArc{ID: "halo_infinite.slayer", TitleSlug: "halo_infinite", Titles: i18n.FromColumns("The Slayer", "Le Slayer")}
	asvc, _, arcRepo, _ := buildPresetService(&stubPresetArcRepo{byID: map[string]PresetArc{preset.ID: preset}})
	arcRepo.getResp = Arc{ID: "a1", IsPreset: true, PresetID: "ascension", Title: "Ascension du Spartan", Description: "Enchaîne les objectifs."}
	got, err := asvc.GetArc(ctxLangue(i18n.LocaleEN), "a1")
	if err != nil || got.Title != "Ascension du Spartan" || got.Description != "Enchaîne les objectifs." {
		t.Fatalf("arc sans preset connu : err=%v arc=%+v", err, got)
	}
}

func TestService_GetArc_CopieDansUneAutreLangueDuPreset(t *testing.T) {
	preset := PresetArc{
		ID: "p1", TitleSlug: "halo_infinite",
		Titles:       i18n.FromColumns("Ascension", "Ascension FR"),
		Descriptions: i18n.FromColumns("Climb the ranks", "Grimpe les rangs"),
	}
	for l, want := range map[i18n.Locale][2]string{
		i18n.LocaleFR: {"Ascension FR", "Grimpe les rangs"},
		i18n.LocaleEN: {"Ascension", "Climb the ranks"},
	} {
		svc, _, arcRepo, _ := buildPresetService(&stubPresetArcRepo{byID: map[string]PresetArc{"p1": preset}})
		arcRepo.getResp = Arc{ID: "a1", IsPreset: true, PresetID: "p1", Title: "Ascension", Description: "Climb the ranks"}
		got, err := svc.GetArc(ctxLangue(l), "a1")
		if err != nil || got.Title != want[0] || got.Description != want[1] {
			t.Errorf("%s: err=%v title=%q description=%q, attendu %q / %q", l, err, got.Title, got.Description, want[0], want[1])
		}
	}
}
