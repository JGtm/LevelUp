package replay

import (
	"encoding/json"
	"reflect"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// coverage_test.go — L'INVARIANT « rien ne se jette en silence ».
//
// Ces tests ne vérifient pas des valeurs : ils verrouillent une PROPRIÉTÉ. Tout événement
// disponible est soit rattaché, soit rejeté sous une cause nommée, et la somme fait le
// total exactement. C'est ce qui empêche qu'un futur `continue` réintroduise le défaut
// qu'on vient de supprimer — un trou de 34 secondes découvert en regardant l'écran.

func TestCoverageBalancedOnEmptyInputs(t *testing.T) {
	// Cas dégénérés : aucune position, aucun pont. Le compte doit rester juste — c'est là
	// que les fuites passent inaperçues, parce qu'on regarde rarement le cas vide.
	events := []filmdec.FireEvent{fireAt(1_000_000, 3, 90), fireAt(1_100_000, 3, 90)}
	for _, tc := range []struct {
		name  string
		pos   []filmdec.BipedPosition
		owner map[uint32]int
	}{
		{"sans position", nil, map[uint32]int{10: 3}},
		{"sans pont", []filmdec.BipedPosition{posAt(10, 1_000_000, 1, 1, 90)}, nil},
		{"sans rien", nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			shots, cov := buildShots(tc.pos, events, 1_000_000, 100_000, tc.owner)
			if len(shots) != 0 {
				t.Errorf("aucun tir ne devrait etre publie, obtenu %d", len(shots))
			}
			if cov.Available != len(events) {
				t.Errorf("denominateur perdu : %+v", cov)
			}
			if !cov.Balanced() {
				t.Errorf("fuite : %+v", cov)
			}
		})
	}
}

func TestCoverageBalancedWithNoEvents(t *testing.T) {
	// Zéro disponible : la somme doit valoir zéro, pas produire un compteur fantôme.
	_, cov := buildShots(nil, nil, 0, 100_000, nil)
	if cov.Available != 0 || !cov.Balanced() {
		t.Errorf("couverture incoherente sur entree vide : %+v", cov)
	}
	_, gcov := buildGrenades(nil, nil, 0, 100_000, nil, nil)
	if gcov.Available != 0 || !gcov.Balanced() {
		t.Errorf("couverture incoherente sur entree vide (grenades) : %+v", gcov)
	}
}

func TestCoverageCountsOutOfWindow(t *testing.T) {
	// Le slot est connu, mais sa position la plus proche est hors de la fenêtre de
	// rattachement. La cause doit être « hors fenêtre » et NON « slot introuvable » : la
	// première désigne le décodage des positions, la seconde le pont. Les confondre
	// enverrait le prochain chantier au mauvais endroit.
	pos := []filmdec.BipedPosition{posAt(10, 1_000_000, 1, 1, 90)}
	far := 1_000_000 + uint64(shotPosToleranceUS) + 500_000
	events := []filmdec.FireEvent{fireAt(far, 3, 90)}
	shots, cov := buildShots(pos, events, 1_000_000, 100_000, map[uint32]int{10: 3})
	if len(shots) != 0 {
		t.Fatalf("un tir hors fenetre ne doit pas etre publie : %+v", shots)
	}
	if cov.NoSlot != 1 {
		t.Errorf("attendu 1 rejet ; couverture %+v", cov)
	}
	if !cov.Balanced() {
		t.Errorf("fuite : %+v", cov)
	}
}

func TestDocumentPublishesCoverage(t *testing.T) {
	// La couverture doit atteindre le DOCUMENT, pas seulement les journaux : c'est la
	// différence entre un décodeur qui sait ce qu'il perd et un écran qui le montre.
	var pos []filmdec.BipedPosition
	for i := 0; i < 40; i++ {
		ts := 1_000_000 + uint64(i)*50_000
		pos = append(pos, posAt(10, ts, 1, 1, 90))
		pos = append(pos, posAt(11, ts, 5, 5, 270))
	}
	events := []filmdec.FireEvent{fireAt(1_200_000, 3, 90), fireAt(1_300_000, 3, 90)}
	doc := BuildFromPositions("m", "halo_infinite", pos, events, Options{})
	if doc.Coverage == nil {
		t.Fatal("le document doit porter sa couverture")
	}
	if doc.Coverage.Shots.Available != len(events) {
		t.Errorf("denominateur des tirs absent du document : %+v", doc.Coverage.Shots)
	}
	if !doc.Coverage.Shots.Balanced() {
		t.Errorf("fuite dans la couverture publiee : %+v", doc.Coverage.Shots)
	}
	if got := len(doc.Shots); got != doc.Coverage.Shots.Attached {
		t.Errorf("la couverture annonce %d tirs, le document en porte %d",
			doc.Coverage.Shots.Attached, got)
	}
}

func TestBridgeHealthJSONKeysAreDistinct(t *testing.T) {
	// PIÈGE DU LANGAGE : plusieurs champs déclarés sur une même ligne partagent leur tag,
	// ce qui écrase des clés JSON en silence. Le compilateur ne dit rien. Ce test compte
	// les clés effectivement sérialisées : elles doivent être aussi nombreuses que les
	// champs de la structure.
	raw, err := json.Marshal(BridgeHealth{Slots: 1, FromReading: 2,
		LivesNamed: 4, LivesTotal: 5, IndexReadings: 26, IndexDisagreements: 0, SlotCollisions: 10})
	if err != nil {
		t.Fatalf("serialisation : %v", err)
	}
	var m map[string]int
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("relecture : %v", err)
	}
	if got, want := len(m), reflect.TypeOf(BridgeHealth{}).NumField(); got != want {
		t.Errorf("%d cles JSON pour %d champs — des tags se recouvrent : %s", got, want, raw)
	}
	// Les valeurs doivent aussi survivre au tour complet : une cle ecrasee garderait la
	// derniere valeur ecrite, ce que le seul compte de cles ne verrait pas toujours.
	if m["slots"] != 1 || m["fromReading"] != 2 || m["indexReadings"] != 26 || m["slotCollisions"] != 10 {
		t.Errorf("valeurs alterees par la serialisation : %s", raw)
	}
}

func TestVerdictRefusesRatherThanWarns(t *testing.T) {
	// La porte doit REFUSER, pas nuancer. Un calque dont le comptage fuit n'est pas
	// « partiel » : il est non publiable, parce qu'on ne sait pas ce qu'il a perdu.
	leak := LayerCoverage{Available: 10, Attached: 3} // 3 + 0 + 0 + 0 + 0 != 10
	if v := verdictOf(leak); v == "nominal" {
		t.Errorf("une fuite ne peut pas etre nominale : %q", v)
	}
	full := LayerCoverage{Available: 10, Attached: 10}
	if v := verdictOf(full); v != "nominal" {
		t.Errorf("un calque complet doit etre nominal, obtenu %q", v)
	}
	thin := LayerCoverage{Available: 10, Attached: 3, NoSlot: 7}
	if v := verdictOf(thin); v == "nominal" {
		t.Errorf("30 %% de rattachement ne peut pas etre nominal : %q", v)
	}
	// Un slot qui change de porteur invalide la TABLE elle-meme, quelle que soit la
	// couverture des calques : le verdict du pont doit le dire.
	if v := verdictOfBridge(BridgeHealth{Slots: 8, FromReading: 8, IndexReadings: 26, SlotCollisions: 1}); v == VerdictNominal {
		t.Errorf("une collision de slot ne peut pas etre nominale : %q", v)
	}
	// UNE SOURCE AUTRE QUE LA LECTURE ne doit plus pouvoir alimenter le pont. Depuis le retrait
	// du repli vote, FromReading DOIT valoir Slots ; un ecart signale qu'une seconde source est
	// reapparue, et le verdict doit refuser plutot que nuancer.
	if v := verdictOfBridge(BridgeHealth{Slots: 8, FromReading: 6, IndexReadings: 26}); v == VerdictNominal {
		t.Errorf("un pont alimente ailleurs que par la lecture ne peut pas etre nominal : %q", v)
	}
	if v := verdictOfBridge(BridgeHealth{Slots: 8, FromReading: 8, IndexReadings: 26}); v != VerdictNominal {
		t.Errorf("un pont entierement lu et tranche doit etre nominal, obtenu %q", v)
	}
}

func TestGrenadePlacedFromProjectileWithoutBridge(t *testing.T) {
	// LE LANCER N'A PAS BESOIN DU PONT. Sa position est celle du projectile qu'il fait naitre,
	// et le film ecrit deja son auteur. Ce test verrouille la correction du 2026-07-28 : avant
	// elle, sept lancers sur soixante-dix etaient perdus parce que la vie du lanceur n'etait
	// pas nommee — alors que leur position etait decodee.
	// Le TypeID est renseigne : depuis le lot 3.1 un lancer porte son RANG, et un tag
	// hors liste blanche n'est pas publie (il n'aurait aucune table pour le nommer).
	throws := []filmdec.GrenadeThrow{
		{TimestampUS: 2_000_000, FilmIndex: 5, TypeID: filmdec.GrenadeFragmentation},
	}
	proj := []filmdec.ProjectileTrack{{Slot: 1024, Gen: 1, Pts: []filmdec.ProjectileSample{
		{TimestampUS: 2_050_000, X: 12, Y: 34, Z: 5},
	}}}
	// AUCUNE position de biped, AUCUN pont : le lancer doit quand meme etre situe.
	gren, cov := buildGrenades(nil, throws, 1_000_000, 100_000, nil, proj)
	if len(gren) != 1 {
		t.Fatalf("le lancer doit etre situe sans pont, obtenu %d : %+v", len(gren), gren)
	}
	if gren[0].Src != GrenadeSrcProjectile {
		t.Errorf("la position doit venir du projectile, obtenu %q", gren[0].Src)
	}
	if gren[0].X != 12 || gren[0].Y != 34 {
		t.Errorf("position attendue (12,34), obtenue (%v,%v)", gren[0].X, gren[0].Y)
	}
	if gren[0].Idx != 5 {
		t.Errorf("l'auteur est ECRIT dans le film : index 5 attendu, obtenu %d", gren[0].Idx)
	}
	if !cov.Balanced() || cov.Attached != 1 {
		t.Errorf("couverture incoherente : %+v", cov)
	}
	// Et il ne doit pas etre filtre par l'absence de trajectoire publiee.
	if kept := keepGrenadesOfPublishedTracks(gren, nil); len(kept) != 1 {
		t.Errorf("un lancer situe par son projectile ne depend d'aucune trajectoire : %+v", kept)
	}
}
