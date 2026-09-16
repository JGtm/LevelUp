package filmdec

// equipment_spawn_events.go — LE FAIT ÉCRIT « une PIÈCE a été engendrée » : l'événement de
// liste de type 103 `EquipmentSpawnedObject`, et la vie d'objet que sa deuxième référence
// DÉSIGNE.
//
// # CE QUE CE LECTEUR REMPLACE (D13 du plan `.ai/PLAN_DECODEUR_FILM_2026-09-13.md`)
//
// L'origine d'une pose d'équipement se décidait par une FENÊTRE TEMPORELLE de 200 ms entre la
// création de l'objet et la fin de la vie de son poseur (aujourd'hui le seul REPLI de
// `replay/equipment_origin.go`). Or le film ÉCRIT le déploiement d'une pièce, et il l'écrit sans
// ambiguïté : le type 103 désigne 216 des 216 poses de panneau de mur publiées du parc — dans
// les TROIS origines mesurées, y compris les 3 `dropped` et les 4 `unknown` qu'un panneau ne
// peut pas être. Une corrélation temporelle ne décide jamais un fait que le film écrit.
//
// # LA GRAMMAIRE, ET D'OÙ ELLE VIENT
//
// Un paquet delta s'ouvre sur [config(1)][continuation(1)][R(7) type] puis, pour l'événement de
// tête, TROIS références gardées (`[garde(1)][R(w) index][R(2) génération]`). Les DOMAINES du
// type 103 sont {0, 0, 7} — extraction mécanique des 123 descripteurs de l'exécutable (thunk
// `vtable+0x58`, lot R7, `r7_grammaire_research_test.go`) —, donc trois index de
// [dom7RefWidth] bits, sans sonde (elle n'existe que pour le domaine 1).
//
// LA BASE DE L'INDEX EST 512, ET C'EST UNE MESURE. Sur 931 occurrences de 25 films (rapport
// F.0 §1.2) : à base 512 la référence 1 se résout à une vie d'objet dans 93,6 % des cas contre
// 2,2 % au témoin de hasard — facteur 42 — et l'archétype dominant est l'objet du monde ; à
// base 0 le taux tombe à 6,7 % et l'archétype dominant redevient le bipède (le décalage fait
// retomber dans la bande des slots de joueur). Le dt médian entre la création et l'événement
// vaut +49 ms : l'événement suit la création d'une à deux images.
//
// # CE QUE CHAQUE RÉFÉRENCE EST, ET CE QU'ELLE N'EST PAS
//
//   - `ref1` (celle que ce lecteur publie) DÉSIGNE L'OBJET ENGENDRÉ : 231 vies `ti=37` et
//     635 vies `ti=41` (des projectiles — le 103 dit « un objet a été engendré », pas « un
//     équipement a été déployé »). Les objets `ti=37` nommés sont `0x528fce46` 227 fois et
//     `0x686b40c9` 3 fois : les DEUX panneaux de mur du manifeste, les deux seuls objets
//     `kind = "deployed"`.
//   - `ref0` désigne un `ti=37` que les images-clés voient (737 sur 739 recensées) et qu'AUCUNE
//     création delta ne porte (2,8 %) : une entité de longue durée, PISTE pour l'équipement
//     SOURCE d'un déploiement, NON INSTRUITE (table D du registre 0.E). Ce lecteur la rend
//     brute et ne l'interprète pas.
//   - `ref2` est ABSENTE : 3 portes posées sur 931.
//
// # LECTURE DE TÊTE SEULEMENT, ET LE CHIFFRE QUI LE JUSTIFIE
//
// Ce lecteur ne lit QUE l'événement de tête de chaque liste. Marcher la liste complète exige la
// table des largeurs de charge des 123 types (lot R7, six fichiers d'instruments) et une
// dérive de marche y coûterait tout le paquet. Or **927 des 931 occurrences du type 103 sont en
// position 1** (rapport F.0 §1.1) : la marche complète n'en ajoute que quatre. Les occurrences
// non lues sont un manque MESURÉ, compté par [EquipmentSpawnStats] au travers du dénominateur
// de listes, et la lecture ne dérive jamais — le cadrage de tête est certain.

// EventEquipmentSpawnedObject est le type de liste « une pièce a été engendrée ».
//
// NUMÉROTATION TRAME (établie le 2026-08-30) : toute numérotation antérieure à cette date est
// sans valeur, piège documenté en tête de `event_list.go`.
const EventEquipmentSpawnedObject = 103

// equipmentSpawnRefBase est la BASE de l'index des références du type 103 : le slot d'entité
// vaut `base + index`. 512, mesuré (cf. l'en-tête de ce fichier) — la même base que le type 117
// `EquipmentTranslocatorTeleportEffects`, établie indépendamment par le lot R1.
const equipmentSpawnRefBase = 512

// EquipmentSpawnEvent est UNE occurrence du type 103, avec la vie d'objet qu'elle désigne.
type EquipmentSpawnEvent struct {
	// Chunk / PacketIndex localisent l'événement dans le film.
	Chunk, PacketIndex int
	// TimestampUS est l'horodatage du paquet — MÊME horloge que
	// [EquipmentPlacement.T0US] et [EquipmentCreation.TimestampUS], donc croisable sans
	// recalage.
	TimestampUS uint64
	// Spawned est la vie d'objet ENGENDRÉE, telle que la référence 1 la désigne : la paire
	// (slot, génération), exactement la clé qu'un record de création écrit. Ne vaut que si
	// [EquipmentSpawnEvent.SpawnedValid].
	Spawned EquipmentLifeKey
	// SpawnedValid : la référence 1 portait sa garde. Faux sur 6 occurrences sur 931.
	SpawnedValid bool
	// Source est la PREMIÈRE référence, rendue BRUTE et non interprétée : elle désigne un
	// `ti=37` de longue durée que les images-clés voient et qu'aucune création delta ne porte.
	// PISTE pour l'équipement source d'un déploiement, non instruite (table D du registre 0.E).
	Source EquipmentLifeKey
	// SourceValid : la référence 0 portait sa garde (929 sur 931).
	SourceValid bool
	// Ref2Present : la TROISIÈME référence portait sa garde. Comptée et JAMAIS LUE : la mesure
	// du parc en trouve 3 sur 931, donc rien ne permet d'établir ce qu'elle désigne. Le compte
	// laisse un futur lot voir si un build la pose davantage, sans qu'aucune décision ne repose
	// dessus aujourd'hui.
	Ref2Present bool
}

// EquipmentSpawnStats dit ce que le balayage a vu — les dénominateurs sans lesquels un compte
// de zéro ne se distingue pas d'un film muet.
type EquipmentSpawnStats struct {
	// Chunks est le nombre de chunks lus ; Packets le nombre de paquets delta traversés.
	Chunks, Packets int
	// Lists est le nombre de paquets dont la liste d'événements n'est PAS vide — le
	// dénominateur de la lecture de tête.
	Lists int
	// Events est le nombre d'occurrences du type 103 lues en tête de liste.
	Events int
	// WithSpawned / WithSource comptent les références présentes ; Ref2 compte les troisièmes
	// références posées (attendu : ~3 sur 931, cf. l'en-tête).
	WithSpawned, WithSource, Ref2 int
}

// ScanEquipmentSpawnEvents lit les événements 103 de tête de liste d'un film DÉJÀ CHARGÉ.
//
// HORS LIGNE, LECTURE PURE : aucune décision, aucun seuil, aucune fenêtre. Ce que ce balayage
// rend est ce que le film écrit ; le rapprochement avec une pose est le travail de l'appelant.
func ScanEquipmentSpawnEvents(fc *FilmContext) ([]EquipmentSpawnEvent, EquipmentSpawnStats, error) {
	var st EquipmentSpawnStats
	nums := fc.ChunkNumbers()
	if len(nums) == 0 {
		return nil, st, ErrNoFilmChunk
	}
	var out []EquipmentSpawnEvent
	for _, c := range nums {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		st.Chunks++
		for _, p := range pks {
			if p.Type != PacketTypeDelta || p.Size < 1 {
				continue
			}
			st.Packets++
			ev, present, ok := decodeEquipmentSpawnEvent(p.Payload(data))
			if present {
				st.Lists++
			}
			if !ok {
				continue
			}
			ev.Chunk, ev.PacketIndex, ev.TimestampUS = c, p.Index, p.TimestampUS
			st.Events++
			if ev.SpawnedValid {
				st.WithSpawned++
			}
			if ev.SourceValid {
				st.WithSource++
			}
			if ev.Ref2Present {
				st.Ref2++
			}
			out = append(out, ev)
		}
	}
	if st.Chunks == 0 {
		return nil, st, ErrNoReadableFilmChunk
	}
	return out, st, nil
}

// decodeEquipmentSpawnEvent décode l'événement de tête d'un payload s'il est du type 103.
//
// Rend `present` (la liste d'événements n'est pas vide : le dénominateur de la lecture) et `ok`
// (la tête EST un 103). Les deux sont distincts parce qu'un zéro d'événements sur un film sans
// liste ne dit pas la même chose qu'un zéro sur un film qui en porte des milliers.
func decodeEquipmentSpawnEvent(pay []byte) (ev EquipmentSpawnEvent, present, ok bool) {
	typ, present := PacketHeadEventType(pay)
	if !present || typ != EventEquipmentSpawnedObject {
		return EquipmentSpawnEvent{}, present, false
	}
	// Domaines {0, 0, 7} : trois références SANS sonde, index de dom7RefWidth bits.
	r0 := readPlainRef(pay, eventPayloadStartBit, dom7RefWidth)
	r1 := readPlainRef(pay, r0.EndBit, dom7RefWidth)
	ev.Ref2Present = readPlainRef(pay, r1.EndBit, dom7RefWidth).Present
	if r0.Present {
		ev.Source, ev.SourceValid = spawnLifeKey(r0), true
	}
	if r1.Present {
		ev.Spawned, ev.SpawnedValid = spawnLifeKey(r1), true
	}
	return ev, true, true
}

// spawnLifeKey applique la base mesurée à l'index d'une référence et rend la clé de vie.
func spawnLifeKey(r guardedRef) EquipmentLifeKey {
	return EquipmentLifeKey{Slot: r.Index + equipmentSpawnRefBase, Gen: r.Gen}
}
