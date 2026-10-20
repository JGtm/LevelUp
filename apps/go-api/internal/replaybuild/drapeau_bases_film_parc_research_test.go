//go:build research

package replaybuild

// drapeau_bases_film_parc_research_test.go — INSTRUMENT DE GATE : LE CALQUE DU DRAPEAU REJOUE
// DEPUIS LES FAITS PERSISTES, AVANT / APRES LA LECTURE DES BASES DANS LE FILM.
//
// « Avant » est l'artefact range dans le cache (cuit par le code de son epoque) ; « apres » est le
// document reconstruit par ce code-ci, par la porte de production (`documentDeLaCuisson`), depuis
// le fichier de faits du match. AUCUN FILM N'EST DECODE, RIEN N'EST ECRIT.
//
// LA CARTE DU MATCH : le module vient de l'en-tete des faits ; le `map_id` du catalogue d'objectifs
// se retrouve par les SOCLES que l'artefact publie (etat `home` de chaque drapeau, a 5 cm pres d'un
// socle de l'entree). Un artefact sans socle (carte hors catalogue) se reconstruit sans `map_id`,
// exactement comme la cuisson de production d'une carte absente.
//
//	L1_DEPOT=<checkout qui porte data/cache> \
//	  go test -tags research -count=1 -run '^TestDrapeauBasesDuFilmAvantApres$' -v ./internal/replaybuild/

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/testutil"
)

// etatHome est l etat `home` d un drapeau, tel que le document le publie (le symbole du paquet
// n est pas cite ici : la surface de `film/replay` citee hors du decodeur est un plafond gele).
const etatHome = "home"

var l1ParcFilms = []string{
	// cartes absentes du catalogue
	"fd247c3f", "f7b74a65", "fa70c437", "92c950ee", "f1db4a07", "798d1ff4",
	"5da3e346", "708abcd1", "73c1df0b", "be758198",
	// temoins a deux socles nommes
	"eba1e63f", "068fb1ac", "1e5e355e", "0ffebf8b", "13d92593",
	// socles sans camp ou surnumeraires
	"a32ee8d2", "61614156", "81cc9952", "084a804d", "db6dc73c", "390b1de5",
	// drapeau neutre
	"e94163af", "a1995edc", "059b721f", "323ec1cf",
}

func TestDrapeauBasesDuFilmAvantApres(t *testing.T) {
	depot := os.Getenv("L1_DEPOT")
	if depot == "" {
		t.Skip("instrument de gate : L1_DEPOT requis")
	}
	repoRoot, err := testutil.RepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewBuilder(context.Background(), repoRoot, title.DefaultSlug)
	if err != nil {
		t.Fatal(err)
	}
	modules := l1ModulesDuCatalogue(t, repoRoot)
	objectifs, err := replay.LoadMapObjectives(title.NewPathResolver(repoRoot).MapObjectivesPath(title.DefaultSlug))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range l1ParcFilms {
		t.Run(id, func(t *testing.T) {
			artefact := filepath.Join(depot, "data", "cache", "replays", "halo_infinite", id+".json")
			avant := l1LireArtefact(t, artefact)
			mapID := l1CarteDesSocles(objectifs, avant)
			facts := l1FeuilleDeLArtefact(t, artefact)
			facts.MapID = mapID
			apres, mode := l1Reconstruire(t, b, modules, filepath.Join(depot, "data", "cache", "film_facts",
				"halo_infinite", id+".filmfacts.bin"), avant.MatchID, facts)
			t.Logf("%s carte=%q (%s)", id, mapID, mode)
			t.Logf("   AVANT %s", l1Resume(avant))
			t.Logf("   APRES %s", l1Resume(apres))
			ja, _ := json.Marshal(avant.FlagCarries)
			jb, _ := json.Marshal(apres.FlagCarries)
			t.Logf("   flagCarries identiques a l'octet : %v", string(ja) == string(jb))
		})
	}
}

// l1Resume resume le calque : drapeaux (camp, etats home), couverture et replis du drapeau.
func l1Resume(d replay.ReplayDocument) string {
	var parts []string
	for _, c := range d.FlagCarries {
		homes, x, y := 0, float32(0), float32(0)
		for _, s := range c.Spans {
			if s.State == etatHome {
				if homes == 0 {
					x, y = s.X, s.Y
				}
				homes++
			}
		}
		parts = append(parts, fmt.Sprintf("drapeau equipe %d home=%d socle=(%.3f,%.3f) spans=%d",
			c.Team, homes, x, y, len(c.Spans)))
	}
	cov := d.Coverage.FlagCarries
	if cov != nil {
		parts = append(parts, fmt.Sprintf("couv: openings=%d noBridge=%d noTrack=%d spawns=%d neutre=%v carries=%d carrierTeamUnknown=%d "+
			"homeByObject=%d closedByHandoff=%d closedByReturn=%d closedByHome=%d ownFlagRefused=%d "+
			"unresolved=%d filmBases=%d spawnsFromFilm=%d accord=%d contradiction=%d",
			cov.Openings, cov.NoBridge, cov.NoTrack, cov.Spawns, cov.NeutralFlag, cov.Carries, cov.CarrierTeamUnknown, cov.HomeByObject,
			cov.ClosedByHandoff, cov.ClosedByReturn, cov.ClosedByHome, cov.OwnFlagRefused, cov.Unresolved,
			cov.FilmBases, cov.SpawnsFromFilm, cov.FilmBaseAgree, cov.FilmBaseContradict))
	}
	for _, f := range d.Coverage.Fallbacks {
		if strings.Contains(f.Name, "drapeau") || strings.Contains(f.Name, "socle") {
			parts = append(parts, fmt.Sprintf("%s=%d", f.Name, f.Hits))
		}
	}
	return strings.Join(parts, " | ")
}

// l1Reconstruire reconstruit le document depuis les faits, par la porte de production.
func l1Reconstruire(t *testing.T, b *Builder, modules map[string]string, chemin, matchID string,
	facts port.MatchFacts) (replay.ReplayDocument, string) {
	blob, err := os.ReadFile(chemin) //nolint:gosec // instrument, chemin du depot
	if err != nil {
		t.Skipf("faits absents : %v", err)
	}
	entete, err := replay.DecodeFilmFactsEntete(blob)
	if err != nil {
		t.Skipf("en-tete illisible : %v", err)
	}
	cle, ok := modules[entete.MapModule]
	if !ok {
		t.Skipf("module %q absent du catalogue de bornes", entete.MapModule)
	}
	entry, err := b.ResolveMapEntry([]string{cle})
	if err != nil {
		t.Skip(err)
	}
	mode := "faits frais, porte de production"
	frais := entete.Frais(entry) == nil
	if !frais {
		// LES FAITS DU CACHE SONT ANTERIEURS A LA REVISION DE GRAMMAIRE DE CE BINAIRE : la porte de
		// production les refuserait (redecodage). L'instrument les rejoue quand meme par
		// `BuildFromFacts` — le calque du drapeau lit le statborg (revision `objectives` inchangee)
		// et les pistes de ces faits-la, les memes que celles de l'artefact « avant ».
		mode = "faits de grammaire anterieure, rejoues hors porte"
	}
	f, err := replay.DecodeFilmFactsFile(blob, entry)
	if err != nil {
		t.Skipf("faits illisibles : %v", err)
	}
	ctx := context.Background()
	src := entreesDesFaits(f)
	src.entete = entete
	stats := assemblerFilmStats(ctx, matchID, src.statborg, facts, src.deaths)
	cat := b.collecterEntreesCatalogue(ctx, matchID, []string{cle}, facts, &stats, src)
	opts := b.buildReplayOptions(ctx, entry, facts, cat, &stats)
	if !frais {
		return replay.BuildFromFacts(ctx, matchID, b.titleSlug, f, opts), mode
	}
	cuit, err := b.documentDeLaCuisson(ctx, matchID, "", opts, src)
	if err != nil {
		t.Skipf("cuisson refusee sans film : %v", err)
	}
	return cuit.doc, mode
}

// l1CarteDesSocles rend le `map_id` dont les socles portent TOUS les socles publies par
// l'artefact (5 cm pres) ; "" quand l'artefact n'en publie aucun.
func l1CarteDesSocles(cat *replay.MapObjectivesCatalog, d replay.ReplayDocument) string {
	var publies [][2]float32
	for _, c := range d.FlagCarries {
		for _, s := range c.Spans {
			if s.State == etatHome {
				publies = append(publies, [2]float32{s.X, s.Y})
				break
			}
		}
	}
	if len(publies) == 0 {
		return ""
	}
	ids := make([]string, 0, len(cat.Maps))
	for id := range cat.Maps {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		socles := cat.Maps[id].SoclesDeDrapeau()
		if l1ToutPres(publies, socles) {
			return id
		}
	}
	return "?"
}

func l1ToutPres(publies [][2]float32, socles []replay.FlagSpawn) bool {
	if len(socles) == 0 {
		return false
	}
	for _, p := range publies {
		ok := false
		for _, s := range socles {
			dx, dy := p[0]-s.X, p[1]-s.Y
			if dx*dx+dy*dy <= 0.05*0.05 {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	return true
}

func l1LireArtefact(t *testing.T, chemin string) replay.ReplayDocument {
	blob, err := os.ReadFile(chemin) //nolint:gosec // instrument, chemin du depot
	if err != nil {
		t.Skipf("artefact absent : %v", err)
	}
	var d replay.ReplayDocument
	if err := json.Unmarshal(blob, &d); err != nil {
		t.Fatal(err)
	}
	return d
}

// l1ModulesDuCatalogue rend, par module de carte, une cle du catalogue de bornes.
func l1ModulesDuCatalogue(t *testing.T, repoRoot string) map[string]string {
	blob, err := os.ReadFile(filepath.Join(repoRoot, "data", "titles", "halo_infinite", "reference", "map_quant_bounds.json")) //nolint:gosec // catalogue versionne
	if err != nil {
		t.Fatal(err)
	}
	var cat struct {
		Maps map[string]struct {
			Module string `json:"module"`
		} `json:"maps"`
	}
	if err := json.Unmarshal(blob, &cat); err != nil {
		t.Fatal(err)
	}
	cles := make([]string, 0, len(cat.Maps))
	for k := range cat.Maps {
		cles = append(cles, k)
	}
	sort.Strings(cles)
	out := map[string]string{}
	for _, k := range cles {
		if _, vu := out[cat.Maps[k].Module]; !vu {
			out[cat.Maps[k].Module] = k
		}
	}
	return out
}

// l1FeuilleDeLArtefact reconstitue les lignes de match que la base fournit a la cuisson : xuid et
// camp du roster publie, frags / morts / assistances au dernier point de la courbe de score. Sans
// elles, le pont du statborg perd l'appariement par triplet et le calque perd des prises
// (`noBridge`) — un ecart d'entree, pas de regle.
func l1FeuilleDeLArtefact(t *testing.T, chemin string) port.MatchFacts {
	blob, err := os.ReadFile(chemin) //nolint:gosec // instrument, chemin du depot
	if err != nil {
		t.Fatal(err)
	}
	type serie struct {
		Total []struct {
			V int `json:"v"`
		} `json:"total"`
	}
	var a struct {
		Roster []struct {
			XUID string `json:"xuid"`
			Team int    `json:"team"`
		} `json:"roster"`
		ScoreTimeline *struct {
			Players []struct {
				XUID    string `json:"xuid"`
				Kills   serie  `json:"kills"`
				Deaths  serie  `json:"deaths"`
				Assists serie  `json:"assists"`
			} `json:"players"`
		} `json:"scoreTimeline"`
	}
	if err := json.Unmarshal(blob, &a); err != nil {
		t.Fatal(err)
	}
	dernier := func(s serie) int {
		if len(s.Total) == 0 {
			return 0
		}
		return s.Total[len(s.Total)-1].V
	}
	kda := map[string][3]int{}
	if a.ScoreTimeline != nil {
		for _, p := range a.ScoreTimeline.Players {
			kda[p.XUID] = [3]int{dernier(p.Kills), dernier(p.Deaths), dernier(p.Assists)}
		}
	}
	var out port.MatchFacts
	for _, r := range a.Roster {
		if r.XUID == "" {
			continue
		}
		k := kda[r.XUID]
		out.Players = append(out.Players, port.MatchPlayerFact{XUID: r.XUID, TeamID: r.Team,
			Kills: k[0], Deaths: k[1], Assists: k[2]})
	}
	return out
}
