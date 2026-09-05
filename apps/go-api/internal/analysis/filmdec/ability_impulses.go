package filmdec

// ability_impulses.go — LES IMPULSIONS DE CAPACITÉ, lues dans les paquets delta.
//
// CE QUE CE BALAYAGE LIT, ET D'OÙ VIENT LA RÈGLE. Le tag externe des composants bipède
// `biped-spartan-ability` (i57, prédit) et `biped-spartan-ability-non-predicted-state`
// (i59) est un `R(2)` : il a QUATRE valeurs, et la production n'en exploitait qu'UNE — le
// corps `tag == 3` d'i59, qui porte le GRAPPIN (grapple_state.go, 2026-08-16). Le lot R8
// du 2026-09-03 a mesuré la deuxième : **le corps `tag == 1` date une IMPULSION du
// PROPULSEUR** (rapport RAPPORT_R8_USAGE_REPULSEUR_PROPULSEUR_2026-09-03.md §8).
//
// CE QUI LE PROUVE, EN TROIS CHIFFRES (R8 §8.8, quatre films de famille A, rang attribué
// DANS LA MÊME VIE) : 0,361 impulsion par vie de propulseur (22 sur 61) contre 0,011 par
// vie de répulseur (1 sur 90 — et le répulseur est PLUS porté), et **0,000 sur 132 vies de
// grappin**, qui a son propre tag. L'oracle physique indépendant (pic de vitesse
// horizontale du porteur) rend 6,2 à 8,8 m/s à ces instants contre 2,9 à 3,6 pour un
// instant tiré au hasard dans la même vie. Vérité terrain utilisateur du 2026-09-03
// (Theater, film `1cd3848a`) : 5 usages relevés, 5 impulsions rendues, écart ≤ 1 s.
//
// CE QUE CE CANAL NE PORTE PAS. Le RÉPULSEUR n'y est pas — négatif MESURÉ, pas supposé
// (R8 §8.7 et rapport R9 : ses trois portes sont fermées). Un appelant ne doit donc jamais
// lire « impulsion » comme « usage d'un équipement quelconque ».
//
// L'IDENTITÉ NE VIENT PAS DU COMPOSANT. Le `sub` (R(2) interne d'i57) a été essayé comme
// discriminant puis RÉFUTÉ par le corpus : sur `1cd3848a`, ses quatre valeurs tombent
// TOUTES majoritairement sur le rang du propulseur (R8 §8.5). L'identité vient du canal
// i48 (`ScanFilmAbilityRanks`), rang lu dans la MÊME VIE et ANTÉRIEUREMENT — et cette
// jointure vit chez l'assembleur (replay/document_ability_impulses.go), qui seul connaît
// les vies. Ce fichier ne rend que QUI et QUAND.
//
// AUCUNE GRAMMAIRE NOUVELLE N'EST PORTÉE ICI : les deux désérialiseurs publient déjà le
// tag depuis le 2026-08-16 (`spartanAbilityHook`, `abilityNonPredictedHook`). Ce balayage
// est le patron exact de `ScanFilmGrappleReads` — même composant, tag 1 au lieu de 3.
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requête.
// L'appelant doit détenir LockProcessDecode (BuildFromFilm le fait) : les hooks installés
// sont des globaux de paquet.

import "levelup/go-api/internal/analysis/filmsource"

// abilityImpulseTag : la valeur du tag externe qui date une impulsion. Le 3 est le grappin
// (grapple_state.go), le 0 et le 2 sont l'état de repos (1 572 et 1 565 lectures sur
// `00ba2e1c`, tous rangs mélangés, pic au niveau du témoin aléatoire — R8 §8.2).
const abilityImpulseTag = 1

// abilityPredictedName / abilityPredictedNameAlt : les deux étiquettes de registre d'i57 —
// les films portent l'une OU l'autre (avec ou sans le suffixe `-component`, même dualité
// que `grappleComponentName` pour i59). L'index d'itérateur est résolu PAR NOM dans le
// registre du film, jamais câblé.
const (
	abilityPredictedName    = "biped-spartan-ability-component"
	abilityPredictedNameAlt = "biped-spartan-ability"
)

// AbilityImpulse est UNE lecture d'impulsion de capacité, localisée dans le film.
type AbilityImpulse struct {
	// Slot est l'identifiant bas du biped porteur — le même que celui des trajectoires,
	// donc UNE VIE et non un joueur (le slot migre aux réapparitions).
	Slot uint32
	// Chunk / PacketIndex localisent la lecture dans le film.
	Chunk, PacketIndex int
	// TimestampUS est l'horodatage du paquet porteur — MÊME horloge que BipedPosition.
	TimestampUS uint64
	// Predicted dit que la lecture vient du composant PRÉDIT i57 plutôt que de son jumeau
	// non prédit i59. LES DEUX SONT CO-TRANSMIS : un même geste apparaît souvent dans les
	// deux, et c'est à l'assembleur de les replier en épisodes plutôt que de compter deux
	// fois. Le témoin est publié pour que la couverture puisse dire lequel a parlé.
	Predicted bool
}

// AbilityImpulseStats compte ce que la marche a rencontré. Sans ces dénominateurs, une
// liste d'impulsions ne se juge pas : « 60 lectures » ne dit rien sans « sur combien de
// records annonçant le composant ».
type AbilityImpulseStats struct {
	// Records est le nombre de records delta biped reconnus.
	Records int
	// WithI57 / WithI59 : records dont le masque annonce le composant prédit / non prédit.
	WithI57, WithI59 int
	// Read / Unread : lectures abouties, et records dont la marche n'a pas atteint la cible
	// (un composant intermédiaire non porté, ou un débordement du payload).
	Read, Unread int
	// Tag1 est le nombre de lectures dont le tag externe vaut abilityImpulseTag — les
	// seules publiées.
	Tag1 int
	// Absent dit qu'AUCUN des deux composants n'est déclaré par l'archétype biped du film.
	// C'est une information, pas une erreur : le film ne transmet alors pas ce canal, et
	// une liste vide sans ce témoin serait indistinguable d'un film sans propulseur.
	Absent bool
	// Scanned dit que LE BALAYAGE A TOURNÉ. Faux = il n'a jamais commencé (une des quatre
	// portes de résolution a refusé : aucun chunk, aucun slot biped aux images-clés, découpage
	// i0 indétectable, registre illisible) — l'appelant reçoit alors une erreur, et tout ce qui
	// suit dans cette structure est un zéro SANS SIGNIFICATION.
	//
	// POURQUOI UN TÉMOIN PLUTÔT QUE L'ERREUR SEULE : l'erreur meurt chez l'appelant immédiat,
	// et le zéro qu'il laisse derrière voyage jusqu'à l'artefact. Sans ce champ, une couverture
	// de zéros affirmerait « le balayage a tourné, le composant est là, personne ne s'en est
	// servi » sur un film où rien n'a jamais été lu — la faute exacte que la doctrine de
	// coverage.go interdit (cf. `attachInventoryCoverage`). Un balayage qui aboutit le pose,
	// `Absent` compris : « aucun composant déclaré » EST un résultat de balayage.
	Scanned bool
}

// ScanFilmAbilityImpulses décode les impulsions de capacité (corps tag==1 d'i57 et d'i59)
// dans les paquets delta du film de dir. Les lectures sortent TRIÉES par instant, puis par
// slot — un ordre total, pour que deux exécutions rendent le même artefact.
//
// UN SEUL DÉCODAGE filmdec À LA FOIS PAR PROCESS : ce balayage installe
// `spartanAbilityHook` et `abilityNonPredictedHook`, qui sont des globaux de paquet.
// L'appelant doit détenir LockProcessDecode (BuildFromFilm le fait). Les hooks sont
// restaurés à la sortie, y compris en cas d'erreur.
//
// ScanFilmAbilityImpulses est l'ENVELOPPE D2, HORS PRODUCTION : elle charge le film, ouvre un
// contexte pour elle seule, puis appelle [ScanAbilityImpulses]. La cuisson, elle, passe le
// contexte qu'elle partage entre tous ses balayages.
func ScanFilmAbilityImpulses(dir string) ([]AbilityImpulse, AbilityImpulseStats, error) {
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		return nil, AbilityImpulseStats{}, err
	}
	return ScanAbilityImpulses(NewFilmContext(film))
}

// ScanAbilityImpulses décode les impulsions de capacité d'un film DEJA CHARGE. Cf.
// [ScanFilmAbilityImpulses] pour la doctrine du balayage.
func ScanAbilityImpulses(fc *FilmContext) ([]AbilityImpulse, AbilityImpulseStats, error) {
	var st AbilityImpulseStats
	s, err := resolveAbilityScan(fc)
	if err != nil {
		return nil, st, err
	}
	sc := &abilityImpulseScanner{st: &st, lay: s.lay, arch: s.arch,
		i57idx: componentIndexOfAny(s.arch, abilityPredictedName, abilityPredictedNameAlt),
		i59idx: componentIndexOfAny(s.arch, grappleComponentName, grappleComponentNameAlt),
	}
	if sc.i57idx < 0 && sc.i59idx < 0 {
		// AUCUNE ERREUR ICI, et c'est délibéré : `ScanFilmGrappleReads` refuse un film sans
		// i59 parce que le grappin EST i59 ; ce canal-ci a deux portes et se contente d'une.
		// Les deux absentes, le film ne transmet pas la capacité — un fait mesuré, que
		// `Absent` publie au lieu de le confondre avec un film sans propulseur.
		st.Absent, st.Scanned = true, true
		return nil, st, nil
	}

	// Le hook est LA grammaire : c'est le déserialiseur lui-même qui publie, on ne relit pas
	// les bits à côté de lui (même règle que ScanFilmGrappleReads et ScanFilmAbilityRanks).
	prev57, prev59 := spartanAbilityHook, abilityNonPredictedHook
	SetSpartanAbilityHook(func(tag, _, _ uint64, _ bool) { sc.tag57, sc.got57 = tag, true })
	SetAbilityNonPredictedHook(func(s AbilityNonPredictedState) { sc.tag59, sc.got59 = uint64(s.Tag), true })
	defer func() {
		SetSpartanAbilityHook(prev57)
		SetAbilityNonPredictedHook(prev59)
	}()

	minRecord := bipedHeaderBits + bipedIndexBits*bipedMinMaskCnt + s.lay.TotalBits()
	for _, c := range s.chunks {
		data, pks, ok := s.fc.ChunkAt(c)
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
				i0, slot, idx, ok := matchBipedHeader(pay, p, total, s.slots, true, s.lay)
				if !ok {
					p++
					continue
				}
				st.Records++
				sc.account(pay, i0, total, idx, slot, c, pk)
				p = i0 + s.lay.TotalBits()
			}
		}
	}
	sortAbilityImpulses(sc.out)
	st.Scanned = true
	return sc.out, st, nil
}

// componentIndexOfAny résout l'index d'itérateur d'un composant par ses étiquettes
// possibles, ou -1. Les films portent l'une OU l'autre (avec ou sans `-component`).
func componentIndexOfAny(arch Archetype, names ...string) int {
	for _, n := range names {
		if ids := arch.indicesOf(n); len(ids) > 0 {
			return ids[0]
		}
	}
	return -1
}

// abilityImpulseScanner porte l'état du balayage : compteurs, capture des deux hooks, et
// sortie.
type abilityImpulseScanner struct {
	st             *AbilityImpulseStats
	out            []AbilityImpulse
	lay            I0Layout
	arch           Archetype
	i57idx, i59idx int
	tag57, tag59   uint64
	got57, got59   bool
}

// account marche UN record et impute ses lectures aux compteurs. LES DEUX COMPOSANTS SE
// LISENT DANS LE MÊME RECORD quand le masque les annonce tous deux : la marche s'arrête au
// PLUS LOINTAIN des deux, et le hook de l'autre a déjà parlé en chemin.
func (sc *abilityImpulseScanner) account(pay []byte, i0, total int, idx []int,
	slot uint32, chunk int, pk FilmPacket) {
	has57 := sc.i57idx >= 0 && maskHas(idx, sc.i57idx)
	has59 := sc.i59idx >= 0 && maskHas(idx, sc.i59idx)
	if !has57 && !has59 {
		return
	}
	if has57 {
		sc.st.WithI57++
	}
	if has59 {
		sc.st.WithI59++
	}
	sc.got57, sc.got59 = false, false
	target := sc.i57idx
	if !has57 || (has59 && sc.i59idx > target) {
		target = sc.i59idx
	}
	walkRecordTo(pay, i0, total, idx, sc.lay, sc.arch, target)
	sc.emit(has57, has59, slot, chunk, pk)
}

// emit impute les DEUX lectures du record — celle du composant prédit et celle de son jumeau —
// puis compte ce que la marche n'a pas atteint.
//
// LES DEUX CONTRIBUENT, ET IL FAUT QUE LES DEUX CONTRIBUENT : i57 et i59 sont co-transmis, et
// n'en publier qu'un ferait tomber `coverage.abilityImpulses.reads` de moitié dans le document
// servi (86 -> 43 sur le film de référence) sans qu'aucun compteur ne le dise. Extrait d'
// `account` pour être testable sans film : le balayage, lui, ne se juge que sur pièces.
func (sc *abilityImpulseScanner) emit(has57, has59 bool, slot uint32, chunk int, pk FilmPacket) {
	sc.publish(has57 && sc.got57, sc.tag57, true, slot, chunk, pk)
	sc.publish(has59 && sc.got59, sc.tag59, false, slot, chunk, pk)
	sc.imputeUnread(has57, has59)
}

// publish compte une lecture aboutie et l'émet si son tag date une impulsion.
func (sc *abilityImpulseScanner) publish(got bool, tag uint64, predicted bool,
	slot uint32, chunk int, pk FilmPacket) {
	if !got {
		return
	}
	sc.st.Read++
	if tag != abilityImpulseTag {
		return
	}
	sc.st.Tag1++
	sc.out = append(sc.out, AbilityImpulse{
		Slot: slot, Chunk: chunk, PacketIndex: pk.Index,
		TimestampUS: pk.TimestampUS, Predicted: predicted,
	})
}

// imputeUnread compte les composants ANNONCÉS par le masque que la marche n'a pas atteints
// — la mesure de ce que ce balayage ne voit pas. Un composant annoncé et non lu n'est pas
// une absence d'impulsion : c'est une lecture perdue, et le dénominateur doit le dire.
func (sc *abilityImpulseScanner) imputeUnread(has57, has59 bool) {
	if has57 && !sc.got57 {
		sc.st.Unread++
	}
	if has59 && !sc.got59 {
		sc.st.Unread++
	}
}

// sortAbilityImpulses ordonne les lectures sur (instant, slot, composant) — un ordre TOTAL.
// Un tri partiel laisserait l'ordre des lectures d'un même paquet dépendre du parcours,
// donc l'artefact dépendre de rien de mesurable.
func sortAbilityImpulses(out []AbilityImpulse) {
	if len(out) < 2 {
		return
	}
	lessImpulse := func(a, b AbilityImpulse) bool {
		if a.TimestampUS != b.TimestampUS {
			return a.TimestampUS < b.TimestampUS
		}
		if a.Slot != b.Slot {
			return a.Slot < b.Slot
		}
		return a.Predicted && !b.Predicted
	}
	// Tri par insertion : la sortie est DÉJÀ presque triée (le film se marche dans l'ordre
	// des chunks et des paquets), et le canal est rare — 0 à 160 lectures par film mesurées
	// sur 22 films (R8 §8.5-8.6). Une dépendance de tri pour cela n'en vaut pas la peine.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && lessImpulse(out[j], out[j-1]); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
}
