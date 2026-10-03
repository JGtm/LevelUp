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
// PRODUCTION : la traversee d un record NEW ([TraverseEntity]), les deux chemins DELTA
// ([decodeDelta] par [DecodeFrameRecords], [decodeDeltaWithArch]), la marche des etats de
// mouvement ([marcheurDesTrames.marcherLePaquet] puis [movementStateScanner.trame]) et son
// compteur publie, et la regle « seule la marche d etat complet pose [Lecteur.etatComplet] ».

// composantTi40 : un composant propre au vehicule, son index dans l archetype 40 et un corps
// ecrit d apres son deserialiseur (memes vecteurs que `composants_vehicule_ti40_test.go`).
type composantTi40 struct {
	idx       int
	nom, bits string
}

// composantsTi40 rend les seize composants propres au vehicule (i30..i42, i45..i47), dans l ordre
// du registre. `i34` est ecrit en mode 2 (R(1) = 1, puis les vecteurs bruts).
func composantsTi40() []composantTi40 {
	treize := strings.Repeat("1", 13)
	return []composantTi40{
		{30, compVehicleAutoTurretTriggers, "101"},
		{31, compVehicleAutoTurretAimingVector, strings.Repeat("10", 9) + "1"},
		{32, compVehicleTransformedOpenState, "1 10000000"},
		{33, compVehicleTypeState, "01 000101"},
		{34, compVehicleTypePhysics, "1" + strings.Repeat("0", fwdUpDynPrecMode2Bits+rawVec3Bits)},
		{35, compVehicleAutoTurretTarget, "1 1 101010101 01"},
		{36, compVehicleSentryState, "110 1"},
		{37, compVehicleEmpTimer, "10101010"},
		{38, compVehicleWeaponSet, "011 001 011"},
		{39, compVehicleAutoTurret, "10"},
		{40, compVehicleEquipmentTurretParent, "1 " + treize + " 10"},
		{41, compVehicleSeatsOverridePitch, strings.Repeat("0", 16)},
		{42, compVehicleSeatsOverrideYaw, strings.Repeat("1", 16)},
		{45, compAirDropFlight, "10 " + strings.Repeat("0", 14) + " " + strings.Repeat("1", 8)},
		{46, compWarp, "11 " + strings.Repeat("0", 16)},
		{47, compVehicleLowFrequency, "0 10101"},
	}
}

// archetypeTi40 rend un registre dont l archetype 40 porte, a leur index, les seize composants
// propres au vehicule ([composantsTi40]) ; les autres index sont absents de tout masque ecrit ici.
func archetypeTi40() *Registry {
	reg := &Registry{Archetypes: make([]Archetype, 41)}
	comps := make([]string, 48)
	for _, c := range composantsTi40() {
		comps[c.idx] = c.nom
	}
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

// ecrireBits ecrit une chaine de `0` et `1` (espaces ignores), MSB d abord.
func ecrireBits(w *bitWriter, s string) {
	for _, c := range strings.ReplaceAll(s, " ", "") {
		w.bit(uint64(c - '0'))
	}
}

// ecrireCorpsDeltaTi40 ecrit le corps d un DELTA `ti=40` apres son en-tete d id : selecteur de
// base R(1) = 0, masque DENSE R(1) = 1 + R(64) des seize composants, puis leurs corps dans l ordre
// du registre. Rend la position de fin attendue.
func ecrireCorpsDeltaTi40(w *bitWriter) int {
	w.bit(0) // selecteur de base (FUN_1406cdc04)
	var masque uint64
	for _, c := range composantsTi40() {
		masque |= uint64(1) << uint(c.idx)
	}
	w.bit(1) // masque dense
	w.bits(masque, 64)
	for _, c := range composantsTi40() {
		ecrireBits(w, c.bits)
	}
	return w.n
}

// verifierDeltaTi40 exige une traversee allee au bout, a `fin`, qui a lu les seize composants
// dans l ordre du registre.
func verifierDeltaTi40(t *testing.T, chemin string, tr EntityTrace, fin int) {
	t.Helper()
	attendus := composantsTi40()
	if tr.TypeIndex != 40 || tr.DesyncAt != -1 || tr.EndBit != fin || len(tr.Comps) != len(attendus) {
		t.Fatalf("%s : ti=%d arret a i%d fin %d (%d composants), attendu ti=40, aucun arret, fin %d, %d "+
			"composants", chemin, tr.TypeIndex, tr.DesyncAt, tr.EndBit, len(tr.Comps), fin, len(attendus))
	}
	for k, c := range attendus {
		if tr.Comps[k].Name != c.nom {
			t.Errorf("%s : composant %d = %s, attendu %s (i%d)", chemin, k, tr.Comps[k].Name, c.nom, c.idx)
		}
	}
}

// TestDeltaTi40LitSesComposants : un record DELTA `ti=40` lit ses seize composants propres
// (i30..i42, i45..i47, `i33` et `i34` porte posee par la loi du masque) et s arrete au bit pres,
// par les deux chemins DELTA de production : la boucle de records [DecodeFrameRecords] (chemin
// [decodeDelta], slot lie par le NEW qui le precede dans la trame) et le decodage a archetype
// explicite de l inference de slot non lie ([decodeDeltaWithArch]). C est la moitie « records a
// masque » de la regle de [Lecteur.etatComplet] ; [TestEtatCompletPoseParLaSeuleMarcheDEtatComplet]
// n en garde que l ecriture litterale. MUTATIONS : `br.etatComplet = !false` dans [decodeDelta]
// — ROUGE (chemin trame) ; dans [decodeDeltaWithArch] — ROUGE (chemin inference).
func TestDeltaTi40LitSesComposants(t *testing.T) {
	cfg := DefaultFrameConfig()
	var w bitWriter
	w.bits(0, cfg.PacketPreambleBits)
	w.bit(0) // NEW : R(1) = 0 puis R(2) = 1
	w.bits(recNew, 2)
	w.bits(77, cfg.IDLowBits)
	w.bits(1, 2) // generation
	ecrireCorpsNeufTi40(t, &w, 37)
	w.bits(0xa5, largeurMinuteurEMPVehicule)
	w.bit(1) // DELTA
	w.bits(77, cfg.IDLowBits)
	w.bits(1, 2)
	debutCorps := w.n
	fin := ecrireCorpsDeltaTi40(&w)
	w.bit(0) // fin des records : R(1) = 0 puis R(2) = 0
	w.bits(recEnd, 2)
	w.bits(^uint64(0), 32) // une lecture trop longue se voit
	recs, err := DecodeFrameRecords(LecteurSur(w.buf), NewWorld(archetypeTi40()), cfg)
	if err != nil || len(recs) != 2 || recs[1].Type != recDelta {
		t.Fatalf("trame NEW + DELTA ti=40 : %d records, erreur %v", len(recs), err)
	}
	verifierDeltaTi40(t, "DecodeFrameRecords", recs[1].Trace, fin)

	var v bitWriter
	ecrireBits(&v, strings.Repeat("0", debutCorps%8)) // meme alignement que dans la trame
	finInf := ecrireCorpsDeltaTi40(&v)
	v.bits(^uint64(0), 32)
	br := LecteurSur(v.buf)
	br.Skip(debutCorps%8 + 1) // l inference part du masque ([inferUnboundArchetype])
	verifierDeltaTi40(t, "decodeDeltaWithArch", decodeDeltaWithArch(br, archetypeTi40().Archetypes[40], 40), finInf)
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
// seul. Le paquet passe par la marche des trames de production
// ([marcheurDesTrames.marcherLePaquet]) puis par le balayage ([movementStateScanner.trame]).
// MUTATION : [lecturesDeComposant] rend toujours 0 — ROUGE.
func TestLeCompteurPublieCompteLesLecturesDeI34(t *testing.T) {
	var st types.MovementStateStats
	var tir types.ContinuousFireStats
	cfg := DefaultFrameConfig()
	cfg.Obs = NouvelleObservation()
	for _, c := range []struct {
		i34    bool
		compte int
	}{{true, 1}, {false, 1}, {true, 2}} {
		m := &marcheurDesTrames{cfg: cfg, monde: NewWorld(archetypeTi40())}
		m.entites = entitesDuMonde{w: m.monde}
		m.trame.paquet = &m.paquet
		pay := paquetDeltaNeufTi40(t, c.i34)
		m.marcherLePaquet(0, FilmPacket{Type: PacketTypeDelta, Size: len(pay)}, pay)
		sc := &movementStateScanner{st: &st, obs: cfg.Obs, paquet: &m.paquet, entites: m.entites,
			tir: nouveauCollecteurTirContinu(&tir)}
		sc.trame(&m.trame)
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
