/**
 * ExplorerAssistExchangeBars — « Part des assistances » de l'encart cible, rendu
 * 1.A HORIZONTAL (maquette du 2026-09-21, décision utilisateur D17).
 *
 * DEUX pistes épaisses superposées, une par sens (« Il me sert » / « Je le sers »), sur un
 * MÊME axe linéaire partant de zéro dont la borne est le plus gros des deux totaux. Chaque
 * piste est segmentée par tranche de part de dégâts (coup de pouce / travail partagé /
 * frag préparé), compte écrit dans le segment quand il tient ; total et part en bout de
 * barre ; un trait de parité vertical (moitié du total des deux sens) traverse les deux.
 *
 * CE QUI CHANGE par rapport à `_shared/assists/AssistExchangeSummary` (le papillon, que le
 * hub Relations rend toujours) : l'échelle. Le papillon est LOGARITHMIQUE et borné par
 * `assist_volume_max` — le plus gros volume d'un sens parmi TOUTES les relations — parce
 * que Relations compare des paires entre elles. Ici une seule paire est affichée : la
 * longueur redevient la donnée elle-même (63 contre 48 se voit), et `assist_volume_max`
 * n'est plus lu. Le papillon est laissé intact pour son autre appelant
 * (`palmares/RelationAssistsCards`).
 */
import { tokenCssVar } from '@/lib/accessibility'
import { ASSIST_GIVEN_TOKEN, ASSIST_RECEIVED_TOKEN } from '@/features/_shared/assists/AssistTierBar'
import { assistTierTone } from '@/features/_shared/assists/assistTierTone'
import {
  assistLinearSegments,
  assistPairBound,
  assistParityPct,
  givenShare,
  receivedShare,
} from '@/features/_shared/assists/assistExchange'
import { ASSISTS_TEXT, type AssistTier } from '@/features/_shared/assists/assistsI18n'
import { formatPercent } from '@/lib/formatters'
import { formatMessage } from '@/lib/i18n/format'
import { explorerManifest, type ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import type { Locale } from '@/lib/i18n/locale'
import type { AssistTiers, RelationAssists } from '@/lib/api/types'

import { ExplorerStackedTrack, type StackedTrackSegment } from './ExplorerStackedTrack'

const TIER_LABEL_KEY: Record<AssistTier, ExplorerManifestKey> = {
  low: 'explorer.target_profile.assists_tier_low',
  mid: 'explorer.target_profile.assists_tier_mid',
  high: 'explorer.target_profile.assists_tier_high',
}

const TIERS: AssistTier[] = ['low', 'mid', 'high']

export function ExplorerAssistExchangeBars({
  assists,
  locale,
}: {
  assists: RelationAssists
  locale: Locale
}) {
  const t = (key: ExplorerManifestKey) => formatMessage(explorerManifest, key, locale)
  const text = ASSISTS_TEXT[locale]
  const bound = assistPairBound(assists)
  const parity = assistParityPct(assists, bound)
  const fmt = (n: number) => n.toLocaleString(locale)

  const track = (tiers: AssistTiers, token: typeof ASSIST_RECEIVED_TOKEN) =>
    assistLinearSegments(tiers, bound).map<StackedTrackSegment>((s) => ({
      key: s.tier,
      widthPct: s.widthPct,
      color: assistTierTone(tokenCssVar(token), s.tier),
      label: fmt(s.count),
      tooltip: text.segment(fmt(s.count), s.tier),
    }))

  return (
    <div className="flex flex-col gap-2" data-testid="explorer-assist-bars">
      <AssistDirectionRow
        head={t('explorer.target_profile.assists_head_received')}
        tail={`${fmt(assists.received.total)} · ${formatPercent(receivedShare(assists), 0)}`}
        token={ASSIST_RECEIVED_TOKEN}
        segments={track(assists.received, ASSIST_RECEIVED_TOKEN)}
        parityPct={parity}
        parityLabel={t('explorer.target_profile.assists_parity')}
        testId="explorer-assist-track-received"
      />
      <AssistDirectionRow
        head={t('explorer.target_profile.assists_head_given')}
        tail={`${fmt(assists.given.total)} · ${formatPercent(givenShare(assists), 0)}`}
        token={ASSIST_GIVEN_TOKEN}
        segments={track(assists.given, ASSIST_GIVEN_TOKEN)}
        parityPct={parity}
        parityLabel={t('explorer.target_profile.assists_parity')}
        testId="explorer-assist-track-given"
      />
      <ul className="flex flex-col gap-0.5 text-3xs text-muted-foreground">
        {TIERS.map((tier) => (
          <li key={tier} className="flex items-center gap-1.5">
            <span
              className="h-2 w-2 shrink-0 rounded-sm"
              style={{ backgroundColor: assistTierTone(tokenCssVar(ASSIST_RECEIVED_TOKEN), tier) }}
              aria-hidden="true"
            />
            <span>{t(TIER_LABEL_KEY[tier])}</span>
          </li>
        ))}
      </ul>
    </div>
  )
}

/** Une ligne de sens : tête + total·part sur la ligne de titre, piste épaisse dessous. */
function AssistDirectionRow({
  head,
  tail,
  token,
  segments,
  parityPct,
  parityLabel,
  testId,
}: {
  head: string
  tail: string
  token: typeof ASSIST_RECEIVED_TOKEN
  segments: StackedTrackSegment[]
  parityPct: number | null
  parityLabel: string
  testId: string
}) {
  return (
    <div className="flex flex-col gap-0.5">
      <div className="flex items-baseline justify-between text-2xs">
        <span className="text-muted-foreground">{head}</span>
        <span className="font-mono tabular-nums" style={{ color: tokenCssVar(token) }}>
          {tail}
        </span>
      </div>
      <ExplorerStackedTrack
        segments={segments}
        ariaLabel={`${head} — ${tail}`}
        parityPct={parityPct}
        parityLabel={parityLabel}
        testId={testId}
      />
    </div>
  )
}
