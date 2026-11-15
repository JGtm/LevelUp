package main

import (
	"context"
	"fmt"
	"image/png"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/mapfondsidecar"
)

// executeConvertir ecrit le WebP, met a jour le sidecar (champ `image`, D3) et supprime le PNG
// de chaque fond selectionne — mais REFUSE d'ecrire un fichier dont l'aller-retour n'est pas
// identique au bit pres (Etape 4 du plan). Cette Etape 0 ne l'execute pas sur `data/` : le mode
// existe, compile et est teste sur repertoire temporaire (main_test.go) ; l'execution reelle
// attend la decision D10.
func executeConvertir(ctx context.Context, fonds []fondPNG) {
	for _, f := range fonds {
		if err := convertitUnFond(ctx, f); err != nil {
			slog.ErrorContext(ctx, "mapfond-webp: conversion en echec", "fichier", f.chemin, "err", err)
			os.Exit(1)
		}
	}
	slog.InfoContext(ctx, "mapfond-webp: conversion terminee", "fichiers", len(fonds))
}

// convertitUnFond traite un fond : verification (obligatoire) puis, seulement si elle passe,
// ecriture du WebP, mise a jour du sidecar et suppression du PNG.
func convertitUnFond(ctx context.Context, f fondPNG) error {
	file, err := os.Open(f.chemin)
	if err != nil {
		return fmt.Errorf("ouverture: %w", err)
	}
	img, err := png.Decode(file)
	closeErr := file.Close()
	if err != nil {
		return fmt.Errorf("decodage png: %w", err)
	}
	if closeErr != nil {
		return fmt.Errorf("fermeture %s: %w", f.chemin, closeErr)
	}

	r, encode, err := allerRetourWebP(img)
	if err != nil {
		return err
	}
	if !r.Identique {
		return fmt.Errorf("aller-retour non identique au bit pres, refus d'ecrire: %s", f.chemin)
	}

	cheminWebP := strings.TrimSuffix(f.chemin, filepath.Ext(f.chemin)) + ".webp"
	if err := os.WriteFile(cheminWebP, encode, 0o644); err != nil {
		return fmt.Errorf("ecriture %s: %w", cheminWebP, err)
	}

	cheminSidecar, estPlan := sidecarDuFond(f.chemin)
	if err := metAJourSidecarImage(cheminSidecar, filepath.Base(cheminWebP), estPlan); err != nil {
		return err
	}

	if err := os.Remove(f.chemin); err != nil {
		return fmt.Errorf("suppression %s: %w", f.chemin, err)
	}

	slog.InfoContext(ctx, "mapfond-webp: fond converti",
		"fichier", filepath.Base(f.chemin), "octets_png", f.octets, "octets_webp", r.OctetsWebP)
	return nil
}

// sidecarDuFond rend le sidecar d'une image de fond et dit si c'est son image plan : l'image plan
// `<cle>.plan.png` partage le sidecar `<cle>.json` du fond et ne le met à jour que par son champ
// `planImage`.
func sidecarDuFond(chemin string) (sidecar string, estPlan bool) {
	dir, nom := filepath.Dir(chemin), filepath.Base(chemin)
	if cle, ok := mapfondsidecar.CleDeImagePlan(nom); ok {
		return filepath.Join(dir, cle+".json"), true
	}
	return strings.TrimSuffix(chemin, filepath.Ext(chemin)) + ".json", false
}

// metAJourSidecarImage relit le sidecar, remplace son champ `image` (ou `planImage` pour l'image
// plan) par nomImage (D3 : le format servi par l'image est une propriete de la donnee, pas du
// code) et le reecrit avec le meme style d'indentation que la cuisson (cmd/mapfond-build/cuisson.go),
// plan des murs compris (le decodeur de film ne le connait pas : `internal/mapfondsidecar`).
func metAJourSidecarImage(chemin, nomImage string, estPlan bool) error {
	meta, err := mapfondsidecar.Load(chemin)
	if err != nil {
		return fmt.Errorf("lecture sidecar %s: %w", chemin, err)
	}
	if estPlan {
		meta.PlanImage = nomImage
	} else {
		meta.Image = nomImage
	}
	if err := mapfondsidecar.Write(chemin, *meta); err != nil {
		return fmt.Errorf("ecriture sidecar %s: %w", chemin, err)
	}
	return nil
}
