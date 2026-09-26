package service

// tactical_service_ventilation_test.go — CE QUE LA PAGE ANNONCE DES MATCHS QU'ELLE NE PEUT
// PAS MONTRER.
//
// Extrait de tactical_service_lectures_test.go le 2026-09-07 (revue de 7.10), quand l'ajout
// des tests d'eligibilite l'a pousse au-dela du seuil de 500 lignes. La coupure suit une
// frontiere nette : LA-BAS ce que les lectures d'artefact RENDENT, ICI ce qu'elles disent de
// ce qui MANQUE.

import (
	"context"
	"sort"
	"testing"

	"levelup/go-api/internal/domain"
)

// ─── VENTILATION DES MATCHS NON RETENUS ────────────────────────────────────────

// universEligibilite pose des matchs dont on choisit, un par un, si la FILE DE CUISSON les
// reprendra.
func universEligibilite(dans map[string]bool) domain.TacticalUnivers {
	u := domain.TacticalUnivers{Equipes: domain.EquipesParMatch{}}
	ids := make([]string, 0, len(dans))
	for id := range dans {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		u.Matchs = append(u.Matchs, domain.TacticalMatch{
			MatchID: id, Outcome: domain.OutcomeWin, Mesure: true, EligibleALaCuisson: dans[id],
		})
		u.Equipes[id] = map[string]int{tsMoi: 0, tsAmi: 0, tsAdv: 1, tsAdv2: 1}
	}
	return u
}

// TestVentilation_EnAttenteContreNonCuisable — LES DEUX ABSENCES NE SE DISENT PAS PAREIL.
//
// Un match sans sidecar que la FILE REPRENDRA sera cuit au fil de l'eau : c'est un
// TRAITEMENT EN COURS, et l'utilisateur n'a qu'a attendre. Un match que rien ne cuira — film
// expire cote serveur, registre incapable de le dater, ou trop ancien — est une DONNEE NON
// DISPONIBLE : il n'y a rien a attendre. « N mesures sur M » servait le meme message aux
// deux.
func TestVentilation_EnAttenteContreNonCuisable(t *testing.T) {
	univ := universEligibilite(map[string]bool{
		"cuit": true, "attend1": true, "attend2": true, "vieux": false,
	})
	store := &mockRasterStore{sidecars: map[string]*domain.TacticalRasterSidecar{
		"cuit": sidecarSpawn("cuit", 0.25, 0.25),
	}}
	svc := svcArtefact(univ, store)
	out := lireTemps(t, svc, "", "cuit", "attend1", "attend2", "vieux")

	if out.MatchsRetenus != 1 {
		t.Fatalf("matchs_retenus = %d, attendu 1", out.MatchsRetenus)
	}
	if out.MatchsEnAttente != 2 {
		t.Fatalf("matchs_en_attente = %d, attendu 2 : deux matchs sans artefact que la file "+
			"reprendra", out.MatchsEnAttente)
	}
	if out.MatchsNonCuisables != 1 {
		t.Fatalf("matchs_non_cuisables = %d, attendu 1 : un match que rien ne cuira",
			out.MatchsNonCuisables)
	}
}

// TestVentilation_LInvariantDeSomme — matchs_filtres = retenus + en_attente + non_cuisables.
//
// UNE VENTILATION QUI NE SOMME PAS AU TOTAL CACHE UN TROISIEME CAS QU'ON N'A PAS NOMME, et
// le pied de carte affiche alors des nombres qui ne se recomposent pas. Le test le verifie
// sur un univers ou les trois situations coexistent — c'est le seul cas ou l'oubli se voit.
func TestVentilation_LInvariantDeSomme(t *testing.T) {
	univ := universEligibilite(map[string]bool{
		"c1": true, "c2": true, "a1": true, "a2": true, "a3": true, "v1": false, "v2": false,
	})
	store := &mockRasterStore{sidecars: map[string]*domain.TacticalRasterSidecar{
		"c1": sidecarSpawn("c1", 0.25, 0.25),
		"c2": sidecarSpawn("c2", 0.65, 0.25),
	}}
	svc := svcArtefact(univ, store)
	out := lireTemps(t, svc, "", "c1", "c2", "a1", "a2", "a3", "v1", "v2")

	somme := out.MatchsRetenus + out.MatchsEnAttente + out.MatchsNonCuisables
	if somme != out.MatchsFiltres {
		t.Fatalf("retenus(%d) + en_attente(%d) + non_cuisables(%d) = %d, attendu "+
			"matchs_filtres = %d", out.MatchsRetenus, out.MatchsEnAttente,
			out.MatchsNonCuisables, somme, out.MatchsFiltres)
	}
	if out.MatchsRetenus != 2 || out.MatchsEnAttente != 3 || out.MatchsNonCuisables != 2 {
		t.Fatalf("ventilation = %d/%d/%d, attendu 2/3/2", out.MatchsRetenus,
			out.MatchsEnAttente, out.MatchsNonCuisables)
	}
}

// TestVentilation_LaLectureDeBaseNeVentilePas — LES COMPTEURS SONT A ZERO SOUS « ou je
// meurs », et ce n'est pas un oubli.
//
// Une lecture de base lit le JOURNAL DES MORTS : elle n'attend aucun fichier, donc rien
// n'y est « en attente » ni « non cuisable ». Les remplir la ferait mentir dans l'autre
// sens — annoncer un traitement en cours a qui a deja toute sa reponse. L'invariant de
// somme ne vaut donc QUE pour les lectures d'artefact.
func TestVentilation_LaLectureDeBaseNeVentilePas(t *testing.T) {
	univ := universEligibilite(map[string]bool{"m1": true, "m2": false, "m3": true})
	pos, ev := posEtEvents(univ)
	repo := &mockTacticalRepo{univ: univ, pos: pos, ev: ev}
	svc := NewTacticalService(repo, capsCompletes(), tsMoi)

	out, err := svc.Raster(context.Background(), domain.TacticalRasterRequest{
		MapID: "streets", Question: domain.TacticalQuestionMorts, Qui: domain.TacticalQuiMoi,
		Scope: domain.TacticalScope{MatchIDs: []string{"m1", "m2", "m3"}},
	})
	if err != nil {
		t.Fatalf("lecture morts: %v", err)
	}
	if out.MatchsEnAttente != 0 || out.MatchsNonCuisables != 0 {
		t.Fatalf("en_attente=%d non_cuisables=%d, attendu 0 et 0 : une lecture de base "+
			"n'attend aucun artefact", out.MatchsEnAttente, out.MatchsNonCuisables)
	}
	if out.MatchsRetenus != 3 {
		t.Fatalf("matchs_retenus = %d, attendu 3 : les matchs MESURES, pas ceux qui ont un "+
			"artefact", out.MatchsRetenus)
	}
}

// TestVentilation_SansFenetreConnue_RienNEstDeclareNonCuisable — le service sans provider de
// retention.
//
// NIL VAUT « FENETRE ILLIMITEE », comme 0 chez la purge et chez la file — pas « pas de
// ventilation ». Ne pas connaitre la fenetre ne doit jamais faire dire « jamais cuit » a un
// match que la file reprendrait : la degradation sure est celle qui promet la cuisson, pas
// celle qui la refuse.
func TestVentilation_SansFenetreConnue_RienNEstDeclareNonCuisable(t *testing.T) {
	univ := universEligibilite(map[string]bool{"cuit": true, "attend": true})
	store := &mockRasterStore{sidecars: map[string]*domain.TacticalRasterSidecar{
		"cuit": sidecarSpawn("cuit", 0.25, 0.25),
	}}
	repo := &mockTacticalRepo{univ: univ}
	// AUCUN `WithRetentionMois` : c'est le cas teste.
	svc := NewTacticalService(repo, capsOccupation(), tsMoi).WithRasterStore(store)
	out := lireTemps(t, svc, "", "cuit", "attend")

	if repo.vuUniv.RetentionMois != 0 {
		t.Fatalf("RetentionMois = %d, attendu 0 (illimitee) sans provider",
			repo.vuUniv.RetentionMois)
	}
	if out.MatchsEnAttente != 1 || out.MatchsNonCuisables != 0 {
		t.Fatalf("en_attente=%d non_cuisables=%d, attendu 1 et 0", out.MatchsEnAttente,
			out.MatchsNonCuisables)
	}
}
