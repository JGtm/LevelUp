package native

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/platform/settings"
)

type fakeStore struct {
	cfg *settings.AppSettings
	err error
}

func (f fakeStore) Load() (*settings.AppSettings, error) { return f.cfg, f.err }

func settingsWith(native bool, lang string) *settings.AppSettings {
	cfg := settings.Defaults()
	cfg.NativeNotifications = native
	cfg.Lang = lang
	return cfg
}

// collect enregistre les notices reçues.
type collect struct{ got []Notice }

func (c *collect) sink(n Notice) { c.got = append(c.got, n) }

func notif(c notifications.Category) *notifications.Notification {
	return &notifications.Notification{ID: 1, Category: c, TitleKey: "notif.x.title"}
}

// TestRelay_ReglageActifRelaieDansLaLangueDuJoueur : un type relayé, réglage actif, sort composé dans la
// langue du joueur ; le titre et le corps viennent du catalogue.
func TestRelay_ReglageActifRelaieDansLaLangueDuJoueur(t *testing.T) {
	for _, lang := range []string{"fr", "en"} {
		r := &Relay{}
		c := &collect{}
		r.Install(c.sink, fakeStore{cfg: settingsWith(true, lang)})
		r.Forward(context.Background(), notif(notifications.CategoryMatchSynced))
		if len(c.got) != 1 {
			t.Fatalf("[%s] %d notices, attendu 1", lang, len(c.got))
		}
		wantTitle, wantBody := Compose(i18n.Parse(lang), KindSync)
		n := c.got[0]
		if n.Kind != KindSync || n.Title != wantTitle || n.Body != wantBody || n.Title == "" || n.Body == "" {
			t.Errorf("[%s] notice = %+v, attendu %q / %q", lang, n, wantTitle, wantBody)
		}
	}
	frT, _ := Compose(i18n.Parse("fr"), KindSync)
	enT, _ := Compose(i18n.Parse("en"), KindSync)
	if frT == enT {
		t.Fatalf("le texte doit suivre la langue : %q == %q", frT, enT)
	}
}

// TestRelay_ReglageCoupeOuNonBranche : par défaut (réglage coupé) ou sans sink, rien ne sort ; le
// réglage coupé est compté.
func TestRelay_ReglageCoupeOuNonBranche(t *testing.T) {
	c := &collect{}
	r := &Relay{}
	r.Forward(context.Background(), notif(notifications.CategoryMatchSynced)) // pas branché
	r.Install(c.sink, fakeStore{cfg: settingsWith(false, "fr")})
	before := observability.LoadCounter(MetricSkipped)
	r.Forward(context.Background(), notif(notifications.CategoryMatchSynced))
	if len(c.got) != 0 {
		t.Fatalf("réglage coupé : %d notices relayées", len(c.got))
	}
	if observability.LoadCounter(MetricSkipped)-before != 1 {
		t.Error("une notification retenue par le réglage coupé doit être comptée")
	}
	r.Uninstall()
	r.Forward(context.Background(), notif(notifications.CategoryMatchSynced))
	if len(c.got) != 0 {
		t.Fatal("relais débranché : rien ne sort")
	}
}

// TestRelay_CategorieNonRelayeeEtReglagesIllisibles : un conseil du coach ne dérange pas Windows ;
// des réglages illisibles ne relaient rien (et ne paniquent pas).
func TestRelay_CategorieNonRelayeeEtReglagesIllisibles(t *testing.T) {
	c := &collect{}
	r := &Relay{}
	r.Install(c.sink, fakeStore{cfg: settingsWith(true, "fr")})
	r.Forward(context.Background(), notif(notifications.CategoryComebackWelcome))
	r.Forward(context.Background(), notif(notifications.CategoryMediaLiked))
	if len(c.got) != 0 {
		t.Fatalf("catégories non relayées : %v", c.got)
	}
	r.Install(c.sink, fakeStore{err: errors.New("illisible")})
	r.Forward(context.Background(), notif(notifications.CategoryMatchSynced))
	if len(c.got) != 0 {
		t.Fatalf("réglages illisibles : %v", c.got)
	}
	r.Forward(context.Background(), nil) // une notification nil ne panique pas
}

// TestKinds_ChaqueTypeEstComposeDansChaqueLangue : un type relayé a un titre et un corps non vides dans
// toutes les langues du catalogue (une langue est complète ou refusée).
func TestKinds_ChaqueTypeEstComposeDansChaqueLangue(t *testing.T) {
	for _, loc := range i18n.Locales() {
		for _, k := range Kinds() {
			title, body := Compose(loc, k)
			if title == "" || body == "" {
				t.Errorf("%s/%s : titre %q, corps %q", loc, k, title, body)
			}
		}
	}
	// Toute catégorie relayée mène à un type composé.
	for cat, k := range kindByCategory {
		if title, _ := Compose(i18n.Default, k); title == "" {
			t.Errorf("catégorie %s : type %s sans texte", cat, k)
		}
	}
}

// TestForPlayer_SeulLeJoueurLocalEstRelaye : le service de chaque profil suivi se branche sur le relais ;
// un ami suivi qui change de palier ne produit pas de notification Windows « ton rang vient de changer ».
func TestForPlayer_SeulLeJoueurLocalEstRelaye(t *testing.T) {
	r := &Relay{}
	c := &collect{}
	r.Install(c.sink, fakeStore{cfg: settingsWith(true, "fr")})
	local := map[string]bool{"2533274001": true}
	r.InstallOwnerFilter(func(xuid string) bool { return local[xuid] })
	ctx := context.Background()
	before := observability.LoadCounter(MetricForeign)

	r.ForPlayer("2533274999").Forward(ctx, notif(notifications.CategoryMatchSynced)) // un ami suivi
	if len(c.got) != 0 {
		t.Fatalf("un profil suivi non local ne produit aucune notification Windows : %d", len(c.got))
	}
	if observability.LoadCounter(MetricForeign)-before != 1 {
		t.Error("une notification d'un autre joueur doit être comptée")
	}
	r.ForPlayer("2533274001").Forward(ctx, notif(notifications.CategoryMatchSynced))
	if len(c.got) != 1 {
		t.Fatalf("le joueur local est relayé : %d notices, attendu 1", len(c.got))
	}
	// Une connexion faite après le démarrage prend effet sans redémarrage : le filtre se relit.
	local["2533274999"] = true
	r.ForPlayer("2533274999").Forward(ctx, notif(notifications.CategoryMatchSynced))
	if len(c.got) != 2 {
		t.Fatalf("filtre relu à chaque notification : %d notices, attendu 2", len(c.got))
	}
}

// TestForPlayer_SansFiltreRienNEstRelaye : le défaut est sûr, aucun joueur n'est local tant que le boot n'a
// pas installé le filtre.
func TestForPlayer_SansFiltreRienNEstRelaye(t *testing.T) {
	r := &Relay{}
	c := &collect{}
	r.Install(c.sink, fakeStore{cfg: settingsWith(true, "fr")})
	r.ForPlayer("2533274001").Forward(context.Background(), notif(notifications.CategoryMatchSynced))
	if len(c.got) != 0 {
		t.Fatalf("sans filtre : %d notices relayées", len(c.got))
	}
	r.ForPlayer("2533274001").Forward(context.Background(), nil)
}
