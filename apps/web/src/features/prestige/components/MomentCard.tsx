/**
 * MomentCard — carte cérémonielle générée à la validation d'un défi.
 *
 * Référence : Annexe F du plan PLAN_challenges_xp_system.md.
 * Format 16:9, ratio = 1200x675 à l'export. Affichage in-app : version compacte.
 *
 * Animations sobres : pas de flip/particules. Glow bordure + fade-in suffit.
 */
import { useAppShellStore } from '@/stores/appShellStore'
import { intlLocale } from '@/lib/formatters'
import type { Challenge } from '@/lib/prestige'
import { TIER_COLORS, TIER_LABELS_FR } from '@/lib/prestige'
import { useAssetLabel } from '@/lib/i18n/fieldMappings'
import { getPrestigeText } from '../i18n'

interface MomentCardProps {
  challenge: Challenge
  /** Valeur réellement atteinte (ex. 1.62 pour un KDA target 1.50). */
  achievedValue: number
  /** Baseline du joueur au moment de la création du défi. */
  baselineValue?: number
  /** Nombre de matchs joués pour valider. */
  matchCount?: number
  /** Affichage compact (vignette galerie) vs plein (modal/export). */
  compact?: boolean
}

export function MomentCard({
  challenge,
  achievedValue,
  baselineValue = 0,
  matchCount = 0,
  compact = false,
}: MomentCardProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getPrestigeText(locale)
  const tier = challenge.tier ?? 'normal'
  const color = TIER_COLORS[tier]
  const isMythic = tier === 'mythic'
  // Phase 4 plan finition multi-titres : libellé du tier via TOML, fallback dict.
  const tierLabelFromTOML = useAssetLabel('challenge_tier', tier)
  const tierLabel = tierLabelFromTOML !== tier ? tierLabelFromTOML : TIER_LABELS_FR[tier]

  const delta =
    baselineValue > 0
      ? `+${Math.round(((achievedValue - baselineValue) / baselineValue) * 100)} %`
      : ''

  const label = challenge.label || challenge.metric
  const date = challenge.completed_at
    ? new Date(challenge.completed_at).toLocaleDateString(intlLocale(locale), {
        day: '2-digit',
        month: 'short',
        year: 'numeric',
      })
    : ''

  return (
    <div
      className={[
        'relative aspect-[16/9] w-full overflow-hidden rounded-lg bg-card',
        isMythic ? 'shadow-lg' : '',
      ].join(' ')}
      style={{
        border: `4px solid ${color}`,
        boxShadow: isMythic ? `0 0 24px ${color}40` : undefined,
      }}
    >
      {/* En-tête : LEVELUP · PRESTIGE (marque produit, non traduit) */}
      <div className="absolute left-3 right-3 top-2 flex items-center justify-between text-2xs uppercase tracking-widest text-muted-foreground">
        {/* eslint-disable-next-line @levelup/no-hardcoded-strings */}
        <span>LevelUp · Prestige</span>
      </div>

      {/* Corps central */}
      <div className="flex h-full flex-col items-center justify-center px-4 text-center">
        <span
          className="rounded-full px-2.5 py-0.5 text-xs font-bold uppercase"
          style={{ backgroundColor: `${color}20`, color }}
        >
          ◆ {tierLabel}
        </span>
        <h2 className={['mt-2 font-semibold', compact ? 'text-base' : 'text-2xl'].join(' ')}>
          {label}
        </h2>

        <div className={['mt-3 flex items-baseline gap-3', compact ? 'text-sm' : 'text-base'].join(' ')}>
          <div className="flex flex-col">
            <span className="text-2xs uppercase text-muted-foreground">{t.momentAchieved}</span>
            <span className="font-bold" style={{ color }}>
              {achievedValue.toFixed(2)}
            </span>
          </div>
          {baselineValue > 0 && (
            <>
              <div className="flex flex-col">
                <span className="text-2xs uppercase text-muted-foreground">{t.momentBaseline}</span>
                <span className="text-foreground">{baselineValue.toFixed(2)}</span>
              </div>
              <div className="flex flex-col">
                <span className="text-2xs uppercase text-muted-foreground">{t.momentDelta}</span>
                <span className="font-semibold" style={{ color }}>
                  {delta}
                </span>
              </div>
            </>
          )}
        </div>
      </div>

      {/* Footer */}
      <div className="absolute bottom-2 left-3 right-3 flex items-center justify-between text-2xs text-muted-foreground">
        {matchCount > 0 ? <span>{matchCount} {t.momentMatches}</span> : <span />}
        <span>{date}</span>
      </div>
    </div>
  )
}
