package replay

// visee_tir_research_test.go — LE TIR FATAL PORTE-T-IL L'ETAT DE LUNETTE ?
//
// L'INSTANT LE MIEUX ETIQUETE DU FILM EST UN RECORD DE DEGAT. Le record delta de type 105
// (fire_events.go) n'existe que lorsqu'un degat PORTE ; le tir qui tue en produit un, date a la
// milliseconde. Or les medailles du feed etiquettent exactement cet instant :
//
//	No Scope      (100,114) -> le tueur N'ETAIT PAS zoome au moment de ce tir.
//	Counter-snipe (100,168) -> le tueur ETAIT zoome (« while YOU BOTH ARE ZOOMED »).
//
// Meme classe d'arme des deux cotes (« Power sniper rifle »), memes circonstances (un kill),
// seule la lunette differe : le confondeur « je tire / je suis au contact » du balayage de
// composants ne peut pas operer ici.
//
// CE QUE L'INSTRUMENT MESURE. La tete du record 105 a des OFFSETS FIXES sur 113 bits
// (fire_events.go), dont une vingtaine ne sont pas expliques (bit 8, preambule 9..35, cause
// 41..43, cinq drapeaux 108..112). Pour chaque bit de la tete, le taux d'allumage du record
// fatal est compare entre les deux classes. Un bit d'etat de lunette separerait presque
// binairement ; les bits de reference d'entite (poignees) bruiteront pareil des deux cotes.
//
// RECALAGE D'HORLOGE, SANS DECODER LES POSITIONS. Le feed date sur l'horloge du match, les
// paquets sur celle du film ; l'ecart est constant par film. Il est estime par VOTE DE MODE :
// chaque kill du feed vote pour les ecarts (record 105 - kill) qu'il observe, un vote par bin
// et par kill ; le vrai ecart recoit le vote de presque chaque kill (le degat fatal), le bruit
// s'etale. Gate declare : le bin gagnant porte au moins tirVoteMin votes ET 1,5 fois le
// meilleur bin hors de son voisinage d'une seconde. Films rejetes comptes, jamais forces.
// Controle externe : sur 000d5950, l'ecart mort<->fin de vie mesure par ailleurs vaut
// 4 517 847 ms ; l'estimateur doit tomber a moins d'une seconde de la.
//
// SEUILS, ECRITS AVANT LA MESURE (methode du depot) :
//
//	population    >= tirMinParClasse instants apparies PAR classe, corpus entier ;
//	CANDIDAT      |taux_zoome - taux_non_zoome| >= tirEcartCandidat (0,50) ;
//	A SUIVRE      ecart >= tirEcartSuivi (0,25), publie comme piste, jamais comme resultat.
//
// AUTO-CONTROLE D'APPARIEMENT : la distribution des armes des records apparies est publiee.
// Si l'appariement est juste, les fusils a lunette dominent DANS LES DEUX classes ; une
// distribution plate dirait « l'appariement attrape n'importe quel degat », et le differentiel
// ne vaudrait rien.
//
// PARALLELISME SANS VERROU, ET POURQUOI C'EST PERMIS : ce fichier n'appelle AUCUN Scan* de
// filmdec — seulement ReadFilmChunk/WalkPackets/ReadBitsAtForDiag, purs, sans etat global de
// decodage. Le verrou de process (LockProcessDecode) protege les globaux des decodeurs ;
// aucun n'est touche ici.
//
// SOUS GARDE D'ENVIRONNEMENT (TIR_FILMS_DIR), saute partout ailleurs, CI comprise.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 TIR_FILMS_DIR=<repo>/data/cache/film_chunks \
//	  TIR_TSV=<repo>/.ai/V7.5/film_re \
//	  go test ./internal/analysis/replay/ -run TestViseeTirFatal -v -timeout 120m

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

const (
	tirFilmsDirEnv = "TIR_FILMS_DIR"
	tirMaxFilmsEnv = "TIR_MAX_FILMS" // plafond de films traites (0 ou absent = tous)
	tirTSVEnv      = "TIR_TSV"
)

// Les seuils de l'instrument, ecrits avant la mesure.
const (
	tirHeadBits       = 113 // la tete a offsets fixes du record 105 (fire_events.go)
	tirMinParClasse   = 30
	tirEcartCandidat  = 0.50
	tirEcartSuivi     = 0.25
	tirFenetreMatchMS = 250 // medaille -> record fatal : au-dela, pas d'appariement
	tirBinGrossierMS  = 100
	tirBinFinMS       = 10
	tirVoteMin        = 10  // votes minimaux du bin gagnant du recalage
	tirVoteDominance  = 1.5 // et facteur sur le meilleur bin hors voisinage 1 s
	tirFondPas        = 97  // 1 record sur 97 entre au temoin de fond (premier > 60)
)

// tirRecord est la tete d'UN record 105 long : l'instant, l'arme, et les 113 bits bruts.
type tirRecord struct {
	tMS    int64
	weapon uint64
	head   [2]uint64 // bits 0..63 / 64..112, MSB du payload en bit 0 de head[0]
}

// tirFeed est ce que le chunk d'evenements d'un film donne a cet instrument.
type tirFeed struct {
	kills   []int64 // instants des events kill (horloge du feed)
	noScope []int64 // instants etiquetes NON zoome
	counter []int64 // instants etiquetes ZOOME
}

// tirClasse accumule le differentiel d'une classe : compte d'instants apparies et, par bit de
// tete, le nombre de records fatals qui le portent.
type tirClasse struct {
	apparies, sansRecord int
	bits                 [tirHeadBits]int
	armes                map[uint64]int
}

func (c *tirClasse) ajoute(r tirRecord) {
	c.apparies++
	for b := 0; b < tirHeadBits; b++ {
		if r.head[b/64]>>(63-b%64)&1 == 1 {
			c.bits[b]++
		}
	}
	if c.armes == nil {
		c.armes = map[uint64]int{}
	}
	c.armes[r.weapon]++
}

// tirBilanFilm est le resultat d'UN film, fusionne ensuite sous mutex.
type tirBilanFilm struct {
	film                   string
	sansFeed, sansRecalage bool
	offsetMS               int64
	zoome, nonZoome, fond  tirClasse
}

// TestViseeTirFatal execute le differentiel sur le corpus.
func TestViseeTirFatal(t *testing.T) {
	root := os.Getenv(tirFilmsDirEnv)
	if root == "" {
		t.Skipf("%s absent : instrument de mesure saute", tirFilmsDirEnv)
	}
	dirs := adsListeFilms(t, root)
	if cap, _ := strconv.Atoi(os.Getenv(tirMaxFilmsEnv)); cap > 0 && cap < len(dirs) {
		dirs = dirs[:cap]
	}
	t.Logf("CORPUS — %d repertoires de film sous %s", len(dirs), root)

	debut := time.Now()
	bilans := tirTraiteCorpus(dirs, root)
	t.Logf("COUT — %d films traites en %s", len(bilans), time.Since(debut).Round(time.Second))

	var zoome, nonZoome, fond tirClasse
	sansFeed, sansRecalage, utiles := 0, 0, 0
	for _, b := range bilans {
		switch {
		case b.sansFeed:
			sansFeed++
		case b.sansRecalage:
			sansRecalage++
		default:
			utiles++
			tirFusionne(&zoome, b.zoome)
			tirFusionne(&nonZoome, b.nonZoome)
			tirFusionne(&fond, b.fond)
		}
		if b.film == "000d5950" && !b.sansFeed && !b.sansRecalage {
			t.Logf("CONTROLE EXTERNE — 000d5950 : recalage estime %d ms (attendu ~4517847,"+
				" ecart %d ms)", b.offsetMS, b.offsetMS-4517847)
		}
	}
	t.Logf("FILMS — %d utiles, %d sans etiquette/feed, %d rejetes par le gate de recalage",
		utiles, sansFeed, sansRecalage)
	t.Logf("INSTANTS — zoome (Counter-snipe) : %d apparies, %d sans record fatal a moins de"+
		" %d ms ; non zoome (No Scope) : %d apparies, %d sans record",
		zoome.apparies, zoome.sansRecord, tirFenetreMatchMS, nonZoome.apparies, nonZoome.sansRecord)

	tirJournaliseArmes(t, "zoome", zoome)
	tirJournaliseArmes(t, "non zoome", nonZoome)
	tirVerdict(t, zoome, nonZoome, fond)
	tirEcrisTSV(t, zoome, nonZoome, fond)
}

// tirTraiteCorpus repartit les films sur un pool de workers ; aucun etat global de decodage
// n'est touche (voir l'en-tete), donc aucun verrou de process n'est pris.
func tirTraiteCorpus(dirs []string, root string) []tirBilanFilm {
	workers := runtime.NumCPU()
	if workers > 8 {
		workers = 8
	}
	in := make(chan string, len(dirs))
	for _, d := range dirs {
		in <- d
	}
	close(in)
	var mu sync.Mutex
	var out []tirBilanFilm
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for d := range in {
				b := tirTraiteFilm(filepath.Join(root, d), d)
				mu.Lock()
				out = append(out, b)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	sort.Slice(out, func(i, j int) bool { return out[i].film < out[j].film })
	return out
}

// tirTraiteFilm fait tout le travail d'UN film : feed, records, recalage, appariement.
func tirTraiteFilm(dir, name string) tirBilanFilm {
	b := tirBilanFilm{film: name}
	feed, ok := tirLitFeed(dir)
	if !ok || (len(feed.noScope) == 0 && len(feed.counter) == 0) || len(feed.kills) == 0 {
		b.sansFeed = true
		return b
	}
	records := tirLitRecords(dir)
	if len(records) == 0 {
		b.sansFeed = true
		return b
	}
	off, ok := tirRecale(feed.kills, records)
	if !ok {
		b.sansRecalage = true
		return b
	}
	b.offsetMS = off
	for _, tms := range feed.counter {
		tirApparie(records, tms+off, &b.zoome)
	}
	for _, tms := range feed.noScope {
		tirApparie(records, tms+off, &b.nonZoome)
	}
	for i := 0; i < len(records); i += tirFondPas {
		b.fond.ajoute(records[i])
	}
	return b
}

// tirLitFeed lit le chunk d'evenements (le dernier) et rend kills + instants etiquetes.
func tirLitFeed(dir string) (tirFeed, bool) {
	n := filmdec.CountFilmChunks(dir)
	if n == 0 {
		return tirFeed{}, false
	}
	raw, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("chunk_%02d.bin", n)))
	if err != nil {
		return tirFeed{}, false
	}
	evs, err := analysis.ParseHighlightEvents(raw, 0)
	if err != nil {
		return tirFeed{}, false
	}
	var f tirFeed
	for _, e := range evs {
		switch {
		case e.EventType == analysis.EventTypeKill:
			f.kills = append(f.kills, int64(e.TimeMS))
		case e.EventType == analysis.EventTypeMedal &&
			e.TypeHint == adsTypeHintMulti && e.MedalType == adsMedalNoScope:
			f.noScope = append(f.noScope, int64(e.TimeMS))
		case e.EventType == analysis.EventTypeMedal &&
			e.TypeHint == adsTypeHintMulti && e.MedalType == adsMedalCounter:
			f.counter = append(f.counter, int64(e.TimeMS))
		}
	}
	return f, true
}

// tirLitRecords balaye les paquets delta et rend la tete de chaque record 105 LONG, trie par
// instant. Memes filtres que ScanFilmFireEvents (variante longue, tete complete).
func tirLitRecords(dir string) []tirRecord {
	n := filmdec.CountFilmChunks(dir)
	var out []tirRecord
	for c := 1; c <= n; c++ {
		chunk, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeDelta || p.Size < 1 {
				continue
			}
			pay := p.Payload(chunk)
			if int(pay[0]>>1) != filmdec.FireEventType || int(pay[0])&1 != 0 ||
				len(pay)*8 < tirHeadBits {
				continue
			}
			r := tirRecord{tMS: int64(p.TimestampUS / 1000)}
			r.head[0] = uint64(filmdec.ReadBitsAtForDiag(pay, 0, 32))<<32 |
				uint64(filmdec.ReadBitsAtForDiag(pay, 32, 32))
			r.head[1] = uint64(filmdec.ReadBitsAtForDiag(pay, 64, 32))<<32 |
				uint64(filmdec.ReadBitsAtForDiag(pay, 96, 17))<<15
			r.weapon = uint64(filmdec.ReadBitsAtForDiag(pay, 44, 32))<<32 |
				uint64(filmdec.ReadBitsAtForDiag(pay, 76, 32))
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].tMS < out[j].tMS })
	return out
}

// tirRecale estime l'ecart d'horloge feed -> film par vote de mode (un vote par bin et par
// kill), puis l'affine au bin fin. ok=false si le gate de dominance echoue.
func tirRecale(kills []int64, records []tirRecord) (int64, bool) {
	votes := map[int64]int{}
	for _, k := range kills {
		vu := map[int64]bool{}
		for _, r := range records {
			bin := (r.tMS - k) / tirBinGrossierMS
			if !vu[bin] {
				vu[bin] = true
				votes[bin]++
			}
		}
	}
	meilleur, meilleurV := int64(0), -1
	for bin, v := range votes {
		if v > meilleurV || (v == meilleurV && bin < meilleur) {
			meilleur, meilleurV = bin, v
		}
	}
	secondV := 0
	for bin, v := range votes {
		if (bin < meilleur-10 || bin > meilleur+10) && v > secondV { // hors voisinage 1 s
			secondV = v
		}
	}
	if meilleurV < tirVoteMin || float64(meilleurV) < tirVoteDominance*float64(secondV) {
		return 0, false
	}
	return tirAffine(kills, records, meilleur*tirBinGrossierMS), true
}

// tirAffine rejoue le vote au bin fin dans le voisinage d'une seconde du bin grossier gagnant.
func tirAffine(kills []int64, records []tirRecord, grossier int64) int64 {
	votes := map[int64]int{}
	for _, k := range kills {
		vu := map[int64]bool{}
		for _, r := range records {
			d := r.tMS - k
			if d < grossier-1000 || d > grossier+1000+tirBinGrossierMS {
				continue
			}
			bin := d / tirBinFinMS
			if !vu[bin] {
				vu[bin] = true
				votes[bin]++
			}
		}
	}
	var meilleur int64
	for bin, v := range votes {
		if v > votes[meilleur] || (v == votes[meilleur] && bin < meilleur) {
			meilleur = bin
		}
	}
	return meilleur*tirBinFinMS + tirBinFinMS/2
}

// tirApparie rattache un instant etiquete (deja recale) au record 105 le plus proche.
func tirApparie(records []tirRecord, tFilm int64, c *tirClasse) {
	i := sort.Search(len(records), func(i int) bool { return records[i].tMS >= tFilm })
	best, bd := -1, int64(tirFenetreMatchMS+1)
	for _, j := range []int{i - 1, i} {
		if j < 0 || j >= len(records) {
			continue
		}
		d := records[j].tMS - tFilm
		if d < 0 {
			d = -d
		}
		if d < bd {
			best, bd = j, d
		}
	}
	if best < 0 {
		c.sansRecord++
		return
	}
	c.ajoute(records[best])
}

// tirFusionne verse une classe de film dans l'agregat corpus.
func tirFusionne(dst *tirClasse, src tirClasse) {
	dst.apparies += src.apparies
	dst.sansRecord += src.sansRecord
	for b := range src.bits {
		dst.bits[b] += src.bits[b]
	}
	for w, n := range src.armes {
		if dst.armes == nil {
			dst.armes = map[uint64]int{}
		}
		dst.armes[w] += n
	}
}

// tirJournaliseArmes publie l'auto-controle : les armes des records apparies d'une classe.
func tirJournaliseArmes(t *testing.T, nom string, c tirClasse) {
	t.Helper()
	type wc struct {
		w uint64
		n int
	}
	var l []wc
	for w, n := range c.armes {
		l = append(l, wc{w, n})
	}
	sort.Slice(l, func(i, j int) bool { return l[i].n > l[j].n })
	var sb strings.Builder
	for i, e := range l {
		if i == 5 {
			break
		}
		fmt.Fprintf(&sb, " %016x=%d", e.w, e.n)
	}
	t.Logf("ARMES (%s, top 5) :%s", nom, sb.String())
}

// tirVerdict applique les seuils declares, bit par bit.
func tirVerdict(t *testing.T, zoome, nonZoome, fond tirClasse) {
	t.Helper()
	if zoome.apparies < tirMinParClasse || nonZoome.apparies < tirMinParClasse {
		t.Logf("VERDICT — POPULATION INSUFFISANTE : %d zoome / %d non zoome apparies"+
			" (seuil %d chacun). RIEN N'EST CONCLU.", zoome.apparies, nonZoome.apparies,
			tirMinParClasse)
		return
	}
	candidats, suivis := 0, 0
	for b := 7; b < tirHeadBits; b++ { // 0..6 = le type, constant par construction
		rz := float64(zoome.bits[b]) / float64(zoome.apparies)
		rn := float64(nonZoome.bits[b]) / float64(nonZoome.apparies)
		rf := 0.0
		if fond.apparies > 0 {
			rf = float64(fond.bits[b]) / float64(fond.apparies)
		}
		ecart := rz - rn
		if ecart < 0 {
			ecart = -ecart
		}
		switch {
		case ecart >= tirEcartCandidat:
			candidats++
			t.Logf("  bit %3d : CANDIDAT LUNETTE — zoome %5.1f %% vs non zoome %5.1f %%"+
				" (fond %5.1f %%)", b, 100*rz, 100*rn, 100*rf)
		case ecart >= tirEcartSuivi:
			suivis++
			t.Logf("  bit %3d : a suivre — zoome %5.1f %% vs non zoome %5.1f %% (fond %5.1f %%)",
				b, 100*rz, 100*rn, 100*rf)
		}
	}
	switch {
	case candidats > 0:
		t.Logf("VERDICT — %d bit(s) CANDIDAT(S) dans la tete du record fatal : a confirmer par"+
			" une seconde chaine (grammaire Ghidra du champ) avant toute publication.", candidats)
	case suivis > 0:
		t.Logf("VERDICT — aucun candidat au seuil declare ; %d bit(s) « a suivre » publies"+
			" comme pistes, pas comme resultat.", suivis)
	default:
		t.Log("VERDICT — NEGATIF : aucun des 106 bits libres de la tete du record fatal ne" +
			" separe les tirs zoomes des tirs non zoomes. L'etat de lunette n'est pas dans la" +
			" tete du record de degat.")
	}
}

// tirEcrisTSV depose les taux par bit : la piece qui permet de refaire le calcul ailleurs.
func tirEcrisTSV(t *testing.T, zoome, nonZoome, fond tirClasse) {
	t.Helper()
	out := os.Getenv(tirTSVEnv)
	if out == "" {
		return
	}
	var sb strings.Builder
	sb.WriteString("bit\tzoome_n\tzoome_taux\tnonzoome_n\tnonzoome_taux\tfond_taux\n")
	for b := 0; b < tirHeadBits; b++ {
		rz := tirTauxDe(zoome.bits[b], zoome.apparies)
		rn := tirTauxDe(nonZoome.bits[b], nonZoome.apparies)
		rf := tirTauxDe(fond.bits[b], fond.apparies)
		fmt.Fprintf(&sb, "%d\t%d\t%.4f\t%d\t%.4f\t%.4f\n",
			b, zoome.bits[b], rz, nonZoome.bits[b], rn, rf)
	}
	path := filepath.Join(out, "visee_tir_differentiel.tsv")
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		t.Fatalf("ecriture du releve : %v", err)
	}
	t.Logf("RELEVE — %s", path)
}

func tirTauxDe(num, den int) float64 {
	if den == 0 {
		return 0
	}
	return float64(num) / float64(den)
}
