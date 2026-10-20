// cmd/equipenergy-build — produit le catalogue de l ENERGIE DES EQUIPEMENTS a partir des tags `eqip`
// et `sofa` du jeu installe, et l ecrit dans data/titles/{slug}/reference/equipment_energy.json
// (PathResolver).
//
// Le film ne porte que l energie d un objet d equipement porte (i24) ; le nombre d usages qu elle
// represente depend de la capacite de l objet, une constante du tag. Cet outil la lit, avec
// l energie initiale des deux objets que chaque fiche `sofa` relie (`himap.LireLesEnergiesDEquipement`).
// Aucune valeur n est saisie : le fichier se regenere depuis le jeu, et `--check` dit s il differe.
//
// Usage : CGO_ENABLED=1 go run ./cmd/equipenergy-build [--title slug] [--out FILE] [--check]
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
	"levelup/go-api/internal/games/halo_infinite/equipenergy"
	"levelup/go-api/internal/himap"
)

// outil est la commande qui produit le catalogue, ecrite dans le fichier.
const outil = "CGO_ENABLED=1 go run ./cmd/equipenergy-build"

// aPropos dit ce que le fichier porte.
const aPropos = "Energie des equipements, lue dans les tags du jeu installe : par fiche `sofa`, l objet " +
	"ramassable (pickup), l objet porte (carried), leurs energies initiales et la capacite de l objet " +
	"porte. Usages restants d un objet porte = trunc((energie + 1e-4) / -delta) quand threshold > 0 " +
	"(FUN_1407e9384). Fabrique par l outil, jamais saisi."

func main() {
	titleSlug := flag.String("title", title.DefaultSlug, "slug du titre")
	out := flag.String("out", "", "fichier de sortie (defaut : PathResolver.EquipmentEnergyPath)")
	check := flag.Bool("check", false, "ne rien ecrire ; sortir en 1 si le fichier commis differe")
	flag.Parse()
	if err := executer(*titleSlug, *out, *check); err != nil {
		slog.Error("fabrication du catalogue de l energie des equipements", "err", err, "titre", *titleSlug)
		os.Exit(1)
	}
}

// errDifferent : le fichier commis n est pas celui que l outil produit (`--check`).
var errDifferent = errors.New("le catalogue commis differe de celui que l outil produit")

func executer(titleSlug, out string, check bool) error {
	if out == "" {
		root, err := title.FindRepoRoot()
		if err != nil {
			return fmt.Errorf("racine du depot : %w", err)
		}
		out = title.NewPathResolver(root).EquipmentEnergyPath(titleSlug)
	}
	deploy, err := himap.DeployRoot()
	if err != nil {
		return err
	}
	paires, err := himap.LireLesEnergiesDEquipement(deploy)
	if err != nil {
		return err
	}
	blob, err := json.MarshalIndent(catalogue(titleSlug, paires), "", "  ")
	if err != nil {
		return err
	}
	blob = append(blob, '\n')
	if check {
		commis, err := os.ReadFile(out) //nolint:gosec // chemin resolu par le PathResolver
		if err != nil {
			return err
		}
		if !bytes.Equal(commis, blob) {
			return errDifferent
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(out, blob, 0o644); err != nil { //nolint:gosec // donnee de reference versionnee
		return err
	}
	slog.Info("catalogue de l energie des equipements ecrit", "chemin", out, "fiches", len(paires))
	return nil
}

// catalogue met les paires lues dans la forme du fichier.
func catalogue(titleSlug string, paires []himap.EnergieDEquipement) equipenergy.Catalog {
	c := equipenergy.Catalog{
		SchemaVersion: equipenergy.SchemaVersion, TitleSlug: titleSlug, Tool: outil, About: aPropos,
		Offsets: map[string]string{
			"pickup_energy":  fmt.Sprintf("eqip+%#x (f32), copiee par FUN_140ea3f68", himap.EqipOffsetEnergieInitiale),
			"carried_energy": fmt.Sprintf("eqip+%#x (f32), copiee par FUN_140ea3f68", himap.EqipOffsetEnergieInitiale),
			"threshold": fmt.Sprintf("eqip+%#x (bloc, element de 0x344, FUN_14049a384) +%#x (f32), lu par FUN_1407e9384",
				himap.EqipOffsetCapacites, himap.CapaciteOffsetSeuil),
			"delta": fmt.Sprintf("eqip+%#x (bloc) +%#x (f32), ajoute a l energie par FUN_143181708",
				himap.EqipOffsetCapacites, himap.CapaciteOffsetVariation),
		},
	}
	for _, p := range paires {
		c.Entries = append(c.Entries, equipenergy.Entry{
			Sofa: fmt.Sprintf("0x%08x", p.Sofa), Pickup: fmt.Sprintf("0x%08x", p.Ramassable),
			Carried: fmt.Sprintf("0x%08x", p.Porte), PickupEnergy: p.EnergieRamassable, CarriedEnergy: p.EnergiePorte,
			Threshold: p.Seuil, Delta: p.Variation, Module: p.Module,
		})
	}
	return c
}
