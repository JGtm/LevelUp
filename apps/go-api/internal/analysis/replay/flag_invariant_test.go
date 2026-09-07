package replay

import (
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// flag_invariant_test.go — L'INVARIANT DUR ET LES RETOURS (revue DRAPEAUX-R1).
//
// En CTF on RENVOIE son drapeau, on ne le porte pas. La regle du mode prime sur toute inference
// geometrique : un portage n'est JAMAIS pose sur le drapeau de l'equipe de son porteur. Les
// tests ci-dessous figent le refus, son unique repli, l'abstention, et le fait qu'un retour
// remet a zero les DEUX etats du sol (`sol` ET `enJeu`).

// flagInvariantSpawns : socle de l'equipe 1 en (0,0), socle de l'equipe 0 en (100,100).
func flagInvariantSpawns() []FlagSpawn {
	return []FlagSpawn{{Team: 1, X: 0, Y: 0}, {Team: 0, X: 100, Y: 100}}
}

// TestFlagInvariantJamaisSonPropreDrapeau — LE FAIT `64e8adfa`, reduit au banc.
//
// Un joueur de l'equipe 0 ramasse a 2 m de SON PROPRE socle, l'autre drapeau etant au sol trop
// loin pour la regle 2 et les deux drapeaux « en jeu » pour la regle 3. Le repli geometrique
// designe son propre drapeau : il est REFUSE, et l'autre — le seul candidat restant — est pris.
func TestFlagInvariantJamaisSonPropreDrapeau(t *testing.T) {
	tracks := []Track{
		flagTestTrack(12, "1", 0, 99, 2, 2),              // equipe 0 : vole le drapeau de l'equipe 1
		flagTestPath(14, "2", 0, 99, 30, 98, 98, 40, 40), // equipe 1 : sort celui de l'equipe 0
		flagTestTrack(16, "3", 0, 99, 98, 98),            // equipe 0 : ramasse a SON socle
	}
	scan := FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 1000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
			{TimeMS: 2000, Slot: 14, Stat: objectiveevents.StatFlagSteals},
			{TimeMS: 5000, Slot: 16, Stat: objectiveevents.StatFlagGrabs},
		},
		Identity: objectiveevents.FlatRoundIdentity(map[int]string{12: "1", 14: "2", 16: "3"}),
		Spawns:   flagInvariantSpawns(),
		TeamOf:   map[string]int{"1": 0, "2": 1, "3": 0},
	}

	got, cov := buildFlagCarries(scan, flagTestCtx(tracks, nil, 100))
	if cov.Carries != 3 || !cov.Balanced() {
		t.Fatalf("couverture %+v : 3 portages attendus, invariant tenu", *cov)
	}
	if cov.OwnFlagRefused != 1 {
		t.Errorf("ownFlagRefused = %d, attendu 1 : le repli designait le drapeau du porteur",
			cov.OwnFlagRefused)
	}
	if cov.Unresolved != 0 {
		t.Errorf("unresolved = %d, attendu 0 : il restait un candidat", cov.Unresolved)
	}
	// « 3 » est de l'equipe 0 : il ne peut porter QUE le drapeau de l'equipe 1.
	assertPorteurs(t, flagOfTeam(t, got, 1), []string{"1", "3"})
	assertPorteurs(t, flagOfTeam(t, got, 0), []string{"2"})
	for _, fl := range got {
		for _, s := range fl.Spans {
			if !(s.State == FlagStateCarried || s.State == FlagStateCarriedOpen) || s.XUID == nil {
				continue
			}
			if scan.TeamOf[*s.XUID] == fl.Team {
				t.Errorf("le porteur %s (equipe %d) tient le drapeau de son propre camp",
					*s.XUID, fl.Team)
			}
		}
	}
}

// TestFlagInvariantSansEquipeConnueSeTait — la table vide ne change RIEN.
//
// L'artefact hors ligne (CLI sans faits de match, ouvrier sans base) doit rester celui d'avant :
// sans equipe LUE, on ne refuse rien — on ne remplace pas une inference par une autre.
func TestFlagInvariantSansEquipeConnueSeTait(t *testing.T) {
	tracks := []Track{
		flagTestTrack(12, "1", 0, 99, 2, 2),
		flagTestPath(14, "2", 0, 99, 30, 98, 98, 40, 40),
		flagTestTrack(16, "3", 0, 99, 98, 98),
	}
	scan := FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 1000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
			{TimeMS: 2000, Slot: 14, Stat: objectiveevents.StatFlagSteals},
			{TimeMS: 5000, Slot: 16, Stat: objectiveevents.StatFlagGrabs},
		},
		Identity: objectiveevents.FlatRoundIdentity(map[int]string{12: "1", 14: "2", 16: "3"}),
		Spawns:   flagInvariantSpawns(),
	}

	got, cov := buildFlagCarries(scan, flagTestCtx(tracks, nil, 100))
	if cov.OwnFlagRefused != 0 || cov.Unresolved != 0 {
		t.Errorf("couverture %+v : sans equipe lue, l'invariant se tait", *cov)
	}
	// Le repli geometrique reprend la main : « 3 » tombe sur le socle le plus proche.
	assertPorteurs(t, flagOfTeam(t, got, 0), []string{"2", "3"})
}

// TestFlagInvariantSansCandidatNInventeRien — L'ABSTENTION.
//
// Sur une carte a UN SEUL socle (variante « drapeau neutre » degradee en un socle d'equipe), le
// refus ne laisse aucun candidat : le portage sort NON ATTRIBUE et n'est publie sur AUCUN
// drapeau. On n'invente jamais un drapeau.
func TestFlagInvariantSansCandidatNInventeRien(t *testing.T) {
	tracks := []Track{flagTestTrack(12, "1", 0, 99, 2, 2)}
	scan := FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 1000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
		},
		Identity: objectiveevents.FlatRoundIdentity(map[int]string{12: "1"}),
		Spawns:   []FlagSpawn{{Team: 0, X: 0, Y: 0}},
		TeamOf:   map[string]int{"1": 0},
	}

	got, cov := buildFlagCarries(scan, flagTestCtx(tracks, nil, 100))
	if cov.Unresolved != 1 || cov.OwnFlagRefused != 1 {
		t.Errorf("couverture %+v : attendu 1 refus et 1 portage non attribue", *cov)
	}
	if cov.Carries != 1 || !cov.Balanced() {
		t.Errorf("couverture %+v : le portage reste COMPTE, seul son drapeau est inconnu", *cov)
	}
	for _, fl := range got {
		for _, s := range fl.Spans {
			if s.State == FlagStateCarried || s.State == FlagStateCarriedOpen {
				t.Errorf("un portage non attribue a ete publie sur le drapeau d'equipe %d", fl.Team)
			}
		}
	}
}

// TestFlagRetourRemetLeSolEtEnJeu — CONSTAT C4 : les DEUX etats, jamais l'un sans l'autre.
//
// Un drapeau lache puis RENVOYE chez lui par un `flag_returns` credite n'est plus au sol. Tant
// que `sol` gardait sa position de lacher, la regle 2 — PRIORITAIRE — rattachait a ce drapeau
// toute prise faite a moins de 8 m de cet endroit, alors que le document le publie `home` : la
// limite ecrite dans le fichier (« ils ne peuvent que FAIRE TAIRE la troisieme regle ») etait
// vraie pour `enJeu` et FAUSSE pour `sol`, qui pouvait faire MENTIR la deuxieme.
//
// L'EQUIPE EST VOLONTAIREMENT INCONNUE ICI. Sur une carte a deux drapeaux, l'invariant dur ne
// laisse qu'un candidat et masquerait l'effet mesure ; sans equipe lue, il se tait, et c'est
// bien l'etat du sol qui tranche — la seule facon d'isoler ce constat.
func TestFlagRetourRemetLeSolEtEnJeu(t *testing.T) {
	tracks := []Track{
		flagTestPath(12, "1", 0, 99, 20, 2, 2, 96, 96), // vole en (2,2), meurt en (96,96)
		flagTestTrack(14, "2", 0, 99, 96, 96),          // ramasse A L'ENDROIT DU LACHER
	}
	deaths := []Death{{XUID: 1, TimeMS: 2000}}
	scan := FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 1000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
			{TimeMS: 3000, Slot: 12, Stat: objectiveevents.StatFlagReturns},
			{TimeMS: 5000, Slot: 14, Stat: objectiveevents.StatFlagGrabs},
		},
		Identity: objectiveevents.FlatRoundIdentity(map[int]string{12: "1", 14: "2"}),
		Spawns:   flagInvariantSpawns(),
	}

	got, cov := buildFlagCarries(scan, flagTestCtx(tracks, deaths, 100))
	if cov.Carries != 2 || !cov.Balanced() {
		t.Fatalf("couverture %+v : 2 portages attendus, invariant tenu", *cov)
	}
	// Le drapeau de l'equipe 1 est RENTRE a la frame 30 : la prise de la frame 50 ne peut pas
	// lui etre rattachee par une position de lacher que le retour a rendue caduque.
	rentre := flagOfTeam(t, got, 1)
	assertPorteurs(t, rentre, []string{"1"})
	if !flagHomeAt(rentre, 30) {
		t.Fatalf("le drapeau de l'equipe 1 devait rentrer a la frame 30, etats %v",
			flagStatesOf(rentre))
	}
}

// TestFlagOverlapsComptesParDrapeau — CONSTAT C3.
//
// Deux portages du MEME drapeau qui se recouvrent sont une incoherence : un drapeau n'a qu'un
// porteur. Le compte portait sur « plus de deux portages, tous drapeaux confondus » et ratait
// donc le cas nominal d'un CTF a deux drapeaux.
func TestFlagOverlapsComptesParDrapeau(t *testing.T) {
	tracks := []Track{
		flagTestTrack(12, "1", 0, 99, 2, 2),
		flagTestTrack(14, "2", 0, 99, 4, 4),
	}
	deaths := []Death{{XUID: 1, TimeMS: 6000}, {XUID: 2, TimeMS: 7000}}
	scan := FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 1000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
			{TimeMS: 2000, Slot: 14, Stat: objectiveevents.StatFlagSteals},
		},
		Identity: objectiveevents.FlatRoundIdentity(map[int]string{12: "1", 14: "2"}),
		Spawns:   flagInvariantSpawns(),
		TeamOf:   map[string]int{"1": 0, "2": 0},
	}

	_, cov := buildFlagCarries(scan, flagTestCtx(tracks, deaths, 100))
	if cov.Overlaps == 0 || cov.ClosedOverlaps == 0 {
		t.Errorf("couverture %+v : deux portages du MEME drapeau se recouvrent — l'incoherence "+
			"doit se compter, elle etait invisible tant que le seuil ignorait `flagIndex`", *cov)
	}
}

// TestAttachFlagCarriesDescendLesEquipesJusquAuScan — LE MAILLON QUE PERSONNE NE GARDAIT (W1).
//
// L'invariant dur ne vaut que si la table des equipes lui parvient, et elle traverse DEUX
// maillons : `replaybuild` -> `FlagInput` (garde chez l'appelant, `flagidentity_test.go`) puis
// `FlagInput` -> `FlagCarryScan`, dans `attachFlagCarries`. Le second n'avait aucun garde-rail :
// retirer la ligne `TeamOf: in.TeamOf` laissait les 166 paquets VERTS, et le premier essai du
// lot avait justement livre `ownFlagRefused = 0` pour cette raison. Un chainon muet ne casse
// aucun test unitaire — celui-ci le casse.
//
// LA PREUVE EST UN EFFET, PAS UN CHAMP : on n'observe pas `scan.TeamOf` (il est interne), on
// verifie que l'invariant A REFUSE. Il ne peut refuser que si la table est arrivee.
func TestAttachFlagCarriesDescendLesEquipesJusquAuScan(t *testing.T) {
	// Un compteur du statborg qui monte a 1 : c'est l'increment que l'oracle date.
	rec := func(ms, slot, comp int) objectiveevents.StatRecord {
		return objectiveevents.StatRecord{TimeMS: ms, Slot: slot, Round: 0,
			Comps: map[int]objectiveevents.StatValue{comp: {A: 1}}}
	}
	const (
		compCaptures = 21
		compGrabs    = 22
		compSteals   = 24
	)
	in := FlagInput{
		Scanned: true,
		Records: []objectiveevents.StatRecord{
			rec(1000, 12, compSteals),   // « 1 » (equipe 0) vole le drapeau de l'equipe 1
			rec(2000, 14, compSteals),   // « 2 » (equipe 1) vole celui de l'equipe 0
			rec(5000, 16, compGrabs),    // « 3 » (equipe 0) ramasse A SON PROPRE SOCLE
			rec(9000, 12, compCaptures), // la capture, sans quoi le film n'est pas reconnu CTF
		},
		Bursts:   []int{9000},
		Spawns:   flagInvariantSpawns(),
		Identity: objectiveevents.FlatRoundIdentity(map[int]string{12: "1", 14: "2", 16: "3"}),
		TeamOf:   map[string]int{"1": 0, "2": 1, "3": 0},
	}
	doc := &ReplayDocument{
		FrameCount: 100,
		Coverage:   &Coverage{},
		Tracks: []Track{
			flagTestTrack(12, "1", 0, 99, 2, 2),
			flagTestTrack(14, "2", 0, 99, 98, 98),
			flagTestTrack(16, "3", 0, 99, 98, 98),
		},
	}

	attachFlagCarries(doc, Options{Flag: in}, OwnerReport{},
		replayClock{origin: 0, step: 100_000, frames: 100})

	cov := doc.Coverage.FlagCarries
	if cov == nil {
		t.Fatalf("aucune couverture publiee : le film temoin n'a pas ete reconnu comme du CTF")
	}
	if cov.Carries != 3 {
		t.Fatalf("couverture %+v : 3 portages attendus — sans eux le test ne prouve rien", *cov)
	}
	if cov.OwnFlagRefused != 1 {
		t.Errorf("ownFlagRefused = %d, attendu 1 : la table des equipes n'a pas atteint le calque. "+
			"C'est le maillon `FlagInput` -> `FlagCarryScan` d'`attachFlagCarries` qui est muet, "+
			"et l'invariant « jamais son propre drapeau » se tait avec lui", cov.OwnFlagRefused)
	}
	// Et l'effet se lit dans le document : « 3 » (equipe 0) ne tient PAS le drapeau de l'equipe 0.
	assertPorteurs(t, flagOfTeam(t, doc.FlagCarries, 0), []string{"2"})
	assertPorteurs(t, flagOfTeam(t, doc.FlagCarries, 1), []string{"1", "3"})
}
