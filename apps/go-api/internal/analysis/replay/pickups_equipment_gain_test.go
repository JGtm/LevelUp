package replay

// pickups_equipment_gain_test.go — FAUT-IL PUBLIER LES RAMASSAGES NON-ARME ?
//
// LA QUESTION EST UNE QUESTION DE PRODUIT, PAS DE DÉCODAGE. Les classes 2 et 3 de l'événement
// natif désignent autre chose qu'une arme (équipement, grenades, consommables) — mesuré : elles
// portent une famille d'arme du canal i43..i46 dans 0,0 % des cas, sur 118 événements. Or le
// document publie DÉJÀ un canal d'équipement daté : `equipmentChanges` (i48). Republier la
// même information sous un autre nom alourdirait l'artefact sans rien apprendre à personne.
//
// LE CRITÈRE, ÉCRIT AVANT LA MESURE : on ne publie les classes 2 et 3 que si elles COMBLENT
// DES TROUS. Un ramassage natif non-arme APPORTE quelque chose s'il n'existe AUCUNE émission
// i48 sur LE MÊME SLOT à moins de 500 ms — sinon c'est un doublon.
//
//	G1 — si le gain (ramassages non-arme sans émission i48 correspondante) dépasse 30 % des
//	     non-armes, on publie : le canal comble un trou réel.
//	G2 — si le gain est sous 10 %, on ne publie pas et on consigne le négatif.
//	G3 — entre les deux, on publie en le disant : l'artefact porte la couverture, un lecteur
//	     peut juger.
//
// Garde BIPED_PICKUP_FILM.

import (
	"os"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

func TestPickupsEquipmentGain(t *testing.T) {
	dir := os.Getenv(pickupsBridgeEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", pickupsBridgeEnv)
	}
	release := filmdec.LockProcessDecode()
	defer release()

	pickups, _, err := filmdec.ScanFilmBipedPickups(dir)
	if err != nil {
		t.Fatalf("ramassages natifs illisibles : %v", err)
	}
	if len(pickups) == 0 {
		t.Skip("aucun ramassage natif sur ce film")
	}
	// SANS témoin de naissance : on n'utilise pas la NATURE du changement (`taken` / `spent`),
	// seulement le couple (instant, slot) — et celui-là est lu, pas déduit. Le sur-classement
	// des premières émissions n'a donc aucun effet sur cette mesure.
	equip, eStats, err := filmdec.ScanFilmEquipmentChanges(dir, nil)
	if err != nil {
		t.Fatalf("changements d equipement illisibles : %v", err)
	}
	const tolUS = 500_000
	covered := func(slot uint32, ts uint64, decalUS int64) bool {
		for _, c := range equip {
			if c.Slot != slot {
				continue
			}
			d := int64(c.TimestampUS) + decalUS - int64(ts)
			if d < 0 {
				d = -d
			}
			if d <= tolUS {
				return true
			}
		}
		return false
	}
	items, gain := 0, 0
	weapons, weaponsCovered := 0, 0
	for _, p := range pickups {
		if filmdec.BipedPickupIsWeaponClass(p.Class) {
			weapons++
			if covered(p.Slot, p.TimestampUS, 0) {
				weaponsCovered++
			}
			continue
		}
		items++
		if !covered(p.Slot, p.TimestampUS, 0) {
			gain++
		}
	}
	// TÉMOIN : le même calcul avec les émissions i48 décalées. Il mesure à quel point
	// « couvert » est facile à obtenir par hasard sur ce film ; un témoin proche du réel
	// voudrait dire que la couverture ne mesure rien.
	temoinCouvert := 0
	for _, p := range pickups {
		if filmdec.BipedPickupIsWeaponClass(p.Class) {
			continue
		}
		if covered(p.Slot, p.TimestampUS, 37_000_000) {
			temoinCouvert++
		}
	}
	t.Logf("== GAIN DES CLASSES NON-ARME contre le canal i48 · %s ==", dir)
	t.Logf("emissions i48 : %d sur %d vies · ramassages natifs : %d armes, %d non-armes",
		len(equip), eStats.Lives, weapons, items)
	t.Logf("CONTROLE — armes couvertes par une emission i48 du meme slot a <= 500 ms : %d / %d (%.1f %%)",
		weaponsCovered, weapons, 100*float64(weaponsCovered)/float64(max(weapons, 1)))
	t.Logf("GAIN — non-armes SANS emission i48 du meme slot a <= 500 ms : %d / %d (%.1f %%)",
		gain, items, 100*float64(gain)/float64(max(items, 1)))
	t.Logf("TEMOIN (i48 decalees de +37 s) — non-armes « couvertes » par hasard : %d / %d (%.1f %%)",
		temoinCouvert, items, 100*float64(temoinCouvert)/float64(max(items, 1)))
	pct := 100 * float64(gain) / float64(max(items, 1))
	switch {
	case pct >= 30:
		t.Logf("VERDICT G1 : gain %.1f %% >= 30 %% — LES CLASSES 2 ET 3 SONT PUBLIEES, elles comblent un trou reel.", pct)
	case pct < 10:
		t.Logf("VERDICT G2 : gain %.1f %% < 10 %% — doublon d'i48, NE PAS PUBLIER les classes 2 et 3.", pct)
	default:
		t.Logf("VERDICT G3 : gain %.1f %% entre 10 et 30 %% — publier en le disant (la couverture porte le compte).", pct)
	}
}
