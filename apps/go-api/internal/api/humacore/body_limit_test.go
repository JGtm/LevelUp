// Package humacore — body_limit_test.go : plafond de corps par défaut des
// opérations Huma, pour chacune des trois fabriques d'API (NewAPI, NewAPIWithConfig,
// NewSubrouterAPI).
//
// Huma ne pose son plafond par défaut que pour un champ Body ; une entrée RawBody
// est lue sans limite. DefaultMaxBodyBytes comble ce manque pour toute opération
// qui ne déclare pas MaxBody ; une déclaration explicite l'emporte.
package humacore

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
)

type rawBodyIn struct{ RawBody []byte }
type rawBodyOut struct{ Body struct{ Bytes int } }

func rawBodyHandler(_ context.Context, in *rawBodyIn) (*rawBodyOut, error) {
	out := &rawBodyOut{}
	out.Body.Bytes = len(in.RawBody)
	return out, nil
}

// limitFactory construit une API par fabrique, montée sur un routeur chi neuf, et
// retourne le routeur et la huma.API (pour lire le document).
type limitFactory struct {
	name  string
	build func() (http.Handler, huma.API)
}

func limitFactories() []limitFactory {
	return []limitFactory{
		{"NewAPI", func() (http.Handler, huma.API) {
			r := chi.NewRouter()
			return r, NewAPI(r)
		}},
		{"NewAPIWithConfig", func() (http.Handler, huma.API) {
			r := chi.NewRouter()
			return r, NewAPIWithConfig(r, NewSharedConfig())
		}},
		{"NewSubrouterAPI", func() (http.Handler, huma.API) {
			r := chi.NewRouter()
			var api huma.API
			r.Route("/sub", func(sr chi.Router) {
				api = NewSubrouterAPI(sr, NewSharedConfig(), "/sub")
			})
			return r, api
		}},
	}
}

func postBody(h http.Handler, path string, size int) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(strings.Repeat("a", size)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func routePrefix(name string) string {
	if name == "NewSubrouterAPI" {
		return "/sub"
	}
	return ""
}

func TestDefaultMaxBody_RawBodyWithoutMaxBodyIsCapped(t *testing.T) {
	for _, f := range limitFactories() {
		t.Run(f.name, func(t *testing.T) {
			h, api := f.build()
			huma.Post(api, "/raw", rawBodyHandler, Op("rawDefault", "défaut", "t"))

			path := routePrefix(f.name) + "/raw"
			if w := postBody(h, path, 2<<20); w.Code != http.StatusRequestEntityTooLarge {
				t.Fatalf("corps de 2 Mio sans MaxBody : statut %d, attendu 413", w.Code)
			}
			if w := postBody(h, path, 1024); w.Code != http.StatusOK {
				t.Fatalf("corps de 1 Kio : statut %d, attendu 200 (%s)", w.Code, w.Body.String())
			}
		})
	}
}

func TestDefaultMaxBody_ExplicitMaxBodyWins(t *testing.T) {
	const explicit = 4 << 20
	for _, f := range limitFactories() {
		t.Run(f.name, func(t *testing.T) {
			h, api := f.build()
			huma.Post(api, "/big", rawBodyHandler, Op("rawExplicit", "explicite", "t"), MaxBody(explicit))

			path := routePrefix(f.name) + "/big"
			if w := postBody(h, path, 2<<20); w.Code != http.StatusOK {
				t.Fatalf("corps de 2 Mio sous MaxBody explicite de 4 Mio : statut %d, attendu 200", w.Code)
			}
			if w := postBody(h, path, explicit+1); w.Code != http.StatusRequestEntityTooLarge {
				t.Fatalf("corps au-delà du plafond explicite : statut %d, attendu 413", w.Code)
			}
		})
	}
}

// Le document porte le plafond effectif : c'est ce que parcourt le garde-rail
// qui exige un plafond strictement positif sur toute opération enregistrée.
func TestDefaultMaxBody_DocumentCarriesEffectiveLimit(t *testing.T) {
	for _, f := range limitFactories() {
		t.Run(f.name, func(t *testing.T) {
			_, api := f.build()
			huma.Post(api, "/raw", rawBodyHandler, Op("rawDoc", "doc", "t"))
			huma.Get(api, "/nobody", func(context.Context, *struct{}) (*rawBodyOut, error) { return &rawBodyOut{}, nil },
				Op("noBodyDoc", "doc", "t"))

			ops := 0
			for _, pi := range api.OpenAPI().Paths {
				for _, op := range []*huma.Operation{pi.Get, pi.Post} {
					if op == nil {
						continue
					}
					ops++
					if op.MaxBodyBytes != DefaultMaxBodyBytes {
						t.Errorf("%s : MaxBodyBytes = %d, attendu %d", op.OperationID, op.MaxBodyBytes, DefaultMaxBodyBytes)
					}
				}
			}
			if ops != 2 {
				t.Fatalf("opérations lues dans le document = %d, attendu 2", ops)
			}
		})
	}
}
