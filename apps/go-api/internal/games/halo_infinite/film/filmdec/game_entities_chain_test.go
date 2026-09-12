package filmdec

// game_entities_chain.go — LA VOIE SEQUENTIELLE : DES RECORDS CERTAINS, MOINS NOMBREUX.
//
// POURQUOI CETTE SECONDE VOIE EXISTE, ET CE QU'ELLE CORRIGE. L'ancrage par bande
// (`game_entities_scan.go`) lit TOUS les paquets, mais c'est un reconnaisseur probabiliste :
// mesure faite sur `000d5950`, la bande de ti=5 rend 301 en-tetes par slot contre 283 sur une
// bande de slots VIDES. Le signal existe (deux valeurs d'i11 reviennent 37 et 8 fois, i20
// concentre 65 % de ses lectures sur deux tuples de 24 entrees — le hasard ne repete pas
// cela), mais il est noye dans un bruit du meme ordre. Aucun seuil de la phase 0 ne peut se
// juger sur un tel melange.
//
// LA CHAINE DE TRAMES, ELLE, NE DEVINE RIEN. `DecodeFrameRecords` lit les records d'un paquet
// EN SEQUENCE : chaque record est identifie par son en-tete, son archetype vient du World
// (bindings d'image-cle), et un record dont la traversee aboutit (`DesyncAt == -1`) est
// CERTAIN. Son prix est connu et mesure par le lot A : la chaine desynchronise 61-68 % des
// paquets delta, donc cette voie ne voit qu'un tiers du film. Un tiers de records certains
// vaut mieux que la totalite d'un melange a moitie faux — et les deux voies se controlent
// l'une l'autre, ce qu'aucune des deux ne pourrait faire seule.
//
// L'ATTRIBUTION DES VALEURS AUX RECORDS EST EXACTE, ET NON REORDONNEE. Les hooks publient dans
// l'ordre ou les deserialiseurs consomment ; `EntityTrace.Comps` liste les composants dans ce
// meme ordre. On depile donc un evenement de hook par composant PORTE dont le nom est l'un des
// champs suivis. Aucune heuristique d'appariement : c'est le meme ordre, lu deux fois.
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requete. Installe et
// restaure les hooks de paquet ; l'appelant detient `LockProcessDecode`.

import "fmt"

// GameChainStats compte ce que la voie sequentielle a rencontre. Le nom porte le prefixe
// `Game` parce que le paquet exporte deja un `ChainStats()` sans rapport (les compteurs de
// l inference de largeur de composant, `frame_chain_infer.go`) : deux noms proches auraient
// ete la meilleure facon de faire lire un chiffre pour un autre.
type GameChainStats struct {
	// Packets : paquets delta lus. PacketsClean : paquets dont la chaine est allee au bout
	// sans desynchroniser. Records : records lus ; RecordsClean : records aboutis.
	Packets, PacketsClean, Records, RecordsClean int
	// ByTI[ti] : records ABOUTIS par archetype, AVANT confirmation par les images-cles.
	// ByTIConfirmed[ti] : les memes, une fois la liaison slot -> archetype confirmee. L ecart
	// entre les deux EST la contamination par liaison NEW desalignee, et il se publie.
	ByTI, ByTIConfirmed map[uint32]int
	// EngineRecords / PlayerRecords : records aboutis de ti=0 et ti=5.
	EngineRecords, PlayerRecords int
	// Chunks : chunks lus.
	Chunks int
	// BipedRecords : records ABOUTIS de bipede (liaison confirmee). HeldWeaponReads : ceux
	// d entre eux qui portent une identite d arme en main. Le rapport des deux EST la
	// couverture du canal que P.0.3 mesure.
	BipedRecords, HeldWeaponReads int
	// BipedMask[i] : records de bipede confirmes dont le masque annonce le composant i.
	//
	// POURQUOI CE RECENSEMENT EST INDISPENSABLE A P.0.3. La premiere mesure a rendu ZERO arme
	// en main sur 18 041 records de bipede certains. Deux causes possibles, qui n appellent pas
	// la meme conclusion : soit i43-i46 ne sont jamais ANNONCES (le canal n existe pas dans le
	// film), soit ils le sont mais rendent toujours un variant vide (le deser ne sait pas les
	// lire). Sans le recensement du masque, les deux sont indistinguables.
	BipedMask [worldObjectMaxComponent]int
}

// HeldWeaponSample est UNE lecture d arme en main sur un slot de bipede, datee. CANAL RETIRE le
// 2026-08-18 (item 4 phase 1.0b : EntityTrace.HeldWeapon supprime, 0 appelant, canal mesure mort) :
// ce balayage ne produit plus aucun echantillon ; le type reste pour les lecteurs de la mesure.
//
// POURQUOI ELLE SORT DE CE BALAYAGE ET PAS D'UN AUTRE. `World.SetHeldWeapon` est alimente
// depuis la CHAINE de trames (`frame_records.go`), et par personne d'autre : la valeur
// n'existe que la ou un record de bipede a ete traverse en entier. La relever ici ne coute
// donc rien de plus qu'une affectation dans une passe deja faite, alors qu'un second
// balayage devrait refaire toute la chaine pour la meme information.
type HeldWeaponSample struct {
	// Slot est le slot du BIPEDE (celui des trajectoires), pas celui de l'entite joueur.
	Slot uint32
	// TimestampUS est l'horodatage du paquet — MEME horloge que `BipedPosition` et
	// `FireEvent`, donc directement croisable avec le flux des tirs.
	TimestampUS uint64
	// Family est le high-32 de l'identifiant d'arme, c'est-a-dire la FAMILLE — la meme cle
	// que `FireEvent.WeaponID >> 32` et que `KeyframeLoadout.Families`.
	Family uint32
}

// ScanFilmGameEntitiesChain lit le film par la CHAINE SEQUENTIELLE et rend les records de ti=0
// et ti=5 dont la traversee a abouti. Le World est reconstruit par chunk depuis les images-cles
// du chunk lui-meme (meme amorcage que le temoin de marche delta du paquet).
func ScanFilmGameEntitiesChain(dir string) ([]GameEntityRecord, []HeldWeaponSample, GameChainStats, error) {
	st := GameChainStats{ByTI: map[uint32]int{}, ByTIConfirmed: map[uint32]int{}}
	n := CountFilmChunks(dir)
	if n == 0 {
		return nil, nil, st, fmt.Errorf("aucun chunk film dans %s", dir)
	}
	st.Chunks = n
	reg, err := gameEntityRegistry(dir)
	if err != nil {
		return nil, nil, st, err
	}
	c := &chainScan{reg: reg, cfg: DefaultFrameConfig()}
	restore := c.install()
	defer restore()
	var out []GameEntityRecord
	for ch := 1; ch <= n; ch++ {
		data, err := ReadFilmChunk(dir, ch)
		if err != nil {
			continue
		}
		out = append(out, c.scanChunk(data, ch, &st)...)
	}
	return out, c.held, st, nil
}

// chainEvent est une publication de hook, telle qu'elle est sortie du deserialiseur.
type chainEvent struct {
	engine  bool
	field   int
	values  []uint64
	present bool
}

type chainScan struct {
	reg    *Registry
	cfg    FrameConfig
	events []chainEvent
	// held accumule les lectures d arme en main des records de bipede confirmes.
	held []HeldWeaponSample
}

// install branche les deux hooks sur la file d'evenements. Le hook de sonde n'est pas touche :
// la voie sequentielle n'a rien a dire de ti=4.
func (c *chainScan) install() func() {
	prevEngine, prevPlayer := gameEngineHook, playerStateHook
	SetGameEngineHook(func(f GameEngineField, values []uint64, present bool) {
		c.events = append(c.events, chainEvent{
			engine: true, field: int(f), values: append([]uint64(nil), values...), present: present,
		})
	})
	SetPlayerStateHook(func(f PlayerStateField, values []uint64, present bool) {
		c.events = append(c.events, chainEvent{
			field: int(f), values: append([]uint64(nil), values...), present: present,
		})
	})
	return func() {
		SetGameEngineHook(prevEngine)
		SetPlayerStateHook(prevPlayer)
	}
}

// scanChunk amorce le World depuis les images-cles du chunk, puis lit ses paquets delta.
//
// LA TABLE DES LIAISONS D'IMAGE-CLE EST CONSERVEE A PART, ET ELLE FAIT AUTORITE. Le World, lui,
// se laisse REECRIRE en cours de route : un record NEW qui suit une fausse-propre est
// desaligne, et son `R(6)` de tete lie alors le slot a un archetype FAUX (le danger est
// documente dans `DecodeFrameRecords` lui-meme). Mesure faite sur `000d5950` sans ce garde-fou :
// le slot 516 — un slot de BIPEDE — rendait 558 « records de ti=5 », et 9 405 records etaient
// comptes en ti=0 alors que le film n'a aucune entite de ce type (les records de SUPPRESSION
// n'ont pas de trace, donc leur `TypeIndex` vaut zero par defaut). Retenir un record seulement
// quand l'image-cle a liee CE slot a CET archetype supprime les deux artefacts d'un coup.
func (c *chainScan) scanChunk(data []byte, ch int, st *GameChainStats) []GameEntityRecord {
	w := NewWorld(c.reg)
	kfTI := map[uint32]uint32{}
	pks := WalkPackets(data)
	for _, pk := range pks {
		if pk.Type != PacketTypeKeyframe {
			continue
		}
		for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
			w.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI))
			if r.Slot >= 0 {
				kfTI[uint32(r.Slot)] = uint32(r.TI)
			}
		}
	}
	var out []GameEntityRecord
	for _, pk := range pks {
		if pk.Type != PacketTypeDelta {
			continue
		}
		st.Packets++
		c.events = c.events[:0]
		br := NewBitReader(pk.Payload(data))
		recs, err := DecodeFrameRecords(br, w, c.cfg)
		if err == nil {
			st.PacketsClean++
		}
		out = append(out, c.attribute(recs, kfTI, ch, pk, st)...)
	}
	return out
}

// attribute depile les evenements de hook sur les records, dans l'ordre de consommation, et
// rend ceux de ti=0 et ti=5 qui ont abouti.
func (c *chainScan) attribute(
	recs []FrameRecord, kfTI map[uint32]uint32, ch int, pk FilmPacket, st *GameChainStats,
) []GameEntityRecord {
	var out []GameEntityRecord
	next := 0
	for _, rec := range recs {
		st.Records++
		clean := rec.Type == recDelta && rec.DesyncAt == -1
		if clean {
			st.RecordsClean++
			st.ByTI[rec.TypeIndex]++
		}
		ti, bound := kfTI[rec.Slot]
		confirmed := clean && bound && ti == rec.TypeIndex
		if confirmed {
			st.ByTIConfirmed[rec.TypeIndex]++
		}
		cur := GameEntityRecord{
			TI: int(rec.TypeIndex), Slot: rec.Slot, Gen: rec.ID >> 30, Chunk: ch,
			PacketIndex: pk.Index, TimestampUS: pk.TimestampUS,
		}
		for _, comp := range rec.Trace.Comps {
			if !comp.Ported {
				continue
			}
			cur.Idx = append(cur.Idx, comp.Index)
			chainCapture(&cur, comp.Payload)
			if !chainIsTracked(comp.Name) {
				continue
			}
			if next >= len(c.events) {
				continue // file epuisee : on ne devine pas une valeur manquante
			}
			chainApply(&cur, c.events[next])
			next++
		}
		if !confirmed {
			continue
		}
		if rec.TypeIndex == BipedTypeIndex {
			st.BipedRecords++
			for _, comp := range rec.Trace.Comps {
				if comp.Index >= 0 && comp.Index < worldObjectMaxComponent {
					st.BipedMask[comp.Index]++
				}
			}
			continue
		}
		switch rec.TypeIndex {
		case GameEngineTypeIndex:
			st.EngineRecords++
			out = append(out, cur)
		case PlayerEngineTypeIndex:
			st.PlayerRecords++
			out = append(out, cur)
		}
	}
	return out
}

// chainTrackedNames est l'ensemble des composants dont un hook publie la valeur : ce sont eux,
// et eux seuls, qui font avancer la file d'evenements.
var chainTrackedNames = func() map[string]bool {
	m := map[string]bool{}
	for f := 0; f < GameEngineFieldCount; f++ {
		m[GameEngineField(f).String()] = true
	}
	for f := 0; f < PlayerStateFieldCount; f++ {
		m[PlayerStateField(f).String()] = true
	}
	return m
}()

func chainIsTracked(name string) bool { return chainTrackedNames[name] }

// chainApply range un evenement de hook dans le record courant.
func chainApply(cur *GameEntityRecord, ev chainEvent) {
	if ev.engine {
		if ev.field >= 0 && ev.field < GameEngineFieldCount {
			f := GameEngineField(ev.field)
			cur.EngineSeen[f], cur.EnginePresent[f], cur.EngineVal[f] = true, ev.present, ev.values
		}
		return
	}
	if ev.field >= 0 && ev.field < PlayerStateFieldCount {
		f := PlayerStateField(ev.field)
		cur.PlayerSeen[f], cur.PlayerPresent[f], cur.PlayerVal[f] = true, ev.present, ev.values
	}
}

// chainCapture range la valeur typee de la couche de capture (horloge de manche, compte a
// rebours de reapparition).
func chainCapture(cur *GameEntityRecord, payload any) {
	switch v := payload.(type) {
	case RoundTimer:
		cur.HasRoundTimer, cur.RoundTimer = true, v
	case RespawnTimer:
		cur.HasRespawn, cur.Respawn = true, v
	}
}
