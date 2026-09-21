package domain

// pad_tiers_web_parity_test.go — LE VOCABULAIRE DES NIVEAUX EST LE MEME DES DEUX COTES.
//
// # POURQUOI UN TEST GO QUI LIT DU TYPESCRIPT
//
// Les valeurs de niveau voyagent en clair dans le contrat (`tier: "puissance"`), donc le web
// les compare a des litteraux. Le generateur de types ne les porte pas comme une enumeration :
// rien, cote TypeScript, ne rougit si le Go renomme un niveau. Or c est exactement la mutation
// qui coute le plus cher — `PadTierGround` de "terrain" a "sol" laisse tout vert et fait
// DISPARAITRE un niveau entier des trois pages, en silence (constat de revue, 2026-09-14).
//
// Ce test confronte donc la liste Go a la liste TypeScript, la ou elle est ecrite. Il n est pas
// elegant ; il est le seul endroit d ou l on peut voir les deux a la fois.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// cheminModeleWeb : le module web qui porte l ordre de lecture des niveaux.
func cheminModeleWeb() string {
	return filepath.Join("..", "..", "..", "web", "src", "features", "_shared", "usage",
		"usagePadTiersModel.ts")
}

func TestPadTierOrder_MemeVocabulaireCoteWeb(t *testing.T) {
	src, err := os.ReadFile(cheminModeleWeb()) //nolint:gosec // chemin fige du depot
	if err != nil {
		t.Fatalf("modele web illisible (%s) : %v — le garde-rail ne garderait rien", cheminModeleWeb(), err)
	}
	txt := string(src)
	debut := strings.Index(txt, "USAGE_PAD_TIER_ORDER = [")
	if debut < 0 {
		t.Fatal("USAGE_PAD_TIER_ORDER introuvable dans le modele web — s'il a ete renomme, " +
			"mettre ce garde-rail a jour dans le meme commit")
	}
	fin := strings.Index(txt[debut:], "]")
	if fin < 0 {
		t.Fatal("liste USAGE_PAD_TIER_ORDER non fermee")
	}
	liste := txt[debut+len("USAGE_PAD_TIER_ORDER = [") : debut+fin]

	// Niveaux que le web NE REND PAS, par decision et non par oubli (2026-09-21, ajustements
	// pre-v7.5) : les socles de bonus (camouflage, surbouclier) sont des EQUIPEMENTS et se
	// lisent dans « Usages d equipement » ; « non identifie » n a pas de ligne, son compte
	// passe dans l infobulle du titre. Le contrat Go continue de les servir : un nouveau
	// niveau Go qui n est ni ici ni cote web fait toujours rougir ce test.
	masquesParDecision := map[string]bool{PadTierPowerup: true, PadTierUnclassified: true}

	for _, tier := range PadTierOrder {
		if masquesParDecision[tier] {
			continue
		}
		if !strings.Contains(liste, "'"+tier+"'") {
			t.Errorf("le niveau %q du contrat Go est ABSENT de USAGE_PAD_TIER_ORDER cote web "+
				"(%s) — il ne s'afficherait sur aucune des trois pages", tier, liste)
		}
	}
	// Et l inverse : un niveau que le web attend et que le Go ne sert plus resterait vide.
	for _, brut := range strings.Split(liste, ",") {
		nom := strings.Trim(strings.TrimSpace(brut), "'\"")
		if nom == "" {
			continue
		}
		connu := false
		for _, tier := range PadTierOrder {
			if tier == nom {
				connu = true
			}
		}
		if !connu {
			t.Errorf("le web attend un niveau %q que le contrat Go ne sert pas : son groupe "+
				"resterait vide a l'ecran", nom)
		}
	}
}
