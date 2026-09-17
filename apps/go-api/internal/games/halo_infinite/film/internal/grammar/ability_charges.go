package grammar

// ability_charges.go — LES CHARGES D'ÉQUIPEMENT RESTANTES, lues dans les paquets delta.
//
// CE QUE CE BALAYAGE LIT, ET D'OÙ VIENT LA RÈGLE. Le composant bipède i56
// `biped-spartan-ability-energy` transmet un masque R(3) puis 7 bits PAR EMPLACEMENT ARMÉ
// (ability_energy.go, désassemblage relu le 2026-07-26). Le lot R11 du 2026-09-03 a mesuré
// ce que ces 7 bits DISENT : le quartet HAUT est un COMPTEUR DE CHARGES ENTIÈRES, le
// quartet bas la recharge fractionnaire — la double lecture que le consommateur
// `FUN_140F8F300` de l'exe fait lui-même (`v / 127.0f` en continu, `(v >> 4) & 0xF` en
// discret ; rapport RAPPORT_R11_REPULSEUR_CHARGES_2026-09-03.md §1.1 et §2).
//
// CE QUI LE PROUVE, EN TROIS CHIFFRES (R11 §2-3) : sur `1cd3848a`, la série de JGtm vaut
// 4, 3, 2, 1, 0 EXACTEMENT aux cinq usages de propulseur que l'utilisateur a relevés au
// Theater (précision 5/5, rappel 5/5, écart ≤ 1 s) ; 52 baisses sur 54 coïncident avec une
// impulsion i57/i59 (canal de R8) sur trois films ; et 36 accroches de grappin sur 36
// (canal `grappleLines`, totalement indépendant) sont appariées à une baisse, contre 2/36
// pour un témoin décalé de 5 s.
//
// LES TROIS PIÈGES DU CANAL, MESURÉS (R11 §2) — et ce fichier tient le premier :
//   (a) UN BIT DE MASQUE À 0 N'EST PAS UNE LECTURE : le moteur pose 0x7F (plein) en RAM.
//       Le film ne transmet RIEN au ramassage — la première valeur transmise est ce qui
//       reste APRÈS le premier usage. Publier un emplacement non armé fabriquerait des
//       lectures ; ce balayage ne rend QUE les emplacements armés.
//   (b) UNE BAISSE PEUT VALOIR PLUSIEURS USAGES (7→3, 4→2, 2→0 observés) : ce balayage
//       publie les LECTURES, jamais un compte d'usages dérivé.
//   (c) LES TROIS EMPLACEMENTS SONT SPÉCIALISÉS DANS LA MESURE (e0 propulseur, e2 grappin,
//       e1 jamais armé sur 16 films) — mais c'est une OBSERVATION, pas une règle :
//       l'identité vient de la jointure par le rang i48 de la même vie
//       (replay/document_ability_charges.go), jamais de l'emplacement. Il est publié pour
//       le débogage, aucun consommateur ne doit brancher dessus.
//
// CE QUE CE CANAL NE PORTE PAS. Le RÉPULSEUR n'y est pas — négatif MESURÉ, pas supposé
// (R11 §4-5 : 218 vies, 111,7 minutes de port, 0 baisse attribuable ; deux porteurs
// consomment leurs trois charges pendant que le canal compte celles du grappin d'autres
// joueurs du même film). Seuls le grappin et le propulseur arment i56 : ce sont les deux
// capacités dont le client PRÉDIT le mouvement du porteur (R11 §8.2).
//
// AUCUNE GRAMMAIRE NOUVELLE N'EST PORTÉE ICI : le désérialiseur publie déjà le masque et
// les trois valeurs depuis le 2026-08-15 (`observateur.AbilityEnergyHook`). Ce balayage est le patron
// exact de `ScanFilmAbilityImpulses` — même contexte partagé, autre composant.
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requête.
// un global de paquet.

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// abilityEnergyName / abilityEnergyNameAlt : les deux étiquettes de registre d'i56 — les
// films portent l'une OU l'autre (avec ou sans le suffixe `-component`, même dualité que
// `abilityPredictedName` pour i57). L'index d'itérateur est résolu PAR NOM EXACT dans le
// registre du film, jamais câblé — et jamais par préfixe : « biped-spartan-ability » (i57)
// est un préfixe de « biped-spartan-ability-energy », un appariement par préfixe lirait le
// mauvais composant.
const (
	abilityEnergyName    = "biped-spartan-ability-energy-component"
	abilityEnergyNameAlt = "biped-spartan-ability-energy"
)

// ScanFilmAbilityCharges décode les lectures de charge d'équipement (les emplacements
// ARMÉS du composant i56) dans les paquets delta du film de dir. Les lectures sortent
// TRIÉES par instant, puis par slot, puis par emplacement — un ordre total, pour que deux
// exécutions rendent le même artefact.
//
// ScanFilmAbilityCharges est l'ENVELOPPE D2, HORS PRODUCTION : elle charge le film, ouvre un
// contexte pour elle seule, puis appelle [ScanAbilityCharges]. La cuisson, elle, passe le
// contexte qu'elle partage entre tous ses balayages.
func ScanFilmAbilityCharges(dir string) ([]types.AbilityCharge, types.AbilityChargeStats, error) {
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		return nil, types.AbilityChargeStats{}, err
	}
	return ScanAbilityCharges(contexteDeBobine(film))
}

// ScanAbilityCharges décode les lectures de charge d'équipement d'un film DEJA CHARGE. Cf.
// [ScanFilmAbilityCharges] pour la doctrine du balayage.
func ScanAbilityCharges(fc *FilmContext) ([]types.AbilityCharge, types.AbilityChargeStats, error) {
	var st types.AbilityChargeStats
	s, err := resolveAbilityScan(fc)
	if err != nil {
		return nil, st, err
	}
	sc := &abilityChargeScanner{st: &st, gram: s.gram,
		idx: componentIndexOfAny(s.gram.arch, abilityEnergyName, abilityEnergyNameAlt)}
	if sc.idx < 0 {
		// AUCUNE ERREUR ICI, et c'est délibéré : le film ne déclare pas le composant, donc il
		// ne transmet pas ce canal — un fait mesuré, que `Absent` publie au lieu de le
		// confondre avec un film où personne n'use ses charges.
		st.Absent, st.Scanned = true, true
		return nil, st, nil
	}

	// Le hook est LA grammaire : c'est le désérialiseur lui-même qui publie, on ne relit
	// pas les bits à côté de lui (même règle que ScanFilmAbilityImpulses).
	obs := NouvelleObservation()
	obs.AbilityEnergyHook = func(mask uint32, ch [AbilityEnergyCharges]int) {
		sc.mask, sc.ch, sc.got = mask, ch, true
	}
	sc.gram.obs = obs

	walkDeltaBipedRecords(s.fc, s.chunks, s.slots, s.gram.lay, func(r deltaBipedRecord) {
		st.Records++
		sc.account(r.Payload, r.I0, r.Total, r.Mask, r.Slot, r.Chunk, r.Packet)
	})
	sortAbilityCharges(sc.out)
	st.Scanned = true
	return sc.out, st, nil
}

// abilityChargeScanner porte l'état du balayage : compteurs, capture du hook, le layout et
// l'archétype du film (portés ici et non passés à chaque record — même patron que
// abilityImpulseScanner), et la sortie.
type abilityChargeScanner struct {
	st   *types.AbilityChargeStats
	out  []types.AbilityCharge
	gram grammaireRecord
	idx  int
	mask uint32
	ch   [AbilityEnergyCharges]int
	got  bool
}

// account marche UN record et impute sa lecture d'i56 aux compteurs.
func (sc *abilityChargeScanner) account(pay []byte, i0, total int, idx []int,
	slot uint32, chunk int, pk FilmPacket) {
	if !maskHas(idx, sc.idx) {
		return
	}
	sc.st.WithI56++
	sc.got = false
	walkRecordTo(pay, i0, total, idx, sc.gram, sc.idx)
	if !sc.got {
		// Composant annoncé et non atteint : une lecture PERDUE, pas une absence de charge —
		// le dénominateur doit le dire.
		sc.st.Unread++
		return
	}
	sc.st.Read++
	sc.publish(slot, chunk, pk)
}

// publish émet une lecture par emplacement ARMÉ du masque. Un bit à 0 signifie « le moteur
// pose 0x7F » (plein) : ce N'EST PAS une lecture, rien n'est publié pour cet emplacement —
// le piège (a) de R11 §2, tenu ici et nulle part ailleurs.
func (sc *abilityChargeScanner) publish(slot uint32, chunk int, pk FilmPacket) {
	for i := 0; i < AbilityEnergyCharges; i++ {
		if sc.mask&(1<<uint(i)) == 0 {
			continue
		}
		v := sc.ch[i]
		sc.st.Armed++
		sc.out = append(sc.out, types.AbilityCharge{
			Slot: slot, Chunk: chunk, PacketIndex: pk.Index, TimestampUS: pk.TimestampUS,
			Emplacement: i, Charges: (v >> 4) & 0xF, Low: v & 0xF,
		})
	}
}

// sortAbilityCharges ordonne les lectures sur (instant, slot, emplacement) — un ordre
// TOTAL. Un tri partiel laisserait l'ordre des lectures d'un même paquet dépendre du
// parcours, donc l'artefact dépendre de rien de mesurable.
func sortAbilityCharges(out []types.AbilityCharge) {
	if len(out) < 2 {
		return
	}
	lessCharge := func(a, b types.AbilityCharge) bool {
		if a.TimestampUS != b.TimestampUS {
			return a.TimestampUS < b.TimestampUS
		}
		if a.Slot != b.Slot {
			return a.Slot < b.Slot
		}
		return a.Emplacement < b.Emplacement
	}
	// Tri par insertion : la sortie est DÉJÀ presque triée (le film se marche dans l'ordre
	// des chunks et des paquets), et le canal est rare — i56 n'est armé qu'au changement de
	// charge (R11 : dizaines de lectures par film). Même choix que sortAbilityImpulses.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && lessCharge(out[j], out[j-1]); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
}
