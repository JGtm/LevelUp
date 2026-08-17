package filmdec

// game_entities_scan.go — LES DEUX ENTITES NON CORPORELLES DU FILM, LUES DANS LES PAQUETS
// DELTA : le MOTEUR DE PARTIE (ti=0, une entite) et le JOUEUR (ti=5, une par joueur).
//
// POURQUOI UN SEUL BALAYAGE POUR DEUX LOTS. Le lot B (horloge officielle, etats de partie,
// temps morts) veut ti=0 ; le lot P (l'entite joueur, l'inventaire) veut ti=5. Les deux
// entites vivent dans les MEMES paquets delta, se reconnaissent par le MEME en-tete de 21
// bits et se marchent avec les MEMES deserialiseurs de production. Deux balayages
// separes liraient deux fois les 11 a 46 Mo de chunks d'un film pour le meme resultat.
//
// CE QUE LE BALAYAGE FAIT, ET CE QU'IL NE FAIT PAS. Il ANCRE (reconnait un en-tete de record
// sur une bande de slots), il MARCHE les composants du masque avec les desers de production,
// et il rend les valeurs que les hooks du lot 0 et la couche de capture publient. Il
// n'interprete AUCUNE valeur : ni pente d'horloge, ni transition d'etat, ni famille d'arme.
// C'est le travail des instruments de mesure, et le garder dehors est ce qui permet de
// changer d'interpretation sans retoucher au decodage.
//
// LES DEUX TEMOINS QUE TOUT RESULTAT DOIT PORTER, parce que l'ancrage par bande de slots
// est un reconnaisseur probabiliste et non une lecture sequentielle :
//
//	LE PLANCHER DE BRUIT (bandes de controle). L'en-tete ne contraint que 21 bits, dont la
//	bande de slots ne fournit qu'une poignee : sur des centaines de millions de positions de
//	bit, il tombe juste par HASARD. Le lot C a chiffre ce fond a 58-375 records par slot sur
//	une bande FANTOME. Deux bandes de controle de meme cardinalite sont donc balayees en
//	meme temps que les vraies — l'une dans le voisinage numerique des slots reels, l'autre
//	dans le HAUT de l'espace de slots (le vide reel) — et leur debit par slot est publie a
//	cote du debit reel. Un archetype qui ne se detache pas de son temoin n'est pas mesure.
//
//	LA PURETE (temoin d'ancrage ti=4). Un record de ti=0 ne peut pas annoncer un index de
//	composant au-dela de la grammaire de son archetype. La proportion de records qui
//	respectent cette borne est la purete de la bande. Le lot C a mesure 98,7-99,8 % sur ti=4
//	(1 slot, 1 composant) : ti=4 est donc balaye ici aussi, comme etalon de la methode.
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requete.
//
// UN SEUL DECODAGE filmdec A LA FOIS PAR PROCESS : ce balayage installe `gameEngineHook`,
// `playerStateHook` et `probeHook`, qui sont des globaux de paquet. Les trois sont restaures
// a la sortie, y compris en cas d'erreur.

import (
	"fmt"
	"sort"
)

// GameEngineTypeIndex / PlayerEngineTypeIndex : les archetypes des deux entites lues ici.
// ProbeWitnessTypeIndex est le temoin d'ancrage du lot C (1 slot, 1 composant).
//
// CES TROIS NUMEROS SONT DES INDEX D'ARCHETYPE, ET ILS SONT CABLES — a la difference des
// index de COMPOSANT, qui sont resolus par nom. La raison est asymetrique et documentee :
// l'index d'archetype est le `ti` de 6 bits du record d'image-cle, la meme constante que
// `BipedTypeIndex`/`EquipmentTypeIndex` du paquet ; le decoupage qui bouge d'un build a
// l'autre (mesure du lot 0 : `06dfe6d9` 116/1 031 slots contre 118/1 067) est celui des
// COMPOSANTS a l'interieur d'un archetype.
const (
	GameEngineTypeIndex   = 0
	PlayerEngineTypeIndex = 5
	ProbeWitnessTypeIndex = 4
)

// Classes de bande de controle. Negatives pour ne jamais collisionner avec un `ti`.
const (
	// GameEntityClassNeighbour : slots libres tires dans le VOISINAGE NUMERIQUE des slots
	// reels. Un reconnaisseur de slot sur 13 bits n'a pas la meme chance de tomber juste sur
	// un petit numero que sur un grand : un temoin tire depuis 1 fausserait la comparaison.
	GameEntityClassNeighbour = -1
	// GameEntityClassVoid : slots du HAUT de l'espace, au-dela de tout slot observe. C'est le
	// vide reel, donc le bruit de reconnaissance pur.
	GameEntityClassVoid = -2
)

// gameEntityWantedEngine liste les composants de ti=0 que la marche vise : les cinq champs
// publies par `components_game_engine.go` PLUS le round-timer i5, qui reste sur la couche de
// capture (`capture.go`) et ne doit pas y etre duplique.
func gameEntityWantedEngine() []string {
	return []string{
		compGameEngineCurrentState,        // i2
		compGameEngineCurrentRound,        // i4
		compGameEngineRoundTimer,          // i5 (capture)
		compGameEngineSuddenDeath,         // i6
		compGameEngineGracePeriod,         // i7
		compGameEngineRoundConditionFlags, // i8
	}
}

// gameEntityWantedPlayer liste les composants de ti=5 que la marche vise : les onze champs
// publies par `components_player.go` PLUS le respawn-timer i1 (couche de capture).
func gameEntityWantedPlayer() []string {
	return []string{
		compPlayerRespawnTimer, // i1 (capture)
		compPlayerSoftKillTimer,
		compPlayerTargetTracking,
		compPlayerDesiredRespawnPlayer,
		compPlayerEngineLoadout,
		compPlayerDesiredRespawnLoc,
		compPlayerLivesRemaining,
		compPlayerLastBetrayer,
		compPlayerControlAiming,
		compPlayerActiveInGame,
		compPlayerPendingJoinInProgress,
		compPlayerMalleableProperties,
	}
}

// Etiquettes de registre des deux composants qui restent sur la couche de CAPTURE. Elles
// existent en litteral dans `capture.go` (`captureNames`) ; les nommer ici est la deuxieme
// copie, pas la troisieme, et le test `TestGameEntityCaptureNamesExist` echoue si l'une des
// deux cesse d'etre capturee.
const (
	compGameEngineRoundTimer = "game-engine-round-timer-component"
	compPlayerRespawnTimer   = "player-respawn-timer-component"
)

// GameEntityRecord est UN record delta de ti=0 ou ti=5 dont la marche des composants a
// abouti jusqu'au dernier composant vise du masque.
type GameEntityRecord struct {
	// TI est l'archetype (0 ou 5). Slot et Gen identifient l'entite ; contrairement aux
	// projectiles, ces deux entites vivent toute la partie et leur slot ne reboucle pas.
	TI        int
	Slot, Gen uint32
	// Chunk / PacketIndex / TimestampUS localisent la lecture (MEME horloge que
	// `BipedPosition.TimestampUS` et `FireEvent.TimestampUS`).
	Chunk, PacketIndex int
	TimestampUS        uint64
	// Idx est le masque du record, tel qu'annonce (indices strictement croissants).
	Idx []int
	// Engine* : les cinq champs de ti=0. Seen = le composant etait au masque ET la marche
	// l'a consomme ; Present = la porte de tete etait ouverte ; Val = les champs lus, dans
	// l'ordre du flux (contrat de `gameEngineHook`).
	EngineSeen, EnginePresent [GameEngineFieldCount]bool
	EngineVal                 [GameEngineFieldCount][]uint64
	// Player* : idem pour les onze champs de ti=5.
	PlayerSeen, PlayerPresent [PlayerStateFieldCount]bool
	PlayerVal                 [PlayerStateFieldCount][]uint64
	// RoundTimer (ti=0 i5) et Respawn (ti=5 i1) viennent de la couche de CAPTURE, donc deja
	// typees et dequantifiees pour l'horloge de manche.
	HasRoundTimer bool
	RoundTimer    RoundTimer
	HasRespawn    bool
	Respawn       RespawnTimer
}

// GameEntityStats compte ce que l'ancrage et la marche ont rencontre sur UNE classe (un
// archetype reel, ou une bande de controle). Sans ces denominateurs, aucun histogramme de
// valeurs ne se juge.
type GameEntityStats struct {
	// Records : en-tetes reconnus sur la bande. Walked / Broken : marches abouties et
	// interrompues (composant non porte, ou debordement du payload).
	Records, Walked, Broken int
	// InGrammar : records dont le masque tient ENTIEREMENT dans la grammaire de l archetype
	// de reference. WithWanted : records qui annoncent au moins un composant vise. Ces deux
	// compteurs sont tenus AUSSI sur les bandes de controle, avec la grammaire et la liste de
	// ti=5 : sans cela le temoin fantome ne serait pas comparable au chiffre reellement
	// utilise, et le rapport publie ne dirait rien de la mesure faite.
	InGrammar, WithWanted int
	// Slots : slots de la bande effectivement peuples ; BandSize : cardinalite de la bande.
	Slots, BandSize int
	// SlotSet est l ensemble des slots peuples — publie pour que le rapport puisse dire
	// QUELS slots ont parle, jamais seulement combien.
	SlotSet map[uint32]bool
	// MaskCensus[i] : records dont le masque annonce le composant d'index i, POUR TOUS les
	// index possibles — c'est le recensement qui dit OU le signal se trouve.
	MaskCensus [worldObjectMaxComponent]int
	// FirstIndex[i] : records dont i est le PREMIER index annonce. MaskCount[n] : records
	// annoncant n composants.
	FirstIndex, MaskCount map[int]int
	// OutOfGrammar : records dont le masque porte un index au-dela de la grammaire de
	// l'archetype — l'impurete de la bande. GrammarLen borne le controle (0 = pas de
	// controle, cas des bandes de controle qui n'ont pas d'archetype).
	OutOfGrammar, GrammarLen int
	// WithField[f] / Read[f] / Gated[f] : masque, lectures abouties, et lectures dont la
	// porte etait fermee, pour les champs de l'archetype de la classe.
	WithField, Read, Gated []int
}

func newGameEntityStats(grammarLen, fields int) *GameEntityStats {
	return &GameEntityStats{
		FirstIndex: map[int]int{}, MaskCount: map[int]int{}, SlotSet: map[uint32]bool{}, GrammarLen: grammarLen,
		WithField: make([]int, fields), Read: make([]int, fields), Gated: make([]int, fields),
	}
}

// RecordsPerSlot rend le debit par slot peuple — la grandeur qui rend les bandes comparables
// malgre des cardinalites differentes (1 slot pour ti=0, 8 pour ti=5).
func (s *GameEntityStats) RecordsPerSlot() float64 {
	if s == nil || s.Slots == 0 {
		return 0
	}
	return float64(s.Records) / float64(s.Slots)
}

// Purity rend la proportion de records dont le masque tient dans la grammaire de
// l'archetype, et sa validite (fausse quand la classe n'a pas de grammaire a opposer).
func (s *GameEntityStats) Purity() (float64, bool) {
	if s == nil || s.GrammarLen <= 0 || s.Records == 0 {
		return 0, false
	}
	return float64(s.Records-s.OutOfGrammar) / float64(s.Records), true
}

// NoiseFloor estime le PLANCHER DE BRUIT de la classe : la MEDIANE des annonces sur les 64
// index de composant possibles. Un tirage au hasard repartit les index a peu pres
// uniformement ; la mediane mesure donc ce fond DANS la bande elle-meme, sans hypothese
// exterieure. Grandeur reprise du lot C (`zcNoiseFloor`), meme definition.
func (s *GameEntityStats) NoiseFloor() float64 {
	if s == nil {
		return 0
	}
	v := make([]int, 0, worldObjectMaxComponent)
	for i := 0; i < worldObjectMaxComponent; i++ {
		v = append(v, s.MaskCensus[i])
	}
	sort.Ints(v)
	mid := len(v) / 2
	return (float64(v[mid-1]) + float64(v[mid])) / 2
}

// GameEntityScan est le resultat d'un balayage complet.
type GameEntityScan struct {
	// Engine / Player : les records dont la marche a abouti, dans l'ordre du film.
	Engine, Player []GameEntityRecord
	// Stats par classe : `GameEngineTypeIndex`, `PlayerEngineTypeIndex`,
	// `ProbeWitnessTypeIndex` (temoin de purete) et les deux classes de controle.
	Stats map[int]*GameEntityStats
	// Bands[classe] = les slots de la bande. Publie pour que le rapport puisse dire QUELS
	// slots ont ete lus, jamais seulement combien.
	Bands map[int]map[uint32]bool
	// Ambiguous : slots ecartes parce que vus porter plusieurs archetypes dans les
	// images-cles. Filled : ce qu'un comblement de plage aurait AJOUTE aux deux bandes
	// reelles — chiffre pour justifier de ne pas combler, jamais utilise pour ancrer.
	Ambiguous, Filled int
	// Packets / Chunks : paquets delta lus et chunks lus (denominateurs de temps).
	Packets, Chunks int
	// FirstPacketUS / LastPacketUS : bornes de l'horloge du film sur les paquets delta.
	FirstPacketUS, LastPacketUS uint64
	// FilmClockUS est l'horodatage du PREMIER paquet du chunk 1, c'est-a-dire le zero de
	// l'horloge du film (meme definition que `replay.ScanFilmClockOrigin`).
	FilmClockUS uint64
	// ProbeWitness compte les annonces du composant sonde de ti=4 recues par `probeHook`
	// pendant ce balayage — le temoin de periodicite demande au lot B.
	ProbeWitness int
	// KeyframeTICensus[ti] = nombre de SLOTS DISTINCTS vus porter cet archetype dans les
	// images-cles. Publie pour une raison mesuree : la premiere passe du lot B a trouve la
	// bande de ti=0 VIDE, et sans ce recensement on ne saurait pas si c'est l'archetype qui
	// n'existe pas, le marcheur d'image-cle qui ne le voit pas, ou la bande qui l'ecarte.
	KeyframeTICensus map[int]int
	// KeyframeSlotTI[slot] = les archetypes vus sur ce slot dans les images-cles. Publie pour
	// que le rapport puisse dire QUI occupe les premiers slots quand un archetype attendu est
	// introuvable : un compte par archetype ne repond pas a cette question.
	KeyframeSlotTI map[uint32][]int
}

// gameEntityCensus est le recensement des images-cles : slot -> archetypes vus.
type gameEntityCensus struct {
	slotTIs map[uint32]map[int]bool
}

// gameEntityKeyframeCensus lit TOUS les records d'image-cle du film et rend, pour chaque
// slot, l'ensemble des archetypes qu'il a portes.
func gameEntityKeyframeCensus(dir string, n int) gameEntityCensus {
	c := gameEntityCensus{slotTIs: map[uint32]map[int]bool{}}
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
				if r.Slot < 0 {
					continue
				}
				s := uint32(r.Slot)
				if c.slotTIs[s] == nil {
					c.slotTIs[s] = map[int]bool{}
				}
				c.slotTIs[s][r.TI] = true
			}
		}
	}
	return c
}

// gameEntityBands construit les bandes reelles et les deux bandes de controle.
//
// LA BANDE EST OBSERVEE, PAS COMBLEE — et c'est le contraire du choix fait pour les
// projectiles. Un projectile vit moins d'une seconde et n'apparait dans presque aucune
// image-cle : sa plage doit etre comblee pour ne pas perdre 90 % des vies
// (`worldObjectSlotBand`). Le moteur de partie et les entites joueur vivent TOUTE la partie :
// elles sont presentes a CHAQUE image-cle, donc combler ne recupere aucune couverture et ne
// peut qu'avaler les slots voisins. Le comblement est tout de meme CHIFFRE (`Filled`) pour
// que ce raisonnement soit verifiable et non seulement affirme.
func gameEntityBands(c gameEntityCensus) (map[int]map[uint32]bool, int, int) {
	bands := map[int]map[uint32]bool{
		GameEngineTypeIndex:   {},
		PlayerEngineTypeIndex: {},
		ProbeWitnessTypeIndex: {},
	}
	ambiguous, taken := 0, map[uint32]bool{}
	lo := uint32(kfTableCap)
	for slot, tis := range c.slotTIs {
		target := -1
		for ti := range tis {
			if _, ok := bands[ti]; ok {
				target = ti
			}
		}
		if target < 0 {
			continue
		}
		if len(tis) > 1 { // slot recycle : non attribuable, ecarte
			ambiguous++
			continue
		}
		bands[target][slot] = true
		taken[slot] = true
		if slot < lo {
			lo = slot
		}
	}
	filled := gameEntityFilledExcess(bands)
	real := len(bands[GameEngineTypeIndex]) + len(bands[PlayerEngineTypeIndex])
	bands[GameEntityClassNeighbour] = gameEntityControlBand(c, taken, lo, real, false)
	bands[GameEntityClassVoid] = gameEntityControlBand(c, taken, lo, real, true)
	return bands, ambiguous, filled
}

// gameEntityFilledExcess compte les slots qu'un comblement de plage AJOUTERAIT aux deux
// bandes reelles — la mesure qui justifie de ne pas combler.
func gameEntityFilledExcess(bands map[int]map[uint32]bool) int {
	excess := 0
	for _, ti := range []int{GameEngineTypeIndex, PlayerEngineTypeIndex} {
		for s := range fillSlotBand(bands[ti]) {
			if !bands[ti][s] {
				excess++
			}
		}
	}
	return excess
}

// gameEntityControlBand tire `size` slots LIBRES (jamais vus dans une image-cle, et non
// deja pris par une bande) : depuis `lo` vers le haut pour le temoin de voisinage, depuis le
// sommet de l'espace de slots vers le bas pour le temoin de vide.
func gameEntityControlBand(
	c gameEntityCensus, taken map[uint32]bool, lo uint32, size int, void bool,
) map[uint32]bool {
	out := map[uint32]bool{}
	if size <= 0 {
		return out
	}
	free := func(s uint32) bool { return !taken[s] && c.slotTIs[s] == nil }
	if void {
		for s := uint32(kfTableCap - 1); s > 0 && len(out) < size; s-- {
			if free(s) {
				out[s], taken[s] = true, true
			}
		}
		return out
	}
	for s := lo; s < kfTableCap && len(out) < size; s++ {
		if free(s) {
			out[s], taken[s] = true, true
		}
	}
	return out
}

// ScanFilmGameEntities balaye les paquets delta du film de `dir` et rend les records de ti=0
// et ti=5, avec les temoins d'ancrage (purete ti=4, deux bandes de controle).
func ScanFilmGameEntities(dir string) (GameEntityScan, error) {
	sc := GameEntityScan{Stats: map[int]*GameEntityStats{}}
	n := CountFilmChunks(dir)
	if n == 0 {
		return sc, fmt.Errorf("aucun chunk film dans %s", dir)
	}
	sc.Chunks = n
	reg, err := gameEntityRegistry(dir)
	if err != nil {
		return sc, err
	}
	census := gameEntityKeyframeCensus(dir, n)
	sc.KeyframeTICensus, sc.KeyframeSlotTI = map[int]int{}, map[uint32][]int{}
	for slot, tis := range census.slotTIs {
		for ti := range tis {
			sc.KeyframeTICensus[ti]++
			sc.KeyframeSlotTI[slot] = append(sc.KeyframeSlotTI[slot], ti)
		}
		sort.Ints(sc.KeyframeSlotTI[slot])
	}
	sc.Bands, sc.Ambiguous, sc.Filled = gameEntityBands(census)
	if len(sc.Bands[GameEngineTypeIndex]) == 0 && len(sc.Bands[PlayerEngineTypeIndex]) == 0 {
		return sc, fmt.Errorf("aucun slot d'archetype ti=%d ni ti=%d dans les images-cles de %s",
			GameEngineTypeIndex, PlayerEngineTypeIndex, dir)
	}
	w, err := newGameEntityWalk(reg, sc.Bands, sc.Stats)
	if err != nil {
		return sc, err
	}
	restore := w.install()
	defer restore()
	for ch := 1; ch <= n; ch++ {
		data, err := ReadFilmChunk(dir, ch)
		if err != nil {
			continue
		}
		pks := WalkPackets(data)
		if ch == 1 && len(pks) > 0 {
			sc.FilmClockUS = pks[0].TimestampUS
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta {
				continue
			}
			sc.Packets++
			if sc.FirstPacketUS == 0 || pk.TimestampUS < sc.FirstPacketUS {
				sc.FirstPacketUS = pk.TimestampUS
			}
			if pk.TimestampUS > sc.LastPacketUS {
				sc.LastPacketUS = pk.TimestampUS
			}
			w.scanPayload(pk.Payload(data), ch, pk, &sc)
		}
	}
	sc.ProbeWitness = w.probeCount
	for class, band := range sc.Bands {
		if st := sc.Stats[class]; st != nil {
			st.BandSize = len(band)
		}
	}
	return sc, nil
}

// gameEntityRegistry charge le registre du film (chunk_00).
func gameEntityRegistry(dir string) (*Registry, error) {
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		return nil, fmt.Errorf("chunk_00 (registre) illisible dans %s : %w", dir, err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		return nil, fmt.Errorf("registre illisible dans %s : %w", dir, err)
	}
	return reg, nil
}

// gameEntityWalk porte ce que la marche d'un record doit connaitre (regle des 5 parametres).
type gameEntityWalk struct {
	// arch[ti] : l'archetype du film pour les classes reelles.
	arch map[int]Archetype
	// want[ti][index d'iterateur] = vrai quand ce composant est vise par la marche.
	want map[int]map[int]bool
	// class[slot] : l'archetype (>= 0) ou la bande de controle (< 0) du slot.
	class map[uint32]int
	// all est l'union de toutes les bandes, sous la forme que `matchWorldObjectRecord`
	// attend. La table `class` ne peut pas lui servir directement (elle rend 0, qui est
	// l'archetype du moteur de partie, pour un slot ABSENT) : la separer est ce qui evite
	// qu'un slot inconnu soit ancre comme ti=0.
	all map[uint32]bool
	// stats est la table partagee avec l'appelant (une entree par classe).
	stats map[int]*GameEntityStats
	// cur accumule les valeurs publiees par les hooks pour le record en cours.
	cur GameEntityRecord
	// probeCount compte les annonces du composant sonde de ti=4.
	probeCount int
}

// newGameEntityWalk resout les archetypes du film et les index vises PAR NOM.
func newGameEntityWalk(
	reg *Registry, bands map[int]map[uint32]bool, stats map[int]*GameEntityStats,
) (*gameEntityWalk, error) {
	w := &gameEntityWalk{
		arch: map[int]Archetype{}, want: map[int]map[int]bool{},
		class: map[uint32]int{}, all: map[uint32]bool{}, stats: stats,
	}
	wanted := map[int][]string{
		GameEngineTypeIndex:   gameEntityWantedEngine(),
		PlayerEngineTypeIndex: gameEntityWantedPlayer(),
		ProbeWitnessTypeIndex: {compHighFrequency},
	}
	fields := map[int]int{
		GameEngineTypeIndex:   GameEngineFieldCount,
		PlayerEngineTypeIndex: PlayerStateFieldCount,
		ProbeWitnessTypeIndex: 1,
	}
	for ti, names := range wanted {
		arch, ok := reg.Archetype(ti)
		if !ok {
			return nil, fmt.Errorf("archetype ti=%d absent du registre du film", ti)
		}
		w.arch[ti] = arch
		w.want[ti] = map[int]bool{}
		for _, nm := range names {
			for _, id := range arch.indicesOf(nm) {
				w.want[ti][id] = true
			}
		}
		stats[ti] = newGameEntityStats(len(arch.Components), fields[ti])
	}
	// LES BANDES DE CONTROLE HERITENT DE LA GRAMMAIRE ET DE LA LISTE DE ti=5, la plus large
	// des deux entites mesurees. Un temoin fantome ne vaut que s'il subit EXACTEMENT les
	// memes filtres que la bande reelle : compter des en-tetes bruts d'un cote et des
	// en-tetes filtres de l'autre comparerait deux grandeurs differentes.
	for _, class := range []int{GameEntityClassNeighbour, GameEntityClassVoid} {
		stats[class] = newGameEntityStats(len(w.arch[PlayerEngineTypeIndex].Components), 0)
		w.want[class] = w.want[PlayerEngineTypeIndex]
	}
	for class, band := range bands {
		for s := range band {
			w.class[s], w.all[s] = class, true
		}
	}
	return w, nil
}

// install pose les trois hooks du lot 0 et rend la fonction de restauration.
func (w *gameEntityWalk) install() func() {
	prevEngine, prevPlayer, prevProbe := gameEngineHook, playerStateHook, probeHook
	SetGameEngineHook(func(f GameEngineField, values []uint64, present bool) {
		w.cur.EngineSeen[f], w.cur.EnginePresent[f] = true, present
		w.cur.EngineVal[f] = append(w.cur.EngineVal[f][:0], values...)
	})
	SetPlayerStateHook(func(f PlayerStateField, values []uint64, present bool) {
		w.cur.PlayerSeen[f], w.cur.PlayerPresent[f] = true, present
		w.cur.PlayerVal[f] = append(w.cur.PlayerVal[f][:0], values...)
	})
	SetProbeHook(func(_ uint32, comp ProbeComponent, _ []uint64) {
		if comp == ProbeHighFrequency {
			w.probeCount++
		}
	})
	return func() {
		SetGameEngineHook(prevEngine)
		SetPlayerStateHook(prevPlayer)
		SetProbeHook(prevProbe)
	}
}

// scanPayload balaye UN payload delta bit a bit et range les records reconnus.
//
// AUCUN FILTRE SUR LE PREMIER INDEX DU MASQUE, a la difference du balayage d'equipement qui
// exige `Idx[0] == 0` (la position ouvre le masque d'un objet du monde). Ni ti=0 ni ti=5 ne
// porte de composant de position : i0 y vaut `game-engine-team-mapping-component` et
// `player-waypoint-component`. Le filtre couterait donc toute la mesure, et son absence est
// exactement ce qui rend les deux bandes de controle indispensables.
func (w *gameEntityWalk) scanPayload(pay []byte, chunk int, pk FilmPacket, sc *GameEntityScan) {
	total := len(pay) * 8
	limit := total - (worldObjectHeaderBits + worldObjectIndexBits)
	for p := 0; p <= limit; p++ {
		rec, ok := matchWorldObjectRecord(pay, p, w.all)
		if !ok {
			continue
		}
		class, known := w.class[rec.Slot]
		if !known {
			continue
		}
		st := w.stats[class]
		gameEntityAccumulate(st, rec)
		if w.lastWanted(class, rec.Idx) >= 0 {
			st.WithWanted++
		}
		if class < 0 { // bande de controle : on compte, on ne marche pas
			p = rec.After - 1
			continue
		}
		w.walkRecord(walkSite{pay: pay, total: total, rec: rec, class: class, chunk: chunk, pk: pk}, st, sc)
		p = rec.After - 1
	}
}

// gameEntityAccumulate range UN en-tete reconnu dans les compteurs de sa classe.
func gameEntityAccumulate(st *GameEntityStats, rec WorldObjectRecord) {
	if st == nil {
		return
	}
	st.Records++
	st.SlotSet[rec.Slot] = true
	st.Slots = len(st.SlotSet)
	st.MaskCount[len(rec.Idx)]++
	st.FirstIndex[rec.Idx[0]]++
	impure := false
	for _, i := range rec.Idx {
		if i >= 0 && i < worldObjectMaxComponent {
			st.MaskCensus[i]++
		}
		if st.GrammarLen > 0 && i >= st.GrammarLen {
			impure = true
		}
	}
	if impure {
		st.OutOfGrammar++
		return
	}
	st.InGrammar++
}

// walkSite localise UNE marche : le payload, sa longueur, le record reconnu, sa classe et le
// paquet qui le porte. Les cinq voyagent toujours ensemble et n'ont de sens qu'ensemble ;
// les grouper est ce qui garde la marche sous la limite de parametres du depot.
type walkSite struct {
	pay   []byte
	total int
	rec   WorldObjectRecord
	class int
	chunk int
	pk    FilmPacket
}

// walkRecord marche les composants du masque jusqu'au dernier composant VISE, puis publie le
// record si la marche a abouti.
func (w *gameEntityWalk) walkRecord(s walkSite, st *GameEntityStats, sc *GameEntityScan) {
	class, rec := s.class, s.rec
	last := w.lastWanted(class, rec.Idx)
	if last < 0 {
		return
	}
	w.countMask(class, rec.Idx, st)
	w.cur = GameEntityRecord{
		TI: class, Slot: rec.Slot, Gen: rec.Gen, Chunk: s.chunk,
		PacketIndex: s.pk.Index, TimestampUS: s.pk.TimestampUS, Idx: rec.Idx,
	}
	if !w.walk(s.pay, rec.After, s.total, class, rec.Idx, last) {
		st.Broken++
		return
	}
	st.Walked++
	w.countRead(class, st)
	switch class {
	case GameEngineTypeIndex:
		sc.Engine = append(sc.Engine, w.cloneCurrent())
	case PlayerEngineTypeIndex:
		sc.Player = append(sc.Player, w.cloneCurrent())
	}
}

// cloneCurrent detache les tranches de valeurs du record courant : `install` les reutilise
// d'un record au suivant (append sur [:0]), donc les publier telles quelles ferait pointer
// tous les records sur le meme tampon.
func (w *gameEntityWalk) cloneCurrent() GameEntityRecord {
	out := w.cur
	for f := range out.EngineVal {
		out.EngineVal[f] = append([]uint64(nil), w.cur.EngineVal[f]...)
	}
	for f := range out.PlayerVal {
		out.PlayerVal[f] = append([]uint64(nil), w.cur.PlayerVal[f]...)
	}
	return out
}

// lastWanted rend le plus grand index du masque qui soit un composant vise, ou -1. La marche
// s'arrete la : pousser plus loin exposerait a un composant non porte sans rien apporter.
func (w *gameEntityWalk) lastWanted(class int, idx []int) int {
	last := -1
	for _, id := range idx {
		if w.want[class][id] && id > last {
			last = id
		}
	}
	return last
}

func (w *gameEntityWalk) countMask(class int, idx []int, st *GameEntityStats) {
	for _, id := range idx {
		if !w.want[class][id] {
			continue
		}
		if f := w.fieldOf(class, id); f >= 0 && f < len(st.WithField) {
			st.WithField[f]++
		}
	}
}

func (w *gameEntityWalk) countRead(class int, st *GameEntityStats) {
	switch class {
	case GameEngineTypeIndex:
		for f := 0; f < GameEngineFieldCount && f < len(st.Read); f++ {
			if !w.cur.EngineSeen[f] {
				continue
			}
			st.Read[f]++
			if !w.cur.EnginePresent[f] {
				st.Gated[f]++
			}
		}
	case PlayerEngineTypeIndex:
		for f := 0; f < PlayerStateFieldCount && f < len(st.Read); f++ {
			if !w.cur.PlayerSeen[f] {
				continue
			}
			st.Read[f]++
			if !w.cur.PlayerPresent[f] {
				st.Gated[f]++
			}
		}
	}
}

// fieldOf resout l'index de champ (au sens des enumerations du lot 0) d'un index de
// composant du registre. Les deux composants de la couche de capture n'ont pas de champ :
// ils rendent -1 et sont comptes par `HasRoundTimer`/`HasRespawn`.
func (w *gameEntityWalk) fieldOf(class, id int) int {
	name := w.arch[class].component(id)
	switch class {
	case GameEngineTypeIndex:
		for f := 0; f < GameEngineFieldCount; f++ {
			if GameEngineField(f).String() == name {
				return f
			}
		}
	case PlayerEngineTypeIndex:
		for f := 0; f < PlayerStateFieldCount; f++ {
			if PlayerStateField(f).String() == name {
				return f
			}
		}
	case ProbeWitnessTypeIndex:
		if name == compHighFrequency {
			return 0
		}
	}
	return -1
}

// walk marche les composants du masque avec les desers de PRODUCTION jusqu'a consommer celui
// d'index `last` — c'est cette consommation qui declenche les hooks et la capture. Rend faux
// des qu'un composant intermediaire n'est pas porte ou que la marche deborde du payload :
// au-dela, la position du curseur ne serait plus digne de confiance, et lire du bruit vaut
// moins que ne rien lire.
func (w *gameEntityWalk) walk(pay []byte, at, total, class int, idx []int, last int) bool {
	arch := w.arch[class]
	for _, id := range idx {
		if at > total {
			return false
		}
		name := arch.component(id)
		if name == "" {
			return false
		}
		br := NewBitReader(pay)
		br.SetBitPos(at)
		_, _, payload, ported := consumeByNameCapturing(br, name, uint32(class), arch.Level(id))
		if !ported || br.BitPos() > total {
			return false
		}
		w.capture(payload)
		at = br.BitPos()
		if id == last {
			return true
		}
	}
	return false
}

// capture range la valeur typee rendue par la couche de capture pour les deux composants qui
// y vivent (ti=0 i5 horloge de manche, ti=5 i1 compte a rebours de reapparition).
func (w *gameEntityWalk) capture(payload any) {
	switch v := payload.(type) {
	case RoundTimer:
		w.cur.HasRoundTimer, w.cur.RoundTimer = true, v
	case RespawnTimer:
		w.cur.HasRespawn, w.cur.Respawn = true, v
	}
}
