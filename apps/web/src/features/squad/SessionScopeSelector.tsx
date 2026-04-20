/**
 * SessionScopeSelector — deux selects (Solo / Escouade) avec navigation ← Précédente / Suivante →.
 * Changer l'un des deux resets l'autre à null.
 * Option "(toutes)" = null.
 */
import type { SessionLabelsList } from '@/lib/api/types'

interface Props {
  sessionLabels: SessionLabelsList
  soloSession: string | null
  squadSession: string | null
  onSoloChange: (value: string | null) => void
  onSquadChange: (value: string | null) => void
}

function SessionSelect({
  label,
  sessions,
  value,
  onChange,
}: {
  label: string
  sessions: string[]
  value: string | null
  onChange: (v: string | null) => void
}) {
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
      <span className="text-xs font-medium text-muted-foreground w-14 shrink-0">{label}</span>
      <button
        onClick={handlePrev}
        disabled={!hasPrev}
        className="px-2 py-1 text-xs border rounded disabled:opacity-30 hover:bg-muted"
        title="Session précédente"
      >
        ←
      </button>
      <select
        value={value ?? ''}
        onChange={(e) => onChange(e.target.value === '' ? null : e.target.value)}
        className="text-sm border rounded px-2 py-1 bg-background max-w-[180px]"
      >
        <option value="">(toutes)</option>
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
        title="Session suivante"
      >
        →
      </button>
    </div>
  )
}

export function SessionScopeSelector({
  sessionLabels,
  soloSession,
  squadSession,
  onSoloChange,
  onSquadChange,
}: Props) {
  const hasAny = sessionLabels.solo.length > 0 || sessionLabels.squad.length > 0
  if (!hasAny) return null

  const handleSoloChange = (v: string | null) => {
    onSoloChange(v)
    if (v !== null) onSquadChange(null)
  }
  const handleSquadChange = (v: string | null) => {
    onSquadChange(v)
    if (v !== null) onSoloChange(null)
  }

  return (
    <div className="flex flex-wrap items-center gap-4 rounded-lg border border-border bg-muted/40 px-4 py-3">
      <span className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
        Session
      </span>
      <SessionSelect
        label="Solo"
        sessions={sessionLabels.solo}
        value={soloSession}
        onChange={handleSoloChange}
      />
      <SessionSelect
        label="Escouade"
        sessions={sessionLabels.squad}
        value={squadSession}
        onChange={handleSquadChange}
      />
      {(soloSession || squadSession) && (
        <button
          onClick={() => {
            onSoloChange(null)
            onSquadChange(null)
          }}
          className="text-xs text-muted-foreground hover:text-foreground ml-auto"
        >
          ✕ Réinitialiser
        </button>
      )}
    </div>
  )
}
