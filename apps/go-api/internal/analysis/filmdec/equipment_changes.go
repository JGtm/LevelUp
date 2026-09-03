package filmdec

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

import "sort"

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
	// Recovered dit que cette émission vient de la RÉCUPÉRATION GATÉE (equipment_recovery.go)
	// et non du balayage strict : ses octets existent dans le film sous une forme que la
	// production rejette par construction, et son compteur comble exactement un saut annoncé.
	// La provenance reste dite — un `from` redevenu fiable grâce à elle n'est pas un `from`
	// lu par le chemin nominal.
	Recovered bool
	// Gap est le saut de compteur RÉSIDUEL constaté depuis l'émission précédente de la même
	// vie, APRÈS récupération : 0 = chaîne saine (pas de 1), n > 0 = n émissions manquent
	// encore juste avant celle-ci — son champ Previous n'est alors PAS une identité fiable.
	// La première émission d'une vie porte 0 (pas d'émission précédente) ; l'incomplétude de
	// tête se lit dans EquipmentChangeStats.LivesFirstOffSpec.
	Gap int
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
	// Recovered compte les émissions issues de la récupération gatée (equipment_recovery.go),
	// À PART des lues par le balayage strict. Les compteurs ci-dessus (CounterJumps,
	// MissedEstimate, LivesFirstOffSpec) décrivent la chaîne FINALE, récupération comprise :
	// ce qui reste manquant après elle — le témoin mesure ce qui est publié, pas un état
	// intermédiaire.
	Recovered int
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
// savoir. Le témoin sert AUSSI la récupération de tête de vie (fenêtre
// [naissance, première émission]) : sans lui, elle n'est pas tentée.
//
// DEUX PASSES : le balayage STRICT d'abord (aucune garde affaiblie), puis la RÉCUPÉRATION
// GATÉE des seules fenêtres de saut de compteur (equipment_recovery.go, décision D1). Les
// émissions récupérées sont étiquetées `Recovered` et comptées à part.
//
// UN SEUL DÉCODAGE filmdec À LA FOIS PAR PROCESS (cf. ScanFilmAbilityRanks).
func ScanFilmEquipmentChanges(
	dir string, bornAt func(slot uint32) (uint64, bool),
) ([]EquipmentChange, EquipmentChangeStats, error) {
	setup, err := resolveAbilityScan(dir)
	if err != nil {
		return nil, EquipmentChangeStats{}, err
	}
	var strict []abilityEmission
	walk := walkAbilityEmissionsWith(setup, func(e abilityEmission) {
		strict = append(strict, e)
	})
	bySlot := map[uint32][]abilityEmission{}
	for _, e := range strict {
		bySlot[e.Slot] = append(bySlot[e.Slot], e)
	}
	for _, list := range bySlot {
		sortEmissionsByFilmOrder(list)
	}
	recovered := scanEquipmentRecovery(setup, buildEquipRecoveryWindows(bySlot, bornAt))
	out, st := assembleEquipmentChanges(strict, recovered, bornAt)
	st.Walk = walk
	return out, st, nil
}

// equipEmission est une émission en cours de FUSION : stricte ou récupérée, avec l'offset de
// bit qui départage deux émissions du même paquet. Le balayage strict ne capture pas son
// offset : il vaut alors -1 — à clé (instant, chunk, paquet) égale, la STRICTE passe avant
// la récupérée. C'est un choix documenté, pas une vérité du film : quand ce départage place
// une récupérée à contre-chaîne (candidat du paquet frontière), le verrou final la retire.
type equipEmission struct {
	abilityEmission
	recovered bool
	off       int
	// head : la récupérée vient de la fenêtre de TÊTE de sa vie (cf. equipRecovered.head).
	head bool
}

// assembleEquipmentChanges est LA FUSION — pure, testable sans film : émissions strictes +
// récupérées, ordre total, verrou final, classification, gap et stats sur la chaîne FINALE.
// C'est elle que les tests unitaires exercent sur des entrées synthétiques (revue ronde 1,
// F4 : une fusion non testée hors gating laissait des mutations survivre à la CI).
func assembleEquipmentChanges(
	strict []abilityEmission, recovered []equipRecovered, bornAt func(uint32) (uint64, bool),
) ([]EquipmentChange, EquipmentChangeStats) {
	var st EquipmentChangeStats
	var out []EquipmentChange
	for _, list := range mergeEquipEmissions(strict, recovered, &st) {
		st.Lives++
		if list[0].Counter != equipmentFirstCounter {
			st.LivesFirstOffSpec++
		}
		rank, seen := AbilitySetNoRank, false
		for i, e := range list {
			ch := EquipmentChange{
				TimestampUS: e.TimestampUS, Chunk: e.Chunk, PacketIndex: e.PacketIndex,
				Slot: e.Slot, Counter: e.Counter, Rank: e.Rank, Previous: AbilitySetNoRank,
				Recovered: e.recovered,
			}
			if i > 0 {
				countEquipmentCounterStep(&st, list[i-1].Counter, e.Counter)
				ch.Gap = counterGap(list[i-1].Counter, e.Counter)
			}
			if seen {
				ch.Previous = rank
			}
			ch.Kind = classifyEquipmentChange(ch, seen, bornAt)
			switch ch.Kind {
			case EquipmentSpawned:
				st.Spawned++
			case EquipmentSpent:
				st.Spent++
			default:
				st.Taken++
			}
			out = append(out, ch)
			rank, seen = e.Rank, true
		}
	}
	sort.Slice(out, func(i, j int) bool { return equipmentChangeFilmOrderLess(out[i], out[j]) })
	return out, st
}

// mergeEquipEmissions fusionne les deux sources par vie, dans l'ordre total du film (offset
// de bit compris), puis passe le VERROU FINAL. st.Recovered ne compte que les récupérées qui
// SURVIVENT au verrou : une récupérée retirée n'est pas publiée, elle ne se compte pas.
func mergeEquipEmissions(
	strict []abilityEmission, recovered []equipRecovered, st *EquipmentChangeStats,
) map[uint32][]equipEmission {
	merged := map[uint32][]equipEmission{}
	for _, e := range strict {
		merged[e.Slot] = append(merged[e.Slot], equipEmission{abilityEmission: e, off: -1})
	}
	// hasHead : les vies dont une récupérée vient de la fenêtre de TÊTE. Leur chaîne commence
	// au compteur VIRTUEL equipRecoveryHeadCounter, et c'est cette amorce que le verrou final
	// doit voir (revue ronde 2, « verrou tête partielle »).
	hasHead := map[uint32]bool{}
	for _, r := range recovered {
		merged[r.Slot] = append(merged[r.Slot], equipEmission{
			abilityEmission: r.abilityEmission, recovered: true, off: r.off, head: r.head})
		if r.head {
			hasHead[r.Slot] = true
		}
	}
	for slot, list := range merged {
		sort.Slice(list, func(i, j int) bool { return equipEmissionLess(list[i], list[j]) })
		list = pruneRecoveredViolations(list, hasHead[slot])
		merged[slot] = list
		for _, e := range list {
			if e.recovered {
				st.Recovered++
			}
		}
	}
	return merged
}

// equipEmissionLess est l'ordre TOTAL de la fusion : instant, chunk, paquet, puis offset de
// bit — la stricte (-1) avant toute récupérée du même paquet, deux récupérées par leur
// position dans le flux. Sans ce dernier critère, sort.Slice (non stable) rendait un ordre
// dépendant de l'exécution au paquet frontière (revue ronde 1, F3).
func equipEmissionLess(a, b equipEmission) bool {
	if a.TimestampUS != b.TimestampUS {
		return a.TimestampUS < b.TimestampUS
	}
	if a.Chunk != b.Chunk {
		return a.Chunk < b.Chunk
	}
	if a.PacketIndex != b.PacketIndex {
		return a.PacketIndex < b.PacketIndex
	}
	return a.off < b.off
}

// pruneRecoveredViolations est LE VERROU FINAL (revue ronde 1, F3) : l'invariant « une
// récupération ne crée ni répétition ni nouveau saut » se vérifie sur la CHAÎNE FINALE
// TRIÉE, pas seulement dans acceptEquipRecovery — un départage de paquet frontière peut
// intercaler une récupérée à contre-chaîne. Toute récupérée dont le RETRAIT fait baisser le
// nombre de répétitions ou de sauts est retirée ; les strictes ne sont jamais touchées.
//
// `head` dit que la vie porte une récupérée de TÊTE : la vérification part alors du compteur
// VIRTUEL qui précède la première émission (equipRecoveryHeadCounter), la MÊME amorce que
// buildEquipRecoveryWindows a employée pour prédire les candidats. Sans elle, le verrou
// comparait une chaîne amorcée à une chaîne SANS amorce et retirait toute récupération de
// tête PARTIELLE non contiguë à la première stricte — perte conservatrice, mais perte
// (revue ronde 2, « verrou tête partielle » ; sonde : première stricte c7, fromC=4, seul c5
// retrouvé -> la récupérée disparaissait).
func pruneRecoveredViolations(list []equipEmission, head bool) []equipEmission {
	for again := true; again; {
		again = false
		baseRep, baseJumps := chainViolations(list, head)
		for i, e := range list {
			if !e.recovered {
				continue
			}
			without := append(append([]equipEmission{}, list[:i]...), list[i+1:]...)
			if rep, jumps := chainViolations(without, head); rep < baseRep || jumps < baseJumps {
				list, again = without, true
				break
			}
		}
	}
	return list
}

// chainViolations compte les répétitions et les sauts d'une chaîne triée d'émissions. Avec
// `head`, la chaîne est amorcée par le compteur VIRTUEL de tête de vie : la première émission
// se juge alors contre lui, exactement comme les fenêtres de tête la prédisent.
func chainViolations(list []equipEmission, head bool) (repeats, jumps int) {
	prev, seen := uint32(equipRecoveryHeadCounter), head
	for _, e := range list {
		if seen {
			switch counterStep(prev, e.Counter) {
			case 0:
				repeats++
			case 1:
			default:
				jumps++
			}
		}
		prev, seen = e.Counter, true
	}
	return repeats, jumps
}

// sortEmissionsByFilmOrder trie des émissions dans l'ordre du film (instant, puis
// localisation) — l'ordre dans lequel le balayage strict les produisait déjà.
func sortEmissionsByFilmOrder(list []abilityEmission) {
	sort.Slice(list, func(i, j int) bool { return emissionFilmOrderLess(list[i], list[j]) })
}

// emissionFilmOrderLess ordonne deux émissions par instant, puis par localisation dans le
// film — un ordre TOTAL, pour une sortie déterministe.
func emissionFilmOrderLess(a, b abilityEmission) bool {
	if a.TimestampUS != b.TimestampUS {
		return a.TimestampUS < b.TimestampUS
	}
	if a.Chunk != b.Chunk {
		return a.Chunk < b.Chunk
	}
	return a.PacketIndex < b.PacketIndex
}

// equipmentChangeFilmOrderLess est le même ordre total, sur les changements assemblés.
func equipmentChangeFilmOrderLess(a, b EquipmentChange) bool {
	if a.TimestampUS != b.TimestampUS {
		return a.TimestampUS < b.TimestampUS
	}
	if a.Chunk != b.Chunk {
		return a.Chunk < b.Chunk
	}
	if a.PacketIndex != b.PacketIndex {
		return a.PacketIndex < b.PacketIndex
	}
	return a.Slot < b.Slot
}

// counterStep rend l'avance du compteur R(3) entre deux émissions, MODULO 8 — LE SEUL
// EXEMPLAIRE de cette arithmétique (règle des <= 2 copies, CLAUDE.md n°6 ; revue ronde 1,
// F2 : cinq copies du littéral). Un garde-rail grep (TestCounterStepLitteralUnique) interdit
// le littéral hors de ce fichier — une factorisation sans garde-rail re-diverge.
func counterStep(from, to uint32) int {
	return (int(to) - int(from) + 8) % 8
}

// counterGap rend le nombre d'émissions encore manquantes entre deux émissions consécutives
// d'une même vie : pas − 1 quand le compteur saute, 0 sur un pas de 1 ET sur une répétition
// (une répétition n'est pas un manque — elle contredit le canal et se compte à part).
func counterGap(from, to uint32) int {
	if step := counterStep(from, to); step > 1 {
		return step - 1
	}
	return 0
}

// countEquipmentCounterStep comptabilise l'avance du compteur entre deux émissions d'une même
// vie. Le compteur est sur 3 bits : l'avance se lit MODULO 8, et un pas de 1 dit « aucune
// émission entre les deux ».
func countEquipmentCounterStep(st *EquipmentChangeStats, from, to uint32) {
	switch step := counterStep(from, to); step {
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
