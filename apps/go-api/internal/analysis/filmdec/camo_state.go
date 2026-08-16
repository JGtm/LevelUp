package filmdec

// camo_state.go — L'ÉTAT ACTIF DU CAMOUFLAGE, lu dans les paquets delta.
//
// CE QUE CE BALAYAGE LIT, ET D'OÙ VIENT LA RÈGLE. Le composant i28
// `unit-active-camo-state-component` (consumeUnitActiveCamoState, unit_weaponstate.go) est
// LE composant nommé camouflage du bipède. Sa mesure (2026-08-16, phase A de
// PLAN_ETAT_ACTIF_EQUIPEMENT) a établi :
//
//   - la « fraction principale » (R12 sous flag0/flag1) n'est JAMAIS transmise
//     (0 sur 13 902 lectures, 3 films) — le canal qui porte l'état est queue[1] ;
//   - queue[1] est un INTERRUPTEUR : 0 ou 4095, pas de rampe (courbe publiée : paliers
//     0 -> 4095 -> 0, plateau de 16,2 s) ;
//   - ses transitions sont EXCLUSIVES aux vies dont i48 transmet le rang 8 (camouflage,
//     famille A) : 39 transitions rang 8 sur 2 films, 0 sur 574 autres vies, témoin
//     inter-films `00ba2e1c` (0 vie rang 8) à 0 transition et 0 valeur 4095.
//
// L'activation se date au passage à 4095, la désactivation au retour à 0, à la précision
// de la retransmission près (le canal n'est transmis que lorsqu'il est au masque). Ce
// fichier PUBLIE les lectures de queue[1] ; l'assemblage du rejeu en fait des épisodes
// datés (cf. replay/equipment_episodes.go). L'exclusivité rang 8 était la VALIDATION du
// canal, pas une condition de lecture : l'état est publié pour toute vie qui le transmet.
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requête.
// L'appelant doit détenir LockProcessDecode (BuildFromFilm le fait) : le hook installé est
// un global de paquet.

import "fmt"

// camoComponentName est l'étiquette de registre d'i28 — celle par laquelle consumeByName
// route vers consumeUnitActiveCamoState. L'index d'itérateur est résolu PAR NOM dans le
// registre du film, jamais câblé : l'index est un numéro de build.
const camoComponentName = "unit-active-camo-state-component"

// camoChannelIndex est l'index, dans la queue 6 x (R1 + optR12) d'i28, de la voie qui
// porte l'état (queue[1]). Mesuré : queue[2] oscille partout (2048/615, non corrélée au
// rang) — ce n'est PAS l'état ; les autres voies sont quasi muettes.
const camoChannelIndex = 1

// Valeurs de l'interrupteur mesuré sur queue[1]. Toute autre valeur est publiée telle
// quelle (champ Q) et l'assemblage la compte sans l'interpréter — on n'invente pas un
// troisième état à un canal mesuré binaire.
const (
	CamoInactiveQ = 0
	CamoActiveQ   = 4095
)

// CamoRead est UNE transmission de la voie d'état du camouflage, localisée dans le film.
type CamoRead struct {
	// Slot est l'identifiant bas du biped porteur — le même que celui des trajectoires,
	// donc UNE VIE et non un joueur (le slot migre aux réapparitions).
	Slot uint32
	// Chunk / PacketIndex localisent la lecture dans le film.
	Chunk, PacketIndex int
	// TimestampUS est l'horodatage du paquet porteur — MÊME horloge que BipedPosition.
	TimestampUS uint64
	// Q est le quantum brut R(12) de queue[1]. Binaire mesuré : CamoInactiveQ ou
	// CamoActiveQ.
	Q uint16
}

// CamoStateStats compte ce que la marche a rencontré. Sans ces dénominateurs, une liste
// de lectures ne se juge pas.
type CamoStateStats struct {
	// Records est le nombre de records delta biped reconnus.
	Records int
	// WithI28 est le nombre de ces records dont le masque annonce i28.
	WithI28 int
	// Read / Unread : lectures abouties, et records dont la marche n'a pas atteint i28.
	Read, Unread int
	// NoChannel est le nombre de lectures abouties dont queue[1] n'était pas transmise —
	// une voie non transmise n'est pas une valeur, elle n'entre pas dans la liste.
	NoChannel int
}

// ScanFilmCamoStates décode les transmissions de la voie d'état du camouflage (i28
// queue[1]) dans les paquets delta du film de dir.
//
// UN SEUL DÉCODAGE filmdec À LA FOIS PAR PROCESS : ce balayage installe `camoStateHook`,
// qui est un global de paquet. L'appelant doit détenir LockProcessDecode (BuildFromFilm le
// fait). Le hook est restauré à la sortie, y compris en cas d'erreur.
func ScanFilmCamoStates(dir string) ([]CamoRead, CamoStateStats, error) {
	var st CamoStateStats
	n := CountFilmChunks(dir)
	if n == 0 {
		return nil, st, fmt.Errorf("aucun chunk film dans %s", dir)
	}
	chunks := make([]int, 0, n)
	for i := 1; i <= n; i++ {
		chunks = append(chunks, i)
	}
	slots := bipedSlotBand(dir, chunks)
	if len(slots) == 0 {
		return nil, st, fmt.Errorf("aucun slot biped (ti=%d) dans les keyframes de %s", BipedTypeIndex, dir)
	}
	lay, _, err := DetectI0Layout(dir)
	if err != nil {
		return nil, st, fmt.Errorf("découpage i0 illisible dans %s : %w", dir, err)
	}
	arch, err := bipedArchetype(dir)
	if err != nil {
		return nil, st, err
	}
	i28idx := -1
	if ids := arch.indicesOf(camoComponentName); len(ids) > 0 {
		i28idx = ids[0]
	}
	if i28idx < 0 {
		return nil, st, fmt.Errorf("composant %q absent de l'archétype biped de %s", camoComponentName, dir)
	}

	// Le hook est LA grammaire : c'est le déserialiseur lui-même qui publie, on ne relit
	// pas les bits à côté de lui (même règle que ScanFilmAbilityRanks).
	var last struct {
		q       uint16
		channel bool
		got     bool
	}
	prev := camoStateHook
	SetCamoStateHook(func(cs CamoState) {
		last.q, last.channel = cs.SubQ[camoChannelIndex], cs.SubPresent[camoChannelIndex]
		last.got = true
	})
	defer SetCamoStateHook(prev)

	var out []CamoRead
	minRecord := bipedHeaderBits + bipedIndexBits*bipedMinMaskCnt + lay.TotalBits()
	for _, c := range chunks {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			total := len(pay) * 8
			for p := 0; p+minRecord <= total; {
				i0, slot, idx, ok := matchBipedHeader(pay, p, total, slots, true, lay)
				if !ok {
					p++
					continue
				}
				st.Records++
				if maskHas(idx, i28idx) {
					st.WithI28++
					last.got = false
					switch {
					case !walkRecordTo(pay, i0, total, idx, lay, arch, i28idx) || !last.got:
						st.Unread++
					case !last.channel:
						st.Read++
						st.NoChannel++
					default:
						st.Read++
						out = append(out, CamoRead{
							Slot: slot, Chunk: c, PacketIndex: pk.Index,
							TimestampUS: pk.TimestampUS, Q: last.q,
						})
					}
				}
				p = i0 + lay.TotalBits()
			}
		}
	}
	return out, st, nil
}
