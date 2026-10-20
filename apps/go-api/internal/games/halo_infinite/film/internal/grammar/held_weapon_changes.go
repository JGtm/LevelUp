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
	"levelup/go-api/internal/games/weapons/filmshell"
)

// HeldWeaponChangeStats compte ce que le balayage a vu, pour que l'appelant puisse juger la
// couverture sans relire le film.
type HeldWeaponChangeStats struct {
	// Records est le nombre de records delta bipède reconnus : lus par la marche des trames, ou
	// rendus par l ancrage derrière elle ([lecturesBipedes]).
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
// `spawn` donne, pour un slot et un instant, ce que la vie portait au dernier relevé PASSÉ que
// l'appelant connaît (sa dotation de naissance, ou un relevé d'image-clé) — il sert à qualifier
// la PREMIÈRE émission d'un emplacement, dont l'état de départ vient du spawn et non du flux. Il
// peut être nil : les premières émissions sont alors qualifiées `taken` par défaut, ce qui
// surestime les prises.
//
// ScanFilmHeldWeaponChanges est l'ENVELOPPE D2, HORS PRODUCTION ; la cuisson appelle
// [ScanHeldWeaponChanges].
func ScanFilmHeldWeaponChanges(
	dir string, spawn SpawnPredicate,
) ([]types.HeldWeaponChange, HeldWeaponChangeStats, error) {
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		return nil, HeldWeaponChangeStats{}, err
	}
	return ScanHeldWeaponChanges(contexteDeBobine(film), spawn)
}

// SpawnState est ce qu'une vie portait au dernier relevé PASSÉ : l'ensemble de ses familles et,
// quand le relevé est une DOTATION DE NAISSANCE, la famille de chaque emplacement (lot M3.2).
type SpawnState struct {
	// Families est l'ensemble des familles portées.
	Families map[uint32]bool
	// ParEmplacement, quand il n'est pas nil, donne la famille de CHAQUE emplacement annoncé
	// (rang -> famille, [NoWeaponVariant] pour un emplacement vide). Une dotation de naissance
	// le porte ; un relevé d'image-clé, qui ne situe pas ses familles, non.
	ParEmplacement map[int]uint32
	// DebutDeVie est l'instant de la CRÉATION du corps qui occupe le slot (0 = inconnu). Un slot
	// se réattribue d'une vie à l'autre : une émission postérieure à une nouvelle création est
	// la PREMIÈRE de sa vie, jamais la suite de la vie précédente.
	DebutDeVie uint64
}

// SpawnPredicate rend ce que la vie du slot portait au dernier relevé passé avant `at`, et s'il
// en existe un (second retour). JAMAIS un relevé à venir : il effacerait une vraie prise faite
// avant lui. `DebutDeVie` est rendu MÊME sans relevé : c'est lui qui coupe la chaîne des
// émissions à chaque nouvelle vie du slot.
type SpawnPredicate func(slot uint32, at uint64) (SpawnState, bool)

// ScanHeldWeaponChanges décode les changements d'arme en main d'un film DEJA CHARGE.
func ScanHeldWeaponChanges(
	fc *FilmContext, spawn SpawnPredicate,
) ([]types.HeldWeaponChange, HeldWeaponChangeStats, error) {
	var st HeldWeaponChangeStats
	cfg, err := newHeldWeaponScan(fc)
	if err != nil {
		return nil, st, err
	}
	lu, err := fc.lecturesBipedes()
	if err != nil {
		return nil, st, err
	}
	var last struct {
		high, low uint32
		got       bool
	}
	obs := NouvelleObservation()
	obs.HeldWeaponHook = func(h, l uint32) { last.high, last.low, last.got = h, l, true }

	chaine := newHeldWeaponChain(spawn)
	var out []types.HeldWeaponChange
	st.Records = lu.examines
	for i := range lu.records {
		r := &lu.records[i]
		if !heldWeaponMaskHas(r, cfg.emplacements) {
			continue
		}
		st.WithComponent++
		r.parcourir(obs, func(id int) bool {
			rang, arme := cfg.emplacements[id]
			if !arme || !last.got {
				last.got = false
				return true
			}
			last.got = false
			st.Emissions++
			ch := types.HeldWeaponChange{
				TimestampUS: r.Packet.TimestampUS, Chunk: r.Chunk, Slot: r.Slot, SlotIndex: id,
				Emplacement: rang, Family: last.high, Low: last.low, Previous: noVariant,
			}
			if chaine.qualifier(&ch, r.Gen) {
				st.Repeats++
			}
			out = append(out, ch)
			return true
		})
	}
	return out, st, nil
}

// heldWeaponChain enchaîne les émissions d'un emplacement : chacune se lit contre la précédente
// de la MÊME VIE, et la première d'une vie contre son spawn. La vie est la clé (slot, génération du
// handle) depuis le lot J5.3 : deux corps successifs d'un slot ne se chaînent plus, même sans
// témoin de naissance (`DebutDeVie` reste la seconde coupure : la génération ne fait que 2 bits).
type heldWeaponChain struct {
	spawn SpawnPredicate
	prev  map[heldWeaponKey]heldWeaponPrev
}

// heldWeaponKey désigne un emplacement d'une VIE (par son index de composant).
type heldWeaponKey struct {
	vie  types.LifeKey
	comp int
}

// heldWeaponPrev est la dernière famille émise sur l'emplacement, et le début de la vie qui l'a
// émise — une émission d'une vie POSTÉRIEURE ne s'enchaîne pas sur elle.
type heldWeaponPrev struct {
	fam   uint32
	debut uint64
}

func newHeldWeaponChain(spawn SpawnPredicate) *heldWeaponChain {
	return &heldWeaponChain{spawn: spawn, prev: map[heldWeaponKey]heldWeaponPrev{}}
}

// qualifier pose `Previous` et `Kind` de l'émission `ch`, émise par la vie (ch.Slot, gen), et dit si elle
// RÉPÈTE la famille précédente de sa vie (propriété que le canal ne devrait jamais violer, cf. `Repeats`).
func (c *heldWeaponChain) qualifier(ch *types.HeldWeaponChange, gen uint32) (repete bool) {
	var sp SpawnState
	spOK := false
	if c.spawn != nil {
		sp, spOK = c.spawn(ch.Slot, ch.TimestampUS)
	}
	k := heldWeaponKey{types.LifeKey{Slot: ch.Slot, Gen: gen}, ch.SlotIndex}
	p, vu := c.prev[k]
	if vu && sp.DebutDeVie > p.debut {
		vu = false // une nouvelle vie a commencé sur ce slot depuis l'émission précédente
	}
	if vu {
		ch.Previous = p.fam
		repete = p.fam == ch.Family
	}
	qualifyHeldWeaponChange(ch, vu, sp, spOK)
	c.prev[k] = heldWeaponPrev{fam: ch.Family, debut: sp.DebutDeVie}
	return repete
}

// qualifyHeldWeaponChange qualifie un changement. Une émission qui SUIT une autre sur le même
// emplacement se lit contre elle ; quand elle en répète la famille, c'est une RÉ-ANNONCE. La
// PREMIÈRE émission d'un emplacement se juge contre le spawn :
//
//   - DOTATION DE NAISSANCE connue pour cet emplacement (lot M3.2) : la même famille est une
//     ré-annonce ; sinon le changement part de l'arme de naissance, qui devient `Previous` — une
//     prise sur emplacement vide, un échange, ou un lâcher qui NOMME l'arme lâchée. Les mains
//     nues de la dotation valent un emplacement vide ([qualifierContre]) ;
//   - sinon, un ENSEMBLE de familles (relevé d'image-clé passé) : une famille déjà portée n'est
//     qu'une ré-annonce, une famille absente est une acquisition ;
//   - un emplacement annoncé VIDE dont l'occupant n'est pas connu (ni dotation qui le situe, ni
//     émission antérieure de la vie) est une RÉ-ANNONCE, pas un lâcher : rien ne dit qu'une arme
//     l'occupait. La marche des trames, qui lit aussi les records sans position, en rend à chaque
//     mise en place des joueurs (l'emplacement 2 de chacun, avant ses premières positions).
func qualifyHeldWeaponChange(ch *types.HeldWeaponChange, hadPrevious bool, st SpawnState, ok bool) {
	if hadPrevious {
		qualifierContre(ch, ch.Previous)
		return
	}
	if prev, connu := st.ParEmplacement[ch.Emplacement]; ok && connu {
		qualifierContre(ch, prev)
		return
	}
	switch {
	case ch.Family == noVariant:
		ch.Kind = types.HeldWeaponRestated
	case ok && st.Families[ch.Family]:
		ch.Kind = types.HeldWeaponRestated
	default:
		ch.Kind = types.HeldWeaponTaken
	}
}

// qualifierContre qualifie `ch` contre la famille `prev` qui occupait son emplacement : la même
// famille est une ré-annonce, rien ne change ; un emplacement vidé est un lâcher, un emplacement
// vide rempli une prise, une autre famille un échange. Un lâcher et un échange nomment l'arme
// précédente.
//
// LES MAINS NUES VALENT « RIEN EN MAIN » ([armeEnMain]) : l'objet que le jeu remet à chaque
// naissance (`filmshell.IsUnarmedFamily`) n'est ni lâché, ni échangé, ni pris. Contre lui, une
// annonce vide est une ré-annonce, une arme une prise (sans `Previous`) ; une émission des mains
// nues après une arme est le lâcher de cette arme, emplacement publié vide. Seule la REMISE (les
// mains nues sur un emplacement vide) garde sa nature de prise : la publication l'écarte et la
// compte (`unarmedGrants`).
func qualifierContre(ch *types.HeldWeaponChange, prev uint32) {
	switch {
	case ch.Family == prev:
		ch.Kind = types.HeldWeaponRestated
	case prev == noVariant && filmshell.IsUnarmedFamily(ch.Family):
		ch.Kind = types.HeldWeaponTaken
	case !armeEnMain(prev) && !armeEnMain(ch.Family):
		ch.Kind = types.HeldWeaponRestated
	case !armeEnMain(ch.Family):
		ch.Previous, ch.Family, ch.Kind = prev, noVariant, types.HeldWeaponDropped
	case !armeEnMain(prev):
		ch.Previous, ch.Kind = noVariant, types.HeldWeaponTaken
	default:
		ch.Previous, ch.Kind = prev, types.HeldWeaponSwapped
	}
}

// armeEnMain dit si une famille d'emplacement est une ARME : ni l'emplacement vide, ni l'objet
// « mains nues », qui vaut rien en main.
func armeEnMain(fam uint32) bool {
	return fam != noVariant && !filmshell.IsUnarmedFamily(fam)
}

// heldWeaponScan porte la configuration résolue une fois pour un film.
type heldWeaponScan struct {
	// emplacements donne le RANG de chaque composant `weapon-state-type-info` (index de
	// composant -> rang), cf. [weaponEmplacements].
	emplacements map[int]int
}

// newHeldWeaponScan résout la configuration. Les index d'emplacement d'arme viennent des NOMS
// du registre du film, jamais de constantes : un index de composant est un numéro de build.
func newHeldWeaponScan(fc *FilmContext) (heldWeaponScan, error) {
	var s heldWeaponScan
	if len(fc.ChunkNumbers()) == 0 {
		return s, ErrNoFilmChunk
	}
	if fc.BipedSlots().Count() == 0 {
		return s, fmt.Errorf("aucun slot biped (ti=%d) dans les keyframes du film", BipedTypeIndex)
	}
	if _, err := fc.I0Layout(); err != nil {
		return s, fmt.Errorf("découpage i0 illisible : %w", err)
	}
	arch, err := fc.bipedArchetype()
	if err != nil {
		return s, err
	}
	s.emplacements = weaponEmplacements(arch)
	if len(s.emplacements) == 0 {
		return s, fmt.Errorf("aucun %s dans l'archétype biped du film", compWeaponStateTypeInfo)
	}
	return s, nil
}

// heldWeaponMaskHas dit si le masque du record annonce au moins un emplacement d'arme.
func heldWeaponMaskHas(r *recordBipedeLu, emplacements map[int]int) bool {
	for id := range emplacements {
		if r.annonce(id) {
			return true
		}
	}
	return false
}

// NoWeaponVariant est la sentinelle d'EMPLACEMENT VIDE, telle que le déserialiseur l'écrit
// quand la porte de présence est fermée. Exportée parce que les consommateurs
// (`types.HeldWeaponChange.Family`, `.Previous`) doivent pouvoir la tester sans redéclarer la valeur.
const NoWeaponVariant = noVariant
