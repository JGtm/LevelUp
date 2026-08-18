package replay

// zone_state_ti13_scan_test.go — LOT C-bis PHASE 2a : LE BALAYAGE DE `ti=13` DANS `replay`.
//
// POURQUOI CE BALAYAGE EST RECOPIE ICI, ET CE QUE LA RECOPIE COUTE. La phase 1 a mesure `ti=13`
// depuis `filmdec`, ou vivent l'ancrage (`matchWorldObjectRecord`) et le rejeu des composants
// (`consumeByName`) — tous deux NON EXPORTES. Le pont geometrique, lui, vit dans `replay`
// (`AttributeZones`, les zones du catalogue, les trajectoires nommees), et `filmdec` ne peut pas
// importer `replay` (cycle). La phase 2a doit donc lire `ti=13` DEPUIS `replay`, et la seule voie
// honnete est de recopier le strict necessaire dans un fichier de TEST : aucun code de production
// n'est ajoute, deplace ou exporte pour cette mesure.
//
// CE QUI EST RECOPIE, ET RIEN DE PLUS :
//
//	l'ancrage      en-tete de record delta d'objet du monde (`filmdec/projectiles.go`) ;
//	la grammaire   ti=13 i0 = R(32) ; i1 = variant mode A ; i2..i33 = variant mode B, avec la
//	               table de largeurs de `filmdec/components_managed_property.go` ;
//	la bande       les slots vus porter ti=13 dans les tables d'image-cle, purges des ambigus.
//
// LA RECOPIE EST CONTROLEE PAR LE FLUX LUI-MEME, pas par la confiance. Deux gardes tournent a
// chaque passe et sont publiees avec les mesures :
//
//	le REGISTRE    l'archetype 13 du film doit declarer les noms attendus aux index attendus,
//	               sinon la mesure s'arrete (le lot 0 a mesure DEUX decoupages de registre) ;
//	le CHAINAGE    part des records dont la position de fin calculee porte un en-tete de record
//	               valide. La phase 0 a mesure 87,0 a 99,3 % par cette voie, contre 2-3 % sur une
//	               bande fantome : une recopie fausse ferait s'effondrer ce taux.
//
// SOUS GARDE D'ENVIRONNEMENT (`ZONE_FILM`), un film par processus, avant-plan.

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	// p2aFilmEnv porte le repertoire des chunks d'UN film (chemin absolu). Meme nom qu'en
	// phase 1 : c'est la meme garde, pour le meme corpus.
	p2aFilmEnv = "ZONE_FILM"
	// p2aOutEnv redirige les TSV ; par defaut `.ai/V7.5/replay2d/registre_film/lotCbis/`.
	p2aOutEnv = "ZONE_OUT"
)

// Grammaire de ti=13, recopiee de `filmdec/components_managed_property.go`. Les valeurs sont
// celles du desassemblage (lot C-bis phase 0 §2.3), et la table de largeurs est la meme.
const (
	p2aTagVide      = 0
	p2aTagEnum      = 1
	p2aTagBool      = 2
	p2aTagQuant     = 3 // mode A : R(24) quantifie sur [-100, +100] — LA JAUGE DE CAPTURE
	p2aTagU32       = 4 // mode A : R(32) — LE CANAL ENUMERABLE de la phase 1
	p2aTagStringID  = 5 // mode A : R(32) « string-id-value » — LA CLE DE NOMMAGE
	p2aTagU32Bis    = 6
	p2aTagQuantJ    = 7
	p2aTagU32J      = 8
	p2aTagStringIDJ = 9
	p2aTagBoolJ     = 10
	p2aTagEnumJ     = 11

	p2aTagBits    = 4
	p2aQuantBits  = 24
	p2aNomBits    = 32 // ti=13 i0 : `managed-object-property-name-component` (R(32))
	p2aScalarIdx  = 1  // i1 : le variant en mode A
	p2aPlayerIdx0 = 2  // i2 porte le joueur 0
	p2aPlayerN    = 32

	// Decoupage de l'en-tete d'un record delta d'objet du monde (`filmdec/projectiles.go`).
	p2aHeaderBits  = 21
	p2aIndexBits   = 6
	p2aMaxMaskCnt  = 7
	p2aTypeIndex13 = 13
)

// p2aPayloadBits rend la largeur de la charge utile d'un variant, TAG EXCLU. Recopie de
// `managedPropertyPayloadBits` : les branches muettes rendent 0 (la garde `index < 0x20` du jeu).
func p2aPayloadBits(tag int, modeA bool) int {
	if modeA {
		switch tag {
		case p2aTagEnum:
			return 4
		case p2aTagBool:
			return 1
		case p2aTagQuant:
			return p2aQuantBits
		case p2aTagU32, p2aTagStringID, p2aTagU32Bis:
			return 32
		}
		return 0
	}
	switch tag {
	case p2aTagQuantJ:
		return p2aQuantBits
	case p2aTagU32J, p2aTagStringIDJ:
		return 32
	case p2aTagBoolJ:
		return 1
	}
	if tag >= p2aTagEnumJ {
		return 4
	}
	return 0
}

// p2aEch est une valeur de ti=13 lue dans le flux, datee sur l'horloge du MANIFESTE (la meme
// que `objectiveevents.StatRecords` : `startMS` du chunk + delta du paquet).
type p2aEch struct {
	tMS  int
	slot uint32
	// idx est l'index du composant dans l'archetype : 1 = scalaire, 2..33 = par joueur.
	idx    int
	tag    int
	pay    uint64
	hasPay bool
}

// p2aScan porte ce qu'une passe a recolte, avec ses temoins d'ancrage.
type p2aScan struct {
	scal []p2aEch // i1, mode A
	joue []p2aEch // i2..i33, mode B
	// records / walked / chained : le denominateur de la recopie et ses deux gardes.
	records, walked, chained int
	// Le chainage decompose, parce que le taux global ne se lit pas seul : la phase 1 a mesure
	// qu'en Strongholds le trafic APPARENT des composants par joueur est de la contamination
	// d'ancrage (0 % de chainage, sous la bande fantome), alors que le canal scalaire, lui,
	// chaine. Melanger les deux populations dans un taux unique cacherait exactement cela.
	walkedScal, chainedScal int
	walkedJoue, chainedJoue int
	// decale : le chainage teste 3 bits plus loin — le niveau du HASARD structurel de la garde.
	decale int
	// t0MS / t1MS bornent le film sur l'horloge du manifeste.
	t0MS, t1MS int
	// noms recolte, par slot, les valeurs d'i0 (le NOM de la propriete reseau).
	noms map[uint32]map[uint64]int
	// bandeSlots est le nombre de slots de la bande ti=13 (denominateur de l'ancrage).
	bandeSlots int
}

// p2aRequireFilm rend le repertoire du film, ou saute le test.
func p2aRequireFilm(t *testing.T) string {
	t.Helper()
	dir := os.Getenv(p2aFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : mesure de la phase 2a sautee", p2aFilmEnv)
	}
	return dir
}

// p2aOutDir rend le repertoire de sortie des TSV et le cree au besoin.
func p2aOutDir(t *testing.T) string {
	t.Helper()
	if v := os.Getenv(p2aOutEnv); v != "" {
		p2aMkdir(t, v)
		return v
	}
	out := filepath.Join(repoRootForTest(t), ".ai", "V7.5", "replay2d", "registre_film", "lotCbis")
	p2aMkdir(t, out)
	return out
}

func p2aMkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creation de %s : %v", dir, err)
	}
}

// p2aCheckRegistre verifie que l'archetype 13 du film porte les noms attendus aux index
// attendus. LE DECOUPAGE DU REGISTRE CHANGE AVEC LE BUILD (lot 0) : sans ce controle, la
// recopie de grammaire pourrait s'appliquer a un tout autre archetype sans que rien ne le dise.
func p2aCheckRegistre(t *testing.T, dir string) {
	t.Helper()
	raw, err := filmdec.ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("lecture de chunk_00 (registre) : %v", err)
	}
	reg, err := filmdec.ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("analyse du registre : %v", err)
	}
	arch, ok := reg.Archetype(p2aTypeIndex13)
	if !ok {
		t.Fatalf("archetype 13 absent du registre de %s", dir)
	}
	attendu := map[int]string{
		0:                              "managed-object-property-name-component",
		p2aScalarIdx:                   "managed-object-property-component",
		p2aPlayerIdx0:                  "managed-object-player-masked-property-component",
		p2aPlayerIdx0 + p2aPlayerN - 1: "managed-object-player-masked-property-component",
	}
	for i, nom := range attendu {
		if i >= len(arch.Components) || arch.Components[i] != nom {
			t.Fatalf("archetype 13 : i%d = %q, attendu %q — le decoupage du registre a change,"+
				" la grammaire recopiee ne s'applique pas", i, p2aComp(arch.Components, i), nom)
		}
	}
	if n := len(arch.Components); n != p2aPlayerIdx0+p2aPlayerN {
		t.Logf("NOTE : archetype 13 a %d composants (attendu %d) — les index au-dela sont ignores",
			n, p2aPlayerIdx0+p2aPlayerN)
	}
}

func p2aComp(cs []string, i int) string {
	if i < 0 || i >= len(cs) {
		return "(absent)"
	}
	return cs[i]
}

// p2aBande rend les slots vus porter ti=13 dans les tables d'image-cle, PURGES des slots vus
// porter un autre archetype (pool de slots qui reboucle). Recopie reduite de `zcBuildBands` :
// la phase 2a n'a pas besoin des bandes de controle, son temoin est le HASARD des appariements.
func p2aBande(dir string) map[uint32]bool {
	tis := map[uint32]map[int]bool{}
	for ch := 1; ch <= filmdec.CountFilmChunks(dir); ch++ {
		data, err := filmdec.ReadFilmChunk(dir, ch)
		if err != nil {
			continue
		}
		for _, pk := range filmdec.WalkPackets(data) {
			if pk.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			for _, r := range filmdec.WalkKeyframeWorld(pk.Payload(data)) {
				s := uint32(r.Slot)
				if tis[s] == nil {
					tis[s] = map[int]bool{}
				}
				tis[s][r.TI] = true
			}
		}
	}
	band := map[uint32]bool{}
	for s, seen := range tis {
		if len(seen) == 1 && seen[p2aTypeIndex13] {
			band[s] = true
		}
	}
	return band
}

// p2aRecord est l'en-tete reconnu d'un record delta d'objet du monde (recopie de
// `filmdec.WorldObjectRecord` et de son reconnaisseur).
type p2aRecord struct {
	Slot  uint32
	Idx   []int
	After int
}

func p2aMatchRecord(pay []byte, p int, band map[uint32]bool) (p2aRecord, bool) {
	var rec p2aRecord
	if filmdec.PeekBits(pay, p, 1) != 1 { // prefixe de record DELTA
		return rec, false
	}
	slot := uint32(filmdec.PeekBits(pay, p+1, 13))
	if !band[slot] {
		return rec, false
	}
	if filmdec.PeekBits(pay, p+16, 2) != 0 { // porte de masque = 0 -> branche eparse
		return rec, false
	}
	mc := int(filmdec.PeekBits(pay, p+18, 3))
	if mc < 1 || mc > p2aMaxMaskCnt {
		return rec, false
	}
	idx := make([]int, mc)
	prev := -1
	for k := 0; k < mc; k++ {
		v := int(filmdec.PeekBits(pay, p+p2aHeaderBits+p2aIndexBits*k, p2aIndexBits))
		if v <= prev {
			return rec, false
		}
		idx[k], prev = v, v
	}
	rec.Slot, rec.Idx, rec.After = slot, idx, p+p2aHeaderBits+p2aIndexBits*mc
	return rec, true
}

// p2aReplay rejoue les composants annonces d'un record ti=13 et recolte leurs valeurs. Rend la
// position du bit de fin et l'aboutissement — c'est l'image exacte de `zsReplay` (phase 1), a
// ceci pres que la grammaire est lue ici au lieu d'etre appelee dans `filmdec`.
func p2aReplay(pay []byte, rec p2aRecord, tMS int, sc *p2aScan) (int, bool) {
	br := filmdec.NewBitReader(pay)
	br.SetBitPos(rec.After)
	for _, i := range rec.Idx {
		if i < 0 || i >= p2aPlayerIdx0+p2aPlayerN {
			return br.BitPos(), false
		}
		if br.Remaining() < p2aTagBits {
			return br.BitPos(), false
		}
		if i == 0 {
			// i0 = `managed-object-property-name-component` : R(32), LE NOM DE LA PROPRIETE.
			// Il est recolte par slot — c'est la seule cle qui nomme l'objet ti=13 lui-meme,
			// et elle explique la structure : un slot ti=13 n'est pas une zone, c'est UNE
			// PROPRIETE RESEAU (nom + valeur scalaire + 32 valeurs par joueur).
			if br.Remaining() < p2aNomBits {
				return br.BitPos(), false
			}
			nom := br.ReadBits(p2aNomBits)
			if sc.noms[rec.Slot] == nil {
				sc.noms[rec.Slot] = map[uint64]int{}
			}
			sc.noms[rec.Slot][nom]++
			continue
		}
		modeA := i == p2aScalarIdx
		tag := int(br.ReadBits(p2aTagBits))
		e := p2aEch{tMS: tMS, slot: rec.Slot, idx: i, tag: tag}
		if n := p2aPayloadBits(tag, modeA); n > 0 {
			if br.Remaining() < n {
				return br.BitPos(), false
			}
			e.pay, e.hasPay = br.ReadBits(uint(n)), true
		}
		if modeA {
			sc.scal = append(sc.scal, e)
			continue
		}
		sc.joue = append(sc.joue, e)
	}
	return br.BitPos(), true
}

// p2aScanFilm balaye le film et recolte les valeurs de ti=13, datees sur l'horloge du manifeste.
//
// UN SEUL FILM PAR PROCESSUS (memoire de depot : le balayage de corpus est une bombe RAM).
func p2aScanFilm(t *testing.T, dir string, startMS map[int]int) *p2aScan {
	t.Helper()
	band := p2aBande(dir)
	sc := &p2aScan{bandeSlots: len(band), t0MS: -1, noms: map[uint32]map[uint64]int{}}
	if len(band) == 0 {
		return sc
	}
	for ch := 1; ch <= filmdec.CountFilmChunks(dir); ch++ {
		data, err := filmdec.ReadFilmChunk(dir, ch)
		if err != nil {
			continue
		}
		st, ok := startMS[ch]
		if !ok {
			continue
		}
		var base uint64
		haveBase := false
		for _, pk := range filmdec.WalkPackets(data) {
			if pk.Type != filmdec.PacketTypeDelta {
				continue
			}
			if !haveBase {
				base, haveBase = pk.TimestampUS, true
			}
			tMS := st + int((pk.TimestampUS-base)/1000)
			p2aNoteTemps(sc, tMS)
			p2aScanPayload(pk.Payload(data), band, tMS, sc)
		}
	}
	sort.SliceStable(sc.scal, func(i, j int) bool { return sc.scal[i].tMS < sc.scal[j].tMS })
	sort.SliceStable(sc.joue, func(i, j int) bool { return sc.joue[i].tMS < sc.joue[j].tMS })
	return sc
}

func p2aNoteTemps(sc *p2aScan, tMS int) {
	if sc.t0MS < 0 || tMS < sc.t0MS {
		sc.t0MS = tMS
	}
	if tMS > sc.t1MS {
		sc.t1MS = tMS
	}
}

// p2aScanPayload ancre les records d'un payload, les rejoue, et compte le CHAINAGE.
func p2aScanPayload(pay []byte, band map[uint32]bool, tMS int, sc *p2aScan) {
	limit := len(pay)*8 - (p2aHeaderBits + p2aIndexBits)
	for p := 0; p <= limit; p++ {
		rec, ok := p2aMatchRecord(pay, p, band)
		if !ok {
			continue
		}
		sc.records++
		end, done := p2aReplay(pay, rec, tMS, sc)
		if done {
			ch := p2aHeaderAt(pay, end)
			sc.walked++
			if ch {
				sc.chained++
			}
			if p2aHeaderAt(pay, end+3) {
				sc.decale++
			}
			p2aCompteChainage(sc, rec, ch)
		}
		p = rec.After // meme avance qu'en phase 1 : l'en-tete consomme n'est pas re-balaye
	}
}

// p2aCompteChainage range le record dans la population SCALAIRE (masque limite a i0/i1) ou dans
// la population PAR JOUEUR (au moins un composant i2..i33), et compte son chainage.
func p2aCompteChainage(sc *p2aScan, rec p2aRecord, chained bool) {
	scal := true
	for _, i := range rec.Idx {
		if i >= p2aPlayerIdx0 {
			scal = false
			break
		}
	}
	if scal {
		sc.walkedScal++
		if chained {
			sc.chainedScal++
		}
		return
	}
	sc.walkedJoue++
	if chained {
		sc.chainedJoue++
	}
}

// p2aHeaderAt dit si un en-tete de record d'objet du monde commence a la position p — sans
// exiger que son slot appartienne a la bande.
//
// C'EST LA DEFINITION DE LA PHASE 0 (`ti13HeaderAt`), et elle est volontairement STRUCTURELLE :
// le record qui SUIT un record ti=13 dans un paquet appartient le plus souvent a un autre
// archetype. Exiger la bande ferait chuter le taux sans rien dire de la grammaire — le chainage
// mesure si la LARGEUR LUE tombe juste, pas qui parle ensuite.
func p2aHeaderAt(pay []byte, p int) bool {
	total := len(pay) * 8
	if p < 0 || p+p2aHeaderBits+p2aIndexBits > total {
		return false
	}
	if filmdec.PeekBits(pay, p, 1) != 1 || filmdec.PeekBits(pay, p+16, 2) != 0 {
		return false
	}
	mc := int(filmdec.PeekBits(pay, p+18, 3))
	if mc < 1 || mc > p2aMaxMaskCnt || p+p2aHeaderBits+p2aIndexBits*mc > total {
		return false
	}
	prev := -1
	for k := 0; k < mc; k++ {
		v := int(filmdec.PeekBits(pay, p+p2aHeaderBits+p2aIndexBits*k, p2aIndexBits))
		if v <= prev {
			return false
		}
		prev = v
	}
	return true
}

// p2aSeries regroupe les echantillons scalaires d'un tag par slot, tries dans le temps.
func p2aSeries(es []p2aEch, tag int) map[uint32][]p2aEch {
	out := map[uint32][]p2aEch{}
	for _, e := range es {
		if e.tag == tag && e.hasPay {
			out[e.slot] = append(out[e.slot], e)
		}
	}
	for s := range out {
		ss := out[s]
		sort.SliceStable(ss, func(i, j int) bool { return ss[i].tMS < ss[j].tMS })
		out[s] = ss
	}
	return out
}

// p2aSlotsTries rend les cles d'une carte de slots, triees.
func p2aSlotsTries[T any](m map[uint32]T) []uint32 {
	out := make([]uint32, 0, len(m))
	for s := range m {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
