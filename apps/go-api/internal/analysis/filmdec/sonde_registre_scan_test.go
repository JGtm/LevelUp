package filmdec

// sonde_registre_scan_test.go — LE MOTEUR DE BALAYAGE des sondes F2/F3/F4/F5 du plan
// `.ai/V7.5/replay2d/PLAN_EXPLOITATION_REGISTRE_FILM.md` (lot F). Les verdicts sont dans
// `sonde_registre_verdicts_test.go` ; ce fichier ne fait que produire la matiere.
//
// CE QU'IL FAIT, ET POURQUOI IL EST UNIQUE. Quatre archetypes sont interroges (ti=47 splash,
// ti=4 high-frequency, ti=13 property-name, ti=10 managed-object pour le controle de F5) plus
// deux recenses (tacmap ti=34 et ti=30, F4). Les faire en quatre balayages coutait quatre
// lectures completes du film ; ils partagent donc UNE SEULE passe sur les paquets delta. C'est
// la regle D17 du plan appliquee au pied de la lettre : la machine de l'utilisateur paie les
// decodages, et une passe qui rend quatre reponses en vaut quatre qui en rendent une.
//
// AUCUN INDEX D'ARCHETYPE N'EST CABLE. Le lot 0 a mesure que le registre CHANGE avec le build
// (`06dfe6d9` : 116 archetypes / 1 031 slots contre 118 / 1 067 ailleurs) : un « ti=47 » ecrit
// en dur serait juste sur ce corpus et faux au prochain patch. Chaque archetype est donc
// retrouve DANS LE REGISTRE DU FILM par un composant qui le nomme.
//
// LES VALEURS VIENNENT DU DESERIALISEUR, PAS D'UN SECOND LECTEUR. La marche consomme les
// composants du masque avec `consumeByName` — le dispatch de PRODUCTION — et c'est lui qui
// declenche `SetProbeHook` (plomberie posee par le lot 0, item 0.6). Poser un lecteur a cote
// aurait fabrique un second decodeur du meme champ, qui aurait diverge au premier correctif.
//
// LA MARCHE S'ARRETE AU PREMIER COMPOSANT NON PORTE, et c'est la seule position tenable : au
// dela, le curseur n'est plus digne de confiance et les bits lus seraient du bruit presente
// comme une mesure.
//
// CHAQUE BANDE A SON FANTOME. L'ancrage « objet du monde » a un plancher de bruit tres eleve
// sur les bandes larges (mesure du lot C : 58 a 375 records par slot fantome). Un compte de
// records ne veut donc rien dire seul : chaque archetype est double d'une bande FANTOME de meme
// cardinalite, faite de slots jamais vus le porter, passee par le MEME code. C'est ce rapport,
// et non le compte brut, qui dit si un canal existe.
//
// LECTURE SEULE : aucun octet ecrit hors des TSV de sortie, aucune base, aucun artefact.
// SOUS GARDE D'ENVIRONNEMENT (PROBE_FILM). UN SEUL FILM PAR PROCESSUS.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 PROBE_FILM=<repo>/data/cache/film_chunks/7344d24f \
//	  PROBE_CACHE=<repo>/data/cache PROBE_SHORT=7344d24f PROBE_OBJTYPE=zone \
//	  PROBE_TSV=<repo>/.ai/V7.5/replay2d/registre_film/lotEF \
//	  go test ./internal/analysis/filmdec/ -run TestSondesRegistre -v -timeout 60m

import (
	"os"
	"sort"
	"testing"
)

const (
	probeFilmEnv    = "PROBE_FILM"    // repertoire des chunks du film
	probeCacheEnv   = "PROBE_CACHE"   // racine du cache (manifeste : horloge du match)
	probeShortEnv   = "PROBE_SHORT"   // identifiant court du film (cle du manifeste)
	probeObjTypeEnv = "PROBE_OBJTYPE" // famille d'objectif FOURNIE : zone | flag
	probeTSVEnv     = "PROBE_TSV"     // repertoire de sortie des TSV (facultatif)
)

// Les composants qui NOMMENT chaque archetype interroge. Un nom, jamais un numero.
const (
	compTacmapWaypointState = "tacmap-waypointstate" // ti=34 i7
	compTacmapPOIIcon       = "tacmap-poiicon"       // ti=30 i0
)

// probeRole designe le role d'un archetype dans ce balayage.
type probeRole int

const (
	roleSplash     probeRole = iota // ti=47 : F2
	roleHighFreq                    // ti=4  : F3
	rolePropName                    // ti=13 : F5
	roleManagedObj                  // ti=10 : controle de F5 (memes instants ?)
	roleTacmapWayp                  // ti=34 : F4
	roleTacmapPOI                   // ti=30 : F4
	probeRoleCount = 6
)

// probeRoleNoms donne, pour chaque role, le composant qui identifie son archetype et un libelle.
var probeRoleNoms = [probeRoleCount]struct{ comp, libelle string }{
	roleSplash:     {compSplashMessageStatic, "splash-message (F2)"},
	roleHighFreq:   {compHighFrequency, "high-frequency (F3)"},
	rolePropName:   {compManagedObjectPropName, "managed-object-property-name (F5)"},
	roleManagedObj: {compManagedObjectBoundaryVisibility, "managed-object (controle F5)"},
	roleTacmapWayp: {compTacmapWaypointState, "tacmap-waypointstate (F4)"},
	roleTacmapPOI:  {compTacmapPOIIcon, "tacmap-poiicon (F4)"},
}

// probeArch est un archetype resolu dans le registre du film.
type probeArch struct {
	ti    int
	arch  Archetype
	band  map[uint32]bool
	ghost map[uint32]bool
	// slotsKF est ce que les images-cle ont REELLEMENT montre.
	slotsKF map[uint32]bool
	// idxIdentifiant est l'index du composant qui nomme l'archetype dans SA grammaire.
	idxIdentifiant int
}

// probeCensus est le recensement des slots par archetype, lu dans les images-cle.
type probeCensus struct {
	parTI map[int]map[uint32]bool
	tous  map[uint32]bool
}

// probeEmission est UNE valeur rendue par le hook, situee dans le temps.
type probeEmission struct {
	ti    uint32
	comp  ProbeComponent
	vals  []uint64
	chunk int
	usPk  uint64 // horodatage du paquet (horloge du film)
	tMS   int    // horloge du MATCH, si le manifeste est disponible ; -1 sinon
}

// probeRecordStat compte ce qu'une bande a rendu.
type probeRecordStat struct {
	records, ghostRecords int
	// marches : marches de composants entamees ; abouties : allees jusqu'au bout du masque.
	marches, abouties int
	// parIndex[i] : records dont le masque annonce l'index i.
	parIndex map[int]int
	// instants : les us de paquet ou la bande a rendu au moins un record (pour F5).
	instants map[uint64]bool
	// horsGrammaire : records dont le masque annonce un index que l'archetype n'a pas. Un
	// record de ti=34 ne PEUT PAS annoncer i40 : ce taux mesure directement la contamination
	// de la bande, et c'est le seul controle de purete qui ne depende d'aucune hypothese.
	horsGrammaire int
	// avecIdentifiant : records dont le masque annonce le composant qui NOMME l'archetype.
	// C'est le compte que le lot C publiait ; le publier aussi ici rend les deux comparables.
	avecIdentifiant int
}

// probeMoisson est la recolte complete d'une passe.
type probeMoisson struct {
	archs     [probeRoleCount]*probeArch
	stats     [probeRoleCount]probeRecordStat
	emissions []probeEmission
	// deltaPaquets / chunks : denominateurs de la passe.
	deltaPaquets, chunks int
}

// probeDir rend le repertoire du film, ou saute le test.
func probeDir(t *testing.T) string {
	t.Helper()
	dir := os.Getenv(probeFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : sondes sautees", probeFilmEnv)
	}
	return dir
}

// probeRecenseKF recense, en UNE passe sur les images-cle, les slots de CHAQUE archetype.
func probeRecenseKF(dir string, n int) probeCensus {
	c := probeCensus{parTI: map[int]map[uint32]bool{}, tous: map[uint32]bool{}}
	for ch := 1; ch <= n; ch++ {
		data, err := ReadFilmChunk(dir, ch)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				if c.parTI[r.TI] == nil {
					c.parTI[r.TI] = map[uint32]bool{}
				}
				c.parTI[r.TI][uint32(r.Slot)] = true
				c.tous[uint32(r.Slot)] = true
			}
		}
	}
	return c
}

// probeResoutArchetypes retrouve chaque archetype par le NOM d'un de ses composants, PUIS
// tranche les ambiguites par le RECENSEMENT du film.
//
// L'AMBIGUITE EST REELLE ET ELLE A COUTE UNE PASSE : `high-frequency` est porte par DEUX
// archetypes (3 et 4) sur les films du corpus. Retenir le premier venu prenait ti=3, qui n'a
// AUCUN slot dans les images-cle, et la sonde F3 rendait « aucune emission » sur un canal qui
// en compte 35 000. On ne tranche donc pas par l'ordre du registre — un ordre n'est pas une
// identite — mais par ce que le film montre : l'archetype qui a des slots. A egalite, le plus
// peuple ; a egalite encore, le premier, et l'ambiguite est journalisee.
func probeResoutArchetypes(t *testing.T, reg *Registry, c probeCensus) [probeRoleCount]*probeArch {
	t.Helper()
	var out [probeRoleCount]*probeArch
	for r := 0; r < probeRoleCount; r++ {
		var vus []int
		for _, a := range reg.Archetypes {
			if len(a.indicesOf(probeRoleNoms[r].comp)) > 0 {
				vus = append(vus, a.Index)
			}
		}
		if len(vus) == 0 {
			t.Logf("  %-38s ABSENT du registre de ce film", probeRoleNoms[r].libelle)
			continue
		}
		choisi := vus[0]
		for _, ti := range vus {
			if len(c.parTI[ti]) > len(c.parTI[choisi]) {
				choisi = ti
			}
		}
		if len(vus) > 1 {
			t.Logf("  %-38s AMBIGU : porte par %v — ti=%d retenu (%d slots en image-cle)",
				probeRoleNoms[r].libelle, vus, choisi, len(c.parTI[choisi]))
		}
		a, _ := reg.Archetype(choisi)
		out[r] = &probeArch{ti: choisi, arch: a, idxIdentifiant: a.indicesOf(probeRoleNoms[r].comp)[0]}
		t.Logf("  %-38s ti=%-3d  %d composants  (composant identifiant en i%d)",
			probeRoleNoms[r].libelle, choisi, len(a.Components), out[r].idxIdentifiant)
	}
	return out
}

// probeBandes pose la bande de chaque archetype et son fantome.
//
// LA BANDE EST CELLE QUE LES IMAGES-CLE MONTRENT, SANS COMBLEMENT — et c'est l'inverse du
// choix fait pour les projectiles. La regle de comblement (`worldObjectSlotBand`) existe parce
// qu'un projectile vit moins d'une seconde et que les images-cle sont espacees de ~20 s :
// l'observe y rate l'essentiel. LES SIX ARCHETYPES SONDES SONT L'EXACT CONTRAIRE — ce sont des
// objets SCRIPTES du mode, qui vivent toute la partie et sont donc presents a CHAQUE image-cle.
// Combler ne recupere alors aucune couverture, ca ne fait qu'avaler les slots voisins :
// mesure de cette passe, ti=10 passait de 81 slots observes a 916 apres comblement, et de
// 35 000 a 334 000 records. On garde l'observe, purge des slots vus porter un autre archetype.
func probeBandes(archs [probeRoleCount]*probeArch, c probeCensus) {
	for _, a := range archs {
		if a == nil {
			continue
		}
		a.slotsKF = c.parTI[a.ti]
		a.band = make(map[uint32]bool, len(a.slotsKF))
		for s := range a.slotsKF {
			a.band[s] = true
		}
		a.ghost = probeFantome(a.slotsKF, c.tous, len(a.band))
	}
}

// probeFantome tire une bande temoin de meme cardinalite, faite de slots JAMAIS vus porter cet
// archetype, DANS LE MEME VOISINAGE NUMERIQUE que les slots reels : un balayage qui reconnait un
// slot sur 13 bits n'a pas la meme chance de tomber juste sur un petit numero que sur un grand,
// et un fantome tire depuis 1 gonflerait le temoin (lecon du lot des armes au sol).
func probeFantome(seen, tous map[uint32]bool, taille int) map[uint32]bool {
	ghost := map[uint32]bool{}
	if len(seen) == 0 || taille == 0 {
		return ghost
	}
	lo := uint32(kfTableCap)
	for s := range seen {
		if s < lo {
			lo = s
		}
	}
	for s := lo; s < kfTableCap && len(ghost) < taille; s++ {
		if seen[s] || tous[s] {
			continue
		}
		ghost[s] = true
	}
	for s := uint32(1); s < lo && len(ghost) < taille; s++ {
		if seen[s] || tous[s] {
			continue
		}
		ghost[s] = true
	}
	return ghost
}

// probeBalaye fait LA passe : une lecture des paquets delta, tous archetypes servis ensemble.
func probeBalaye(dir string, n int, archs [probeRoleCount]*probeArch, hor probeHorloge) probeMoisson {
	m := probeMoisson{archs: archs}
	union := map[uint32]bool{}
	roleDe := map[uint32]int{}
	fantomeDe := map[uint32]int{}
	for r, a := range archs {
		if a == nil {
			continue
		}
		m.stats[r] = probeRecordStat{parIndex: map[int]int{}, instants: map[uint64]bool{}}
		for s := range a.band {
			union[s], roleDe[s] = true, r
		}
		for s := range a.ghost {
			if !union[s] {
				union[s], fantomeDe[s] = true, r+1
			}
		}
	}
	var cur probeEmission
	prev := probeHookCourant()
	SetProbeHook(func(ti uint32, comp ProbeComponent, values []uint64) {
		e := cur
		e.ti, e.comp = ti, comp
		e.vals = append([]uint64(nil), values...)
		m.emissions = append(m.emissions, e)
	})
	defer SetProbeHook(prev)

	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		m.chunks++
		base := uint64(0)
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			if base == 0 {
				base = pk.TimestampUS
			}
			m.deltaPaquets++
			cur = probeEmission{chunk: c, usPk: pk.TimestampUS, tMS: hor.matchMS(c, pk.TimestampUS, base)}
			probeBalayePaquet(pk.Payload(data), union, roleDe, fantomeDe, archs, &m, &cur)
		}
	}
	return m
}

// probeBalayePaquet balaye UN payload delta et marche les records reconnus.
func probeBalayePaquet(pay []byte, union map[uint32]bool, roleDe, fantomeDe map[uint32]int,
	archs [probeRoleCount]*probeArch, m *probeMoisson, cur *probeEmission) {
	total := len(pay) * 8
	limit := total - (worldObjectHeaderBits + worldObjectIndexBits)
	for p := 0; p <= limit; p++ {
		rec, ok := matchWorldObjectRecord(pay, p, union)
		if !ok {
			continue
		}
		if r, isGhost := fantomeDe[rec.Slot]; isGhost {
			m.stats[r-1].ghostRecords++
			p = rec.After
			continue
		}
		r := roleDe[rec.Slot]
		a := archs[r]
		st := &m.stats[r]
		st.records++
		st.instants[cur.usPk] = true
		hors := false
		for _, i := range rec.Idx {
			st.parIndex[i]++
			if i >= len(a.arch.Components) {
				hors = true
			}
			if i == a.idxIdentifiant {
				st.avecIdentifiant++
			}
		}
		if hors {
			st.horsGrammaire++
		}
		st.marches++
		if probeMarche(pay, rec, archs[r], total) {
			st.abouties++
		}
		p = rec.After
	}
}

// probeMarche consomme les composants du masque avec le dispatch de PRODUCTION, dans l'ordre, et
// s'arrete au premier composant non porte ou au debordement du payload. Rend vrai si la marche a
// consomme TOUT le masque.
func probeMarche(pay []byte, rec WorldObjectRecord, a *probeArch, total int) bool {
	at := rec.After
	for _, id := range rec.Idx {
		name := a.arch.component(id)
		if name == "" {
			return false
		}
		br := NewBitReader(pay)
		br.SetBitPos(at)
		_, _, ported := consumeByName(br, name, uint32(a.ti), a.arch.Level(id))
		if !ported || br.BitPos() > total {
			return false
		}
		at = br.BitPos()
	}
	return true
}

// probeHookCourant rend le hook installe, pour le restaurer a la sortie.
func probeHookCourant() func(uint32, ProbeComponent, []uint64) { return probeHook }

// probeHorloge convertit l'horodatage d'un paquet en horloge du MATCH, quand le manifeste du
// film est disponible. Meme formule que `objectiveevents.StatRecords` — la seule qui aligne les
// emissions sur les evenements d'objectif :
//
//	tMS = debut du chunk (manifeste) + (us du paquet − us du premier paquet delta du chunk)/1000
type probeHorloge struct{ startMS map[int]int }

// matchMS rend l'instant sur l'horloge du match, ou -1 si le manifeste manque.
func (h probeHorloge) matchMS(chunk int, us, base uint64) int {
	if h.startMS == nil {
		return -1
	}
	s, ok := h.startMS[chunk]
	if !ok {
		return -1
	}
	return s + int((us-base)/1000)
}

// probeTriEmissions ordonne les emissions par instant de paquet : la passe les produit deja dans
// l'ordre des chunks, mais un tri explicite rend le resultat independant de cet ordre.
func probeTriEmissions(es []probeEmission) {
	sort.SliceStable(es, func(i, j int) bool {
		if es[i].chunk != es[j].chunk {
			return es[i].chunk < es[j].chunk
		}
		return es[i].usPk < es[j].usPk
	})
}
