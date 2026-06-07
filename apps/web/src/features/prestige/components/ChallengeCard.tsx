/**
 * ChallengeCard — tuile d'affichage d'un défi (carousel home + liste Objectifs).
 *
 * Affiche : nom du défi, palier (couleur), progression, fenêtre temporelle.
 * Cohérent visuellement avec les tuiles de matchs existantes.
 */
import type { Challenge } from '@/lib/prestige'
import { TIER_COLORS, TIER_LABELS_FR, TIER_LABELS_EN } from '@/lib/prestige'
import { useAssetLabel } from '@/lib/i18n/fieldMappings'
import { useAppShellStore } from '@/stores/appShellStore'

interface ChallengeCardProps {
  challenge: Challenge
  /** Valeur courante mesurée (vient d'une éval ou d'un calcul client). 0 si inconnu. */
  currentValue?: number
  onClick?: () => void
}

export function ChallengeCard({ challenge, currentValue, onClick }: ChallengeCardProps) {
  const locale = useAppShellStore((s) => s.locale)
  const tier = challenge.tier ?? 'normal'
  const tierColor = TIER_COLORS[tier]
  // Phase 4.1 plan finition multi-titres : libellé du tier via le TOML backend.
  // Fallback localisé si le manifest n'a pas le libellé (cas courant hors test) :
  // FR = "Héroïque", EN = "Heroic". Sans le fallback localisé, "Heroic" en français.
  const tierLabelFromTOML = useAssetLabel('challenge_tier', tier)
  const tierLabelFallback = locale === 'en' ? TIER_LABELS_EN[tier] : TIER_LABELS_FR[tier]
  const tierLabel = tierLabelFromTOML !== tier ? tierLabelFromTOML : tierLabelFallback
  // Valeur courante : prop explicite (éval) sinon current_value de la réponse
  // backend (aligné sur ObjectiveRow). Sans ça la barre reste à 0 quand le parent
  // ne passe pas la prop (ex: liste d'objectifs Ascension).
  const effectiveValue = currentValue ?? challenge.current_value ?? 0
  const progress = challenge.target > 0
    ? Math.min(100, Math.round((effectiveValue / challenge.target) * 100))
    : 0
  const isComplete = challenge.status === 'completed'
  const label = challenge.label || challenge.metric

  return (
    <button
      type="button"
      onClick={onClick}
      className="group flex h-full w-64 shrink-0 flex-col gap-2 rounded-lg border bg-card p-3 text-left transition-colors hover:bg-accent/40"
      style={{ borderColor: tierColor, borderLeftWidth: 4 }}
    >
      <div className="flex items-center justify-between">
        <span
          className="text-xs font-semibold uppercase tracking-wider"
          style={{ color: tierColor }}
        >
          {tierLabel}
        </span>
        {challenge.mode === 'pilote' && (
          <span className="rounded-full border border-border px-1.5 py-0.5 text-2xs uppercase text-muted-foreground">
            Pilote
          </span>
        )}
      </div>

      <h3 className="line-clamp-2 text-sm font-medium text-foreground">{label}</h3>

      <div className="mt-auto space-y-1">
        <div className="flex items-baseline justify-between text-xs text-muted-foreground">
          <span>
            {effectiveValue.toFixed(2)} / {challenge.target.toFixed(2)}
          </span>
          <span>{progress}%</span>
        </div>
        <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted">
          <div
            className="h-full transition-all"
            style={{
              width: `${progress}%`,
              backgroundColor: isComplete ? tierColor : tierColor,
              opacity: isComplete ? 1 : 0.7,
            }}
          />
        </div>
      </div>
    </button>
  )
}
