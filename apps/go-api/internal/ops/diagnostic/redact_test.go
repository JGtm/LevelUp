package diagnostic

import (
	"encoding/json"
	"strings"
	"testing"
)

// opaque : une valeur sans aucune forme de jeton connue. Seule la CLÉ qui la porte peut la faire retirer : c'est ce
// qui permet à un test de désigner le filtre par clé, et lui seul.
const opaque = "valeur-opaque-8127"

func TestForbiddenKey_SecretNamesAreCaught(t *testing.T) {
	for _, k := range []string{
		"refresh_token", "oauth_refresh_token", "OAuthRefreshToken", "access_token", "AccessToken", "id_token",
		"xsts_token", "SpartanToken", "spartan_token", "token", "Token", "bearer", "authorization", "Authorization",
		"cookie", "Set-Cookie", "session_cookie", "desktop_secret", "session_secret", "client_secret", "SecretKey",
		"password", "PasswordHash", "admin_password", "discord_webhook_url", "DiscordWebhookURL", "webhook_url",
		"api_key", "apiKey", "private_key", "code", "auth_code", "device_code", "credentials", "session_key",
	} {
		if !ForbiddenKey(k) {
			t.Errorf("ForbiddenKey(%q) = false : un nom de secret doit être retiré", k)
		}
	}
}

// TestForbiddenKey_CountersAndSettingsAreKept : une clé qui contient seulement un mot de secret (compteur, réglage,
// capacité, drapeau de présence) reste dans le rapport : il sert au diagnostic.
func TestForbiddenKey_CountersAndSettingsAreKept(t *testing.T) {
	for _, k := range []string{
		"scheduled_refresh", "ScheduledRefresh", "refresh_clears_caches", "spnkr_refresh_with_backfill",
		"spnkr_refresh_backfill_aliases", "discord_webhook_url_present", "TokenVault", "PoolTokens", "CanPoolTokens",
		"players_with_token", "xsts_refreshed", "has_refresh_token", "status_code", "error_code", "tokens_error",
		"color_token", "match_id", "xuid", "gamertag", "client_secret_count",
	} {
		if ForbiddenKey(k) {
			t.Errorf("ForbiddenKey(%q) = true : une clé d'état ou de réglage ne doit pas être effacée", k)
		}
	}
}

// TestScrub_EachFilterCatchesWhatItIsFor : chaque filtre de Scrub est exercé SEUL, par une entrée que les autres
// laissent passer. Inverser un filtre (le retirer, le laisser passer) fait rougir son cas, et lui seul.
func TestScrub_EachFilterCatchesWhatItIsFor(t *testing.T) {
	jwt := "eyJ" + strings.Repeat("a", 12) + "." + strings.Repeat("b", 10) + "." + strings.Repeat("c", 10)
	ms := "M.C5_" + strings.Repeat("Ab1Cd2Ef3", 6)
	for name, c := range map[string]struct {
		in      string
		secrets []string
		gone    string
	}{
		"valeur exacte":            {in: "env launcher is " + opaque, secrets: []string{opaque}, gone: opaque},
		"en-tête d'autorisation":   {in: "header Bearer " + strings.Repeat("Zy9Xw8", 6), gone: strings.Repeat("Zy9Xw8", 6)},
		"JWT":                      {in: "got " + jwt + " from server", gone: jwt},
		"jeton Microsoft":          {in: "rt " + ms + " rotated", gone: ms},
		"paramètre d'adresse":      {in: "GET /cb?state=" + opaque + " 200", gone: opaque},
		"paire de clé sensible":    {in: "auth: refresh_token=" + opaque + " xuid=1", gone: opaque},
		"paire JSON dans du texte": {in: `auth: {"webhook_url":"` + opaque + `","n":1}`, gone: opaque},
		"paire entre apostrophes":  {in: "cfg password: '" + opaque + "'", gone: opaque},
	} {
		if got := Scrub(c.in, c.secrets); strings.Contains(got, c.gone) {
			t.Errorf("%s : %q survit à Scrub : %q", name, c.gone, got)
		}
	}
	// Ce qui n'est pas secret ressort tel quel.
	for _, keep := range []string{"refresh-only: terminé", "players_with_token=3 xsts_refreshed=2", "status_code=200 duration_ms=12"} {
		if got := Scrub(keep, nil); got != keep {
			t.Errorf("Scrub(%q) = %q : rien à retirer", keep, got)
		}
	}
}

// TestScrubLog_JSONLineStaysValidJSON : la ligne JSON d'un journal par module reste du JSON (le guillemet fermant
// n'est jamais avalé), les compteurs et le message survivent, l'ordre des clés est gardé, seule la valeur de la
// clé sensible disparaît (T2).
func TestScrubLog_JSONLineStaysValidJSON(t *testing.T) {
	line := `{"time":"2026-11-03T09:30:00Z","level":"INFO","msg":"refresh-only: terminé","players_with_token":3,` +
		`"xsts_refreshed":2,"refresh_token":"` + opaque + `","password":"a \"` + opaque + `\" tail","credentials":{"user":"x","blob":["` + opaque + `",1]},"nested":{"cookie":"` + opaque + `","keep":"visible","list":[1,"a"]},"last":true}`
	got := ScrubLog(line, nil)
	if !json.Valid([]byte(got)) {
		t.Fatalf("la ligne n'est plus du JSON : %s", got)
	}
	if strings.Contains(got, opaque) {
		t.Fatalf("la valeur d'une clé sensible survit : %s", got)
	}
	for _, want := range []string{`"msg":"refresh-only: terminé"`, `"players_with_token":3`, `"xsts_refreshed":2`, `"keep":"visible"`, `"last":true`} {
		if !strings.Contains(got, want) {
			t.Errorf("%s a disparu : %s", want, got)
		}
	}
	if strings.Index(got, `"time"`) > strings.Index(got, `"msg"`) || strings.Index(got, `"msg"`) > strings.Index(got, `"nested"`) {
		t.Errorf("l'ordre des clés n'est pas gardé : %s", got)
	}
}

// TestScrubLog_TextLineKeepsItsStructure : une ligne de texte garde ses compteurs ; la valeur retirée est bornée au
// guillemet (un fragment JSON dans une ligne de texte reste du JSON).
func TestScrubLog_TextLineKeepsItsStructure(t *testing.T) {
	in := "2026-11-03 INFO refresh-only: terminé players_with_token=3 xsts_refreshed=2\n" +
		`2026-11-03 ERROR auth: {"refresh_token":"` + opaque + `","n":1}` + "\n" +
		"2026-11-03 WARN auth: password=" + opaque + " user=x\n"
	got := ScrubLog(in, nil)
	if strings.Contains(got, opaque) {
		t.Fatalf("une valeur sensible survit : %s", got)
	}
	lines := strings.Split(got, "\n")
	if lines[0] != "2026-11-03 INFO refresh-only: terminé players_with_token=3 xsts_refreshed=2" {
		t.Errorf("une ligne sans secret a été modifiée : %q", lines[0])
	}
	frag := lines[1][strings.Index(lines[1], "{"):]
	if !json.Valid([]byte(frag)) {
		t.Errorf("le fragment JSON de la ligne n'est plus valide : %q", frag)
	}
	if !strings.Contains(lines[2], "user=x") {
		t.Errorf("la paire voisine a été avalée : %q", lines[2])
	}
}

// TestScrubLog_ExactSecretsAndShapesInsideJSONStrings : dans une ligne JSON, le texte des chaînes passe par Scrub (une
// forme de jeton dans un message est retirée) sans casser le document.
func TestScrubLog_ExactSecretsAndShapesInsideJSONStrings(t *testing.T) {
	line := `{"msg":"launcher ` + opaque + ` bearer ` + strings.Repeat("Zy9Xw8", 6) + `","n":1}`
	got := ScrubLog(line, []string{opaque})
	if !json.Valid([]byte(got)) || strings.Contains(got, opaque) || strings.Contains(got, strings.Repeat("Zy9Xw8", 6)) {
		t.Fatalf("ligne : %s", got)
	}
}
