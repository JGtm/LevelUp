package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain/replaydoc"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// Les matchs de la carte : le match prevu et quatre autres.
const (
	prevuID     = "00000001-0000-4000-8000-000000000000"
	carteID     = "carte-test"
	varianteAss = "Team Slayer:Arena"
	varianteBas = "Strongholds:Arena"
)

func autreID(n int) string { return fmt.Sprintf("0000001%d-0000-4000-8000-000000000000", n) }

// registreSpawnStub est un registre des matchs : il rend les matchs qu'on lui donne.
type registreSpawnStub struct {
	mapNamesStub
	matchs []port.SpawnMatch
	absent bool
}

func (r *registreSpawnStub) SpawnMatchOf(_ context.Context, id string) (port.SpawnMatch, error) {
	for _, m := range r.matchs {
		if m.MatchID == id && !r.absent {
			return m, nil
		}
	}
	return port.SpawnMatch{}, port.ErrMatchMapUnknown
}

func (r *registreSpawnStub) SpawnMatchesOnMap(context.Context, string) ([]port.SpawnMatch, error) {
	return r.matchs, nil
}

const famillesTest = `
[meta]
title_slug = "halo_infinite"
schema_version = 1
[[families]]
id = "bases"
variant_contains = ["stronghold"]
[[families]]
id = "assassin"
variant_contains = ["slayer"]
`

const poidsTest = `{"revision":1,"date":"2026-11-15","definition":"test","familles":{` +
	`"assassin":{"poids":{"ennemi_le_plus_proche_10m":2}},"tous":{"poids":{"ennemi_le_plus_proche_10m":1}}}}`

// artefactPrevu : « a » (equipe 0) meurt a la frame 100 en (50, 0, 0) et reapparait a la frame 340
// en (10, 0, 0) ; « e » (equipe 1) est loin de la zone 1, dans la zone 2.
const artefactPrevu = `{"schemaVersion":124,"matchId":"` + prevuID + `","titleSlug":"halo_infinite","frameCount":600,` +
	`"frameIntervalMs":33,"tracks":[` +
	`{"slot":1,"team":0,"xuid":"xuid-a","points":[{"t":0,"x":50,"y":0},{"t":100,"x":50,"y":0}]},` +
	`{"slot":1,"team":0,"xuid":"xuid-a","points":[{"t":340,"x":10,"y":0},{"t":500,"x":12,"y":0}]},` +
	`{"slot":2,"team":1,"xuid":"xuid-e","points":[{"t":0,"x":110,"y":0},{"t":500,"x":110,"y":0}]}],` +
	`"respawnCountdowns":[{"filmIndex":0,"from":100,"to":340,"seconds":8,"startRead":true}],` +
	`"identity":{"players":[{"filmIndex":0,"xuid":"xuid-a"},{"filmIndex":1,"xuid":"xuid-e"}]}}`

// sidecarAvecReapparitions : un sidecar courant dont le joueur reapparait aux points donnes (x, y=0, z=0).
func sidecarAvecReapparitions(schema int, xs ...float64) string {
	var spawns []string
	for _, x := range xs {
		spawns = append(spawns, fmt.Sprintf(
			`{"frame":10,"x":%v,"y":0,"z":0,"reapparition":true,"mort":{"frame":5,"x":0,"y":0,"z":0},"instantane":[{"equipe":1,"x":1,"y":1,"z":0}]}`, x))
	}
	return fmt.Sprintf(`{"schema_version":%d,"pas_m":0.5,"pas_echantillon_ms":250,"joueurs":[{"xuid":"x","equipe":0,"spawns":[%s]}]}`,
		schema, strings.Join(spawns, ","))
}

type decorPrevision struct {
	root string
	reg  *registreSpawnStub
	caps games.CapabilityMap
}

// nouveauDecor pose les fichiers de reference (familles, poids, zones de la carte) et l'artefact du match prevu.
func nouveauDecor(t *testing.T) *decorPrevision {
	t.Helper()
	root := t.TempDir()
	res := title.NewPathResolver(root)
	ecrire(t, filepath.Join(res.TitleMappingsDir(title.DefaultSlug), fichierFamillesReapparition), famillesTest)
	ecrire(t, res.SpawnModelPath(title.DefaultSlug), poidsTest)
	ecrire(t, res.MapQuantBoundsPath(title.DefaultSlug),
		`{"schemaVersion":1,"source":"test","maps":{"cliffhanger":{"module":"ridgeline","min":[-10,-10,-10],"max":[10,10,10],"axisWidths":[11,11,11]}}}`)
	ecrire(t, res.MapCalloutsPath(title.DefaultSlug),
		`{"schema_version":1,"title_slug":"halo_infinite","source":"test","maps":{"ridgeline":{"module":"ridgeline","provenance":"decoupe","zones":[`+
			`{"volume_index":1,"name":"one","en":"Zone One","fr":"Zone Un","x":15,"y":0,"z":0,"z_bottom":-5,"z_top":5,"big":false,"polygon":[[0,-10],[30,-10],[30,10],[0,10]]},`+
			`{"volume_index":2,"name":"two","en":"Zone Two","fr":"Zone Deux","x":115,"y":0,"z":0,"z_bottom":-5,"z_top":5,"big":false,"polygon":[[100,-10],[130,-10],[130,10],[100,10]]}]}}}`)
	ecrire(t, res.ReplayArtifactPath(title.DefaultSlug, prevuID), artefactPrevu)
	d := &decorPrevision{root: root, caps: games.CapabilityMap{games.CapFilmSpawnForecast: games.CapSupported}}
	d.reg = &registreSpawnStub{mapNamesStub: mapNamesStub{mapID: carteID, names: []string{"Cliffhanger"}}}
	d.reg.matchs = []port.SpawnMatch{{MatchID: prevuID, MapID: carteID, Variant: varianteAss, Category: "Assassin"}}
	return d
}

// sidecarPlein : un sidecar courant qui reapparait `tours` fois en zone 1 puis en zone 2 (2 x tours
// reapparitions, deux points candidats distincts).
func sidecarPlein(tours int) string {
	var xs []float64
	for i := 0; i < tours; i++ {
		xs = append(xs, 10, 110)
	}
	return sidecarAvecReapparitions(7, xs...)
}

// autres ajoute `n` matchs d'Assassin sur la carte, chacun avec un sidecar plein de dix tours : trois
// matchs depassent le plancher de reapparitions de la lecture Tactique.
func (d *decorPrevision) autres(t *testing.T, n int) {
	t.Helper()
	d.autresDeVariante(t, n, varianteAss)
}

func (d *decorPrevision) autresDeVariante(t *testing.T, n int, variante string) {
	t.Helper()
	for i := 1; i <= n; i++ {
		d.match(t, autreID(i), variante, sidecarPlein(10))
	}
}

func (d *decorPrevision) match(t *testing.T, id, variante, sidecar string) {
	t.Helper()
	d.reg.matchs = append(d.reg.matchs, port.SpawnMatch{MatchID: id, MapID: carteID, Variant: variante, Category: "Assassin"})
	if sidecar != "" {
		ecrire(t, title.NewPathResolver(d.root).TacticalRasterPath(title.DefaultSlug, id), sidecar)
	}
}

func (d *decorPrevision) previsionDans(t *testing.T, langue string) (replaydoc.SpawnForecast, error) {
	t.Helper()
	svc := NewReplayService(title.DefaultSlug, d.root, d.reg, nil, WithReplayCapabilities(d.caps))
	return svc.SpawnForecast(ctxkeys.WithLocale(context.Background(), langue), prevuID)
}

func (d *decorPrevision) prevision(t *testing.T) replaydoc.SpawnForecast {
	t.Helper()
	f, err := d.previsionDans(t, "fr")
	if err != nil {
		t.Fatalf("SpawnForecast: %v", err)
	}
	return f
}

func TestSpawnForecast_Servie(t *testing.T) {
	d := nouveauDecor(t)
	d.autres(t, 3)
	f := d.prevision(t)
	if f.Status != replaydoc.SpawnForecastStatusReady || f.Reason != "" || f.Message != "" {
		t.Fatalf("statut = %q (%s), attendu ready", f.Status, f.Reason)
	}
	if f.Family != "assassin" || f.Model != "assassin" || f.ModelDate != "2026-11-15" || f.SampleStepMs != 1000 {
		t.Errorf("modele = %q/%q/%q/%d", f.Family, f.Model, f.ModelDate, f.SampleStepMs)
	}
	if f.MatchesOnMap != 4 || f.Candidates != 2 {
		t.Errorf("matchs = %d, candidats = %d, attendu 4 et 2", f.MatchesOnMap, f.Candidates)
	}
	if len(f.Deaths) != 1 {
		t.Fatalf("morts = %d, attendu 1", len(f.Deaths))
	}
	m := f.Deaths[0]
	if m.XUID != "xuid-a" || m.DeathFrame != 100 || m.RespawnFrame != 340 || len(m.Samples) != 8 {
		t.Errorf("mort = %+v", m)
	}
	if m.Actual.X != 10 || m.ActualZone == nil || *m.ActualZone != 1 {
		t.Errorf("point reel = %+v zone %v, attendu (10, 0, 0) en zone 1", m.Actual, m.ActualZone)
	}
	// L'ennemi est dans la zone 2 : loin de lui, la zone 1 est la plus probable, des la mort.
	if first := m.Samples[0]; len(first.Zones) == 0 || first.Zones[0].Zone != 1 || first.Zones[0].P < 0.99 {
		t.Errorf("premier echantillon = %+v, attendu la zone 1 devant", first)
	}
	if len(f.Zones) != 1 || f.Zones[0].VolumeIndex != 1 || f.Zones[0].Name != "Zone Un" {
		t.Errorf("zones = %+v, attendu la seule zone designee (la zone 2 ne porte aucune probabilite), en francais", f.Zones)
	}
	if en, _ := d.previsionDans(t, "en"); en.Zones[0].Name != "Zone One" {
		t.Errorf("zone en anglais = %q, attendu le libelle de la langue de la requete", en.Zones[0].Name)
	}
}

// Une prevision ne voit pas la reponse : les reapparitions du match prevu ne changent ni les candidats,
// ni le nombre de matchs de la carte.
func TestSpawnForecast_NeLitPasLeSidecarDuMatchPrevu(t *testing.T) {
	d := nouveauDecor(t)
	d.autres(t, 2)
	sans := d.prevision(t)
	ecrire(t, title.NewPathResolver(d.root).TacticalRasterPath(title.DefaultSlug, prevuID),
		sidecarAvecReapparitions(7, 10, 110, 20, 25))
	avec := d.prevision(t)
	if sans.Candidates != 2 || avec.Candidates != 2 || avec.MatchesOnMap != sans.MatchesOnMap {
		t.Errorf("candidats %d -> %d, matchs %d -> %d : le match prevu a fuite", sans.Candidates, avec.Candidates,
			sans.MatchesOnMap, avec.MatchesOnMap)
	}
	a, _ := json.Marshal(avec.Deaths)
	b, _ := json.Marshal(sans.Deaths)
	if string(a) != string(b) {
		t.Errorf("la prevision change avec le sidecar du match prevu")
	}
}

func TestSpawnForecast_UneAutreFamilleNAlimentePasLesCandidats(t *testing.T) {
	d := nouveauDecor(t)
	d.autres(t, 1)
	for i := 2; i <= 3; i++ {
		d.match(t, autreID(i), varianteBas, sidecarAvecReapparitions(7, 10, 110))
	}
	if f := d.prevision(t); f.Status != replaydoc.SpawnForecastStatusUnavailable || f.Reason != replaydoc.SpawnForecastReasonNotEnough {
		t.Errorf("statut = %q (%s), attendu not_enough_matches : deux matchs d'une autre famille ne comptent pas", f.Status, f.Reason)
	}
}

func TestSpawnForecast_SidecarPerimeOuAbsentIgnore(t *testing.T) {
	d := nouveauDecor(t)
	d.match(t, autreID(1), varianteAss, sidecarAvecReapparitions(7, 10, 110))
	d.match(t, autreID(2), varianteAss, sidecarAvecReapparitions(6, 10, 110)) // schema anterieur
	d.match(t, autreID(3), varianteAss, "")                                   // pas encore cuit
	d.match(t, autreID(4), varianteAss, "pas du json")                        // illisible
	f := d.prevision(t)
	if f.Status != replaydoc.SpawnForecastStatusUnavailable || f.Reason != replaydoc.SpawnForecastReasonNotEnough {
		t.Errorf("statut = %q (%s), attendu not_enough_matches", f.Status, f.Reason)
	}
	if f.Message == "" || f.Zones == nil || f.Deaths == nil {
		t.Errorf("une reponse indisponible porte un message et des listes vides, pas nil : %+v", f)
	}
}

func TestSpawnForecast_ReponsesPropres(t *testing.T) {
	cas := []struct {
		nom    string
		poser  func(*testing.T, *decorPrevision)
		raison string
	}{
		{"modele absent", func(t *testing.T, d *decorPrevision) {
			if err := os.Remove(title.NewPathResolver(d.root).SpawnModelPath(title.DefaultSlug)); err != nil {
				t.Fatal(err)
			}
		}, replaydoc.SpawnForecastReasonNoModel},
		{"match inconnu du registre", func(t *testing.T, d *decorPrevision) { d.reg.absent = true }, replaydoc.SpawnForecastReasonMapUnknown},
		{"carte sans zone", func(t *testing.T, d *decorPrevision) {
			ecrire(t, title.NewPathResolver(d.root).MapCalloutsPath(title.DefaultSlug),
				`{"schema_version":1,"title_slug":"halo_infinite","source":"test","maps":{}}`)
			d.autres(t, 3)
		}, replaydoc.SpawnForecastReasonNoZone},
		{"carte pauvre", func(t *testing.T, d *decorPrevision) { d.autres(t, 1) }, replaydoc.SpawnForecastReasonNotEnough},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			d := nouveauDecor(t)
			c.poser(t, d)
			f, err := d.previsionDans(t, "fr")
			if err != nil {
				t.Fatalf("une prevision sans objet est une reponse, pas une erreur : %v", err)
			}
			if f.Status != replaydoc.SpawnForecastStatusUnavailable || f.Reason != c.raison || f.Message == "" {
				t.Errorf("statut = %q raison = %q message = %q, attendu unavailable/%s avec un message", f.Status, f.Reason, f.Message, c.raison)
			}
			if len(f.Deaths) != 0 {
				t.Errorf("aucune mort ne se sert sans prevision : %+v", f.Deaths)
			}
		})
	}
}

// Aucun filtre par mode : une famille sans poids propres, un match sans famille et un titre sans table de
// familles se servent avec les poids « tous », les candidats venant des matchs de la meme famille.
func TestSpawnForecast_AucunFiltreParMode(t *testing.T) {
	cas := []struct {
		nom     string
		poser   func(*testing.T, *decorPrevision)
		famille string
	}{
		{"famille sans poids propres", func(t *testing.T, d *decorPrevision) {
			d.reg.matchs[0].Variant = varianteBas
			d.autresDeVariante(t, 3, varianteBas)
		}, "bases"},
		{"mode sans famille", func(t *testing.T, d *decorPrevision) {
			d.reg.matchs[0].Variant = "Arena:VIP"
			d.autresDeVariante(t, 3, "Arena:VIP")
		}, ""},
		{"fichier de familles absent", func(t *testing.T, d *decorPrevision) {
			if err := os.Remove(filepath.Join(title.NewPathResolver(d.root).TitleMappingsDir(title.DefaultSlug), fichierFamillesReapparition)); err != nil {
				t.Fatal(err)
			}
			d.autres(t, 3)
		}, ""},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			d := nouveauDecor(t)
			c.poser(t, d)
			f := d.prevision(t)
			if f.Status != replaydoc.SpawnForecastStatusReady || f.Family != c.famille || f.Model != "tous" {
				t.Errorf("statut = %q (%s) famille = %q modele = %q, attendu ready, %q, tous", f.Status, f.Reason, f.Family, f.Model, c.famille)
			}
		})
	}
}

// Les seuils de donnees sont ceux de la lecture Tactique : trois matchs et vingt reapparitions des autres
// matchs de la carte et de la famille. Trois matchs aux reapparitions trop rares ne suffisent pas.
func TestSpawnForecast_SeuilsDeDonnees(t *testing.T) {
	d := nouveauDecor(t)
	for i := 1; i <= 2; i++ {
		d.match(t, autreID(i), varianteAss, sidecarPlein(4)) // 16 reapparitions : sous le plancher de 20
	}
	if f := d.prevision(t); f.Reason != replaydoc.SpawnForecastReasonNotEnough || f.MatchesOnMap != 3 {
		t.Errorf("statut = %q (%s), matchs = %d, attendu not_enough_matches (16 reapparitions sous 20)", f.Status, f.Reason, f.MatchesOnMap)
	}
	d.match(t, autreID(3), varianteAss, sidecarPlein(1)) // 18 reapparitions : encore sous 20
	if f := d.prevision(t); f.Reason != replaydoc.SpawnForecastReasonNotEnough {
		t.Errorf("raison = %q avec 18 reapparitions, attendu not_enough_matches", f.Reason)
	}
	d.match(t, autreID(4), varianteAss, sidecarPlein(1)) // 20 : plancher atteint
	if f := d.prevision(t); f.Status != replaydoc.SpawnForecastStatusReady {
		t.Errorf("statut = %q (%s) avec 20 reapparitions, attendu ready", f.Status, f.Reason)
	}
}

func TestSpawnForecast_ModeleDeRepliTous(t *testing.T) {
	d := nouveauDecor(t)
	ecrire(t, title.NewPathResolver(d.root).SpawnModelPath(title.DefaultSlug),
		`{"revision":1,"date":"2026-11-15","definition":"t","familles":{"tous":{"poids":{"ennemi_le_plus_proche_10m":1}}}}`)
	d.autres(t, 3)
	if f := d.prevision(t); f.Status != replaydoc.SpawnForecastStatusReady || f.Model != "tous" || f.Family != "assassin" {
		t.Errorf("modele = %q famille = %q statut = %q, attendu le jeu « tous » pour la famille assassin", f.Model, f.Family, f.Status)
	}
}

func TestSpawnForecast_Refus(t *testing.T) {
	d := nouveauDecor(t)
	d.autres(t, 3)
	d.caps = games.CapabilityMap{}
	if _, err := d.previsionDans(t, "fr"); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("sans la capability du titre : %v, attendu ErrCapabilityNotSupported", err)
	}
	d.caps = games.CapabilityMap{games.CapFilmSpawnForecast: games.CapSupported}
	if err := os.Remove(title.NewPathResolver(d.root).ReplayArtifactPath(title.DefaultSlug, prevuID)); err != nil {
		t.Fatal(err)
	}
	if _, err := d.previsionDans(t, "fr"); !errors.Is(err, port.ErrReplayNotAvailable) {
		t.Errorf("sans artefact : %v, attendu ErrReplayNotAvailable", err)
	}
}

// Un registre qui ne sait pas lire les matchs d'une carte (service sans la capacite) sert map_unknown.
func TestSpawnForecast_RegistreNonCable(t *testing.T) {
	d := nouveauDecor(t)
	svc := NewReplayService(title.DefaultSlug, d.root, &mapNamesStub{mapID: carteID}, nil, WithReplayCapabilities(d.caps))
	f, err := svc.SpawnForecast(context.Background(), prevuID)
	if err != nil || f.Reason != replaydoc.SpawnForecastReasonMapUnknown {
		t.Errorf("registre non cable : %+v, %v, attendu map_unknown", f, err)
	}
}

func TestMasquerPrevisionDemo(t *testing.T) {
	index := filepath.Join(t.TempDir(), "index.json")
	if err := os.WriteFile(index, []byte(`{"identities":[{"xuid":"reel","demo_xuid":"demo","demo_gamertag":"D"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	f := replaydoc.SpawnForecast{Deaths: []replaydoc.SpawnForecastDeath{{XUID: "reel"}, {XUID: "bot"}}}
	if err := masquerPrevisionDemo(index, &f); err != nil {
		t.Fatalf("masquerPrevisionDemo: %v", err)
	}
	if f.Deaths[0].XUID != "demo" || f.Deaths[1].XUID != "bot" {
		t.Errorf("xuid = %q, %q, attendu demo puis bot inchange", f.Deaths[0].XUID, f.Deaths[1].XUID)
	}
	if err := masquerPrevisionDemo(filepath.Join(t.TempDir(), "absent.json"), &f); err == nil {
		t.Errorf("un index illisible refuse la prevision : les xuid reels ne sortent jamais")
	}
}
