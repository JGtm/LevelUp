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
// l'autre (mesure du lot 0, comptes re-mesures au lot 3 : `06dfe6d9` 49 blocs/1 031 slots
// contre 50/1 067) est celui des COMPOSANTS a l'interieur d'un archetype.
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
