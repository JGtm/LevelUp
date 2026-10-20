/**
 * useEmpriseModels — les modèles des cartes de l'onglet Emprise, mémoïsés, depuis la réponse de
 * `/pages/teammates` déjà chargée (bloc `squad_emprise`, historique de matchs, emblèmes des
 * fiches de médailles). Aucune requête. Les cartes reçoivent des modèles stables : `ChartCard`
 * ne rejoue pas son animation à chaque rendu du layout.
 */
import { useCallback, useMemo } from 'react'

import { USAGE_TEXT } from '@/features/_shared/usage/usageI18n'
import type { SquadEmpriseObject, TeammatesPageResponse } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { useSquadPlayerPalette } from '../useSquadPlayerPalette'
import { buildControlRows, buildMatchGrid, buildPickupSheets, buildResourceFil, empriseMatchIndex, squadPickupSheets } from './emprise.logic'
import { buildHabitView } from './habit.logic'
import { empriseObjectName } from './objectName'
import type { PickupIdentity } from './PickupSheetsCard'
import { buildProductionRows, buildYieldRows } from './production.logic'
import { EMPRISE_TEXT } from './empriseStrings'

export function useEmpriseModels(pageData: TeammatesPageResponse | null, mainPlayerLabel: string, locale: Locale) {
  const block = pageData?.squad_emprise
  const index = useMemo(() => empriseMatchIndex(pageData?.match_history ?? []), [pageData?.match_history])
  const medalDigest = useMemo(() => pageData?.medal_digest ?? [], [pageData?.medal_digest])
  const usageText = USAGE_TEXT[locale]

  // Un bonus est nommé par le web (famille du résumé d'usage), une arme par le titre, un véhicule
  // par son libellé de titre (famille qualifiée) ou le nom propre tiré de sa clé.
  const unknownVehicle = EMPRISE_TEXT[locale].vehicles.unknown
  const objectName = useCallback(
    (o: SquadEmpriseObject) => empriseObjectName(o, usageText, unknownVehicle),
    [usageText, unknownVehicle],
  )

  const controlRows = useMemo(() => (block ? buildControlRows(block) : []), [block])
  const fil = useMemo(() => (block ? buildResourceFil(block, index) : null), [block, index])
  // Les fiches des seuls joueurs de l'escouade : le reste du camp (joueurs inconnus) n'en a pas.
  const sheets = useMemo(() => (block ? squadPickupSheets(buildPickupSheets(block, objectName)) : null), [block, objectName])
  const grid = useMemo(() => (block ? buildMatchGrid(block, index) : null), [block, index])
  const production = useMemo(() => (block ? buildProductionRows(block) : []), [block])
  const yieldRows = useMemo(() => (block ? buildYieldRows(block) : []), [block])
  const habit = useMemo(() => (block ? buildHabitView(block) : ({ kind: 'none' } as const)), [block])
  const placement = block?.placement ?? null

  // Couleurs : la palette de la page (ordre de la sélection), jamais l'ordre des fiches.
  const { inkOf } = useSquadPlayerPalette()
  const identities = useMemo<PickupIdentity[]>(() => {
    const emblems = new Map(medalDigest.map((e) => [e.player.toLowerCase(), e.emblem_url]))
    return (sheets?.owners ?? []).map((o, i) => {
      const label = o.gamertag || (i === 0 ? mainPlayerLabel : '') || (o.xuid ?? '')
      return { label, color: inkOf(label), emblemUrl: emblems.get(label.toLowerCase()) ?? undefined }
    })
  }, [sheets, medalDigest, mainPlayerLabel, inkOf])

  const playerName = useCallback(
    (xuid: string) => block?.players?.find((p) => p.xuid === xuid)?.gamertag ?? '',
    [block],
  )

  return { objectName, controlRows, fil, sheets, grid, production, yieldRows, habit, placement, identities, playerName }
}
