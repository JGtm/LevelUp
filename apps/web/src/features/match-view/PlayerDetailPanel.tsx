/**
 * PlayerDetailPanel — panneau détail expandable d'un joueur dans le scoreboard.
 * E1 NATIVE_COMPONENTS — implémenté avec React useState (pas de shadcn installé).
 *
 * Pour is_me, affiche armes + médailles + citations si fournies optional.
 * Pour les autres joueurs, affiche les stats complètes du scoreboard.
 */
import type { MatchScoreboardRow, MatchWeaponKill, MatchMedal, MatchCitation } from '@/lib/api/types'
import { tokenCssVar } from '@/lib/accessibility'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'

interface Props {
  row: MatchScoreboardRow
  /** Données armes — disponibles uniquement pour le joueur principal */
  weaponKills?: MatchWeaponKill[]
  /** Médailles — disponibles uniquement pour le joueur principal */
  medals?: MatchMedal[]
  /** Citations — disponibles uniquement pour le joueur principal */
  citations?: MatchCitation[]
}

function StatLine({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between text-xs py-0.5">
      <span className="text-muted-foreground">{label}</span>
      <span className="text-foreground font-mono">{value ?? '—'}</span>
    </div>
  )
}

export function PlayerDetailPanel({ row, weaponKills, medals, citations }: Props) {
  const { data: fieldMappings } = useFieldMappings()
  const labelOf = (key: string): string =>
    fieldMappings?.fields[key]?.label ?? key
  return (
    <div className="bg-card/80 border border-border rounded-b px-4 py-3 space-y-3">
      {/* Stats de combat */}
      <div>
        <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground mb-1">Statistiques</p>
        <div className="grid grid-cols-2 sm:grid-cols-3 gap-x-6">
          <StatLine label={labelOf('kills')} value={row.kills} />
          <StatLine label={labelOf('deaths')} value={row.deaths} />
          <StatLine label={labelOf('assists')} value={row.assists} />
          <StatLine label={labelOf('kda')} value={
            row.kills != null && row.deaths != null
              ? ((row.kills + (row.assists ?? 0)) / Math.max(1, row.deaths)).toFixed(2)
              : null
          } />
          <StatLine label="Damage infligé" value={row.damage_dealt?.toFixed(0)} />
          <StatLine label="Damage reçu" value={row.damage_taken?.toFixed(0)} />
          <StatLine label={labelOf('accuracy')} value={row.shots_accuracy != null ? `${(row.shots_accuracy * 100).toFixed(1)}%` : null} />
          <StatLine label={labelOf('shots_fired')} value={row.shots_fired} />
          <StatLine label={labelOf('shots_hit')} value={row.shots_hit} />
          <StatLine label="Efficacité" value={row.damage_efficiency?.toFixed(2)} />
          <StatLine label="Vie moyenne" value={row.average_life} />
          <StatLine label="Trahisons" value={row.betrayals} />
          <StatLine label="Suicides" value={row.suicides} />
          <StatLine label="Objectifs volés" value={row.objectives_stolen} />
        </div>
      </div>

      {/* Armes — uniquement si disponibles */}
      {weaponKills && weaponKills.length > 0 && (
        <div>
          <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground mb-1">Armes</p>
          <div className="space-y-0.5">
            {weaponKills.slice(0, 5).map((w) => (
              <div key={w.weapon_id} className="flex items-center justify-between text-xs">
                <span className="text-muted-foreground">{w.weapon_label ?? `#${w.weapon_id}`}</span>
                <span className="font-mono font-semibold" style={{ color: tokenCssVar('perf-tier-2') }}>{w.kill_count} kills</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Médailles */}
      {medals && medals.length > 0 && (
        <div>
          <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground mb-1">Médailles</p>
          <div className="flex flex-wrap gap-1">
            {medals.map((m) => (
              <span
                key={m.medal_name_id}
                className="rounded bg-muted/40 px-1.5 py-0.5 text-[11px] text-muted-foreground"
                title={m.description ?? undefined}
              >
                {m.name} ×{m.count}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Citations */}
      {citations && citations.length > 0 && (
        <div>
          <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground mb-1">Citations</p>
          <div className="flex flex-wrap gap-1">
            {citations.map((c) => (
              <span
                key={c.key}
                className="rounded px-1.5 py-0.5 text-[11px] font-semibold text-white"
                style={{ backgroundColor: c.color ?? '#4B5563' }} // color-allow: fallback gris neutre quand l'API ne fournit pas de couleur de citation
              >
                {c.label}{c.value != null && ` · ${c.value}`}
              </span>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
