// Package api — server_presence_test.go : LA JONCTION entre le daemon watcher et
// GET /api/v1/presence, jouée bout en bout sur un routeur httptest.
//
// POURQUOI CE TEST, alors que le handler et le service ont déjà les leurs. Les deux
// autres partent d'une `service.TrackedPresence` DÉJÀ CONSTRUITE : ils ne peuvent
// donc rien dire de l'adaptateur qui la construit — celui qui lit `status.Running`,
// recopie le titre courant et parse `last_event_at`. C'est pourtant là que vivent
// les deux règles que l'écran voit : « daemon arrêté ⇒ on ne sait rien » et « un
// titre dont le poll s'est tu n'est plus un titre ».
//
// La dernière section va un cran plus haut : elle exerce buildPresenceService,
// c'est-à-dire le MONTAGE lui-même. Tous les autres harnais recomposent le
// service à la main et ne verraient donc pas un branchement manquant.
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/service"
	"levelup/go-api/internal/watcher"
)

// presenceFakeDaemon : un watcher.DaemonController réduit à ce que l'adaptateur
// lui demande — son état. Les autres méthodes sont des no-op : l'adaptateur ne
// contrôle pas le daemon, il le lit.
type presenceFakeDaemon struct{ status watcher.WatcherStatus }

var _ watcher.DaemonController = (*presenceFakeDaemon)(nil)

func (d *presenceFakeDaemon) Start(context.Context, string, []domain.PlayerSummary) {}
func (d *presenceFakeDaemon) Stop()                                                 {}
func (d *presenceFakeDaemon) UpdateAuth(string)                                     {}
func (d *presenceFakeDaemon) UpdateSubscriptions([]string)                          {}
func (d *presenceFakeDaemon) IsRunning() bool                                       { return d.status.Running }
func (d *presenceFakeDaemon) AddPlayer(context.Context, domain.PlayerSummary) error { return nil }
func (d *presenceFakeDaemon) GetStatus() watcher.WatcherStatus                      { return d.status }

// presenceSnapshotFrom monte l'endpoint sur le service RÉEL branché sur
// l'ADAPTATEUR RÉEL (trackedPresenceFrom), et rend le JSON servi.
func presenceSnapshotFrom(t *testing.T, daemon watcher.DaemonController) map[string]any {
	t.Helper()
	return presenceBodyOf(t, service.NewPresenceService(
		func(context.Context, *domain.SessionData) ([]domain.PlayerSummary, error) {
			return []domain.PlayerSummary{{PlayerSlug: "jgtm", Gamertag: "JGtm"}}, nil
		},
		trackedPresenceFrom(daemon),
	))
}

// presenceBodyOf monte l'endpoint sur le service donné et rend le JSON servi.
func presenceBodyOf(t *testing.T, svc *service.PresenceService) map[string]any {
	t.Helper()
	r := chi.NewRouter()
	handlers.NewPresenceHandler(svc).Mount(r)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/presence", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, attendu 200 : %s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("JSON invalide : %v (%s)", err, w.Body.String())
	}
	return body
}

// statusWithPlayerInGame : le daemon a vu JGtm sur Halo Infinite il y a `age`.
func statusWithPlayerInGame(running bool, age time.Duration) watcher.WatcherStatus {
	return watcher.WatcherStatus{
		Running: running,
		Players: []watcher.PlayerPresenceStatus{{
			Gamertag:    "JGtm",
			XUID:        "111",
			TitleSlug:   "halo_infinite",
			TitleName:   "Halo Infinite",
			LastEventAt: time.Now().Add(-age).UTC().Format(time.RFC3339),
		}},
	}
}

// Daemon vivant, joueur en jeu : le JSON porte le joueur, son slug de
// configuration et le titre RÉEL capté par le watcher.
func TestPresenceJunction_RunningDaemon_ServesFullPayload(t *testing.T) {
	body := presenceSnapshotFrom(t, &presenceFakeDaemon{status: statusWithPlayerInGame(true, 5*time.Second)})

	players, ok := body["players"].([]any)
	if !ok || len(players) != 1 {
		t.Fatalf("players = %v, attendu 1 entrée", body["players"])
	}
	p, _ := players[0].(map[string]any)
	if p["player_slug"] != "jgtm" || p["gamertag"] != "JGtm" {
		t.Errorf("identité servie = %+v", p)
	}
	if p["in_game"] != true || p["title_slug"] != "halo_infinite" || p["title_name"] != "Halo Infinite" {
		t.Errorf("titre servi = %+v", p)
	}
	// Rien n'est asserté ici sur friends_in_game : ce harnais ne branche AUCUN
	// compteur, un zéro y serait vrai par construction. Le compteur se teste sur
	// le montage réel, plus bas (TestPresenceJunction_BuildPresenceService_...).
}

// Daemon ARRÊTÉ après avoir vu ce même joueur en jeu : l'état est encore dans la
// structure, mais il ne dit plus rien du présent. La liste doit être VIDE — pas
// une manette figée sur un joueur dont plus personne n'observe la présence.
func TestPresenceJunction_StoppedDaemon_ServesEmptyList(t *testing.T) {
	body := presenceSnapshotFrom(t, &presenceFakeDaemon{status: statusWithPlayerInGame(false, 5*time.Second)})

	players, ok := body["players"].([]any)
	if !ok {
		t.Fatalf("players = %v (%T), attendu un tableau", body["players"], body["players"])
	}
	if len(players) != 0 {
		t.Errorf("players = %d, attendu 0 (daemon arrêté : on ne sait rien)", len(players))
	}
}

// Daemon vivant mais poll MUET depuis longtemps : le joueur reste listé (on le
// suit), sans titre ni « en jeu » — la borne de fraîcheur traverse l'adaptateur.
func TestPresenceJunction_StaleLastEvent_ClearsTitle(t *testing.T) {
	body := presenceSnapshotFrom(t, &presenceFakeDaemon{status: statusWithPlayerInGame(true, time.Hour)})

	players, _ := body["players"].([]any)
	if len(players) != 1 {
		t.Fatalf("players = %v, attendu 1 entrée", body["players"])
	}
	p, _ := players[0].(map[string]any)
	if p["in_game"] != false {
		t.Errorf("in_game = %v, attendu false (dernier event il y a une heure)", p["in_game"])
	}
	// `title_slug` / `title_name` sont omitempty : blanchis, ils disparaissent.
	if _, present := p["title_slug"]; present {
		t.Errorf("title_slug encore servi = %+v", p)
	}
}

// `last_event_at` vide (aucun event depuis le boot) : aucun titre servi, et
// surtout aucune panique au parsing.
func TestPresenceJunction_MissingLastEvent_ClearsTitle(t *testing.T) {
	status := statusWithPlayerInGame(true, 0)
	status.Players[0].LastEventAt = ""
	body := presenceSnapshotFrom(t, &presenceFakeDaemon{status: status})

	players, _ := body["players"].([]any)
	if len(players) != 1 {
		t.Fatalf("players = %v, attendu 1 entrée", body["players"])
	}
	if p, _ := players[0].(map[string]any); p["in_game"] != false {
		t.Errorf("in_game = %v, attendu false (aucun event reçu)", p["in_game"])
	}
}

// Daemon absent (watcher désactivé par la configuration) : l'adaptateur rend nil,
// et l'endpoint sert quand même une liste vide en 200.
func TestPresenceJunction_NoDaemon_ServesEmptyList(t *testing.T) {
	if got := trackedPresenceFrom(nil); got != nil {
		t.Fatal("trackedPresenceFrom(nil) doit rendre nil (aucune source)")
	}
	body := presenceSnapshotFrom(t, nil)
	if players, ok := body["players"].([]any); !ok || len(players) != 0 {
		t.Errorf("players = %v, attendu tableau vide", body["players"])
	}
}

// ─── LE MONTAGE : buildPresenceService ────────────────────────────────────────

// presenceBootstrap monte un BootstrapService RÉEL sur un db_profiles temporaire
// de deux joueurs, en mode d'authentification `none` — la configuration PAR
// DÉFAUT d'une instance. Sans identités, personne ne possède rien en propre :
// les deux joueurs visibles en jeu comptent comme des amis (règle des deux
// régimes, plan amis-app lot H).
//
// Le repo de bootstrap est nil à dessein : le chemin exercé ici (OwnedPlayers →
// LoadPlayers → filtre de propriété) ne le touche pas.
func presenceBootstrap(t *testing.T) *service.BootstrapService {
	t.Helper()
	profilesPath := filepath.Join(t.TempDir(), "db_profiles.json")
	profiles := `{"version":"3.0","profiles":{"halo_infinite":{
	  "JGtm":{"db_path":"a.duckdb","xuid":"111"},
	  "Madina":{"db_path":"b.duckdb","xuid":"222"}}}}`
	if err := os.WriteFile(profilesPath, []byte(profiles), 0o644); err != nil {
		t.Fatal(err)
	}
	return service.NewBootstrapService(
		&config.AppConfig{AuthMode: "none", DBProfilesPath: profilesPath}, nil)
}

// statusWithTwoPlayersInGame : les deux joueurs du parc, vus à l'instant.
func statusWithTwoPlayersInGame() watcher.WatcherStatus {
	now := time.Now().UTC().Format(time.RFC3339)
	return watcher.WatcherStatus{
		Running: true,
		Players: []watcher.PlayerPresenceStatus{
			{Gamertag: "JGtm", XUID: "111", TitleSlug: "halo_infinite", TitleName: "Halo Infinite", LastEventAt: now},
			{Gamertag: "Madina", XUID: "222", TitleSlug: "halo_infinite", TitleName: "Halo Infinite", LastEventAt: now},
		},
	}
}

// LE CÂBLAGE RÉEL, celui que le serveur exécute. Les autres tests de ce fichier
// recomposent le service à la main : aucun ne verrait `WithFriends` disparaître
// du montage. Celui-ci part de buildPresenceService et exige un compteur NON NUL
// — retirer le branchement le fait tomber (2 → 0).
func TestPresenceJunction_BuildPresenceService_WiresFriendsCounter(t *testing.T) {
	daemon := &presenceFakeDaemon{status: statusWithTwoPlayersInGame()}

	body := presenceBodyOf(t, buildPresenceService(presenceBootstrap(t), daemon))

	if players, ok := body["players"].([]any); !ok || len(players) != 2 {
		t.Fatalf("players = %v, attendu les 2 joueurs du parc", body["players"])
	}
	if body["friends_in_game"] != float64(2) {
		t.Errorf("friends_in_game = %v, attendu 2 — le compteur n'est pas branché au montage",
			body["friends_in_game"])
	}
}

// Branche sans bootstrap (boot partiel, service indisponible) : ni liste ni
// compteur, mais toujours une réponse 200 bien formée.
func TestPresenceJunction_BuildPresenceService_NoBootstrap_ServesEmpty(t *testing.T) {
	daemon := &presenceFakeDaemon{status: statusWithTwoPlayersInGame()}

	body := presenceBodyOf(t, buildPresenceService(nil, daemon))

	if players, ok := body["players"].([]any); !ok || len(players) != 0 {
		t.Errorf("players = %v, attendu tableau vide (aucune source de joueurs)", body["players"])
	}
	if body["friends_in_game"] != float64(0) {
		t.Errorf("friends_in_game = %v, attendu 0 (aucune propriété connaissable)", body["friends_in_game"])
	}
}
