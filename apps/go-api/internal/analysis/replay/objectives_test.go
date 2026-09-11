package replay

import (
	"encoding/json"
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// objectives_test.go — le calque des actions d'objectif.
//
// Ces tests sont PURS : aucun film, aucune base. Ce qu'ils gardent, c'est la conversion vers
// l'axe de temps du rejeu et l'invariant de couverture, qui sont les deux endroits ou une
// action peut disparaitre sans bruit.

// ident construit un evenement identifie, pour alleger les cas de test.
func ident(timeMS int, xuid, stat string) objectiveevents.IdentifiedEvent {
	return objectiveevents.IdentifiedEvent{
		NamedEvent: objectiveevents.NamedEvent{TimeMS: timeMS, Stat: stat},
		XUID:       xuid,
	}
}

// TestBuildObjectiveActionsMapsOntoFrameAxis — la conversion vers l'index de frame est une
// simple division : meme horloge des deux cotes, donc ni appariement ni tolerance.
func TestBuildObjectiveActionsMapsOntoFrameAxis(t *testing.T) {
	evs := []objectiveevents.IdentifiedEvent{
		ident(0, "a", objectiveevents.StatFlagCaptures),
		ident(250, "b", objectiveevents.StatFlagReturns),
		ident(1_050, "a", objectiveevents.StatFlagGrabs),
	}
	got, cov := buildObjectiveActions(evs, 0, 0, scoreClock{intervalMS: 100, frames: 20})
	if len(got) != 3 || cov.Attached != 3 {
		t.Fatalf("%d actions (couverture %d), attendu 3", len(got), cov.Attached)
	}
	for i, want := range []int{0, 2, 10} {
		if got[i].T != want {
			t.Errorf("action %d : T = %d, attendu %d", i, got[i].T, want)
		}
	}
	// L'instant exact survit a la grille : deux actions d'une meme frame restent ordonnables.
	if got[1].TimeMS != 250 {
		t.Errorf("TimeMS = %d, attendu 250 (l'instant exact doit survivre a la grille)", got[1].TimeMS)
	}
}

// TestBuildObjectiveActionsCountsOutOfWindow — une action posterieure a la derniere frame
// publiee est COMPTEE hors fenetre, jamais rattachee a une frame qui n'existe pas.
//
// Le cas est reel : le film continue apres la derniere position rendue, donc les actions de
// fin de partie tombent au-dela de l'axe.
func TestBuildObjectiveActionsCountsOutOfWindow(t *testing.T) {
	evs := []objectiveevents.IdentifiedEvent{
		ident(500, "a", objectiveevents.StatZoneCaptures),
		ident(999_000, "a", objectiveevents.StatZoneSecures),
	}
	got, cov := buildObjectiveActions(evs, 0, 0, scoreClock{intervalMS: 100, frames: 10})
	if len(got) != 1 {
		t.Fatalf("%d actions publiees, attendu 1", len(got))
	}
	if cov.OutOfWindow != 1 {
		t.Errorf("horsFenetre = %d, attendu 1", cov.OutOfWindow)
	}
	if !cov.Balanced() {
		t.Errorf("couverture desequilibree : %+v", cov)
	}
}

// TestBuildObjectiveActionsRefusesUnidentified — un evenement sans xuid n'est PAS posable.
// Le rattacher a un slot arbitraire serait exactement l'erreur que le pont existe pour
// eviter (etat de l'art §20.1).
func TestBuildObjectiveActionsRefusesUnidentified(t *testing.T) {
	evs := []objectiveevents.IdentifiedEvent{
		ident(100, "", objectiveevents.StatFlagSteals),
		ident(200, "a", objectiveevents.StatFlagSteals),
	}
	got, cov := buildObjectiveActions(evs, 0, 0, scoreClock{intervalMS: 100, frames: 10})
	if len(got) != 1 || cov.NoSlot != 1 {
		t.Errorf("%d actions, sansSlot = %d ; attendu 1 et 1", len(got), cov.NoSlot)
	}
	if !cov.Balanced() {
		t.Errorf("couverture desequilibree : %+v", cov)
	}
}

// TestActionsSansTrajectoirePubliéeSontQuandMemePubliees — UNE ACTION EST UNE ACTION.
//
// Ce test RETOURNE `TestDropUnpublishedActionsKeepsTheInvariant` (baseline gelee, entree
// corrigee dans le meme commit, lot P2/R1 2026-09-08). L'ancien exigeait que l'action d'un
// joueur sans trajectoire publiee soit SUPPRIMEE et comptee sous `Unpublished` ; la doctrine
// du plan v2 dit l'inverse : une lecture vraie du film n'est jamais jetee parce qu'un autre
// calque est incomplet. Le compte reste, mais au journal — plus rien n'est rejete ici.
//
// MUTATION : reintroduire le filtre -> l'action « fantome » disparait, rouge.
func TestActionsSansTrajectoirePubliéeSontQuandMemePubliees(t *testing.T) {
	actions := []ObjectiveAction{
		{T: 1, XUID: "a", Stat: objectiveevents.StatFlagCaptures},
		{T: 2, XUID: "fantome", Stat: objectiveevents.StatFlagReturns},
	}
	if n := countActionsWithoutTrack(actions, []Track{{XUID: "a"}}, nil); n != 1 {
		t.Fatalf("actions sans trajectoire = %d, attendu 1", n)
	}
	doc := ReplayDocument{Tracks: []Track{{XUID: "a"}}}
	cov := attachObjectiveActions(&doc, Options{
		Objectives: []objectiveevents.IdentifiedEvent{
			ident(100, "a", objectiveevents.StatFlagCaptures),
			ident(200, "fantome", objectiveevents.StatFlagReturns),
		},
	}, IdentityRegistry{}, scoreClock{intervalMS: 100, frames: 10})
	if len(doc.Objectives) != 2 {
		t.Fatalf("%d actions publiees, attendu 2 — une lecture vraie a ete jetee", len(doc.Objectives))
	}
	if cov.Unpublished != 0 {
		t.Errorf("nonPublies = %d, attendu 0 : plus rien n'est rejete par ce calque", cov.Unpublished)
	}
	if cov.Attached != 2 || !cov.Balanced() {
		t.Errorf("couverture desequilibree : %+v", cov)
	}
}

// TestObjectiveActionJSONShape — le document est un contrat avec le client web : les cles
// sont verrouillees ici, un renommage silencieux casserait le rendu.
func TestObjectiveActionJSONShape(t *testing.T) {
	raw, err := json.Marshal(ObjectiveAction{
		T: 3, XUID: "2533274823110022", Stat: objectiveevents.StatFlagGrabs, TimeMS: 350,
	})
	if err != nil {
		t.Fatalf("marshal : %v", err)
	}
	want := `{"t":3,"xuid":"2533274823110022","stat":"flag_grabs","timeMs":350}`
	if string(raw) != want {
		t.Errorf("JSON = %s\nattendu %s", raw, want)
	}
}

// TestDocumentOmitsEmptyObjectives — un mode sans objectif ne doit pas alourdir le document
// d'un tableau vide (meme convention omitempty que les autres calques optionnels).
func TestDocumentOmitsEmptyObjectives(t *testing.T) {
	raw, err := json.Marshal(ReplayDocument{SchemaVersion: SchemaVersion})
	if err != nil {
		t.Fatalf("marshal : %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal : %v", err)
	}
	if _, present := m["objectives"]; present {
		t.Error("la cle objectives ne doit pas etre emise quand le calque est vide")
	}
}

// TestBuildObjectiveActionsSubtractsOrigin — LE CALQUE ETAIT DECALE DE `originMs`, ET IL NE
// L'EST PLUS (report `:123` du registre, corrige le 2026-08-18).
//
// Les evenements sont dates depuis le premier paquet du FILM, la grille de frames compte depuis
// le premier paquet de POSITION. Sans la soustraction, une action de 10 500 ms de film tombait a
// la frame 105 au lieu de 5 — soit 10 s trop tard, donc frequemment sur la mauvaise zone.
func TestBuildObjectiveActionsSubtractsOrigin(t *testing.T) {
	evs := []objectiveevents.IdentifiedEvent{
		ident(10_500, "a", objectiveevents.StatFlagCaptures),
		ident(20_000, "a", objectiveevents.StatFlagReturns),
	}
	clock := scoreClock{intervalMS: 100, frames: 200, originMS: 10_000}
	got, cov := buildObjectiveActions(evs, 0, 0, clock)
	if len(got) != 2 || cov.Attached != 2 {
		t.Fatalf("%d action(s) (couverture %d), attendu 2", len(got), cov.Attached)
	}
	for i, want := range []int{5, 100} {
		if got[i].T != want {
			t.Errorf("action %d : T = %d, attendu %d (origine non retranchee ?)", i, got[i].T, want)
		}
	}
	// L'instant EXACT reste sur l'horloge du film : lui recaler serait figer dans l'artefact
	// un decalage que le client sait deja appliquer a ses autres lignes de fil.
	if got[0].TimeMS != 10_500 {
		t.Errorf("TimeMS = %d, attendu 10500 (l'instant du film ne se recale pas)", got[0].TimeMS)
	}
}

// TestBuildObjectiveActionsRefusesBeforeFrameZero — une action ANTERIEURE a la frame 0 (mise en
// place du match) est comptee hors fenetre, jamais ecrasee sur la frame 0.
func TestBuildObjectiveActionsRefusesBeforeFrameZero(t *testing.T) {
	evs := []objectiveevents.IdentifiedEvent{
		ident(9_950, "a", objectiveevents.StatFlagGrabs),
		ident(10_100, "a", objectiveevents.StatFlagGrabs),
	}
	got, cov := buildObjectiveActions(evs, 0, 0, scoreClock{intervalMS: 100, frames: 200, originMS: 10_000})
	if len(got) != 1 || cov.OutOfWindow != 1 {
		t.Errorf("%d action(s), horsFenetre = %d ; attendu 1 et 1", len(got), cov.OutOfWindow)
	}
	if !cov.Balanced() {
		t.Errorf("couverture desequilibree : %+v", cov)
	}
}

// TestCouvertureCompteCeQueLePontNaPasNomme — LE DENOMINATEUR VOIT L'AMONT (constat P1-4 de
// l'audit du 2026-09-06).
//
// Les actions que le pont d'identite n'a pas su attribuer n'arrivent JAMAIS dans `evs` : les
// ignorer faisait de `Available` un compte de RESCAPES, et le rapport rattache/disponible se
// lisait ~100 % sur un calque partiel. `noSlot` — le seul champ du contrat public prevu pour
// dire « le pont ne couvre pas ce joueur » — valait alors 0 sur les 111 artefacts du parc,
// sans une seule exception.
//
// MUTATION : rendre `cov := LayerCoverage{Available: len(evs)}` (sans `unnamed`) rougit sur
// les deux assertions — `disponibles = 2, attendu 5` et `sansSlot = 0, attendu 3`.
func TestCouvertureCompteCeQueLePontNaPasNomme(t *testing.T) {
	evs := []objectiveevents.IdentifiedEvent{
		ident(100, "a", objectiveevents.StatFlagCaptures),
		ident(200, "b", objectiveevents.StatFlagGrabs),
	}
	got, cov := buildObjectiveActions(evs, 3, 0, scoreClock{intervalMS: 100, frames: 20})
	if len(got) != 2 {
		t.Fatalf("%d action(s) publiee(s), attendu 2", len(got))
	}
	if cov.Available != 5 {
		t.Errorf("disponibles = %d, attendu 5 (2 identifiees + 3 que le pont n'a pas nommees)",
			cov.Available)
	}
	if cov.NoSlot != 3 {
		t.Errorf("sansSlot = %d, attendu 3 : la perte amont doit se publier sous la categorie "+
			"que le contrat lui reserve", cov.NoSlot)
	}
	if !cov.Balanced() {
		t.Errorf("couverture desequilibree : %+v — la fuite serait EN AMONT du point d'equilibre", cov)
	}
}

// TestActionDunJoueurSansVieNommeeEstPubliee — UNE LECTURE VRAIE N'EST PAS JETEE PARCE QU'UN
// NOM MANQUE (constat P1-3 de l'audit du 2026-09-06).
//
// Un joueur dont AUCUNE vie n'est nommee perdait TOUTES ses actions d'objectif, alors que sa
// trajectoire EST publiee et que le pont canonique nomme son slot. Trois consommateurs perdaient
// la donnee, dont deux qui n'ont jamais eu besoin d'une trajectoire (le SON d'objectif et la
// garde tout-ou-rien de l'armement de bombe).
//
// LE DEFAUT EST DEMONTRE ICI, PAR MUTATION, ET NULLE PART AILLEURS : le chiffre de `3372e7eb`
// (35 actions sur 76) que la premiere redaction citait ne le mesure pas — ces actions viennent
// de deux joueurs SANS AUCUNE piste dans le film, que le pont ne peut pas atteindre (revue
// VIES-R1, C3). Le parc local ne porte aucun temoin de la configuration declenchante.
//
// MUTATION : revenir a l'index bati sur `tr.XUID != ""` rougit (« publiees = 0, attendu 1 »).
// TestActionDunJoueurSansVieNommeeEstPubliee — UNE LECTURE VRAIE N'EST PAS JETEE PARCE QU'UN
// NOM MANQUE (constat P1-3 de l'audit du 2026-09-06, elargi par le lot P2/R1 du 2026-09-08).
//
// Un joueur dont AUCUNE vie n'est nommee perdait TOUTES ses actions d'objectif, alors que sa
// trajectoire EST publiee et que le pont canonique nomme son slot. Depuis R1, plus AUCUNE action
// n'est jetee : ce test verifie que le COMPTE d'actions sans trajectoire est nul des que le pont
// nomme le slot de la piste — c'est lui qui declenche l'alarme du journal.
//
// MUTATION : revenir a l'index bati sur `tr.XUID != ""` rougit (« sansTrajectoire = 1, attendu 0 »).
func TestActionDunJoueurSansVieNommeeEstPubliee(t *testing.T) {
	actions := []ObjectiveAction{{T: 1, XUID: "42", Stat: objectiveevents.StatFlagCaptures}}
	tracks := []Track{{Slot: 536}} // la piste est PUBLIEE, son nommage a echoue
	if n := countActionsWithoutTrack(actions, tracks, map[uint32]uint64{536: 42}); n != 0 {
		t.Fatalf("sansTrajectoire = %d, attendu 0 : le pont nomme le slot 536", n)
	}
}

// TestActionSansPontEstComptee — LA CONTRE-EPREUVE : quand rien ne nomme le slot de la piste,
// l'action reste PUBLIEE (doctrine R1) mais elle est COMPTEE — le defaut du calque des positions
// doit rester visible au journal, et on n'invente aucun joueur pour le masquer.
func TestActionSansPontEstComptee(t *testing.T) {
	actions := []ObjectiveAction{{T: 1, XUID: "42", Stat: objectiveevents.StatFlagCaptures}}
	tracks := []Track{{Slot: 536}}
	for nom, pont := range map[string]map[uint32]uint64{
		"pont muet":           nil,
		"pont sur autre slot": {999: 42},
		"pont sur autre nom":  {536: 77},
		"pont a zero":         {536: 0},
	} {
		t.Run(nom, func(t *testing.T) {
			if n := countActionsWithoutTrack(actions, tracks, pont); n != 1 {
				t.Errorf("sansTrajectoire = %d, attendu 1 : rien ne nomme ce slot", n)
			}
		})
	}
}
