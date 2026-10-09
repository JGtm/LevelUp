// Package handlers — map_service_error_test.go : tri des erreurs de service des pages
// (plan perf du 2026-09-23, D3.2) — statut, code, en-tête Retry-After et NIVEAU de log.
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"

	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// pageErrorLogRecord : une ligne JSON du logger capturé.
type pageErrorLogRecord struct {
	Level string `json:"level"`
	Msg   string `json:"msg"`
	Code  string `json:"code"`
}

// capturePageErrorLogs redirige le logger par défaut (niveau DEBUG, format JSON) le temps
// du test et rend les lignes émises par mapServiceError (message préfixé « page: »).
func capturePageErrorLogs(t *testing.T) func() []pageErrorLogRecord {
	t.Helper()
	var buf bytes.Buffer
	ancien := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(ancien) })
	return func() []pageErrorLogRecord {
		var out []pageErrorLogRecord
		for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
			var rec pageErrorLogRecord
			if json.Unmarshal([]byte(line), &rec) == nil && strings.HasPrefix(rec.Msg, "page:") {
				out = append(out, rec)
			}
		}
		return out
	}
}

// pageErrorView : ce que le client (ou le journal d'accès) verrait de l'erreur rendue.
type pageErrorView struct {
	status     int
	code       string
	retryable  bool
	retryAfter string
}

func viewPageError(t *testing.T, err error) pageErrorView {
	t.Helper()
	var se huma.StatusError
	if !errors.As(err, &se) {
		t.Fatalf("erreur rendue sans statut HTTP : %T %v", err, err)
	}
	body, merr := json.Marshal(se)
	if merr != nil {
		t.Fatalf("corps d'erreur non sérialisable : %v", merr)
	}
	var env struct {
		Code      string `json:"code"`
		Retryable bool   `json:"retryable"`
	}
	if uerr := json.Unmarshal(body, &env); uerr != nil {
		t.Fatalf("corps d'erreur illisible %s : %v", body, uerr)
	}
	v := pageErrorView{status: se.GetStatus(), code: env.Code, retryable: env.Retryable}
	var he huma.HeadersError
	if errors.As(err, &he) {
		v.retryAfter = he.GetHeaders().Get(headerRetryAfter)
	}
	return v
}

func TestMapServiceError(t *testing.T) {
	annule, cancel := context.WithCancel(context.Background())
	cancel()
	vivant := context.Background()

	cases := []struct {
		name     string
		ctx      context.Context
		err      error
		want     pageErrorView
		wantLogs string // niveau attendu de l'UNIQUE ligne « page: »
	}{
		{
			name:     "client parti (contexte de la requete annule)",
			ctx:      annule,
			err:      fmt.Errorf("teammates.GetPage: %w", context.Canceled),
			want:     pageErrorView{status: 499, code: "client_closed", retryable: false},
			wantLogs: "DEBUG",
		},
		{
			name:     "client parti pendant une attente de verrou : 499 prime sur 503",
			ctx:      annule,
			err:      fmt.Errorf("acquire: %w", dblease.ErrDBLocked),
			want:     pageErrorView{status: 499, code: "client_closed", retryable: false},
			wantLogs: "DEBUG",
		},
		{
			name:     "verrou d'ecriture non obtenu",
			ctx:      vivant,
			err:      fmt.Errorf("acquire: %w", dblease.ErrDBLocked),
			want:     pageErrorView{status: 503, code: "db_busy", retryable: true, retryAfter: "5"},
			wantLogs: "WARN",
		},
		{
			name:     "bascule RO/RW trop longue",
			ctx:      vivant,
			err:      fmt.Errorf("PlayerMatchesRepo.Load: %w", sharedprovider.ErrSwapTimeout),
			want:     pageErrorView{status: 503, code: "db_busy", retryable: true, retryAfter: "5"},
			wantLogs: "WARN",
		},
		{
			name:     "provider partage en erreur",
			ctx:      vivant,
			err:      sharedprovider.ErrSwapFailed,
			want:     pageErrorView{status: 503, code: "db_busy", retryable: true, retryAfter: "5"},
			wantLogs: "WARN",
		},
		{
			name:     "annulation interne alors que le client attend : erreur serveur",
			ctx:      vivant,
			err:      fmt.Errorf("errgroup: %w", context.Canceled),
			want:     pageErrorView{status: 500, code: "teammates_error", retryable: true},
			wantLogs: "ERROR",
		},
		{
			name:     "erreur quelconque",
			ctx:      vivant,
			err:      errors.New("boom"),
			want:     pageErrorView{status: 500, code: "teammates_error", retryable: true},
			wantLogs: "ERROR",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			logs := capturePageErrorLogs(t)
			got := viewPageError(t, mapServiceError(c.ctx, c.err, "teammates_error"))
			if got != c.want {
				t.Errorf("rendu = %+v, attendu %+v", got, c.want)
			}
			recs := logs()
			if len(recs) != 1 {
				t.Fatalf("attendu UNE ligne « page: », got %d : %+v", len(recs), recs)
			}
			if recs[0].Level != c.wantLogs {
				t.Errorf("niveau de log = %s, attendu %s (%q)", recs[0].Level, c.wantLogs, recs[0].Msg)
			}
			if recs[0].Code != "teammates_error" {
				t.Errorf("la ligne de log doit porter le code de la page, got %q", recs[0].Code)
			}
		})
	}
}

// TestHomePageError_GardeSesCodes : l'Accueil garde ses deux 503 historiques, et gagne
// le 499 du client parti et le 503 `db_busy` du verrou d'écriture.
func TestHomePageError_GardeSesCodes(t *testing.T) {
	annule, cancel := context.WithCancel(context.Background())
	cancel()
	vivant := context.Background()

	cases := []struct {
		name string
		ctx  context.Context
		err  error
		want pageErrorView
	}{
		{"bascule RO/RW", vivant, sharedprovider.ErrSwapTimeout,
			pageErrorView{status: 503, code: "home_page_db_busy", retryable: true, retryAfter: "2"}},
		{"base en recuperation", vivant, errors.New("sql: database is closed"),
			pageErrorView{status: 503, code: "home_page_db_recovering", retryable: true, retryAfter: "5"}},
		{"verrou d'ecriture", vivant, dblease.ErrDBLocked,
			pageErrorView{status: 503, code: "db_busy", retryable: true, retryAfter: "5"}},
		{"client parti pendant une bascule", annule, sharedprovider.ErrSwapTimeout,
			pageErrorView{status: 499, code: "client_closed", retryable: false}},
		{"erreur quelconque", vivant, errors.New("boom"),
			pageErrorView{status: 500, code: "home_page_error", retryable: true}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := viewPageError(t, homePageError(c.ctx, c.err, "gt")); got != c.want {
				t.Errorf("rendu = %+v, attendu %+v", got, c.want)
			}
		})
	}
}
