//go:build research

package killsource

// botmeta_equipe_research_test.go — LA LECTURE DE L EQUIPE DES BOTS CONTRE SON ORACLE, SUR LE PARC
// (lot « toute entree du roster a l'equipe que le film ecrit », 2026-10-06,
// `.ai/PLAN_REJEU_EQUIPES_SOURCE_2026-10-06.md`, G3.0).
//
// L ORACLE est independant de la lecture : l equipe que l entite `ti=9` d un bot lui donne, liee par
// ses declarations BOT_METADATA (TSV de `replay/rejeu_equipes_research_test.go`,
// `TestRJEEquipesDesBotsParLeurEntite`). La LECTURE est celle de production
// ([poserLesEquipesDesBots]), sur le build que la table du film a lu. Le critere, ecrit avant la
// mesure : 100 % d accord sur les bots a l equipe connue, aucun bot illisible ni contradictoire sur
// un build du profil, et chaque paquet BOT_METADATA ferme.
//
//	RJE_BOTS=<tsv> RJE_CHUNKS=<data>/cache/film_chunks RJE_VERROU=<racine de cache du verrou solo> \
//	  go test -tags research -count=1 -run '^TestRJELectureDeLEquipeDesBots$' -v \
//	  ./internal/games/halo_infinite/film/internal/facts/killsource/

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// rjeOracle : un bot du TSV, son equipe par entite (-2 : inconnue).
type rjeOracle struct {
	film, nom string
	slot, bid int
	equipe    int
}

func rjeLireOracle(t *testing.T, chemin string) map[string][]rjeOracle {
	t.Helper()
	blob, err := os.ReadFile(chemin) //nolint:gosec // instrument : chemin fourni par l operateur
	if err != nil {
		t.Fatalf("tsv : %v", err)
	}
	out := map[string][]rjeOracle{}
	for i, ligne := range strings.Split(strings.TrimSpace(string(blob)), "\n") {
		if i == 0 {
			continue
		}
		c := strings.Split(ligne, "\t")
		slot, _ := strconv.Atoi(c[2])
		bid, _ := strconv.Atoi(c[3])
		eq := -2
		if v, err := strconv.Atoi(c[4]); err == nil {
			eq = v
		}
		out[c[0]] = append(out[c[0]], rjeOracle{film: c[0], nom: c[1], slot: slot, bid: bid, equipe: eq})
	}
	return out
}

// rjeBilan : les comptes du parc.
type rjeBilan struct {
	bots, connus, accords, desaccords, lusSansOracle, illisibles, contradictoires int
	jumeaux, horsDomaine, nonFermes, paquets, persoInconnue, horsGrammaire        int
	horsBalayage                                                                  int
	parBuild                                                                      map[string]int
}

func TestRJELectureDeLEquipeDesBots(t *testing.T) {
	tsv, chunks, verrou := os.Getenv("RJE_BOTS"), os.Getenv("RJE_CHUNKS"), os.Getenv("RJE_VERROU")
	if tsv == "" || chunks == "" || verrou == "" {
		t.Skip("RJE_BOTS, RJE_CHUNKS ou RJE_VERROU absent : instrument saute")
	}
	lock, err := filmproc.AcquireSolo(verrou, "rje-equipe-bots", "parc")
	if err != nil {
		t.Fatalf("verrou : %v", err)
	}
	defer lock.Release()
	filmproc.LowerOwnPriority("rje-equipe-bots")
	garde := filmproc.Arm("rje-equipe-bots", 3, func(pic uint64) { panic(fmt.Sprintf("plafond memoire franchi : %d", pic)) })
	defer garde.Disarm()
	oracle := rjeLireOracle(t, tsv)
	films := make([]string, 0, len(oracle))
	for f := range oracle {
		films = append(films, f)
	}
	sort.Strings(films)
	b := &rjeBilan{parBuild: map[string]int{}}
	for _, id := range films {
		rjeLireLeFilm(t, filepath.Join(chunks, id), oracle[id], b)
		debug.FreeOSMemory()
	}
	t.Logf("BILAN : %d bots du lecteur historique sur %d films ; equipe connue par entite %d dont %d en accord, "+
		"%d en DESACCORD ; lus sans oracle %d ; illisibles %d ; contradictoires %d ; hors grammaire %d ; "+
		"entrees hors balayage %d ; jumeaux discordants %d ; hors domaine %d ; paquets non fermes %d sur %d ; "+
		"films a perso inconnue %d ; bots par build %v",
		b.bots, len(films), b.connus, b.accords, b.desaccords, b.lusSansOracle, b.illisibles, b.contradictoires,
		b.horsGrammaire, b.horsBalayage, b.jumeaux, b.horsDomaine, b.nonFermes, b.paquets, b.persoInconnue, b.parBuild)
	if b.desaccords > 0 || b.illisibles > 0 || b.contradictoires > 0 || b.nonFermes > 0 || b.horsBalayage > 0 {
		t.Errorf("critere non tenu : desaccords %d, illisibles %d, contradictoires %d, paquets non fermes %d, "+
			"entrees hors balayage %d", b.desaccords, b.illisibles, b.contradictoires, b.nonFermes, b.horsBalayage)
	}
}

// rjeLireLeFilm lit l equipe des bots d UN film par la lecture de production et la confronte a
// l oracle.
func rjeLireLeFilm(t *testing.T, dir string, attendus []rjeOracle, b *rjeBilan) {
	t.Helper()
	src, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("%s : %v", dir, err)
	}
	f, err := loadFilm(src)
	if err != nil {
		t.Fatalf("%s : %v", dir, err)
	}
	table := readFilmTable(f)
	perso, connue := profile.PersonnalisationOctets(table.Build)
	paquets := grammar.PaquetsBotMetadata(f.src, perso*8, connue)
	m := loadBotMeta(paquets)
	poserLesEquipesDesBots(paquets, &m, connue)
	id := filepath.Base(dir)
	b.paquets += len(paquets)
	b.illisibles += m.Equipes.Illisibles
	b.contradictoires += m.Equipes.Contradictoires
	b.jumeaux += m.Equipes.JumeauxDiscordants
	b.horsDomaine += m.Equipes.HorsDomaine
	b.horsGrammaire += m.Equipes.HorsGrammaire
	b.horsBalayage += m.Equipes.EntreesHorsBalayage
	b.nonFermes += m.Equipes.PaquetsNonFermes
	if m.Equipes.PersoInconnue {
		b.persoInconnue++
	}
	for _, bt := range m.Bots {
		b.bots++
		b.parBuild[table.Build]++
		lu := "?"
		if bt.equipeLue {
			lu = strconv.Itoa(bt.equipe)
		}
		att := -2
		for _, o := range attendus {
			if o.slot == bt.Slot && o.bid == bt.BotID && o.nom == bt.Name {
				att = o.equipe
			}
		}
		verdict := "sans oracle"
		switch {
		case att >= 0 && bt.equipeLue && bt.equipe == att:
			b.connus++
			b.accords++
			verdict = "ACCORD"
		case att >= 0:
			b.connus++
			b.desaccords++
			verdict = "DESACCORD"
		case bt.equipeLue:
			b.lusSansOracle++
		}
		t.Logf("  %s %s %-16q slot %2d bid %2d : lue %s, entite %d -> %s", id, table.Build, bt.Name, bt.Slot,
			bt.BotID, lu, att, verdict)
	}
	if m.Equipes.PaquetsNonFermes > 0 || m.Equipes.aLire() || m.Equipes.EntreesHorsBalayage > 0 {
		t.Logf("  %s : paquets non fermes %d, illisibles %d, contradictoires %d, incomplets %d, hors grammaire %d, "+
			"hors balayage %d", id,
			m.Equipes.PaquetsNonFermes, m.Equipes.Illisibles, m.Equipes.Contradictoires, m.Incomplets, m.Equipes.HorsGrammaire,
			m.Equipes.EntreesHorsBalayage)
	}
}
