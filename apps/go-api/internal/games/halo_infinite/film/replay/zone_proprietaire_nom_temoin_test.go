package replay

// zone_proprietaire_nom_temoin_test.go — LE TEMOIN SUR FILM REEL DU PROPRIETAIRE DESIGNE PAR LE NOM.
//
// Un Bastion court : trois jauges appariees, mais deux zones n'ont qu'une capture concordante, sous
// le seuil du vote. Le temoin assemble le calque des zones DEUX FOIS par le chemin de production
// (`BuildFromPositions`, calque du drapeau exclu, cf. `zone_etat_initial_temoin_test.go`) :
//
//	sans les lectures d'image-cle   aucun nom : le vote seul, une zone publiee sur trois ;
//	avec                            le nom de chaque jauge designe son proprietaire : les trois
//	                                zones sont publiees, le controle du proprietaire reste entier,
//	                                et la zone que le vote elisait garde le meme canal.
//
// SOUS GARDE D'ENVIRONNEMENT (`ZONE_FILM`), un film par processus, avant-plan :
//
//	$env:CGO_ENABLED=0
//	$env:ZONE_FILM="C:/Users/Guillaume/Downloads/Scripts/LevelUp/data/cache/film_chunks/114b0040"
//	go test -count=1 -run TestZoneProprietaireNomTemoin -v -timeout 30m ./internal/games/halo_infinite/film/replay/

import (
	"reflect"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
)

// proprietaireNommeAttendu : par rang de lettre (0 = A), le camp du PREMIER intervalle publie et
// celui du DERNIER, tels que le canal du bloc les porte ; le score les confirme (un camp ne marque
// qu'avec deux bases tenues). Seul `114b0040` porte une attente.
var proprietaireNommeAttendu = map[string]map[int][2]int{"114b0040": {
	0: {0, 1}, // A : prise et reprise quatre fois, la seule que le vote elisait
	1: {1, 1}, // B : neutre jusqu'a sa seule capture, tenue ensuite
	2: {1, 0}, // C : tenue au coup d'envoi, perdue a sa seule capture
}}

// TestZoneProprietaireNomTemoin assemble le calque des zones sans puis avec les noms d'image-cle.
func TestZoneProprietaireNomTemoin(t *testing.T) {
	dir := p2aRequireFilm(t)
	short, film := p2aFilmOf(t, dir)
	attendu, ok := proprietaireNommeAttendu[short]
	if !ok {
		t.Skipf("film %s sans proprietaire nomme attendu", short)
	}
	sc := p2bScan(t, dir)
	zone := ZoneInput{Scanned: true, Reads: sc.Reads, Zones: p2aZones(t, film.MapID, p2aRolesDuMode(film)...),
		Roles: p2bRoles(film), TeamByXUID: film.p2aTeams()}
	cuire := etatInitialCuisson(t, dir, short, p2aQuant(t, film.Carte))
	caps := p2aCaptures(p2aBobine(t, dir), film)
	avant, _ := cuire(zone, caps)
	zone.KeyReads = sc.KeyReads
	apres, _ := cuire(zone, caps)
	ca, cp := avant.Coverage.Zones, apres.Coverage.Zones
	t.Logf("FILM %s — avant : %d zones publiees, ownerUnpaired %d, controle %d/%d ; apres : %d zones, "+
		"ownerNamed %d, ownerVoteDisagreed %d, controle %d/%d", short, len(avant.ZoneStates),
		ca.OwnerUnpaired, ca.OwnerAgreed, ca.OwnerChecked, len(apres.ZoneStates), cp.OwnerNamed,
		cp.OwnerVoteDisagreed, cp.OwnerAgreed, cp.OwnerChecked)

	if len(avant.ZoneStates) >= cp.Catalog {
		t.Errorf("sans nom, %d zones publiees : le temoin ne montre plus le defaut du vote", len(avant.ZoneStates))
	}
	if len(apres.ZoneStates) != cp.Catalog || cp.OwnerNamed != cp.Catalog || cp.OwnerUnpaired != 0 {
		t.Fatalf("avec les noms : %d zones publiees, %d nommees, %d sans proprietaire sur %d au catalogue",
			len(apres.ZoneStates), cp.OwnerNamed, cp.OwnerUnpaired, cp.Catalog)
	}
	if cp.OwnerVoteDisagreed != 0 || cp.OwnerAgreed != cp.OwnerChecked || cp.OwnerChecked <= ca.OwnerChecked {
		t.Errorf("controle : vote discordant %d, proprietaire %d/%d (avant %d/%d)", cp.OwnerVoteDisagreed,
			cp.OwnerAgreed, cp.OwnerChecked, ca.OwnerAgreed, ca.OwnerChecked)
	}
	// LE POUSSEUR AUSSI SE LIT PAR LE NOM : chaque zone a le sien, l election ne le contredit
	// nulle part, et aucune rampe n est plus deduite de son issue.
	if cp.CapturerNamed != cp.Catalog || cp.CapturerElectionDisagreed != 0 {
		t.Errorf("pousseur : %d zone(s) nommee(s) sur %d, %d discordance(s) avec l election",
			cp.CapturerNamed, cp.Catalog, cp.CapturerElectionDisagreed)
	}
	for _, f := range apres.Coverage.Fallbacks {
		if f.Name == string(fallback.NomZoneCampDeCaptureDeduitDeLIssue) && f.Hits > 0 {
			t.Errorf("%d rampe(s) encore deduite(s) de leur issue", f.Hits)
		}
	}
	for _, st := range apres.ZoneStates {
		verifierProprietaireNomme(t, st, attendu)
		for _, av := range avant.ZoneStates {
			if av.ZoneRef == st.ZoneRef && (len(av.Spans) > len(st.Spans) ||
				!reflect.DeepEqual(st.Spans[len(st.Spans)-len(av.Spans):], av.Spans)) {
				t.Errorf("zone %d, elue par le vote : ses intervalles d avant ne se retrouvent pas", st.ZoneRef)
			}
		}
	}
}

// verifierProprietaireNomme confronte le premier et le dernier camp publies d'une zone a l'attente.
func verifierProprietaireNomme(t *testing.T, st ZoneState, attendu map[int][2]int) {
	t.Helper()
	if st.LetterRank == nil {
		t.Fatalf("zone %d sans lettre", st.ZoneRef)
	}
	camps, ok := attendu[*st.LetterRank]
	if !ok {
		t.Fatalf("lettre %d hors attente", *st.LetterRank)
	}
	premier, dernier := ownerOf(st.Spans[0]), ownerOf(st.Spans[len(st.Spans)-1])
	t.Logf("  lettre %d : %d intervalles, premier t0=%d camp %d, dernier t0=%d camp %d", *st.LetterRank,
		len(st.Spans), st.Spans[0].T0, premier, st.Spans[len(st.Spans)-1].T0, dernier)
	if premier != camps[0] || dernier != camps[1] {
		t.Errorf("lettre %d : camps %d -> %d, attendu %d -> %d", *st.LetterRank, premier, dernier,
			camps[0], camps[1])
	}
}
