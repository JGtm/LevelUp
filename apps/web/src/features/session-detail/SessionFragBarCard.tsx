/**
 * SessionFragBarCard — carte A de « Frags et usages » (plan PLAN_SESSIONS_EMPRISE_2026-10-06, §3) :
 * « Répartition des frags » du joueur de la page, UNE barre (V1 : jamais un segment par coéquipier).
 *
 * La carte de l'Escouade (`SquadFragBreakdownCard`) montée pour un seul joueur, à l'encre
 * `squad-player-1` ; textes de la vue (pleine page : comptes et total au bout ; compacte : parts,
 * total en sous-libellé du nom). Les classes sont nommées par le manifeste `frags`.
 */
import { useCallback, useMemo } from 'react'

import { getSquadPlayerColors } from '@/features/squad/colors'
import { SquadFragBreakdownCard } from '@/features/squad/SquadFragBreakdownCard'
import type { SessionCompareEntry } from '@/lib/api/types'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest } from '@/lib/i18n/generated/common'
import { fragsManifest } from '@/lib/i18n/generated/frags'
import type { Locale } from '@/lib/i18n/locale'

import type { SessionCardTexts, SessionCompactCards } from './sessionEmpriseText'

interface Props {
  entry: SessionCompareEntry | null
  /** Le nom du joueur de la page (`sessionPlayerName`). */
  player: string
  locale: Locale
  texts: SessionCardTexts
  /** Vue compacte : ses formateurs ; absent en pleine page. */
  compact?: SessionCompactCards['frag']
}

export function SessionFragBarCard({ entry, player, locale, texts, compact }: Props) {
  const classLabel = useCallback(
    (c: string) => formatMessage(fragsManifest, `frags.class.${c}` as never, locale),
    [locale],
  )
  const byPlayer = useMemo(() => ({ [player]: entry?.frag_distribution?.classes ?? [] }), [entry, player])
  const order = useMemo(() => [player], [player])
  const colors = useMemo(() => getSquadPlayerColors(player, []), [player])
  return (
    <SquadFragBreakdownCard
      fragClassesByPlayer={byPlayer}
      playerOrder={order}
      playerColors={colors}
      classLabel={classLabel}
      emptyTitle={formatMessage(commonManifest, 'common.charts.empty_title', locale)}
      t={texts.squad}
      compact={compact}
    />
  )
}
