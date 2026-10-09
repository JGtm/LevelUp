package playerdirectory

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
)

// ─── Doubles d'écriture ──────────────────────────────────────────────────────

// fakeCreator : le créateur de profil. Il note le gamertag créé dans `créés`,
// ce qui sert au test d'ORDRE (le watcher n'ouvre sa porte que si le profil y
// figure déjà) — la contrainte que la porte profil du daemon impose réellement.
type fakeCreator struct {
	key      string
	warnings []string
	err      error
	calls    int
	lastReq  domain.CreatePlayerProfileRequest
	created  map[string]bool
}

func newFakeCreator(key string) *fakeCreator {
	return &fakeCreator{key: key, created: map[string]bool{}}
}

func (f *fakeCreator) CreatePlayer(req domain.CreatePlayerProfileRequest) (string, []string, error) {
	f.calls++
	f.lastReq = req
	if f.err != nil {
		return "", nil, f.err
	}
	f.created[req.TitleSlug+"/"+req.Gamertag] = true
	return f.key, f.warnings, nil
}

// fakeWatcher : le daemon. `gate` reproduit la porte « profil suivi » de
// watcher.Daemon.AddPlayer (ADR 0035 D3) : elle refuse tant que le profil
// n'existe pas.
type fakeWatcher struct {
	running bool
	err     error
	gate    func(p domain.PlayerSummary) bool
	calls   int
	last    domain.PlayerSummary
}

func (f *fakeWatcher) IsRunning() bool { return f.running }

func (f *fakeWatcher) AddPlayer(_ context.Context, p domain.PlayerSummary) error {
	f.calls++
	f.last = p
	if f.gate != nil && !f.gate(p) {
		return errors.New("watcher_daemon: joueur sans profil suivi")
	}
	return f.err
}

func onboardRequest() domain.OnboardRequest {
	return domain.OnboardRequest{
		TitleSlug:         testTitle,
		Gamertag:          "Spartan",
		XUID:              "111",
		InitialMaxMatches: 42,
		ActorUsername:     "admin",
	}
}

// ─── Tests ───────────────────────────────────────────────────────────────────

// Le chemin nominal : profil créé, watcher notifié, résultat complet.
func TestOnboard_ProfilCreeEtWatcherNotifie(t *testing.T) {
	creator := newFakeCreator("Spartan")
	creator.warnings = []string{"dossier deja present"}
	watcher := &fakeWatcher{running: true}
	d := newTestDirectory(t, Deps{
		Profiles: &fakeProfiles{},
		FS:       &fakeFS{dbs: map[string]bool{testTitle + "/Spartan": true}},
		Creator:  creator,
		Watcher:  watcher,
	})

	res, err := d.Onboard(context.Background(), onboardRequest())
	if err != nil {
		t.Fatalf("Onboard: %v", err)
	}
	if res.PlayerKey != "Spartan" {
		t.Errorf("PlayerKey = %q, attendu \"Spartan\"", res.PlayerKey)
	}
	if !res.WatcherNotified {
		t.Error("WatcherNotified devrait etre vrai quand le daemon tourne")
	}
	if !res.DBCreated {
		t.Error("DBCreated devrait refleter le temoin disque")
	}
	if res.DBPath != testTitle+"/Spartan/stats.duckdb" {
		t.Errorf("DBPath = %q", res.DBPath)
	}
	if len(res.Warnings) != 1 {
		t.Errorf("les warnings du createur doivent remonter, recu %v", res.Warnings)
	}
	if creator.lastReq.InitialMaxMatches != 42 || creator.lastReq.XUID != "111" {
		t.Errorf("la demande n'a pas ete transmise telle quelle : %+v", creator.lastReq)
	}
	if watcher.last.SyncEnabled != true || watcher.last.TitleSlug != testTitle {
		t.Errorf("le joueur passe au watcher est incomplet : %+v", watcher.last)
	}
}

// L'ORDRE est la raison d'être de ce chemin unique : la porte profil du daemon
// lit db_profiles.json, donc notifier avant d'avoir créé le profil se solderait
// par un refus. Ce test échoue si l'ordre s'inverse un jour.
func TestOnboard_ProfilAvantWatcher(t *testing.T) {
	creator := newFakeCreator("Spartan")
	watcher := &fakeWatcher{running: true}
	watcher.gate = func(p domain.PlayerSummary) bool {
		return creator.created[p.TitleSlug+"/"+p.Gamertag]
	}
	d := newTestDirectory(t, Deps{Profiles: &fakeProfiles{}, Creator: creator, Watcher: watcher})

	res, err := d.Onboard(context.Background(), onboardRequest())
	if err != nil {
		t.Fatalf("Onboard: %v", err)
	}
	if !res.WatcherNotified {
		t.Fatal("la porte profil a refuse : le watcher a ete notifie AVANT la creation du profil")
	}
}

// Daemon arrêté : pas de notification, pas d'erreur — `initPlayers` reprendra le
// joueur au prochain démarrage.
func TestOnboard_DaemonArrete(t *testing.T) {
	creator := newFakeCreator("Spartan")
	watcher := &fakeWatcher{running: false}
	d := newTestDirectory(t, Deps{Profiles: &fakeProfiles{}, Creator: creator, Watcher: watcher})

	res, err := d.Onboard(context.Background(), onboardRequest())
	if err != nil {
		t.Fatalf("un daemon arrete n'est pas une erreur de mise en place : %v", err)
	}
	if res.WatcherNotified {
		t.Error("WatcherNotified devrait etre faux")
	}
	if watcher.calls != 0 {
		t.Errorf("AddPlayer ne doit pas etre appele, %d appels", watcher.calls)
	}
	if creator.calls != 1 {
		t.Errorf("le profil doit etre cree quand meme, %d appels", creator.calls)
	}
}

// Notification en échec : le profil reste la vérité, l'erreur n'est pas propagée.
func TestOnboard_EchecNotificationNonPropage(t *testing.T) {
	creator := newFakeCreator("Spartan")
	watcher := &fakeWatcher{running: true, err: errors.New("poller indisponible")}
	d := newTestDirectory(t, Deps{Profiles: &fakeProfiles{}, Creator: creator, Watcher: watcher})

	res, err := d.Onboard(context.Background(), onboardRequest())
	if err != nil {
		t.Fatalf("un echec de notification ne doit pas faire echouer Onboard : %v", err)
	}
	if res.WatcherNotified {
		t.Error("WatcherNotified devrait etre faux apres un echec d'AddPlayer")
	}
	if res.PlayerKey != "Spartan" {
		t.Errorf("le profil cree doit etre rendu, PlayerKey = %q", res.PlayerKey)
	}
}

// Création en échec : erreur propagée, watcher jamais appelé.
func TestOnboard_CreationEchoue(t *testing.T) {
	creator := newFakeCreator("Spartan")
	creator.err = errors.New("disque plein")
	watcher := &fakeWatcher{running: true}
	d := newTestDirectory(t, Deps{Profiles: &fakeProfiles{}, Creator: creator, Watcher: watcher})

	if _, err := d.Onboard(context.Background(), onboardRequest()); err == nil {
		t.Fatal("erreur attendue")
	}
	if watcher.calls != 0 {
		t.Errorf("AddPlayer ne doit pas etre appele sans profil, %d appels", watcher.calls)
	}
}

// Pas de watcher dans ce process (CLI) : le profil suffit.
func TestOnboard_SansWatcher(t *testing.T) {
	creator := newFakeCreator("Spartan")
	d := newTestDirectory(t, Deps{Profiles: &fakeProfiles{}, Creator: creator})

	res, err := d.Onboard(context.Background(), onboardRequest())
	if err != nil {
		t.Fatalf("Onboard: %v", err)
	}
	if res.WatcherNotified {
		t.Error("sans watcher, WatcherNotified doit etre faux")
	}
}

// Profil manuel (aucune identité Xbox) : le watcher suit PAR xuid, il n'y a donc
// rien à suivre — cas normal, pas une panne, donc pas d'appel du tout.
func TestOnboard_SansXUIDPasDeSuiviLive(t *testing.T) {
	creator := newFakeCreator("Spartan")
	watcher := &fakeWatcher{running: true}
	d := newTestDirectory(t, Deps{Profiles: &fakeProfiles{}, Creator: creator, Watcher: watcher})

	req := onboardRequest()
	req.XUID = ""
	res, err := d.Onboard(context.Background(), req)
	if err != nil {
		t.Fatalf("Onboard: %v", err)
	}
	if res.WatcherNotified || watcher.calls != 0 {
		t.Errorf("aucun suivi live sans xuid (notifie=%v, appels=%d)", res.WatcherNotified, watcher.calls)
	}
}

// Titre absent de la demande : normalisé sur le titre par défaut AVANT d'écrire
// le profil — un titre vide serait lu « tous les titres » par le chargeur.
func TestOnboard_TitreVideNormalise(t *testing.T) {
	creator := newFakeCreator("Spartan")
	d := newTestDirectory(t, Deps{Profiles: &fakeProfiles{}, Creator: creator})

	req := onboardRequest()
	req.TitleSlug = ""
	if _, err := d.Onboard(context.Background(), req); err != nil {
		t.Fatalf("Onboard: %v", err)
	}
	if creator.lastReq.TitleSlug != testTitle {
		t.Errorf("titre normalise attendu %q, recu %q", testTitle, creator.lastReq.TitleSlug)
	}
}

// Sans créateur câblé, Onboard refuse franchement : rendre un succès sans profil
// serait reproduire l'état exact du 2026-07-23.
func TestOnboard_SansCreateur(t *testing.T) {
	d := newTestDirectory(t, Deps{Profiles: &fakeProfiles{}})

	_, err := d.Onboard(context.Background(), onboardRequest())
	if !errors.Is(err, ErrOnboardNoCreator) {
		t.Fatalf("ErrOnboardNoCreator attendue, recu %v", err)
	}
}

// Gamertag vide : refus avant toute écriture.
func TestOnboard_GamertagVide(t *testing.T) {
	creator := newFakeCreator("Spartan")
	d := newTestDirectory(t, Deps{Profiles: &fakeProfiles{}, Creator: creator})

	req := onboardRequest()
	req.Gamertag = "   "
	_, err := d.Onboard(context.Background(), req)
	if !errors.Is(err, ErrOnboardInvalidGamertag) {
		t.Fatalf("ErrOnboardInvalidGamertag attendue, recu %v", err)
	}
	if creator.calls != 0 {
		t.Errorf("aucune ecriture ne doit avoir lieu, %d appels", creator.calls)
	}
}
