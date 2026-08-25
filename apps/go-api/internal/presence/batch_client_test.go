package presence

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// fixtureBatchThreeUsers : réponse batch observable — un ami en jeu sur Halo
// Infinite, un sur un autre jeu, un hors jeu (state seul, aucun device). Le
// format d'un élément est celui du poll unitaire (cf. rest_client_test.go).
const fixtureBatchThreeUsers = `[
  {"xuid":"111","state":"Online","devices":[{"type":"WindowsOneCore","titles":[
    {"id":"2043073184","name":"Halo Infinite","placement":"Full","state":"Active"}]}]},
  {"xuid":"222","state":"Online","devices":[{"type":"WindowsOneCore","titles":[
    {"id":"2076696971","name":"Counter-Strike 2","placement":"Full","state":"Active"}]}]},
  {"xuid":"333","state":"Offline"}
]`

// batchServer monte un httptest qui vérifie la forme de la requête batch et
// renvoie le corps fourni. capturedBody reçoit le corps décodé de la requête.
func batchServer(t *testing.T, status int, body string, captured *batchPresenceRequest) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("méthode = %q, attendu POST", r.Method)
		}
		if r.URL.Path != "/users/batch" {
			t.Errorf("path = %q, attendu /users/batch", r.URL.Path)
		}
		if r.Header.Get("Authorization") == "" {
			t.Error("Authorization header manquant")
		}
		if got := r.Header.Get("x-xbl-contract-version"); got != "3" {
			t.Errorf("x-xbl-contract-version = %q, attendu 3", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, attendu application/json", got)
		}
		if captured != nil {
			raw, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(raw, captured); err != nil {
				t.Errorf("corps de requête illisible: %v", err)
			}
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
}

// Le lot rend un événement par élément, avec le titre actif de chacun : c'est
// la donnée dont dépend tout le comptage « amis en jeu ».
func TestGetPresenceBatch_ParsesEachRecord(t *testing.T) {
	var req batchPresenceRequest
	ts := batchServer(t, http.StatusOK, fixtureBatchThreeUsers, &req)
	defer ts.Close()

	events, err := presenceClientPointingTo(ts.URL).
		GetPresenceBatch(context.Background(), []string{"111", "222", "333"})
	if err != nil {
		t.Fatalf("GetPresenceBatch error = %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("events = %d, attendu 3", len(events))
	}
	if events[0].XUID != "111" || events[0].PresenceDetail == nil ||
		events[0].PresenceDetail.TitleID != "2043073184" {
		t.Errorf("1er élément mal parsé: %+v", events[0])
	}
	if events[1].PresenceDetail == nil || events[1].PresenceDetail.TitleID != "2076696971" {
		t.Errorf("2e élément (autre jeu) mal parsé: %+v", events[1])
	}
	if events[2].PresenceDetail != nil {
		t.Errorf("3e élément (Offline) ne doit porter aucun titre: %+v", events[2].PresenceDetail)
	}
}

// `level: all` est ce qui fait rendre devices[].titles[] par Xbox : sans lui la
// réponse n'a pas de titre et le compteur d'amis serait toujours nul.
func TestGetPresenceBatch_SendsUsersAndLevelAll(t *testing.T) {
	var req batchPresenceRequest
	ts := batchServer(t, http.StatusOK, `[]`, &req)
	defer ts.Close()

	// Le xuid vide doit être écarté avant l'envoi (Xbox rejette le lot entier).
	if _, err := presenceClientPointingTo(ts.URL).
		GetPresenceBatch(context.Background(), []string{"111", "", "222"}); err != nil {
		t.Fatalf("GetPresenceBatch error = %v", err)
	}
	if req.Level != "all" {
		t.Errorf("level = %q, attendu all", req.Level)
	}
	if len(req.Users) != 2 || req.Users[0] != "111" || req.Users[1] != "222" {
		t.Errorf("users = %v, attendu [111 222] (xuid vide écarté)", req.Users)
	}
}

// Aucun xuid exploitable = aucun appel réseau : le serveur de test échouerait
// s'il était touché (il n'attend pas d'appel), et le résultat est vide sans erreur.
func TestGetPresenceBatch_EmptyInput_NoCall(t *testing.T) {
	called := false
	ts := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	defer ts.Close()

	events, err := presenceClientPointingTo(ts.URL).
		GetPresenceBatch(context.Background(), []string{"", ""})
	if err != nil {
		t.Fatalf("GetPresenceBatch error = %v", err)
	}
	if len(events) != 0 {
		t.Errorf("events = %d, attendu 0", len(events))
	}
	if called {
		t.Error("aucun appel réseau ne doit partir pour une liste vide")
	}
}

// Un élément illisible ne doit pas emporter le lot : une présence perdue coûte
// un ami non compté, une erreur coûterait le compteur entier.
func TestGetPresenceBatch_SkipsUnparsableRecord(t *testing.T) {
	body := `[42, {"xuid":"111","state":"Online","devices":[{"type":"PC","titles":[
	  {"id":"2043073184","name":"Halo Infinite","placement":"Full","state":"Active"}]}]}]`
	ts := batchServer(t, http.StatusOK, body, nil)
	defer ts.Close()

	events, err := presenceClientPointingTo(ts.URL).
		GetPresenceBatch(context.Background(), []string{"111"})
	if err != nil {
		t.Fatalf("GetPresenceBatch error = %v", err)
	}
	if len(events) != 1 || events[0].XUID != "111" {
		t.Fatalf("events = %+v, attendu le seul élément lisible", events)
	}
}

// Une réponse non-OK remonte en *HTTPError pour que l'appelant discrimine
// 401 (XSTS expiré) / 429 / 5xx, comme sur le chemin unitaire.
func TestGetPresenceBatch_HTTPErrorIsTyped(t *testing.T) {
	ts := batchServer(t, http.StatusUnauthorized, `{"error":"expired"}`, nil)
	defer ts.Close()

	_, err := presenceClientPointingTo(ts.URL).
		GetPresenceBatch(context.Background(), []string{"111"})
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("err = %v, attendu *HTTPError", err)
	}
	if !httpErr.IsAuthExpired() {
		t.Errorf("status = %d, attendu 401 reconnu comme auth expirée", httpErr.StatusCode)
	}
}
