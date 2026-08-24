package replay

// lettres_ordre_research_test.go — INSTRUMENT DE MESURE du lot « lettres A/B/C des bases »
// (phase 0.2 de `.ai/V7.5/replay2d/PLAN_LETTRES_BASES_FALLBACK.md`, 2026-08-24).
//
// # LA QUESTION, ET RIEN D'AUTRE
//
// Le fallback des lettres repose sur UNE hypothese : sur une meme carte, l'ordre des SLOTS `ti=13`
// designe toujours les MEMES zones, dans le meme ordre. Si c'est vrai, ranger les zones par slot
// croissant donne une permutation stable — et une permutation stable suffit a dire A, B, C.
// Si c'est faux, le fallback tombe (et le plan dit STOP, pas bricolage).
//
// # L'ORACLE EST CELUI DE LA PRODUCTION, ET C'EST LE POINT
//
// L'appariement mesure ici est `pairGaugeSlots` — LA fonction que le calque appelle, sans copie ni
// variante : captures nommees du statborg, posees sur l'axe du rejeu, attribuees geometriquement
// (`AttributeZones`, 5 m), puis vote modal sur le sommet de rampe. Mesurer autre chose que ce que
// le calque publiera n'aurait pas de valeur de preuve.
//
// UNE PREMIERE ECRITURE A ETE ABANDONNEE, et le negatif merite d'etre garde : elle remplacait
// l'oracle nomme par la GRAPPE des positions (la methode du volet colline), pour n'avoir besoin
// ni du roster ni des lignes de match. Sur les trois films de Vagabond, elle retrouve bien les
// trois slots canoniques (1532 -> zone 1, 1537 -> zone 2, 1542 -> zone 0, comme la phase 2a), mais
// elle donne AUSSI une zone au quatrieme bloc de slots (1545-1547, l'objet de MODE : ses rampes
// suivent toutes les captures de la carte) et le fait gagner l'election sur un film sur trois. La
// grappe ne separe pas une prise d'un passage ; les captures nommees, si.
//
// # UN FILM = UN PROCESSUS, ET C'EST UNE LECON PAYEE (2026-08-24)
//
// La premiere version de cet instrument BOUCLAIT sur le corpus dans un seul processus de test.
// Laissee orpheline par un redemarrage de l'hote, elle est montee a 18,4 Gio residents et 95 Gio
// de commit, et a mis la machine a genoux. Trois causes, toutes corrigees ici :
//
//	la BOUCLE        chaque film laissait son pic dans le MEME processus — le tas ne revient pas
//	                 a l'OS entre deux films. Desormais : `LETTRES_FILM` designe UN film, la
//	                 comparaison inter-match se fait hors film (cf. TestLettresOrdreStabilite).
//	la BOMBE         `objectiveevents.NamedEvents` -> `incrementTimes` emet UN evenement PAR UNITE
//	                 de compteur (`for ; prev < p.Value; prev++`) : une emission aberrante sur un
//	                 composant fait croitre la sortie en centaines de millions d'entrees (registre
//	                 des reports : OOM ~26 Gio sur `51101d1d`). C'est du code de PRODUCTION, hors
//	                 perimetre de ce lot — l'instrument ne le corrige pas, il s'en PROTEGE par la
//	                 SENTINELLE ci-dessous, seule barriere possible puisque la croissance a lieu
//	                 A L'INTERIEUR de l'appel.
//	la RETENTION     positions, lectures `ti=13` et evenements nommes vivaient tous en meme temps.
//	                 Chaque structure est desormais LIBEREE des qu'elle est consommee, et chacune
//	                 porte un plafond explicite avec erreur franche.
//
// # LES LIGNES DE MATCH SONT GELEES, AUCUNE BASE N'EST OUVERTE
//
// La carte, l'identifiant de carte, la variante et le roster sont relus dans les exports
// versionnes du registre (`oracle_lotA*.tsv`, `oracle_lotA*_participants.tsv`) — meme convention
// que la phase 2a. Le paquet `replay` n'ouvre aucune DuckDB, et un serveur tient de toute facon la
// base en RW.
//
// MESURE SEULEMENT : aucun fichier de production n'est touche par ce fichier.
//
// USAGE (depuis apps/go-api, UN FILM PAR PROCESSUS, en avant-plan) :
//
//	$env:GOMEMLIMIT='4GiB'; $env:CGO_ENABLED='0'
//	$env:LETTRES_OUT="<worktree>/.ai/V7.5/replay2d/registre_film/lotLettres"
//	$env:LETTRES_FILM="<mainrepo>/data/cache/film_chunks/7344d24f"
//	go test -count=1 -run TestLettresOrdreSlots -v -timeout 10m ./internal/analysis/replay/
//
// puis, une fois les films mesures (aucun film relu, une fraction de seconde) :
//
//	go test -count=1 -run TestLettresOrdreStabilite -v ./internal/analysis/replay/

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/analysis/replay/mapvar"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

const (
	lettresFilmEnv = "LETTRES_FILM"
	lettresOutEnv  = "LETTRES_OUT"
)

// PLAFONDS DE L'INSTRUMENT. Ils ne sont pas des reglages de confort : chacun ferme une des trois
// causes de l'incident du 2026-08-24, et un depassement est une ERREUR FRANCHE, jamais un
// elargissement.
const (
	// lettresHeapMaxBytes : la SENTINELLE. Au-dela, le processus s'arrete NET (code 9). C'est la
	// seule barriere contre une croissance interne a un appel de production (cf. l'en-tete) :
	// aucun plafond de slice ne peut l'attraper, puisque la slice n'est pas la notre. Trois Gio
	// laissent une marge confortable au-dessus du regime mesure (150 a 250 Mio par film) et
	// restent sous le `GOMEMLIMIT` de 4 Gio que l'appelant pose.
	lettresHeapMaxBytes = 3 << 30
	// lettresHeapSampleMS : periode d'echantillonnage de la sentinelle.
	lettresHeapSampleMS = 200
	// lettresMaxPositions : un film de dix minutes a huit joueurs rend ~1 a 2 millions de points.
	lettresMaxPositions = 8_000_000
	// lettresMaxReads : les lectures `ti=13` d'un film mesure comptent quelques dizaines de
	// milliers d'entrees ; le plafond est deux ordres de grandeur au-dessus.
	lettresMaxReads = 4_000_000
	// lettresMaxNamed : les evenements nommes d'un match se comptent en milliers. Ce plafond ne
	// PEUT PAS arreter la bombe (elle croit dans l'appel), il la CONSTATE apres coup quand elle
	// reste sous la sentinelle — et il refuse alors de continuer sur une donnee absurde.
	lettresMaxNamed = 2_000_000
)

// lettresFilm est l'identite d'un film mesure, relue des exports geles du registre.
type lettresFilm struct {
	short, carte, mapID, variant string
}

// lettresMesure est le resultat d'un film : la carte zone -> slot de jauge que la production
// construirait, et de quoi la juger.
type lettresMesure struct {
	film      lettresFilm
	catalog   int
	gauge     map[int]uint32
	unpaired  int
	owner     map[int]uint32
	captures  int
	posees    int
	attribues int
	frames    int
	tracks    int
	duration  time.Duration
	heapPeak  uint64
}

// TestLettresOrdreSlots mesure UN film : la carte zone -> slot de jauge, et la permutation
// zone -> rang de lettre qui en decoule.
func TestLettresOrdreSlots(t *testing.T) {
	dir := os.Getenv(lettresFilmEnv)
	if dir == "" {
		t.Skipf("mesure non demandee : %s vide", lettresFilmEnv)
	}
	outDir := os.Getenv(lettresOutEnv)
	if outDir == "" {
		t.Fatalf("%s est requis", lettresOutEnv)
	}
	pic, stop := lettresSentinelle(t)
	defer stop()
	film := lettresIdentite(t, filepath.Base(dir))
	m := lettresMesureFilm(t, dir, film)
	m.heapPeak = pic.Load()
	lettresLogFilm(t, m)
	lettresWrite(t, outDir, film.short+"_lettres.tsv", lettresLigneFilm(m))
}

// lettresSentinelle arme la garde memoire : elle echantillonne le tas, RETIENT LE PIC et ARRETE
// LE PROCESSUS au premier depassement, avant que la machine ne souffre.
//
// POURQUOI `os.Exit` ET PAS `t.Fatal`. `t.Fatal` ne peut etre appele que depuis la goroutine du
// test ; la croissance, elle, a lieu dans un appel de production qui ne rend pas la main. Le seul
// arret qui vaille est donc brutal — et c'est exactement ce qu'on veut : mieux vaut un code de
// sortie a expliquer qu'un poste bloque. Le film en cours est consigne sur stderr avant de sortir.
//
// LE PIC EST RENDU A L'APPELANT parce qu'un tas lu A LA FIN ne dit rien du regime : le ramasse-
// miettes a deja repris les positions et les lectures `ti=13`. C'est le maximum echantillonne qui
// se compare au plafond, et c'est lui que la mesure publie.
func lettresSentinelle(t *testing.T) (*atomic.Uint64, func()) {
	t.Helper()
	pic := &atomic.Uint64{}
	done := make(chan struct{})
	go func() {
		var ms runtime.MemStats
		tick := time.NewTicker(lettresHeapSampleMS * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-done:
				return
			case <-tick.C:
				runtime.ReadMemStats(&ms)
				if ms.HeapAlloc > pic.Load() {
					pic.Store(ms.HeapAlloc)
				}
				if ms.HeapAlloc <= lettresHeapMaxBytes {
					continue
				}
				fmt.Fprintf(os.Stderr,
					"SENTINELLE MEMOIRE : tas %d Mio > plafond %d Mio sur %s — ARRET NET\n",
					ms.HeapAlloc>>20, uint64(lettresHeapMaxBytes)>>20, os.Getenv(lettresFilmEnv))
				os.Exit(9)
			}
		}
	}()
	return pic, func() { close(done) }
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

// lettresMesureFilm decode UN film et rend la carte zone -> slot de jauge de la production.
//
// L'ORDRE DES ETAPES EST CELUI QUI MINIMISE LE PIC : chaque grande structure est consommee puis
// LIBEREE avant que la suivante ne soit allouee. Les positions sortent du decodage, deviennent des
// trajectoires decimees, et la slice brute est laissee au ramasse-miettes ; les lectures `ti=13`
// deviennent des series par slot et disparaissent de meme.
func lettresMesureFilm(t *testing.T, dir string, film lettresFilm) lettresMesure {
	t.Helper()
	start := time.Now()
	if !lettresEstZonesSimultanees(film.variant) {
		t.Skipf("film %s : variante %q — hors des modes a zones simultanees, rien a mesurer",
			film.short, film.variant)
	}
	zones := lettresZones(t, film.mapID)
	quant := lettresQuant(t, film.carte)
	release := filmdec.LockProcessDecode()
	defer release()
	doc, origin := lettresDoc(t, dir, film, quant)
	m := lettresMesure{film: film, catalog: len(zones), frames: doc.FrameCount,
		tracks: len(doc.Tracks)}
	c := zoneCtx{origin: origin, step: uint64(doc.FrameIntervalMS) * 1000,
		frames: doc.FrameCount, intervalMS: doc.FrameIntervalMS, tracks: doc.Tracks,
		matchID: doc.MatchID}
	ser := lettresSeries(t, dir, c)
	pairs := lettresPaires(t, dir, film, doc, zoneCatalogOf(zones), &m)
	win := zoneWindowFrames(c.intervalMS)
	m.gauge, m.unpaired = pairGaugeSlots(zoneRampsOf(ser), pairs, win)
	m.owner = pairOwnerSlots(ser, pairs, lettresTeams(t, film.short), win)
	m.duration = time.Since(start)
	return m
}

// lettresEstZonesSimultanees dit si la variante est un mode a BASES SIMULTANEES — le seul
// perimetre du lot. Un mode a colline, a drapeau ou a balle n'a pas de lettre, et son film ne doit
// meme pas etre decode (c'est aussi ce qui tient l'instrument a l'ecart des films dont les
// evenements nommes explosent : la bombe consignee a frappe un CTF).
func lettresEstZonesSimultanees(variant string) bool {
	return objectiveevents.ObjectiveTypeOf(variant) == objectiveevents.ObjectiveTypeZone
}

// lettresDoc assemble le document minimal dont l'appariement a besoin : positions en metres, vies
// nommees (le pont xuid des trajectoires) et origine d'horloge. Rend aussi l'origine de l'axe des
// frames, LA SEULE chose que la slice brute de positions laisse derriere elle.
func lettresDoc(t *testing.T, dir string, film lettresFilm,
	quant filmdec.MapQuantEntry,
) (ReplayDocument, uint64) {
	t.Helper()
	world := quant.Range()
	scan := filmdec.DefaultScanFilmOptions()
	scan.WorldRange = &world
	pos, err := filmdec.ScanFilmBipedPositions(dir, scan)
	switch {
	case err != nil:
		t.Fatalf("film %s : positions illisibles : %v", film.short, err)
	case len(pos) == 0:
		t.Skipf("film %s : aucune position — rien a mesurer", film.short)
	case len(pos) > lettresMaxPositions:
		t.Fatalf("film %s : %d positions, plafond %d — donnee absurde ou plafond a revoir"+
			" (jamais a elargir sans mesure)", film.short, len(pos), lettresMaxPositions)
	}
	origin := pos[0].TimestampUS
	for _, p := range pos {
		if p.TimestampUS < origin {
			origin = p.TimestampUS
		}
	}
	opt := lettresOptions(t, dir, film)
	doc := BuildFromPositions(film.short, title.DefaultSlug, pos, nil, opt)
	return doc, origin
}

// lettresOptions monte le fil des morts, le pont d'index et l'origine d'horloge.
func lettresOptions(t *testing.T, dir string, film lettresFilm) Options {
	t.Helper()
	var opt Options
	deaths, err := ScanFilmDeaths(dir)
	if err != nil {
		t.Fatalf("film %s : fil des morts illisible : %v", film.short, err)
	}
	opt.Deaths = deaths
	if len(deaths) > 0 {
		if idx, err := ScanFilmPlayerIndices(dir, rosterFromDeaths(deaths)); err == nil {
			opt.PlayerIndices, _ = injectiveOrEmpty(idx)
		}
	}
	clockUS, err := ScanFilmClockOrigin(dir)
	if err != nil {
		t.Skipf("film %s : origine d'horloge illisible (%v) — les captures ne sont pas posables",
			film.short, err)
	}
	opt.FilmClockOriginUS = clockUS
	return opt
}

// lettresSeries balaye `ti=13` et rend les SERIES par slot. La slice de lectures ne sort pas de
// cette fonction : elle est consommee ici et laissee au ramasse-miettes.
func lettresSeries(t *testing.T, dir string, c zoneCtx) zoneSeries {
	t.Helper()
	sc, err := filmdec.ScanFilmManagedProperties(dir)
	if err != nil {
		t.Skipf("proprietes ti=13 illisibles (%s) : %v", dir, err)
	}
	if len(sc.Reads) > lettresMaxReads {
		t.Fatalf("%d lectures ti=13, plafond %d — donnee absurde ou plafond a revoir", len(sc.Reads),
			lettresMaxReads)
	}
	return zoneSeriesOf(sc.Reads, c)
}

// lettresPaires rejoue la chaine de l'oracle : captures nommees -> axe du rejeu -> attribution
// geometrique. Les evenements nommes ne sortent pas de cette fonction.
func lettresPaires(t *testing.T, dir string, film lettresFilm, doc ReplayDocument, cat []Zone,
	m *lettresMesure,
) []zonePair {
	t.Helper()
	src, ok, err := filmcache.OpenChunkDir(dir)
	if err != nil || !ok {
		t.Fatalf("film %s : manifeste illisible (ok=%v err=%v)", film.short, ok, err)
	}
	caps := lettresCaptures(t, src, lettresRoster(t, film.short))
	m.captures = len(caps)
	actions := make([]ObjectiveAction, 0, len(caps))
	for _, e := range caps {
		f, ok := p2aFrameOf(doc, e.TimeMS)
		if !ok {
			continue
		}
		actions = append(actions, ObjectiveAction{T: f, XUID: e.XUID, Stat: e.Stat, TimeMS: e.TimeMS})
	}
	m.posees = len(actions)
	att, _ := AttributeZones(actions, doc.Tracks, cat,
		AttributeOptions{MaxDistanceM: zoneCaptureDistanceM})
	pairs := zonePairsOf(att)
	m.attribues = len(pairs)
	return pairs
}

// lettresCaptures rend les captures de zone nommees et identifiees par xuid.
func lettresCaptures(t *testing.T, src objectiveevents.FilmSource,
	roster []p2aPlayer,
) []objectiveevents.IdentifiedEvent {
	t.Helper()
	lines := make([]objectiveevents.PlayerLine, 0, len(roster))
	for _, p := range roster {
		lines = append(lines, objectiveevents.PlayerLine{
			XUID: p.XUID, Kills: p.Kills, Deaths: p.Deaths, Assists: p.Assists,
		})
	}
	named := objectiveevents.NamedEvents(src, objectiveevents.ObjectiveTypeZone)
	if len(named) > lettresMaxNamed {
		t.Fatalf("%d evenements nommes, plafond %d — la bombe `incrementTimes` du registre",
			len(named), lettresMaxNamed)
	}
	identity := objectiveevents.SlotIdentity(src, lines)
	out := make([]objectiveevents.IdentifiedEvent, 0, 256)
	for _, e := range objectiveevents.IdentifyNamedEvents(named, identity) {
		if e.Stat == objectiveevents.StatZoneCaptures || e.Stat == objectiveevents.StatZoneSecures {
			out = append(out, e)
		}
	}
	return out
}

// lettresTeams rend le roster xuid -> camp du film.
func lettresTeams(t *testing.T, short string) map[string]int {
	t.Helper()
	out := map[string]int{}
	for _, p := range lettresRoster(t, short) {
		out[p.XUID] = p.Team
	}
	return out
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

// lettresPermutation rend la permutation zone -> rang de lettre : les zones appariees, triees par
// NUMERO DE SLOT croissant, chacune prenant son rang.
//
// LA BIJECTION EST EXIGEE, et c'est la garde du fallback : tant que les zones appariees ne
// couvrent pas TOUT le catalogue de la carte, une zone muette decalerait les lettres des
// suivantes. Une table incomplete ne rend donc aucune lettre.
func lettresPermutation(m lettresMesure) (map[int]int, bool) {
	refs := make([]int, 0, len(m.gauge))
	for ref := range m.gauge {
		refs = append(refs, ref)
	}
	sort.Slice(refs, func(i, j int) bool { return m.gauge[refs[i]] < m.gauge[refs[j]] })
	out := make(map[int]int, len(refs))
	for i, ref := range refs {
		out[ref] = i
	}
	return out, len(out) == m.catalog && m.catalog > 0
}

// lettresDescPermutation ecrit une permutation sous la forme `z0=A,z1=C,...`.
func lettresDescPermutation(p map[int]int, full bool) string {
	if !full {
		return "INCOMPLETE"
	}
	refs := make([]int, 0, len(p))
	for r := range p {
		refs = append(refs, r)
	}
	sort.Ints(refs)
	parts := make([]string, 0, len(refs))
	for _, r := range refs {
		parts = append(parts, fmt.Sprintf("z%d=%c", r, 'A'+rune(p[r])))
	}
	return strings.Join(parts, ",")
}

// lettresDescSlots ecrit la carte zone -> slot d'un film, slot croissant.
func lettresDescSlots(m lettresMesure) string {
	refs := make([]int, 0, len(m.gauge))
	for ref := range m.gauge {
		refs = append(refs, ref)
	}
	sort.Slice(refs, func(i, j int) bool { return m.gauge[refs[i]] < m.gauge[refs[j]] })
	parts := make([]string, 0, len(refs))
	for _, ref := range refs {
		p := fmt.Sprintf("%d>z%d", m.gauge[ref], ref)
		if o, ok := m.owner[ref]; ok {
			p += fmt.Sprintf("(p%d)", o)
		} else {
			p += "(sans_proprietaire)"
		}
		parts = append(parts, p)
	}
	return strings.Join(parts, " ")
}

// lettresLigneFilm rend la ligne TSV d'un film.
func lettresLigneFilm(m lettresMesure) string {
	p, full := lettresPermutation(m)
	return fmt.Sprintf("film\t%s\t%s\t%s\tcatalogue\t%d\tapparies\t%d\tnon_apparies\t%d"+
		"\tproprietaires\t%d\tcaptures\t%d\tposees\t%d\tattribuees\t%d\tframes\t%d\ttrajectoires\t%d"+
		"\tduree_s\t%.0f\ttas_Mio\t%d\tperm\t%s\tslots\t%s\n",
		m.film.short, m.film.carte, m.film.mapID, m.catalog, len(m.gauge), m.unpaired,
		len(m.owner), m.captures, m.posees, m.attribues, m.frames, m.tracks,
		m.duration.Seconds(), m.heapPeak>>20, lettresDescPermutation(p, full), lettresDescSlots(m))
}

// lettresLogFilm journalise la mesure d'un film.
func lettresLogFilm(t *testing.T, m lettresMesure) {
	t.Helper()
	p, full := lettresPermutation(m)
	t.Logf("FILM %s (%s) — catalogue %d, apparies %d (+%d ecartes), proprietaires %d,"+
		" captures %d/%d/%d, %.0f s, tas %d Mio · %s · %s",
		m.film.short, m.film.carte, m.catalog, len(m.gauge), m.unpaired, len(m.owner),
		m.attribues, m.posees, m.captures, m.duration.Seconds(), m.heapPeak>>20,
		lettresDescSlots(m), lettresDescPermutation(p, full))
}

// lettresZones rend les zones de Bastion de la carte, dans l'ordre SERVI (rang spatial du role) —
// le meme que `replaybuild.matchZones` pour un mode a une seule famille de zone.
func lettresZones(t *testing.T, mapID string) []Zone {
	t.Helper()
	cat, err := LoadMapObjectives(filepath.Join(p2aRefDir(t), "map_objectives.json"))
	if err != nil {
		t.Fatalf("catalogue d'objectifs illisible : %v", err)
	}
	e, err := cat.Lookup(mapID)
	if err != nil {
		t.Skipf("carte %s absente du catalogue de formes — appariement geometrique impossible", mapID)
	}
	z := e.ZonesOfRole(mapvar.RoleStrongholdZone).Zones
	if len(z) == 0 {
		t.Skipf("carte %s sans zone de Bastion au catalogue — rien a apparier", mapID)
	}
	return z
}

// lettresQuant rend les bornes de quantification de la carte.
func lettresQuant(t *testing.T, carte string) filmdec.MapQuantEntry {
	t.Helper()
	cat, err := filmdec.LoadMapQuantCatalog(filepath.Join(p2aRefDir(t), "map_quant_bounds.json"))
	if err != nil {
		t.Fatalf("catalogue de bornes illisible : %v", err)
	}
	e, err := cat.Lookup(carte)
	if err != nil {
		t.Skipf("bornes de quantification de %q absentes — aucune distance n'a de sens", carte)
	}
	return e
}

// lettresWrite ecrit le TSV de mesure d'un film.
func lettresWrite(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creation de %s : %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("ecriture de %s : %v", name, err)
	}
}

// TestLettresOrdreStabilite compare les permutations des films d'une MEME carte.
//
// CE TEST NE DECODE AUCUN FILM — il relit les TSV produits film par film, donc il tourne en une
// fraction de seconde et ne peut rien faire exploser. Meme patron que
// `TestZoneEtatPhase2aStabilite` : la comparaison inter-match n'a aucune raison de vivre dans le
// processus qui decode.
func TestLettresOrdreStabilite(t *testing.T) {
	dir := os.Getenv(lettresOutEnv)
	if dir == "" {
		t.Skipf("comparaison non demandee : %s vide", lettresOutEnv)
	}
	byMap := lettresLitMesures(t, dir)
	if len(byMap) == 0 {
		t.Skipf("aucune mesure sous %s — jouer TestLettresOrdreSlots film par film d'abord", dir)
	}
	ids := make([]string, 0, len(byMap))
	for k := range byMap {
		ids = append(ids, k)
	}
	sort.Strings(ids)
	comparees, stables := 0, 0
	for _, id := range ids {
		rows := byMap[id]
		if len(rows) < 2 {
			t.Logf("  CARTE %-14s : un seul film mesure (%s) — non comparable", rows[0].carte,
				rows[0].short)
			continue
		}
		comparees++
		if lettresMemePermutation(t, rows) {
			stables++
		}
	}
	t.Logf("VERDICT — %d cartes a >= 2 films mesures, %d rendent la MEME permutation", comparees,
		stables)
	if comparees > 0 && stables < comparees {
		t.Errorf("%d carte(s) INSTABLE(S) : le fallback des lettres ne tient pas en l'etat",
			comparees-stables)
	}
}

// lettresLigne est une ligne de mesure relue d'un TSV.
type lettresLigne struct {
	short, carte, mapID, perm, slots string
}

// lettresLitMesures relit toutes les mesures d'un repertoire, groupees par carte.
func lettresLitMesures(t *testing.T, dir string) map[string][]lettresLigne {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("repertoire de mesures illisible (%s) : %v", dir, err)
	}
	out := map[string][]lettresLigne{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), "_lettres.tsv") {
			continue
		}
		blob, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("mesure illisible (%s) : %v", e.Name(), err)
		}
		for _, line := range strings.Split(string(blob), "\n") {
			f := strings.Split(strings.TrimRight(line, "\r"), "\t")
			if len(f) < 28 || f[0] != "film" {
				continue
			}
			out[f[3]] = append(out[f[3]],
				lettresLigne{short: f[1], carte: f[2], mapID: f[3], perm: f[27], slots: f[29]})
		}
	}
	for id := range out {
		rows := out[id]
		sort.Slice(rows, func(i, j int) bool { return rows[i].short < rows[j].short })
	}
	return out
}

// lettresMemePermutation dit si tous les films COMPLETS d'une carte rendent la meme permutation.
func lettresMemePermutation(t *testing.T, rows []lettresLigne) bool {
	t.Helper()
	ref, refShort, complets, identiques := "", "", 0, 0
	for _, r := range rows {
		if r.perm == "INCOMPLETE" {
			t.Logf("    %s %-10s : permutation INCOMPLETE (aucune lettre publiee) · %s", r.carte,
				r.short, r.slots)
			continue
		}
		complets++
		switch {
		case ref == "":
			ref, refShort, identiques = r.perm, r.short, 1
		case r.perm == ref:
			identiques++
		default:
			t.Errorf("    %s %s : permutation %s, mais %s rend %s — ORDRE INSTABLE", r.carte,
				r.short, r.perm, refShort, ref)
		}
		t.Logf("    %s %-10s : %s · %s", r.carte, r.short, r.perm, r.slots)
	}
	verdict := complets >= 2 && identiques == complets
	t.Logf("  CARTE %-14s : %d films, %d permutations completes, %d identiques — stable=%v",
		rows[0].carte, len(rows), complets, identiques, verdict)
	return verdict
}
