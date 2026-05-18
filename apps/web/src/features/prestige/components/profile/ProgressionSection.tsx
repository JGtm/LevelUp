/**
 * ProgressionSection — Section C du profil (leviers + défis suggérés + CTA campagne).
 *
 * Cf. PLAN_PLAYER_PROFILE_ASCENSION.md §4.4 + §4.5.
 *
 * CTA "Démarrer une campagne sur cet axe" : déclenche onStartCampaign (parent).
 * La modale est rendue par le parent (PlayerProfileCard ou CampaignTracker).
 */
import type {
  ProgressionLeverage,
  SuggestedChallenge,
} from '@/lib/playerProfile'
import type { ProfileManifestKey } from '@/lib/i18n/generated/profile'
import { useProfileI18n } from '../../hooks/useProfileI18n'

interface ProgressionSectionProps {
  leverages?: ProgressionLeverage[]
  suggestions?: SuggestedChallenge[]
  onStartCampaign?: (axis: string, axisKind: 'radar' | 'lusr_component') => void
  onLaunchTemplate?: (templateId: string) => void
}

export function ProgressionSection({
  leverages,
  suggestions,
  onStartCampaign,
  onLaunchTemplate,
}: ProgressionSectionProps) {
  const { t } = useProfileI18n()
  if (!leverages?.length && !suggestions?.length) return null
  return (
    <section className="space-y-3 rounded-lg border border-border bg-card p-4">
      <h2 className="text-sm font-semibold uppercase text-muted-foreground">
        {t('profile.section.progression.title')}
      </h2>

      {leverages && leverages.length > 0 && (
        <LeveragesList leverages={leverages} onStartCampaign={onStartCampaign} />
      )}

      {suggestions && suggestions.length > 0 && (
        <SuggestionsList
          suggestions={suggestions}
          onLaunchTemplate={onLaunchTemplate}
        />
      )}
    </section>
  )
}

interface LeveragesListProps {
  leverages: ProgressionLeverage[]
  onStartCampaign?: (axis: string, axisKind: 'radar' | 'lusr_component') => void
}

function LeveragesList({ leverages, onStartCampaign }: LeveragesListProps) {
  const { t } = useProfileI18n()
  return (
    <div>
      <h3 className="mb-2 text-xs font-semibold uppercase text-muted-foreground">
        {t('profile.leverages.title')}
      </h3>
      <ul className="space-y-2">
        {leverages.map((lv) => {
          const labelKey = `profile.lusr.${lv.component}` as ProfileManifestKey
          const axesLabel =
            lv.narrative_axes.length > 0
              ? lv.narrative_axes
                  .map((axis) => t(`profile.axis.${axis}` as ProfileManifestKey))
                  .join(' · ')
              : t('profile.leverages.axes_empty')
          return (
            <li
              key={lv.component}
              className="flex items-center justify-between rounded border border-border bg-background p-2"
            >
              <div>
                <p className="text-sm font-semibold">{t(labelKey)}</p>
                <p className="text-xs text-muted-foreground">
                  {t('profile.leverages.row_subtitle', {
                    value: lv.leverage_value.toFixed(2),
                    axes: axesLabel,
                  })}
                </p>
              </div>
              {onStartCampaign && (
                <button
                  type="button"
                  onClick={() => onStartCampaign(lv.component, 'lusr_component')}
                  className="rounded-md border border-border px-3 py-1 text-xs font-medium hover:bg-muted"
                >
                  {t('profile.cta.start_campaign')}
                </button>
              )}
            </li>
          )
        })}
      </ul>
      <p className="mt-2 text-xs italic text-muted-foreground">
        {t('profile.r2_disclaimer')}
      </p>
    </div>
  )
}

interface SuggestionsListProps {
  suggestions: SuggestedChallenge[]
  onLaunchTemplate?: (templateId: string) => void
}

function SuggestionsList({ suggestions, onLaunchTemplate }: SuggestionsListProps) {
  const { t } = useProfileI18n()
  return (
    <div>
      <h3 className="mb-2 text-xs font-semibold uppercase text-muted-foreground">
        {t('profile.suggestions.title')}
      </h3>
      <ul className="space-y-1">
        {suggestions.map((s) => {
          const tierKey = `profile.tier.${s.target_tier}` as ProfileManifestKey
          return (
            <li
              key={s.template_id}
              className="flex items-center justify-between rounded border border-border bg-background p-2 text-xs"
            >
              <div>
                <p className="font-medium">{s.template_id}</p>
                <p className="text-muted-foreground">
                  {t('profile.suggestions.row_subtitle', {
                    tier: t(tierKey),
                    arc: s.is_arc_step ? 'true' : 'false',
                  })}
                </p>
              </div>
              {onLaunchTemplate && (
                <button
                  type="button"
                  onClick={() => onLaunchTemplate(s.template_id)}
                  className="rounded-md border border-border px-3 py-1 text-xs font-medium hover:bg-muted"
                >
                  {t('profile.cta.launch_template')}
                </button>
              )}
            </li>
          )
        })}
      </ul>
    </div>
  )
}
