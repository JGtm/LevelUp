package replay

// golden_entites_test.go — LES OCCUPANTS `ti=9` DES HUIT BUILDS, FIGES A COTE DE LEURS ENTREES
// (revue adverse du lot M2, constat M2-R3, 2026-09-24).
//
// # LE TROU QU'IL FERME
//
// Les entites `ti=9` (lot M2.1) voyagent dans le COMPLEMENT de la section 1 du fichier de faits,
// pas dans le blob des entrees : les huit `inputs_<short8>.bin.gz` sont restes valides a l'octet,
// et SANS entites. Tous les goldens d'assemblage, toutes les fixtures de contrat publiees au web
// sortaient donc par le repli `presences = vies` — le chemin `film`, celui qui servira le parc
// re-decode entier, n'etait exerce que par un temoin SYNTHETIQUE.
//
// Chaque build porte desormais `entites_<short8>.bin.gz` : le balayage `ti=9` de SON film, tel que
// la production l'a persiste, encode par le codec de production ([encodeEntitesDesJoueurs]). Il se
// pose sur les entrees relues ([chargerGoldenBuild], [loadGoldenInputs]) : goldens et fixtures de
// contrat decrivent le chemin `film`. Quelques kilo-octets par build.
//
// # LA PROPRIETE QU'IL PERMET, CONTRE UNE SOURCE INDEPENDANTE
//
// [TestRegleDesPlacesContreLesEntitesDesBuilds] confronte les presences PUBLIEES aux entites
// BRUTES, sans passer par la liaison ni par la pose des places : un humain seul sur son index
// n'est jamais affiche au-dela de la premiere image-cle porteuse qui ne porte plus aucune entite
// de son index (sauf une vie qui le montre), ni avant la premiere qui en porte une ; son equipe est
// le designateur de ses entites ; et a chaque frame une equipe affiche au plus la taille d'equipe
// du mode, ecrite ici et non estimee.
//
// # REGENERATION (jamais d'edition a la main)
//
// Depuis les faits persistes par la production (un `<short8>.filmfacts.bin` du schema de faits 4
// par build, un film a la fois, sous le verrou de la voie film) :
//
//	REPLAY_ENTITES_FAITS=<dossier des .filmfacts.bin> \
//	  go test ./internal/games/halo_infinite/film/replay/ -run 'TestGoldenEntitesRegenerate/<short8>' -update
//
// Elle echoue TOUJOURS en nommant ce qu'elle a ecrit (lecon C1 du lot 0.A) ; on relance sans les
// drapeaux pour verifier.

import (
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// entitesFaitsEnv : le dossier des fichiers de faits dont la regeneration relit les entites.
const entitesFaitsEnv = "REPLAY_ENTITES_FAITS"

// entitesPath : le fichier des entites d'un build.
func entitesPath(short8 string) string {
	return filepath.Join(goldenDir, "entites_"+short8+".bin.gz")
}

// chargerEntitesDuBuild relit le balayage `ti=9` fige d'un build. ABSENT = ECHEC : les huit builds
// le portent, et un build qui le perdrait retomberait en silence sur le repli des vies.
func chargerEntitesDuBuild(t *testing.T, short8 string) grammar.PlayerEntityScan {
	t.Helper()
	raw, err := os.ReadFile(entitesPath(short8)) //nolint:gosec // chemin construit depuis la table des builds
	if err != nil {
		t.Fatalf("entites du build %s absentes (%v) — regenerer avec %s=<faits> -run "+
			"TestGoldenEntitesRegenerate -update", short8, err, entitesFaitsEnv)
	}
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("entites %s : gzip illisible : %v", short8, err)
	}
	defer func() { _ = zr.Close() }()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(zr); err != nil {
		t.Fatalf("entites %s : decompression : %v", short8, err)
	}
	r := &greader{b: buf.Bytes()}
	scan := decodeEntitesDesJoueurs(r)
	if r.err != nil || r.off != len(r.b) || !scan.Scanned {
		t.Fatalf("entites %s : relecture invalide (err %v, %d/%d octets lus, balaye %v)", short8,
			r.err, r.off, len(r.b), scan.Scanned)
	}
	return scan
}

// TestGoldenEntitesRegenerate : LA SEULE PORTE D'ECRITURE des entites figees (cf. l'en-tete).
func TestGoldenEntitesRegenerate(t *testing.T) {
	dir := os.Getenv(entitesFaitsEnv)
	switch {
	case !*updateGolden:
		t.Skip("regeneration des entites des builds : passer -update (et " + entitesFaitsEnv + ")")
	case dir == "":
		t.Skip("regeneration des entites des builds : " + entitesFaitsEnv + " non defini")
	}
	for _, b := range goldenBuilds() {
		t.Run(b.Short8, func(t *testing.T) {
			entry, err := b.mapQuant()
			if err != nil {
				t.Fatalf("carte %q : %v", b.Map, err)
			}
			blob, err := os.ReadFile(filepath.Join(dir, b.Short8+".filmfacts.bin")) //nolint:gosec // dossier donne par l'operateur
			if err != nil {
				t.Fatalf("faits du build %s : %v", b.Short8, err)
			}
			f, err := DecodeFilmFactsFile(blob, entry)
			if err != nil {
				t.Fatalf("faits du build %s : %v", b.Short8, err)
			}
			if !f.Facts.PlayerEntities.Scanned {
				t.Fatalf("faits du build %s : aucune entite balayee (schema de faits anterieur au 4 ?)", b.Short8)
			}
			w := &gwriter{}
			encodeEntitesDesJoueurs(w, f.Facts.PlayerEntities)
			var out bytes.Buffer
			zw := gzip.NewWriter(&out)
			if _, err := zw.Write(w.b); err != nil {
				t.Fatalf("gzip : %v", err)
			}
			if err := zw.Close(); err != nil {
				t.Fatalf("gzip : %v", err)
			}
			if err := os.WriteFile(entitesPath(b.Short8), out.Bytes(), 0o600); err != nil {
				t.Fatalf("ecriture %s : %v", entitesPath(b.Short8), err)
			}
			t.Fatalf("REECRIT %s (%d entites, %d images-cles porteuses, %d octets) — relancer sans "+
				"-update pour verifier", entitesPath(b.Short8), len(f.Facts.PlayerEntities.Entities),
				len(f.Facts.PlayerEntities.KeyframesUS), out.Len())
		})
	}
}

// tailleDEquipeDesBuilds : la taille d'equipe du MODE de chaque build, ecrite ici (la meme table
// que `TAILLE_D_EQUIPE` du contrat web) — une source que la cuisson n'estime pas.
var tailleDEquipeDesBuilds = map[string]int{
	goldenFilm: 4, "60ae07c4": 4, "bcb6d393": 4, "fb1a1a72": 4,
	"a521164d": 12, "11de8353": 12, "111fa685": 12, "e5adf7b2": 12,
}

// TestRegleDesPlacesContreLesEntitesDesBuilds : cf. l'en-tete.
func TestRegleDesPlacesContreLesEntitesDesBuilds(t *testing.T) {
	for _, b := range goldenBuilds() {
		t.Run(b.Build+"/"+b.Short8, func(t *testing.T) {
			g, entry := chargerGoldenBuild(t, b)
			doc := assemblerGoldenBuild(t, b, g, entry)
			if doc.Coverage.Seats == nil || doc.Coverage.Seats.Presences != PresencesDuFilm {
				t.Fatalf("presences %+v : le build doit sortir par le chemin `film`", doc.Coverage.Seats)
			}
			h := horlogeDuDocument(g, doc)
			controlerLesHumainsContreLeursEntites(t, doc, g.PlayerEntities, h)
			controlerLaTailleDesEquipes(t, doc, tailleDEquipeDesBuilds[b.Short8])
		})
	}
}

// horlogeDuDocument rend la grille d'un document assemble depuis des positions : origine au plus
// petit horodatage (cf. `assemblage.ouvrir`), pas de la frame.
func horlogeDuDocument(g *FilmFacts, doc ReplayDocument) replayClock {
	origin := g.Positions[0].TimestampUS
	for _, p := range g.Positions {
		origin = min(origin, p.TimestampUS)
	}
	return replayClock{origin: origin, step: uint64(doc.FrameIntervalMS) * 1000, frames: doc.FrameCount}
}

// controlerLesHumainsContreLeursEntites : pour chaque humain SEUL sur son index, ses presences
// publiees contre les entites brutes de cet index (cf. l'en-tete).
func controlerLesHumainsContreLeursEntites(t *testing.T, doc ReplayDocument, scan grammar.PlayerEntityScan,
	h replayClock) {
	t.Helper()
	parIndex := map[int]int{}
	for _, e := range doc.Roster {
		parIndex[e.FilmIndex]++
	}
	vies := viesParIdentite(doc.Tracks)
	for _, e := range doc.Roster {
		if e.Bot || parIndex[e.FilmIndex] != 1 || len(e.Presence) == 0 {
			continue
		}
		var ents []grammar.PlayerEntity
		for _, x := range scan.Entities {
			if x.Index == e.FilmIndex && !x.Unstable {
				ents = append(ents, x)
			}
		}
		if len(ents) == 0 {
			continue
		}
		premier, dernier := ents[0].FirstKF, ents[0].LastKF
		for _, x := range ents {
			premier, dernier = min(premier, x.FirstKF), max(dernier, x.LastKF)
			if e.Team == nil || *e.Team != x.Team {
				t.Errorf("%s (index %d) : equipe %v, son entite dit %d", e.Name, e.FilmIndex, deref(e.Team), x.Team)
			}
		}
		debutVie, finVie := h.frames, -1
		for _, v := range vies[cleDeRoster(e)] {
			debutVie, finVie = min(debutVie, v[0]), max(finVie, v[1])
		}
		fin := e.Presence[len(e.Presence)-1]
		affiche := fin.To
		if fin.ToMax != nil {
			affiche = *fin.ToMax
		}
		if us, ok := scan.KeyframeUS(dernier + 1); ok {
			if borne := max(frameBrute(h, us)-1, finVie); affiche > borne {
				t.Errorf("%s (index %d) affiche jusqu'a %d : son index n'a plus d'entite a l'image-cle "+
					"f%d et sa derniere vie finit a %d — un parti affiche", e.Name, e.FilmIndex, affiche,
					frameBrute(h, us), finVie)
			}
		}
		if us, ok := scan.KeyframeUS(premier); ok && premier > 0 {
			if borne := min(frameBrute(h, us), debutVie); e.Presence[0].From < borne {
				t.Errorf("%s (index %d) affiche des %d : sa premiere entite est a f%d, sa premiere vie a %d",
					e.Name, e.FilmIndex, e.Presence[0].From, frameBrute(h, us), debutVie)
			}
		}
	}
}

// controlerLaTailleDesEquipes : a chaque frame, une equipe affiche au plus `taille` occupants, et
// sa colonne a au plus `taille` places.
func controlerLaTailleDesEquipes(t *testing.T, doc ReplayDocument, taille int) {
	t.Helper()
	if taille == 0 {
		t.Fatal("taille d'equipe du mode non declaree dans tailleDEquipeDesBuilds")
	}
	places := map[int]map[int]bool{}
	bornes := map[int][]borneDAffichage{}
	for _, e := range doc.Roster {
		if e.Team == nil || len(e.Presence) == 0 {
			continue
		}
		if places[*e.Team] == nil {
			places[*e.Team] = map[int]bool{}
		}
		places[*e.Team][e.Seat] = true
		for _, p := range e.Presence {
			fin := p.To
			if p.ToMax != nil {
				fin = *p.ToMax
			}
			bornes[*e.Team] = append(bornes[*e.Team], borneDAffichage{p.From, +1}, borneDAffichage{fin + 1, -1})
		}
	}
	equipes := make([]int, 0, len(places))
	for eq := range places {
		equipes = append(equipes, eq)
	}
	sort.Ints(equipes)
	for _, eq := range equipes {
		if n := len(places[eq]); n > taille {
			t.Errorf("equipe %d : %d places pour %d joueurs par equipe", eq, n, taille)
		}
		if maxi, au := balayerLesBornes(bornes[eq], taille); au > 0 {
			t.Errorf("equipe %d : jusqu'a %d occupants affiches, %d frames au-dela de %d", eq, maxi, au, taille)
		}
	}
}
