package mappings

import "testing"

// loader_replay_labels_flagzone_test.go — LA ZONE DE RETOUR REFUSE D'ETRE A MOITIE DECLAREE.
//
// Une section absente est un SILENCE licite (le titre n'a pas de zone, le rejeu ne dessine rien).
// Une section presente mais incomplete est une ERREUR DE CONFIGURATION : publiee telle quelle,
// elle ferait dessiner un cercle de rayon nul ou une jauge qui n'avance jamais, et rien a l'ecran
// ne dirait que le manifeste est en cause.

const flagZoneMeta = "[meta]\ntitle_slug=\"x\"\nschema_version=2\n"

// flagZoneComplete — les trois grandeurs, bien formees.
const flagZoneComplete = "radius_m=1.3\nreset_seconds=30\nsolo_seconds=3.1\n"

// TestFlagReturnZoneAbsenteEstUnSilence — pas de section, pas de zone, pas d'erreur.
func TestFlagReturnZoneAbsenteEstUnSilence(t *testing.T) {
	set, err := LoadReplayLabelsFromBytes("x.toml", []byte(flagZoneMeta))
	if err != nil {
		t.Fatalf("un manifeste sans zone doit charger : %v", err)
	}
	if z := set.FlagReturnZone(); z.Declared() {
		t.Fatalf("zone declaree alors qu'aucune section ne l'ecrit : %+v", z)
	}
}

// TestFlagReturnZoneComplete — les trois grandeurs traversent le chargeur.
func TestFlagReturnZoneComplete(t *testing.T) {
	set, err := LoadReplayLabelsFromBytes("x.toml",
		[]byte(flagZoneMeta+"[flag_return_zone]\n"+flagZoneComplete))
	if err != nil {
		t.Fatalf("chargement : %v", err)
	}
	z := set.FlagReturnZone()
	if !z.Declared() || z.RadiusM != 1.3 || z.ResetSeconds != 30 || z.SoloSeconds != 3.1 {
		t.Fatalf("zone lue %+v", z)
	}
}

// TestFlagReturnZoneIncompleteEstFatale — chaque grandeur manquante ou absurde arrete tout.
func TestFlagReturnZoneIncompleteEstFatale(t *testing.T) {
	cas := []struct {
		nom  string
		body string
	}{
		{"rayon absent", "reset_seconds=30\nsolo_seconds=3.1\n"},
		{"rayon negatif", "radius_m=-1\nreset_seconds=30\nsolo_seconds=3.1\n"},
		{"minuterie absente", "radius_m=1.3\nsolo_seconds=3.1\n"},
		{"duree solo absente", "radius_m=1.3\nreset_seconds=30\n"},
		// S'y tenir ACCELERE le retour : une duree solo superieure a la minuterie nue dirait le
		// contraire, et la jauge irait a rebours de ce que le joueur voit.
		{"solo plus lent que la minuterie", "radius_m=1.3\nreset_seconds=30\nsolo_seconds=31\n"},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			raw := flagZoneMeta + "[flag_return_zone]\n" + c.body
			if _, err := LoadReplayLabelsFromBytes("x.toml", []byte(raw)); err == nil {
				t.Fatal("attendu une erreur de configuration, obtenu un chargement reussi")
			}
		})
	}
}
