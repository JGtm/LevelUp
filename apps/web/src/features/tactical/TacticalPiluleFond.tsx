/**
 * TacticalPiluleFond — la pilule « Fond » (Plan / Rendu) du bandeau du plan. Elle lit ET écrit le style du fond
 * du rejeu 2D (`useMapViewPreferences`, une seule clé) : choisir le plan ici le choisit au rejeu, et inversement.
 * Elle n'est proposée que pour une carte qui porte une image plan : sans elle, le choix ne changerait rien.
 */
import { useMapViewPreferences } from '@/features/match-replay/settings/useMapViewPreferences'
import { resolveBackgroundVariant, type MapBackgroundVariant } from '@/lib/replay/mapBackgroundVariant'

import type { TacticalText } from './i18n'
import { ETIQUETTE, PILULE } from './pilules'
import { useTacticalMapPlanImage } from './queries'

const VALEURS: readonly MapBackgroundVariant[] = ['plan', 'rendu']

export function TacticalPiluleFond({ t, playerSlug, mapId }: { t: TacticalText; playerSlug: string; mapId: string }) {
  const { backgroundStyle, setBackgroundStyle } = useMapViewPreferences(mapId)
  const { planImage } = useTacticalMapPlanImage(playerSlug, mapId)
  if (!planImage) return null
  const posee = resolveBackgroundVariant(backgroundStyle, planImage)
  const libelle: Record<MapBackgroundVariant, string> = { plan: t.backgroundPlan, rendu: t.backgroundRender }
  return (
    <span role="group" aria-label={t.pillBackground} className={PILULE} data-testid="tactical-plan-fond-pilule">
      <span className={ETIQUETTE}>{t.pillBackground}</span>
      {VALEURS.map((valeur, i) => (
        <button
          key={valeur}
          type="button"
          aria-pressed={posee === valeur}
          onClick={() => setBackgroundStyle(valeur)}
          className={`px-[7px] text-foreground aria-pressed:bg-primary aria-pressed:font-medium aria-pressed:text-primary-foreground${i > 0 ? ' border-l border-input' : ''}`}
        >
          {libelle[valeur]}
        </button>
      ))}
    </span>
  )
}
