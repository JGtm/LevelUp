// Package domain — career_live.go : DTOs du flow live carrière (XP/identité
// Spartan), partagés entre la couche duckdb (lecture/écriture career_progression)
// et le service CareerLiveService.
//
// Relocalisés ici depuis internal/platform/duckdb (Phase 2 canonical) pour que le
// service ne dépende plus du package duckdb : ce sont des structures de données
// neutres (aucun type DB), naturellement à leur place dans domain. duckdb conserve
// des alias (`type CareerRankRow = domain.CareerRankRow`) pour le code interne.
package domain

// CareerRankRow est la projection raw d'une ligne `career_progression` après
// per-field merge (dernière valeur non-vide par colonne via ARG_MAX + FILTER).
// Les URLs d'images sont les valeurs absolues stockées en DB (résolues à la
// sync précédente) — la construction des URLs d'asset internes (/api/v1/...)
// reste l'affaire du service consommateur.
type CareerRankRow struct {
	Rank             int
	RankName         string
	RankTier         string
	CurrentXP        int
	XPForNextRank    int
	XPTotal          int
	IsMaxRank        bool
	SpartanID        string
	BannerImageURL   string
	EmblemImageURL   string
	BackdropImageURL string
	AdornmentPath    string
}

// IsEmpty retourne true si la row ne porte aucune donnée exploitable
// (utilisé pour décider si on retourne nil au caller).
func (r *CareerRankRow) IsEmpty() bool {
	if r == nil {
		return true
	}
	return r.Rank <= 0 &&
		r.CurrentXP == 0 &&
		r.SpartanID == "" &&
		r.BannerImageURL == "" &&
		r.EmblemImageURL == "" &&
		r.BackdropImageURL == ""
}

// CareerProgressionPartial = sous-ensemble des colonnes de career_progression.
// Chaque champ pointer = nil (non renseigné par l'API) ou set (rendu non-vide).
// IsEmpty() permet de skip un INSERT bidon.
type CareerProgressionPartial struct {
	Rank             *int
	CurrentXP        *int
	XPForNextRank    *int
	XPTotal          *int
	IsMaxRank        *bool
	RankName         *string
	RankTier         *string
	SpartanID        *string
	BannerImageURL   *string
	EmblemImageURL   *string
	BackdropImageURL *string
	AdornmentPath    *string
	// LastFetchStatus (Phase 6 PLAN_V2) trace l'issue du dernier fetch live.
	// Valeurs : "ok" / "api_empty" / "forbidden_403" / "auth_missing" / "failed".
	// Set même quand tous les autres champs sont nil (signal "j'ai essayé").
	LastFetchStatus *string
}

// IsEmpty retourne true si aucun champ de DATA n'est set. Le LastFetchStatus
// est ignoré ici : un partial avec status="forbidden_403" et tout le reste nil
// est considéré "empty data" mais on l'écrira quand même pour tracer la
// tentative (cf. HasOnlyStatus).
func (p *CareerProgressionPartial) IsEmpty() bool {
	if p == nil {
		return true
	}
	return p.Rank == nil && p.CurrentXP == nil && p.XPForNextRank == nil &&
		p.XPTotal == nil && p.IsMaxRank == nil &&
		p.RankName == nil && p.RankTier == nil &&
		p.SpartanID == nil &&
		p.BannerImageURL == nil && p.EmblemImageURL == nil &&
		p.BackdropImageURL == nil && p.AdornmentPath == nil
}

// HasOnlyStatus retourne true si seul LastFetchStatus est set (data vide).
// Sert au caller à décider s'il insère quand même pour tracer la tentative.
func (p *CareerProgressionPartial) HasOnlyStatus() bool {
	if p == nil || p.LastFetchStatus == nil {
		return false
	}
	return p.IsEmpty()
}

// MatchesLast retourne true si tous les champs set de p ont déjà la même valeur
// dans last. Permet de skip un INSERT redondant qui n'apporterait rien.
// Si last est nil, on considère qu'il y a forcément du nouveau (sauf si p est empty).
func (p *CareerProgressionPartial) MatchesLast(last *CareerRankRow) bool {
	if p == nil || p.IsEmpty() {
		return true
	}
	if last == nil {
		return false
	}
	if p.Rank != nil && *p.Rank != last.Rank {
		return false
	}
	if p.CurrentXP != nil && *p.CurrentXP != last.CurrentXP {
		return false
	}
	if p.XPForNextRank != nil && *p.XPForNextRank != last.XPForNextRank {
		return false
	}
	if p.XPTotal != nil && *p.XPTotal != last.XPTotal {
		return false
	}
	if p.IsMaxRank != nil && *p.IsMaxRank != last.IsMaxRank {
		return false
	}
	if p.SpartanID != nil && *p.SpartanID != last.SpartanID {
		return false
	}
	if p.BannerImageURL != nil && *p.BannerImageURL != last.BannerImageURL {
		return false
	}
	if p.EmblemImageURL != nil && *p.EmblemImageURL != last.EmblemImageURL {
		return false
	}
	if p.BackdropImageURL != nil && *p.BackdropImageURL != last.BackdropImageURL {
		return false
	}
	if p.AdornmentPath != nil && *p.AdornmentPath != last.AdornmentPath {
		return false
	}
	if p.RankName != nil && *p.RankName != last.RankName {
		return false
	}
	if p.RankTier != nil && *p.RankTier != last.RankTier {
		return false
	}
	return true
}

// CareerRankSnapshot est le DTO brut d'un snapshot de rang carrière live (fetch Economy
// API : rang courant, XP, IsMaxRank + images d'identité Spartan). DISTINCT de
// CareerRankData (transfert calculé de progression, career.go) et de CareerRankRow
// (projection DB ci-dessus). Promu depuis internal/sync (K1k) pour que la couche service
// et l'interface CareerFetcher ne dépendent plus de sync pour ce type ; sync garde
// l'alias `type CareerRankData = domain.CareerRankSnapshot`.
type CareerRankSnapshot struct {
	XUID             string
	CurrentRank      int
	CurrentRankName  string
	CurrentRankTier  string
	CurrentXP        int
	XPForNextRank    int
	XPTotal          int
	IsMaxRank        bool
	AdornmentPath    string
	SpartanID        string
	BannerImageURL   string
	EmblemImageURL   string
	BackdropImageURL string
}

// SpartanCustomizationData : identité visuelle Spartan (ServiceTag via SpartanID +
// images banner/emblem/backdrop) résolue depuis /customization?view=public. URLs vides
// si le resolve GameCMS a échoué. Promu depuis internal/sync (K1k) ; sync garde l'alias.
type SpartanCustomizationData struct {
	SpartanID        string
	BannerImageURL   string
	EmblemImageURL   string
	BackdropImageURL string
}
