package main

import (
	"context"
	"fmt"
	"image/png"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// enteteTableau nomme les colonnes du tableau de mesure ecrit sur stdout par -verifier.
const enteteTableau = "fichier\toctets_png\toctets_webp\tgain_pct\tduree_encode\tduree_decode\tidentique"

// executeVerifier mesure l'aller-retour sur chaque fond SANS RIEN ECRIRE dans data/. Toute
// difference (Pix ou Rect) est une erreur fatale nommant le fichier (Etape 0 du plan) : on ne
// continue pas sur un fond dont l'aller-retour a menti, le chiffre cumule en serait faux.
func executeVerifier(ctx context.Context, fonds []fondPNG) {
	var totalPNG, totalWebP int64
	lignes := make([]string, 0, len(fonds))

	for _, f := range fonds {
		r, err := verifieUnFond(f)
		if err != nil {
			slog.ErrorContext(ctx, "mapfond-webp: aller-retour en echec", "fichier", f.chemin, "err", err)
			os.Exit(1)
		}
		if !r.Identique {
			slog.ErrorContext(ctx, "mapfond-webp: aller-retour NON identique au bit pres", "fichier", f.chemin)
			os.Exit(1)
		}

		gain := gainPct(int(f.octets), r.OctetsWebP)
		totalPNG += f.octets
		totalWebP += int64(r.OctetsWebP)
		slog.InfoContext(ctx, "mapfond-webp: fond verifie",
			"fichier", filepath.Base(f.chemin),
			"octets_png", f.octets, "octets_webp", r.OctetsWebP,
			"gain_pct", fmt.Sprintf("%.1f", gain),
			"duree_encode", r.DureeEncode, "duree_decode", r.DureeDecode,
			"identique", r.Identique)
		lignes = append(lignes, ligneTableau(f, r, gain))
	}

	imprimeTableau(lignes, totalPNG, totalWebP)
	gainCumule := gainPct(int(totalPNG), int(totalWebP))
	slog.InfoContext(ctx, "mapfond-webp: verification terminee",
		"fichiers", len(fonds), "octets_png", totalPNG, "octets_webp", totalWebP,
		"gain_cumule_pct", fmt.Sprintf("%.1f", gainCumule))
}

// verifieUnFond decode le PNG et rejoue l'aller-retour WebP — la meme logique que -convertir
// (roundtrip.go), sans jamais ecrire.
func verifieUnFond(f fondPNG) (resultatAllerRetour, error) {
	file, err := os.Open(f.chemin)
	if err != nil {
		return resultatAllerRetour{}, fmt.Errorf("ouverture: %w", err)
	}
	defer file.Close()

	img, err := png.Decode(file)
	if err != nil {
		return resultatAllerRetour{}, fmt.Errorf("decodage png: %w", err)
	}

	r, _, err := allerRetourWebP(img)
	return r, err
}

// imprimeTableau ecrit le tableau de mesure sur stdout : c'est le PRODUIT de -verifier (le
// commanditaire doit pouvoir le lire d'un coup), pas une trace de diagnostic — d'ou
// `fmt.Fprintf(os.Stdout, ...)` plutot que `slog`, conformement au contrat du lot.
func imprimeTableau(lignes []string, totalPNG, totalWebP int64) {
	fmt.Fprintln(os.Stdout, enteteTableau)
	for _, l := range lignes {
		fmt.Fprintln(os.Stdout, l)
	}
	gainCumule := gainPct(int(totalPNG), int(totalWebP))
	fmt.Fprintf(os.Stdout, "CUMUL\t%d\t%d\t%.1f\t-\t-\t-\n", totalPNG, totalWebP, gainCumule)
}

// ligneTableau met un resultat en une ligne TSV.
func ligneTableau(f fondPNG, r resultatAllerRetour, gain float64) string {
	ident := "oui"
	if !r.Identique {
		ident = "NON"
	}
	return strings.Join([]string{
		filepath.Base(f.chemin),
		fmt.Sprint(f.octets),
		fmt.Sprint(r.OctetsWebP),
		fmt.Sprintf("%.1f", gain),
		r.DureeEncode.String(),
		r.DureeDecode.String(),
		ident,
	}, "\t")
}
