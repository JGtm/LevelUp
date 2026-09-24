package service

import (
	"context"
	"encoding/json"
	"os"
	"runtime"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/testutil"
)

// replay_vehicle_scenery_test.go — LE DECOR DE CARTE SUR LES ZONES JOUABLES REELLES (lot M7 des
// retours du rejeu, 2026-09-24). Les fonds et calages sont ceux du DEPOT (references versionnees,
// servies en production) ; les vies et le sol foule sont recopies des documents reels reconstruits
// depuis les faits au schema 69 (tete de la vague C). Les noms de carte sont ceux du registre ; la
// cle de production d une carte Forge (son map_id) est prouvee par [TestGetReplay_DecorDeStarboardParLeMapID].

// cleFondStarboard : la cle de fond de Starboard au depot, qui est sa cle de PRODUCTION (le map_id
// de la carte Forge).
const cleFondStarboard = "7a9265af-a880-487b-8829-68d88fcfb145"

// sceneryService rend un service sur la racine du depot, dont la carte du match porte `noms`.
func sceneryService(t *testing.T, noms ...string) *replayService {
	t.Helper()
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	return NewReplayService(title.DefaultSlug, root, &mapNamesStub{names: noms}).(*replayService)
}

// viePosee rend une vie qui remplit les cinq conditions de pose.
func viePosee(slot uint32, family string, x, y, z float32) replay.VehicleTrack {
	return replay.VehicleTrack{
		Slot: slot, Gen: 1, Family: family, T0: 0, T1: 5000, T1Max: 5000, End: replay.VehicleEndFilmEnd,
		Samples: []replay.VehicleSample{{T: 0, X: x, Y: y, Z: z}},
	}
}

// docJoue rend un document dont le sol FOULE est `sol` : une piste qui s y tient 1 s (10 images de
// 100 ms, la grille des documents du parc).
func docJoue(sol float32, vies ...replay.VehicleTrack) *replay.ReplayDocument {
	return &replay.ReplayDocument{
		FrameIntervalMS: 100,
		Bounds:          replay.Bounds{MinZ: sol},
		Tracks:          []replay.Track{{Points: []replay.Point{{T: 0, Z: sol}, {T: 10, Z: sol}}}},
		Vehicles:        vies,
	}
}

func resoudreDecor(t *testing.T, doc *replay.ReplayDocument, noms ...string) *replay.VehicleScenery {
	t.Helper()
	s := sceneryService(t, noms...)
	keys := s.matchMapKeys(context.Background(), "m7")
	s.resolveVehicleScenery(context.Background(), doc, "m7", keys)
	return doc.VehicleScenery
}

func raisons(v *replay.VehicleScenery) map[uint32]string {
	out := map[uint32]string{}
	for _, h := range v.Hidden {
		out[h.Slot] = h.Reason
	}
	return out
}

// decorsStarboard : les six vies de ab526724 et f0220a96, identiques au centimetre.
func decorsStarboard() []replay.VehicleTrack {
	return []replay.VehicleTrack{
		viePosee(771, "scorpion", 8.96, -132.79, 81.7),
		viePosee(772, "wasp", -1.82, -130.09, 80.8),
		viePosee(773, "wasp", -1.82, -133.38, 80.8),
		viePosee(774, "warthog", 6.42, -129.74, 81.32),
		viePosee(776, "warthog", 1.58, -133.84, 81.33),
		viePosee(778, "warthog", 1.67, -132.04, 81.3),
	}
}

// Starboard (ab526724 et f0220a96) : hors de la matiere praticable (hors du cadre du fond).
func TestVehicleScenery_StarboardReel(t *testing.T) {
	for _, sol := range []float32{80.61, 80.63} { // sols foules de ab526724, f0220a96
		v := resoudreDecor(t, docJoue(sol, decorsStarboard()...), "Starboard")
		if v == nil || v.Zone != sceneryZoneMap || v.Candidates != 6 || len(v.Hidden) != 6 {
			t.Fatalf("Starboard : 6 decors masques attendus, rendu %+v", v)
		}
		for slot, r := range raisons(v) {
			if r != sceneryReasonOffPlayArea {
				t.Errorf("slot %d : raison %q", slot, r)
			}
		}
	}
}

// Goliath (d8b13ec2 768) : DANS la matiere en plan, 3,03 m sous le sol foule.
func TestVehicleScenery_GoliathReel(t *testing.T) {
	v := resoudreDecor(t, docJoue(64.47, viePosee(768, "wasp", -0.44, -8.16, 61.44)), "Goliath")
	if v == nil || v.Zone != sceneryZoneMap || raisons(v)[768] != sceneryReasonBelowPlayedFloor {
		t.Fatalf("Goliath : le Wasp sous le sol doit etre masque, rendu %+v", v)
	}
}

// Behemoth hors Super Fiesta (7b0d89c4 773 et 774, f2966f08 769 et 770 : Mongoose poses au depart
// aux quatre coins des bases, RAMENES A UNE POSE SEULE) : dans l aire de jeu, jamais masques. La
// decision utilisateur nomme les Mongoose ET les Gungoose : le Gungoose est la variante nommee du
// Mongoose (M4a.2, meme famille) ; il est pose ici sur le socle de 7b0d89c4 768. Le Mongoose de
// f2966f08 770 est a 2,4 m du bord des positions jouees du match — le masque, lui, n en depend pas.
func TestVehicleScenery_BehemothPoseDansLAireDeJeuAffiche(t *testing.T) {
	gungoose := viePosee(768, "mongoose", -146.02, 27.43, 8.59)
	gungoose.Variant = "gungoose"
	v := resoudreDecor(t, docJoue(3.86, // sol foule de 7b0d89c4 et f2966f08
		viePosee(773, "mongoose", -101.61, 27.63, 8.8),
		viePosee(774, "mongoose", -101.65, 80.2, 8.31),
		gungoose,
		viePosee(769, "mongoose", -146.11, 80.31, 8.33),
	), "Behemoth")
	if v == nil || v.Zone != sceneryZoneMap || v.Candidates != 4 || v.InPlayArea != 4 || len(v.Hidden) != 0 {
		t.Fatalf("Behemoth : 4 vehicules poses dans l aire de jeu, affiches ; rendu %+v", v)
	}
}

// Launch Site (fccc61cd 772) : un Gungoose REEL, ne a l origine, ramene a une pose seule — dans
// l aire de jeu d une carte native, affiche.
func TestVehicleScenery_GungooseReelDansLAireDeJeuAffiche(t *testing.T) {
	g := viePosee(772, "mongoose", -14.22, -35.2, -0.54)
	g.Variant = "gungoose"
	v := resoudreDecor(t, docJoue(-3.92, g), "Launch Site") // sol foule des 3 matchs de Launch Site
	if v == nil || v.Zone != sceneryZoneMap || v.InPlayArea != 1 || len(v.Hidden) != 0 {
		t.Fatalf("Gungoose pose dans l aire de jeu : affiche, rendu %+v", v)
	}
}

// Carte sans fond publie : repli nomme, rien n est masque, tout est compte.
func TestVehicleScenery_CarteSansZoneReel(t *testing.T) {
	v := resoudreDecor(t, docJoue(80.61, viePosee(771, "scorpion", 8.96, -132.79, 81.7)),
		"Carte inexistante du test M7")
	if v == nil || v.Zone != sceneryZoneUnknown || v.ZoneUnknown != 1 || len(v.Hidden) != 0 {
		t.Fatalf("carte sans zone : rien masque, rendu %+v", v)
	}
}

// Aucune candidate : aucun chargement de masque, aucun verdict.
func TestVehicleScenery_SansCandidateAucunVerdict(t *testing.T) {
	simule := viePosee(771, "warthog", 1, 1, 1)
	simule.Samples = append(simule.Samples, replay.VehicleSample{T: 5, X: 1, Y: 1, Z: 1})
	doc := docJoue(0, simule)
	s := sceneryService(t, "Starboard")
	s.resolveVehicleScenery(context.Background(), doc, "m7", port.MatchMapKeys{Names: []string{"Starboard"}})
	if doc.VehicleScenery != nil {
		t.Fatalf("aucune candidate : verdict absent, rendu %+v", doc.VehicleScenery)
	}
}

// copieFond recopie un fond REEL du depot (sidecar + image) sous une racine de test.
func copieFond(t *testing.T, root, cle string) {
	t.Helper()
	depot, err := testutil.RepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	src, dst := title.NewPathResolver(depot), title.NewPathResolver(root)
	meta, err := replay.LoadMapBackground(src.MapBackgroundMetaPath(title.DefaultSlug, cle))
	if err != nil {
		t.Fatal(err)
	}
	for from, to := range map[string]string{
		src.MapBackgroundMetaPath(title.DefaultSlug, cle):             dst.MapBackgroundMetaPath(title.DefaultSlug, cle),
		src.MapBackgroundImageFilePath(title.DefaultSlug, meta.Image): dst.MapBackgroundImageFilePath(title.DefaultSlug, meta.Image),
	} {
		blob, err := os.ReadFile(from)
		if err != nil {
			t.Fatal(err)
		}
		ecrire(t, to, string(blob))
	}
}

// TestGetReplay_DecorDeStarboardParLeMapID — LE CABLAGE DE PRODUCTION (revue RR-M7-02) : un
// artefact servi par `GetReplay`, la carte resolue par sa cle de PRODUCTION (le map_id Forge que
// `MapKeysForMatch` rend, aucun nom), le fond REEL de Starboard. Le document SERVI porte les six
// decors. Sans l appel `resolveVehicleScenery` de `GetReplay`, ce test rougit : sans verdict, le
// client dessine tout.
func TestGetReplay_DecorDeStarboardParLeMapID(t *testing.T) {
	root := t.TempDir()
	copieFond(t, root, cleFondStarboard)
	blob, err := json.Marshal(docJoue(80.61, decorsStarboard()...))
	if err != nil {
		t.Fatal(err)
	}
	ecrire(t, title.NewPathResolver(root).ReplayArtifactPath(title.DefaultSlug, "ab526724"), string(blob))
	svc := NewReplayService(title.DefaultSlug, root, &mapNamesStub{mapID: cleFondStarboard})

	doc, err := svc.GetReplay(context.Background(), "ab526724")
	if err != nil {
		t.Fatalf("GetReplay : %v", err)
	}
	v := doc.VehicleScenery
	if v == nil || v.Zone != sceneryZoneMap || len(v.Hidden) != 6 {
		t.Fatalf("document servi : 6 decors attendus dans vehicleScenery.hidden, rendu %+v", v)
	}
	for _, h := range v.Hidden {
		if h.Reason != sceneryReasonOffPlayArea || h.Gen != 1 {
			t.Errorf("vie %d/%d : raison %q", h.Slot, h.Gen, h.Reason)
		}
	}
}

// TestVehicleScenery_LeCadreDecideSansDecoder (revue RR-M7-01) : quand aucune candidate ne tombe
// dans le cadre, l image n est PAS decodee — prouve en remplacant l image par des octets
// indecodables : le verdict tient (le cadre suffit), alors qu une candidate DANS le cadre, sur la
// meme image, rend la zone inconnue (image illisible, rien de masque).
func TestVehicleScenery_LeCadreDecideSansDecoder(t *testing.T) {
	root := t.TempDir()
	copieFond(t, root, cleFondStarboard)
	res := title.NewPathResolver(root)
	meta, err := replay.LoadMapBackground(res.MapBackgroundMetaPath(title.DefaultSlug, cleFondStarboard))
	if err != nil {
		t.Fatal(err)
	}
	ecrire(t, res.MapBackgroundImageFilePath(title.DefaultSlug, meta.Image), "RIFF-indecodable")
	svc := NewReplayService(title.DefaultSlug, root, &mapNamesStub{mapID: cleFondStarboard}).(*replayService)
	keys := svc.matchMapKeys(context.Background(), "m7")

	horsCadre := docJoue(80.61, decorsStarboard()...)
	svc.resolveVehicleScenery(context.Background(), horsCadre, "m7", keys)
	if v := horsCadre.VehicleScenery; v == nil || v.Zone != sceneryZoneMap || len(v.Hidden) != 6 {
		t.Fatalf("hors du cadre : le calage suffit, 6 decors attendus, rendu %+v", v)
	}
	dansLeCadre := docJoue(80.61, viePosee(780, "warthog", 3, -90, 81))
	svc.resolveVehicleScenery(context.Background(), dansLeCadre, "m7", keys)
	if v := dansLeCadre.VehicleScenery; v == nil || v.Zone != sceneryZoneUnknown || len(v.Hidden) != 0 {
		t.Fatalf("dans le cadre, image illisible : zone inconnue, rien masque ; rendu %+v", v)
	}
}

// TestVehicleScenery_CoutMemoire (revue RR-M7-01) : ce qu une requete ALLOUE. Avant le correctif,
// chaque requete fermait le masque entier (713 Mo pour Behemoth, 194 Mo pour Goliath, 163 Mo pour
// Starboard). Apres : le premier decodage d un fond est paye une fois par processus, les requetes
// suivantes ne paient que la fermeture en un point.
func TestVehicleScenery_CoutMemoire(t *testing.T) {
	behemoth := func() *replay.ReplayDocument {
		return docJoue(3.86, viePosee(773, "mongoose", -101.61, 27.63, 8.8),
			viePosee(774, "mongoose", -101.65, 80.2, 8.31),
			viePosee(768, "mongoose", -146.02, 27.43, 8.59),
			viePosee(769, "mongoose", -146.11, 80.31, 8.33))
	}
	mesure := func(doc *replay.ReplayDocument, noms ...string) uint64 {
		var avant, apres runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&avant)
		resoudreDecor(t, doc, noms...)
		runtime.ReadMemStats(&apres)
		return apres.TotalAlloc - avant.TotalAlloc
	}
	videCacheDesMasques()
	premier := mesure(behemoth(), "Behemoth")
	suivant := mesure(behemoth(), "Behemoth")
	goliath := mesure(docJoue(64.47, viePosee(768, "wasp", -0.44, -8.16, 61.44)), "Goliath")
	starboard := mesure(docJoue(80.61, decorsStarboard()...), "Starboard")
	t.Logf("alloue : Behemoth 1re requete %.1f Mo, suivante %.1f Mo ; Goliath %.1f Mo ; Starboard %.1f Mo",
		mo(premier), mo(suivant), mo(goliath), mo(starboard))
	const plafondRequete = 32 << 20 // une requete servie depuis le cache, ou sans decodage
	if suivant > plafondRequete || starboard > plafondRequete {
		t.Errorf("requete servie : Behemoth %.1f Mo, Starboard %.1f Mo (plafond %.0f Mo)",
			mo(suivant), mo(starboard), mo(plafondRequete))
	}
}

func mo(n uint64) float64 { return float64(n) / (1 << 20) }

// videCacheDesMasques remet le cache du processus a vide : la premiere requete mesuree est FROIDE.
func videCacheDesMasques() {
	sceneryMasksMu.Lock()
	defer sceneryMasksMu.Unlock()
	sceneryMasksCache = map[string]sceneryMaskEntry{}
	sceneryMasksOrdre = nil
}
