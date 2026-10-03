package grammar

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// composants_vehicule_ti40_chemins_test.go — les composants `ti=40` lus par les CHEMINS DE
// PRODUCTION : la traversee d un record NEW ([TraverseEntity]), la marche des etats de mouvement
// ([movementStateScanner.paquet]) et son compteur publie, et la regle « seule la marche d etat
// complet pose [Lecteur.etatComplet] ».

// archetypeTi40 rend un registre dont l archetype 40 porte, a leur index, les composants
// propres au vehicule utilises ici (les autres index sont absents de tout masque ecrit ici).
func archetypeTi40() *Registry {
	reg := &Registry{Archetypes: make([]Archetype, 41)}
	comps := make([]string, 48)
	comps[30], comps[31] = compVehicleAutoTurretTriggers, compVehicleAutoTurretAimingVector
	comps[34], comps[37] = compVehicleTypePhysics, compVehicleEmpTimer
	reg.Archetypes[40] = Archetype{Index: 40, Components: comps}
	return reg
}

// largeurEtatParDefautTi40Nul rend la largeur que lit [consumeDefaultStateTI40] sur un flux nul :
// le record ecrit ici porte autant de bits nuls, que la traversee lit a l identique.
func largeurEtatParDefautTi40Nul() int {
	br := LecteurSur(make([]byte, 64))
	consumeDefaultStateTI40(br)
	return br.BitPos()
}

// ecrireCorpsNeufTi40 ecrit le corps d un record NEW `ti=40` a partir de son typeIndex : R(6),
// etat par defaut nul, porte R(1) = 0, masque epars croissant `idx`.
func ecrireCorpsNeufTi40(t *testing.T, w *bitWriter, idx ...int) {
	t.Helper()
	w.bits(40, 6)
	w.bits(0, largeurEtatParDefautTi40Nul())
	w.bit(0)                    // porte du record NEW
	w.bit(0)                    // masque epars
	w.bits(uint64(len(idx)), 3) // nombre d index
	for _, i := range idx {
		w.bits(uint64(i), 6)
	}
}

// TestRecordNeufTi40TraverseJusquAuBout : un record NEW `ti=40` de masque {i30, i31}, lu par
// [TraverseEntity] (le chemin NEW), va au bout et s arrete au bit pres apres i31. MUTATION :
// `br.etatComplet = true` avant `traverseComponentLoop` dans [TraverseEntity] — ROUGE (i30 refuse).
func TestRecordNeufTi40TraverseJusquAuBout(t *testing.T) {
	var w bitWriter
	ecrireCorpsNeufTi40(t, &w, 30, 31)
	fin := w.n + 3 + largeurVecteurViseeTourelle
	w.bits(0b101, 3)                             // i30 : trois R(1)
	w.bits(0x5a5a5, largeurVecteurViseeTourelle) // i31 : R(19)
	w.bits(^uint64(0), 32)                       // une lecture trop longue se voit
	tr := TraverseEntity(LecteurSur(w.buf), archetypeTi40(), 0)
	if tr.TypeIndex != 40 || tr.DesyncAt != -1 || tr.EndBit != fin || len(tr.Comps) != 2 {
		t.Fatalf("NEW ti=40 {i30,i31} : ti=%d arret %d fin %d (%d composants), attendu ti=40, "+
			"aucun arret, fin %d, 2 composants", tr.TypeIndex, tr.DesyncAt, tr.EndBit, len(tr.Comps), fin)
	}
}

// paquetDeltaNeufTi40 rend un paquet delta dont la vue B porte UN record NEW `ti=40` (slot 77)
// de masque {i34} (mode 2 si `i34`) ou {i37}, puis le terminateur, et une vue C vide.
func paquetDeltaNeufTi40(t *testing.T, i34 bool) []byte {
	t.Helper()
	var w bitWriter
	w.bit(1) // configuration
	w.bit(0) // vue A vide
	w.bit(0)
	w.bits(recNew, 2)
	w.bits(77, 13) // id : R(13) + R(2)
	w.bits(1, 2)
	if i34 {
		ecrireCorpsNeufTi40(t, &w, 34)
		w.bit(1) // mode 2 : avant/haut puis vitesse brutes
		w.bits(0, fwdUpDynPrecMode2Bits)
		w.bits(0, rawVec3Bits)
	} else {
		ecrireCorpsNeufTi40(t, &w, 37)
		w.bits(0xa5, largeurMinuteurEMPVehicule)
	}
	w.bits(0, 3) // fin de la vue B
	w.bit(0)     // vue C vide
	return w.buf
}

// TestLeCompteurPublieCompteLesLecturesDeI34 : la marche des etats de mouvement compte dans
// [types.MovementStateStats.VehicleTypePhysicsByWriterLaw] chaque record qui lit `i34`, et lui
// seul. MUTATION : [lecturesDeComposant] rend toujours 0 — ROUGE.
func TestLeCompteurPublieCompteLesLecturesDeI34(t *testing.T) {
	var st types.MovementStateStats
	var tir types.ContinuousFireStats
	cfg := DefaultFrameConfig()
	cfg.Obs = NouvelleObservation()
	for _, c := range []struct {
		i34    bool
		compte int
	}{{true, 1}, {false, 1}, {true, 2}} {
		sc := &movementStateScanner{st: &st, monde: NewWorld(archetypeTi40()), obs: cfg.Obs,
			tir: nouveauCollecteurTirContinu(&tir)}
		pay := paquetDeltaNeufTi40(t, c.i34)
		sc.paquet(0, FilmPacket{Type: PacketTypeDelta, Size: len(pay)}, pay, cfg)
		if st.VehicleTypePhysicsByWriterLaw != c.compte {
			t.Fatalf("apres un NEW ti=40 (i34 %v) : compteur %d, attendu %d", c.i34,
				st.VehicleTypePhysicsByWriterLaw, c.compte)
		}
	}
}

// TestEtatCompletPoseParLaSeuleMarcheDEtatComplet : `etatComplet = true` n est ecrit que dans
// `keyframe_fullstate_loop.go` (la marche sans masque, `FUN_142e2c690`) ; tout autre chemin
// lit un record a masque et le laisse faux. Les tests sont hors portee (ils posent l etat
// voulu). L ecriture par classe de caracteres evite que ce fichier se denonce lui-meme.
func TestEtatCompletPoseParLaSeuleMarcheDEtatComplet(t *testing.T) {
	motif := regexp.MustCompile(`etatComplet\s*[=]\s*true`)
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("glob : %v (%d fichiers)", err, len(files))
	}
	hote := 0
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		data, err := os.ReadFile(f) //nolint:gosec // fichiers du paquet lui-meme
		if err != nil {
			t.Fatalf("lecture %s : %v", f, err)
		}
		n := len(motif.FindAll(data, -1))
		if f == "keyframe_fullstate_loop.go" {
			hote = n
		} else if n > 0 {
			t.Errorf("%s pose etatComplet (%d fois) : seule la marche d etat complet le pose", f, n)
		}
	}
	if hote != 1 {
		t.Errorf("keyframe_fullstate_loop.go pose etatComplet %d fois, attendu 1", hote)
	}
}
