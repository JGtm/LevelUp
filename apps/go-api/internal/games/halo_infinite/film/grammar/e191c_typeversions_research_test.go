//go:build research

package grammar

// e191c_typeversions_research_test.go — LOT 1.9.1 bis, PAS 3 BIS : LA CLE EST DANS LE FILM.
//
// # LE FAIT QUI RETOURNE LE PAS 3
//
// Tranche par l utilisateur le 2026-09-16 (mecanique de jeu, il fait autorite) : « les films
// sont independants des builds ; ils sont enregistres a l instant T et jamais touches ensuite ;
// le film ne depend que de lui-meme pour expliquer au mode Theater comment le lire ».
//
// Consequence directe : l executable OUVERT (un build recent) LIT les films anciens. La
// grammaire `8/3` des films anciens EXISTE donc dans cet executable, et ce qui choisit entre
// `8/3` et `9/5` est une DONNEE ECRITE DANS LE FILM — pas le build. Le profil n est pas « par
// build » : il est « par version ecrite dans le film ». Aucun executable ancien n est necessaire.
//
// # LA CLE CANDIDATE
//
// La section 2 de `chunk_00` porte une TABLE PAR TYPE (`FilmIdentity.TypeVersions`, lot 1.5) :
// une version de serialisation par type, en clair, de cardinal 116 a 123 selon le film. Si la
// largeur MPP bascule avec la version d UN type, ce type est le discriminant.
//
// Cet instrument colle, type par type, les versions des sept bobines, et NOMME les index dont
// la version bascule exactement a la frontiere mesuree (`8/3` pour `a521164d`, `60ae07c4`,
// `11de8353`, `111fa685`, `e5adf7b2` ; `9/5` pour `bcb6d393`, `fb1a1a72`).
//
// LECTURE SEULE, sans garde d environnement (bobines versionnees).
//
//	go test ./internal/games/halo_infinite/film/filmdec/ -run '^TestE191cVersionsParType$' -v -count=1

import (
	"path/filepath"
	"testing"
)

// e191cLargeMPP : la largeur MPP mesuree de chaque bobine (D13). `false` = 8/3, `true` = 9/5.
var e191cLargeMPP = map[string]bool{
	"a521164d": false, "60ae07c4": false, "11de8353": false, "111fa685": false,
	"e5adf7b2": false, "bcb6d393": true, "fb1a1a72": true,
}

// TestE191cVersionsParType cherche, dans la table par type, l index qui bascule avec la largeur.
func TestE191cVersionsParType(t *testing.T) {
	t.Logf("######## PAS 3 BIS — LA TABLE PAR TYPE, ET L INDEX QUI BASCULE AVEC LA LARGEUR ########")
	vers := map[string][]uint32{}
	ordre := closureMiniFilms()
	for _, court := range ordre {
		dir := filepath.Join("..", "replay", "testdata", "minifilm_"+court)
		_, d0 := readChunk00(t, dir)
		id, err := ReadFilmIdentity(d0)
		if err != nil {
			t.Logf("  %-10s ECARTE : %v", court, err)
			continue
		}
		vers[court] = id.TypeVersions
		t.Logf("  %-10s build=%-12s types=%-4d MPP=%s", court, id.Build, len(id.TypeVersions),
			e191cLibelleMPP(e191cLargeMPP[court]))
	}
	e191cChercherDiscriminant(t, ordre, vers)
}

// e191cLibelleMPP nomme la largeur mesuree.
func e191cLibelleMPP(large bool) string {
	if large {
		return "9/5"
	}
	return "8/3"
}

// e191cChercherDiscriminant colle les index dont la version SEPARE exactement les deux groupes.
//
// Le critere est strict : toutes les bobines `8/3` portent la MEME version a cet index, toutes
// les `9/5` en portent une AUTRE, elle aussi commune. Un index qui varie a l interieur d un
// groupe ne discrimine pas.
func e191cChercherDiscriminant(t *testing.T, ordre []string, vers map[string][]uint32) {
	t.Helper()
	min := 1 << 30
	for _, v := range vers {
		if len(v) < min {
			min = len(v)
		}
	}
	if min == 1<<30 {
		t.Fatalf("aucune bobine ne porte de table par type")
	}
	t.Logf("")
	t.Logf("  ==== index dont la version SEPARE exactement les deux groupes (sur %d communs) ====", min)
	trouves := 0
	for i := 0; i < min; i++ {
		petit, grand := map[uint32]bool{}, map[uint32]bool{}
		for _, court := range ordre {
			v, ok := vers[court]
			if !ok {
				continue
			}
			if e191cLargeMPP[court] {
				grand[v[i]] = true
				continue
			}
			petit[v[i]] = true
		}
		if len(petit) != 1 || len(grand) != 1 {
			continue
		}
		var a, b uint32
		for k := range petit {
			a = k
		}
		for k := range grand {
			b = k
		}
		if a == b {
			continue
		}
		trouves++
		t.Logf("     type[%-3d] : %d sur les cinq films 8/3, %d sur les deux films 9/5", i, a, b)
	}
	t.Logf("  BILAN : %d index discriminants sur %d", trouves, min)
}

// TestE191cVersionsParTypeAlignees rejoue la recherche en alignant les tables PAR LA FIN, puis
// colle les tables brutes. Les types sont ajoutes au fil des versions du jeu : un alignement par
// le debut suppose qu ils le sont A LA FIN, un alignement par la fin suppose l inverse. Aucun
// des deux n est vrai a priori — d ou les deux mesures, et le vidage qui permet de trancher.
func TestE191cVersionsParTypeAlignees(t *testing.T) {
	ordre := closureMiniFilms()
	vers := map[string][]uint32{}
	for _, court := range ordre {
		dir := filepath.Join("..", "replay", "testdata", "minifilm_"+court)
		_, d0 := readChunk00(t, dir)
		id, err := ReadFilmIdentity(d0)
		if err == nil {
			vers[court] = id.TypeVersions
		}
	}
	min := 1 << 30
	for _, v := range vers {
		if len(v) < min {
			min = len(v)
		}
	}
	t.Logf("######## ALIGNEMENT PAR LA FIN (%d positions communes) ########", min)
	trouves := 0
	for k := 1; k <= min; k++ {
		petit, grand := map[uint32]bool{}, map[uint32]bool{}
		for _, court := range ordre {
			v, ok := vers[court]
			if !ok {
				continue
			}
			x := v[len(v)-k]
			if e191cLargeMPP[court] {
				grand[x] = true
				continue
			}
			petit[x] = true
		}
		if len(petit) != 1 || len(grand) != 1 {
			continue
		}
		var a, b uint32
		for x := range petit {
			a = x
		}
		for x := range grand {
			b = x
		}
		if a == b {
			continue
		}
		trouves++
		t.Logf("  fin-%-3d : %d sur les cinq films 8/3, %d sur les deux films 9/5", k, a, b)
	}
	t.Logf("  BILAN alignement par la fin : %d positions discriminantes", trouves)
	t.Logf("")
	t.Logf("######## LES TABLES BRUTES (30 premieres et 10 dernieres valeurs) ########")
	for _, court := range ordre {
		v, ok := vers[court]
		if !ok {
			continue
		}
		t.Logf("  %-10s (%d) tete=%v", court, len(v), v[:30])
		t.Logf("  %-10s      queue=%v", court, v[len(v)-10:])
	}
}
