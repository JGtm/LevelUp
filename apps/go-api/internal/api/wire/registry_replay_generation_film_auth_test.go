package wire

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth/tokencache"

	"github.com/JGtm/halo-api/infinite"
)

const (
	genSpartanPerime = "SPARTAN-PERIME"
	genSpartanFrais  = "SPARTAN-FRAIS"
)

// authRig : le téléchargeur sous le VRAI filet d'authentification (tokencache.RetryWithFreshTokens), avec un client
// qui refuse le jeton périmé et un re-mint compté.
type authRig struct {
	*fetchRig
	f         *filmDemandFetcher
	attempts  []string // jeton de chaque essai, dans l'ordre
	reminted  atomic.Int32
	refresher func(context.Context, string) (*domain.HaloTokens, error)
	refusal   error // l'erreur que rend le client pour tout jeton refusé
	refused   func(token string) bool
}

func newAuthRig(t *testing.T) *authRig {
	t.Helper()
	g, f := newFetchRig(t)
	r := &authRig{fetchRig: g, f: f}
	g.tokenOf[genXUIDPage] = &domain.HaloTokens{SpartanToken: genSpartanPerime, ClearanceToken: "clr"}
	r.refusal = &infinite.HTTPError{StatusCode: 401, URL: "u"}
	r.refused = func(token string) bool { return token == genSpartanPerime }
	r.refresher = func(context.Context, string) (*domain.HaloTokens, error) {
		r.reminted.Add(1)
		return &domain.HaloTokens{SpartanToken: genSpartanFrais, ClearanceToken: "clr", SpartanExpiresAt: time.Now().Add(time.Hour)}, nil
	}
	f.newClient = func(tok *domain.HaloTokens) filmDownloader {
		r.attempts = append(r.attempts, tok.SpartanToken)
		if r.refused(tok.SpartanToken) {
			return &fakeDownloader{err: r.refusal}
		}
		return g.client
	}
	f.withFreshTokens = retryFilmWithFreshTokens
	tokencache.SetPlayerTokenRefresher(func(ctx context.Context, xuid string) (*domain.HaloTokens, error) {
		return r.refresher(ctx, xuid)
	})
	tokencache.InvalidateCachedPlayerTokens(genXUIDPage)
	t.Cleanup(func() {
		tokencache.SetPlayerTokenRefresher(nil)
		tokencache.InvalidateCachedPlayerTokens(genXUIDPage)
	})
	return r
}

// Un 401 ou un 403 pendant la résolution du film : le cache du joueur est invalidé, le jeton re-dérivé, et UN nouvel
// essai avec le jeton frais aboutit ; le film est archivé, aucun marqueur n'est posé.
func TestFilmDemande_RefusDuJetonUnNouvelEssaiAvecUnJetonFrais(t *testing.T) {
	for nom, refusal := range map[string]error{
		"401": &infinite.HTTPError{StatusCode: 401, URL: "u"},
		"403": &infinite.HTTPError{StatusCode: 403, URL: "u"},
	} {
		t.Run(nom, func(t *testing.T) {
			r := newAuthRig(t)
			r.refusal = refusal
			if err := r.f.fetch(context.Background(), genDemand()); err != nil {
				t.Fatalf("fetch: %v", err)
			}
			if got := strings.Join(r.attempts, ","); got != genSpartanPerime+","+genSpartanFrais {
				t.Fatalf("essais = %s, attendu le jeton périmé puis le jeton frais", got)
			}
			if r.reminted.Load() != 1 || len(r.written) != 1 || len(r.absentFor) != 0 {
				t.Fatalf("re-mints=%d écritures=%v marqueurs=%v", r.reminted.Load(), r.written, r.absentFor)
			}
			if len(r.tokenCalls) != 1 || r.tokenCalls[0] != genXUIDPage {
				t.Fatalf("le premier jeton se prend une fois, pour le xuid de la demande : %v", r.tokenCalls)
			}
		})
	}
}

// Un second refus rend l'erreur d'origine (un seul nouvel essai, pas de boucle) : échec de téléchargement, ni jeton
// absent ni film disparu, aucun marqueur.
func TestFilmDemande_DeuxiemeRefusFinitEnEchecSansBoucle(t *testing.T) {
	r := newAuthRig(t)
	r.refused = func(string) bool { return true }
	err := r.f.fetch(context.Background(), genDemand())
	if err == nil || !errors.Is(err, r.refusal) || errors.Is(err, domain.ErrReplayNoToken) || errors.Is(err, domain.ErrReplayFilmGone) {
		t.Fatalf("err = %v, attendu le refus d'origine", err)
	}
	if len(r.attempts) != 2 || r.reminted.Load() != 1 {
		t.Fatalf("essais=%v re-mints=%d, attendu deux essais et un re-mint", r.attempts, r.reminted.Load())
	}
	if len(r.absentFor) != 0 || len(r.written) != 0 {
		t.Fatalf("marqueurs=%v écritures=%v, attendu aucun", r.absentFor, r.written)
	}
}

// Un re-mint impossible rend l'erreur d'origine sans second essai.
func TestFilmDemande_ReMintImpossibleRendLErreurDOrigine(t *testing.T) {
	r := newAuthRig(t)
	r.refresher = func(context.Context, string) (*domain.HaloTokens, error) { return nil, errors.New("échange refusé") }
	err := r.f.fetch(context.Background(), genDemand())
	if err == nil || !errors.Is(err, r.refusal) {
		t.Fatalf("err = %v, attendu le refus d'origine", err)
	}
	if len(r.attempts) != 1 {
		t.Fatalf("essais = %v, attendu un seul", r.attempts)
	}
}

// Une erreur qui n'est pas un refus d'authentification (film disparu, 503) ne déclenche aucun nouvel essai.
func TestFilmDemande_AutreErreurNeRetentePas(t *testing.T) {
	for nom, cause := range map[string]error{
		"404": &infinite.HTTPError{StatusCode: 404, URL: "u"},
		"503": &infinite.HTTPError{StatusCode: 503, URL: "u"},
	} {
		t.Run(nom, func(t *testing.T) {
			r := newAuthRig(t)
			r.refusal = cause
			if err := r.f.fetch(context.Background(), genDemand()); err == nil {
				t.Fatal("une erreur est attendue")
			}
			if len(r.attempts) != 1 || r.reminted.Load() != 0 {
				t.Fatalf("essais=%v re-mints=%d, attendu un essai et aucun re-mint", r.attempts, r.reminted.Load())
			}
		})
	}
}

// Un film introuvable pour un match fini depuis moins de cinq minutes n'est pas tenu pour expiré : échec « film pas
// encore prêt » réessayable, aucun marqueur « film absent ». Passé ce délai, ou sans heure de fin connue, le verdict
// définitif tombe et le marqueur est posé. Le délai vaut pour les trois formes d'un film introuvable.
func TestFilmDemande_FilmIntrouvableApresLaFinDuMatch(t *testing.T) {
	maintenant := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	formes := map[string]func(g *fetchRig){
		"404":            func(g *fetchRig) { g.client.err = &infinite.HTTPError{StatusCode: 404, URL: "u"} },
		"410 d'un blob":  func(g *fetchRig) { g.client.err = &infinite.BlobHTTPError{StatusCode: 410, URL: "u"} },
		"manifeste vide": func(g *fetchRig) { g.client.found, g.client.chunks = false, nil },
	}
	cas := []struct {
		nom    string
		finiIl time.Duration // 0 = heure de fin inconnue
		neuf   bool          // attendu : film pas encore prêt
	}{
		{"fini depuis 2 minutes", 2 * time.Minute, true},
		{"fini depuis 4 min 59", 4*time.Minute + 59*time.Second, true},
		{"fini depuis 5 minutes pile", 5 * time.Minute, false},
		{"fini depuis 2 heures", 2 * time.Hour, false},
		{"heure de fin inconnue", 0, false},
	}
	for forme, prepare := range formes {
		for _, c := range cas {
			t.Run(forme+"/"+c.nom, func(t *testing.T) {
				g, f := newFetchRig(t)
				prepare(g)
				f.now = func() time.Time { return maintenant }
				d := genDemand()
				if c.finiIl > 0 {
					d.MatchEndedAt = maintenant.Add(-c.finiIl)
				}
				err := f.fetch(context.Background(), d)
				if len(g.written) != 0 {
					t.Fatalf("aucun film ne doit être archivé : %v", g.written)
				}
				if c.neuf {
					if !errors.Is(err, domain.ErrReplayFilmNotReady) || errors.Is(err, domain.ErrReplayFilmGone) {
						t.Fatalf("err = %v, attendu ErrReplayFilmNotReady", err)
					}
					if len(g.absentFor) != 0 {
						t.Fatalf("le bit « film absent » ne doit pas être posé : %v", g.absentFor)
					}
					return
				}
				if !errors.Is(err, domain.ErrReplayFilmGone) || errors.Is(err, domain.ErrReplayFilmNotReady) {
					t.Fatalf("err = %v, attendu ErrReplayFilmGone", err)
				}
				if len(g.absentFor) != 1 || g.absentFor[0] != d.MatchID {
					t.Fatalf("le bit « film absent » doit être posé pour le match : %v", g.absentFor)
				}
			})
		}
	}
}
