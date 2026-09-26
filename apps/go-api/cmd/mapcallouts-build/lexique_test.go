package main

// Témoins du LEXIQUE des noms de lieu — tous HORS LIGNE : ils ne lisent que de la donnée
// versionnée (callouts_lexique.csv, callouts_i18n.csv), jamais les fichiers du jeu. C'est
// voulu : le lexique est une donnée de référence, et une régression du décodeur se voit
// sur le fichier PRODUIT, pas seulement sur la machine qui a le jeu installé.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/himap"
)

// cheminReference rend le dossier reference/ du titre par défaut depuis ce paquet.
func cheminReference(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "..", "..", "data", "titles", "halo_infinite", "reference")
}

func lexiqueVersionne(t *testing.T) libellesParStringID {
	t.Helper()
	p := filepath.Join(cheminReference(t), nomLexique)
	lex, err := chargeLexique(p)
	if err != nil {
		t.Fatalf("lexique versionné (%s) : %v", p, err)
	}
	return lex
}

// TestLexiqueVersionnePorteDesLibellesReels — LE TÉMOIN. Des noms de zone que le jeu
// affiche réellement, dans les deux langues, choisis parmi ceux qui manquaient au
// catalogue avant l'extraction uslg (2026-09-02) : ils sont ABSENTS de callouts_i18n.csv
// et pourtant employés par des dizaines de zones Forge.
func TestLexiqueVersionnePorteDesLibellesReels(t *testing.T) {
	lex := lexiqueVersionne(t)
	cas := []struct {
		sid    uint32
		en, fr string
	}{
		{0x4E9E5E09, "River", "Rivière"},                // 55 zones du corpus Forge
		{0x75F9173F, "South Outside", "Extérieur sud"},  // 38
		{0x20E36662, "Lift", "Monte-charge"},            // 33
		{0x9B11C6EA, "Ledge", "Corniche"},               // 27
		{0xAB50B931, "Underpass", "Passage souterrain"}, // 26
		{0x22908E62, "Ramp", "Rampe"},                   // 25
		{0x885AA3D0, "Courtyard", "Grande cour"},        // 21
	}
	for _, c := range cas {
		got, ok := lex[c.sid]
		if !ok {
			t.Errorf("string_id %08X absent du lexique", c.sid)
			continue
		}
		if got.en != c.en || got.fr != c.fr {
			t.Errorf("string_id %08X = (%q, %q), attendu (%q, %q)", c.sid, got.en, got.fr, c.en, c.fr)
		}
	}
}

// TestLexiqueCouvreLeCSVFigeAuCaractereRes — LE GARDE-RAIL DU DÉCODEUR.
//
// callouts_i18n.csv est l'extraction FIGÉE, validée zone par zone sur les 22 cartes
// intégrées (816/816). Le lexique vient d'un décodeur écrit après coup : s'il se trompait
// d'octet, de langue ou de table d'index, il produirait des textes décalés — un nom de zone
// FAUX, ce qui en compétitif est pire qu'un nom absent. Ce test exige donc les 463
// string_id du CSV, avec un EN et un FR identiques au caractère près.
func TestLexiqueCouvreLeCSVFigeAuCaractereRes(t *testing.T) {
	lex := lexiqueVersionne(t)
	_, parSID, err := chargeLibelles(filepath.Join(cheminReference(t), "callouts_i18n.csv"))
	if err != nil {
		t.Fatalf("CSV figé : %v", err)
	}
	if len(parSID) == 0 {
		t.Fatal("CSV figé vide")
	}
	var absents, divergents int
	for sid, attendu := range parSID {
		got, ok := lex[sid]
		if !ok {
			absents++
			if absents <= 5 {
				t.Errorf("string_id %08X (%q) absent du lexique", sid, attendu.en)
			}
			continue
		}
		if got.en != attendu.en || got.fr != attendu.fr {
			divergents++
			if divergents <= 5 {
				t.Errorf("string_id %08X : lexique (%q, %q) != CSV (%q, %q)",
					sid, got.en, got.fr, attendu.en, attendu.fr)
			}
		}
	}
	if absents != 0 || divergents != 0 {
		t.Fatalf("%d/%d string_id absents, %d divergents — le décodeur uslg a bougé",
			absents, len(parSID), divergents)
	}
	t.Logf("%d string_id du CSV figé, tous présents et identiques dans un lexique de %d entrées",
		len(parSID), len(lex))
}

// TestLexiqueEstTrieEtSansTexteVide — la forme du fichier : en-tête, tri par string_id
// (diff stable d'une extraction à l'autre), aucun libellé vide dans l'une des langues.
func TestLexiqueEstTrieEtSansTexteVide(t *testing.T) {
	p := filepath.Join(cheminReference(t), nomLexique)
	brut, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("lecture : %v", err)
	}
	lignes := strings.Split(strings.TrimRight(string(brut), "\n"), "\n")
	if len(lignes) < 2 {
		t.Fatal("lexique vide")
	}
	if lignes[0] != strings.Join(colonnesLexique, ";") {
		t.Fatalf("en-tête = %q", lignes[0])
	}
	precedent := ""
	for i, l := range lignes[1:] {
		champs := strings.SplitN(l, ";", 3)
		if len(champs) != 3 {
			continue // un libellé contenant « ; » est cité par encoding/csv : le fond est testé ailleurs
		}
		if champs[0] <= precedent {
			t.Fatalf("ligne %d : %s n'est pas après %s (fichier non trié)", i+2, champs[0], precedent)
		}
		precedent = champs[0]
	}
}

// TestFusionneLexiqueRefuseUneDivergence — la fusion ne tranche JAMAIS entre deux textes
// concurrents pour un même string_id : elle échoue, et rien n'est publié.
func TestFusionneLexiqueRefuseUneDivergence(t *testing.T) {
	base := libellesParStringID{0x11111111: {en: "Cave", fr: "Grotte", stringID: 0x11111111}}
	lex := libellesParStringID{
		0x11111111: {en: "Cavern", fr: "Caverne", stringID: 0x11111111},
		0x22222222: {en: "River", fr: "Rivière", stringID: 0x22222222},
	}
	if _, _, err := fusionneLexique(base, lex); err == nil {
		t.Fatal("une divergence de texte doit faire échouer la fusion")
	}
	// sans divergence : le lexique complète, le CSV reste intact.
	lex[0x11111111] = base[0x11111111]
	out, ajouts, err := fusionneLexique(base, lex)
	if err != nil {
		t.Fatalf("fusion saine : %v", err)
	}
	if ajouts != 1 || len(out) != 2 || out[0x11111111].en != "Cave" || out[0x22222222].en != "River" {
		t.Fatalf("fusion = %d ajouts, %d entrées : %+v", ajouts, len(out), out)
	}
}

// TestEcritLexiqueEcarteUnCoupleIncomplet — une entrée sans FR (ou sans EN) ne s'écrit pas :
// elle ferait une zone nommée dans une langue et muette dans l'autre.
func TestEcritLexiqueEcarteUnCoupleIncomplet(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, nomLexique)
	n, ecartes, err := ecritLexique(p, map[uint32]himap.LibelleLieu{
		0x00000002: {EN: "River", FR: "Rivière"},
		0x00000001: {EN: "Ledge", FR: ""},
		0x00000003: {EN: "", FR: "Rampe"},
	})
	if err != nil {
		t.Fatalf("écriture : %v", err)
	}
	if n != 1 || ecartes != 2 {
		t.Fatalf("écrites %d, écartées %d ; attendu 1 et 2", n, ecartes)
	}
	relu, err := chargeLexique(p)
	if err != nil {
		t.Fatalf("relecture : %v", err)
	}
	if len(relu) != 1 || relu[0x00000002].fr != "Rivière" {
		t.Fatalf("relu = %+v", relu)
	}
}
