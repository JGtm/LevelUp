package filmdec

// version_grenade_tags_research_test.go — INSTRUMENT H.2 : LE CALQUE « LANCERS DE GRENADE »,
// DESCENDU JUSQU'AU CHAMP QUI DIVERGE.
//
// CE QUE LA MESURE DE COUVERTURE A MONTRE (lot H, passe de cuisson du 2026-09-13). Sur les
// artefacts cuits en racine jetable, `coverage.grenades.available` vaut ZERO sur tous les films
// des builds `HI_1_8_0`, `HI_1_10_0` et `HI_1_11_0` (versions majeures 33, 37, 39 et une partie
// des 40) et reste NON NUL sur `HI_1_12_0` et `HI_1_13_0` — y compris sur des paires de films
// de la MEME CARTE (Command : 0 lancer en 39 contre 292 en 40 ; Live Fire : 0 en 37 contre 102
// en 41). La carte est donc hors de cause.
//
// POURQUOI CE BALAYAGE-LA EST LE BON ENDROIT OU DESCENDRE. `ScanGrenadeThrows` est un balayage
// d'octets PUR : il ne consulte ni bornes de carte, ni catalogue, ni registre. Il cherche un
// marqueur de 24 bits (`grenadeMarker`) dans les paquets delta, lit l'identifiant de 32 bits
// qui suit et le compare a une LISTE BLANCHE de quatre valeurs — les identifiants globaux de
// tag du groupe `proj` des quatre grenades, decales d'un bit a gauche (cf. l'en-tete de
// `grenade_events.go`). Deux causes seulement peuvent rendre zero :
//
//	(a) le MARQUEUR n'est pas trouve — la grammaire du record a bouge ;
//	(b) le marqueur est trouve mais l'IDENTIFIANT n'est pas dans la liste blanche — les
//	    identifiants de tag ont change de build, et la liste blanche est datee d'un seul.
//
// L'instrument separe (a) de (b) : il compte les marqueurs, puis histogramme les identifiants
// de 32 bits qui les suivent, avec le rang connu quand il y en a un. Il publie aussi le champ
// d'index joueur (5 bits a +103) des candidats les plus frequents, pour dire si la STRUCTURE
// tient (index dans 0..31, plausible) ou non.
//
// IL N'ASSERTE RIEN, et il ne modifie aucun code de production : c'est une mesure.
// Sans films ni variables d'environnement, il se saute proprement.
//
// USAGE (lecture seule, aucune ecriture, aucune base) :
//
//	HGREN_ROOT=<parc>/data/cache/film_chunks \
//	HGREN_IDS=111fa685,b81f6415,60ae07c4,0797ce72 \
//	  go test ./internal/games/halo_infinite/film/filmdec -run TestVersionGrenadeTags -v -timeout 3600s
//
// Reglage : HGREN_TOP (nombre d'identifiants detailles par film, defaut 8).

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
)

const (
	hgrenRootEnv = "HGREN_ROOT"
	hgrenIDsEnv  = "HGREN_IDS"
	hgrenTopEnv  = "HGREN_TOP"
)

// hgrenCand : un identifiant de 32 bits observe derriere le marqueur, et ce qu'on en sait.
type hgrenCand struct {
	id       uint32
	n        int
	rank     int  // rang si l'identifiant est dans la liste blanche
	connu    bool // appartenance a la liste blanche
	idxMin   int  // index joueur (5 bits a +103) minimum observe
	idxMax   int  // maximum observe
	idxHorsN int  // occurrences d'un index >= 32 (impossible : le champ fait 5 bits)
}

// TestVersionGrenadeTags — le banc H.2 du calque « lancers de grenade ».
func TestVersionGrenadeTags(t *testing.T) {
	root := os.Getenv(hgrenRootEnv)
	ids := strings.Split(os.Getenv(hgrenIDsEnv), ",")
	if root == "" || len(ids) == 0 || ids[0] == "" {
		t.Skipf("instrument de mesure : %s et %s requis", hgrenRootEnv, hgrenIDsEnv)
	}
	top := 8
	if v, err := strconv.Atoi(os.Getenv(hgrenTopEnv)); err == nil && v > 0 {
		top = v
	}
	release := LockProcessDecode()
	defer release()
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		film, err := filmsource.LoadDir(filepath.Join(root, id), nil)
		if err != nil {
			t.Errorf("%s : film illisible : %v", id, err)
			continue
		}
		version := 0
		if v, lue := FilmMajorVersion(film); lue {
			version = v
		}
		marqueurs, cands := hgrenBalayage(film)
		reconnus := 0
		for _, c := range cands {
			if c.connu {
				reconnus += c.n
			}
		}
		t.Logf("%-8s version=%2d marqueurs=%5d reconnus=%4d identifiants_distincts=%d",
			id, version, marqueurs, reconnus, len(cands))
		for i, c := range cands {
			if i >= top {
				break
			}
			etiquette := "inconnu"
			if c.connu {
				etiquette = fmt.Sprintf("rang=%d", c.rank)
			}
			t.Logf("    0x%08X n=%5d %-8s index[%d..%d] hors5bits=%d",
				c.id, c.n, etiquette, c.idxMin, c.idxMax, c.idxHorsN)
		}
	}
}

// hgrenBalayage rend le nombre de marqueurs trouves et les identifiants observes derriere eux,
// tries par frequence decroissante. Meme parcours que `ScanGrenadeThrows`, SANS la liste
// blanche : c'est tout l'interet.
func hgrenBalayage(film *filmsource.Film) (int, []hgrenCand) {
	compte := map[uint32]*hgrenCand{}
	marqueurs := 0
	for _, c := range FilmChunkNumbers(film) {
		chunk, pks, ok := FilmChunkAt(film, c)
		if !ok {
			continue
		}
		for _, p := range pks {
			if p.Type != PacketTypeDelta {
				continue
			}
			marqueurs += hgrenPayload(p.Payload(chunk), compte)
		}
	}
	out := make([]hgrenCand, 0, len(compte))
	for _, v := range compte {
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].n != out[j].n {
			return out[i].n > out[j].n
		}
		return out[i].id < out[j].id
	})
	return marqueurs, out
}

// hgrenPayload balaye UN payload de paquet delta et rend le nombre de marqueurs vus.
func hgrenPayload(pay []byte, compte map[uint32]*hgrenCand) int {
	limit := len(pay)*8 - grenadeThrowBits
	vus := 0
	for bp := 0; bp <= limit; bp++ {
		if PeekBits(pay, bp, 24) != grenadeMarker {
			continue
		}
		vus++
		id := uint32(PeekBits(pay, bp+24, 32))
		idx := int(PeekBits(pay, bp+24+32+47, 5))
		c := compte[id]
		if c == nil {
			rang, connu := GrenadeRankOf(id)
			c = &hgrenCand{id: id, rank: rang, connu: connu, idxMin: idx, idxMax: idx}
			compte[id] = c
		}
		c.n++
		if idx < c.idxMin {
			c.idxMin = idx
		}
		if idx > c.idxMax {
			c.idxMax = idx
		}
		if idx >= 32 {
			c.idxHorsN++
		}
	}
	return vus
}
