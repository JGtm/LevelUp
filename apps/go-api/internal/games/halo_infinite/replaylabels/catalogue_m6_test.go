package replaylabels

// catalogue_m6_test.go — TROIS ENTREES DU CATALOGUE D ARMES DU REJEU, etablies sur pieces le
// 2026-09-24 (retours du rejeu, lots M6.3 et M6.4 ; instrument
// `internal/himodule/m6_bobine_mains_nues_research_test.go`, modules installes en lecture seule).
//
//   - `00007CA9` = « Mains nues » / « Unarmed » : `WeaponTags.unarmed` du script Lua global. Le jeu
//     le remet a chaque bipede ; nomme pour le jour ou un joueur le TIENT en cours de partie.
//   - `E9E7FF79` (`forge_fusion_coil_mp`) et `1D63A8CD` (`fusion_coil`) : la bobine a fusion UNSC —
//     chacune declare le degat que `damagetag/data/labels.tsv` range en explosion
//     `sb_008_exp_single_small_kineticunsc`, celle de `hinf_coil_kinetic`. Observees dans 31 et 5
//     documents du parc (ramassees, tenues, lachees) sous leur seul hexadecimal.
//   - VERIFICATION, SANS CORRECTION : `2AC9C2FF` (nom de code Lua `hotrod`) est le Calcineur
//     (Heatwave) et `230447B1` (`proto_heatwave`) le Cremateur (Cindershot). Le nom de code ment
//     (meme defaut deja mesure sur deux atlas d icones) ; les pieces qui tranchent : la variante
//     classee `ranked_heatwave` = `5AC6CFB2` porte la MEME vignette que `2AC9C2FF` (index 21 de
//     l atlas, ou l image est a deux branches, celle du Heatwave), et la passe humaine du
//     2026-08-09 lit l index 23 du kill feed « Calcineur », l index 22 « Cremateur ».
//
// LES LIBELLES ATTENDUS SONT DES LITTERAUX : les deriver du fichier teste rendrait le test
// tautologique.

import "testing"

func TestCatalogueM6MainsNuesBobineEtHeatwave(t *testing.T) {
	cat, err := Load(repoRoot(t), "halo_infinite")
	if err != nil {
		t.Fatalf("chargement du catalogue : %v", err)
	}
	attendus := map[uint32]struct{ key, en, fr string }{
		0x00007ca9: {"hinf_unarmed", "Unarmed", "Mains nues"},
		0xe9e7ff79: {"hinf_coil_kinetic", "UNSC Fusion Coil", "Bobine à fusion UNSC"},
		0x1d63a8cd: {"hinf_coil_kinetic", "UNSC Fusion Coil", "Bobine à fusion UNSC"},
		0x2ac9c2ff: {"hinf_heatwave", "Heatwave", "Calcineur"},
		0x230447b1: {"hinf_cindershot", "Cindershot", "Crémateur"},
	}
	for fam, a := range attendus {
		w, ok := cat.Weapons[fam]
		if !ok || cat.Keys[fam] != a.key || w.En != a.en || w.Fr != a.fr {
			t.Errorf("famille %08X : cle %q libelle %+v, attendu %q / %q / %q",
				fam, cat.Keys[fam], w, a.key, a.en, a.fr)
		}
	}
}

// TestCatalogueM6EffetsDeMortDesArmesDeVehicule — LES EXPLOSIONS A LA MORT (lot M6.2, decisions de
// l utilisateur du 2026-09-23 nuit ; revue adverse du lot, constat R5). La mort se dessine par la
// cle du kill (`killEffects`, table `[shot_effects]`) : l obus du Scorpion explose comme la grenade,
// les roquettes du Rockethog comme le SPNKR — deux cles dont TOUTES les sources de degat sont
// l arme visee. `hinf_banshee` n a PAS d entree : ses sources melent la bombe et d autres
// projectiles, une famille par cle serait fausse sur l un des deux (limite ecrite).
func TestCatalogueM6EffetsDeMortDesArmesDeVehicule(t *testing.T) {
	cat, err := Load(repoRoot(t), "halo_infinite")
	if err != nil {
		t.Fatalf("chargement du catalogue : %v", err)
	}
	for _, cle := range []string{"hinf_scorpion", "hinf_rockethog"} {
		if got := cat.Effects[cle]; got != cat.Effects["hinf_frag_grenade"] || got != cat.Effects["hinf_m41_spnkr"] {
			t.Errorf("effet de mort de %s = %q, attendu l explosion de la grenade et du SPNKR (%q / %q)",
				cle, got, cat.Effects["hinf_frag_grenade"], cat.Effects["hinf_m41_spnkr"])
		}
	}
	if got, ok := cat.Effects["hinf_banshee"]; ok {
		t.Errorf("hinf_banshee a un effet de mort (%q) : ses sources melent la bombe et d autres projectiles", got)
	}
}
