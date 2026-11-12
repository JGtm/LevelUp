/**
 * ReplayViewPresets — la section PRÉRÉGLAGES du panneau de la vue : quatre commandes à choix
 * exclusif (Complet, Combat, Déplacements, Objectif), la vue courante pressée.
 *
 * C'EST UN CHOIX EXCLUSIF (`SettingsChoice`, bouton pressé) ET NON UN INTERRUPTEUR : exactement
 * un préréglage est actif, ou aucun — « Personnalisé » n'est pas une cinquième commande, c'est
 * l'état où aucune n'est pressée, dit par une phrase sous les quatre. On ne le choisit pas, on y
 * arrive en basculant un calque à la main.
 */
import { SettingsChoice } from './ReplaySettingsToggle'
import { CUSTOM_VIEW, VIEW_PRESET_IDS, type ViewId, type ViewPresetId } from './viewPresetLogic'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'

export function ViewPresetsSection({
  locale, active, onSelect,
}: {
  locale: ReplayLocale
  active: ViewId
  onSelect: (id: ViewPresetId) => void
}) {
  const t = REPLAY_TEXT[locale]
  return (
    <section className="space-y-1">
      <h3 className="text-xs font-medium text-muted-foreground">{t.viewPresetsTitle}</h3>
      <div className="grid grid-cols-2 gap-1">
        {VIEW_PRESET_IDS.map((id) => (
          <SettingsChoice
            key={id}
            label={t.viewPreset[id]}
            description={t.viewPresetHint[id]}
            hint={t.viewPresetHint[id]}
            pressed={active === id}
            onToggle={() => onSelect(id)}
          />
        ))}
      </div>
      {active === CUSTOM_VIEW && (
        <p className="px-1 text-[11px] text-muted-foreground">
          <span className="font-medium text-foreground">{t.viewPreset[CUSTOM_VIEW]}</span>
          {' · '}
          {t.viewCustomNote}
        </p>
      )}
    </section>
  )
}
