// cmd/film-profiles-build — produit la PART DERIVEE du catalogue des profils de film et
// l ecrit dans data/titles/{slug}/reference/film_profiles.json (PathResolver, D12).
//
// # CE QU IL PRODUIT, ET CE QU IL NE TOUCHE PAS
//
// Le catalogue des profils porte DEUX natures d entrees, et une seule est fabriquee :
//
//	DERIVEE  le bloc `derived` : l empreinte du catalogue des bornes de carte
//	         (`map_quant_bounds.json`), qui dit de quelle derivation des fichiers du jeu ce
//	         profil est solidaire. C EST TOUT CE QUE CET OUTIL ECRIT.
//	SAISIE   le bloc `entries` : ce qui vient de l executable ou d un film temoin, avec sa
//	         provenance. Cet outil le RECOPIE tel quel, il ne le fabrique jamais — une valeur
//	         de profil se relit chez l ecrivain ou se mesure sur un temoin, elle ne se
//	         regenere pas depuis du code (D3 du plan decodeur, docs/RUNBOOK_FILM_PROFILES.md).
//
// LES BORNES NE SONT PAS RECOPIEES. Elles ont deja leur catalogue, produit par
// `cmd/mapquant-build` depuis les .module du jeu ; le profil n en porte que l empreinte. Une
// seconde copie divergerait au premier build ajoute (regle 6 du depot).
//
// Usage : go run ./cmd/film-profiles-build [--title slug] [--check] [--out FILE] [--bounds FILE]
//
// `--check` n ecrit RIEN et sort en 1 si le fichier commis n est pas celui que cet outil
// produirait : c est la forme que prend le gate en integration.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/filmprofile"
)

// outilDesBornes : la commande qui PRODUIT le catalogue derive. Ecrite dans le bloc `derived`
// pour que le fichier dise lui-meme comment se rejoue sa part fabriquee.
const outilDesBornes = "CGO_ENABLED=1 go run ./cmd/mapquant-build"

// aProposParDefaut : ce que le fichier dit de lui-meme quand cet outil le cree.
const aProposParDefaut = "Catalogue des profils de film : ce que le depot sait de la grammaire " +
	"d un film, indexe par les TROIS clefs que le film ECRIT (format chunk_00+4, build de la " +
	"section 2, version majeure chunk_00+0). `derived` est fabrique par " +
	"`go run ./cmd/film-profiles-build` ; `entries` est SAISI, chaque ligne avec sa provenance " +
	"(relue / mesuree / presumee), sa preuve et sa date. Procedure : docs/RUNBOOK_FILM_PROFILES.md."

func main() {
	titleSlug := flag.String("title", title.DefaultSlug, "slug du titre")
	check := flag.Bool("check", false, "ne rien ecrire ; sortir en 1 si le fichier commis differe")
	out := flag.String("out", "", "fichier de sortie (defaut : PathResolver.FilmProfilesPath)")
	bornes := flag.String("bounds", "", "catalogue des bornes (defaut : PathResolver.MapQuantBoundsPath)")
	flag.Parse()

	if err := executer(*titleSlug, *check, *out, *bornes); err != nil {
		slog.Error("fabrication du catalogue des profils de film", "err", err, "titre", *titleSlug)
		os.Exit(1)
	}
}

// executer fait le travail et rend une erreur au lieu de sortir : c est ce qui le rend
// testable sans processus fils.
func executer(titleSlug string, check bool, out, bornes string) error {
	cheminProfils, cheminBornes, err := resoudreChemins(titleSlug, out, bornes)
	if err != nil {
		return err
	}

	blobBornes, err := os.ReadFile(cheminBornes) //nolint:gosec // chemin resolu par le PathResolver
	if err != nil {
		return fmt.Errorf("lecture du catalogue des bornes (%s) : %w", cheminBornes, err)
	}
	resume, err := filmprofile.ResumerBornesDeCarte(blobBornes)
	if err != nil {
		return fmt.Errorf("%s : %w", cheminBornes, err)
	}

	cat, err := catalogueExistantOuNeuf(cheminProfils, titleSlug)
	if err != nil {
		return err
	}
	cat.Derive = filmprofile.Derive{
		Catalogue:       filepath.Base(cheminBornes),
		SchemaVersion:   resume.SchemaVersion,
		Cartes:          resume.Cartes,
		EmpreinteCartes: resume.Empreinte,
		Outil:           outilDesBornes,
	}
	if err := cat.Valide(); err != nil {
		return fmt.Errorf("le catalogue produit n est pas valide, rien ecrit : %w", err)
	}

	blob, err := serialiser(cat)
	if err != nil {
		return err
	}

	if check {
		return verifier(cheminProfils, blob, resume)
	}
	if err := os.WriteFile(cheminProfils, blob, 0o644); err != nil { //nolint:gosec // donnee de reference versionnee, relue en revue
		return fmt.Errorf("ecriture (%s) : %w", cheminProfils, err)
	}
	slog.Info("catalogue des profils de film ecrit", "path", cheminProfils,
		"entrees", len(cat.Entrees), "cartes_derivees", resume.Cartes,
		"empreinte_bornes", resume.Empreinte)
	return nil
}

// serialiser rend le fichier tel qu il doit etre commis.
//
// SANS ECHAPPEMENT HTML : `json.Marshal` remplace les chevrons par leurs sequences `\uXXXX`,
// ce qui rendrait les clefs de version majeure (`majeure<=38`) illisibles dans un fichier que
// l on relit et que l on edite a la main. Le JSON reste valide dans les deux formes ; celle-ci
// se lit.
func serialiser(cat *filmprofile.Catalogue) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(cat); err != nil {
		return nil, fmt.Errorf("serialisation : %w", err)
	}
	return buf.Bytes(), nil
}

// resoudreChemins rend le chemin du catalogue des profils et celui des bornes. Les deux
// viennent du PathResolver sauf si l appelant les impose (D12 : aucun `filepath.Join` maison).
func resoudreChemins(titleSlug string, out, bornes string) (profils, bornesOut string, err error) {
	if out != "" && bornes != "" {
		return out, bornes, nil
	}
	racine, err := title.FindRepoRoot()
	if err != nil {
		return "", "", fmt.Errorf("racine du depot : %w", err)
	}
	res := title.NewPathResolver(racine)
	if out == "" {
		out = res.FilmProfilesPath(titleSlug)
	}
	if bornes == "" {
		bornes = res.MapQuantBoundsPath(titleSlug)
	}
	return out, bornes, nil
}

// catalogueExistantOuNeuf lit le catalogue commis pour en RECOPIER la part saisie, ou en cree
// un vide si le fichier n existe pas encore.
//
// Il le lit SANS le valider : le bloc derive qu on vient corriger peut etre perime, c est
// meme la raison d etre de cet outil. La validation se fait sur le resultat, avant l ecriture.
func catalogueExistantOuNeuf(chemin, titleSlug string) (*filmprofile.Catalogue, error) {
	blob, err := os.ReadFile(chemin) //nolint:gosec // chemin resolu par le PathResolver
	if errors.Is(err, os.ErrNotExist) {
		slog.Info("aucun catalogue commis — creation", "path", chemin)
		return &filmprofile.Catalogue{
			SchemaVersion: filmprofile.SchemaVersionCourante,
			TitleSlug:     titleSlug,
			APropos:       aProposParDefaut,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lecture du catalogue commis (%s) : %w", chemin, err)
	}
	cat, err := filmprofile.DecoderSansValider(blob)
	if err != nil {
		return nil, fmt.Errorf("%s : %w", chemin, err)
	}
	if cat.TitleSlug != titleSlug {
		return nil, fmt.Errorf("%s porte titleSlug %q, demande %q — un catalogue de reference "+
			"appartient a UN titre", chemin, cat.TitleSlug, titleSlug)
	}
	return cat, nil
}

// verifier compare le fichier commis a ce que cet outil produirait, sans rien ecrire.
func verifier(chemin string, attendu []byte, resume filmprofile.BornesResumees) error {
	commis, err := os.ReadFile(chemin) //nolint:gosec // chemin resolu par le PathResolver
	if err != nil {
		return fmt.Errorf("lecture du catalogue commis (%s) : %w", chemin, err)
	}
	if string(commis) != string(attendu) {
		return fmt.Errorf("%s n est PAS le fichier que cet outil produit (bornes : %d cartes, "+
			"empreinte %s) — rejouer `go run ./cmd/film-profiles-build`",
			chemin, resume.Cartes, resume.Empreinte)
	}
	slog.Info("catalogue des profils de film conforme", "path", chemin,
		"cartes_derivees", resume.Cartes, "empreinte_bornes", resume.Empreinte)
	return nil
}
