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

const COMPONENT_LABELS_FR: Record<string, string> = {
  kills_vs_expected: 'Kills vs attendus',
  deaths_vs_expected: 'Morts vs attendues',
  win_factor: 'Facteur de victoire',
  damage_efficiency: 'Efficacité dégâts',
  accuracy_delta: 'Précision (delta)',
  medal_exploit: 'Exploits / médailles',
  offensive_conversion: 'Conversion offensive',
  defensive_resistance: 'Résistance défensive',
}

const TIER_LABELS_FR: Record<string, string> = {
  normal: 'Normal',
  heroic: 'Héroïque',
  legendary: 'Légendaire',
  mythic: 'Mythique',
}

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
  if (!leverages?.length && !suggestions?.length) return null
  return (
    <section className="space-y-3 rounded-lg border border-border bg-card p-4">
      <h2 className="text-sm font-semibold uppercase text-muted-foreground">
        Pistes de progression
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
  return (
    <div>
      <h3 className="mb-2 text-xs font-semibold uppercase text-muted-foreground">
        Leviers prioritaires
      </h3>
      <ul className="space-y-2">
        {leverages.map((lv) => (
          <li
            key={lv.component}
            className="flex items-center justify-between rounded border border-border bg-background p-2"
          >
            <div>
              <p className="text-sm font-semibold">
                {COMPONENT_LABELS_FR[lv.component] ?? lv.component}
              </p>
              <p className="text-xs text-muted-foreground">
                Levier {lv.leverage_value.toFixed(2)} · axes :{' '}
                {lv.narrative_axes.join(' · ') || '—'}
              </p>
            </div>
            {onStartCampaign && (
              <button
                type="button"
                onClick={() => onStartCampaign(lv.component, 'lusr_component')}
                className="rounded-md border border-border px-3 py-1 text-xs font-medium hover:bg-muted"
              >
                Démarrer une campagne
              </button>
            )}
          </li>
        ))}
      </ul>
      <p className="mt-2 text-xs italic text-muted-foreground">
        On t&apos;aide à voir ta trajectoire — pas à la garantir.
      </p>
    </div>
  )
}

interface SuggestionsListProps {
  suggestions: SuggestedChallenge[]
  onLaunchTemplate?: (templateId: string) => void
}

function SuggestionsList({ suggestions, onLaunchTemplate }: SuggestionsListProps) {
  return (
    <div>
      <h3 className="mb-2 text-xs font-semibold uppercase text-muted-foreground">
        Défis suggérés
      </h3>
      <ul className="space-y-1">
        {suggestions.map((s) => (
          <li
            key={s.template_id}
            className="flex items-center justify-between rounded border border-border bg-background p-2 text-xs"
          >
            <div>
              <p className="font-medium">{s.template_id}</p>
              <p className="text-muted-foreground">
                Palier : {TIER_LABELS_FR[s.target_tier] ?? s.target_tier}
                {s.is_arc_step && ' · étape d’arc'}
              </p>
            </div>
            {onLaunchTemplate && (
              <button
                type="button"
                onClick={() => onLaunchTemplate(s.template_id)}
                className="rounded-md border border-border px-3 py-1 text-xs font-medium hover:bg-muted"
              >
                Lancer
              </button>
            )}
          </li>
        ))}
      </ul>
    </div>
  )
}
