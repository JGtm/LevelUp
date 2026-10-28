package replay

// document_effets_test.go — L OBJET D EFFET de l ecran occultant (plan des durees d effets, B.2) : l association des
// candidates que la grammaire rend, par la famille, le repos de la grenade et le nombre de candidates.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

const origineEcran = uint64(10_000_000)

// ecranDeTest : un ecran occultant (rang 10) de l objet 2039, porteur 539 : la grenade 1123 est nommee a 118 775 ms et
// supprimee a 119 100 ms ; `cands` sont les candidates lues a cette suppression, `fins` les suppressions d effet.
func ecranDeTest(rang int, repos []types.ProjectileTrack, cands []types.SpawnedObject,
	fins ...types.CarriedEquipmentRead) entreeDEquipementPorte {
	grenade := types.LifeKey{Slot: 1123, Gen: 1}
	objet := types.LifeKey{Slot: 2039, Gen: 1}
	reads := []types.CarriedEquipmentRead{
		lecture(90183.0, 2039, 539, types.CarriedNew),
		avecActivation(avecSuivis(lecture(118775.4, 2039, 539, types.CarriedDelta), 1123)),
		{TimestampUS: 119_100_000, Object: objet, Carrier: 539, Kind: types.CarriedTrackedEnd, Tracked: []types.LifeKey{grenade}},
	}
	for _, c := range cands {
		reads = append(reads, types.CarriedEquipmentRead{TimestampUS: 119_100_000, Object: objet, Carrier: 539,
			Kind: grammar.CarriedEffectBirth, Tracked: []types.LifeKey{grenade}, Spawned: []types.SpawnedObject{c}})
	}
	reads = append(reads, fins...)
	reads = append(reads, lecture(130000.0, 2039, 539, types.CarriedDel))
	return entreeDEquipementPorte{reads: reads,
		palette:     &AbilityPalette{Families: map[int]string{10: "shroud_screen", 23: "repair_field"}},
		changes:     []types.EquipmentChange{{TimestampUS: 90183000, Slot: 539, Kind: types.EquipmentTaken, Rank: rang}},
		projectiles: repos, tracks: []Track{{Slot: 539}}, origin: origineEcran, step: pas100ms}
}

// reposDeLaGrenade : la piste de la grenade 1123, nee a la frame de son nommage, qui se pose en (10, 20, 5).
func reposDeLaGrenade() []types.ProjectileTrack {
	return []types.ProjectileTrack{{Slot: 1123, Gen: 1, Pts: []types.ProjectileSample{
		echantillon(118_700_000, 9, 19, 6, false), echantillon(118_800_000, 9.5, 19.5, 5.5, false),
		echantillon(118_900_000, 10, 20, 5, true)}}}
}

func candidate(slot uint32, ts uint64, x, y, z float32) types.SpawnedObject {
	return types.SpawnedObject{Life: types.LifeKey{Slot: slot, Gen: 1}, TimestampUS: ts, TypeIndex: 38,
		HasPosition: true, X: x, Y: y, Z: z}
}

func finDEffetLue(slot uint32, ts uint64) types.CarriedEquipmentRead {
	return types.CarriedEquipmentRead{TimestampUS: ts, Object: types.LifeKey{Slot: 2039, Gen: 1}, Carrier: 539,
		Kind: grammar.CarriedEffectEnd, Tracked: []types.LifeKey{{Slot: slot, Gen: 1}}}
}

// TestEffetDeLEcranLu : la candidate a 0,3 m du repos de la grenade devient l objet d effet : sa frame de
// naissance, sa suppression lue, sa position arrondie.
func TestEffetDeLEcranLu(t *testing.T) {
	out, _, cov := buildCarriedEquipment(ecranDeTest(10, reposDeLaGrenade(),
		[]types.SpawnedObject{candidate(1478, 118_850_000, 10.2, 20.2, 5.1)}, finDEffetLue(1478, 126_357_000)))
	e := out[0].Tracked[0].Effect
	if e == nil || e.T0 != 1088 || e.T1 == nil || *e.T1 != 1163 || e.X == nil || *e.X != 10.2 || *e.Y != 20.2 || *e.Z != 5.1 {
		t.Fatalf("effet %+v", e)
	}
	if cov.TrackedEffects != 1 || cov.TrackedEffectEnds != 1 || cov.TrackedEffectsAmbiguous != 0 || cov.TrackedEffectsUnlinked != 0 {
		t.Fatalf("couverture %+v", cov)
	}
}

// TestEffetRivaleEcartee : une candidate lointaine (la rivale mesuree est a 9,5 m) ne gene pas la bonne.
func TestEffetRivaleEcartee(t *testing.T) {
	out, _, cov := buildCarriedEquipment(ecranDeTest(10, reposDeLaGrenade(), []types.SpawnedObject{
		candidate(1500, 118_700_000, 19.5, 20, 5), candidate(1478, 118_850_000, 10.2, 20.2, 5.1)}))
	e := out[0].Tracked[0].Effect
	if e == nil || *e.X != 10.2 || e.T1 != nil || cov.TrackedEffects != 1 || cov.TrackedEffectEnds != 0 {
		t.Fatalf("effet %+v, couverture %+v", e, cov)
	}
}

// TestEffetBordDuRayon : a 3 m du repos la candidate est retenue, un peu plus loin elle ne l est pas.
func TestEffetBordDuRayon(t *testing.T) {
	for _, k := range []struct {
		dx      float32
		retenue bool
	}{{3, true}, {3.01, false}} {
		out, _, cov := buildCarriedEquipment(ecranDeTest(10, reposDeLaGrenade(),
			[]types.SpawnedObject{candidate(1478, 118_850_000, 10+k.dx, 20, 5)}))
		if got := out[0].Tracked[0].Effect != nil; got != k.retenue {
			t.Fatalf("dx %.2f : retenue %v, attendu %v (%+v)", k.dx, got, k.retenue, cov)
		}
		if !k.retenue && cov.TrackedEffectsUnlinked != 1 {
			t.Fatalf("couverture %+v : sans lien compte", cov)
		}
	}
}

// TestEffetAmbigu : deux candidates a portee du repos : rien n est publie, l ambiguite se compte.
func TestEffetAmbigu(t *testing.T) {
	out, _, cov := buildCarriedEquipment(ecranDeTest(10, reposDeLaGrenade(), []types.SpawnedObject{
		candidate(1478, 118_850_000, 10.2, 20.2, 5.1), candidate(1479, 118_870_000, 10.4, 20, 5)}))
	if out[0].Tracked[0].Effect != nil || cov.TrackedEffects != 0 || cov.TrackedEffectsAmbiguous != 1 {
		t.Fatalf("effet %+v, couverture %+v", out[0].Tracked[0].Effect, cov)
	}
}

// TestEffetSansRepos : sans repos publie (aucune trajectoire), la candidate unique de la fenetre est retenue avec sa
// position ; deux candidates sont ambigues. Aucune position ne vient d ailleurs que de la candidate.
func TestEffetSansRepos(t *testing.T) {
	out, _, cov := buildCarriedEquipment(ecranDeTest(10, nil,
		[]types.SpawnedObject{candidate(1478, 118_850_000, 40, 50, 6)}))
	e := out[0].Tracked[0].Effect
	if out[0].Tracked[0].Land != nil || e == nil || *e.X != 40 || cov.TrackedEffects != 1 {
		t.Fatalf("effet %+v, couverture %+v", e, cov)
	}
	out, _, cov = buildCarriedEquipment(ecranDeTest(10, nil, []types.SpawnedObject{
		candidate(1478, 118_850_000, 40, 50, 6), candidate(1479, 118_850_000, 60, 50, 6)}))
	if out[0].Tracked[0].Effect != nil || cov.TrackedEffectsAmbiguous != 1 {
		t.Fatalf("effet %+v, couverture %+v", out[0].Tracked[0].Effect, cov)
	}
}

// TestEffetSansPositionDeCandidate : une candidate sans position est retenue quand le repos manque (naissance lue,
// aucune coordonnee), et ecartee quand le repos est publie (rien ne prouve qu elle est a portee).
func TestEffetSansPositionDeCandidate(t *testing.T) {
	sans := candidate(1478, 118_850_000, 0, 0, 0)
	sans.HasPosition = false
	out, _, _ := buildCarriedEquipment(ecranDeTest(10, nil, []types.SpawnedObject{sans}))
	if e := out[0].Tracked[0].Effect; e == nil || e.X != nil || e.Y != nil || e.Z != nil {
		t.Fatalf("effet %+v : les trois coordonnees sont absentes ensemble", e)
	}
	out, _, cov := buildCarriedEquipment(ecranDeTest(10, reposDeLaGrenade(), []types.SpawnedObject{sans}))
	if out[0].Tracked[0].Effect != nil || cov.TrackedEffectsUnlinked != 1 {
		t.Fatalf("effet %+v, couverture %+v", out[0].Tracked[0].Effect, cov)
	}
}

// TestEffetAutreFamille : seule la famille de l ecran occultant lit un effet ; un champ de reparation dont le film
// rend les memes lectures n en publie pas.
func TestEffetAutreFamille(t *testing.T) {
	out, _, cov := buildCarriedEquipment(ecranDeTest(23, reposDeLaGrenade(),
		[]types.SpawnedObject{candidate(1478, 118_850_000, 10.2, 20.2, 5.1)}, finDEffetLue(1478, 126_357_000)))
	if out[0].Tracked[0].Effect != nil || cov.TrackedEffects != 0 || cov.TrackedEffectsUnlinked != 0 {
		t.Fatalf("effet %+v, couverture %+v", out[0].Tracked[0].Effect, cov)
	}
}

// TestEffetFinAvantLaNaissance : une suppression lue AVANT la naissance de la candidate est celle d une vie anterieure
// de la meme cle ; elle ne date pas la fin de l effet.
func TestEffetFinAvantLaNaissance(t *testing.T) {
	out, _, cov := buildCarriedEquipment(ecranDeTest(10, reposDeLaGrenade(),
		[]types.SpawnedObject{candidate(1478, 118_850_000, 10.2, 20.2, 5.1)}, finDEffetLue(1478, 118_000_000)))
	if e := out[0].Tracked[0].Effect; e == nil || e.T1 != nil || cov.TrackedEffectEnds != 0 {
		t.Fatalf("effet %+v, couverture %+v", e, cov)
	}
}

// TestEffetSansCandidate : un ecran dont la grenade n a aucune candidate (fin non lue, ou aucune ti=38) ne publie
// pas d effet et se compte sans lien.
func TestEffetSansCandidate(t *testing.T) {
	out, _, cov := buildCarriedEquipment(ecranDeTest(10, reposDeLaGrenade(), nil))
	if out[0].Tracked[0].Effect != nil || cov.TrackedEffectsUnlinked != 1 || cov.TrackedObjects != 1 {
		t.Fatalf("effet %+v, couverture %+v", out[0].Tracked[0].Effect, cov)
	}
}

// TestEffetFinRetrouveeSousUnAutreObjet : la grammaire rend la suppression d une candidate une fois, sous le PREMIER objet
// porte qui l a retenue ; un second objet suivi qui la retient aussi (ici l ecran, la lecture est rangee sous un autre objet,
// absent du document) retrouve sa fin par la vie de l effet.
func TestEffetFinRetrouveeSousUnAutreObjet(t *testing.T) {
	fin := finDEffetLue(1478, 126_357_000)
	fin.Object = types.LifeKey{Slot: 2050, Gen: 1}
	fin.Carrier = 540
	out, _, cov := buildCarriedEquipment(ecranDeTest(10, reposDeLaGrenade(),
		[]types.SpawnedObject{candidate(1478, 118_850_000, 10.2, 20.2, 5.1)}, fin))
	e := out[0].Tracked[0].Effect
	if e == nil || e.T1 == nil || *e.T1 != 1163 || cov.TrackedEffectEnds != 1 {
		t.Fatalf("effet %+v, couverture %+v", e, cov)
	}
}

// deuxLancers : l objet 2039 lance deux fois la MEME cle de vie 1123/1 (la generation revient) : la grenade du premier est
// supprimee a 119 100 ms, celle du second nommee a 125 000 ms et supprimee a 126 100 ms. Chaque suppression rend ses
// candidates.
func deuxLancers(premiere, seconde types.SpawnedObject) entreeDEquipementPorte {
	in := ecranDeTest(10, reposDeLaGrenade(), []types.SpawnedObject{premiere})
	grenade := types.LifeKey{Slot: 1123, Gen: 1}
	objet := types.LifeKey{Slot: 2039, Gen: 1}
	suite := []types.CarriedEquipmentRead{
		avecSuivis(lecture(125000.0, 2039, 539, types.CarriedDelta), 1123),
		{TimestampUS: 126_100_000, Object: objet, Carrier: 539, Kind: types.CarriedTrackedEnd, Tracked: []types.LifeKey{grenade}},
		{TimestampUS: 126_100_000, Object: objet, Carrier: 539, Kind: grammar.CarriedEffectBirth,
			Tracked: []types.LifeKey{grenade}, Spawned: []types.SpawnedObject{seconde}},
	}
	n := len(in.reads) - 1 // avant la suppression de l objet porte
	in.reads = append(in.reads[:n:n], append(suite, in.reads[n])...)
	return in
}

// TestEffetCleRecycleeDansLeMemeObjet : la candidate du SECOND lancer, rendue sous la meme cle de vie suivie, n est pas
// attribuee au premier (meme garde que celle du repos) : seule la candidate lue a la suppression de CETTE vie compte.
func TestEffetCleRecycleeDansLeMemeObjet(t *testing.T) {
	// La candidate du premier lancer est loin, celle du second est a portee du repos du premier : sans la garde la bulle
	// du second serait attribuee au premier.
	out, _, cov := buildCarriedEquipment(deuxLancers(
		candidate(1478, 118_850_000, 40, 50, 6), candidate(1479, 125_850_000, 10.5, 20.5, 5)))
	if e := out[0].Tracked[0].Effect; e != nil || cov.TrackedEffectsUnlinked != 1 || cov.TrackedEffects != 0 {
		t.Fatalf("la bulle du second lancer attribuee au premier : %+v, couverture %+v", e, cov)
	}
	// Deux candidates a portee, une par lancer : le premier lancer garde la sienne, il n est pas ambigu.
	out, _, cov = buildCarriedEquipment(deuxLancers(
		candidate(1478, 118_850_000, 10.2, 20.2, 5.1), candidate(1479, 125_850_000, 10.5, 20.5, 5)))
	e := out[0].Tracked[0].Effect
	if e == nil || e.T0 != 1088 || *e.X != 10.2 || cov.TrackedEffectsAmbiguous != 0 || cov.TrackedEffects != 1 {
		t.Fatalf("effet %+v, couverture %+v", e, cov)
	}
}
