package grammar

// malleable_du_film_test.go — LE DRAPEAU FINAL DU BLOC MALLEABLE DU BIPEDE SUIT LA VERSION MAJEURE QUE
// LE FILM DECLARE (plan `.ai/archive/V7.5/PLAN_RI_RESIDU_IMAGES_CLES_2026-10-14.md`, R2).

import "testing"

// TestLaVersionMajeureDecideDuDrapeauFinal : sur les bobines par build, la regle vaut pour la seule bobine
// HI_1_8_0 (majeure 0x25), ni pour la bobine HI_1_4_1 (0x21) ni pour les builds suivants (>= 0x26) ;
// une bobine sans `chunk_00` garde la grammaire de l executable. MUTATIONS — `<= 0x25`, ou 0x26 : ROUGE.
func TestLaVersionMajeureDecideDuDrapeauFinal(t *testing.T) {
	attendu := map[string]bool{"60ae07c4": true}
	for _, court := range closureMiniFilms() {
		f := bobineParBuild(t, court)
		v, _ := FilmMajorVersion(f)
		if got := malleableSansDrapeauFinal(f); got != attendu[court] {
			t.Errorf("%s (majeure %#x) : sans drapeau final %v, attendu %v", court, v, got, attendu[court])
		}
		if got := NewFilmContext(f).ProfilDeBalayage().Grammaire.MalleableSansDrapeauFinal; got != attendu[court] {
			t.Errorf("%s : le profil du contexte porte %v, attendu %v", court, got, attendu[court])
		}
	}
	if malleableSansDrapeauFinal(bobineParBuild(t, "000d5950")) {
		t.Error("bobine sans chunk_00 : la grammaire de l executable doit rester")
	}
}

// TestLeBlocMalleableSansDrapeauFinalLitUnBitDeMoins : portes fermees, niveau 1, le bloc lit 1 + 11 +
// 7 + 1 = 20 bits ; sans drapeau final, 19. MUTATION — ignorer le champ : ROUGE.
func TestLeBlocMalleableSansDrapeauFinalLitUnBitDeMoins(t *testing.T) {
	for _, c := range []struct {
		sans bool
		bits int
	}{{false, 20}, {true, 19}} {
		br := LecteurSur(make([]byte, 8))
		ctx := ContexteParDefaut()
		ctx.Profil.Grammaire.MalleableSansDrapeauFinal = c.sans
		br.PoserContexte(ctx)
		consumeBipedMalleablePropertyBlock(br, 1)
		if br.BitPos() != c.bits {
			t.Errorf("sans drapeau final %v : %d bits lus, attendu %d", c.sans, br.BitPos(), c.bits)
		}
	}
}
