package filmdec

// sonde_ti11_objectifs_test.go — INSTRUMENT DE MESURE de l'archetype OBJECTIFS (ti=11),
// phase 1 du plan .ai/V7.5/replay2d/PLAN_R4_OBJECTIFS_VIVANTS_TI11.md (lot R4).
//
// CE QU'IL MESURE, et pourquoi il existe. Le corpus documentaire du depot se CONTREDIT sur
// l'endroit ou la traversee de ti=11 s'arrete : PLAN_RETOURS_PLANCHE §R4 et
// SUIVI_REPLAY_2D.md:320 disent `interaction-filter` en i4 ; components_batch3.go:12 et
// .ai/PLAN_OBJECTIFS_TEMPS_REEL.md:25 disent i0 (« Le SUIVI accusait i4 [...] C'est faux »).
// Un lot qui part sur la mauvaise version perd son temps sur le mauvais composant. Cet
// instrument remplace les deux affirmations par une MESURE datee, sur pieces.
//
// Quatre grandeurs, toutes avec leur denominateur :
//
//	1.1 CORPUS    combien d'entites ti=11 chaque film porte (le Slayer est le temoin negatif
//	              natif : il doit en porter ZERO).
//	1.2 GRAMMAIRE la liste ORDONNEE des composants de ti=11, lue dans le registre du film.
//	1.3 MASQUE    quels composants sont REELLEMENT PRESENTS dans les records — c'est ce qui
//	              situe le mur, puisque le traverseur ne consomme AUCUN bit pour un composant
//	              absent (traverse.go:1183) et ne s'arrete qu'au premier PRESENT non porte.
//	1.4 BANDE     la bande de slots de ti=11, et sa bande FANTOME de controle.
//
// IL NE MODIFIE RIEN : ni le decodeur, ni l'assemblage, ni l'artefact. Lecture seule du film,
// aucune ecriture disque. SOUS GARDE D'ENVIRONNEMENT (TI11_FILM), donc saute partout ailleurs,
// CI comprise — patron de replay/ground_weapon_research_test.go.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 TI11_FILM=<repo>/data/cache/film_chunks/64e8adfa \
//	  go test ./internal/analysis/filmdec/ -run TI11 -v

import (
	"fmt"
	"os"
	"sort"
	"testing"
)

const ti11FilmEnv = "TI11_FILM" // repertoire des chunks du film

// ObjectiveTypeIndex est l'archetype des OBJECTIFS. Meme reserve que GroundWeaponTypeIndex :
// c'est un index de build, pas une constante du format.
const ti11TypeIndex = 11

// ti11MaxComponentIndex est le dernier index de composant que la grammaire de ti=11 admet
// (34 composants, lus dans le registre : i0..i33). Un masque qui porte un index superieur ne
// peut PAS venir d un record d objectif — c est le controle de purete de la bande.
const ti11MaxComponentIndex = 33

// ti11Dir rend le repertoire du film, ou saute le test.
func ti11Dir(t *testing.T) string {
	t.Helper()
	dir := os.Getenv(ti11FilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", ti11FilmEnv)
	}
	return dir
}

// ti11Census est le recensement des records de keyframe par archetype.
type ti11Census struct {
	keyframes int
	recordsTI map[int]int
	slotsTI   map[int]map[uint32]bool
}

// ti11KeyframeCensus recense les records de keyframe par archetype. HORS LIGNE.
func ti11KeyframeCensus(dir string) ti11Census {
	c := ti11Census{recordsTI: map[int]int{}, slotsTI: map[int]map[uint32]bool{}}
	n := CountFilmChunks(dir)
	for i := 1; i <= n; i++ {
		data, err := ReadFilmChunk(dir, i)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			c.keyframes++
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				c.recordsTI[r.TI]++
				if c.slotsTI[r.TI] == nil {
					c.slotsTI[r.TI] = map[uint32]bool{}
				}
				c.slotsTI[r.TI][uint32(r.Slot)] = true
			}
		}
	}
	return c
}

// TestTI11Corpus (1.1) rend le recensement des entites ti=11 du film, et le situe parmi les
// autres archetypes. Le denominateur est le nombre total de records de keyframe.
func TestTI11Corpus(t *testing.T) {
	dir := ti11Dir(t)
	c := ti11KeyframeCensus(dir)

	total := 0
	for _, v := range c.recordsTI {
		total += v
	}
	t.Logf("CORPUS — %s", dir)
	t.Logf("  chunks %d · keyframes %d · records de keyframe %d · archetypes distincts %d",
		CountFilmChunks(dir), c.keyframes, total, len(c.recordsTI))
	t.Logf("  ti=%d (OBJECTIFS) : %d records · %d slots distincts",
		ti11TypeIndex, c.recordsTI[ti11TypeIndex], len(c.slotsTI[ti11TypeIndex]))

	// Le classement complet situe ti=11 : un archetype a 5 entites n'a pas le meme regime
	// qu'un archetype a 20 000, et le lecteur du journal doit le voir sans relancer.
	type row struct {
		ti, rec, slots int
	}
	rows := make([]row, 0, len(c.recordsTI))
	for ti, rec := range c.recordsTI {
		rows = append(rows, row{ti, rec, len(c.slotsTI[ti])})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].rec > rows[j].rec })
	t.Logf("  classement des archetypes par records de keyframe :")
	for i, r := range rows {
		if i >= 15 && r.ti != ti11TypeIndex {
			continue
		}
		mark := ""
		if r.ti == ti11TypeIndex {
			mark = "   <-- OBJECTIFS"
		}
		t.Logf("    ti=%-3d %7d records · %5d slots%s", r.ti, r.rec, r.slots, mark)
	}
	// Les VALEURS de slot sont publiees : le balayage d objet du monde code le slot sur 13
	// bits (projectiles.go:290), donc un slot >= 8192 lui est structurellement inaccessible.
	if ss := c.slotsTI[ti11TypeIndex]; len(ss) > 0 {
		vals := make([]int, 0, len(ss))
		for s := range ss {
			vals = append(vals, int(s))
		}
		sort.Ints(vals)
		over := 0
		for _, v := range vals {
			if v >= 1<<13 {
				over++
			}
		}
		t.Logf("  slots ti=%d observes : %v", ti11TypeIndex, vals)
		t.Logf("  slots hors de portee d un champ 13 bits (>= 8192) : %d/%d", over, len(vals))
	}
	if c.recordsTI[ti11TypeIndex] == 0 {
		t.Logf("  VERDICT : ce film ne porte AUCUNE entite d'objectif (temoin negatif attendu"+
			" sur un Slayer ; sur un mode a objectif portable, c'est une REFUTATION de la"+
			" presence supposee). recordsTI[%d]=0", ti11TypeIndex)
	}
}

// TestTI11Grammaire (1.2) lit la liste ORDONNEE des composants de ti=11 dans le registre du
// film (chunk_00), et dit lesquels sont dispatches aujourd'hui.
//
// Le registre est bit-a-bit identique d'un film a l'autre (registry.go:44-47) : une lecture
// suffit, mais l'empreinte est publiee pour que deux films puissent etre compares.
func TestTI11Grammaire(t *testing.T) {
	dir := ti11Dir(t)
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("lecture de chunk_00 (registre) : %v", err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("analyse du registre : %v", err)
	}
	arch, ok := reg.Archetype(ti11TypeIndex)
	if !ok {
		t.Fatalf("archetype %d absent du registre (%d archetypes)", ti11TypeIndex, len(reg.Archetypes))
	}
	t.Logf("GRAMMAIRE — archetype ti=%d : %d composants (registre : %d archetypes)",
		ti11TypeIndex, len(arch.Components), len(reg.Archetypes))

	// « Porte » = le dispatch consomme ce composant. On le demande au dispatch lui-meme
	// plutot qu'a une liste ecrite a la main : une liste diverge, le dispatch fait foi.
	ported := 0
	for i, name := range arch.Components {
		p := ti11ComponentIsPorted(name, ti11TypeIndex, arch.Level(i))
		flag := "NON PORTE"
		if p {
			flag = "porte   "
		}
		if p {
			ported++
		}
		t.Logf("    i%-2d  %s  L=%-3d  %s", i, flag, arch.Level(i), name)
	}
	t.Logf("  COUVERTURE DE DISPATCH : %d/%d composants portes", ported, len(arch.Components))
}

// ti11ComponentIsPorted interroge le DISPATCH REEL (consumeByName) pour savoir si un composant
// est porte. Sur un lecteur vide : un composant porte peut consommer 0 bit, mais il rend
// ported=true, et c'est cette valeur seule qui gouverne l'arret du traverseur.
func ti11ComponentIsPorted(name string, typeIndex, level uint32) bool {
	br := NewBitReader(make([]byte, 64))
	_, _, ported := consumeByName(br, name, typeIndex, level)
	return ported
}

// TestTI11MasqueEtBande (1.3 + 1.4) mesure la bande de slots de ti=11, sa bande FANTOME de
// controle, et — c'est le coeur — la distribution des composants REELLEMENT PRESENTS dans les
// records delta de ces slots.
//
// POURQUOI CETTE MESURE SITUE LE MUR. Le traverseur ne consomme aucun bit pour un composant
// absent du masque (traverse.go:1183) : il ne s'arrete donc PAS au premier composant non porte
// de l'archetype, mais au premier composant non porte QUI EST PRESENT. Dire « le mur est a i0 »
// ou « a i4 » sans avoir regarde les masques reels, c'est parler de l'archetype, pas du flux.
//
// Le balayage utilise matchWorldObjectRecord (projectiles.go), le meme reconnaisseur d'en-tete
// que les projectiles et l'equipement — il rend directement `Idx`, la liste des index de
// composant du masque. Reserve a garder en tete : cet en-tete est celui des OBJETS DU MONDE
// (ti=37/38/41/42) ; qu'il reconnaisse ou non les records de ti=11 est en soi un resultat.
func TestTI11MasqueEtBande(t *testing.T) {
	dir := ti11Dir(t)
	n := CountFilmChunks(dir)
	c := ti11KeyframeCensus(dir)

	// TROIS bandes, et la comparaison des trois est le resultat.
	//
	// POURQUOI LA BANDE COMBLEE EST SUSPECTE ICI, alors qu'elle est la bonne pour les
	// projectiles. La regle de comblement (projectiles.go:365) existe parce qu'un projectile
	// vit moins d'une seconde et que les keyframes sont espaces de ~20 s : l'observe rate
	// l'essentiel. UN OBJECTIF EST L'EXACT CONTRAIRE — il vit toute la partie, donc il est
	// present a CHAQUE keyframe, et l'observe est deja complet. Combler ne recupere alors
	// aucune couverture : ca ne fait qu'avaler les slots voisins.
	observed := map[uint32]bool{}
	for s := range c.slotsTI[ti11TypeIndex] {
		observed[s] = true
	}
	band := worldObjectSlotBand(dir, n, ti11TypeIndex)
	ghost := ti11GhostBand(c, len(band))
	t.Logf("BANDE — ti=%d : %d slots OBSERVES en keyframe -> %d slots apres comblement et"+
		" retrait des slots vus porter un autre archetype (facteur %s)",
		ti11TypeIndex, len(observed), len(band), ti11Ratio(len(band), len(observed)))
	t.Logf("  bande FANTOME de controle : %d slots (jamais vus porter ti=%d)", len(ghost), ti11TypeIndex)
	if len(observed) == 0 {
		t.Logf("  VERDICT : aucun slot observe — aucune mesure de masque possible sur ce film.")
		return
	}

	obs := ti11ScanMasks(dir, n, observed)
	real := ti11ScanMasks(dir, n, band)
	fake := ti11ScanMasks(dir, n, ghost)
	ghostObs := ti11GhostBand(c, len(observed))
	fakeObs := ti11ScanMasks(dir, n, ghostObs)
	t.Logf("MASQUE — records delta reconnus :")
	t.Logf("  bande OBSERVEE  %8d records · %d/%d slots peuples", obs.records, len(obs.slots), len(observed))
	t.Logf("  temoin FANTOME de meme taille %8d records · %d/%d slots (rapport observee/fantome %s)",
		fakeObs.records, len(fakeObs.slots), len(ghostObs), ti11Ratio(obs.records, fakeObs.records))
	t.Logf("  bande COMBLEE   %8d records · %d/%d slots peuples", real.records, len(real.slots), len(band))
	t.Logf("  temoin FANTOME de meme taille %8d records · %d/%d slots (rapport comblee/fantome %s)",
		fake.records, len(fake.slots), len(ghost), ti11Ratio(real.records, fake.records))
	t.Logf("  INDICE HORS GRAMMAIRE (un record de ti=11 ne peut porter QUE i0..i33) :")
	t.Logf("    bande observee : %5.1f %% des records portent un index > 33"+
		" · bande comblee : %5.1f %% · fantome : %5.1f %%",
		100*ti11OutOfGrammarRate(obs), 100*ti11OutOfGrammarRate(real), 100*ti11OutOfGrammarRate(fake))

	// La suite detaille la bande OBSERVEE : c'est la seule dont on puisse esperer qu'elle
	// porte ti=11 et rien d'autre.
	real = obs
	band = observed

	if real.records == 0 {
		t.Logf("  VERDICT : l'en-tete d'OBJET DU MONDE ne reconnait AUCUN record sur la bande"+
			" ti=%d. Les records d'objectif n'ont donc pas cette forme d'en-tete — resultat"+
			" NEGATIF utile : la voie 'objet du monde' ne s'applique pas telle quelle.", ti11TypeIndex)
		return
	}

	t.Logf("  presence par index de composant (denominateur = %d records reconnus) :", real.records)
	maxIdx := 0
	for i := range real.byIndex {
		if i > maxIdx {
			maxIdx = i
		}
	}
	for i := 0; i <= maxIdx; i++ {
		cnt := real.byIndex[i]
		if cnt == 0 {
			continue
		}
		t.Logf("    i%-2d  %6d  %5.1f %%   (fantome %6d)", i, cnt,
			100*float64(cnt)/float64(real.records), fake.byIndex[i])
	}
	t.Logf("  PREMIER INDEX PRESENT par record — c'est lui qui decide du mur :")
	firsts := ti11SortedKeys(real.firstIndex)
	for _, i := range firsts {
		t.Logf("    i%-2d  %6d records  %5.1f %%", i, real.firstIndex[i],
			100*float64(real.firstIndex[i])/float64(real.records))
	}
	t.Logf("  nombre de composants par record : %s", ti11Histogram(real.maskCount))
}

// ti11MaskStats agrege les masques d'un balayage.
type ti11MaskStats struct {
	records      int
	slots        map[uint32]bool
	byIndex      map[int]int
	firstIndex   map[int]int
	maskCount    map[int]int
	outOfGrammar int
}

// ti11ScanMasks balaye les paquets delta et agrege les masques des records reconnus sur band.
// HORS LIGNE (I/O disque sur tout le film).
func ti11ScanMasks(dir string, n int, band map[uint32]bool) ti11MaskStats {
	s := ti11MaskStats{
		slots: map[uint32]bool{}, byIndex: map[int]int{},
		firstIndex: map[int]int{}, maskCount: map[int]int{},
	}
	if len(band) == 0 {
		return s
	}
	for ch := 1; ch <= n; ch++ {
		data, err := ReadFilmChunk(dir, ch)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			limit := len(pay)*8 - (worldObjectHeaderBits + worldObjectIndexBits)
			for p := 0; p <= limit; p++ {
				rec, ok := matchWorldObjectRecord(pay, p, band)
				if !ok {
					continue
				}
				s.records++
				s.slots[rec.Slot] = true
				s.maskCount[len(rec.Idx)]++
				s.firstIndex[rec.Idx[0]]++
				for _, i := range rec.Idx {
					if i > ti11MaxComponentIndex {
						s.outOfGrammar++
						break
					}
				}
				for _, i := range rec.Idx {
					s.byIndex[i]++
				}
				p = rec.After // un record accepte n'est pas re-balaye
			}
		}
	}
	return s
}

// ti11GhostBand construit une bande FANTOME de meme cardinalite que la bande reelle, faite de
// slots JAMAIS vus porter ti=11 dans aucun keyframe. Elle passe par le MEME code de balayage
// que la mesure : sans cela le controle ne controlerait pas le decodeur mais une variante de
// lui (regle du lot armes au sol, keyframe_ground_weapons.go:150).
//
// LE FANTOME EST TIRE DANS LE MEME VOISINAGE NUMERIQUE que les slots reels, et cette precaution
// n'est pas cosmetique : un balayage qui reconnait un slot sur 13 bits n'a pas la meme chance de
// tomber juste sur un petit numero que sur un grand (les motifs de bits ne sont pas
// equiprobables). Un fantome tire a partir de 1 gonflerait artificiellement le temoin et
// donnerait au signal reel l'air d'etre sous le bruit alors que la comparaison serait faussee.
// On encadre donc [min, max] des slots observes, en s'ecartant vers le haut si la place manque.
func ti11GhostBand(c ti11Census, size int) map[uint32]bool {
	seen := c.slotsTI[ti11TypeIndex]
	ghost := map[uint32]bool{}
	if len(seen) == 0 || size == 0 {
		return ghost
	}
	lo, hi := uint32(kfTableCap), uint32(0)
	for s := range seen {
		if s < lo {
			lo = s
		}
		if s > hi {
			hi = s
		}
	}
	for s := lo; s < kfTableCap && len(ghost) < size; s++ {
		if seen[s] {
			continue
		}
		ghost[s] = true
	}
	// Si la fenetre [lo, cap) n'a pas suffi, on complete vers le bas — cas des bandes comblees,
	// tres larges, ou l'encadrement strict ne peut pas tenir.
	for s := uint32(1); s < lo && len(ghost) < size; s++ {
		if seen[s] {
			continue
		}
		ghost[s] = true
	}
	return ghost
}

// ti11OutOfGrammarRate rend la part des records dont le masque porte au moins un index HORS
// de la grammaire de ti=11 (34 composants, donc i0..i33). C'est le controle de purete le plus
// direct qui soit : un record d'objectif ne PEUT PAS porter i40. Un taux non nul mesure
// exactement la contamination de la bande par d'autres archetypes.
func ti11OutOfGrammarRate(s ti11MaskStats) float64 {
	if s.records == 0 {
		return 0
	}
	return float64(s.outOfGrammar) / float64(s.records)
}

func ti11SortedKeys(m map[int]int) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

func ti11Histogram(m map[int]int) string {
	s := ""
	for _, k := range ti11SortedKeys(m) {
		s += fmt.Sprintf("%d:%d ", k, m[k])
	}
	return s
}

func ti11Ratio(a, b int) string {
	if b == 0 {
		if a == 0 {
			return "0/0"
		}
		return "fantome nul"
	}
	return fmt.Sprintf("%.2fx", float64(a)/float64(b))
}
