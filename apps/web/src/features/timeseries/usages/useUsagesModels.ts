/**
 * useUsagesModels — les modèles des cartes de l'onglet « Usages » des Séries temporelles, mémoïsés,
 * depuis la réponse de page déjà chargée (`emprise`, `lives_near_teammate`, `formes_retenues`,
 * `match_rows`, `player_emblem_url`). Aucune requête. Rend aussi LES BLOCS à monter (`show`, le
 * prédicat unique de `usages.logic.ts`) et les noms dont les cartes ont besoin.
 */
import { useCallback, useMemo } from 'react'

import { equipmentFamilyLabel, USAGE_TEXT } from '@/features/_shared/usage/usageI18n'
import { RESOURCE_POWERUP, RESOURCE_VEHICLE } from '@/features/squad/emprise/emprise.logic'
import { vehicleFamilyName } from '@/features/squad/emprise/vehicles.logic'
import { FORMES_CARDS_TEXT } from '@/features/squad/formes/cardsI18n'
import { FORMES_TEXT } from '@/features/squad/formes/i18n'
import { buildObjectiveBalance, buildSoloObjectiveSheet } from '@/features/squad/objectif/objectif.logic'
import type { SquadEmpriseObject, TimeseriesPageResponse } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import type { FilAxe } from '@/features/squad/emprise/empriseCharts'
import { buildUsagesModels, usagesSections } from './usages.logic'
import { EMPRISE_TEXT_SOLO, USAGES_TEXT } from './usagesText'

export function useUsagesModels(data: TimeseriesPageResponse, locale: Locale, hasWeaponRange: boolean) {
  const usageText = USAGE_TEXT[locale]
  const ut = USAGES_TEXT[locale]
  const unknownVehicle = EMPRISE_TEXT_SOLO[locale].vehicles.unknown

  // Un bonus est nommé par le web (famille du résumé d'usage), une arme par le titre, un véhicule par
  // son libellé de titre ou le nom propre tiré de sa clé — la règle de l'Emprise de l'Escouade.
  const objectName = useCallback(
    (o: SquadEmpriseObject) => {
      if (o.resource === RESOURCE_POWERUP) return equipmentFamilyLabel(o.key, usageText)
      if (o.resource === RESOURCE_VEHICLE) return vehicleFamilyName(o.key, o.label, unknownVehicle)
      return o.label || o.key
    },
    [usageText, unknownVehicle],
  )

  const models = useMemo(() => buildUsagesModels(data, objectName), [data, objectName])
  const show = usagesSections(data, hasWeaponRange, objectName, models)

  const playerName = useCallback(
    (xuid: string) => models.block?.players?.find((p) => p.xuid === xuid)?.gamertag ?? '',
    [models.block],
  )
  const equipmentLabel = useCallback(
    (family: string) => ut.cards.equipment.unmeasuredNames[family] ?? equipmentFamilyLabel(family, usageText),
    [ut, usageText],
  )

  const families = FORMES_CARDS_TEXT[locale].families
  const familyLabel = useCallback((f: string) => families[f] ?? f, [families])
  const balance = useMemo(() => (models.objective ? buildObjectiveBalance(models.objective) : []), [models.objective])
  const soloSheet = useMemo(() => (models.objective ? buildSoloObjectiveSheet(models.objective) : null), [models.objective])
  const sheetName = useMemo(() => {
    const main = models.objective?.main_xuid ?? ''
    return models.objective?.squad?.find((p) => p.xuid === main)?.gamertag || playerName(main) || main
  }, [models.objective, playerName])

  const filAxe = useMemo<FilAxe>(
    () => ({ kind: 'period', dateOf: ut.cards.dayFmt, caption: ut.cards.filCaption(models.fil?.matches.length ?? 0, models.coverage.filmed) }),
    [ut, models.fil, models.coverage.filmed],
  )

  return {
    models,
    show,
    objectName,
    playerName,
    equipmentLabel,
    familyLabel,
    columns: FORMES_TEXT[locale].columns,
    balance,
    soloSheet,
    sheetName,
    emblemUrl: data.player_emblem_url || undefined,
    filAxe,
  }
}
