package killsource

// version_evenements_research_test.go — INSTRUMENT H.2 : LE CALQUE « EVENEMENTS DU PIED DE FILM »
// (kill feed, medailles, evenements de mode) MESURE PAR VERSION MAJEURE DE FILM.
//
// LA QUESTION DU LOT H. Depuis le lot G, la version majeure du film (u32 LE en tete de
// `chunk_00`, cf. `filmdec.FilmMajorVersion`) commande le decoupage du gamertag du bloc
// d'evenement : en tete pour les versions <= 38 et >= 41, a l'octet 12 pour les versions 39-40.
// Le lot G l'a prouve SUR LES KILLS. Il restait a savoir si les AUTRES natures d'evenement du
// meme flux — medailles, morts, evenements de mode — divergent de la meme maniere, et si le
// volume brut d'evenements (donc la grammaire du bloc, pas seulement le champ de nom) change
// avec la version.
//
// CE QUE L'INSTRUMENT FAIT. Pour chaque film demande, il lit la version dans l'en-tete du
// registre, puis analyse le chunk de temps forts SOUS LES DEUX DECOUPAGES (en tete et decale)
// et publie, pour chacun : le nombre d'evenements reconnus par nature (kill / mort / medaille /
// mode), les `type_hint` distincts, les types de medaille distincts, les XUID distincts, les
// gamertags distincts non vides et les blocs sans gamertag. Le decoupage que la PRODUCTION
// choisirait (celui que la version lue impose) est marque.
//
// POURQUOI LES DEUX DECOUPAGES. Le compte d'evenements RECONNUS ne depend pas du champ de nom
// (le `type_hint` est lu a un offset fixe) : si les deux decoupages rendent le meme nombre
// d'evenements et les memes natures, alors la divergence de version porte UNIQUEMENT sur
// l'implantation du gamertag — ce qui est deja corrige. Si le compte change, c'est la grammaire
// du bloc elle-meme qui bouge, et le profil de dechiffrage devra brancher plus large.
//
// IL N'ASSERTE RIEN : c'est un instrument de mesure, pas un garde-rail. Sans films ni variables
// d'environnement, il se saute proprement.
//
// USAGE (lecture seule, aucune ecriture, aucune base) :
//
//	HVER_ROOT=<parc>/data/cache/film_chunks \
//	HVER_IDS=bcb6d393,000d5950,60ae07c4 \
//	  go test ./internal/games/halo_infinite/film/killsource/ -run TestVersionEvenements -v -timeout 3600s

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmsource"
	"levelup/go-api/internal/analysis/weaponv3"
)

const (
	hverRootEnv = "HVER_ROOT"
	hverIDsEnv  = "HVER_IDS"
)

// hverStats : le releve d'UN decoupage sur UN film.
type hverStats struct {
	chunk      int // indice du chunk retenu (celui qui porte le plus d'evenements)
	events     int
	kills      int
	morts      int
	medailles  int
	modes      int
	typeHints  []int
	medalTypes []int
	xuids      int
	gamertags  int
	vides      int
}

// TestVersionEvenements — le banc H.2 du calque « evenements du pied de film ».
func TestVersionEvenements(t *testing.T) {
	root := os.Getenv(hverRootEnv)
	ids := strings.Split(os.Getenv(hverIDsEnv), ",")
	if root == "" || len(ids) == 0 || ids[0] == "" {
		t.Skipf("instrument de mesure : %s et %s requis", hverRootEnv, hverIDsEnv)
	}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		src, err := filmsource.LoadDir(filepath.Join(root, id), nil)
		if err != nil {
			t.Errorf("%s : film illisible : %v", id, err)
			continue
		}
		f, err := loadFilm(src)
		if err != nil {
			t.Errorf("%s : chargement : %v", id, err)
			continue
		}
		t.Logf("%-8s version=%d (lue=%v) chunks=%d", id, f.majorVersion, f.versionLue, len(f.chunks))
		for _, decoupage := range []int{versionGamertagEnTeteTest, versionGamertagDecaleTest} {
			s := hverMesure(f, decoupage)
			marque := " "
			if hverMemeDecoupage(decoupage, f.majorVersion) {
				marque = "*" // le decoupage que la production choisit pour ce film
			}
			t.Logf("%-8s%s decoupage=%2d chunk=%2d events=%4d kills=%4d morts=%4d medailles=%4d modes=%3d "+
				"typeHints=%v medailles_types=%v xuids=%2d gamertags=%2d vides=%4d",
				id, marque, decoupage, s.chunk, s.events, s.kills, s.morts, s.medailles, s.modes,
				s.typeHints, s.medalTypes, s.xuids, s.gamertags, s.vides)
		}
	}
}

// hverMemeDecoupage dit si `decoupage` est celui que la production applique a un film de
// version `version`. La regle est celle d'`analysis.ParseHighlightEvents` : gamertag decale
// sur 39 et 40, en tete partout ailleurs (y compris version inconnue).
func hverMemeDecoupage(decoupage, version int) bool {
	decale := version == 39 || version == 40
	if decale {
		return decoupage == versionGamertagDecaleTest
	}
	return decoupage == versionGamertagEnTeteTest
}

// hverMesure analyse tous les chunks du film sous un decoupage donne et rend le releve du
// chunk le plus fourni — c'est le chunk de temps forts, les autres ne rendent que du bruit.
func hverMesure(f *film, decoupage int) hverStats {
	var best hverStats
	best.chunk = -1
	for ch := range f.chunks {
		evs, err := analysis.ParseHighlightEvents(f.chunks[ch], decoupage)
		if err != nil || len(evs) == 0 {
			continue
		}
		s := hverCompte(evs)
		s.chunk = ch
		if s.events > best.events {
			best = s
		}
	}
	return best
}

// hverCompte ventile une liste d'evenements par nature et rend les cardinaux distincts.
func hverCompte(evs []analysis.HighlightEvent) hverStats {
	var s hverStats
	s.events = len(evs)
	hints, medals := map[int]bool{}, map[int]bool{}
	xu, gt := map[uint64]bool{}, map[string]bool{}
	for _, e := range evs {
		switch e.EventType {
		case analysis.EventTypeKill:
			s.kills++
		case analysis.EventTypeDeath:
			s.morts++
		case analysis.EventTypeMedal:
			s.medailles++
			medals[e.MedalType] = true
		case analysis.EventTypeMode:
			s.modes++
		}
		hints[e.TypeHint] = true
		xu[e.XUID] = true
		if e.Gamertag == "" {
			s.vides++
			continue
		}
		gt[e.Gamertag] = true
	}
	s.typeHints = hverTriees(hints)
	s.medalTypes = hverTriees(medals)
	s.xuids, s.gamertags = len(xu), len(gt)
	return s
}

// hverTriees rend les cles d'un ensemble d'entiers, triees — un journal reproductible.
func hverTriees(m map[int]bool) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

// TestVersionIndexJoueur — INSTRUMENT H.2 : LE CALQUE « VIES ET IDENTITE », descendu au champ.
//
// CE QUE LA CUISSON A MONTRE (lot H). Sur cinq films du cache — les trois de version 31 et deux
// des trois de version 33, exactement les cinq dont `chunk_00` ne porte PAS de section
// d'identification (pas de chaine `HI_1_x_y`) — l'artefact sort sans AUCUNE identite :
// `identity.players` vide, zero piste porteuse de xuid, `coverage.bridge.indexReadings` a zero,
// roster vide, et donc zero tir publie (tous « sans slot »). Le meme film rend pourtant un kill
// feed complet (24 a 27 gamertags distincts).
//
// OU CELA SE JOUE. Le pont identite du rejeu part de `replay.ScanPlayerIndices`, qui cherche le
// XUID de chaque joueur du roster comme un motif de 64 bits (petit-boutiste) ALIGNE AU BIT dans
// chaque chunk de replication, puis lit les CINQ BITS qui le precedent
// (`weaponv3.ResolveXuidToPI`, `PIBits = 5`). Si aucun chunk ne rend d'index, la table n'est pas
// publiee et tout le calque s'eteint.
//
// CE QUE CET INSTRUMENT TRANCHE. Il prend le roster DANS LE FILM LUI-MEME (les XUID distincts du
// pied de film, qui se lisent sur toutes les versions) et compte, chunk de replication par chunk
// de replication, combien de ces XUID s'y trouvent comme motif de 64 bits. Deux issues
// exclusives :
//
//	le motif est INTROUVABLE  -> le film n'ecrit pas le XUID en clair dans la trame a ce build ;
//	                             le pont d'identite doit passer par une autre voie ;
//	le motif est TROUVE       -> la cause est ailleurs (roster d'entree vide, porte d'injectivite).
//
// Il ne consulte ni carte, ni catalogue, ni base : la recherche de motif est un balayage
// d'octets pur. Il n'asserte rien et se saute sans films.
//
// USAGE :
//
//	HVER_ROOT=<parc>/data/cache/film_chunks HVER_IDS=13b00e35,a349fea8,000d5950 \
//	  go test ./internal/games/halo_infinite/film/killsource/ -run TestVersionIndexJoueur -v -timeout 3600s
func TestVersionIndexJoueur(t *testing.T) {
	root := os.Getenv(hverRootEnv)
	ids := strings.Split(os.Getenv(hverIDsEnv), ",")
	if root == "" || len(ids) == 0 || ids[0] == "" {
		t.Skipf("instrument de mesure : %s et %s requis", hverRootEnv, hverIDsEnv)
	}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		src, err := filmsource.LoadDir(filepath.Join(root, id), nil)
		if err != nil {
			t.Errorf("%s : film illisible : %v", id, err)
			continue
		}
		f, err := loadFilm(src)
		if err != nil {
			t.Errorf("%s : chargement : %v", id, err)
			continue
		}
		roster := hverRosterDuFilm(f)
		chunksLus, trouves, indices := hverCherche(f, roster)
		t.Logf("%-8s version=%2d roster_pied_de_film=%2d chunks_replication=%2d chunks_avec_motif=%2d "+
			"xuids_trouves=%2d indices=%v",
			id, f.majorVersion, len(roster), chunksLus, trouves, len(indices), hverTriees(indices))
	}
}

// hverRosterDuFilm rend les XUID distincts du pied de film, sous le decoupage que la version
// impose. Le pied de film se lit sur toutes les versions mesurees : c'est le seul roster
// disponible sans base.
func hverRosterDuFilm(f *film) []uint64 {
	decoupage := versionGamertagEnTeteTest
	if f.majorVersion == 39 || f.majorVersion == 40 {
		decoupage = versionGamertagDecaleTest
	}
	var best []analysis.HighlightEvent
	for ch := range f.chunks {
		evs, err := analysis.ParseHighlightEvents(f.chunks[ch], decoupage)
		if err != nil || len(evs) <= len(best) {
			continue
		}
		best = evs
	}
	vus := map[uint64]bool{}
	out := make([]uint64, 0, len(best))
	for _, e := range best {
		if e.XUID == 0 || vus[e.XUID] {
			continue
		}
		vus[e.XUID] = true
		out = append(out, e.XUID)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// hverCherche compte les chunks de replication qui portent au moins un XUID du roster, les XUID
// distincts trouves, et les valeurs d'index de joueur lues devant eux. Meme decoupage de chunks
// que `replay.ScanPlayerIndices` : le premier est le registre, le dernier le pied de film.
func hverCherche(f *film, roster []uint64) (chunks, avecMotif int, indices map[int]bool) {
	indices = map[int]bool{}
	if len(f.chunks) < 2 || len(roster) == 0 {
		return 0, 0, indices
	}
	distincts := map[uint64]bool{}
	for c := 1; c < len(f.chunks)-1; c++ {
		chunks++
		got := weaponv3.ResolveXuidToPI(roster, f.chunks[c])
		if len(got) == 0 {
			continue
		}
		avecMotif++
		for x, pi := range got {
			distincts[x] = true
			indices[pi] = true
		}
	}
	_ = distincts
	return chunks, avecMotif, indices
}
