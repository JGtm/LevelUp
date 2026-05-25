// Package service — career_live_partial_test.go : tests de PartialFromLive.
//
// Couvre TOUTES les combinaisons de retours API live possibles, avec une
// attention particulière sur les cas "un seul champ rendu" qui sont au cœur
// du fix de pollution (Phase 2/3 PLAN_V2 §5).
package service

import (
	"testing"

	syncpkg "levelup/go-api/internal/sync"
)

func TestPartialFromLive_BothNil(t *testing.T) {
	p := PartialFromLive(nil, nil)
	if !p.IsEmpty() {
		t.Fatalf("expected empty, got %+v", p)
	}
}

func TestPartialFromLive_ProgressNil_CustomFull(t *testing.T) {
	custom := &syncpkg.SpartanCustomizationData{
		SpartanID:        "OKLM",
		BannerImageURL:   "https://cdn/banner.png",
		EmblemImageURL:   "https://cdn/emblem.png",
		BackdropImageURL: "https://cdn/backdrop.png",
	}
	p := PartialFromLive(nil, custom)

	if p.Rank != nil || p.CurrentXP != nil || p.IsMaxRank != nil {
		t.Errorf("progress fields should be nil: %+v", p)
	}
	if p.SpartanID == nil || *p.SpartanID != "OKLM" {
		t.Errorf("SpartanID: got %v, want OKLM", p.SpartanID)
	}
	if p.BannerImageURL == nil || *p.BannerImageURL != "https://cdn/banner.png" {
		t.Errorf("BannerImageURL: got %v", p.BannerImageURL)
	}
	if p.EmblemImageURL == nil || *p.EmblemImageURL != "https://cdn/emblem.png" {
		t.Errorf("EmblemImageURL: got %v", p.EmblemImageURL)
	}
	if p.BackdropImageURL == nil || *p.BackdropImageURL != "https://cdn/backdrop.png" {
		t.Errorf("BackdropImageURL: got %v", p.BackdropImageURL)
	}
}

func TestPartialFromLive_ProgressFull_CustomNil(t *testing.T) {
	progress := &syncpkg.CareerRankData{
		CurrentRank:     184,
		CurrentXP:       1234,
		XPForNextRank:   5000,
		XPTotal:         100000,
		IsMaxRank:       false,
		CurrentRankName: "General 2 Platinum",
		CurrentRankTier: "Platinum",
		AdornmentPath:   "rank/184.png",
	}
	p := PartialFromLive(progress, nil)

	if p.Rank == nil || *p.Rank != 184 {
		t.Errorf("Rank: got %v, want 184", p.Rank)
	}
	if p.CurrentXP == nil || *p.CurrentXP != 1234 {
		t.Errorf("CurrentXP: got %v", p.CurrentXP)
	}
	if p.XPForNextRank == nil || *p.XPForNextRank != 5000 {
		t.Errorf("XPForNextRank: got %v", p.XPForNextRank)
	}
	if p.XPTotal == nil || *p.XPTotal != 100000 {
		t.Errorf("XPTotal: got %v", p.XPTotal)
	}
	if p.RankName == nil || *p.RankName != "General 2 Platinum" {
		t.Errorf("RankName: got %v", p.RankName)
	}
	if p.SpartanID != nil || p.BannerImageURL != nil ||
		p.EmblemImageURL != nil || p.BackdropImageURL != nil {
		t.Errorf("custom fields should be nil: %+v", p)
	}
}

// TestPartialFromLive_ProgressAllZero : API a renvoyé un objet vide (CurrentRank=0,
// CurrentXP=0, IsMaxRank=false). On n'écrit RIEN — clé pour éviter la pollution.
func TestPartialFromLive_ProgressAllZero(t *testing.T) {
	progress := &syncpkg.CareerRankData{
		CurrentRank: 0,
		CurrentXP:   0,
		IsMaxRank:   false,
	}
	p := PartialFromLive(progress, nil)

	if !p.IsEmpty() {
		t.Errorf("expected empty (API rendered nothing usable), got %+v", p)
	}
}

// TestPartialFromLive_ProgressRankOnly_XPZero : début de palier (CurrentXP=0
// explicite). On doit écrire Rank ET CurrentXP=0 (vraie valeur).
func TestPartialFromLive_ProgressRankOnly_XPZero(t *testing.T) {
	progress := &syncpkg.CareerRankData{
		CurrentRank: 184,
		CurrentXP:   0,
	}
	p := PartialFromLive(progress, nil)

	if p.Rank == nil || *p.Rank != 184 {
		t.Errorf("Rank should be 184: %v", p.Rank)
	}
	if p.CurrentXP == nil {
		t.Errorf("CurrentXP=0 should be persisted (real value, début palier)")
	} else if *p.CurrentXP != 0 {
		t.Errorf("CurrentXP: got %d, want 0", *p.CurrentXP)
	}
}

// TestPartialFromLive_IsMaxRankOnly : joueur au rang max, IsMaxRank=true même
// si CurrentRank=0 dans la payload (cas exotique mais possible).
func TestPartialFromLive_IsMaxRankOnly(t *testing.T) {
	progress := &syncpkg.CareerRankData{
		CurrentRank: 0,
		IsMaxRank:   true,
	}
	p := PartialFromLive(progress, nil)

	if p.IsMaxRank == nil || *p.IsMaxRank != true {
		t.Errorf("IsMaxRank should be true: %v", p.IsMaxRank)
	}
	if p.Rank != nil {
		t.Errorf("Rank should be nil when CurrentRank=0: %v", p.Rank)
	}
}

// TestPartialFromLive_CustomBannerOnly : API customisation rend uniquement la
// bannière, on n'écrit QUE la bannière.
func TestPartialFromLive_CustomBannerOnly(t *testing.T) {
	custom := &syncpkg.SpartanCustomizationData{
		BannerImageURL:   "https://cdn/new-banner.png",
		EmblemImageURL:   "",
		BackdropImageURL: "",
		SpartanID:        "",
	}
	p := PartialFromLive(nil, custom)

	if p.BannerImageURL == nil || *p.BannerImageURL != "https://cdn/new-banner.png" {
		t.Errorf("BannerImageURL should be set: %v", p.BannerImageURL)
	}
	if p.EmblemImageURL != nil {
		t.Errorf("EmblemImageURL should be nil: %v", p.EmblemImageURL)
	}
	if p.BackdropImageURL != nil {
		t.Errorf("BackdropImageURL should be nil: %v", p.BackdropImageURL)
	}
	if p.SpartanID != nil {
		t.Errorf("SpartanID should be nil: %v", p.SpartanID)
	}
}

func TestPartialFromLive_CustomWhitespaceFields(t *testing.T) {
	custom := &syncpkg.SpartanCustomizationData{
		BannerImageURL: "   ",
		EmblemImageURL: "\t\n",
		SpartanID:      "  ",
	}
	p := PartialFromLive(nil, custom)
	if !p.IsEmpty() {
		t.Errorf("whitespace-only fields should be ignored: %+v", p)
	}
}

// TestPartialFromLive_BothPartial : progress rank-only + custom emblem-only.
// L'INSERT doit contenir Rank+CurrentXP+EmblemImageURL ; tout le reste nil.
func TestPartialFromLive_BothPartial(t *testing.T) {
	progress := &syncpkg.CareerRankData{
		CurrentRank: 150,
		CurrentXP:   2500,
	}
	custom := &syncpkg.SpartanCustomizationData{
		EmblemImageURL: "https://cdn/emblem.png",
	}
	p := PartialFromLive(progress, custom)

	if p.Rank == nil || *p.Rank != 150 {
		t.Errorf("Rank: %v", p.Rank)
	}
	if p.CurrentXP == nil || *p.CurrentXP != 2500 {
		t.Errorf("CurrentXP: %v", p.CurrentXP)
	}
	if p.EmblemImageURL == nil || *p.EmblemImageURL != "https://cdn/emblem.png" {
		t.Errorf("EmblemImageURL: %v", p.EmblemImageURL)
	}
	if p.BannerImageURL != nil {
		t.Errorf("BannerImageURL should be nil: %v", p.BannerImageURL)
	}
	if p.SpartanID != nil {
		t.Errorf("SpartanID should be nil: %v", p.SpartanID)
	}
	if p.BackdropImageURL != nil {
		t.Errorf("BackdropImageURL should be nil: %v", p.BackdropImageURL)
	}
}

func TestPartialFromLive_CustomSpartanIDOverridesProgressLegacy(t *testing.T) {
	progress := &syncpkg.CareerRankData{
		CurrentRank: 100,
		SpartanID:   "OLD",
	}
	custom := &syncpkg.SpartanCustomizationData{
		SpartanID: "NEW",
	}
	p := PartialFromLive(progress, custom)

	if p.SpartanID == nil || *p.SpartanID != "NEW" {
		t.Errorf("SpartanID should be NEW (from custom): %v", p.SpartanID)
	}
}
