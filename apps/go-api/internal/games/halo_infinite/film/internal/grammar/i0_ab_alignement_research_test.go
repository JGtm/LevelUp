package grammar

// i0_ab_alignement_research_test.go — QUELLE VALEUR DU PROFIL DEPLACE LES LECTEURS DERRIERE i0.
// L INSTRUMENT QUI A NOMME LA REGRESSION DU 2026-09-17.
//
// # CE QU IL A SERVI A TRANCHER
//
// L equivalence des 20 films du lot 3.4.1 faisait bouger quatre etapes que le brief
// n attendait pas — `abilityImpulses`, `grappleReads.stats`, `pads`, `vehicles` — dont une
// PERTE publiee (`a521164d` : une impulsion de moins, artefact -14 octets). Trois valeurs du
// profil que `killsource` transmet a la cuisson pouvaient en etre la cause. Cet instrument les
// balaie UNE PAR UNE sur le meme film, et il a rendu le verdict :
//
//	largeurs d axe 14/14/14 -> celles de la carte   0 impulsion -> 0   SANS EFFET
//	`param_4` de 0 a 5                              0 partout          SANS EFFET
//	`Traversal.IndexW` = 1 / 2 / 3                  0 / 1 / 0          C EST ELLE
//
// La largeur du MOT DE POIGNEE n a aucune source lue ; le lot 3.4.1 l avait emportee en
// demotant l inference des largeurs d AXE en oracle. Elle est redevenue une valeur DECIDEE par
// le balayage (`repli_largeur_mot_de_poignee_inferee` au registre), et la confondre avec la
// largeur d INDEX DE PLAGE — celle de la carte — etait D5 (3.4.1), fermee depuis.
//
// # POURQUOI IL RESTE AU DEPOT
//
// Parce que la question se reposera : chaque fois qu une valeur du profil cessera d etre
// devinee, il faudra savoir laquelle deplace quoi. Il coute une variable d environnement et
// une trentaine de secondes sur un film, il n asserte RIEN et ne cuit aucun artefact — c est un
// instrument de mesure, comme les autres `*_research_test.go` du paquet.
//
// LECTURE SEULE, garde par I0AB_FILM — saute partout ailleurs, CI comprise :
//
//	I0AB_FILM=<repo>/data/cache/film_chunks/a521164d \
//	  go test ./internal/games/halo_infinite/film/internal/grammar/ -run '^TestI0AlignementAB$' -v

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

const i0ABEnv = "I0AB_FILM"

// largeursUniformesDAvant : ce que `Movement.AbsoluteAxisW` imposait a TOUTES les cartes.
var largeursUniformesDAvant = profile.PrecisionDescriptor{IndexW: 1, AxisW: [3]uint{14, 14, 14}}

func TestI0AlignementAB(t *testing.T) {
	dir := os.Getenv(i0ABEnv)
	if dir == "" {
		t.Skipf("%s absent : mesure sautee", i0ABEnv)
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("film %s : %v", dir, err)
	}
	nom := filepath.Base(dir)

	apres := contexteDeBobine(film)
	lay, _, errL := DetectI0LayoutOf(film)
	if errL != nil {
		t.Fatalf("decoupage i0 de %s : %v", nom, errL)
	}
	t.Logf("%s : decoupage lu dans le film %s", nom, lay)
	pApres := apres.ProfilDeBalayage()
	pApres.PoserLargeursObjetDuMondeDepuisDecoupage(lay)
	apres.PoserProfilDeBalayage(pApres)

	avant := contexteDeBobine(film)
	pAvant := avant.ProfilDeBalayage()
	pAvant.PoserLargeursObjetDuMonde(largeursUniformesDAvant)
	avant.PoserProfilDeBalayage(pAvant)

	t.Logf("AVANT largeurs %v indexW=%d (l uniforme de `Movement.AbsoluteAxisW`)",
		largeursUniformesDAvant.AxisW, largeursUniformesDAvant.IndexW)
	t.Logf("APRES largeurs %v indexW=%d (la carte)",
		pApres.LargeursObjetDuMonde().AxisW, pApres.LargeursObjetDuMonde().IndexW)

	impA, stA, errA := ScanAbilityImpulses(avant)
	impB, stB, errB := ScanAbilityImpulses(apres)
	if errA != nil || errB != nil {
		t.Fatalf("impulsions : avant %v / apres %v", errA, errB)
	}
	t.Logf("IMPULSIONS  records %d -> %d · masqueI57 %d -> %d · masqueI59 %d -> %d",
		stA.Records, stB.Records, stA.WithI57, stB.WithI57, stA.WithI59, stB.WithI59)
	t.Logf("IMPULSIONS  LUES %d -> %d · NON LUES %d -> %d · tag1 %d -> %d · absent %v -> %v",
		stA.Read, stB.Read, stA.Unread, stB.Unread, stA.Tag1, stB.Tag1, stA.Absent, stB.Absent)
	t.Logf("IMPULSIONS  publiees par le balayage : %d -> %d", len(impA), len(impB))
	diffImpulsions(t, impA, impB)

	grA, gA, errGA := ScanGrappleReads(avant)
	grB, gB, errGB := ScanGrappleReads(apres)
	if errGA != nil || errGB != nil {
		t.Fatalf("grappin : avant %v / apres %v", errGA, errGB)
	}
	t.Logf("GRAPPIN     records %d -> %d · masqueI59 %d -> %d · LUES %d -> %d · NON LUES %d -> %d",
		gA.Records, gB.Records, gA.WithI59, gB.WithI59, gA.Read, gB.Read, gA.Unread, gB.Unread)
	t.Logf("GRAPPIN     tag3 %d -> %d · corps casses %d -> %d · lectures rendues %d -> %d",
		gA.Tag3, gB.Tag3, gA.BodyBroken, gB.BodyBroken, len(grA), len(grB))

	// LE SECOND SUSPECT, ET C EST LUI : `param_4`. Il n est PAS une largeur d axe — il fait
	// varier la largeur de TROIS composants — et le profil que `killsource` transmet a la
	// cuisson le porte. Son balayage se juge desormais sur le profil LU, il peut donc retenir
	// une autre valeur, et cette valeur atteint tous les balayages de l artefact.
	for iw := uint(1); iw <= 3; iw++ {
		p := apres.ProfilDeBalayage()
		p.Mouvement.Traversal.IndexW = iw
		fc := contexteDeBobine(film)
		fc.PoserProfilDeBalayage(p)
		imp, st, e := ScanAbilityImpulses(fc)
		gr, g, e2 := ScanGrappleReads(fc)
		if e != nil || e2 != nil {
			t.Fatalf("traversal.IndexW=%d : %v / %v", iw, e, e2)
		}
		t.Logf("traversal.IndexW=%d : impulsions lues=%d rendues=%d · grappin lues=%d tag3=%d casses=%d rendues=%d",
			iw, st.Read, len(imp), g.Read, g.Tag3, g.BodyBroken, len(gr))
	}
	for r := uint32(0); r <= 5; r++ {
		p := apres.ProfilDeBalayage()
		p.PoserParamEtat(r)
		fc := contexteDeBobine(film)
		fc.PoserProfilDeBalayage(p)
		imp, st, e := ScanAbilityImpulses(fc)
		gr, g, e2 := ScanGrappleReads(fc)
		if e != nil || e2 != nil {
			t.Fatalf("param_4=%d : %v / %v", r, e, e2)
		}
		t.Logf("param_4=%d : impulsions lues=%d rendues=%d · grappin lues=%d tag3=%d casses=%d rendues=%d",
			r, st.Read, len(imp), g.Read, g.Tag3, g.BodyBroken, len(gr))
	}
}

// diffImpulsions nomme les lectures qui apparaissent et celles qui disparaissent.
// `itoa` est celui du harnais de mesure du paquet (`lot1_visee_calib_research_test.go`).
func diffImpulsions(t *testing.T, avant, apres []types.AbilityImpulse) {
	t.Helper()
	cle := func(i types.AbilityImpulse) string {
		p := "0"
		if i.Predicted {
			p = "1"
		}
		return "slot=" + itoa(int(i.Slot)) + " chunk=" + itoa(i.Chunk) +
			" paquet=" + itoa(i.PacketIndex) + " us=" + itoa(int(i.TimestampUS)) + " predit=" + p
	}
	a := map[string]int{}
	for _, i := range avant {
		a[cle(i)]++
	}
	for _, i := range apres {
		if a[cle(i)] > 0 {
			a[cle(i)]--
			continue
		}
		t.Logf("  + APPARAIT  %s", cle(i))
	}
	for k, n := range a {
		for ; n > 0; n-- {
			t.Logf("  - DISPARAIT %s", k)
		}
	}
}
