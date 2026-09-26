// cmd/replay-diff — COMPARE DEUX ARTEFACTS DE REJEU, AXE PAR AXE, A TRAVERS LES SCHEMAS.
//
// # CE QU'IL REPOND
//
// « Entre l'artefact cuit au schema N et celui que le code d'aujourd'hui produit, qu'est-ce qui
// a ete PERDU ? » — la question qu'aucun test ne pose, parce qu'aucun test ne tient un artefact
// d'il y a quarante bumps de schema. Le parc local en porte 161, du schema 1 au schema 38 : ce
// sont les seuls temoins de ce que la cuisson savait faire a chaque epoque.
//
// # POURQUOI IL NE DESERIALISE PAS `replay.ReplayDocument`
//
// Le deserialiser JETTERAIT precisement ce qu'on cherche : `encoding/json` ignore en silence
// tout champ que la structure d'aujourd'hui ne declare plus. Un calque supprime entre le schema
// 20 et le 39 disparaitrait donc des DEUX cotes de la comparaison, et l'outil rendrait
// « aucune difference » sur la regression la plus grave qui soit. La lecture est donc GENERIQUE
// (`map[string]any`), et c'est la seule forme qui puisse voir un champ dont le code ne sait plus
// rien.
//
// # LE PRINCIPE : DES MESURES, PAS UNE EGALITE OCTET A OCTET
//
// Deux artefacts du meme match a deux epoques ne sont JAMAIS egaux (bornes affinees, ordre des
// tableaux, champs neufs). Comparer les octets ne dirait rien. L'outil reduit donc chaque
// artefact a une EMPREINTE : un jeu de mesures nommees (`objectifs/par-stat/flag_captures` = 3,
// `joueur/2533.../kills` = 12, `pistes/points` = 41 260...), et compare les mesures deux a deux —
// y compris la SOMME DES DUREES de chaque calque a intervalles [t0,t1] (ports, vies de vehicule,
// tractions de grappin...) : un rognage de bordure qui garde le NOMBRE d'elements intact reste
// invisible au seul comptage, jamais a la somme des durees.
//
// # LA TOLERANCE A L'EVOLUTION DU SCHEMA
//
// Une mesure ABSENTE DE L'ANCIEN et presente dans le nouveau est un GAIN (champ neuf), jamais une
// regression. L'inverse — presente dans l'ancien, absente ou plus basse dans le nouveau — est une
// PERTE, et c'est le seul signal que ce balayage cherche. Une valeur textuelle differente est un
// CHANGEMENT, a instruire au cas par cas.
//
// # OU VIT LA LOGIQUE
//
// Empreinte, Comparer et le rendu vivent dans `internal/replaydiff` — un `package main` ne
// s'importe pas, et `cmd/replay-corpus-gate` (le gate de non-regression sur corpus temoin,
// docs/COMMANDS.md) a besoin de la MEME comparaison sans dupliquer une ligne (CLAUDE.md n°6).
// Ce fichier n'est plus que l'enveloppe CLI historique : memes flags, meme sortie, zero
// changement de comportement.
//
// Usage :
//
//	replay-diff -ancien <a.json> -nouveau <b.json> [-json <sortie.json>] [-tout] [-quiet]
//
// Codes de sortie : 0 = compare (avec ou sans difference), 1 = erreur d'entree/sortie,
// 2 = usage. La PRESENCE de differences n'est pas un echec : c'est le resultat.
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"levelup/go-api/internal/replaydiff"
)

func main() {
	ancien := flag.String("ancien", "", "artefact de REFERENCE (le plus ancien) — obligatoire")
	nouveau := flag.String("nouveau", "", "artefact RE-CUIT au code courant — obligatoire")
	sortieJSON := flag.String("json", "", "fichier ou ecrire le rapport JSON de la paire (vide = aucun)")
	tout := flag.Bool("tout", false, "afficher AUSSI les mesures identiques (defaut : seules les differences)")
	quiet := flag.Bool("quiet", false, "n'ecrire aucun tableau sur la sortie standard (utile avec -json)")
	flag.Parse()
	if *ancien == "" || *nouveau == "" {
		fmt.Fprintln(os.Stderr,
			"usage: replay-diff -ancien <a.json> -nouveau <b.json> [-json <sortie.json>] [-tout] [-quiet]")
		os.Exit(2)
	}
	if err := executer(*ancien, *nouveau, *sortieJSON, *tout, *quiet); err != nil {
		slog.Error("replay-diff", "err", err)
		os.Exit(1)
	}
}

// executer lit les deux artefacts, calcule leurs empreintes, les compare et rend le rapport.
func executer(ancien, nouveau, sortieJSON string, tout, quiet bool) error {
	docA, err := replaydiff.LireDocument(ancien)
	if err != nil {
		return fmt.Errorf("artefact ancien : %w", err)
	}
	docB, err := replaydiff.LireDocument(nouveau)
	if err != nil {
		return fmt.Errorf("artefact nouveau : %w", err)
	}
	empA := replaydiff.Empreindre(docA)
	empB := replaydiff.Empreindre(docB)
	rap := replaydiff.Comparer(empA, empB)
	rap.FichierAncien = ancien
	rap.FichierNouveau = nouveau
	if !quiet {
		replaydiff.AfficherTableau(os.Stdout, rap, tout)
	}
	if sortieJSON != "" {
		if err := replaydiff.EcrireJSON(sortieJSON, rap); err != nil {
			return err
		}
	}
	return nil
}
