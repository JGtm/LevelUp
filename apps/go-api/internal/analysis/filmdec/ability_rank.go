package filmdec

// ability_rank.go — LE RANG DE PALETTE DE LA CAPACITÉ PORTÉE, lu dans les paquets delta.
//
// CE QUE CE BALAYAGE LIT, ET POURQUOI IL EXISTE. Le composant i48
// (`biped-desired-ability-set-component`, cf. components_biped_ability.go) transmet
// l'IDENTITÉ de la capacité d'armure du joueur : R(3) compteur de rotation, R(1) porte,
// puis — porte fermée — R(6) le rang dans la palette `sofd` du match. Le déserialiseur
// consommait ces six bits pour rester aligné et les JETAIT ; ils sont désormais publiés par
// `abilitySetHook`, et ce fichier est ce qui les fait sortir du film.
//
// POURQUOI CE CANAL PLUTÔT QUE CELUI DES IMAGES-CLÉS. Le décodeur d'inventaire lit un champ
// de 3 bits ancré dans les images-clés (replay/inventory_decode.go, règle R1). Son motif
// d'ancrage se termine par `010`, qui sont les bits de POIDS FORT du rang : ce canal ne voit
// donc QUE les rangs 16 à 23 et rend `rang − 16` (RECETTE_LOADOUT §14). Il est exact dans
// sa fenêtre, et aveugle en dehors — d'où les 21 films sur 40 qui ne rendaient aucune
// lecture. i48 porte le rang COMPLET, sur toute la palette.
//
// CE QU'IL COÛTE, MESURÉ : i48 est RARE (0,03 à 0,09 % des records delta, à peu près une
// transmission par vie) mais LU À 100 % — 748 lectures sur 8 films, zéro illisible. Une vie
// qui n'en porte aucune n'a pas d'identité par ce canal : c'est un trou assumé, pas une
// occasion de deviner.
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requête.

import "fmt"

// i48Index est l'index d'itérateur du composant `biped-desired-ability-set-component` dans
// l'archétype biped (cf. components_biped_ability.go, section i48).
const i48Index = 48

// AbilityRank est UNE transmission d'identité de capacité, localisée dans le film.
type AbilityRank struct {
	// Slot est l'identifiant bas du biped porteur — le même que celui des trajectoires, donc
	// UNE VIE et non un joueur (le slot migre aux réapparitions).
	Slot uint32
	// Chunk / PacketIndex localisent la lecture dans le film.
	Chunk, PacketIndex int
	// TimestampUS est l'horodatage du paquet porteur — MÊME horloge que BipedPosition.
	TimestampUS uint64
	// Counter est le compteur de rotation R(3). Il n'identifie rien à lui seul ; il est
	// conservé parce qu'il BORNE l'interprétation (une valeur hors 0..7 dirait que la lecture
	// est mal placée).
	Counter uint32
	// Rank est le rang dans la palette du match. Jamais AbilitySetNoRank : les lectures dont
	// la porte est ouverte ne sont pas émises — une identité non transmise n'est pas une
	// identité nulle.
	Rank int
}

// AbilityRankStats compte ce que la marche a rencontré. Sans ces dénominateurs, un
// histogramme de rangs ne se juge pas.
type AbilityRankStats struct {
	// Records est le nombre de records delta biped reconnus.
	Records int
	// WithI48 est le nombre de ces records dont le masque annonce i48.
	WithI48 int
	// Read / Unread : lectures abouties, et records dont la marche n'a pas atteint i48 (un
	// composant intermédiaire non porté, ou un débordement du payload).
	Read, Unread int
	// Gated est le nombre de lectures abouties SANS identité (porte à 1).
	Gated int
}

// ScanFilmAbilityRanks décode les identités de capacité transmises par i48 dans les paquets
// delta du film de dir.
//
// UN SEUL DÉCODAGE filmdec À LA FOIS PAR PROCESS : ce balayage installe `abilitySetHook`,
// qui est un global de paquet. L'appelant doit détenir LockProcessDecode (BuildFromFilm le
// fait). Le hook est restauré à la sortie, y compris en cas d'erreur.
func ScanFilmAbilityRanks(dir string) ([]AbilityRank, AbilityRankStats, error) {
	var st AbilityRankStats
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

	// Le hook est LA grammaire : c'est le déserialiseur lui-même qui publie, on ne relit pas
	// les bits à côté de lui. Deux lecteurs du même champ divergeraient.
	var last struct {
		counter uint32
		rank    int
		got     bool
	}
	prev := abilitySetHook
	SetAbilitySetHook(func(counter uint64, rank, _ int) {
		last.counter, last.rank, last.got = uint32(counter), rank, true
	})
	defer SetAbilitySetHook(prev)

	var out []AbilityRank
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
				if maskHasI48(idx) {
					st.WithI48++
					last.got = false
					if walkRecordToI48(pay, i0, total, idx, lay, arch) && last.got {
						st.Read++
						if last.rank == AbilitySetNoRank {
							st.Gated++
						} else {
							out = append(out, AbilityRank{
								Slot: slot, Chunk: c, PacketIndex: pk.Index,
								TimestampUS: pk.TimestampUS,
								Counter:     last.counter, Rank: last.rank,
							})
						}
					} else {
						st.Unread++
					}
				}
				p = i0 + lay.TotalBits()
			}
		}
	}
	return out, st, nil
}

// bipedArchetype charge l'archétype biped depuis le registre du film (chunk_00).
func bipedArchetype(dir string) (Archetype, error) {
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		return Archetype{}, fmt.Errorf("chunk_00 (registre) illisible dans %s : %w", dir, err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		return Archetype{}, fmt.Errorf("registre illisible dans %s : %w", dir, err)
	}
	arch, ok := reg.Archetype(BipedTypeIndex)
	if !ok {
		return Archetype{}, fmt.Errorf("archétype biped %d absent du registre de %s", BipedTypeIndex, dir)
	}
	return arch, nil
}

// maskHasI48 dit si le masque du record annonce le composant i48.
func maskHasI48(idx []int) bool {
	for _, id := range idx {
		if id == i48Index {
			return true
		}
	}
	return false
}

// walkRecordToI48 marche les composants du masque avec les désers de PRODUCTION jusqu'à
// consommer i48 — c'est cette consommation qui déclenche le hook. Rend false dès qu'un
// composant intermédiaire n'est pas porté ou que la marche déborde du payload : au-delà, la
// position du curseur ne serait plus digne de confiance, et lire du bruit vaut moins que ne
// rien lire.
func walkRecordToI48(pay []byte, i0, total int, idx []int, lay I0Layout, arch Archetype) bool {
	at := i0 + lay.TotalBits() + i0TailBits
	for _, id := range idx[1:] {
		name := arch.component(id)
		if name == "" {
			return false
		}
		br := NewBitReader(pay)
		br.SetBitPos(at)
		_, _, ported := consumeByName(br, name, uint32(BipedTypeIndex), arch.Level(id))
		if !ported || br.BitPos() > total {
			return false
		}
		at = br.BitPos()
		if id == i48Index {
			return true
		}
	}
	return false
}
