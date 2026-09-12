package filmdec

// game_entities_walk_test.go — LA MARCHE DES COMPOSANTS D'UN RECORD ANCRE.
//
// Troisieme morceau de l'instrument des entites non corporelles (`game_entities_scan_test.go`
// en porte l'en-tete et le contrat) : la resolution des composants vises PAR NOM, la pose des
// hooks du lot 0, et la marche elle-meme, qui s'arrete au dernier composant vise. Scinde des
// deux autres pour tenir le seuil de 500 lignes par fichier.

import "fmt"

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
