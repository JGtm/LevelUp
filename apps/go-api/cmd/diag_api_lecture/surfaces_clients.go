package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"

	"levelup/go-api/internal/assets"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/mapcatalog"
	"levelup/go-api/internal/platform/auth"
	syncpkg "levelup/go-api/internal/sync"
)

// Les quatre surfaces ci-dessous lisent les clients HTTP Halo/Xbox qui ne passent pas par
// l'adaptateur `haloadapter` : Halo 5, Discovery UGC des cartes, actifs GameCMS, succès Xbox. Elles
// sont le témoin T3 du plan PUBLICATION (`.ai/V8/PLAN_PUBLICATION_BRIQUES_2026-10-30.md`).

// requetesH5ParSeconde : débit du client Halo 5 (le même que les autres lectures du diagnostic).
const requetesH5ParSeconde = 3

// modeServiceRecordH5 : le mode de service record que l'adaptateur Halo 5 consomme
// (`games/halo_5/adapter_data.go`, h5RecordModeArena).
const modeServiceRecordH5 = "arena"

// langueSucces : langue demandée pour les noms et descriptions des succès (valeur du protocole).
const langueSucces = "en-US"

// lireServiceRecordH5 écrit le service record arena du compte tel que le client Halo 5 le décode.
// Le jeton Spartan v4 qui sert Halo Infinite sert aussi Halo 5.
func lireServiceRecordH5(s *session, _ []string) error {
	rec, err := halo5.NewClient(s.tokens.SpartanToken, requetesH5ParSeconde).
		GetServiceRecords(s.ctx, s.gamertag, modeServiceRecordH5)
	if err != nil {
		return err
	}
	return s.ecrireJSON(rec)
}

// lireCarteUGC écrit le DTO Discovery d'une carte (jamais le fichier .mvar). Sans `version_id` :
// dernière version publiée.
func lireCarteUGC(s *session, args []string) error {
	version := ""
	if len(args) > 1 {
		version = args[1]
	}
	asset, err := mapcatalog.NewClient(s.tokens).FetchAsset(s.ctx, args[0], version)
	if err != nil {
		return err
	}
	return s.ecrireJSON(asset)
}

// empreinteActif : la forme écrite pour un actif GameCMS (la longueur et le SHA-256, jamais les octets).
type empreinteActif struct {
	Longueur int    `json:"longueur"`
	SHA256   string `json:"sha256"`
}

// lireActifGameCMS télécharge un actif binaire public par le chemin donné (construction d'URL
// d'image générique de GameCMS) et en écrit la longueur et le SHA-256.
func lireActifGameCMS(s *session, args []string) error {
	jetons := func(context.Context) (*domain.HaloTokens, error) { return s.tokens, nil }
	payload, err := assets.NewGameCMSFetcher(http.DefaultClient, jetons, "").
		Fetch(s.ctx, assets.Ref{Kind: assets.KindBPTrackImage, TitleID: title.DefaultSlug, ID: args[0]})
	if err != nil {
		return err
	}
	bin, ok := payload.(assets.BinaryPayload)
	if !ok {
		return fmt.Errorf("charge utile inattendue %T", payload)
	}
	somme := sha256.Sum256(bin.Bytes)
	return s.ecrireJSON(empreinteActif{Longueur: len(bin.Bytes), SHA256: hex.EncodeToString(somme[:])})
}

// lireSuccesXbox écrit la liste des succès Halo Infinite du compte (toutes les pages). Même chemin
// que la passe de succès de la synchronisation : access_token par le magasin d'abord (ADR 0023),
// XSTS Xbox Live, client de `sync`. La porte des jetons morts n'est pas consultée : la lecture
// n'enregistre aucun état.
func lireSuccesXbox(s *session, _ []string) error {
	accessToken, err := auth.ResolveMSAccessTokenStoreFirst(s.ctx, auth.NewSISUProvider(), s.store, s.xuid, s.gamertag)
	if err != nil {
		return fmt.Errorf("access_token Xbox Live: %w", err)
	}
	if accessToken == "" {
		return fmt.Errorf("aucun access_token Xbox Live pour %s", s.gamertag)
	}
	xsts, err := auth.AcquireXSTSForRTA(s.ctx, accessToken)
	if err != nil {
		return fmt.Errorf("XSTS Xbox Live: %w", err)
	}
	succes, err := syncpkg.NewXboxHTTPClient(xsts, title.XboxTitleIDFor(title.DefaultSlug)).
		GetPlayerAchievements(s.ctx, s.xuid, langueSucces)
	if err != nil {
		return err
	}
	return s.ecrireJSON(succes)
}
