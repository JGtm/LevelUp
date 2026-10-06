/**
 * useEmpriseModels — les modèles des cartes de l'onglet Emprise, mémoïsés, depuis la réponse de
 * `/pages/teammates` déjà chargée (bloc `squad_emprise`, historique de matchs, emblèmes des
 * fiches de médailles). Aucune requête. Les cartes reçoivent des modèles stables : `ChartCard`
 * ne rejoue pas son animation à chaque rendu du layout.
 */
import { useCallback, useMemo } from 'react'

import { equipmentFamilyLabel, USAGE_TEXT } from '@/features/_shared/usage/usageI18n'
import type { SquadEmpriseObject, TeammatesPageResponse } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { TEAM_REST_INK, squadPlayerInk } from '../formes/colors'
import { RESOURCE_POWERUP, RESOURCE_VEHICLE, buildControlRows, buildMatchGrid, buildPickupSheets, buildResourceFil, empriseMatchIndex } from './emprise.logic'
import { buildHabitView } from './habit.logic'
import type { PickupIdentity } from './PickupSheetsCard'
import { buildProductionRows, buildYieldRows } from './production.logic'
import { EMPRISE_TEXT } from './empriseStrings'
import { buildVehicleCoverage, vehicleFamilyName } from './vehicles.logic'

export function useEmpriseModels(pageData: TeammatesPageResponse | null, mainPlayerLabel: string, restLabel: string, locale: Locale) {
  const block = pageData?.squad_emprise
  const index = useMemo(() => empriseMatchIndex(pageData?.match_history ?? []), [pageData?.match_history])
  const medalDigest = useMemo(() => pageData?.medal_digest ?? [], [pageData?.medal_digest])
  const usageText = USAGE_TEXT[locale]

  // Un bonus est nommé par le web (famille du résumé d'usage), une arme par le titre, un véhicule
  // par son libellé de titre (famille qualifiée) ou le nom propre tiré de sa clé.
  const unknownVehicle = EMPRISE_TEXT[locale].vehicles.unknown
  const objectName = useCallback(
    (o: SquadEmpriseObject) => {
      if (o.resource === RESOURCE_POWERUP) return equipmentFamilyLabel(o.key, usageText)
      if (o.resource === RESOURCE_VEHICLE) return vehicleFamilyName(o.key, o.label, unknownVehicle)
      return o.label || o.key
    },
    [usageText, unknownVehicle],
  )

  const controlRows = useMemo(() => (block ? buildControlRows(block) : []), [block])
  const fil = useMemo(() => (block ? buildResourceFil(block, index) : null), [block, index])
  const sheets = useMemo(() => (block ? buildPickupSheets(block, objectName) : null), [block, objectName])
  const grid = useMemo(() => (block ? buildMatchGrid(block, index) : null), [block, index])
  const production = useMemo(() => (block ? buildProductionRows(block) : []), [block])
  const yieldRows = useMemo(() => (block ? buildYieldRows(block) : []), [block])
  const vehicleCoverage = useMemo(() => (block ? buildVehicleCoverage(block) : null), [block])
  const habit = useMemo(() => (block ? buildHabitView(block) : ({ kind: 'none' } as const)), [block])
  const placement = block?.placement ?? null

  const identities = useMemo<PickupIdentity[]>(() => {
    const emblems = new Map(medalDigest.map((e) => [e.player.toLowerCase(), e.emblem_url]))
    const squad = (sheets?.owners ?? [])
      .filter((o) => o.xuid != null)
      .map((o, i) => {
        const label = o.gamertag || (i === 0 ? mainPlayerLabel : '') || (o.xuid ?? '')
        return { label, color: squadPlayerInk(i), emblemUrl: emblems.get(label.toLowerCase()) ?? undefined }
      })
    return [...squad, { label: restLabel, color: TEAM_REST_INK, initial: '+' }]
  }, [sheets, medalDigest, mainPlayerLabel, restLabel])

  const playerName = useCallback(
    (xuid: string) => block?.players?.find((p) => p.xuid === xuid)?.gamertag ?? '',
    [block],
  )

  return { objectName, controlRows, fil, sheets, grid, production, yieldRows, vehicleCoverage, habit, placement, identities, playerName }
}
