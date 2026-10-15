package grammar

// relecture_a_l_etendue.go — L ASSISTANT UNIQUE DE RELECTURE D UNE OCCURRENCE DE COMPOSANT A SON
// ETENDUE (plan `.ai/PLAN_RI_ETAT_COMPLET_IMAGES_CLES_2026-10-08.md`, D1.1.1).
//
// La phase des images-cles range chaque occurrence d un record d etat complet avec son etendue
// ([lecture.Composant] : premier bit et longueur). Un canal qui veut la VALEUR d une occurrence la
// relit, a son premier bit, sous le cadre que la marche avait pose en la traversant : le contexte de
// lecture du film, l etat complet ([Lecteur.etatComplet]) et la portee `DAT_144e61ea0`
// ([Lecteur.portee]) que la boucle de composants d etat complet pose sur toute sa duree
// (`FUN_142e2c690`, [traverserSousLaPortee]). Relue sous un autre cadre, une occurrence lirait
// d autres largeurs que celles qui l ont bornee.
//
// C EST LA SEULE ECRITURE de ces deux champs hors de la marche d etat complet
// (`keyframe_fullstate_loop.go`) : garde-rails `portee_ecriture_guard_test.go` (AST, tous les
// fichiers du paquet) et `TestEtatCompletPoseParLaSeuleMarcheDEtatComplet` (production). Les
// relectures de la phase (canal de l etat complet du bipede, instruments) passent par ici.

import "levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"

// relecteurSur rend un lecteur sur `pay` sous le contexte de lecture `ctx`, hors etat complet :
// le cadre d un relecteur d etat par defaut ([lireEquipeA]).
func relecteurSur(pay []byte, ctx ContexteDeLecture) *Lecteur {
	br := LecteurSur(pay)
	br.PoserContexte(ctx)
	return br
}

// relecteurDEtatComplet rend un lecteur sur `pay` sous le cadre de la boucle de composants d etat
// complet : contexte `ctx`, etat complet et portee poses, observation `obs` (nil : aucune).
func relecteurDEtatComplet(pay []byte, ctx ContexteDeLecture, obs *Observation) *Lecteur {
	br := relecteurSur(pay, ctx)
	br.etatComplet = true
	br.portee = true
	br.obs = obs
	return br
}

// relireLOccurrence relit l occurrence `co` d un record d etat complet de l archetype `arch`
// (index de type `ti`) a son premier bit, par le dispatch de production
// ([consumeByNameCapturing]), sous le cadre de [relecteurDEtatComplet] ; les crochets de `obs`
// recoivent ses valeurs. Elle rend FAUX quand la relecture ne tient pas l etendue que la marche a
// donnee a l occurrence (lecteur non porte, ou autre longueur) : l appelant le compte, ce n est
// jamais une valeur.
func relireLOccurrence(pay []byte, ctx ContexteDeLecture, arch Archetype, ti int, co lecture.Composant,
	obs *Observation) bool {
	br := relecteurDEtatComplet(pay, ctx, obs)
	br.SetBitPos(int(co.Debut))
	_, _, _, porte := consumeByNameCapturing(br, arch.component(int(co.Index)), uint32(ti), //nolint:gosec // archetype < 64
		arch.Level(int(co.Index)))
	return porte && br.BitPos()-int(co.Debut) == int(co.Bits)
}
