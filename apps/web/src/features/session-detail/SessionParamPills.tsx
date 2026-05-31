/**
 * SessionParamPills — pills colorées résumant les PARAMÈTRES d'une session
 * (nb de matchs · catégorie FR · durée). Partagé entre le header L3 (session
 * active) et le header du drawer compare → l'utilisateur compare directement les
 * paramètres et comprend pourquoi telle session est suggérée (la suggestion est
 * pilotée par catégorie + nb de matchs + proximité temporelle).
 *
 * Couleurs via tokens sémantiques uniquement (skill color-tokens) : catégorie
 * colorée par type, nb de matchs + durée en `info`. Pills plates (rounded, pas
 * rounded-full) cohérentes avec l'esthétique data-viz moderne du projet.
 */
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility'
import type { SessionCompareEntry } from '@/lib/api/types'

import { useSessionT } from './_shared'

// Catégorie backend ("Ranked"/"Arena"/"Firefight"/"BTB") → clé i18n FR + token couleur.
const CATEGORY_LABEL_KEY: Record<
  string,
  'session.detail.category_ranked' | 'session.detail.category_arena' | 'session.detail.category_firefight' | 'session.detail.category_btb'
> = {
  Ranked: 'session.detail.category_ranked',
  Arena: 'session.detail.category_arena',
  Firefight: 'session.detail.category_firefight',
  BTB: 'session.detail.category_btb',
}

const CATEGORY_TOKEN: Record<string, SemanticToken> = {
  Ranked: 'chart-series-1',
  Arena: 'chart-series-2',
  Firefight: 'chart-series-3',
  BTB: 'chart-series-4',
}

/** Durée de session (start→end) en libellé court : "35 min" ou "1 h 05". */
function sessionDurationLabel(start: string | null, end: string | null): string | null {
  if (!start || !end) return null
  const ms = new Date(end).getTime() - new Date(start).getTime()
  if (!Number.isFinite(ms) || ms <= 0) return null
  const totalMin = Math.round(ms / 60_000)
  if (totalMin < 60) return `${totalMin} min`
  const h = Math.floor(totalMin / 60)
  const m = totalMin % 60
  return m === 0 ? `${h} h` : `${h} h ${String(m).padStart(2, '0')}`
}

function Pill({ label, token }: { label: string; token: SemanticToken }) {
  const color = tokenCssVar(token)
  return (
    <span
      className="inline-flex items-center whitespace-nowrap rounded border px-2 py-0.5 text-2xs font-semibold"
      style={{
        backgroundColor: `color-mix(in oklab, ${color} 14%, transparent)`,
        borderColor: `color-mix(in oklab, ${color} 45%, transparent)`,
        color,
      }}
    >
      {label}
    </span>
  )
}

export function SessionParamPills({ entry }: { entry: SessionCompareEntry | null }) {
  const t = useSessionT()
  if (!entry) return null

  const cat = entry.dominant_category
  const catKey = cat ? CATEGORY_LABEL_KEY[cat] : undefined
  const duration = sessionDurationLabel(entry.start_time, entry.end_time)

  return (
    // xl:flex-nowrap : en vue côte-à-côte (headers L3 à hauteur fixe) les pills tiennent
    // sur une seule ligne ; en dessous de xl elles wrappent (mobile).
    <div className="flex flex-wrap items-center gap-1.5 xl:flex-nowrap">
      {/* Identité solo/escouade de la session (uniforme sur tous ses matchs). */}
      <Pill
        label={entry.with_friends ? t('session.detail.pill_squad') : t('session.detail.pill_solo')}
        token={entry.with_friends ? 'team-ally' : 'chart-series-6'}
      />
      <Pill label={t('session.detail.pill_matches', { count: String(entry.total_matches) })} token="info" />
      {cat && <Pill label={catKey ? t(catKey) : cat} token={CATEGORY_TOKEN[cat] ?? 'info'} />}
      {duration && <Pill label={duration} token="info" />}
    </div>
  )
}
