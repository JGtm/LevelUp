// Package domain — validate_test.go : tests table-driven pour les méthodes Validate().
//
// Sprint 52 B9 : couvre FilterContextInput.Validate(), MatchHistoryQueryRequest.Validate()
// et SyncOptions.Validate(). Aucune dépendance CGO — tests purs.
package domain_test

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// ── FilterContextInput.Validate() ────────────────────────────────────────────

func TestFilterContextInput_Validate(t *testing.T) {
	now := time.Now()
	later := now.Add(time.Hour)
	earlier := now.Add(-time.Hour)

	tests := []struct {
		name    string
		input   domain.FilterContextInput
		wantErr bool
	}{
		{
			name:    "zero value (défaut accepté)",
			input:   domain.FilterContextInput{},
			wantErr: false,
		},
		{
			name:    "filter_mode period valide",
			input:   domain.FilterContextInput{FilterMode: "period"},
			wantErr: false,
		},
		{
			name:    "filter_mode sessions valide",
			input:   domain.FilterContextInput{FilterMode: "sessions"},
			wantErr: false,
		},
		{
			name:    "filter_mode invalide",
			input:   domain.FilterContextInput{FilterMode: "weekly"},
			wantErr: true,
		},
		{
			name: "start_date < end_date (valide)",
			input: domain.FilterContextInput{
				Period: domain.PeriodInput{StartDate: &now, EndDate: &later},
			},
			wantErr: false,
		},
		{
			name: "start_date = end_date (invalide : pas strictement antérieure)",
			input: domain.FilterContextInput{
				Period: domain.PeriodInput{StartDate: &now, EndDate: &now},
			},
			wantErr: true,
		},
		{
			name: "start_date > end_date (invalide)",
			input: domain.FilterContextInput{
				Period: domain.PeriodInput{StartDate: &later, EndDate: &earlier},
			},
			wantErr: true,
		},
		{
			name: "end_date seul (valide : start_date absent)",
			input: domain.FilterContextInput{
				Period: domain.PeriodInput{EndDate: &later},
			},
			wantErr: false,
		},
		{
			name: "gap_minutes négatif (invalide)",
			input: domain.FilterContextInput{
				Sessions: domain.SessionsFilter{GapMinutes: -1},
			},
			wantErr: true,
		},
		{
			name: "gap_minutes zéro (valide)",
			input: domain.FilterContextInput{
				Sessions: domain.SessionsFilter{GapMinutes: 0},
			},
			wantErr: false,
		},
		{
			name: "gap_minutes positif (valide)",
			input: domain.FilterContextInput{
				Sessions: domain.SessionsFilter{GapMinutes: 30},
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.input.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() erreur = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

// ── MatchHistoryQueryRequest.Validate() ──────────────────────────────────────

func TestMatchHistoryQueryRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     domain.MatchHistoryQueryRequest
		wantErr bool
	}{
		{
			name:    "zero value (page=0 et pageSize=0 acceptés comme defaults)",
			req:     domain.MatchHistoryQueryRequest{},
			wantErr: false,
		},
		{
			name: "page=1, pageSize=20 (valide)",
			req: domain.MatchHistoryQueryRequest{
				Pagination: domain.PaginationRequest{Page: 1, PageSize: 20},
			},
			wantErr: false,
		},
		{
			name: "page négatif (invalide)",
			req: domain.MatchHistoryQueryRequest{
				Pagination: domain.PaginationRequest{Page: -1, PageSize: 20},
			},
			wantErr: true,
		},
		{
			name: "pageSize négatif (invalide)",
			req: domain.MatchHistoryQueryRequest{
				Pagination: domain.PaginationRequest{Page: 1, PageSize: -5},
			},
			wantErr: true,
		},
		{
			name: "pageSize trop grand (invalide)",
			req: domain.MatchHistoryQueryRequest{
				Pagination: domain.PaginationRequest{Page: 1, PageSize: 10001},
			},
			wantErr: true,
		},
		{
			name: "pageSize=10000 (limite max valide)",
			req: domain.MatchHistoryQueryRequest{
				Pagination: domain.PaginationRequest{Page: 1, PageSize: 10000},
			},
			wantErr: false,
		},
		{
			name: "filter_mode invalide propagé depuis Filters.Validate()",
			req: domain.MatchHistoryQueryRequest{
				Filters:    domain.FilterContextInput{FilterMode: "invalid_mode"},
				Pagination: domain.PaginationRequest{Page: 1, PageSize: 20},
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() erreur = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

// ── SyncOptions.Validate() ───────────────────────────────────────────────────

func TestSyncOptions_Validate(t *testing.T) {
	tests := []struct {
		name    string
		opts    domain.SyncOptions
		wantErr bool
	}{
		{
			name:    "options par défaut (valides)",
			opts:    domain.DefaultSyncOptions(),
			wantErr: false,
		},
		{
			name:    "MatchType vide (invalide)",
			opts:    domain.SyncOptions{MatchType: "", MaxMatches: 100},
			wantErr: true,
		},
		{
			name:    "MatchType invalide",
			opts:    domain.SyncOptions{MatchType: "ranked", MaxMatches: 100},
			wantErr: true,
		},
		{
			name:    "MatchType all (valide)",
			opts:    domain.SyncOptions{MatchType: "all", MaxMatches: 100},
			wantErr: false,
		},
		{
			name:    "MatchType matchmaking (valide)",
			opts:    domain.SyncOptions{MatchType: "matchmaking", MaxMatches: 50},
			wantErr: false,
		},
		{
			name:    "MatchType custom (valide)",
			opts:    domain.SyncOptions{MatchType: "custom", MaxMatches: 50},
			wantErr: false,
		},
		{
			name:    "MatchType local (valide)",
			opts:    domain.SyncOptions{MatchType: "local", MaxMatches: 50},
			wantErr: false,
		},
		{
			name:    "MaxMatches négatif (invalide)",
			opts:    domain.SyncOptions{MatchType: "all", MaxMatches: -1},
			wantErr: true,
		},
		{
			name:    "RequestsPerSecond négatif (invalide)",
			opts:    domain.SyncOptions{MatchType: "all", MaxMatches: 100, RequestsPerSecond: -1},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.opts.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() erreur = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}
