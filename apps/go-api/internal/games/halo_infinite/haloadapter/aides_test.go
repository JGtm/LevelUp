package haloadapter

// aides_test.go — le montage commun des tests de l'adaptateur.

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"golang.org/x/time/rate"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/haloapi/infinite"
	"levelup/go-api/internal/platform/auth/tokencache"
)

// avecRafraichisseur câble `fn` comme rafraîchisseur de jetons du processus pour la durée du
// test, et vide le cache de jetons de `xuid` avant et après (chaque test a son xuid).
func avecRafraichisseur(t *testing.T, xuid string, fn func(context.Context, string) (*domain.HaloTokens, error)) {
	t.Helper()
	tokencache.InvalidateCachedPlayerTokens(xuid)
	tokencache.SetPlayerTokenRefresher(fn)
	t.Cleanup(func() {
		tokencache.SetPlayerTokenRefresher(nil)
		tokencache.InvalidateCachedPlayerTokens(xuid)
	})
}

// fastLimiter retourne un rate.Limiter quasi illimité (1000 RPS) pour les tests qui ne testent
// pas le débit.
func fastLimiter() *rate.Limiter {
	return rate.NewLimiter(rate.Every(time.Millisecond), 1)
}

// redirectTransport redirige toutes les requêtes HTTP vers un serveur httptest : les adresses de
// production restent celles des requêtes.
type redirectTransport struct{ host string }

func (rt *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	req2.URL.Scheme = "http"
	req2.URL.Host = rt.host
	return http.DefaultTransport.RoundTrip(req2)
}

// Les fonctions pures de la brique, sous les noms que les tests de l'adaptateur emploient.
var (
	filterHumanXUIDs       = infinite.FilterHumanXUIDs
	unwrapXUID             = infinite.UnwrapXUID
	firstPositiveEmblemCfg = infinite.FirstPositiveEmblemCfg
	extractEmblemStem      = infinite.ExtractEmblemStem
)

// matchSkillResponse : la réponse skill telle que la brique la décode.
type matchSkillResponse = infinite.MatchSkillDTO

// parsePlaylistCSR décode une réponse `/playlist/{id}/csrs` et rend le CSR du xuid demandé (nil
// sans entrée), comme le chemin live.
func parsePlaylistCSR(body []byte, playlistID, xuid string) (*PlayerPlaylistCSR, error) {
	var dto infinite.PlaylistCSRDTO
	if err := json.Unmarshal(body, &dto); err != nil {
		return nil, err
	}
	return playlistCSRDepuisDTO(&dto, playlistID, xuid), nil
}

// resetEmblemMappingCacheForTest vide le cache de processus de la table emblème -> nameplate.
func resetEmblemMappingCacheForTest() {
	emblemMappingMu.Lock()
	emblemMappingData = nil
	emblemMappingFetched = time.Time{}
	emblemMappingMu.Unlock()
}

// seedEmblemMappingCacheForTest amorce le cache avec une table arbitraire (branche « table
// trouvée » sans réseau).
func seedEmblemMappingCacheForTest(data map[string]map[string]emblemMappingEntry) {
	emblemMappingMu.Lock()
	emblemMappingData = data
	emblemMappingFetched = time.Now()
	emblemMappingMu.Unlock()
}
