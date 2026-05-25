/**
 * BehaviorAlertList — Phase 2 : liste des patterns comportementaux détectés.
 *
 * Affiche sévérité + Evidence + conseil actionnable pour chaque pattern.
 * Masqué si la liste est vide.
 */
import type { BehavioralPattern, PatternSeverity } from './types'
import type { AscensionText } from './i18n'

interface BehaviorAlertListProps {
  patterns: BehavioralPattern[]
  t: AscensionText
}

export function BehaviorAlertList({ patterns, t }: BehaviorAlertListProps) {
  const visible = patterns.filter((p) => p.severity !== 'low' || p.confirmed)
  if (visible.length === 0) return null

  return (
    <div className="space-y-2">
      {visible.map((p, i) => (
        <BehaviorAlert key={`${p.type}-${i}`} pattern={p} t={t} />
      ))}
    </div>
  )
}

function BehaviorAlert({ pattern: p, t }: { pattern: BehavioralPattern; t: AscensionText }) {
  const { borderCls, bgCls, labelCls } = severityStyle(p.severity)
  const typeLabel = t.behaviorType?.[p.type] ?? p.type
  const advice = t.behaviorAdvice?.[p.type]

  return (
    <div className={`rounded-md border p-3 ${borderCls} ${bgCls}`}>
      <div className="mb-1 flex items-center gap-2">
        <SeverityBadge severity={p.severity} t={t} />
        <span className={`text-sm font-semibold ${labelCls}`}>{typeLabel}</span>
        {p.confirmed && (
          <span className="rounded bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground">
            {t.behaviorConfirmed ?? 'Confirmé'}
          </span>
        )}
      </div>
      <p className="text-xs text-muted-foreground">
        <span className="font-medium text-foreground">{p.trigger}</span> — {p.evidence}
      </p>
      {advice && (
        <p className="mt-1 text-xs text-muted-foreground/80 italic">{advice}</p>
      )}
    </div>
  )
}

function SeverityBadge({ severity, t }: { severity: PatternSeverity; t: AscensionText }) {
  const label = t.patternSeverity?.[severity] ?? severity
  const { badgeCls } = severityStyle(severity)
  return (
    <span className={`rounded px-1.5 py-0.5 text-[10px] font-bold uppercase ${badgeCls}`}>
      {label}
    </span>
  )
}

function severityStyle(severity: PatternSeverity) {
  switch (severity) {
    case 'high':
      return {
        borderCls: 'border-red-500/40', // color-allow: severity badge — CLAUDE.md §20
        bgCls: 'bg-red-500/5', // color-allow: severity badge — CLAUDE.md §20
        labelCls: 'text-red-700 dark:text-red-300', // color-allow: severity badge — CLAUDE.md §20
        badgeCls: 'bg-red-500/20 text-red-700 dark:text-red-300', // color-allow: severity badge — CLAUDE.md §20
      }
    case 'medium':
      return {
        borderCls: 'border-amber-500/40', // color-allow: severity badge — CLAUDE.md §20
        bgCls: 'bg-amber-500/5', // color-allow: severity badge — CLAUDE.md §20
        labelCls: 'text-amber-700 dark:text-amber-300', // color-allow: severity badge — CLAUDE.md §20
        badgeCls: 'bg-amber-500/20 text-amber-700 dark:text-amber-300', // color-allow: severity badge — CLAUDE.md §20
      }
    default:
      return {
        borderCls: 'border-border',
        bgCls: 'bg-card',
        labelCls: 'text-foreground',
        badgeCls: 'bg-muted text-muted-foreground',
      }
  }
}
