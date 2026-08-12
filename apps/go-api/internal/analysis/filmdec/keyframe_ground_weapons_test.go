package filmdec

import "testing"

// keyframe_ground_weapons_test.go — l'invariant testé ici est l'ATTRIBUTION PAR ARCHÉTYPE : la
// même occurrence de famille d'arme doit aller au loadout d'un joueur si son record porteur est un
// biped (ti=35), et à l'arme au sol si c'est un objet du monde (ti=42). Un filtre relâché d'un
// cran donnerait « l'arme posée par terre » comme arme portée (et réciproquement).

// keyframeRecord écrit un en-tête de record keyframe ([id:32][field26=0:26][ti:6]) puis une charge utile
// contenant la famille d'arme fam, entourée de bits denses (un long train de zéros créerait une
// fausse ancre : le filtre fort du walker n'exige que « mot de 32 bits à q+32 < 50 »).
func (w *bitWriter) keyframeRecord(gen, slot uint32, ti int, fam uint32) {
	w.bits(uint64(gen)<<30|uint64(slot), 32)
	w.bits(uint64(ti), 32) // field26 = 0 -> le mot de 32 bits vaut ti
	w.bits(0xABCD, 16)
	w.bits(uint64(fam), 32)
	w.bits(0xDCBA, 16)
}

// famPorte / famAuSol sont deux familles arbitraires du catalogue de test : seule leur
// APPARTENANCE au prédicat `known` compte pour le balayage.
const (
	famPorte uint32 = 0xC1A5B00D
	famAuSol uint32 = 0xD0F0BEEF
)

// twoRecordPayload rend un payload de keyframe portant un record biped (slot 10, famPorte) suivi
// d'un record d'arme au sol (slot 11, famAuSol).
func twoRecordPayload() []byte {
	w := &bitWriter{}
	w.bit(1) // préfixe 1 bit de la table keyframe
	w.keyframeRecord(1, 10, keyframeBipedTI, famPorte)
	w.keyframeRecord(1, 11, GroundWeaponTypeIndex, famAuSol)
	return w.buf
}

func knownFamilies() map[uint32]bool {
	return map[uint32]bool{famPorte: true, famAuSol: true}
}

// TestKeyframeGroundWeapons_AttributionParArchetype : le record ti=42 rend l'arme au sol, et
// SEULEMENT elle ; l'arme du biped ne fuit pas dans le calque au sol.
func TestKeyframeGroundWeapons_AttributionParArchetype(t *testing.T) {
	pay := twoRecordPayload()
	got := keyframeGroundWeapons(pay, knownFamilies())
	if len(got) != 1 {
		t.Fatalf("1 arme au sol attendue, obtenu %d : %+v", len(got), got)
	}
	if got[0].Slot != 11 || got[0].Gen != 1 {
		t.Fatalf("slot/gen attendus 11/1, obtenus %d/%d", got[0].Slot, got[0].Gen)
	}
	if len(got[0].Families) != 1 || got[0].Families[0] != famAuSol {
		t.Fatalf("famille attendue %#x, obtenu %#x", famAuSol, got[0].Families)
	}
}

// TestKeyframeGroundWeapons_LoadoutInchange : le MÊME payload rend toujours l'arme PORTÉE au
// biped — la levée du filtre pour les armes au sol ne doit rien changer aux loadouts (c'est le
// témoin de non-régression de la factorisation familiesByRecord).
func TestKeyframeGroundWeapons_LoadoutInchange(t *testing.T) {
	pay := twoRecordPayload()
	got := keyframeLoadouts(pay, knownFamilies())
	if len(got) != 1 {
		t.Fatalf("1 loadout attendu, obtenu %d : %+v", len(got), got)
	}
	if got[0].Slot != 10 {
		t.Fatalf("slot 10 attendu, obtenu %d", got[0].Slot)
	}
	if len(got[0].Families) != 1 || got[0].Families[0] != famPorte {
		t.Fatalf("famille portée attendue %#x, obtenu %#x", famPorte, got[0].Families)
	}
}

// TestKeyframeGroundWeapons_PayloadVide : un payload sans record ne produit aucune arme au sol.
func TestKeyframeGroundWeapons_PayloadVide(t *testing.T) {
	if got := keyframeGroundWeapons(nil, knownFamilies()); got != nil {
		t.Fatalf("aucune arme attendue, obtenu %+v", got)
	}
}

// TestScanFilmKeyframeGroundWeapons_SansCatalogue : sans familles à chercher, le balayage ne
// parcourt pas le film pour rien.
func TestScanFilmKeyframeGroundWeapons_SansCatalogue(t *testing.T) {
	got, err := ScanFilmKeyframeGroundWeapons("répertoire inexistant", nil)
	if err != nil || got != nil {
		t.Fatalf("attendu (nil, nil), obtenu (%v, %v)", got, err)
	}
}

// TestScanFilmKeyframeGroundWeapons_FilmAbsent : un répertoire sans chunk lisible est une erreur,
// pas un résultat vide silencieux.
func TestScanFilmKeyframeGroundWeapons_FilmAbsent(t *testing.T) {
	if _, err := ScanFilmKeyframeGroundWeapons("répertoire inexistant", knownFamilies()); err == nil {
		t.Fatal("erreur attendue pour un film illisible")
	}
}

// TestGroundWeaponPositions_SansBornes : sans bornes de carte, aucun quantum n'est une position —
// le décodeur rend une table vide plutôt que des coordonnées fausses.
func TestGroundWeaponPositions_SansBornes(t *testing.T) {
	if got := GroundWeaponPositions("répertoire inexistant", nil); len(got) != 0 {
		t.Fatalf("table vide attendue, obtenu %d entrées", len(got))
	}
}

// TestNearestGroundWeaponSample : l'échantillon retenu est le plus proche dans le temps, l'écart
// est rendu en valeur absolue, et une liste vide se dit (ok=false) au lieu de rendre un zéro.
func TestNearestGroundWeaponSample(t *testing.T) {
	pts := []WorldObjectSample{
		{TimestampUS: 1_000_000, X: 1},
		{TimestampUS: 3_000_000, X: 3},
	}
	got, gap, ok := NearestWorldObjectSample(pts, 2_600_000)
	if !ok || got.X != 3 || gap != 400_000 {
		t.Fatalf("attendu (X=3, gap=400000, ok), obtenu (X=%v, gap=%d, ok=%v)", got.X, gap, ok)
	}
	if _, _, ok := NearestWorldObjectSample(nil, 42); ok {
		t.Fatal("liste vide : ok=false attendu")
	}
}
