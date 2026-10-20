package weapons

// filmshell_covers_labels_test.go — GARDE-RAIL « une arme connue est une arme nommée dans
// le rejeu » (2026-09-13).
//
// CE QUI S'EST PASSÉ. Le Mutilator était connu du seed de `labels.go` depuis avril
// (0xd791556542c9679f, « Mutilator » / « Mutilateur ») et avait reçu sa ligne de registre le
// 2026-09-10 — mais AUCUN identifiant filmshell. Or `FilmshellWeaponKeysByFamily` est la
// SEULE jointure `famille d'arme -> weapon_key` du catalogue du rejeu 2D : sans entrée là,
// l'arme n'a pas de famille, donc pas de libellé, et le rejeu comme la vue de match
// affichaient « 0xD7915565 » — un identifiant brut, au milieu d'armes toutes nommées.
//
// CE QUE CE TEST VERROUILLE. Tout identifiant du seed de `labels.go` qui est une VRAIE ARME
// (suffixe `42c9679f` — la moitié basse commune aux armes de l'arsenal Infinite) doit avoir
// une entrée dans `weaponRegistryInfiniteFilmshell`. L'allowlist ci-dessous est la liste
// FERMÉE des exceptions justifiées : elle ne s'agrandit pas sans une raison datée écrite.

import "testing"

// filmshellSuffixeArme — moitié basse commune aux identifiants d'arme de l'arsenal Halo
// Infinite. Les lignes du seed qui ne la portent pas sont soit des sentinelles (0/1/2), soit
// des variantes cosmétiques dont l'identifiant complet vit déjà au registre.
const filmshellSuffixeArme uint64 = 0x42c9679f

// idsSansEntreeFilmshell — exceptions JUSTIFIÉES, une par une. Vide ailleurs : un
// identifiant d'arme sans entrée filmshell est un défaut, pas un cas particulier.
var idsSansEntreeFilmshell = map[uint64]string{
	// 2026-09-13 — « Sandwich » n'est pas une arme de l'arsenal : c'est un objet de plaisanterie
	// des cartes de Forge, sans ligne au registre d'armes, sans nom dans `weapon_names.toml` et
	// sans icône. Lui poser une entrée filmshell obligerait à lui inventer un weapon_key et une
	// classe. Il garde donc son hexadécimal — assumé, contrairement au Mutilator.
	// (Son cousin « Mythic Sandwich », 0xb7262ca1c8fb11d0, ne porte même pas le suffixe d'arme
	// et n'entre donc pas dans le périmètre de ce test.)
	0x880fe0bc42c9679f: "Sandwich — objet de Forge, hors arsenal",
}

func TestFilmshellCouvreLeSeedDeLabels(t *testing.T) {
	couvert := map[uint64]bool{}
	for _, f := range weaponRegistryInfiniteFilmshell {
		couvert[f.id] = true
	}

	var vues int
	for _, l := range weaponLabelSeeds() {
		if l.id&0xFFFFFFFF != filmshellSuffixeArme {
			continue // sentinelle (0/1/2) ou variante cosmétique : hors périmètre
		}
		vues++
		if couvert[l.id] {
			continue
		}
		if raison, ok := idsSansEntreeFilmshell[l.id]; ok {
			t.Logf("0x%016x (%s) hors registre filmshell — %s", l.id, l.en, raison)
			continue
		}
		t.Errorf("0x%016x (%q / %q) est nommé dans le seed de labels.go mais n'a AUCUNE entrée "+
			"dans weaponRegistryInfiniteFilmshell : le rejeu 2D et la vue de match afficheront "+
			"« 0x%08X » au lieu du nom. Ajouter la ligne {weapon_key, 0x%016x}.",
			l.id, l.en, l.fr, uint32(l.id>>32), l.id)
	}
	if vues == 0 {
		t.Fatal("aucun identifiant d'arme relevé dans le seed : le test ne verrouille plus rien")
	}

	// L'allowlist est un cliquet : une exception qui ne correspond plus à aucune ligne du seed
	// doit être RETIRÉE, sinon elle couvrirait un jour un identifiant réintroduit par erreur.
	dansLeSeed := map[uint64]bool{}
	for _, l := range weaponLabelSeeds() {
		dansLeSeed[l.id] = true
	}
	for id, raison := range idsSansEntreeFilmshell {
		if !dansLeSeed[id] {
			t.Errorf("allowlist : 0x%016x (%s) n'est plus dans le seed de labels.go — retirer l'entrée", id, raison)
		}
		if couvert[id] {
			t.Errorf("allowlist : 0x%016x (%s) a désormais une entrée filmshell — retirer l'exception", id, raison)
		}
	}
}
