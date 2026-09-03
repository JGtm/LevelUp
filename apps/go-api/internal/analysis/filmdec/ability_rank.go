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

import (
	"fmt"

	"levelup/go-api/internal/analysis/filmsource"
)

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
// LES LECTURES À PORTE OUVERTE SONT ÉCARTÉES ici, et c'est le contrat d'AbilityRank.Rank :
// une identité non transmise n'est pas une identité nulle. L'autre vue du même composant,
// `ScanFilmEquipmentChanges`, les GARDE — pour elle, « le joueur n'a plus d'équipement » EST
// l'événement. Les deux vues partagent le balayage `walkAbilityEmissions` : le composant n'a
// qu'un seul lecteur.
//
// UN SEUL DÉCODAGE filmdec À LA FOIS PAR PROCESS : ce balayage installe `abilitySetHook`,
// qui est un global de paquet. L'appelant doit détenir LockProcessDecode (BuildFromFilm le
// fait). Le hook est restauré à la sortie, y compris en cas d'erreur.
//
// ScanFilmAbilityRanks est l'ENVELOPPE D2, HORS PRODUCTION : elle charge le film, ouvre un
// contexte pour elle seule, puis appelle [ScanAbilityRanks]. La cuisson, elle, passe le contexte
// qu'elle partage entre tous ses balayages.
func ScanFilmAbilityRanks(dir string) ([]AbilityRank, AbilityRankStats, error) {
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		return nil, AbilityRankStats{}, err
	}
	return ScanAbilityRanks(NewFilmContext(film))
}

// ScanAbilityRanks decode les identites de capacite d'un film DEJA CHARGE. Cf.
// [ScanFilmAbilityRanks] pour la doctrine du balayage.
func ScanAbilityRanks(fc *FilmContext) ([]AbilityRank, AbilityRankStats, error) {
	var out []AbilityRank
	st, err := walkAbilityEmissions(fc, func(e abilityEmission) {
		if e.Rank == AbilitySetNoRank {
			return
		}
		// Les deux types ont la MÊME forme et des CONTRATS différents — l'un peut porter la
		// porte ouverte, l'autre jamais —, ce qui est exactement pourquoi ils restent deux :
		// la conversion est le point où le contrat se resserre, juste après le filtre.
		out = append(out, AbilityRank(e))
	})
	if err != nil {
		return nil, st, err
	}
	return out, st, nil
}

// abilityEmission est UNE lecture d'i48 telle que le déserialiseur la publie, LA PORTE
// OUVERTE COMPRISE. C'est la matière brute des deux vues du composant.
type abilityEmission struct {
	Slot        uint32
	Chunk       int
	PacketIndex int
	TimestampUS uint64
	Counter     uint32
	// Rank vaut AbilitySetNoRank quand la porte est ouverte : le joueur ne porte PAS de
	// capacité à cet instant. C'est une information, pas un défaut de lecture.
	Rank int
}

// walkAbilityEmissions est LE SEUL balayage d'i48 (règle des <= 2 copies, CLAUDE.md n°6) :
// `ScanFilmAbilityRanks` et `ScanFilmEquipmentChanges` le partagent, chacune ne différant
// que par ce qu'elle fait des émissions.
//
// Le hook est LA grammaire : c'est le déserialiseur lui-même qui publie, on ne relit pas les
// bits à côté de lui. Deux lecteurs du même champ divergeraient.
func walkAbilityEmissions(fc *FilmContext, visit func(abilityEmission)) (AbilityRankStats, error) {
	var st AbilityRankStats
	chunks := fc.ChunkNumbers()
	if len(chunks) == 0 {
		return st, ErrNoFilmChunk
	}
	slots := fc.BipedSlots()
	if slots.Count() == 0 {
		return st, fmt.Errorf("aucun slot biped (ti=%d) dans les keyframes du film", BipedTypeIndex)
	}
	lay, err := fc.I0Layout()
	if err != nil {
		return st, fmt.Errorf("découpage i0 illisible : %w", err)
	}
	arch, err := fc.bipedArchetype()
	if err != nil {
		return st, err
	}

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

	minRecord := bipedHeaderBits + bipedIndexBits*bipedMinMaskCnt + lay.TotalBits()
	for _, c := range chunks {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
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
				if maskHas(idx, i48Index) {
					st.WithI48++
					last.got = false
					if walkRecordTo(pay, i0, total, idx, lay, arch, i48Index) && last.got {
						st.Read++
						if last.rank == AbilitySetNoRank {
							st.Gated++
						}
						visit(abilityEmission{
							Slot: slot, Chunk: c, PacketIndex: pk.Index,
							TimestampUS: pk.TimestampUS,
							Counter:     last.counter, Rank: last.rank,
						})
					} else {
						st.Unread++
					}
				}
				p = i0 + lay.TotalBits()
			}
		}
	}
	return st, nil
}

// bipedArchetype rend l'archétype biped du registre du film (chunk_00), ANALYSE UNE FOIS par le
// contexte (lot 2, 2026-09-03 : cinq balayages delta le redemandaient, donc cinq re-analyses).
func (c *FilmContext) bipedArchetype() (Archetype, error) {
	arch, _, ok, err := c.archetype(BipedTypeIndex)
	if err != nil {
		return Archetype{}, err
	}
	if !ok {
		return Archetype{}, fmt.Errorf("archétype biped %d absent du registre", BipedTypeIndex)
	}
	return arch, nil
}

// maskHas dit si le masque du record annonce le composant d'index target.
func maskHas(idx []int, target int) bool {
	for _, id := range idx {
		if id == target {
			return true
		}
	}
	return false
}

// walkRecordTo marche les composants du masque avec les désers de PRODUCTION jusqu'à
// consommer celui d'index target — c'est cette consommation qui déclenche le hook. Rend
// false dès qu'un composant intermédiaire n'est pas porté ou que la marche déborde du
// payload : au-delà, la position du curseur ne serait plus digne de confiance, et lire du
// bruit vaut moins que ne rien lire.
//
// walkRecordTo s'exprime en UNE ligne de walkRecordComponents : la marche elle-même n'existe
// qu'à un seul exemplaire (règle des <= 2 copies, CLAUDE.md n°6).
func walkRecordTo(pay []byte, i0, total int, idx []int, lay I0Layout, arch Archetype, target int) bool {
	found := false
	walkRecordComponents(pay, i0, total, idx, lay, arch, func(id int) bool {
		if id == target {
			found = true
			return false
		}
		return true
	})
	return found
}

// walkRecordComponents est LE SEUL EXEMPLAIRE de la marche biped de production (règle des
// <= 2 copies) : i48 (ScanFilmAbilityRanks), i28 (ScanFilmCamoStates) et l'inventaire delta
// (ScanFilmInventoryDeltas) la partagent. La marche ti=37 vit à part (equipmentWalk.walk) :
// autre archétype, autre en-tête.
//
// Elle appelle visit(id) APRÈS la consommation de chaque composant du masque — c'est cette
// consommation qui a déclenché les hooks du déser, donc à cet instant la publication du
// composant `id` est disponible. visit rend false pour arrêter la marche (l'appelant a ce
// qu'il voulait) ; walkRecordComponents rend alors false comme sur un arrêt d'erreur : c'est
// visit, et lui seul, qui sait si la marche a abouti.
//
// LE PARCOURS S'INTERROMPT dès qu'un composant intermédiaire n'est pas porté ou que la marche
// déborde du payload : au-delà, la position du curseur ne serait plus digne de confiance, et
// lire du bruit vaut moins que ne rien lire.
//
// UN SEUL PARCOURS POUR N CIBLES. Appeler walkRecordTo une fois par composant recherché
// relirait le record autant de fois ; l'inventaire en veut six (i22, i30, i31, i33, i34,
// i47) et paierait six fois le même travail.
func walkRecordComponents(
	pay []byte, i0, total int, idx []int, lay I0Layout, arch Archetype, visit func(id int) bool,
) {
	at := i0 + lay.TotalBits() + i0TailBits
	for _, id := range idx[1:] {
		name := arch.component(id)
		if name == "" {
			return
		}
		br := NewBitReader(pay)
		br.SetBitPos(at)
		_, _, ported := consumeByName(br, name, uint32(BipedTypeIndex), arch.Level(id))
		if !ported || br.BitPos() > total {
			return
		}
		at = br.BitPos()
		if !visit(id) {
			return
		}
	}
}
