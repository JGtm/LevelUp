// Package handlers — assets.go : handler cache-aside pour les assets visuels Waypoint.
//
// Sprint 54 D1.6 / D2.5 :
//
//	GET /api/v1/assets/medals/{title_id}/{medal_id}/image
//
// Pattern : check registry DuckDB → fetch Waypoint si absent → stockage local → réponse.
// Le frontend ne sait pas si l'image vient du cache ou du réseau.
// Singleflight par (title_id, medal_id) pour éviter N fetches parallèles sur le même asset.
package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/sync/singleflight"

	"levelup/go-api/internal/platform/duckdb"
)

var medalSFGroup singleflight.Group
var mapSFGroup singleflight.Group // cache-aside maps (Phase 2)

// MedalImageRepo est l'interface minimale requise par le handler (testable par injection).
type MedalImageRepo interface {
	GetMedalImageCache(ctx context.Context, titleID string, medalID int64) (*duckdb.MedalImageEntry, error)
	UpsertMedalImageCache(ctx context.Context, e duckdb.MedalImageEntry) error
}

// MapImageRepo est l'interface minimale pour le cache-aside des images de maps.
type MapImageRepo interface {
	GetMapImageCache(ctx context.Context, titleID string, mapID string) (*duckdb.MapImageEntry, error)
	UpsertMapImageCache(ctx context.Context, e duckdb.MapImageEntry) error
}

// AssetHandler gère le cache-aside des images de médailles et assets Waypoint.
// Utilise un MetadataRepo concret ou toute implémentation de MedalImageRepo/MapImageRepo.
type AssetHandler struct {
	repo    MedalImageRepo
	mapRepo MapImageRepo
}

// NewAssetHandler crée un AssetHandler depuis un MetadataRepo réel.
func NewAssetHandler(repo *duckdb.MetadataRepo) *AssetHandler {
	return &AssetHandler{repo: repo, mapRepo: repo}
}

// NewAssetHandlerWithRepo crée un AssetHandler depuis une interface (pour les tests).
func NewAssetHandlerWithRepo(repo MedalImageRepo) *AssetHandler {
	return &AssetHandler{repo: repo}
}

// GetMedalImage sert l'image d'une médaille depuis le cache local ou Waypoint.
// GET /api/v1/assets/medals/{title_id}/{medal_id}/image
func (h *AssetHandler) GetMedalImage(w http.ResponseWriter, r *http.Request) {
	titleID := chi.URLParam(r, "title_id")
	medalIDStr := chi.URLParam(r, "medal_id")

	medalID, err := strconv.ParseInt(medalIDStr, 10, 64)
	if err != nil {
		http.Error(w, "medal_id invalide", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	entry, err := h.repo.GetMedalImageCache(ctx, titleID, medalID)
	if err != nil {
		http.Error(w, "erreur registry", http.StatusInternalServerError)
		return
	}

	// Cache hit : redirection directe, aucun appel Waypoint.
	if entry != nil && entry.ImageURL != "" {
		http.Redirect(w, r, entry.ImageURL, http.StatusFound)
		return
	}

	// Cache miss : fetch Waypoint via singleflight (anti-doublon si N requêtes simultanées).
	sfKey := fmt.Sprintf("%s:%d", titleID, medalID)
	imgURLRaw, err, _ := medalSFGroup.Do(sfKey, func() (any, error) {
		return fetchMedalImageURL(titleID, medalID)
	})
	if err != nil {
		http.Error(w, "fetch image Waypoint échoué", http.StatusBadGateway)
		return
	}

	imgURL, _ := imgURLRaw.(string)

	_ = h.repo.UpsertMedalImageCache(ctx, duckdb.MedalImageEntry{
		TitleID:     titleID,
		MedalID:     medalID,
		ImageURL:    imgURL,
		LocalPath:   "",
		FetchedAt:   time.Now().UTC(),
		ContentHash: sha256Hex([]byte(imgURL)),
	})

	http.Redirect(w, r, imgURL, http.StatusFound)
}

// fetchMedalImageURL tente de résoudre l'URL d'image individuelle d'une médaille.
// Waypoint expose des spritesheets — si l'URL individuelle est introuvable (404),
// on retourne l'URL du spritesheet global ; le frontend découpe via sprite_index.
func fetchMedalImageURL(titleID string, medalID int64) (string, error) {
	url := fmt.Sprintf(
		"https://gamecms-hacs.svc.halowaypoint.com/hi/Progression/file/medals/%s/%d.png",
		titleID, medalID,
	)

	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return "", fmt.Errorf("GET medal image: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusNotFound {
		url = fmt.Sprintf(
			"https://gamecms-hacs.svc.halowaypoint.com/hi/Progression/file/medals/sprites/%s.png",
			titleID,
		)
	}
	return url, nil
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// GetMapImage sert l'image d'une map depuis le cache local ou Waypoint.
// GET /api/v1/assets/maps/{title_id}/{map_id}/image
func (h *AssetHandler) GetMapImage(w http.ResponseWriter, r *http.Request) {
	titleID := chi.URLParam(r, "title_id")
	mapID := chi.URLParam(r, "map_id")

	if mapID == "" {
		http.Error(w, "map_id requis", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	entry, err := h.mapRepo.GetMapImageCache(ctx, titleID, mapID)
	if err != nil {
		http.Error(w, "erreur registry", http.StatusInternalServerError)
		return
	}

	// Cache hit : redirection directe (local_path statique ou URL Waypoint).
	if entry != nil && entry.ImageURL != "" {
		http.Redirect(w, r, entry.ImageURL, http.StatusFound)
		return
	}

	// Cache miss : fetch Waypoint via singleflight.
	sfKey := fmt.Sprintf("map:%s:%s", titleID, mapID)
	imgURLRaw, err, _ := mapSFGroup.Do(sfKey, func() (any, error) {
		return fetchMapImageURL(titleID, mapID)
	})
	if err != nil {
		http.Error(w, "fetch image Waypoint échoué", http.StatusBadGateway)
		return
	}

	imgURL, _ := imgURLRaw.(string)

	_ = h.mapRepo.UpsertMapImageCache(ctx, duckdb.MapImageEntry{
		TitleID:     titleID,
		MapID:       mapID,
		ImageURL:    imgURL,
		LocalPath:   "",
		FetchedAt:   time.Now().UTC(),
		ContentHash: sha256Hex([]byte(imgURL)),
	})

	http.Redirect(w, r, imgURL, http.StatusFound)
}

// fetchMapImageURL tente de résoudre l'URL d'image d'une map depuis Waypoint.
// Pattern URL : https://gamecms-hacs.svc.halowaypoint.com/hi/images/map-variants/{title_id}/{map_id}.png
// Si 404, retourne URL placeholder ou fallback.
func fetchMapImageURL(titleID string, mapID string) (string, error) {
	// URL standard Waypoint pour les images de maps
	url := fmt.Sprintf(
		"https://gamecms-hacs.svc.halowaypoint.com/hi/images/map-variants/%s/%s.png",
		titleID, mapID,
	)

	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return "", fmt.Errorf("GET map image: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusNotFound {
		// Fallback : retourner une URL de placeholder ou essayer un autre endpoint.
		// Pour l'instant, retourner l'URL même si 404 (frontend gérera le fallback).
		return url, fmt.Errorf("map image not found on Waypoint: %s", mapID)
	}

	return url, nil
}
