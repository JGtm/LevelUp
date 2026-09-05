package replay

// assaut_a5_explosions_test.go — LA CONFRONTATION DE PUBLICATION : les explosions que le rejeu
// PUBLIE contre celles que le releve A0.3 a DATEES.
//
// # Ce que ce test ferme
//
// Le lot A s'etait arrete au diagnostic : `comp 0 A` des slots de joueur replique les points de
// mode (phase A4, moities disjointes 4/4 + 4/4), mais rien n'etait publie et `RealRounds`
// tronquait One Bomb a la manche 0. Depuis le 2026-08-31, `ObjectiveTypeBomb` nomme
// l'emplacement et le second critere d'admission des manches ouvre les manches 1..3.
//
// Ce test confronte la sortie de PRODUCTION (`NamedEventsFrom(recs, ObjectiveTypeBomb)`, ce que
// `replaybuild` publie dans `doc.objectives`) au releve A0.3 FIGE — `A_PROTOCOLE.md` §2, ecrit et
// commite le 2026-08-27, bien avant ce lot. L'oracle est donc anterieur a la mesure.
//
// # Les deux quantites, et pourquoi elles different
//
//	DATEES      les instants d'explosion du releve, releves sur les slots d'EQUIPE (6 et 8).
//	ATTRIBUEES  les evenements publies, qui vivent sur les slots de JOUEUR.
//
// Deux explosions sur 28 n'ont AUCUN slot de joueur porteur : `df8fcbef` manche 3 et `c75f33b8`
// manche 1 ne portent le point que sur le slot d'EQUIPE (diagnostic du 2026-08-31, tous les
// enregistrements `comp 0` de ces manches imprimes). Ce n'est pas un filtre : le film ne
// replique pas le compteur par joueur pour ces deux-la. La courbe de score, elle, les porte.
//
// LE CRITERE : tout evenement publie doit tomber sur un instant DATE du releve, A LA
// MILLISECONDE (aucune tolerance — les deux viennent de la meme horloge de manifeste), et le
// taux d'attribution doit valoir exactement les 26 sur 28 mesures. Un ecart dans un sens comme
// dans l'autre est une regression.
//
// REGIME : garde `ASSAUT_CACHE`. Aucune base, aucun reseau.
//
//	$env:ASSAUT_CACHE="C:/.../data/cache"
//	go test ./internal/analysis/replay/ -run AssautA5Explosions -v -timeout 30m

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

// a5Explosions : les instants d'explosion DATES du releve A0.3 (`A_PROTOCOLE.md` §2), recopies
// sans modification. Les 9 films d'Assaut du corpus, `ce083875` compris (exclu du lot A pour son
// pont d'identite a 10,6 %, mais son score de mode est releve comme les autres).
var a5Explosions = map[string][]int{
	"35b75a31": {304013, 541270, 787051},
	"69b16f5d": {154305, 278617, 310215},
	"3d58eb37": {203065, 342196, 386280},
	"34bb3bc8": {427120},
	"1c01e34f": {150546, 273787, 335637, 400853},
	"ce083875": {512505, 686401, 947537},
	"df8fcbef": {255767, 309284, 485860, 778033},
	"c75f33b8": {109549, 395724, 450833},
	"9f57c612": {83322, 298489, 353160, 469057},
}

// a5SansPorteur : les explosions DONT AUCUN SLOT DE JOUEUR ne porte le point — mesure du
// 2026-08-31, tous les enregistrements `comp 0` de la manche imprimes (le point n'existe que sur
// le slot d'EQUIPE). Elles sont attendues MANQUANTES a l'attribution, et leur absence est donc
// un resultat, pas un echec.
var a5SansPorteur = map[string]int{"df8fcbef": 778033, "c75f33b8": 395724}

// TestAssautA5Explosions confronte la publication au releve.
func TestAssautA5Explosions(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	defer amArmeSentinelle(t, "TestAssautA5Explosions")()
	films := make([]string, 0, len(a5Explosions))
	for id := range a5Explosions {
		films = append(films, id)
	}
	sort.Strings(films)

	var datees, attribuees, horsReleve int
	for _, id := range films {
		src, ok, err := filmcache.LoadFilm(cache, id)
		if err != nil || !ok {
			t.Fatalf("film %s absent du cache (%s) : %v — la mesure serait partielle", id, cache, err)
		}
		recs, _ := objectiveevents.StatRecordsCtx(context.Background(), src, id)
		attendus := map[int]bool{}
		for _, ms := range a5Explosions[id] {
			attendus[ms] = true
		}
		datees += len(a5Explosions[id])

		publies := map[int]bool{}
		for _, e := range objectiveevents.NamedEventsFrom(recs, objectiveevents.ObjectiveTypeBomb) {
			if e.Stat != objectiveevents.StatBombDetonations {
				t.Errorf("%s : statistique inattendue %q", id, e.Stat)
				continue
			}
			if !attendus[e.TimeMS] {
				horsReleve++
				t.Errorf("%s : explosion publiee a %d ms ABSENTE du releve A0.3 %v",
					id, e.TimeMS, a5Explosions[id])
				continue
			}
			if publies[e.TimeMS] {
				t.Errorf("%s : explosion a %d ms publiee DEUX fois", id, e.TimeMS)
				continue
			}
			publies[e.TimeMS] = true
		}
		attribuees += len(publies)

		for _, ms := range a5Explosions[id] {
			if publies[ms] {
				continue
			}
			if a5SansPorteur[id] == ms {
				continue // connue sans porteur, cf. l'en-tete
			}
			t.Errorf("%s : explosion datee a %d ms NON publiee — regression d'attribution", id, ms)
		}
		t.Logf("%s : %d/%d explosions attribuees a un joueur", id, len(publies), len(a5Explosions[id]))
	}

	const (
		a5Datees     = 28
		a5Attribuees = 26
	)
	t.Logf("BILAN NOMMAGE : %d explosions datees au releve, %d attribuees a un SLOT (%.1f %%), "+
		"%d publiees hors releve", datees, attribuees, 100*float64(attribuees)/float64(datees), horsReleve)
	if datees != a5Datees {
		t.Errorf("le releve fige porte %d explosions, attendu %d", datees, a5Datees)
	}
	if attribuees != a5Attribuees {
		t.Errorf("%d explosions attribuees, attendu %d — la publication a change", attribuees, a5Attribuees)
	}
}

// a5Publiees : le nombre d'explosions qui arrivent A L'ECRAN, apres le PONT D'IDENTITE.
//
// # DEUX CHIFFRES, ET IL FAUT LES DEUX
//
// Le nommage attribue une explosion a un SLOT d'entite statborg (26/28). Le rejeu, lui, joint sur
// le XUID : une explosion dont le pont ne sait pas nommer le slot POUR SA MANCHE n'est pas
// publiable — la rattacher a un slot d'une autre manche serait exactement l'erreur que le pont
// par manche existe pour eviter (le slot est REATTRIBUE d'une manche a l'autre).
//
// Mesure du 2026-08-31 : **21 sur 28**. Les 5 pertes sont toutes sur les 3 films One Bomb, les
// seuls multi-manches — le pont y resout chaque manche separement, sur les seules progressions du
// compteur de morts DE CETTE MANCHE, et une manche courte n'en offre pas assez. Ce n'est pas une
// regression de ce lot : avant lui, `RealRounds` ne retenait qu'une manche par film One Bomb et
// il n'y avait qu'UNE explosion a publier par film (3 au total contre 21 aujourd'hui).
//
// Le chiffre est FIGE ici pour qu'une amelioration du pont d'identite se voie, et qu'une
// degradation se voie aussi.
const a5Publiees = 21

// TestAssautA5PontIdentite mesure ce qui arrive a l'ecran, pont d'identite compris.
func TestAssautA5PontIdentite(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	defer amArmeSentinelle(t, "TestAssautA5PontIdentite")()
	films := make([]string, 0, len(a5Explosions))
	for id := range a5Explosions {
		films = append(films, id)
	}
	sort.Strings(films)

	var nommees, publiees int
	for _, id := range films {
		src, ok, err := filmcache.LoadFilm(cache, id)
		if err != nil || !ok {
			t.Fatalf("film %s absent du cache : %v", id, err)
		}
		recs, _ := objectiveevents.StatRecordsCtx(context.Background(), src, id)
		deaths, err := ScanFilmDeaths(filepath.Join(cache, "film_chunks", id))
		if err != nil {
			t.Fatalf("%s : fil des morts illisible : %v", id, err)
		}
		instants := make([]objectiveevents.DeathInstant, 0, len(deaths))
		for _, d := range deaths {
			instants = append(instants, objectiveevents.DeathInstant{
				XUID: strconv.FormatUint(d.XUID, 10), TimeMS: int(d.TimeMS)})
		}
		named := objectiveevents.NamedEventsFrom(recs, objectiveevents.ObjectiveTypeBomb)
		ident := objectiveevents.IdentifyNamedEventsByRound(named,
			objectiveevents.ResolveRoundIdentity(recs, instants))
		nommees += len(named)
		publiees += len(ident)
		t.Logf("%s : %d nommee(s) -> %d publiee(s)", id, len(named), len(ident))
	}
	t.Logf("BILAN PONT : %d nommee(s) -> %d publiee(s) a l'ecran (%.1f %% des 28 datees)",
		nommees, publiees, 100*float64(publiees)/28)
	if publiees != a5Publiees {
		t.Errorf("%d explosions publiees, attendu %d — le pont d'identite a change (une hausse "+
			"est une bonne nouvelle a figer ici, une baisse est une regression)", publiees, a5Publiees)
	}
}
