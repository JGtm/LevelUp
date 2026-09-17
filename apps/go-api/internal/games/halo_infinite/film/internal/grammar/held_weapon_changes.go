package grammar

// held_weapon_changes.go — LES CHANGEMENTS D'ARME EN MAIN, lus dans le flux delta.
//
// CE QUE C'EST. Le composant `weapon-state-type-info` (i43..i46 sur le bipède) porte l'IDENTITÉ
// de l'arme de chaque emplacement. En DELTA, il n'entre au masque que lorsque cette identité
// CHANGE : mesuré sur 171 851 records d'un film de référence, 31 émissions et ZÉRO répétition.
// Chaque émission est donc une prise, un lâcher ou un échange, daté à la milliseconde du paquet.
//
// CE QUE ÇA CORRIGE. Le négatif du 2026-08-12 (« le film ne porte aucun événement de
// ramassage ») visait l'archétype ARME AU SOL et un événement typé. Il n'est pas faux pour ce
// qu'il regardait : le signal est ailleurs, sur le porteur.
//
// CE QUE ÇA NE DONNE PAS. Ni le socle d'origine d'une prise, ni la fin de vie de l'arme lâchée.
// Le lâcher fait naître une entité `ti=42` dans le MÊME paquet (mesuré, écart médian nul), mais
// sa disparition n'est enregistrée que dans 5 à 14 % des cas — et c'est le comportement du jeu,
// pas un défaut de lecture : le despawn dépend de la position et du regard des joueurs
// (cf. `.ai/V7.5/reference/DESPAWN_ARMES_HALO_INFINITE.md`).
//
// HORS LIGNE par construction (I/O disque sur tout le film) — jamais depuis un chemin de requête.

import (
	"fmt"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// HeldWeaponChangeStats compte ce que le balayage a vu, pour que l'appelant puisse juger la
// couverture sans relire le film.
type HeldWeaponChangeStats struct {
	// Records est le nombre de records bipède ancrés dans le flux delta.
	Records int
	// WithComponent est le nombre de records dont le masque annonce un emplacement d'arme.
	WithComponent int
	// Emissions est le nombre de lectures d'identité effectivement obtenues.
	Emissions int
	// Repeats compte les émissions qui ne changent RIEN (même famille que la précédente sur
	// le même emplacement). Une valeur non nulle contredirait la propriété qui fonde ce
	// fichier — le composant ne devrait entrer au masque QUE sur changement.
	Repeats int
}

// ScanFilmHeldWeaponChanges décode tous les changements d'arme en main du film de `dir`.
//
// `spawnSet` donne, pour un slot et un instant, l'ensemble des familles portées au dernier
// relevé d'image-clé qui précède — il sert à qualifier la PREMIÈRE émission d'un emplacement,
// dont l'état de départ vient du spawn et non du flux. Il peut être nil : les premières
// émissions sont alors qualifiées `taken` par défaut, ce qui surestime les prises.
//
// ScanFilmHeldWeaponChanges est l'ENVELOPPE D2, HORS PRODUCTION ; la cuisson appelle
// [ScanHeldWeaponChanges].
func ScanFilmHeldWeaponChanges(
	dir string, spawnSet func(slot uint32, at uint64) (map[uint32]bool, bool),
) ([]types.HeldWeaponChange, HeldWeaponChangeStats, error) {
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		return nil, HeldWeaponChangeStats{}, err
	}
	return ScanHeldWeaponChanges(contexteDeBobine(film), spawnSet)
}

// ScanHeldWeaponChanges décode les changements d'arme en main d'un film DEJA CHARGE.
func ScanHeldWeaponChanges(
	fc *FilmContext, spawnSet func(slot uint32, at uint64) (map[uint32]bool, bool),
) ([]types.HeldWeaponChange, HeldWeaponChangeStats, error) {
	var st HeldWeaponChangeStats
	cfg, err := newHeldWeaponScan(fc)
	if err != nil {
		return nil, st, err
	}
	var last struct {
		high, low uint32
		got       bool
	}
	obs := NouvelleObservation()
	obs.HeldWeaponHook = func(h, l uint32) { last.high, last.low, last.got = h, l, true }
	cfg.gram.obs = obs

	type key struct {
		slot uint32
		comp int
	}
	prevFam, seen := map[key]uint32{}, map[key]bool{}
	var out []types.HeldWeaponChange
	walkDeltaBipedRecords(fc, cfg.chunks, cfg.slots, cfg.gram.lay, func(r deltaBipedRecord) {
		st.Records++
		if !heldWeaponMaskHas(r.Mask, cfg.weaponIdx) {
			return
		}
		st.WithComponent++
		walkRecordComponents(r.Payload, r.I0, r.Total, r.Mask, cfg.gram, func(id int) bool {
			if !cfg.weaponIdx[id] || !last.got {
				last.got = false
				return true
			}
			last.got = false
			st.Emissions++
			k := key{r.Slot, id}
			ch := types.HeldWeaponChange{
				TimestampUS: r.Packet.TimestampUS, Chunk: r.Chunk, Slot: r.Slot, SlotIndex: id,
				Family: last.high, Low: last.low, Previous: noVariant,
			}
			if seen[k] {
				ch.Previous = prevFam[k]
				if prevFam[k] == last.high {
					st.Repeats++
				}
			}
			ch.Kind = classifyHeldWeaponChange(ch, seen[k], spawnSet)
			out = append(out, ch)
			seen[k], prevFam[k] = true, last.high
			return true
		})
	})
	return out, st, nil
}

// classifyHeldWeaponChange qualifie un changement. La PREMIÈRE émission d'un emplacement se
// juge contre le loadout de spawn : une famille absente du spawn est une acquisition, une
// famille déjà présente n'est qu'une ré-annonce.
func classifyHeldWeaponChange(
	ch types.HeldWeaponChange, hadPrevious bool,
	spawnSet func(uint32, uint64) (map[uint32]bool, bool),
) types.HeldWeaponChangeKind {
	switch {
	case ch.Family == noVariant:
		return types.HeldWeaponDropped
	case hadPrevious && ch.Previous == noVariant:
		return types.HeldWeaponTaken
	case hadPrevious:
		return types.HeldWeaponSwapped
	}
	if spawnSet != nil {
		if set, ok := spawnSet(ch.Slot, ch.TimestampUS); ok && set[ch.Family] {
			return types.HeldWeaponRestated
		}
	}
	return types.HeldWeaponTaken
}

// heldWeaponScan porte la configuration résolue une fois pour un film.
type heldWeaponScan struct {
	chunks    []int
	slots     SlotBand
	gram      grammaireRecord
	weaponIdx map[int]bool
}

// newHeldWeaponScan résout la configuration. Les index d'emplacement d'arme viennent des NOMS
// du registre du film, jamais de constantes : un index de composant est un numéro de build.
func newHeldWeaponScan(fc *FilmContext) (heldWeaponScan, error) {
	var s heldWeaponScan
	s.chunks = fc.ChunkNumbers()
	if len(s.chunks) == 0 {
		return s, ErrNoFilmChunk
	}
	s.slots = fc.BipedSlots()
	if s.slots.Count() == 0 {
		return s, fmt.Errorf("aucun slot biped (ti=%d) dans les keyframes du film", BipedTypeIndex)
	}
	lay, err := fc.I0Layout()
	if err != nil {
		return s, fmt.Errorf("découpage i0 illisible : %w", err)
	}
	arch, err := fc.bipedArchetype()
	if err != nil {
		return s, err
	}
	s.gram = grammaireRecord{lay: lay, arch: arch, prof: fc.ProfilDeBalayage()}
	s.weaponIdx = map[int]bool{}
	for id := 0; id < archetypeBlockSlots; id++ {
		if arch.component(id) == compWeaponStateTypeInfo {
			s.weaponIdx[id] = true
		}
	}
	if len(s.weaponIdx) == 0 {
		return s, fmt.Errorf("aucun %s dans l'archétype biped du film", compWeaponStateTypeInfo)
	}
	return s, nil
}

// heldWeaponMaskHas dit si le masque annonce au moins un emplacement d'arme.
func heldWeaponMaskHas(idx []int, weaponIdx map[int]bool) bool {
	for _, id := range idx {
		if weaponIdx[id] {
			return true
		}
	}
	return false
}

// NoWeaponVariant est la sentinelle d'EMPLACEMENT VIDE, telle que le déserialiseur l'écrit
// quand la porte de présence est fermée. Exportée parce que les consommateurs
// (`types.HeldWeaponChange.Family`, `.Previous`) doivent pouvoir la tester sans redéclarer la valeur.
const NoWeaponVariant = noVariant
