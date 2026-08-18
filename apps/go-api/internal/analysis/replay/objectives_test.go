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
	got, cov := buildObjectiveActions(evs, scoreClock{intervalMS: 100, frames: 20})
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
	got, cov := buildObjectiveActions(evs, scoreClock{intervalMS: 100, frames: 10})
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
	got, cov := buildObjectiveActions(evs, scoreClock{intervalMS: 100, frames: 10})
	if len(got) != 1 || cov.NoSlot != 1 {
		t.Errorf("%d actions, sansSlot = %d ; attendu 1 et 1", len(got), cov.NoSlot)
	}
	if !cov.Balanced() {
		t.Errorf("couverture desequilibree : %+v", cov)
	}
}

// TestDropUnpublishedActionsKeepsTheInvariant — une action dont le joueur n'a aucune track
// publiee est comptee a part, pas silencieusement perdue.
//
// C'est la meme regle que pour les tirs : ce n'est pas un echec de rattachement, mais le
// client n'aurait pas de trajectoire ou l'accrocher. L'annoncer comme rattachee promettrait
// une couverture que l'ecran ne tiendrait pas.
func TestDropUnpublishedActionsKeepsTheInvariant(t *testing.T) {
	actions := []ObjectiveAction{
		{T: 1, XUID: "a", Stat: objectiveevents.StatFlagCaptures},
		{T: 2, XUID: "fantome", Stat: objectiveevents.StatFlagReturns},
	}
	cov := LayerCoverage{Available: 2, Attached: 2}
	got, cov := dropUnpublishedActions(actions, []Track{{XUID: "a"}}, cov)
	if len(got) != 1 || got[0].XUID != "a" {
		t.Fatalf("%d actions gardees, attendu 1 (celle du joueur publie)", len(got))
	}
	if cov.Unpublished != 1 || cov.Attached != 1 {
		t.Errorf("nonPublies = %d, rattaches = %d ; attendu 1 et 1", cov.Unpublished, cov.Attached)
	}
	if !cov.Balanced() {
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
	got, cov := buildObjectiveActions(evs, clock)
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
	got, cov := buildObjectiveActions(evs, scoreClock{intervalMS: 100, frames: 200, originMS: 10_000})
	if len(got) != 1 || cov.OutOfWindow != 1 {
		t.Errorf("%d action(s), horsFenetre = %d ; attendu 1 et 1", len(got), cov.OutOfWindow)
	}
	if !cov.Balanced() {
		t.Errorf("couverture desequilibree : %+v", cov)
	}
}
