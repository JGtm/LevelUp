/**
 * SquadSessionSelector — sélecteur de session escouade avec navigation
 * ← Précédente / Suivante →.
 *
 * Page Escouade = sessions multijoueur avec coéquipiers uniquement. Le
 * concept "Solo" est géré par une autre page dédiée — il a été retiré de
 * cette UI (était présent dans la version SessionScopeSelector précédente).
 *
 * Option "(toutes)" = null.
 */
import { useAppShellStore } from '@/stores/appShellStore'
import type { SessionLabelsList } from '@/lib/api/types'
import { getSquadText } from './i18n'

interface Props {
  sessionLabels: SessionLabelsList
  squadSession: string | null
  onSquadChange: (value: string | null) => void
}

interface SessionSelectProps {
  label: string
  sessions: string[]
  value: string | null
  onChange: (v: string | null) => void
  prevTitle: string
  nextTitle: string
  allLabel: string
}

function SessionSelect({
  label,
  sessions,
  value,
  onChange,
  prevTitle,
  nextTitle,
  allLabel,
}: SessionSelectProps) {
  if (sessions.length === 0) return null

  const idx = value ? sessions.indexOf(value) : -1
  const hasPrev = idx > 0
  const hasNext = idx < sessions.length - 1 && idx !== -1

  const handlePrev = () => {
    if (hasPrev) onChange(sessions[idx - 1])
  }
  const handleNext = () => {
    if (hasNext) onChange(sessions[idx + 1])
    else if (idx === -1 && sessions.length > 0) onChange(sessions[sessions.length - 1])
  }

  return (
    <div className="flex items-center gap-2">
      <span className="text-xs font-medium text-muted-foreground w-16 shrink-0">{label}</span>
      <button
        onClick={handlePrev}
        disabled={!hasPrev}
        className="px-2 py-1 text-xs border rounded disabled:opacity-30 hover:bg-muted"
        title={prevTitle}
      >
        ←
      </button>
      <select
        value={value ?? ''}
        onChange={(e) => onChange(e.target.value === '' ? null : e.target.value)}
        className="text-sm border rounded px-2 py-1 bg-background max-w-[180px]"
      >
        <option value="">{allLabel}</option>
        {sessions.map((s) => (
          <option key={s} value={s}>
            {s}
          </option>
        ))}
      </select>
      <button
        onClick={handleNext}
        disabled={idx === sessions.length - 1}
        className="px-2 py-1 text-xs border rounded disabled:opacity-30 hover:bg-muted"
        title={nextTitle}
      >
        →
      </button>
    </div>
  )
}

export function SquadSessionSelector({
  sessionLabels,
  squadSession,
  onSquadChange,
}: Props) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadText(locale)

  if (sessionLabels.squad.length === 0) return null

  return (
    <div className="flex flex-wrap items-center gap-4 rounded-lg border border-border bg-muted/40 px-4 py-3">
      <span className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
        {t.session.label}
      </span>
      <SessionSelect
        label={t.session.squad}
        sessions={sessionLabels.squad}
        value={squadSession}
        onChange={onSquadChange}
        prevTitle={t.session.prev}
        nextTitle={t.session.next}
        allLabel={t.session.all}
      />
      {squadSession && (
        <button
          onClick={() => onSquadChange(null)}
          className="text-xs text-muted-foreground hover:text-foreground ml-auto"
        >
          {t.session.reset}
        </button>
      )}
    </div>
  )
}
