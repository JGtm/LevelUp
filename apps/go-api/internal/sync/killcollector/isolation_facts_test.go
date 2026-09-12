package killcollector

// isolation_facts_test.go — LA PROJECTION DES FAITS D'ISOLEMENT, SANS FIXTURE DE FILM.
//
// # POURQUOI CES TESTS EXISTENT
//
// Toute la couverture de la projection tenait à un test d'intégration qui se SKIPPE sans les
// films (107 Mo, non versionnés) : ses gardes, ses traductions et son chemin d'erreur n'étaient
// donc vérifiés nulle part en CI. Ceux-ci tournent partout — même patron que
// `positions_test.go`, qui couvre les gardes de la passe de positions pour la même raison.

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/port"
)

// panicWriterIso : un writer qui explose si on l'appelle. Il prouve qu'une garde a coupé AVANT
// toute tentative d'écriture — une assertion sur un compteur ne dirait pas la même chose.
func panicWriterIso(context.Context) (*sql.DB, func(), error) {
	panic("le writer ne doit PAS etre acquis : une garde aurait du couper avant")
}

// TestProjeterFaitsDIsolement_SansEquipes_NeTenteAucuneEcriture — « isolé » se mesure ENTRE
// COÉQUIPIERS, et le film ne porte aucun camp.
//
// Un match dont `match_participants.team_id` est vide sort de la lecture au lieu d'y entrer
// avec des camps devinés. La garde coupe AVANT le lease.
func TestProjeterFaitsDIsolement_SansEquipes_NeTenteAucuneEcriture(t *testing.T) {
	c := &KillSourceCollector{acquireShared: panicWriterIso}
	c.projeterFaitsDIsolement(context.Background(), "m1", materiauDIsolement{},
		MatchIdentities{}, nil)
}

// TestProjeterFaitsDIsolement_SansVieNommee_NeTenteAucuneEcriture — sans vie, le pont
// slot->xuid n'a pas été construit, et un contexte calculé sans lui ne reposerait sur rien.
func TestProjeterFaitsDIsolement_SansVieNommee_NeTenteAucuneEcriture(t *testing.T) {
	c := &KillSourceCollector{acquireShared: panicWriterIso}
	ids := MatchIdentities{Equipes: map[string]int{"111": 0, "222": 0}}
	c.projeterFaitsDIsolement(context.Background(), "m1", materiauDIsolement{}, ids,
		[]persist.KillEventInsert{{TimeMS: 1000, VictimXUID: "111"}})
}

// TestProjeterFaitsDIsolement_EchecDEcriture_NEstPasBloquant — le contrat de l'étape.
//
// Son échec ne doit coûter NI le journal des morts NI les positions : deux écritures déjà
// faites, et bien plus centrales au produit. La fonction ne rend donc aucune erreur — elle
// journalise et compte.
func TestProjeterFaitsDIsolement_EchecDEcriture_NEstPasBloquant(t *testing.T) {
	c := &KillSourceCollector{
		acquireShared: func(context.Context) (*sql.DB, func(), error) {
			return nil, nil, errors.New("lease indisponible")
		},
	}
	ids := MatchIdentities{Equipes: map[string]int{"111": 0}}
	// Aucune panique, aucun retour : la seule sortie possible est le journal.
	c.projeterFaitsDIsolement(context.Background(), "m1", materiauAvecUneVie(), ids, nil)
}

// materiauAvecUneVie : un matériau minimal portant UNE vie nommée.
//
// LE REGISTRE EST LE VRAI (`replay.BuildIdentityRegistry`) : une trajectoire de slot, une mort
// du fil qui la clôt, une table d index. Fabriquer un pont à la main aurait demandé d exporter un
// constructeur de test depuis `replay` — du code de production qui n existe que pour les tests,
// et qui aurait de surcroît court-circuité le nommage.
func materiauAvecUneVie() materiauDIsolement {
	pos := positionsDUneVie()
	return materiauDIsolement{registre: registreDeTest(pos, 0), positions: pos}
}

// TestToLifeRows_TraduitSansRienInventer — la traduction pure vers les lignes écrivables.
func TestToLifeRows_TraduitSansRienInventer(t *testing.T) {
	rows := toLifeRows([]replay.VieNommee{
		{XUID: 111, DebutMS: 0, FinMS: 10_000, Cause: replay.CauseVieMort, NomPar: replay.NomParMort},
		{XUID: 222, DebutMS: 5, FinMS: 30_000, Cause: replay.CauseVieFinFilm, NomPar: replay.NomParFermeture},
	})
	if len(rows) != 2 {
		t.Fatalf("lignes = %d, attendu 2", len(rows))
	}
	if rows[0].XUID != "111" || rows[0].EndCause != persist.CauseFinMort || rows[0].NamedBy != persist.NommeParMort {
		t.Fatalf("ligne 0 = %+v", rows[0])
	}
	// LE SURVIVANT NOMMÉ PAR FERMETURE : les deux colonnes disent deux choses différentes, et
	// c'est tout l'objet de leur séparation.
	if rows[1].EndCause != persist.CauseFinFilm || rows[1].NamedBy != persist.NommeParFermeture {
		t.Fatalf("ligne 1 = %+v : un joueur nomme par fermeture n'est pas mort", rows[1])
	}
}

// TestJournalDesMorts_EcarteUneVictimeNonResolue — un bot, ou un nom que le roster ne résout
// pas, n'a pas de xuid : il ne peut ni être situé dans une équipe ni être joint au journal.
func TestJournalDesMorts_EcarteUneVictimeNonResolue(t *testing.T) {
	out, nonResolues := journalDesMorts([]persist.KillEventInsert{
		{TimeMS: 1000, VictimXUID: "111"},
		{TimeMS: 2000, VictimXUID: ""},          // bot
		{TimeMS: 3000, VictimXUID: "pas-un-id"}, // non decimal
	})
	if len(out) != 1 || out[0].VictimeXUID != 111 {
		t.Fatalf("journal = %+v, attendu la seule mort resolue", out)
	}
	// LE COMPTE DES ECARTEES SORT A PART : un seul compteur global melangerait trois causes
	// qui ne se diagnostiquent pas de la meme facon.
	if nonResolues != 2 {
		t.Fatalf("victimes non resolues = %d, attendu 2", nonResolues)
	}
}

// TestToDeathContextRows_PontNonPublicable_CompteDedie — Q8 (2026-09-07) : un pont refuse
// (IndexDisagreements > 0) ne doit PAS tomber dans « morts sans lieu ». Avant ce compteur
// dedie, cette cause (pont casse pour TOUT le match) se melangeait avec « cette victime
// precise n'a pas de position au film », deux diagnostics differents sous un seul nombre.
func TestToDeathContextRows_PontNonPublicable_CompteDedie(t *testing.T) {
	// UN DESACCORD DE LECTURE se fabrique par la TABLE D INDEX, la seule entree publique qui le
	// porte : le registre le recopie tel quel. Aucun constructeur de test cote production.
	mat := materiauDIsolement{registre: registreDeTest(positionsDUneVie(), 1)}
	ids := MatchIdentities{Equipes: map[string]int{"111": 0}}
	deaths := []persist.KillEventInsert{{TimeMS: 1000, VictimXUID: "111"}}

	rows, ecarts := toDeathContextRows(mat, ids, deaths)

	if rows != nil {
		t.Fatalf("contextes = %+v, attendu aucun : le pont est refuse pour tout le match", rows)
	}
	if ecarts.pontNonPublicable != 1 {
		t.Fatalf("pontNonPublicable = %d, attendu 1", ecarts.pontNonPublicable)
	}
	if ecarts.sansLieu != 0 {
		t.Fatalf("sansLieu = %d, attendu 0 : la cause est le pont, pas l'absence de lieu au "+
			"film — les deux compteurs ne doivent jamais se substituer l'un a l'autre",
			ecarts.sansLieu)
	}
}

// TestPostSyncDeps_SansResolveurDeCarte_LaCaptureEstDesarmee — la garde de la capture au
// post-sync.
//
// UN CATALOGUE ABSENT DÉGRADE, IL NE CASSE PAS : le journal des morts, raison d'être de
// l'étape, ne doit pas dépendre d'une brique tierce.
func TestPostSyncDeps_SansResolveurDeCarte_LaCaptureEstDesarmee(t *testing.T) {
	h := NewPostSyncHook(t.TempDir(), 0)
	if d := h.capture(context.Background(), PostSyncDeps{TitleSlug: "halo_infinite"}); d.Cablee() {
		t.Fatal("capture cablee sans resolveur de carte")
	}
}

// TestCaptureDepuisCatalogue_RefuseUnResolveurNil — l'erreur est RENDUE, jamais avalée : c'est
// à l'appelant de décider s'il dégrade. Une fonction qui rendrait des deps vides en silence
// fabriquerait le silence que ce lot corrige.
func TestCaptureDepuisCatalogue_RefuseUnResolveurNil(t *testing.T) {
	if _, err := CaptureDepuisCatalogue(t.TempDir(), "halo_infinite", nil); err == nil {
		t.Fatal("aucune erreur pour un resolveur nil")
	}
}

// TestCaptureDepuisCatalogue_CatalogueAbsent_RendUneErreur — même règle pour un catalogue
// illisible : l'appelant journalise et dégrade, la fonction ne ment pas.
func TestCaptureDepuisCatalogue_CatalogueAbsent_RendUneErreur(t *testing.T) {
	repo := fakeMapNames{keys: port.MatchMapKeys{Names: []string{"Catalyst"}}}
	if _, err := CaptureDepuisCatalogue(t.TempDir(), "halo_infinite", repo); err == nil {
		t.Fatal("aucune erreur alors que le catalogue de bornes n'existe pas dans ce TempDir")
	}
}

// TestAvecCapture_CableLesDeuxOuAucune — un collecteur ne doit jamais avoir une seule des deux
// dépendances : sans bornes un quantum n'est pas une coordonnée, sans identité de carte on ne
// sait pas quelles bornes chercher.
func TestAvecCapture_CableLesDeuxOuAucune(t *testing.T) {
	c := &KillSourceCollector{}
	if c.AvecCapture(DepsCapture{}).CaptureCablee() {
		t.Fatal("capture cablee avec des deps vides")
	}
	if c.AvecCapture(DepsCapture{Bounds: testMapQuantCatalog()}).CaptureCablee() {
		t.Fatal("capture cablee sans resolveur de carte")
	}
	complet := DepsCapture{
		MapNames: fakeMapNames{keys: port.MatchMapKeys{Names: []string{"Catalyst"}}},
		Bounds:   testMapQuantCatalog(),
	}
	if !c.AvecCapture(complet).CaptureCablee() {
		t.Fatal("capture NON cablee avec des deps completes")
	}
}

// positionsDUneVie : une trajectoire continue de dix secondes sur un slot.
func positionsDUneVie() []filmdec.BipedPosition {
	pos := []filmdec.BipedPosition{}
	for t := int64(0); t <= 10_000; t += 100 {
		pos = append(pos, filmdec.BipedPosition{
			Slot: 1, TimestampUS: uint64(t) * 1000, HasWorld: true,
		})
	}
	return pos
}

// registreDeTest construit le VRAI registre (`replay.BuildIdentityRegistry`) : une trajectoire,
// une mort du fil qui la clôt, une table d'index. `desaccords` injecte un désaccord de lecture
// par la table d'index — la seule entrée publique qui le porte, et celle que la production
// alimente (`replay.ScanPlayerIndices`).
func registreDeTest(pos []filmdec.BipedPosition, desaccords int) replay.IdentityRegistry {
	return replay.BuildIdentityRegistry(replay.IdentityInput{
		Positions: pos,
		Deaths:    []replay.Death{{XUID: 111, TimeMS: 10_000}},
		PlayerIndices: replay.PlayerIndexTable{
			ByXUID: map[uint64]int{111: 0}, Readings: 26, Disagreements: desaccords,
		},
	})
}

// TestRegistreDuCollecteurNommeParElimination — LE COLLECTEUR NOMME CE QU'IL NE NOMMAIT PAS.
//
// # CE QUE CE TEST EXIGE, ET POURQUOI IL JUSTIFIE LE BUMP D'[IsolationDecoderRev]
//
// Avant le lot P2, le pont du collecteur (`replay.ResolveSlotXUID`) ne recevait PAS le roster de
// la feuille : un joueur qui ne meurt jamais n'avait aucune vie nommee, et ses lignes de
// `match_lives` n'existaient tout simplement pas. Le registre, lui, recoit `ids.XUIDs` et ferme
// le cas d'unicite par ELIMINATION. Les deux tables changent donc de CONTENU — c'est cela que la
// revision de decodeur fait re-ecrire.
//
// MUTATION : retirer `RosterXUIDs` de l'entree du registre (positions.go) -> le slot 2 reste
// anonyme, `ViesNommees()` n'en porte qu'une, ROUGE.
func TestRegistreDuCollecteurNommeParElimination(t *testing.T) {
	// Slot 1 : une vie que la mort de 111 termine. Slot 2 : une vie continue, jamais terminee.
	var pos []filmdec.BipedPosition
	for t := int64(0); t <= 10_000; t += 100 {
		pos = append(pos,
			filmdec.BipedPosition{Slot: 1, TimestampUS: uint64(t) * 1000, HasWorld: true},
			filmdec.BipedPosition{Slot: 2, TimestampUS: uint64(t) * 1000, HasWorld: true})
	}
	entree := replay.IdentityInput{
		Positions: pos,
		Deaths:    []replay.Death{{XUID: 111, TimeMS: 10_000}},
		PlayerIndices: replay.PlayerIndexTable{
			ByXUID: map[uint64]int{111: 0}, Readings: 26,
		},
	}

	sansRoster := replay.BuildIdentityRegistry(entree)
	if n := len(sansRoster.ViesNommees()); n != 1 {
		t.Fatalf("sans roster : %d vie(s) nommee(s), attendu 1 (le seul joueur qui meurt)", n)
	}

	// AVEC LE ROSTER DE LA FEUILLE — ce que la production passe depuis le lot P2.
	entree.RosterXUIDs = rosterUint64([]string{"111", "222"})
	avecRoster := replay.BuildIdentityRegistry(entree)
	vies := avecRoster.ViesNommees()
	if len(vies) != 2 {
		t.Fatalf("avec roster : %d vie(s) nommee(s), attendu 2", len(vies))
	}
	var parElimination int
	for _, v := range vies {
		if v.NomPar == replay.NomParElimination {
			parElimination++
			if v.Cause == replay.CauseVieMort {
				t.Fatal("l'elimination a fabrique une MORT : une deduction ajoute une presence, " +
					"elle ne termine jamais une vie")
			}
		}
	}
	if parElimination != 1 {
		t.Fatalf("vies nommees par elimination = %d, attendu 1", parElimination)
	}
}

// TestRegistreDuCollecteurSeTaitADeuxCandidats — LA CONTRE-EPREUVE : deux joueurs du roster sans
// aucune vie, l'unicite disparait, et rien n'est nomme. On n'invente jamais un occupant.
func TestRegistreDuCollecteurSeTaitADeuxCandidats(t *testing.T) {
	var pos []filmdec.BipedPosition
	for t := int64(0); t <= 10_000; t += 100 {
		pos = append(pos,
			filmdec.BipedPosition{Slot: 1, TimestampUS: uint64(t) * 1000, HasWorld: true},
			filmdec.BipedPosition{Slot: 2, TimestampUS: uint64(t) * 1000, HasWorld: true})
	}
	reg := replay.BuildIdentityRegistry(replay.IdentityInput{
		Positions:     pos,
		Deaths:        []replay.Death{{XUID: 111, TimeMS: 10_000}},
		PlayerIndices: replay.PlayerIndexTable{ByXUID: map[uint64]int{111: 0}, Readings: 26},
		RosterXUIDs:   rosterUint64([]string{"111", "222", "333"}),
	})
	if n := len(reg.ViesNommees()); n != 1 {
		t.Fatalf("%d vie(s) nommee(s), attendu 1 : deux candidats libres ne se departagent pas", n)
	}
}
