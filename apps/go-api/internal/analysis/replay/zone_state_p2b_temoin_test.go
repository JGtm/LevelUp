package replay

// zone_state_p2b_temoin_test.go — LOT C-bis PHASE 2b : LE TEMOIN SUR FILM REEL.
//
// CE QU'IL FAIT, ET POURQUOI IL EXISTE A COTE DE `cmd/replay-build`. Le calque publie
// (`zoneStates`) doit se juger SUR LA FORME PUBLIEE, pas sur les compteurs internes : ce test
// decode un film du corpus, assemble le document par le chemin de PRODUCTION
// (`BuildFromPositions`), puis relit les intervalles SORTIS pour les confronter aux captures
// nommees. Il publie les chiffres de controle du journal du lot.
//
// POURQUOI PAS LE CLI. La cuisson complete d'un artefact passe par le calque du DRAPEAU, et sur
// les deux films de Bastion du corpus ce calque part en vrille (22 Go de memoire, aucune sortie
// en 15 min — mesure du 2026-08-18, decouverte hors perimetre ecrite au journal). Le temoin
// isole donc ce que la phase 2b publie : `FlagInput` reste vide, tout le reste est le chemin de
// production.
//
// SOUS GARDE D'ENVIRONNEMENT (`ZONE_FILM`), un film par processus, avant-plan — meme regime que
// la phase 2a, dont ce fichier reprend le corpus gele, les entrees et l'oracle.
//
//	$env:CGO_ENABLED=0
//	$env:ZONE_FILM="C:/Users/Guillaume/Projects/LevelUp/data/cache/film_chunks/7344d24f"
//	go test -count=1 -run TestZoneEtatPhase2bTemoin -v -timeout 30m ./internal/analysis/replay/

import (
	"encoding/json"
	"fmt"
	"sort"

	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/analysis/replay/mapvar"
	"levelup/go-api/internal/domain/title"
)

// p2bRoles nomme les roles du catalogue employes, dans l'ordre — la meme chaine que
// `replaybuild.matchZones` publie dans la couverture.
func p2bRoles(f p2aFilm) string {
	if f.Mode == "Strongholds" {
		return string(mapvar.RoleStrongholdZone)
	}
	return string(mapvar.RoleStrongholdZone) + "," + string(mapvar.RoleExtractionZone)
}

// TestZoneEtatPhase2bTemoin cuit le calque des zones sur UN film et publie ses chiffres.
func TestZoneEtatPhase2bTemoin(t *testing.T) {
	dir := p2aRequireFilm(t)
	short, film := p2aFilmOf(t, dir)
	src := p2aSource(t, dir)
	quant := p2aQuant(t, film.Carte)
	zones := p2aZones(t, film.MapID, p2aRolesDuMode(film)...)
	caps := p2aCaptures(src, film)
	sc := p2bScan(t, dir)
	doc, origin := p2bBuild(t, dir, short, quant, ZoneInput{
		Scanned: true, Reads: sc.Reads, Zones: zones, Roles: p2bRoles(film),
		TeamByXUID: film.p2aTeams(),
	}, caps)

	t.Logf("FILM %s (%s, %s) — %d zones au catalogue, %d captures nommees, %d lectures ti=13",
		short, film.Mode, film.Carte, len(zones), len(caps), len(sc.Reads))
	t.Logf("  BALAYAGE : %d slots, %d records ancres, %d marches abouties, %d CHAINEES (%.1f %%)",
		sc.Slots, sc.Records, sc.Walked, sc.Chained, 100*p2aRate(sc.Chained, sc.Walked))
	t.Logf("  REJEU : %d trajectoires, %d frames de %d ms · actions publiees %d",
		len(doc.Tracks), doc.FrameCount, doc.FrameIntervalMS, len(doc.Objectives))
	p2bLogCouverture(t, doc)
	p2bInventaireCanaux(t, doc, film, sc, origin)
	p2bLogEtats(t, doc)
	p2bControleCaptures(t, doc, film)
	p2bControleColline(t, doc)
	p2bLogTaille(t, doc)
}

// p2bScan balaye les proprietes reseau de `ti=13` par le chemin de PRODUCTION.
func p2bScan(t *testing.T, dir string) filmdec.ManagedPropertyScan {
	t.Helper()
	release := filmdec.LockProcessDecode()
	defer release()
	sc, err := filmdec.ScanFilmManagedProperties(dir)
	if err != nil {
		t.Fatalf("balayage ti=13 impossible (%s) : %v", dir, err)
	}
	return sc
}

// p2bBuild assemble le document avec le calque des zones — chemin de production, calque du
// drapeau EXCLU (cf. l'en-tete).
func p2bBuild(t *testing.T, dir, short string, quant *filmdec.MapQuantEntry, zone ZoneInput,
	caps []objectiveevents.IdentifiedEvent,
) (ReplayDocument, uint64) {
	t.Helper()
	release := filmdec.LockProcessDecode()
	defer release()
	worldRange := quant.Range()
	scan := filmdec.DefaultScanFilmOptions()
	scan.WorldRange = &worldRange
	pos, err := filmdec.ScanFilmBipedPositions(dir, scan)
	if err != nil {
		t.Fatalf("positions illisibles (%s) : %v", dir, err)
	}
	opt := Options{Objectives: caps, Zone: zone, MapQuant: quant}
	deaths, err := ScanFilmDeaths(dir)
	if err != nil {
		t.Fatalf("fil des morts illisible (%s) : %v", dir, err)
	}
	opt.Deaths = deaths
	if idx, err := ScanFilmPlayerIndices(dir, rosterFromDeaths(deaths)); err == nil {
		table, _ := injectiveOrEmpty(idx)
		opt.PlayerIndices = table
	}
	clockUS, err := ScanFilmClockOrigin(dir)
	if err != nil {
		t.Logf("origine d'horloge illisible : %v — les instants ne seront pas recales", err)
	}
	opt.FilmClockOriginUS = clockUS
	origin := uint64(0)
	for i, p := range pos {
		if i == 0 || p.TimestampUS < origin {
			origin = p.TimestampUS
		}
	}
	return BuildFromPositions(short, title.DefaultSlug, pos, nil, opt), origin
}

// p2bLogCouverture publie `coverage.zones` tel que l'artefact le porte.
func p2bLogCouverture(t *testing.T, doc ReplayDocument) {
	t.Helper()
	if doc.Coverage == nil || doc.Coverage.Zones == nil {
		t.Fatalf("aucune couverture de zone publiee — le calque n'a rien lu")
	}
	c := doc.Coverage.Zones
	t.Logf("  COUVERTURE : methode %s · roles %q · catalogue %d · slots %d · apparies %d ·"+
		" non apparies %d · captures %d dont %d attribuees · intervalles %d · periodes colline %d"+
		" · proprietaire %d/%d · valeurs inconnues %d",
		c.Method, c.Roles, c.Catalog, c.Slots, c.Paired, c.Unpaired, c.Captures, c.Attributed,
		c.Spans, c.HillPeriods, c.OwnerAgreed, c.OwnerChecked, c.UnknownOwner)
}

// p2bLogEtats publie la forme SERIALISEE des premiers intervalles de chaque zone.
func p2bLogEtats(t *testing.T, doc ReplayDocument) {
	t.Helper()
	for _, st := range doc.ZoneStates {
		extrait := st
		if len(extrait.Spans) > 4 {
			extrait.Spans = extrait.Spans[:4]
		}
		blob, err := json.Marshal(extrait)
		if err != nil {
			t.Fatalf("serialisation de l'etat de zone %d : %v", st.ZoneRef, err)
		}
		t.Logf("  zone %d : %d intervalle(s) — extrait %s", st.ZoneRef, len(st.Spans), blob)
	}
}

// p2bControleCaptures est LE CONTROLE DU LOT : chaque capture nommee attribuee est-elle suivie
// d'un intervalle appartenant a l'equipe du capteur ?
//
// IL SE LIT SUR LA FORME PUBLIEE, pas sur les compteurs internes : c'est ce que le client
// dessinera. Une capture dont la zone n'a pas d'etat publie compte dans le denominateur — la
// taire ferait passer un calque partiel pour un calque juste.
func p2bControleCaptures(t *testing.T, doc ReplayDocument, film p2aFilm) {
	t.Helper()
	if doc.Coverage == nil || doc.Coverage.Zones == nil || len(doc.ZoneStates) == 0 {
		t.Logf("  CONTROLE captures -> proprietaire : SANS OBJET (aucun etat publie)")
		return
	}
	teams := film.p2aTeams()
	// Le catalogue est renumerote comme en production : le rang spatial EST l'index servi.
	cat := zoneCatalogOf(p2aZones(t, film.MapID, p2aRolesDuMode(film)...))
	caps := zoneCapturesOf(doc.Objectives)
	att, cov := AttributeZones(caps, doc.Tracks, cat, AttributeOptions{MaxDistanceM: zoneCaptureDistanceM})
	t.Logf("  ATTRIBUTION geometrique des captures : %d/%d (hors %d, sans position %d, ambigues %d)",
		cov.Attributed, cov.Actions, cov.Outside, cov.NoPosition, cov.Ambiguous)
	ok, total := 0, 0
	for _, a := range att {
		team, known := teams[a.Action.XUID]
		if !a.Attributed || !known {
			continue
		}
		total++
		if p2bOwnerAt(doc.ZoneStates, a.SpatialRank, a.Action.T) == team {
			ok++
		}
	}
	t.Logf("  CONTROLE captures -> intervalle du capteur : %d/%d = %.1f %% (seuil 90 %%)",
		ok, total, 100*p2aRate(ok, total))
}

// p2bOwnerAt rend le camp publie pour une zone a une frame, ou -1 quand aucun intervalle ne la
// couvre ou que personne ne la tient. Deux frames de tolerance : le canal de propriete est
// replique JUSTE APRES la statistique.
func p2bOwnerAt(states []ZoneState, ref, frame int) int {
	for _, st := range states {
		if st.ZoneRef != ref {
			continue
		}
		for _, sp := range st.Spans {
			if frame < sp.T0-2 || frame > sp.T1 {
				continue
			}
			if sp.Owner == nil {
				return -1
			}
			return *sp.Owner
		}
	}
	return -1
}

// p2bControleColline mesure la part de l'axe couverte par des intervalles ACTIFS (clause KOTH).
func p2bControleColline(t *testing.T, doc ReplayDocument) {
	t.Helper()
	actives := 0
	for _, st := range doc.ZoneStates {
		for _, sp := range st.Spans {
			if sp.Active {
				actives += sp.T1 - sp.T0 + 1
			}
		}
	}
	if actives == 0 {
		return // mode a zones simultanees : la clause est sans objet
	}
	t.Logf("  CONTROLE colline : %d frames ACTIVES sur %d = %.1f %% (seuil 80 %%)",
		actives, doc.FrameCount, p2aRate(actives, doc.FrameCount)*100)
}

// p2bLogTaille publie le poids du calque dans l'artefact — avant et apres, en octets.
func p2bLogTaille(t *testing.T, doc ReplayDocument) {
	t.Helper()
	avec, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("serialisation du document : %v", err)
	}
	sans := doc
	sans.ZoneStates = nil
	if sans.Coverage != nil {
		cov := *sans.Coverage
		cov.Zones = nil
		sans.Coverage = &cov
	}
	blob, err := json.Marshal(sans)
	if err != nil {
		t.Fatalf("serialisation du document sans zones : %v", err)
	}
	t.Logf("  TAILLE : %d octets avec le calque, %d sans — %d octets (%.3f %%)",
		len(avec), len(blob), len(avec)-len(blob),
		100*float64(len(avec)-len(blob))/float64(len(blob)))

}

// p2bInventaireCanaux publie, pour CHAQUE slot porteur du canal de propriete, ses valeurs et le
// nombre de ses changements qui SUIVENT une capture attribuee.
//
// C'EST LA MESURE QUI DEPARTAGE LES CANAUX. Le corpus en porte plusieurs par carte : celui qui
// dit le proprietaire change QUAND la zone est prise ; les autres bougent pour d'autres raisons.
// Sans cet inventaire, un taux de concordance faible ne dit pas SI la regle est fausse ou si elle
// a elu le mauvais slot.
func p2bInventaireCanaux(t *testing.T, doc ReplayDocument, film p2aFilm,
	sc filmdec.ManagedPropertyScan, origin uint64,
) {
	t.Helper()
	cat := zoneCatalogOf(p2aZones(t, film.MapID, p2aRolesDuMode(film)...))
	att, _ := AttributeZones(zoneCapturesOf(doc.Objectives), doc.Tracks, cat,
		AttributeOptions{MaxDistanceM: zoneCaptureDistanceM})
	pairs := zonePairsOf(att)
	c := zoneCtx{origin: origin, step: uint64(doc.FrameIntervalMS) * 1000,
		frames: doc.FrameCount, intervalMS: doc.FrameIntervalMS}
	ser := zoneSeriesOf(sc.Reads, c)
	win := zoneWindowFrames(doc.FrameIntervalMS)
	for _, slot := range sortedZoneSlots(ser.owner) {
		ss := ser.owner[slot]
		ch := zoneChanges(ss)
		apres := 0
		for _, e := range ch {
			for _, p := range pairs {
				if d := e.t - p.t; d >= 0 && d <= win {
					apres++
					break
				}
			}
		}
		vals := map[uint64]int{}
		for _, s := range ss {
			vals[s.v]++
		}
		t.Logf("    slot %-5d tag4 : %d emissions, %d changements dont %d SUIVENT une capture ·"+
			" valeurs %v", slot, len(ss), len(ch), apres, p2bValeurs(vals))
	}
}

// p2bValeurs rend une distribution triee par frequence.
func p2bValeurs(m map[uint64]int) []string {
	vals := make([]uint64, 0, len(m))
	for v := range m {
		vals = append(vals, v)
	}
	sort.Slice(vals, func(i, j int) bool { return m[vals[i]] > m[vals[j]] })
	out := make([]string, 0, len(vals))
	for i, v := range vals {
		if i == 4 {
			out = append(out, "...")
			break
		}
		out = append(out, fmt.Sprintf("0x%08X x%d", v, m[v]))
	}
	return out
}
