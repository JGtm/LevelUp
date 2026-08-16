package replaybuild

// grenade_corpus_test.go — LE RACCORD DES DEUX LECTURES D'UN MEME MATCH, pour les instruments
// de la question « le type d'une grenade qui tue se lit-il dans le LANCER ». Plomberie
// partagee ; les mesures elles-memes vivent dans `grenade_join_corpus_test.go` (phases 0 et 1)
// et `grenade_ambigu_sweep_test.go` (denominateur elargi).
//
// # LES DEUX SOURCES, ET POURQUOI ELLES SE RENCONTRENT ICI
//
// Le TAG de la mort se lit dans le film (`film/killsource`, dead-state de la victime) ; le
// LANCER se lit dans l'artefact de rejeu (`analysis/replay`, `doc.grenades`). `replaybuild` est
// la seule couche qui compose deja les deux — c'est la meme raison qui y a mis le typage des
// morts neutres.
//
// # LE RACCORD DES HORLOGES N'EST PAS UN APPARIEMENT
//
// `Kill.TimeMS` date la mort sur l'horloge du fil des eliminations (ms depuis le debut du
// film). `doc.originMs` EST l'instant de la frame 0 sur cette meme horloge (cf. origin.go, deux
// en-tetes de paquet soustraits, temoin independant a moins de 100 ms). Un lancer publie a la
// frame `t` vaut donc `t * frameIntervalMs`, et la mort vaut `TimeMS - originMs` : une
// soustraction, aucune estimation. Un film SANS origine publiee est ECARTE — poser les deux
// axes l'un sur l'autre sans elle serait exactement la faute que ces instruments mesurent.
//
// # GARDES
//
// `REPLAY_CORPUS` pointe le dossier d'artefacts (data/cache/replays/halo_infinite) ; sans lui
// les tests SAUTENT — le corpus n'est pas versionne, la CI ne l'a pas. Le cache de films se
// derive de la racine du depot (data/cache/film_chunks) ; son absence fait sauter aussi.
// Aucune base n'est ouverte, ni en lecture ni en ecriture.

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/damagetag"
	"levelup/go-api/internal/games/halo_infinite/film/killsource"
)

// ggglEntreeRe extrait l'entree de la liste des grenades du champ `detail` de labels.tsv.
// MEME expression que killicon.ggglRe, et pour la meme raison : la forme AMBIGUE
// (« entrees `gggl` atteintes : 0+1 sur 4 ») ne matche pas, donc ne rend aucun rang.
var ggglEntreeRe = regexp.MustCompile(`gggl entree (\d+)/`)

// mortGrenade : une mort dont la source de degat est de classe GRENADE, ramenee sur l'axe des
// frames du rejeu.
type mortGrenade struct {
	film string
	tag  uint32
	// rangAttendu : le rang de type lu dans l'oracle (entree `gggl`), -1 quand l'etiquette ne
	// le donne pas (statut AMBIGU) — c'est exactement la population que ce chantier vise.
	rangAttendu int
	// lanceurIdx : l'index de film de CELUI A QUI LA GRENADE APPARTIENT. Pour une mort dont
	// les deux verites divergent, la source appartient a la VICTIME : joindre sur le credit du
	// jeu chercherait le lancer de quelqu'un qui n'a rien lance. -1 = aucun pont.
	lanceurIdx int
	// tMS : l'instant de la mort sur l'axe des frames du rejeu, en millisecondes.
	tMS int
	// revendiquee : la mort porte un tueur au kill-feed (par opposition a UnclaimedDeath).
	revendiquee bool
	// diverge : la source appartenait a la victime alors que le feed credite un autre joueur.
	diverge bool
}

// lancerGrenade : un lancer publie par l'artefact, sur le meme axe que mortGrenade.tMS.
type lancerGrenade struct {
	idx  int
	tMS  int
	rang int
}

// corpusRejeu : le dossier d'artefacts, ou saute le test.
func corpusRejeu(t *testing.T) string {
	t.Helper()
	dir := strings.TrimSpace(os.Getenv("REPLAY_CORPUS"))
	if dir == "" {
		t.Skip("REPLAY_CORPUS non defini : corpus d'artefacts absent (non versionne)")
	}
	return dir
}

// cacheDesFilms : la racine des chunks de film, derivee de la racine du depot.
//
// LE LITTERAL `film_chunks` EST UNE COPIE DE PLUS, ET ELLE EST DECLAREE. La disposition du
// cache est nommee dans `film/filmcache` (`chunksDir`), qui n'expose que `ManifestPath` — il
// n'y a pas de helper public rendant le repertoire de chunks, donc rien a reutiliser ici.
// Le registre porte deja la dette (« Litteral `film_chunks`/`film_manifests` en dur dans ~7
// CLI », lot hygiene de cloture v7.5) : ce fichier entre dans le meme balayage, il ne cree
// pas un cas nouveau.
func cacheDesFilms(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("repertoire courant : %v", err)
	}
	for i := 0; i < 8; i++ {
		cand := filepath.Join(dir, "data", "cache", "film_chunks")
		if st, err := os.Stat(cand); err == nil && st.IsDir() {
			return cand
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Skip("data/cache/film_chunks absent : les chunks de film ne sont pas versionnes")
	return ""
}

// artefactsDuCorpus liste les documents de rejeu, extension .json STRICTE (le dossier porte
// aussi des copies datees, qui ne sont pas des artefacts courants).
func artefactsDuCorpus(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("lecture du corpus : %v", err)
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		out = append(out, filepath.Join(dir, e.Name()))
	}
	sort.Strings(out)
	return out
}

func lireArtefact(t *testing.T, path string) replay.ReplayDocument {
	t.Helper()
	raw, err := os.ReadFile(path) //nolint:gosec // chemin de corpus fourni par l'operateur
	if err != nil {
		t.Fatalf("lecture %s : %v", path, err)
	}
	var doc replay.ReplayDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("decodage %s : %v", path, err)
	}
	return doc
}

// rangOracle rend le rang de type d'une etiquette de classe GRENADE, -1 quand l'etiquette ne le
// donne pas. C'est l'ORACLE : le rang de l'entree `gggl` et le rang des lancers sont le meme
// index (rules.tsv, genre GGGL : 0 frag, 1 plasma, 2 dynamo, 3 spike).
func rangOracle(l damagetag.Label) int {
	m := ggglEntreeRe.FindStringSubmatch(l.Detail)
	if m == nil {
		return -1
	}
	n := 0
	if _, err := fmt.Sscanf(m[1], "%d", &n); err != nil {
		return -1
	}
	return n
}

// part rend un pourcentage, 0 quand le denominateur est nul.
func part(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return 100 * float64(n) / float64(d)
}

// borneHauteWilson rend la borne SUPERIEURE a 95 % de l'intervalle de Wilson, en pourcentage.
//
// POURQUOI CETTE BORNE ACCOMPAGNE TOUJOURS LE TAUX, ET PAS SEULEMENT QUAND IL EST NUL. Le gate
// du plan compare une proportion a un seuil de 1 % : un taux sans sa precision ne peut pas
// trancher ce genre de comparaison. « 0 sur 63 » ne veut pas dire « moins de 1 % » (la borne
// vaut 5,7 %), et « 1 sur 696 » ne le veut pas non plus par evidence (la borne vaut 0,81 %,
// c'est elle qui tranche). Wilson plutot que la regle de trois : il vaut pour TOUT numerateur,
// nul ou non, et il reste ferme sur les petits denominateurs la ou l'approximation normale
// donnerait des bornes negatives.
func borneHauteWilson(x, n int) float64 {
	if n == 0 {
		return 0
	}
	const z = 1.96
	fn, p := float64(n), float64(x)/float64(n)
	centre := p + z*z/(2*fn)
	demiLargeur := z * math.Sqrt(p*(1-p)/fn+z*z/(4*fn*fn))
	return 100 * (centre + demiLargeur) / (1 + z*z/fn)
}

// balayage : la ventilation des morts d'une population de films par source de degat. UN SEUL
// compteur pour les deux instruments (corpus d'artefacts et cache de films) : deux ventilations
// du meme fait divergeraient, et c'est le meme denominateur qui decide du gate 0.
type balayage struct {
	films, echecs      int
	morts              int
	grenades           int
	ambigues, valides  int
	autresStatuts      int
	sansEtiquette      int
	inconnues          int
	parTag             map[uint32]int
	filmsAvecAmbiguite map[string]int
}

func nouveauBalayage() balayage {
	return balayage{parTag: map[uint32]int{}, filmsAvecAmbiguite: map[string]int{}}
}

// compter ventile une source de degat : etiquette absente du catalogue, statut INCONNU, et
// — pour la seule classe GRENADE — le detail par tag et par statut.
func (b *balayage) compter(tag uint32, court string) {
	b.morts++
	l, connu := damagetag.Lookup(tag)
	if !connu {
		b.sansEtiquette++
		return
	}
	if l.Status == damagetag.StatusInconnu {
		b.inconnues++
	}
	if l.Class != damagetag.ClassGrenade {
		return
	}
	b.grenades++
	b.parTag[tag]++
	switch l.Status {
	case damagetag.StatusAmbigu:
		b.ambigues++
		b.filmsAvecAmbiguite[court]++
	case damagetag.StatusValide:
		b.valides++
	default:
		b.autresStatuts++
	}
}

// compterResultat ventile les deux populations de morts d'un decodage.
func (b *balayage) compterResultat(res *killsource.Result, court string) {
	for _, k := range res.Kills {
		b.compter(k.Source.Tag, court)
	}
	for _, d := range res.UnclaimedDeaths {
		b.compter(d.Source.Tag, court)
	}
}

// publierVentilation rend la repartition par tag et les comptes hors classe. Les libelles
// propres a chaque instrument (numeros d'item, gate, borne statistique) sont ajoutes par
// l'appelant : ce qui est partage ici, c'est la MESURE, pas son commentaire.
func (b *balayage) publierVentilation(t *testing.T) {
	t.Helper()
	t.Logf("morts de classe GRENADE : %d / %d = %.2f %% des morts decodees",
		b.grenades, b.morts, part(b.grenades, b.morts))
	tags := make([]uint32, 0, len(b.parTag))
	for tag := range b.parTag {
		tags = append(tags, tag)
	}
	sort.Slice(tags, func(i, j int) bool { return b.parTag[tags[i]] > b.parTag[tags[j]] })
	t.Logf("%-10s %-8s %8s %9s   %s", "tag", "statut", "morts", "part", "rang oracle")
	for _, tag := range tags {
		l, _ := damagetag.Lookup(tag)
		t.Logf("%08x   %-8s %8d %8.2f %%   %d", tag, l.Status, b.parTag[tag],
			part(b.parTag[tag], b.grenades), rangOracle(l))
	}
	t.Logf("hors classe GRENADE : %d mort(s) dont le tag est ABSENT du catalogue, %d dont le "+
		"statut est INCONNU — leur classe n'est pas etablie, une grenade pourrait s'y cacher",
		b.sansEtiquette, b.inconnues)
}

// publierFilmsAmbigus nomme les films porteurs d'une etiquette AMBIGU, s'il y en a.
func (b *balayage) publierFilmsAmbigus(t *testing.T) {
	t.Helper()
	if len(b.filmsAvecAmbiguite) == 0 {
		return
	}
	films := make([]string, 0, len(b.filmsAvecAmbiguite))
	for f := range b.filmsAvecAmbiguite {
		films = append(films, f)
	}
	sort.Strings(films)
	for _, f := range films {
		t.Logf("  film %s : %d mort(s) a etiquette AMBIGU", f, b.filmsAvecAmbiguite[f])
	}
}

// filmDeLArtefact : les deux lectures d'un meme match, cote a cote.
type filmDeLArtefact struct {
	doc      replay.ReplayDocument
	res      *killsource.Result
	idParNom map[string]int
	lancers  []lancerGrenade
}

// chargerFilm lit l'artefact et decode le film du meme match. (nil, false) quand l'un des deux
// manque ou refuse.
func chargerFilm(t *testing.T, path, cacheFilms string) (*filmDeLArtefact, bool) {
	t.Helper()
	doc := lireArtefact(t, path)
	if doc.OriginMs == nil {
		t.Logf("  %s : aucune origine d'horloge publiee — film ecarte", filepath.Base(path))
		return nil, false
	}
	court := title.FilmShortMatchID(doc.MatchID)
	src, err := killsource.DirChunks(filepath.Join(cacheFilms, court))
	if err != nil {
		t.Logf("  %s : chunks absents du cache (%v) — film ecarte", court, err)
		return nil, false
	}
	res, err := killsource.Decode(context.Background(), doc.MatchID, src, nil)
	if err != nil {
		t.Logf("  %s : source de degat non decodee (%v) — film ecarte", court, err)
		return nil, false
	}
	f := &filmDeLArtefact{doc: doc, res: res, idParNom: map[string]int{}}
	for _, r := range doc.Roster {
		if r.Name != "" {
			f.idParNom[r.Name] = r.FilmIndex
		}
	}
	pas := doc.FrameIntervalMS
	if pas <= 0 {
		pas = 100
	}
	for _, g := range doc.Grenades {
		f.lancers = append(f.lancers, lancerGrenade{idx: g.Idx, tMS: g.T * pas, rang: g.Rank})
	}
	sort.Slice(f.lancers, func(i, j int) bool { return f.lancers[i].tMS < f.lancers[j].tMS })
	return f, true
}

// mortsGrenade extrait les morts de classe GRENADE des deux populations du decodeur.
func (f *filmDeLArtefact) mortsGrenade(court string) []mortGrenade {
	var out []mortGrenade
	ajouter := func(tag uint32, tMS int, proprietaire string, revendiquee, diverge bool) {
		l, connu := damagetag.Lookup(tag)
		if !connu || l.Class != damagetag.ClassGrenade {
			return
		}
		idx, ponte := f.idParNom[proprietaire]
		if !ponte {
			idx = -1
		}
		out = append(out, mortGrenade{
			film: court, tag: tag, rangAttendu: rangOracle(l), lanceurIdx: idx,
			tMS: tMS - int(*f.doc.OriginMs), revendiquee: revendiquee, diverge: diverge,
		})
	}
	for _, k := range f.res.Kills {
		// LE PROPRIETAIRE DE LA SOURCE, PAS LE CREDIT. Quand les deux verites divergent, la
		// grenade appartenait a la VICTIME (killsource, doc.go).
		proprietaire := k.Feed.Killer
		if k.Diverges {
			proprietaire = k.Victim
		}
		ajouter(k.Source.Tag, k.TimeMS, proprietaire, true, k.Diverges)
	}
	for _, d := range f.res.UnclaimedDeaths {
		ajouter(d.Source.Tag, d.TimeMS, d.Victim, false, false)
	}
	return out
}
