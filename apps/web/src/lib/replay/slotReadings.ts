/**
 * slotReadings.ts — LES LECTURES PAR SLOT À UNE IMAGE : les armes portées (`loadoutAt`), la capacité d'armure
 * (`abilityAt`), et le foyer canonique du report de lecture (`nearestReading`) dont elles dépendent, avec
 * `currentLifeOf`, la vie qui couvre une image. Le joueur et ses vies sont dans `rosterLogic.ts` ; son état à
 * une image dans `playerState.ts`.
 *
 * Tout ce fichier est PUR : aucun React, aucun canvas, donc testable.
 */
import { refineAbilityReading, refineWeaponsReading } from './changeRefine'
import { isAliveAt, trackWindow } from './replayLogic'
import type { ReplayDocumentReady, ReplayTrackReady } from './replayNormalize'

/**
 * LOADOUT_SRC_BIRTH — la provenance d'un relevé lu dans le record de CRÉATION du corps
 * (`loadouts[].src`, schéma 69 ; miroir de `replay.LoadoutSrcBirth` côté Go).
 */
export const LOADOUT_SRC_BIRTH = 'birth'

/**
 * LoadoutReading — les armes portées et l'ÂGE de cette lecture.
 *
 * L'ÂGE EST INDISSOCIABLE DE LA VALEUR. Le loadout ne se lit qu'aux images-clés, une toutes les
 * ~20 s. Entre deux, ce qui est affiché est la dernière lecture connue, et la faire passer pour
 * l'état courant était un défaut réel : sur les 21 899 fiches d'un match, l'âge médian de cette
 * lecture est de 8,4 s, et 7,1 % seulement ont moins d'une seconde. L'estompage n'est donc pas
 * un ornement rare — il sert neuf fois sur dix.
 */
export interface LoadoutReading {
  weapons: string[]
  age: number
  /**
   * PROVENANCE du relevé de base (schéma 69) : `birth` = la DOTATION DE NAISSANCE, lue dans le
   * record de création du corps ; absent = un relevé d'image-clé. L'infobulle de la fiche la dit.
   */
  src?: string
  /** Emplacement de chaque arme de `weapons`, quand le relevé le situe (dotation de naissance). */
  k?: number[]
}

/**
 * loadoutAt rend les armes portées par un SLOT à une image, avec l'âge de la lecture.
 *
 * La recherche porte sur le slot et non sur le joueur : le loadout est écrit pour un biped, et
 * un slot est réattribué à chaque réapparition. C'est aussi ce qui rend le report SÛR — une
 * dotation ne peut pas franchir une mort, puisqu'elle ne survit pas à son porteur.
 *
 * BORNÉE À LA VIE EN COURS DU SLOT (correctif du 2026-09-06) : un slot réattribué n'accueille pas
 * qu'une mort, il accueille une AUTRE vie — parfois un AUTRE joueur. Chercher « la dernière
 * lecture du slot » sans borne de vie reportait donc l'armement de la vie PRÉCÉDENTE.
 * `currentLifeOf` désigne la vie qui couvre `frame` sur ce slot ; sans elle, aucune recherche
 * n'a lieu — `null`, jamais une lecture volée à une autre vie.
 *
 * LA DOTATION DE NAISSANCE EST LE PREMIER RELEVÉ DE LA VIE (schéma 69, lot M3.3). Le document
 * publie, dans `loadouts`, les armes que le record de création du corps transmet (`src:
 * 'birth'`), posées sur la vie qu'il ouvre : c'est le relevé PASSÉ du début de vie, et
 * `nearestReading` le retient comme n'importe quel autre.
 *
 * LA LECTURE « À VENIR » EST RETIRÉE D'UNE VIE DONT LA NAISSANCE EST LUE (décision utilisateur
 * Q18 du 2026-09-23) : avant la première image-clé, elle montrait les armes lues JUSQU'À vingt
 * secondes plus tard — dont une arme ramassée entre-temps. La règle est PAR VIE, parce que la
 * lecture de naissance l'est : sur les films des builds antérieurs à HI_1_12_0, le record de
 * création ne se ferme pas et aucune dotation n'est publiée (mesure du lot M3 : neuf témoins sur
 * vingt-deux) — une règle par DOCUMENT les privait du repli, et la fiche « armes non lues »
 * passait de 4,1 % à 13,9 % du temps de vie sur les dix-neuf témoins du gate de corpus.
 * Une vie SANS dotation garde donc l'ancien repli : la plus proche lecture À VENIR de la même
 * vie, âge NÉGATIF publié tel quel et dit « à venir ».
 */
export function loadoutAt(doc: ReplayDocumentReady, slot: number, frame: number): LoadoutReading | null {
  const life = currentLifeOf(doc, slot, frame)
  if (!life) return null
  const window = trackWindow(life)
  const loadouts = doc.loadouts ?? []
  const read = nearestReading(loadouts, slot, frame, window, !lifeHasBirthLoadout(loadouts, slot, window))
  if (!read) return null
  // LA DATATION FINE (schéma 25) : le relevé donne l'ÉTAT, les changements d'arme datés donnent
  // les TRANSITIONS survenues depuis. Ce qu'ils appliquent et ce qu'ils refusent d'appliquer est
  // écrit dans changeRefine.ts — un artefact qui n'en porte aucun rend la lecture inchangée.
  return refineWeaponsReading(
    {
      weapons: read.value.w,
      age: read.age,
      src: read.value.src,
      // SITUÉ seulement quand chaque arme a son emplacement : un relevé d'image-clé n'en porte
      // aucun, et une rangée à moitié située ne l'est pas.
      k: read.value.k.length > 0 && read.value.k.length === read.value.w.length ? read.value.k : undefined,
    },
    doc.weaponChanges,
    slot,
    frame,
  )
}

/** lifeHasBirthLoadout dit si la vie du slot, dans ces bornes, porte sa DOTATION DE NAISSANCE. */
function lifeHasBirthLoadout(
  loadouts: ReadonlyArray<{ slot: number; t: number; src?: string | null }>,
  slot: number,
  window: { start: number; end: number },
): boolean {
  return loadouts.some(
    (l) => l.slot === slot && l.src === LOADOUT_SRC_BIRTH && l.t >= window.start && l.t <= window.end,
  )
}

/**
 * currentLifeOf — LA VIE DU SLOT QUI COUVRE CETTE IMAGE, ou `undefined`.
 *
 * SŒUR DE `lifeOfSlotAt` (`features/match-replay/model/livesPosition.ts`), PAS UNE QUATRIÈME
 * COPIE : ce module ne peut pas l'importer — `lib/` ne dépend jamais de `features/` (sens
 * unique du dépôt, aucun fichier de production ne l'inverse) — et les appelants d'ici
 * (`loadoutAt`, `abilityAt`, et `inventoryAt`/`grenadeReadingAt` côté feature, qui réutilisent
 * cette même fonction) n'ont qu'un slot et une image, jamais la map par-slot déjà construite :
 * la reconstruire à chaque appel coûterait plus cher qu'un balayage direct de `doc.tracks`,
 * déjà la complexité de `nearestReading` lui-même. Le même balayage direct existe déjà en
 * production (`abilityChargeLogic.ts:133`) et l'audit du 2026-09-06 l'a examiné et écarté
 * comme non fautif — même garde, même verdict.
 *
 * MÊME RÈGLE QUE `lifeOfSlotAt` : « prendre la première venue serait une faute » — un slot
 * sans vie à cet instant ne rend rien, jamais une vie voisine.
 */
export function currentLifeOf(
  doc: ReplayDocumentReady,
  slot: number,
  frame: number,
): ReplayTrackReady | undefined {
  return doc.tracks.find((t) => t.slot === slot && isAliveAt(t, frame))
}

/**
 * nearestReading — LE FOYER CANONIQUE du report de lecture (règle ≤2 copies, CLAUDE.md n°6).
 *
 * Trois calques lisent le film par ÉCHANTILLONS ESPACÉS et doivent afficher, à une image
 * donnée, la dernière lecture connue : les armes portées, l'inventaire, et — depuis le
 * 2026-08-14 — la capacité d'armure. La troisième copie a déclenché la centralisation ; le
 * garde-rail `rosterLogic.guard.test.ts` interdit d'en réécrire une quatrième à la main.
 *
 * LA RÈGLE, IDENTIQUE POUR LES TROIS :
 *   - la recherche porte sur le SLOT, jamais sur le joueur. Un slot est réattribué à chaque
 *     réapparition : c'est ce qui rend le report SÛR, une lecture ne peut pas franchir une
 *     mort puisqu'elle ne survit pas à son porteur ;
 *   - avant la première lecture d'une vie, on rend la plus proche À VENIR du même slot, avec
 *     un âge NÉGATIF publié tel quel. L'affichage l'estompe sur sa valeur absolue et le dit
 *     « à venir » — jamais déguisé en lecture passée.
 *
 * `life` BORNE DÉSORMAIS LA RECHERCHE À LA VIE EN COURS DU SLOT (correctif du 2026-09-06) — NI `best` NI `ahead` NE REGARDAIENT LA
 * FRONTIÈRE DE VIE AVANT CE CORRECTIF : sur un slot recyclé ou un film multi-manche, `best`
 * retenait TOUJOURS la lecture passée la plus proche, quelle que soit sa vie, avant même de
 * considérer `ahead` — le repli « à venir » documenté ci-dessus était donc INERTE dès que le
 * slot portait une vie antérieure, et la fiche affichait l'armement de cette vie précédente,
 * parfois celui d'un AUTRE joueur sur un slot recyclé. `life` vient de `currentLifeOf` ; sans
 * vie couvrante, ou sans lecture dans ses bornes, la fonction rend `null`.
 *
 * `allowAhead` (vrai par défaut) autorise ce repli « à venir » ; `loadoutAt` le refuse à une vie
 * dont la dotation de naissance est lue (Q18, cf. sa note) — les autres calques le gardent.
 *
 * `null` NE VEUT PAS DIRE « IDENTITÉ INCONNUE » — décision produit du 2026-09-06 : une vie est
 * un humain ou un bot, jamais une entité anonyme (le nommage se corrige à la source côté Go,
 * hors de ce lot). `null` dit uniquement « aucune lecture encore observée depuis le début de
 * CETTE vie » — un repli neutre, jamais la lecture volée à une autre vie.
 */
export function nearestReading<T extends { slot: number; t: number }>(
  samples: readonly T[],
  slot: number,
  frame: number,
  life: { start: number; end: number } | undefined,
  allowAhead = true,
): { value: T; age: number } | null {
  if (!life) return null
  let best: { value: T; age: number } | null = null
  let ahead: { value: T; age: number } | null = null
  for (const s of samples) {
    if (s.slot !== slot) continue
    // BORNE DE VIE : une lecture hors de [life.start, life.end] appartient à une AUTRE vie du
    // même slot (recyclé) — jamais candidate, ni comme « best » ni comme « ahead ».
    if (s.t < life.start || s.t > life.end) continue
    const age = frame - s.t
    if (age < 0) {
      // La plus PROCHE à venir : l'âge le moins négatif.
      if (!ahead || age > ahead.age) ahead = { value: s, age }
      continue
    }
    if (!best || age < best.age) best = { value: s, age }
  }
  return best ?? (allowAhead ? ahead : null)
}

/**
 * AbilityReading — le RANG de palette de la capacité portée, et l'âge de cette lecture.
 *
 * DEUX CANAUX, UNE SEULE GRANDEUR (cf. l'artefact, abilities.go) : `i48` transmet le rang
 * complet dans les paquets delta, environ une fois par vie ; le canal d'image-clé est dense
 * mais BORGNE — il ne voit que les rangs 16 à 23. On ne les départage pas : la lecture la
 * plus récente gagne, quel que soit son canal, parce que les deux disent la même chose.
 */
export interface AbilityReading {
  rank: number
  age: number
  src: string
}

/**
 * abilityAt rend le dernier rang de capacité lu pour un SLOT, avec l'âge de la lecture.
 *
 * TROIS SOURCES DEPUIS LE SCHÉMA 25, ET LA PLUS RÉCENTE GAGNE : les deux canaux d'`abilities`
 * (image-clé et paquet delta, qui disent ce que le joueur PORTE) et les CHANGEMENTS
 * d'équipement (qui datent ce qui lui ARRIVE). Le départage — et le cas de la consommation,
 * qui rend `null` parce que le joueur ne porte alors plus rien — vit dans changeRefine.ts.
 *
 * BORNÉE À LA VIE EN COURS DU SLOT (correctif P0-2, 2026-09-06, même défaut et même remède
 * que `loadoutAt`) : sans vie couvrante pour ce slot à cette image, aucune des deux sources ne
 * répond — `null`, jamais la capacité d'une autre vie. `window.start` descend jusqu'à
 * `refineAbilityReading`, qui borne à son tour le canal des changements d'équipement (site
 * frère du même défaut, `changeRefine.ts`).
 */
export function abilityAt(
  doc: ReplayDocumentReady,
  slot: number,
  frame: number,
): AbilityReading | null {
  const life = currentLifeOf(doc, slot, frame)
  if (!life) return null
  const window = trackWindow(life)
  const read = nearestReading(doc.abilities ?? [], slot, frame, window)
  const base = read ? { rank: read.value.r, age: read.age, src: read.value.src } : null
  return refineAbilityReading(base, doc.equipmentChanges, slot, frame, window.start)
}
