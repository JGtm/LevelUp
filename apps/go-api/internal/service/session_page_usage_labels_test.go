package service

// session_page_usage_labels_test.go — les trois promesses de la résolution des noms
// d'armes du bloc usage (D7 du plan de lisibilité 2026-09-09) :
//
//  1. une famille QUE LE CATALOGUE CONNAÎT reçoit son nom, dans la langue demandée ;
//  2. une famille INCONNUE garde sa clé et n'invente rien — c'est la règle du catalogue
//     du rejeu, reprise ici : un nom approchant se lit comme une certitude ;
//  3. sans repoRoot (montage sans catalogue), la résolution ne fait RIEN et ne casse rien.
//
// Le test lit le VRAI catalogue du titre (config/titles/halo_infinite/mappings), comme
// replaylabels/catalog_test.go : une fixture recopierait la jointure famille → clé →
// nom, et une jointure recopiée dans un test est une jointure qui dérive.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/halo_infinite/replaylabels"
	"levelup/go-api/internal/games/weapons"
)

// sortedFamilies rend les familles du registre dans un ordre STABLE : le parcours d'une
// map Go est randomisé, et un test qui pioche « une famille nommée » au hasard échoue un
// jour sur deux quand une seule famille perd son nom.
func sortedFamilies(m map[uint32]string) []uint32 {
	out := make([]uint32, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(a, b int) bool { return out[a] < out[b] })
	return out
}

// formatFamilyKey écrit une famille comme le fait le contrat : huit hexa minuscules
// sans préfixe (la forme produite par replay.PadWeaponFamilyKey).
func formatFamilyKey(f uint32) string { return fmt.Sprintf("%08x", f) }

// usageLabelsRepoRoot remonte jusqu'au répertoire qui porte config/titles.
func usageLabelsRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("répertoire courant : %v", err)
	}
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "config", "titles")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("racine du dépôt introuvable")
	return ""
}

// aKnownFamilyKey rend la clé de contrat (huit hexa minuscules, forme
// replay.PadWeaponFamilyKey) d'une famille QUE LE CATALOGUE DU TITRE NOMME, plus le nom
// FR et le nom EN attendus. Choisie dans le catalogue réel plutôt qu'écrite en dur : une
// famille codée ici deviendrait fausse au premier remaniement de weapon_names.toml.
func aKnownFamilyKey(t *testing.T, repoRoot string) (key, fr, en string) {
	t.Helper()
	cat, err := replaylabels.Load(repoRoot, "halo_infinite")
	if err != nil {
		t.Fatalf("chargement du catalogue : %v", err)
	}
	// Ordre stable : on parcourt les familles du registre, pas la map du catalogue.
	for _, family := range sortedFamilies(weapons.FilmshellWeaponKeysByFamily()) {
		lbl, ok := cat.Weapons[family]
		if !ok || lbl.Fr == "" || lbl.En == "" {
			continue
		}
		return formatFamilyKey(family), lbl.Fr, lbl.En
	}
	t.Fatal("aucune famille nommée dans le catalogue du titre")
	return "", "", ""
}

func TestResolvePadFamilyLabels_FamilleConnueNommeeDansLaLangueDemandee(t *testing.T) {
	root := usageLabelsRepoRoot(t)
	key, fr, en := aKnownFamilyKey(t, root)

	svc := NewSessionPageService(nil)
	svc.repoRoot, svc.titleSlug = root, "halo_infinite"

	block := domain.SessionUsageBlock{
		PadFamilies: []domain.SessionUsagePadFamily{{FamilyKey: key}},
	}
	svc.resolvePadFamilyLabels(context.Background(), &block, "fr")
	if got := block.PadFamilies[0].FamilyLabel; got != fr {
		t.Errorf("nom FR = %q, attendu %q", got, fr)
	}
	// La CLÉ reste servie : c'est elle qui identifie la ligne côté client.
	if block.PadFamilies[0].FamilyKey != key {
		t.Errorf("clé = %q, attendu %q (la clé ne doit jamais être remplacée)", block.PadFamilies[0].FamilyKey, key)
	}

	blockEN := domain.SessionUsageBlock{
		PadFamilies: []domain.SessionUsagePadFamily{{FamilyKey: key}},
	}
	svc.resolvePadFamilyLabels(context.Background(), &blockEN, "en")
	if got := blockEN.PadFamilies[0].FamilyLabel; got != en {
		t.Errorf("nom EN = %q, attendu %q", got, en)
	}
}

func TestResolvePadFamilyLabels_FamilleInconnueGardeSaCle(t *testing.T) {
	svc := NewSessionPageService(nil)
	svc.repoRoot, svc.titleSlug = usageLabelsRepoRoot(t), "halo_infinite"

	block := domain.SessionUsageBlock{
		PadFamilies: []domain.SessionUsagePadFamily{
			{FamilyKey: "ffffffff"},     // hexa valide, hors registre
			{FamilyKey: "powerup_camo"}, // pas une famille d'arme : ne se parse pas
		},
	}
	svc.resolvePadFamilyLabels(context.Background(), &block, "fr")
	for i, fam := range block.PadFamilies {
		if fam.FamilyLabel != "" {
			t.Errorf("famille %d : nom = %q, attendu vide (aucun nom approchant)", i, fam.FamilyLabel)
		}
	}
}

func TestResolvePadFamilyLabels_SansRepoRootNeFaitRien(t *testing.T) {
	svc := NewSessionPageService(nil)
	svc.titleSlug = "halo_infinite" // repoRoot volontairement vide

	block := domain.SessionUsageBlock{
		PadFamilies: []domain.SessionUsagePadFamily{{FamilyKey: "aabbccdd"}},
	}
	svc.resolvePadFamilyLabels(context.Background(), &block, "fr")
	if block.PadFamilies[0].FamilyLabel != "" {
		t.Error("un montage sans catalogue ne doit nommer aucune famille")
	}
}

func TestParseWeaponFamilyKey(t *testing.T) {
	cases := []struct {
		in   string
		want uint32
		ok   bool
	}{
		{"00000000", 0, true},
		{"a2b3c4d5", 0xa2b3c4d5, true},
		{"ffffffff", 0xffffffff, true},
		{"powerup_camo", 0, false}, // nom canonique de bonus, pas une famille d'arme
		{"0xa2b3c4d5", 0, false},   // la clé du contrat est SANS préfixe
		{"", 0, false},
	}
	for _, c := range cases {
		got, ok := parseWeaponFamilyKey(c.in)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("parseWeaponFamilyKey(%q) = (%#x, %v), attendu (%#x, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}
