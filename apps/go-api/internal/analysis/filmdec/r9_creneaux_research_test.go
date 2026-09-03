package filmdec

// r9_creneaux_research_test.go — DES CRENEAUX A REGARDER DANS LE THEATER, PAS UNE STATISTIQUE.
//
// R8 a etabli le canal du PROPULSEUR : l'impulsion `tag == 1` du composant bipede i57
// `biped-spartan-ability` (et de son jumeau non predit i59). Cet instrument-ci ne re-etablit
// rien : il REUTILISE la detection de `r8_i59_tags_research_test.go` et repond a une question
// differente — « ou, exactement, l'utilisateur doit-il poser le curseur du visionneur ? ».
//
// DEUX CHOSES SONT DONC NEUVES ICI, ET ELLES SONT TOUTES DEUX DES CONVERSIONS :
//
//  1. L'IDENTITE. Le film ne connait que des SLOTS. L'artefact de rejeu (`roster[]` +
//     `tracks[]`) porte le gamertag. Le slot MIGRE aux reapparitions : la jointure se fait
//     donc slot x FRAME, jamais slot seul.
//  2. L'HORLOGE. Le film date ses paquets en microsecondes moteur. Le visionneur, lui,
//     affiche un temps ECOULE DEPUIS LE DEBUT DU FILM. Le manifeste du film
//     (`data/cache/film_manifests/<id8>.json`) publie `start_ms` par chunk, et le chunk 1 y
//     vaut 0 : c'est L'HORLOGE DU JEU LUI-MEME. La conversion retenue est donc
//     `msEcoule = (tsUS - tsUS(premier paquet du chunk 1)) / 1000`, et `TestR9Horloge` la
//     CONTROLE en confrontant, chunk par chunk, ce calcul au `start_ms` declare. Sans ce
//     controle, un creneau serait une promesse invérifiable.
//
// PIEGE HERITE DE R8, RECOPIE ICI EXPRES : `WorldObjectPrecision` est un GLOBAL DE PAQUET.
// Un instrument qui oublie `SetWorldObjectPrecisionFromLayout` rend 13 poses au lieu de 537
// sans lever la moindre erreur.
//
// GARDES : `R9_FILMS`, `R8_BOUNDS`, `R9_ARTIFACTS`, `R9_MANIFESTS`, `R9_IDS`. Aucune
// ecriture, aucune DuckDB, `CGO_ENABLED=0`. USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 R9_FILMS=<repo>/data/cache/film_chunks \
//	  R9_MANIFESTS=<repo>/data/cache/film_manifests \
//	  R9_ARTIFACTS=<repo>/data/cache/replays/halo_infinite \
//	  R8_BOUNDS=<wt>/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	  R9_IDS=8a485699 go test ./internal/analysis/filmdec/ \
//	  -run '^TestR9(Horloge|CreneauxPropulseur)$' -count=1 -timeout 60m -v

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const (
	r9FilmsEnv     = "R9_FILMS"
	r9ArtifactsEnv = "R9_ARTIFACTS"
	r9ManifestsEnv = "R9_MANIFESTS"
	r9IDsEnv       = "R9_IDS"
	// r9JGXUID : le xuid de l'utilisateur. Le Theater du jeu ne montre QUE les matchs ou il a
	// joue — un creneau pris dans un film ou il est absent serait inutilisable.
	r9JGXUID = "2533274823110022"
)

// --- Artefact de rejeu : la vue minimale qui porte l'IDENTITE ---

type r9RosterEntry struct {
	XUID string `json:"xuid"`
	Name string `json:"name"`
}

type r9TrackPoint struct {
	T int     `json:"t"`
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type r9TrackLite struct {
	Slot     uint32         `json:"slot"`
	Team     int            `json:"team"`
	XUID     string         `json:"xuid"`
	EndFrame int            `json:"endFrame"`
	Points   []r9TrackPoint `json:"points"`
}

type r9AbilityRead struct {
	T    int    `json:"t"`
	Slot uint32 `json:"slot"`
	R    int    `json:"r"`
}

type r9Art struct {
	ID              string
	MatchID         string                    `json:"matchId"`
	SchemaVersion   int                       `json:"schemaVersion"`
	OriginMs        int64                     `json:"originMs"`
	FrameIntervalMs int                       `json:"frameIntervalMs"`
	FrameCount      int                       `json:"frameCount"`
	Roster          []r9RosterEntry           `json:"roster"`
	Tracks          []r9TrackLite             `json:"tracks"`
	Abilities       []r9AbilityRead           `json:"abilities"`
	AbilityLabels   map[string]r8Label        `json:"abilityLabels"`
	Names           map[string]string         `json:"-"`
	BySlot          map[uint32][]*r9TrackLite `json:"-"`
}

// r9LoadArt lit l'artefact de rejeu du film `id`.
func r9LoadArt(t *testing.T, id string) *r9Art {
	t.Helper()
	dir := os.Getenv(r9ArtifactsEnv)
	if dir == "" {
		t.Skipf("%s absent : sans artefact, aucun gamertag — instrument saute", r9ArtifactsEnv)
	}
	raw, err := os.ReadFile(filepath.Join(dir, id+".json")) //nolint:gosec // chemin sous garde
	if err != nil {
		t.Fatalf("artefact %s illisible : %v", id, err)
	}
	var a r9Art
	if err := json.Unmarshal(raw, &a); err != nil {
		t.Fatalf("artefact %s hors schema : %v", id, err)
	}
	a.ID = id
	a.Names = map[string]string{}
	for _, r := range a.Roster {
		a.Names[r.XUID] = r.Name
	}
	a.BySlot = map[uint32][]*r9TrackLite{}
	for i := range a.Tracks {
		tr := &a.Tracks[i]
		if len(tr.Points) > 0 {
			a.BySlot[tr.Slot] = append(a.BySlot[tr.Slot], tr)
		}
	}
	return &a
}

// r9WhoAt rend le gamertag, le xuid et l'equipe du slot A CETTE FRAME. Le slot MIGRE aux
// reapparitions : la piste doit COUVRIR la frame, sans quoi on nommerait le joueur precedent.
func (a *r9Art) r9WhoAt(slot uint32, frame int) (name, xuid string, team int) {
	for _, tr := range a.BySlot[slot] {
		if frame < tr.Points[0].T || frame > tr.EndFrame {
			continue
		}
		n := a.Names[tr.XUID]
		if n == "" {
			n = "(inconnu)"
		}
		return n, tr.XUID, tr.Team
	}
	return "(hors piste)", "", -1
}

// r9RankOf rend le rang que la palette de CE film donne a la capacite nommee (`thruster`,
// `repulsor`), ou -1. La palette varie d'un film a l'autre : aucune constante en dur.
func (a *r9Art) r9RankOf(en string) int {
	for k, lab := range a.AbilityLabels {
		if !strings.EqualFold(lab.EN, en) {
			continue
		}
		n := 0
		for _, c := range k {
			if c < '0' || c > '9' {
				return -1
			}
			n = n*10 + int(c-'0')
		}
		return n
	}
	return -1
}

// --- L'HORLOGE, ET SON CONTROLE ---

type r9ManifestChunk struct {
	Index      int   `json:"index"`
	ChunkType  int   `json:"chunk_type"`
	StartMs    int64 `json:"start_ms"`
	DurationMs int64 `json:"duration_ms"`
}

type r9Manifest struct {
	Chunks []r9ManifestChunk `json:"chunks"`
}

// r9FirstPacketUS rend l'horodatage du premier paquet d'un chunk, et l'origine du film
// (premier paquet du chunk 1) quand `chunk == 1`.
func r9FirstPacketUS(dir string, chunk int) (uint64, bool) {
	raw, err := ReadFilmChunk(dir, chunk)
	if err != nil {
		return 0, false
	}
	pk := WalkPackets(raw)
	if len(pk) == 0 {
		return 0, false
	}
	return pk[0].TimestampUS, true
}

// r9MMSS met en forme un instant du visionneur. Les millisecondes sont TUES : un creneau se
// vise a la seconde, une precision affichee que la mesure ne tient pas serait un mensonge.
func r9MMSS(ms int64) string {
	if ms < 0 {
		return fmt.Sprintf("-%s", r9MMSS(-ms))
	}
	s := ms / 1000
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}

// TestR9Horloge CONTROLE la conversion « horodatage moteur -> temps du visionneur » contre le
// manifeste du film, qui est l'horloge du JEU et non la notre. Test peu couteux : il ne lit
// que le premier paquet de chaque chunk.
func TestR9Horloge(t *testing.T) {
	root := os.Getenv(r9ManifestsEnv)
	if root == "" {
		t.Skipf("%s absent : la conversion de temps ne peut pas etre controlee", r9ManifestsEnv)
	}
	for _, dir := range r9FilmDirs(t) {
		r9HorlogeOneFilm(t, root, dir)
	}
}

func r9HorlogeOneFilm(t *testing.T, manifRoot, dir string) {
	t.Helper()
	id := filepath.Base(dir)
	raw, err := os.ReadFile(filepath.Join(manifRoot, id+".json")) //nolint:gosec // sous garde
	if err != nil {
		t.Logf("%s : manifeste illisible (%v) — conversion NON controlee", id, err)
		return
	}
	var m r9Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Logf("%s : manifeste hors schema (%v)", id, err)
		return
	}
	origin, ok := r9FirstPacketUS(dir, 1)
	if !ok {
		t.Logf("%s : chunk 1 illisible", id)
		return
	}
	var worst int64
	var worstIdx, n int
	for _, c := range m.Chunks {
		if c.Index < 1 {
			continue
		}
		ts, ok := r9FirstPacketUS(dir, c.Index)
		if !ok {
			continue
		}
		n++
		d := (int64(ts)-int64(origin))/1000 - c.StartMs
		if d < 0 {
			d = -d
		}
		if d > worst {
			worst, worstIdx = d, c.Index
		}
	}
	t.Logf("%s : CONTROLE HORLOGE sur %d chunks — ecart max %d ms (chunk %d) ; "+
		"conversion retenue msEcoule = (tsUS - tsUS_chunk1) / 1000", id, n, worst, worstIdx)
	r9LogCarte(t, dir)
}

// r9LogCarte nomme la ou les cartes dont la quantification correspond a celle du film.
// PLUSIEURS NOMS EST UNE REPONSE VALIDE : des cartes de Forge partagent le meme canevas,
// donc les memes largeurs d'axe. Ne fait jamais tomber le test : c'est un confort de
// lecture, pas une mesure.
func r9LogCarte(t *testing.T, dir string) {
	t.Helper()
	path := os.Getenv("R8_BOUNDS")
	if path == "" {
		return
	}
	cat, err := LoadMapQuantCatalog(path)
	if err != nil {
		t.Logf("  catalogue de bornes illisible : %v", err)
		return
	}
	lay, _, err := DetectI0Layout(dir)
	if err != nil {
		t.Logf("  decoupage i0 illisible : %v", err)
		return
	}
	var names []string
	for name, e := range cat.Maps {
		if e.AxisWidths == lay.AxisW && e.Region == lay.Region {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	t.Logf("  carte(s) compatibles : %v", names)
}

// r9FilmDirs rend les dossiers de film a balayer. `R9_IDS` est OBLIGATOIRE : un film coute
// des dizaines de secondes de decodage, un balayage non borne serait un piege.
func r9FilmDirs(t *testing.T) []string {
	t.Helper()
	root := os.Getenv(r9FilmsEnv)
	if root == "" {
		t.Skipf("%s absent : instrument saute", r9FilmsEnv)
	}
	var out []string
	for _, s := range strings.Split(os.Getenv(r9IDsEnv), ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, filepath.Join(root, s))
		}
	}
	if len(out) == 0 {
		t.Fatalf("%s obligatoire avec %s (un film coute cher a decoder)", r9IDsEnv, r9FilmsEnv)
	}
	return out
}

// --- LES CRENEAUX DU PROPULSEUR ---

// r9Creneau est UNE impulsion prete a etre regardee.
type r9Creneau struct {
	Film    string
	MatchID string
	Name    string
	XUID    string
	Rank    int
	RankLab string
	// VieLab nomme la ou les capacites lues sur la MEME VIE, avant OU apres l'instant.
	// C'est une lecture PLUS FAIBLE que `RankLab` (elle n'exclut pas qu'un echange ait eu
	// lieu entre les deux), et elle est publiee a part pour cette raison : le canal i48
	// n'emet qu'environ une fois par vie, donc la lecture stricte laisse la moitie des
	// impulsions sans nom, ce qui n'est pas une information sur la capacite.
	VieLab string
	MS     int64
	Frame  int
	Peak   float64
	Slot   uint32
}

// r9RanksInLife rend les rangs DISTINCTS lus pour ce slot dans la vie qui couvre `at`,
// sans contrainte d'anteriorite. Complement declare de `r8RankInLife`, jamais son remplacant.
func r9RanksInLife(ranks []AbilityRank, lives map[uint32][]r8LifeSpan,
	slot uint32, at uint64) []int {
	var span r8LifeSpan
	found := false
	for _, l := range lives[slot] {
		if at+r8LifeGapUS >= l.from && at <= l.to+r8LifeGapUS {
			span, found = l, true
			break
		}
	}
	if !found {
		return nil
	}
	seen := map[int]bool{}
	var out []int
	for _, r := range ranks {
		if r.Slot != slot || r.TimestampUS < span.from || r.TimestampUS > span.to {
			continue
		}
		if !seen[int(r.Rank)] {
			seen[int(r.Rank)] = true
			out = append(out, int(r.Rank))
		}
	}
	sort.Ints(out)
	return out
}

func TestR9CreneauxPropulseur(t *testing.T) {
	for _, dir := range r9FilmDirs(t) {
		r9CreneauxOneFilm(t, dir)
	}
}

func r9CreneauxOneFilm(t *testing.T, dir string) {
	t.Helper()
	id := filepath.Base(dir)
	art := r9LoadArt(t, id)
	entry := r8MapEntry(t, dir)
	wr := entry.Range()
	release := LockProcessDecode()
	defer release()
	saved := WorldObjectPrecision
	SetWorldObjectPrecisionFromLayout(entry.Layout())
	defer func() { WorldObjectPrecision = saved }()

	origin, ok := r9FirstPacketUS(dir, 1)
	if !ok {
		t.Fatalf("%s : chunk 1 illisible, aucune origine d'horloge", id)
	}
	s := r8MobResolve(t, dir)
	opt := DefaultScanFilmOptions()
	opt.WorldRange = &wr
	pos, err := ScanFilmBipedPositions(dir, opt)
	if err != nil {
		t.Fatalf("positions illisibles : %v", err)
	}
	speeds := r8BuildSpeeds(pos)
	ranks, _, err := ScanFilmAbilityRanks(dir)
	if err != nil {
		t.Logf("%s : rangs de capacite illisibles : %v", id, err)
	}
	i57, i59 := r8ScanTags(s)
	lives := r8Lives(speeds)
	rThr := art.r9RankOf("thruster")
	t.Logf("%s (%s) : i57=%d i59=%d ranks=%d | originMs=%d pas=%d ms | rang propulseur=%d",
		id, art.MatchID, len(i57), len(i59), len(ranks), art.OriginMs, art.FrameIntervalMs, rThr)
	cre := r9Impulsions(art, i57, i59, ranks, lives, speeds, origin)
	r9LogCreneaux(t, id, cre, rThr)
}

// r9Impulsions replie les lectures `tag == 1` d'i57 ET d'i59 en EPISODES par slot, puis les
// nomme. i57 et i59 sont co-transmis : une meme impulsion apparait souvent dans les deux, et
// les replier ensemble evite de compter deux fois le meme geste.
func r9Impulsions(art *r9Art, i57, i59 []r8TagRead, ranks []AbilityRank,
	lives map[uint32][]r8LifeSpan, speeds r8SpeedIndex, origin uint64) []r9Creneau {
	type ev struct {
		slot uint32
		ts   uint64
	}
	var all []ev
	for _, r := range i57 {
		if r.Tag == 1 {
			all = append(all, ev{r.Slot, r.TSUS})
		}
	}
	for _, r := range i59 {
		if r.Tag == 1 {
			all = append(all, ev{r.Slot, r.TSUS})
		}
	}
	sort.Slice(all, func(a, b int) bool {
		if all[a].slot != all[b].slot {
			return all[a].slot < all[b].slot
		}
		return all[a].ts < all[b].ts
	})
	last := map[uint32]uint64{}
	var out []r9Creneau
	for _, e := range all {
		if p, ok := last[e.slot]; ok && e.ts-p <= r8MobEpisodeGapUS {
			last[e.slot] = e.ts
			continue
		}
		last[e.slot] = e.ts
		out = append(out, r9Creneau1(art, e.slot, e.ts, ranks, lives, speeds, origin))
	}
	sort.Slice(out, func(a, b int) bool { return out[a].MS < out[b].MS })
	return out
}

// r9Creneau1 construit UN creneau : instant du visionneur, acteur nomme, rang porte, pic.
func r9Creneau1(art *r9Art, slot uint32, ts uint64, ranks []AbilityRank,
	lives map[uint32][]r8LifeSpan, speeds r8SpeedIndex, origin uint64) r9Creneau {
	ms := (int64(ts) - int64(origin)) / 1000
	frame := 0
	if art.FrameIntervalMs > 0 {
		frame = int((ms - art.OriginMs) / int64(art.FrameIntervalMs))
	}
	name, xuid, _ := art.r9WhoAt(slot, frame)
	rank := r8RankInLife(ranks, lives, slot, ts)
	peak, _ := speeds.peak(slot, ts, r8PeakWindowUS)
	var vie []string
	for _, r := range r9RanksInLife(ranks, lives, slot, ts) {
		vie = append(vie, art.r9LabelOf(r))
	}
	vieLab := "(aucune lecture)"
	if len(vie) > 0 {
		vieLab = strings.Join(vie, "+")
	}
	return r9Creneau{Film: art.ID, MatchID: art.MatchID, Name: name, XUID: xuid,
		Rank: rank, RankLab: art.r9LabelOf(rank), VieLab: vieLab,
		MS: ms, Frame: frame, Peak: peak, Slot: slot}
}

// r9LabelOf nomme un rang de capacite dans la langue de l'artefact, ou le dit non lu.
func (a *r9Art) r9LabelOf(rank int) string {
	if rank < 0 {
		return "(non lu)"
	}
	if l, ok := a.AbilityLabels[fmt.Sprintf("%d", rank)]; ok && l.FR != "" {
		return l.FR
	}
	return fmt.Sprintf("rang %d", rank)
}

// r9LogCreneaux publie les creneaux, JGtm EN TETE — ce sont les seuls que l'utilisateur
// retrouvera de memoire, et le Theater ne lui montre que ses propres matchs de toute facon.
func r9LogCreneaux(t *testing.T, id string, cre []r9Creneau, rThr int) {
	t.Helper()
	var mine, autres int
	t.Logf("  CRENEAUX %s — impulsions tag==1 (episodes), %d au total", id, len(cre))
	t.Logf("    %-7s %-7s %-17s %6s %-16s %-16s %s",
		"acteur", "mm:ss", "gamertag", "pic", "rang AVANT", "rangs de la vie", "frame")
	for _, c := range cre {
		who := "-"
		if c.XUID == r9JGXUID {
			who, mine = "JGtm", mine+1
		} else {
			autres++
		}
		flag := ""
		if rThr >= 0 && c.Rank >= 0 && c.Rank != rThr {
			flag = "  [rang lu != propulseur]"
		}
		t.Logf("    %-7s %-7s %-17s %6.2f %-16s %-16s %d%s",
			who, r9MMSS(c.MS), c.Name, c.Peak, c.RankLab, c.VieLab, c.Frame, flag)
	}
	t.Logf("    -> %d impulsions de JGtm, %d des autres joueurs", mine, autres)
}
