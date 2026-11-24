/**
 * ReplayMyViews — LE GESTIONNAIRE DE « MES VUES » : jusqu'à trois vues nommées, renommées en place, supprimables,
 * avec l'étoile de la vue d'ouverture, le rattachement facultatif à un mode de jeu, et la liste « À l'ouverture d'un
 * rejeu ». Les réducteurs sont dans `savedViews`, branchés par `useViewsController` : ce composant ne décide de rien.
 */
import { REPLAY_TEXT } from '../i18n/i18n'
import { MATCH_MODE_KEYS, type MatchModeKey } from './settingsAvailability'
import { StarIcon, TrashIcon } from './SettingIcons'
import { PRESET_IDS } from './viewPresets'
import type { ReplayViewPanelModel } from './viewPanelModel'

interface Props {
  model: ReplayViewPanelModel
  onStatus: (message: string) => void
}

const ICON_BUTTON =
  'flex h-6 w-6 shrink-0 cursor-pointer items-center justify-center rounded text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40'
const SELECT_CLASS =
  'h-[26px] max-w-full rounded-md border border-input bg-background px-1.5 text-xs focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40'

export function MyViews({ model, onStatus }: Props) {
  const t = REPLAY_TEXT[model.locale].settings
  const { views } = model
  const { state } = views
  return (
    <div className="mt-2 flex flex-col gap-2 rounded-lg border border-border p-2 text-xs">
      {state.views.length === 0 && <p className="text-muted-foreground">{t.noViews}</p>}
      {state.views.map((v) => (
        <div key={v.id} className="flex flex-col gap-1 border-b border-border pb-2 last:border-b-0 last:pb-0">
          <div className="flex items-center gap-1.5">
            <input
              key={`${v.id}:${v.name}`}
              type="text"
              maxLength={24}
              defaultValue={v.name}
              aria-label={t.renameAria}
              onBlur={(e) => views.rename(v.id, e.currentTarget.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') e.currentTarget.blur()
              }}
              className="h-6 min-w-0 flex-1 rounded border border-transparent bg-transparent px-1.5 text-[12.5px] font-medium hover:border-input focus-visible:border-ring focus-visible:bg-background focus-visible:outline-none"
            />
            <button
              type="button"
              aria-pressed={state.defaultId === v.id}
              aria-label={t.defaultViewAriaFmt(v.name)}
              title={t.defaultViewHint}
              onClick={() => views.setDefault(state.defaultId === v.id ? null : v.id)}
              className={`${ICON_BUTTON} aria-pressed:text-foreground`}
            >
              <StarIcon filled={state.defaultId === v.id} />
            </button>
            <button
              type="button"
              aria-label={t.deleteAriaFmt(v.name)}
              title={t.deleteHint}
              onClick={() => {
                views.remove(v.id)
                onStatus(t.statusDeletedFmt(v.name))
              }}
              className={ICON_BUTTON}
            >
              <TrashIcon />
            </button>
          </div>
          <div className="flex items-center gap-2 text-[11.5px] text-muted-foreground">
            <label className="flex items-center gap-1.5">
              {t.modeLabel}
              <select
                value={v.mode ?? ''}
                aria-label={t.modeAriaFmt(v.name)}
                onChange={(e) => views.attach(v.id, (e.target.value || null) as MatchModeKey | null)}
                className={SELECT_CLASS}
              >
                <option value="">{t.modeNone}</option>
                {MATCH_MODE_KEYS.map((m) => (
                  <option key={m} value={m}>{t.modes[m]}</option>
                ))}
              </select>
            </label>
            <span>{t.optional}</span>
          </div>
        </div>
      ))}
      <label className="flex flex-wrap items-center gap-1.5 text-[11.5px] text-muted-foreground">
        {t.openingLabel}
        <select
          value={state.defaultId ?? ''}
          aria-label={t.openingAria}
          onChange={(e) => views.setDefault(e.target.value || null)}
          className={SELECT_CLASS}
        >
          <option value="">{t.openingLast}</option>
          {PRESET_IDS.map((id) => (
            <option key={id} value={id}>{t.presets[id].label}</option>
          ))}
          {state.views.map((v) => (
            <option key={v.id} value={v.id}>{v.name}</option>
          ))}
        </select>
      </label>
      <p className="text-[11px] text-muted-foreground">{t.openingNote}</p>
    </div>
  )
}
