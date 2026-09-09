/**
 * equipmentKeptLogic.ts — LA TROISIÈME ISSUE (« gardé sans l'utiliser ») et la reconnaissance
 * rang -> famille de `equipmentChanges` qui la nourrit.
 *
 * Extrait de `equipmentUsageLogic.ts` le 2026-09-09 (scission obligatoire n°2,
 * PLAN_EQUIPEMENT_GACHIS_2026-09-09.md, étape E5 — 582 L, seuil 500, CLAUDE.md n°5). Même
 * découpe de principe que `equipmentUsageColumns.ts` extrait le 2026-08-25 : d'un côté ce
 * que le pont slot -> joueur -> équipe mesure (le fichier voisin), de l'autre ce que la
 * TROISIÈME ISSUE ajoute par-dessus (E2, décision utilisateur 2026-09-09, amende P1).
 *
 * `kept` se DÉRIVE, il n'est JAMAIS lu d'un canal direct : `max(0, taken - utilisé - lâché)`
 * par famille de `KEPT_FAMILIES`, où `taken` vient de `equipmentChanges` (kind `taken`) via
 * `equipmentChangeFamilyOf` — la reconnaissance par la RACINE du libellé publié
 * (`abilityLabels`), dans les deux langues, sur le même patron que `CHARGE_FAMILY_STEMS`
 * (`abilityChargeLogic.ts`) et `translocatorRanks` (`placementTeleport.ts`) : l'artefact ne
 * publie aucune table rang -> famille, seulement rang -> texte bilingue.
 *
 * Tout est PUR : aucun React, aucune couleur, aucune langue.
 */
import { REPLAY_NO_ABILITY_RANK } from '@/lib/api/types'

import { EQUIP_FAMILY_CAMO, EQUIP_FAMILY_OVERSHIELD } from './equipmentFx'
import { droppedFamilyOf } from './gameChangers'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { EquipmentUsageTally } from './equipmentUsageLogic'

/**
 * LES FAMILLES DU BILAN « SERVI OU GÂCHÉ » (E2, PLAN_EQUIPEMENT_GACHIS_2026-09-09.md) : les
 * six déployables (mêmes clés que `deployed`/`dropped`, tirées de `PLACEMENT_RENDER`) et les
 * deux power-ups (mêmes clés que `episodes` — `camo`/`overshield`, PAS `powerup_*`, qui nomme
 * le MÊME objet côté pose : cf. `droppedFamilyOf`). Le répulseur, le grappin, le propulseur et
 * le translocateur (en tant qu'ACTIVATION, distincte de sa balise `translocator_beacon` posée)
 * n'y figurent PAS : leur activation n'est mesurée par aucun canal ici mobilisé (P4), ou ils
 * portent déjà leur propre colonne (grappin).
 */
export const KEPT_FAMILIES: readonly string[] = [
  'wall',
  'sensor',
  'translocator_beacon',
  'shroud_screen',
  'threat_seeker',
  'repair_field',
  EQUIP_FAMILY_CAMO,
  EQUIP_FAMILY_OVERSHIELD,
]

/** Vrai pour les deux familles dont le côté « utilisé » vient des ÉPISODES, pas des poses. */
export function isEpisodeMeasuredFamily(family: string): boolean {
  return family === EQUIP_FAMILY_CAMO || family === EQUIP_FAMILY_OVERSHIELD
}

/**
 * EQUIPMENT_CHANGE_FAMILY_STEMS — LA RECONNAISSANCE RANG -> FAMILLE DE `equipmentChanges`, sur
 * la RACINE du libellé publié par `abilityLabels`. Une famille absente d'ici — grappin,
 * propulseur, répulseur — reste HORS BILAN (P4) sans qu'on y pense : elle a un libellé (donc
 * n'entre pas dans la réserve « sans famille connue »), simplement aucun stem ne la reconnaît.
 */
const EQUIPMENT_CHANGE_FAMILY_STEMS: Readonly<Record<string, readonly string[]>> = {
  wall: ['mur', 'wall'],
  sensor: ['capteur', 'sensor'],
  translocator_beacon: ['translocat'],
  shroud_screen: ['occultant', 'shroud'],
  threat_seeker: ['traqueur', 'seeker'],
  repair_field: ['réparation', 'repair'],
  [EQUIP_FAMILY_CAMO]: ['camouflage'],
  [EQUIP_FAMILY_OVERSHIELD]: ['surbouclier', 'overshield'],
}

/**
 * equipmentChangeFamilyOf — la famille CANONIQUE (vocabulaire `KEPT_FAMILIES`) que nomme le
 * rang `r` d'un `equipmentChanges`, ou `null` quand :
 *  - le rang est HORS BILAN (label connu mais hors `EQUIPMENT_CHANGE_FAMILY_STEMS` : grappin,
 *    propulseur, répulseur) — silencieux, une exclusion PRODUIT, pas une mesure manquante ;
 *  - le rang n'a PAS DE LABEL DU TOUT (`abilityLabels?.[String(r)]` absent) — c'est la RÉSERVE
 *    « objets pris sans famille connue » — COMPTÉE mais JAMAIS AFFICHÉE (décision utilisateur
 *    2026-09-09 : identification par relevé Theater guidé, hors interface),
 *    distinguée par l'appelant via `labels?.[String(r)] == null`, jamais devinée ici.
 */
export function equipmentChangeFamilyOf(
  labels: ReplayDocumentReady['abilityLabels'],
  rank: number,
): string | null {
  const label = labels?.[String(rank)]
  if (!label) return null
  const text = `${label.fr ?? ''} ${label.en ?? ''}`.toLowerCase()
  for (const [family, stems] of Object.entries(EQUIPMENT_CHANGE_FAMILY_STEMS)) {
    if (stems.some((stem) => text.includes(stem))) return family
  }
  return null
}

/**
 * deriveKeptFromTaken — LA TROISIÈME ISSUE, en une passe sur `equipmentChanges` (kind
 * `taken`) : collecte les prises par famille CANONIQUE et par compteur (via `tallyOfSlotAt`,
 * le propriétaire de la VIE qui occupe le slot à l'instant de la prise), puis dérive
 * `tally.kept[famille] = max(0, taken - utilisé - lâché)` une fois `episodes`/`deployed`/
 * `dropped` déjà posés pour ce compteur (règle validée par E0.4, écart médian 0,00 %).
 *
 * Rend le compte des objets pris dont le rang n'a AUCUN label dans CE film (réserve du
 * MATCH, `EquipmentUsage.unnamedTaken` — jamais un porteur mis en cause pour une non-mesure).
 */
export function deriveKeptFromTaken(
  doc: Pick<ReplayDocumentReady, 'equipmentChanges' | 'abilityLabels'>,
  tallyOfSlotAt: (slot: number, t0: number) => EquipmentUsageTally,
): number {
  // LES PRISES, par compteur ET par famille CANONIQUE — un scratch LOCAL, jamais posé sur
  // `EquipmentUsageTally` : ce n'est pas une grandeur affichée, seulement l'entrée du calcul
  // de `kept` ci-dessous.
  const takenByTally = new Map<EquipmentUsageTally, Record<string, number>>()
  let unnamedTaken = 0
  for (const c of doc.equipmentChanges) {
    if (c.kind !== 'taken') continue
    if (!doc.abilityLabels?.[String(c.r)]) {
      // Aucun label pour ce rang, dans CE film : réserve « sans famille connue » (P13
      // amendée), au niveau du match — jamais un porteur mis en cause pour une non-mesure.
      if (c.r !== REPLAY_NO_ABILITY_RANK) unnamedTaken += 1
      continue
    }
    const family = equipmentChangeFamilyOf(doc.abilityLabels, c.r)
    // Labellisé mais hors bilan (grappin, propulseur, répulseur) : exclusion PRODUIT (P4),
    // pas une famille inconnue — elle ne rejoint donc PAS la réserve ci-dessus.
    if (!family) continue
    const t = tallyOfSlotAt(c.slot, c.t)
    const rec = takenByTally.get(t) ?? {}
    rec[family] = (rec[family] ?? 0) + 1
    takenByTally.set(t, rec)
  }
  // LE GARDÉ SE DÉRIVE APRÈS COUP, une fois `deployed`/`dropped`/`episodes` posés pour CE
  // compteur : `taken` sans usage ni lâcher pour la même famille.
  for (const [t, taken] of takenByTally) {
    for (const family of KEPT_FAMILIES) {
      const takenN = taken[family] ?? 0
      if (takenN === 0) continue
      const usedN = isEpisodeMeasuredFamily(family)
        ? (t.episodes[family]?.count ?? 0)
        : (t.deployed[family] ?? 0)
      const droppedN = t.dropped[droppedFamilyOf(family)] ?? 0
      t.kept[family] = Math.max(0, takenN - usedN - droppedN)
    }
  }
  return unnamedTaken
}
