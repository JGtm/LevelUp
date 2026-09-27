package replay

import "testing"

// vies_bornes_incluses_test.go — AUCUNE VIE ANONYME, MEME D'UN SEUL ECHANTILLON (RA2-6, lot J5.4 du
// plan de suite de l'audit du decodeur, 2026-09-27).

// TestNameTracksByLives_VieDUnSeulEchantillonNommeeParSaVie : une vie d'un seul echantillon a
// `from == to` ; la piste qui la porte la recouvre sur UN instant. Le recouvrement se mesure a
// bornes INCLUSES, et cette vie nomme sa piste — regle produit : aucune vie publiee n'est anonyme.
//
// ROUGE AVANT : `ov > bestOverlap` avec un recouvrement a bornes exclusives valait 0 — la piste
// restait sans nom (mesure de l'audit : `084a804d`, `unnamedLives` 0 -> 1).
//
// MUTATION : retirer le `+ 1` du recouvrement -> la piste reste anonyme, rouge.
func TestNameTracksByLives_VieDUnSeulEchantillonNommeeParSaVie(t *testing.T) {
	const t0 = 5_050_000 // l'unique echantillon, dans la frame 40 (axe 1 s + 100 ms)
	tracks := []Track{{Slot: 7, StartFrame: 40, EndFrame: 40}}
	lives := []lifeSpan{{slot: 7, from: t0, to: t0, xuid: 111}}
	nameTracksByLives(tracks, lives, 1_000_000, 100_000, nil)
	if tracks[0].XUID != "111" {
		t.Fatalf("la piste de la vie d'un seul echantillon est restee %q, attendu 111", tracks[0].XUID)
	}
	// Une vie qui ne TOUCHE pas la piste ne la nomme toujours pas : bornes incluses, pas elargies.
	loin := []Track{{Slot: 7, StartFrame: 60, EndFrame: 61}}
	nameTracksByLives(loin, lives, 1_000_000, 100_000, nil)
	if loin[0].XUID != "" {
		t.Fatalf("une piste que la vie ne recouvre pas a ete nommee %q", loin[0].XUID)
	}
}
