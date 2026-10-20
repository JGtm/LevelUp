package replay

// zone_etat_initial_temoin_test.go — LE TEMOIN SUR FILM REEL DE L'ETAT INITIAL DES ZONES.
//
// Un Bastion dont la variante donne A au camp 1 et C au camp 0 au coup d'envoi (B neutre) : le
// canal de propriete n'est emis qu'a la premiere reprise de chaque base, et seules les images-cles
// disent l'etat de depart. Le temoin assemble le calque des zones DEUX FOIS par le chemin de
// production (`BuildFromPositions`, meme regime que `zone_state_p2b_temoin_test.go` : calque du
// drapeau exclu), sans puis avec les lectures d'image-cle, et confronte :
//
//	A et C   apres : premier intervalle a la frame de la premiere image-cle, tenu par le camp
//	         attendu ; avant : il ne s'ouvrait qu'a la reprise ; la suite est identique.
//	B        identique avant et apres (neutre au depart, rien ne s'ouvre).
//	controle `ownerChecked` / `ownerAgreed` identiques.
//
// SOUS GARDE D'ENVIRONNEMENT (`ZONE_FILM`), un film par processus, avant-plan :
//
//	$env:CGO_ENABLED=0
//	$env:ZONE_FILM="C:/Users/Guillaume/Downloads/Scripts/LevelUp/data/cache/film_chunks/572e236b"
//	go test -count=1 -run TestZoneEtatInitialTemoin -v -timeout 30m ./internal/games/halo_infinite/film/replay/

import (
	"context"
	"reflect"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/objectives"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// etatInitialAttendu : le camp qui tient chaque base au coup d'envoi, par rang de lettre (0 = A),
// tel que le Theater le montre ; absent = neutre au depart. Seul `572e236b` porte une attente.
var etatInitialAttendu = map[string]map[int]int{"572e236b": {0: 1, 2: 0}}

// TestZoneEtatInitialTemoin assemble le calque des zones sans puis avec l'etat d'image-cle.
func TestZoneEtatInitialTemoin(t *testing.T) {
	dir := p2aRequireFilm(t)
	short, film := p2aFilmOf(t, dir)
	attendu, ok := etatInitialAttendu[short]
	if !ok {
		t.Skipf("film %s sans etat initial attendu", short)
	}
	sc := p2bScan(t, dir)
	t.Logf("FILM %s — images-cles ti=13 : %d records, %d fermes, %d casses, %d non prouves, %d refuses, %d lectures",
		short, sc.KeyRecords, sc.KeyClosed, sc.KeyBroken, sc.KeyUnproven, sc.KeyRefused, len(sc.KeyReads))
	if len(sc.KeyReads) == 0 {
		t.Fatalf("aucune lecture d'image-cle")
	}
	zone := ZoneInput{Scanned: true, Reads: sc.Reads, Zones: p2aZones(t, film.MapID, p2aRolesDuMode(film)...),
		Roles: p2bRoles(film), TeamByXUID: film.p2aTeams()}
	cuire := etatInitialCuisson(t, dir, short, p2aQuant(t, film.Carte))
	caps := p2aCaptures(p2aBobine(t, dir), film)
	avant, _ := cuire(zone, caps)
	zone.KeyReads = sc.KeyReads
	apres, origin := cuire(zone, caps)
	premiere := premiereFrameDImageCle(sc.KeyReads, origin, apres)
	t.Logf("  premiere image-cle : frame %d", premiere)

	if avant.Coverage.Zones.OwnerChecked != apres.Coverage.Zones.OwnerChecked ||
		avant.Coverage.Zones.OwnerAgreed != apres.Coverage.Zones.OwnerAgreed {
		t.Errorf("controle du proprietaire change : %d/%d -> %d/%d", avant.Coverage.Zones.OwnerAgreed,
			avant.Coverage.Zones.OwnerChecked, apres.Coverage.Zones.OwnerAgreed, apres.Coverage.Zones.OwnerChecked)
	}
	for _, st := range apres.ZoneStates {
		if st.LetterRank == nil {
			continue
		}
		av := etatParLettre(t, avant, *st.LetterRank)
		t.Logf("  lettre %d : avant t0=%d camp %d (%d intervalles) ; apres t0=%d camp %d (%d intervalles)",
			*st.LetterRank, av.Spans[0].T0, ownerOf(av.Spans[0]), len(av.Spans),
			st.Spans[0].T0, ownerOf(st.Spans[0]), len(st.Spans))
		camp, tenue := attendu[*st.LetterRank]
		if !tenue {
			if !reflect.DeepEqual(st.Spans, av.Spans) {
				t.Errorf("lettre %d, neutre au depart : ses intervalles changent", *st.LetterRank)
			}
			continue
		}
		verifierBaseTenue(t, *st.LetterRank, camp, premiere, av.Spans, st.Spans)
	}
}

// verifierBaseTenue : la base tenue des le depart s'ouvre a la premiere image-cle, au bon camp,
// et la suite est celle d'avant.
func verifierBaseTenue(t *testing.T, lettre, camp, premiere int, avant, apres []ZoneSpan) {
	t.Helper()
	if apres[0].T0 != premiere || ownerOf(apres[0]) != camp {
		t.Errorf("lettre %d : premier intervalle t0=%d camp %d, attendu t0=%d camp %d",
			lettre, apres[0].T0, ownerOf(apres[0]), premiere, camp)
	}
	if avant[0].T0 <= premiere {
		t.Errorf("lettre %d : sans image-cle l'intervalle s'ouvrait deja a %d", lettre, avant[0].T0)
	}
	if !reflect.DeepEqual(apres[1:], avant) {
		t.Errorf("lettre %d : les intervalles suivants changent", lettre)
	}
}

// etatParLettre rend l'etat publie de la zone de rang de lettre `lettre`.
func etatParLettre(t *testing.T, doc ReplayDocument, lettre int) ZoneState {
	t.Helper()
	for _, st := range doc.ZoneStates {
		if st.LetterRank != nil && *st.LetterRank == lettre {
			return st
		}
	}
	t.Fatalf("lettre %d absente de l'etat publie", lettre)
	return ZoneState{}
}

// premiereFrameDImageCle rend la frame de la premiere lecture d'image-cle (0 si elle precede
// l'origine des positions).
func premiereFrameDImageCle(reads []grammar.ManagedPropertyRead, origin uint64, doc ReplayDocument) int {
	first := reads[0].TimestampUS
	for _, r := range reads {
		first = min(first, r.TimestampUS)
	}
	if first < origin {
		return 0
	}
	return int((first - origin) / (uint64(doc.FrameIntervalMS) * 1000)) //nolint:gosec // pas positif
}

// etatInitialCuisson decode UNE fois ce que l'assemblage lit (positions, fil des morts, index,
// origine d'horloge) et rend un assembleur du document pour une entree de zones donnee — calque du
// drapeau exclu (cf. `zone_state_p2b_temoin_test.go`).
func etatInitialCuisson(t *testing.T, dir, short string, quant *profile.MapQuantEntry) func(ZoneInput,
	[]objectives.IdentifiedEvent) (ReplayDocument, uint64) {
	t.Helper()
	worldRange := quant.Range()
	scan := grammar.DefaultScanFilmOptions()
	scan.WorldRange = &worldRange
	pos, err := grammar.ScanFilmBipedPositions(dir, scan)
	if err != nil {
		t.Fatalf("positions illisibles (%s) : %v", dir, err)
	}
	deaths, err := grammar.ScanFilmDeaths(dir)
	if err != nil {
		t.Fatalf("fil des morts illisible (%s) : %v", dir, err)
	}
	base := Options{Deaths: deaths, MapQuant: quant}
	if idx, err := grammar.ScanFilmPlayerIndices(dir, rosterFromDeaths(deaths)); err == nil {
		base.PlayerIndices, _ = injectiveOrEmpty(idx)
	}
	if base.FilmClockOriginUS, err = grammar.ScanFilmClockOrigin(dir); err != nil {
		t.Logf("origine d'horloge illisible : %v", err)
	}
	origin := uint64(0)
	for i, p := range pos {
		if i == 0 || p.TimestampUS < origin {
			origin = p.TimestampUS
		}
	}
	return func(zone ZoneInput, caps []objectives.IdentifiedEvent) (ReplayDocument, uint64) {
		opt := base
		opt.Objectives, opt.Zone = caps, zone
		return BuildFromPositions(context.Background(), short, title.DefaultSlug, pos, nil, opt), origin
	}
}
