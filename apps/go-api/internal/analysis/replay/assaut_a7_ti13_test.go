package replay

// assaut_a7_ti13_test.go — L'ARMEMENT DANS LA TROISIEME MAISON : le canal `ti=13`.
//
// # LES TROIS MAISONS, ET POURQUOI CELLE-CI EST LA BONNE A FOUILLER
//
// L'utilisateur, le 2026-08-31 : « statborg a les prises de colline en KOTH, les jauges et
// minuteurs pour CTF, les zones en capture et tout pour Strongholds. Ce serait etrange d'avoir
// ca a un autre endroit. » Il a raison sur le voisinage, et la carte exacte est celle-ci :
//
//	1. COMPOSANTS DU STATBORG   les compteurs PAR JOUEUR — `flag_captures`, `zone_captures`,
//	                            `vip_selected`, et depuis ce jour `bomb_detonations`.
//	2. PIED DE FILM, `th=10`    les INTERACTIONS d'objectif — prises de zone, de colline,
//	                            possession du crane (`extractFromTh10`).
//	3. ARCHETYPE `ti=13`        les PROPRIETES D'OBJET GERE : la JAUGE de capture, le
//	                            proprietaire d'une zone, la colline active
//	                            (`filmdec.ScanFilmManagedProperties`).
//
// Les jauges et les minuteurs que l'utilisateur cite vivent dans la TROISIEME, pas dans le
// statborg. Les deux premieres ont ete balayees pour l'Assaut, et rendues negatives :
// phase A6 (112 canaux de composant) et sonde du pied (tous les indices de type, `th=10` quasi
// absent en Assaut — 6 blocs sur 9 films). Restait celle-ci.
//
// # LE DECALAGE D'HORLOGE NE GENE PAS, ET C'EST CE QUI REND LA MESURE POSSIBLE
//
// `ti=13` est date sur l'horloge MOTEUR (`TimestampUS`), les explosions sur celle du MANIFESTE.
// L'ecart entre les deux est inconnu. Mais le critere ne porte PAS sur la valeur du delai — il
// porte sur sa DISPERSION, et un decalage constant ne change pas une dispersion. La mediane
// rendue vaut donc « meche + decalage » ; c'est la CONSTANCE qui designe le canal, et le
// recalage se fait apres, une fois le canal connu.
//
// # LE CRITERE, ecrit avant la mesure — le meme que les deux autres maisons
//
//	COUVERTURE   au moins une progression de ce (slot, tag) avant CHAQUE explosion ;
//	CONSTANCE    dispersion des delais <= 20 % de la mediane ;
//	SENS         delai positif, sous 120 s.
//
// # LE VERDICT N'EST PAS UN NEGATIF : C'EST UN CANAL ILLISIBLE (mesure du 2026-08-31)
//
// La premiere passe n'a garde que les lectures CHAINEES (`Chained`, le seul temoin de fiabilite
// par lecture) et a rendu ZERO progression — parce que le chainage vaut **1,9 a 16,4 %** sur ces
// neuf films, contre 87 a 99 % sur un KOTH de reference. C'est la contamination d'ancrage etablie
// en phase A3, et elle ne laisse rien passer.
//
// La seconde passe relache le filtre ([a7ExigeChainage]) pour voir s'il y a du signal SOUS le
// bruit : 7 couples (slot, tag) portent une progression, chacun couvrant 1 explosion sur 28.
// Rien de coherent.
//
// **Il faut donc lire ce resultat comme « ce canal n'est pas lisible en Assaut », et non comme
// « l'armement n'y est pas ».** Les 8 slots `ti=13` sont bien la, a chaque film ; c'est leur
// ancrage qui est casse. Reparer l'ancrage de `ti=13` sur les cartes d'Assaut est le prealable,
// et c'est un chantier deja ouvert (phase A3).
//
// REGIME : garde `ASSAUT_CACHE`. Aucune base, aucun reseau, sentinelle memoire armee.
//
//	$env:ASSAUT_CACHE="C:/.../data/cache"
//	go test ./internal/analysis/replay/ -run AssautA7Ti13 -v -timeout 60m

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// a7ExigeChainage : le filtre de fiabilite. MIS A FAUX le 2026-08-31 apres une premiere passe
// qui rendait ZERO progression — le chainage vaut 1,9 a 16,4 % en Assaut (contamination
// d'ancrage etablie en phase A3), si bien que le filtre ne laissait rien passer. La passe
// relachee ne REPARE rien : elle dit seulement s'il y a du signal SOUS le bruit d'ancrage. Un
// candidat qui en sortirait devrait etre reconfirme sur un canal correctement ancre.
const a7ExigeChainage = false

// a7Cle designe une propriete reseau : le slot de l'objet gere, et le tag du variant.
type a7Cle struct {
	slot uint32
	tag  int
}

func (k a7Cle) String() string { return fmt.Sprintf("slot %d tag %d", k.slot, k.tag) }

// TestAssautA7Ti13 balaie le canal des proprietes d'objet gere sur les 9 films d'Assaut.
func TestAssautA7Ti13(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	defer amArmeSentinelle(t, "TestAssautA7Ti13")()

	films := make([]string, 0, len(a5Explosions))
	for id := range a5Explosions {
		films = append(films, id)
	}
	sort.Strings(films)

	delais := map[a7Cle][]float64{}
	couverts := map[a7Cle]int{}
	total := 0
	for _, id := range films {
		sc, err := filmdec.ScanFilmManagedProperties(filepath.Join(cache, "film_chunks", id))
		if err != nil {
			t.Logf("%s : balayage ti=13 impossible (%v)", id, err)
			continue
		}
		chainage := 0.0
		if sc.Walked > 0 {
			chainage = 100 * float64(sc.Chained) / float64(sc.Walked)
		}
		t.Logf("%s : %d slots, %d records, %d lectures, chainage %.1f %%",
			id, sc.Slots, sc.Records, len(sc.Reads), chainage)

		// LES PROGRESSIONS PAR (slot, tag), en microsecondes moteur.
		prog := map[a7Cle][]int{}
		dernier := map[a7Cle]uint64{}
		vus := map[a7Cle]bool{}
		lect := append([]filmdec.ManagedPropertyRead(nil), sc.Reads...)
		sort.SliceStable(lect, func(i, j int) bool { return lect[i].TimestampUS < lect[j].TimestampUS })
		for _, r := range lect {
			if !r.HasValue || (a7ExigeChainage && !r.Chained) {
				continue
			}
			k := a7Cle{slot: r.Slot, tag: r.Tag}
			if vus[k] && r.Value > dernier[k] {
				prog[k] = append(prog[k], int(r.TimestampUS/1000))
			}
			dernier[k], vus[k] = r.Value, true
		}

		exps := a5Explosions[id]
		total += len(exps)
		for k, ts := range prog {
			for _, ms := range exps {
				meilleur := -1
				for _, p := range ts {
					d := ms - p
					if d > 0 && d <= a6MecheMaxMS && (meilleur < 0 || d < meilleur) {
						meilleur = d
					}
				}
				if meilleur >= 0 {
					couverts[k]++
					delais[k] = append(delais[k], float64(meilleur))
				}
			}
		}
	}

	type verdict struct {
		k       a7Cle
		couvert int
		med, cv float64
	}
	var tenus []verdict
	for k, ds := range delais {
		if couverts[k] < total {
			continue
		}
		med, cv := a6MedianeEtCV(ds)
		if cv <= a6CVMax {
			tenus = append(tenus, verdict{k, couverts[k], med, cv})
		}
	}
	sort.Slice(tenus, func(i, j int) bool { return tenus[i].cv < tenus[j].cv })

	t.Logf("TI=13 : %d explosions sur %d films, %d couples (slot, tag) porteurs de progressions",
		total, len(films), len(delais))
	if len(tenus) == 0 {
		t.Logf("AUCUN COUPLE NE TIENT LES TROIS CRITERES. Les meilleures couvertures :")
		type l struct {
			k       a7Cle
			n       int
			med, cv float64
		}
		ls := make([]l, 0, len(couverts))
		for k, n := range couverts {
			med, cv := a6MedianeEtCV(delais[k])
			ls = append(ls, l{k, n, med, cv})
		}
		sort.Slice(ls, func(i, j int) bool {
			if ls[i].n != ls[j].n {
				return ls[i].n > ls[j].n
			}
			return ls[i].cv < ls[j].cv
		})
		for i, x := range ls {
			if i >= 12 {
				break
			}
			t.Logf("  %s : %d/%d couvertes, mediane %.1f s, dispersion %.0f %%",
				x.k, x.n, total, x.med/1000, x.cv*100)
		}
		return
	}
	for _, v := range tenus {
		t.Logf("CANDIDAT %s : %d/%d couvertes, delai median %.1f s (meche + decalage d'horloge), "+
			"dispersion %.0f %%", v.k, v.couvert, total, v.med/1000, v.cv*100)
	}
}
