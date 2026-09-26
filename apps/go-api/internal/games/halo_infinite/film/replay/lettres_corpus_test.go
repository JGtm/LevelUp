package replay

// lettres_corpus_test.go — LES ENTREES GELEES du lot « lettres A/B/C des bases » : ce qu'on sait
// d'un film SANS ouvrir de base.
//
// AUCUNE DUCKDB N'EST OUVERTE, ET CE N'EST PAS UN CONFORT. Le pont slot statborg -> xuid exige
// les lignes de match, l'equipe exige le roster, et l'appariement geometrique exige la carte :
// trois choses qui vivent en base. Elles sont relues ici dans les exports VERSIONNES du registre
// du film (`oracle_lotA*.tsv`, `oracle_lotA*_participants.tsv`) — meme convention que la phase 2a
// du lot C-bis, qui gelait les memes lignes en Go. Le paquet `replay` n'ouvre aucune base, et un
// serveur tient de toute facon celle du depot en RW.
//
// Ce fichier vit a part de l'instrument (`lettres_ordre_research_test.go`) pour la meme raison
// que `zone_state_p2a_corpus_test.go` vit a part de `zone_state_p2a_test.go` : les ENTREES d'une
// mesure et la MESURE elle-meme sont deux sujets, et les separer garde chacun sous le seuil du
// depot.

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// lettresFilm est l'identite d'un film mesure, relue des exports geles du registre.
type lettresFilm struct {
	short, carte, mapID, variant string
}

// lettresIdentite relit la ligne de match GELEE du film : carte, identifiant de carte, variante.
func lettresIdentite(t *testing.T, short string) lettresFilm {
	t.Helper()
	for _, name := range []string{"oracle_lotA.tsv", "oracle_lotA_bis.tsv"} {
		if f, ok := lettresLitLigneMatch(t, filepath.Join(lettresRegistreDir(t), name), short); ok {
			return f
		}
	}
	t.Skipf("film %s absent des exports de match geles — identite inconnue, mesure impossible", short)
	return lettresFilm{}
}

// lettresLitLigneMatch cherche le film dans un export de matchs et rend son identite.
func lettresLitLigneMatch(t *testing.T, path, short string) (lettresFilm, bool) {
	t.Helper()
	blob, err := os.ReadFile(path)
	if err != nil {
		t.Logf("export de matchs absent (%s) : %v", path, err)
		return lettresFilm{}, false
	}
	cols := map[string]int{}
	for i, line := range strings.Split(string(blob), "\n") {
		f := strings.Split(strings.TrimRight(line, "\r"), "\t")
		if i == 0 {
			for j, name := range f {
				cols[name] = j
			}
			continue
		}
		if len(f) < len(cols) || !strings.HasPrefix(f[0], short) {
			continue
		}
		return lettresFilm{
			short:   short,
			carte:   strings.ToLower(f[cols["map_name"]]),
			mapID:   f[cols["map_id"]],
			variant: f[cols["game_variant_name"]],
		}, true
	}
	return lettresFilm{}, false
}

// lettresRegistreDir rend le repertoire des exports geles du registre du film.
func lettresRegistreDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRootForTest(t), ".ai", "V7.5", "replay2d", "registre_film")
}

// lettresRoster relit les lignes de match GELEES du film.
//
// LES DEUX EXPORTS SE RECOUVRENT, ET LE DOUBLON EST FATAL : `oracle_lotA_participants.tsv` et
// `oracle_lotA_bis_participants.tsv` portent tous deux les 12 matchs du lot A. Concatener les
// lignes donnait 16 joueurs pour une partie a 8, et `SlotIdentity` — qui apparie les slots du
// statborg aux lignes de match par leurs compteurs — n'identifiait alors PLUS AUCUNE capture
// (0/0/0 sur `7344d24f` et `696a9d7c`, la ou le meme film en rend 71 en phase 2a). La table est
// donc dedoublonnee par xuid, premier export lu gagnant.
func lettresRoster(t *testing.T, short string) []p2aPlayer {
	t.Helper()
	var out []p2aPlayer
	vus := map[string]bool{}
	for _, name := range []string{"oracle_lotA_participants.tsv", "oracle_lotA_bis_participants.tsv"} {
		out = append(out, lettresLitParticipants(t, filepath.Join(lettresRegistreDir(t), name),
			short, vus)...)
	}
	if len(out) == 0 {
		t.Skipf("film %s : aucune ligne de match gelee — le pont slot -> xuid est impossible", short)
	}
	return out
}

// lettresLitParticipants rend les lignes d'un export TSV pour un film, hors doublons.
func lettresLitParticipants(t *testing.T, path, short string, vus map[string]bool) []p2aPlayer {
	t.Helper()
	blob, err := os.ReadFile(path)
	if err != nil {
		t.Logf("export de participants absent (%s) : %v", path, err)
		return nil
	}
	cols := map[string]int{}
	var out []p2aPlayer
	for i, line := range strings.Split(string(blob), "\n") {
		f := strings.Split(strings.TrimRight(line, "\r"), "\t")
		if i == 0 {
			for j, name := range f {
				cols[name] = j
			}
			continue
		}
		if len(f) < len(cols) || !strings.HasPrefix(f[0], short) || vus[f[cols["xuid"]]] {
			continue
		}
		vus[f[cols["xuid"]]] = true
		out = append(out, p2aPlayer{
			XUID:    f[cols["xuid"]],
			Kills:   lettresAtoi(f[cols["kills"]]),
			Deaths:  lettresAtoi(f[cols["deaths"]]),
			Assists: lettresAtoi(f[cols["assists"]]),
			Team:    lettresAtoi(f[cols["team_id"]]),
		})
	}
	return out
}

// lettresAtoi lit un entier, 0 a defaut.
func lettresAtoi(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return n
}
