package killsource

// mot_de_poignee_test.go — LE GARDE-RAIL DE LA SEULE VALEUR QUE LA CALIBRATION DECIDE ENCORE.
//
// CE QU IL INTERDIT, ET POURQUOI IL EXISTE. Le 2026-09-17, `replay-equiv` a montre que
// `abilityImpulses` bougeait sur 14 films sur 20, avec QUATRE lectures publiees perdues. La
// cause, mesuree sur deux films : le balayage qui DECIDE `Traversal.IndexW` scorait ses
// candidats sous une largeur d axe UNIFORME que la production avait cesse de lire, et son
// critere etait de toute facon AVEUGLE a cette grandeur (272/272/272 sur `a521164d`, 61/61/61
// sur `64e8adfa`). `out[0]` sortait alors d un `sort.Slice` instable : la valeur publiee roulait
// sur un ex aequo.
//
// LA REGLE POSEE : une valeur NON DISCRIMINEE n est JAMAIS ecrite au profil de balayage.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// TestMotDePoigneeRetenu — LA DECISION, EXHAUSTIVEMENT, SANS FILM. Tourne en CI.
func TestMotDePoigneeRetenu(t *testing.T) {
	const invariant = uint(1)
	cas := []struct {
		nom         string
		scores      []int
		retenu      uint
		discriminee bool
	}{
		{"egalite parfaite (a521164d, triplet lu)", []int{272, 272, 272}, invariant, false},
		{"egalite parfaite (64e8adfa, triplet lu)", []int{61, 61, 61}, invariant, false},
		{"l ecart qui decidait autrefois : 4 records sur 226", []int{226, 226, 230}, invariant, false},
		{"domination nette de la mediane", []int{500, 10, 10}, 1, true},
		{"domination nette, candidat du milieu", []int{10, 500, 10}, 2, true},
		{"domination nette, dernier candidat", []int{10, 10, 500}, 3, true},
		{"exactement au seuil (2x la mediane) : retenu", []int{200, 100, 10}, 1, true},
		{"juste sous le seuil : invariant", []int{199, 100, 10}, invariant, false},
		{"tout a zero : rien a voir", []int{0, 0, 0}, invariant, false},
		{"deux ex aequo en tete : la mediane EST le second, donc rien n est discrimine",
			[]int{500, 500, 10}, invariant, false},
		{"aucun candidat", nil, invariant, false},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			retenu, _, _, disc := motDePoigneeRetenu(c.scores, invariant)
			if retenu != c.retenu || disc != c.discriminee {
				t.Fatalf("scores %v : retenu=%d discriminee=%v, want retenu=%d discriminee=%v",
					c.scores, retenu, disc, c.retenu, c.discriminee)
			}
			// DETERMINISME : deux appels, meme verdict. `sort.Slice` n est pas stable, et c est
			// precisement ce qui rendait la valeur d avant arbitraire sur des ex aequo.
			for i := 0; i < 8; i++ {
				r2, _, _, d2 := motDePoigneeRetenu(append([]int(nil), c.scores...), invariant)
				if r2 != retenu || d2 != disc {
					t.Fatalf("appel %d : retenu=%d discriminee=%v, want %d / %v", i, r2, d2, retenu, disc)
				}
			}
		})
	}
}

// TestUneValeurNonDiscrimineeNEstJamaisEcriteAuProfil — RATCHET DE SOURCE.
//
// Le profil de balayage ne doit recevoir la largeur du mot de poignee QUE par le retour de
// `motDePoigneeRetenu`, qui porte le test de discrimination. Toute autre ecriture de
// `Traversal.IndexW` dans le paquet remettrait une valeur devinee sur le chemin du rejeu — c est
// le defaut du 2026-09-17, et il ne doit pas pouvoir revenir par une autre porte.
func TestUneValeurNonDiscrimineeNEstJamaisEcriteAuProfil(t *testing.T) {
	const ancre = "res.Profil.Mouvement.Traversal.IndexW = retenu"
	entrees, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du paquet : %v", err)
	}
	total := 0
	for _, e := range entrees {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		b, err := os.ReadFile(e.Name())
		if err != nil {
			t.Fatalf("%s : %v", e.Name(), err)
		}
		for _, ligne := range strings.Split(string(b), "\n") {
			l := strings.TrimSpace(ligne)
			if strings.HasPrefix(l, "//") || !strings.Contains(l, "Traversal.IndexW =") {
				continue
			}
			total++
			if l != ancre {
				t.Errorf("%s : ecriture de la largeur du mot de poignee HORS de la decision :\n  %s\n"+
					"  seule ecriture autorisee : %q (le retour de motDePoigneeRetenu, qui porte le "+
					"test de discrimination)", e.Name(), l, ancre)
			}
		}
	}
	if total != 1 {
		t.Errorf("ecritures de `Traversal.IndexW` dans le paquet = %d, want 1 — si la decision a "+
			"ete deplacee, deplacer l ancre de ce ratchet dans le MEME commit", total)
	}
}

// TestMotDePoigneeDeterministeSurFilm — LE TEST CIBLE, SUR LES DEUX FILMS INSTRUITS.
//
// Il rejoue la calibration complete DEUX FOIS par film et exige le meme verdict. Sur
// `a521164d` (Fragmentation Heavies) et `64e8adfa` (Catalyst), la mesure du 2026-09-17 dit que
// le critere ne discrimine pas : le verdict attendu est donc l INVARIANT, et il doit etre stable.
//
// GARDE `KS_POIGNEE_FILMS` (repertoires absolus separes par `;`) et `KS_POIGNEE_CARTES` (les
// noms de carte, liste parallele) — les films ne sont pas au depot, la CI saute.
//
//	KS_POIGNEE_FILMS='<cache>/a521164d;<cache>/64e8adfa' \
//	KS_POIGNEE_CARTES='Fragmentation Heavies;Catalyst' \
//	  go test ./internal/games/halo_infinite/film/internal/facts/killsource/ \
//	    -run '^TestMotDePoigneeDeterministeSurFilm$' -v
func TestMotDePoigneeDeterministeSurFilm(t *testing.T) {
	dirs := strings.Split(os.Getenv("KS_POIGNEE_FILMS"), ";")
	if len(dirs) == 0 || dirs[0] == "" {
		t.Skip("test cible : KS_POIGNEE_FILMS requis")
	}
	cartes := strings.Split(os.Getenv("KS_POIGNEE_CARTES"), ";")
	for i, dir := range dirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		nom := ""
		if i < len(cartes) {
			nom = strings.TrimSpace(cartes[i])
		}
		t.Run(filepath.Base(dir), func(t *testing.T) {
			o := DefaultOptions()
			if nom != "" {
				e := carteDuCatalogue(t, nom)
				o.Carte = &e
			}
			o.normalize()
			var prem calibration
			for passe := 0; passe < 2; passe++ {
				c := calibrerUnFilm(t, dir, o)
				t.Logf("passe %d : %s", passe+1, c.String())
				if passe == 0 {
					prem = c
					continue
				}
				if c.PoigneeIndexW != prem.PoigneeIndexW ||
					c.PoigneeDiscriminee != prem.PoigneeDiscriminee ||
					c.Profil.Mouvement.Traversal.IndexW != prem.Profil.Mouvement.Traversal.IndexW {
					t.Fatalf("verdict NON DETERMINISTE : passe 1 indexW=%d discriminee=%v profil=%d, "+
						"passe 2 indexW=%d discriminee=%v profil=%d",
						prem.PoigneeIndexW, prem.PoigneeDiscriminee,
						prem.Profil.Mouvement.Traversal.IndexW,
						c.PoigneeIndexW, c.PoigneeDiscriminee, c.Profil.Mouvement.Traversal.IndexW)
				}
			}
			if prem.PoigneeDiscriminee {
				t.Logf("NOTE : le critere DISCRIMINE sur ce film (indexW=%d) — la mesure du "+
					"2026-09-17 disait le contraire, le dire au pilote", prem.PoigneeIndexW)
				return
			}
			if got := prem.Profil.Mouvement.Traversal.IndexW; got != 1 {
				t.Errorf("non discriminee mais le profil porte indexW=%d, want l invariant 1", got)
			}
		})
	}
}

// calibrerUnFilm : une calibration complete, sur une timeline NEUVE (le balayage la consomme).
func calibrerUnFilm(t *testing.T, dir string, o Options) calibration {
	t.Helper()
	src, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("chargement %s : %v", dir, err)
	}
	f, err := loadFilm(src)
	if err != nil {
		t.Fatalf("film %s : %v", dir, err)
	}
	tl, err := newTimeline(f)
	if err != nil {
		t.Fatalf("timeline %s : %v", dir, err)
	}
	return calibrate(f, tl, o.Views, o.Carte)
}
