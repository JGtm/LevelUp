// Package haloclient — appearance_inputs.go : accès aux champs BRUTS du payload
// /customization/appearance nécessaires au diagnostic apparence par composant
// (volet 2 du plan .ai/PLAN_DIAG_APPARENCE_ADMIN_2026-07.md, Lot F).
//
// GetSpartanCustomization ne renvoie que les URLs RÉSOLUES (bannière/emblème/
// backdrop) et jette l'EmblemPath + la ConfigurationId : or les primitives de
// diagnostic du Lot E (DiagnoseNameplate exige emblemPath+cfg,
// DiagnoseCustomizationImage exige un inventory path) en ont besoin. Ce fichier
// expose ces champs sans dupliquer le fetch/parse (réutilise
// fetchCustomizationAppearance).
package haloclient

import (
	"context"
	"strconv"
	"strings"
)

// AppearanceInputs porte les champs bruts du payload /customization/appearance
// requis par le diagnostic par composant (Lot F). Ce sont des ENTRÉES du
// diagnostic (chemins/identifiants), pas des URLs résolues : la résolution +
// verdict se fait via DiagnoseNameplate / DiagnoseCustomizationImage /
// DiagnoseServiceTag.
type AppearanceInputs struct {
	// ServiceTag : tag de service Spartan (ex. "ABC1"), diagnostiqué par
	// DiagnoseServiceTag (présent/absent).
	ServiceTag string
	// BannerImagePath : inventory path de la bannière quand l'API le fournit
	// directement (souvent vide → fallback nameplate dérivée de l'emblème).
	BannerImagePath string
	// BackdropImagePath : inventory path du backdrop (DiagnoseCustomizationImage).
	BackdropImagePath string
	// EmblemPath : inventory path de l'emblème — sert À LA FOIS l'image emblème
	// (DiagnoseCustomizationImage) et la résolution nameplate/bannière
	// (DiagnoseNameplate).
	EmblemPath string
	// EmblemConfigID : ConfigurationId de l'emblème (0 si absent → le resolver
	// nameplate va chercher la 1re cfg positive au CMS).
	EmblemConfigID int64
}

// FetchAppearanceInputs récupère /customization/appearance et retourne les champs
// bruts nécessaires au diagnostic apparence, SANS résoudre aucune image (les
// primitives Diagnose* s'en chargent, avec leur verdict). Retourne (nil, nil) si
// les tokens sont absents/insuffisants (401/403) ou si la réponse est vide — le
// service (Lot F) en dérive un verdict dégradé (transient/auth_required), jamais
// un 500.
func (c *HaloAPIClient) FetchAppearanceInputs(ctx context.Context, xuid string) (*AppearanceInputs, error) {
	appearance, err := c.fetchCustomizationAppearance(ctx, xuid)
	if err != nil {
		return nil, err
	}
	if appearance == nil {
		return nil, nil
	}
	cfg, _ := strconv.ParseInt(strings.TrimSpace(appearance.EmblemConfigurationID), 10, 64)
	return &AppearanceInputs{
		ServiceTag:        appearance.ServiceTag,
		BannerImagePath:   appearance.BannerImagePath,
		BackdropImagePath: appearance.BackdropImagePath,
		EmblemPath:        appearance.EmblemPath,
		EmblemConfigID:    cfg,
	}, nil
}
