package filmdec

import "levelup/go-api/internal/analysis/filmsource"

// equipment_changes.go — LES RAMASSAGES ET LES CONSOMMATIONS D'ÉQUIPEMENT, datés.
//
// CE QUE C'EST. L'équipement d'un joueur — la capacité d'armure : grappin, répulseur, mur de
// protection, capteur, propulseur, translocateur — est transmis par i48
// `biped-desired-ability-set-component`. En DELTA, ce composant n'entre au masque que lorsque
// l'équipement porté CHANGE. Chaque émission est donc un événement, daté à la milliseconde du
// paquet : le joueur porte désormais tel équipement, ou il n'en porte plus.
//
// LA MESURE QUI FONDE LE FICHIER (3 films, 319 émissions, 269 vies) :
//
//   - LE COMPTEUR R(3) AVANCE À CHAQUE ÉMISSION, sans une seule répétition sur 50 transitions.
//     Il repart à 5 à la première émission de chaque vie dans 264 cas sur 269 (98,1 %).
//     C'est ce qui fait de ce canal le seul du rejeu à porter son PROPRE TÉMOIN DE
//     COMPLÉTUDE : entre deux lectures, un pas différent de 1 (modulo 8) dénonce des
//     émissions manquées, et les compte. Mesuré : 12 sauts sur 50 transitions, soit environ
//     16 émissions manquées pour 319 vues — de l'ordre de 95 % de couverture, CHIFFRÉE et
//     non supposée. Les armes n'avaient rien de tel.
//   - LA PORTE OUVERTE EST LA CONSOMMATION, PAS LA MORT. Sur les 17 émissions à porte
//     ouverte des trois films, ZÉRO ne tombe dans la dernière seconde de la vie ; la plus
//     tardive laisse encore 8,8 s à vivre, la médiane 23 à 48 s. Le joueur a donc USÉ son
//     équipement — un événement de jeu à part entière, que le fil des morts ne dit pas.
//   - LA PREMIÈRE ÉMISSION D'UNE VIE N'A PAS UN SENS UNIQUE, et c'est pourquoi
//     `ScanFilmEquipmentChanges` exige un témoin de naissance. Sur deux films d'arène elle
//     tombe 16 à 18 s après la naissance du bipède (0 % sous la seconde) : c'est un
//     RAMASSAGE sur la carte. Sur un troisième elle tombe à 0 ms dans 83 % des cas : les
//     joueurs y réapparaissent AVEC leur équipement, et l'émission n'est qu'une annonce.
//     Compter l'une pour l'autre fausserait le décompte des ramassages du simple au double.
//
// CE QUE ÇA NE DONNE PAS. Ni le socle d'où vient l'équipement ramassé (même impasse que pour
// les armes : les trois hypothèses de lien vers l'objet du monde sont réfutées), ni ce que
// portait le joueur avant sa première émission.
//
// HORS LIGNE par construction (I/O disque sur tout le film) — jamais depuis un chemin de requête.

// EquipmentChangeKind qualifie un changement d'équipement porté.
type EquipmentChangeKind string

const (
	// EquipmentTaken : le joueur porte désormais cet équipement. C'est un RAMASSAGE.
	EquipmentTaken EquipmentChangeKind = "taken"
	// EquipmentSpent : le joueur n'en porte plus. Il l'a CONSOMMÉ (mesuré : jamais à la mort).
	EquipmentSpent EquipmentChangeKind = "spent"
	// EquipmentSpawned : première émission d'une vie, contemporaine de la naissance du
	// bipède — le joueur RÉAPPARAÎT avec cet équipement. Ce n'est PAS un ramassage.
	EquipmentSpawned EquipmentChangeKind = "spawned"
)

// equipmentSpawnWindowUS est l'écart maximal entre la naissance d'une vie et sa première
// émission i48 pour que celle-ci compte comme une annonce de réapparition.
//
// LE CORPUS SÉPARE NETTEMENT LES DEUX CAS et le seuil tombe dans le vide entre eux : sur le
// film où les joueurs réapparaissent équipés, les trois quarts des premières émissions
// tombent à 0 ms EXACTEMENT (même paquet que le premier échantillon de position) ; sur les
// films d'arène, le ramassage le plus précoce du corpus est à 1 134 ms. N'importe quelle
// valeur entre ~100 ms et ~1 100 ms donne le même classement.
const equipmentSpawnWindowUS = 1_000_000

// EquipmentChange est UN changement d'équipement porté, daté et attribué.
type EquipmentChange struct {
	// TimestampUS est l'horodatage du paquet — MÊME horloge que BipedPosition.TimestampUS.
	TimestampUS uint64
	// Chunk / PacketIndex localisent l'événement dans le film.
	Chunk, PacketIndex int
	// Slot est le slot du bipède : il désigne une VIE, pas un joueur.
	Slot uint32
	// Counter est le compteur de rotation R(3). Il est conservé parce qu'il est LE témoin de
	// complétude : un pas différent de 1 (modulo 8) entre deux émissions d'une même vie
	// dénonce des émissions manquées.
	Counter uint32
	// Rank est le rang de palette de l'équipement désormais porté, dans la même convention
	// qu'AbilityRank.Rank. Vaut AbilitySetNoRank sur EquipmentSpent.
	Rank int
	// Previous est le rang précédent sur cette vie, quand il est connu. AbilitySetNoRank
	// sinon — y compris à la première émission, dont l'état d'avant n'est pas lisible.
	Previous int
	// Kind qualifie le changement.
	Kind EquipmentChangeKind
}

// EquipmentChangeStats dit ce que le balayage a vu ET ce qu'il a MANQUÉ. Le second est la
// raison d'être de la structure : c'est le seul canal du rejeu qui sache s'auto-mesurer.
type EquipmentChangeStats struct {
	// Walk porte les dénominateurs du balayage (records, lectures, portes ouvertes).
	Walk AbilityRankStats
	// Lives est le nombre de vies ayant émis au moins une fois.
	Lives int
	// Repeats compte les transitions dont le compteur ne bouge PAS. Une valeur non nulle
	// contredirait la propriété qui fonde ce fichier — le composant ne devrait entrer au
	// masque QUE sur changement.
	Repeats int
	// CounterJumps compte les transitions dont le compteur avance d'autre chose que 1
	// (modulo 8), et MissedEstimate le nombre d'émissions que ces sauts impliquent.
	CounterJumps, MissedEstimate int
	// LivesFirstOffSpec compte les vies dont la PREMIÈRE émission n'a pas le compteur
	// attendu : des émissions antérieures ont été manquées, ou le slot a été mal ancré.
	LivesFirstOffSpec int
	// Spawned / Taken / Spent ventilent les changements rendus.
	Spawned, Taken, Spent int
}

// equipmentFirstCounter est la valeur du compteur R(3) à la première émission d'une vie,
// mesurée sur 264 vies de 269. Elle sert de témoin : une première émission qui ne la porte
// pas signale des émissions manquées en amont.
const equipmentFirstCounter = 5

// ScanFilmEquipmentChanges décode tous les changements d'équipement porté du film de `dir`.
//
// `bornAt` donne l'instant de naissance d'une vie — c'est lui qui distingue une annonce de
// réapparition d'un ramassage, et le balayage REFUSE de trancher sans lui : sans témoin, une
// première émission est rendue `EquipmentTaken`, ce qui SURESTIME les ramassages sur les
// modes où les joueurs réapparaissent équipés. L'appelant qui n'a pas de témoin doit le
// savoir.
//
// UN SEUL DÉCODAGE filmdec À LA FOIS PAR PROCESS (cf. ScanFilmAbilityRanks).
//
// ScanFilmEquipmentChanges est l'ENVELOPPE D2, HORS PRODUCTION ; la cuisson appelle
// [ScanEquipmentChanges].
func ScanFilmEquipmentChanges(
	dir string, bornAt func(slot uint32) (uint64, bool),
) ([]EquipmentChange, EquipmentChangeStats, error) {
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		return nil, EquipmentChangeStats{}, err
	}
	return ScanEquipmentChanges(NewFilmContext(film), bornAt)
}

// ScanEquipmentChanges décode les changements d'équipement porté d'un film DEJA CHARGE.
func ScanEquipmentChanges(
	fc *FilmContext, bornAt func(slot uint32) (uint64, bool),
) ([]EquipmentChange, EquipmentChangeStats, error) {
	var st EquipmentChangeStats
	var out []EquipmentChange
	type state struct {
		rank    int
		counter uint32
		seen    bool
	}
	per := map[uint32]*state{}

	walk, err := walkAbilityEmissions(fc, func(e abilityEmission) {
		s := per[e.Slot]
		if s == nil {
			s = &state{rank: AbilitySetNoRank}
			per[e.Slot] = s
			st.Lives++
			if e.Counter != equipmentFirstCounter {
				st.LivesFirstOffSpec++
			}
		} else {
			countEquipmentCounterStep(&st, s.counter, e.Counter)
		}
		ch := EquipmentChange{
			TimestampUS: e.TimestampUS, Chunk: e.Chunk, PacketIndex: e.PacketIndex,
			Slot: e.Slot, Counter: e.Counter, Rank: e.Rank, Previous: AbilitySetNoRank,
		}
		if s.seen {
			ch.Previous = s.rank
		}
		ch.Kind = classifyEquipmentChange(ch, s.seen, bornAt)
		switch ch.Kind {
		case EquipmentSpawned:
			st.Spawned++
		case EquipmentSpent:
			st.Spent++
		default:
			st.Taken++
		}
		out = append(out, ch)
		s.rank, s.counter, s.seen = e.Rank, e.Counter, true
	})
	if err != nil {
		return nil, st, err
	}
	st.Walk = walk
	return out, st, nil
}

// countEquipmentCounterStep comptabilise l'avance du compteur entre deux émissions d'une même
// vie. Le compteur est sur 3 bits : l'avance se lit MODULO 8, et un pas de 1 dit « aucune
// émission entre les deux ».
func countEquipmentCounterStep(st *EquipmentChangeStats, from, to uint32) {
	step := (int(to) - int(from) + 8) % 8
	switch step {
	case 0:
		st.Repeats++
	case 1:
	default:
		st.CounterJumps++
		st.MissedEstimate += step - 1
	}
}

// classifyEquipmentChange qualifie un changement. La PREMIÈRE émission d'une vie se juge
// contre la NAISSANCE du bipède : contemporaine, c'est une annonce de réapparition ; tardive,
// c'est un ramassage sur la carte.
func classifyEquipmentChange(
	ch EquipmentChange, hadPrevious bool, bornAt func(uint32) (uint64, bool),
) EquipmentChangeKind {
	if ch.Rank == AbilitySetNoRank {
		return EquipmentSpent
	}
	if hadPrevious {
		return EquipmentTaken
	}
	if bornAt != nil {
		if birth, ok := bornAt(ch.Slot); ok && ch.TimestampUS <= birth+equipmentSpawnWindowUS {
			return EquipmentSpawned
		}
	}
	return EquipmentTaken
}
