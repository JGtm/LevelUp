/**
 * ReplaySettingRows — LES LIGNES DU PANNEAU « VUE » : une commande par forme de réglage (interrupteur, choix en
 * segmenté, liste, curseur), choisie par la table des réglages (`SettingDef.control`). Une ligne ne sait rien du
 * réglage qu'elle porte : son texte vient de `settingText`, sa valeur et son geste du modèle du panneau.
 *
 * LA FORME DIT LA NATURE DE LA QUESTION. Un oui/non est un interrupteur (`role="switch"`, annoncé « activé » ou
 * « désactivé ») ; un choix parmi N est un segmenté (`role="radiogroup"`, une option vraie), qui ne promet pas un
 * « tout éteint » ; la liste du filtre du fil est un `<select>` natif, dont la liste ouverte n'est pas bornée par
 * la largeur du champ.
 *
 * UNE FILLE SE LIT SOUS SA MÈRE, en retrait, d'un ton plus discret. Les interrupteurs épinglables portent une
 * punaise, visible au survol, au focus et quand le réglage est épinglé.
 *
 * COULEURS : jetons sémantiques seulement (`primary`, `muted`, `border`, `accent`), le panneau suit le thème.
 */
import { REPLAY_TEXT } from '../i18n/i18n'
import { choiceText, settingText } from './settingsText'
import { optionsOf } from './settingsAvailability'
import type { SettingDef } from './settingsTable'
import type { ReplayViewPanelModel } from './viewPanelModel'
import { PinIcon } from './SettingIcons'

/** L'interrupteur : un rail et sa pastille. `mixed` est l'état d'un maître dont une partie seulement est allumée. */
export function Switch({
  checked, label, onChange, disabled = false, title,
}: {
  checked: boolean | 'mixed'
  label: string
  onChange: () => void
  disabled?: boolean
  title?: string
}) {
  const on = checked === true
  const mixed = checked === 'mixed'
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={label}
      title={title}
      disabled={disabled}
      onClick={onChange}
      className={`flex h-4 w-7 shrink-0 cursor-pointer items-center rounded-full border px-[2px] transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40 disabled:cursor-not-allowed disabled:opacity-50 ${
        on ? 'border-primary bg-primary' : mixed ? 'border-primary bg-primary/45' : 'border-border bg-muted'
      }`}
    >
      {/* LA COURSE EST CALCULÉE : 28 px de rail, 2 px de bordure et 4 px de retrait, soit 22 px utiles ; la pastille
          en prend 12 et en parcourt 10 (ou 5, à mi-course, pour l'état mixte). */}
      <span
        aria-hidden
        className={`h-3 w-3 rounded-full transition-transform duration-150 ${
          on ? 'translate-x-[10px] bg-primary-foreground' : mixed ? 'translate-x-[5px] bg-primary' : 'translate-x-0 bg-muted-foreground'
        }`}
      />
    </button>
  )
}

interface RowProps {
  def: SettingDef
  model: ReplayViewPanelModel
  /** Épingler ou retirer ce réglage ; le panneau dit la suite (cinq places, par exemple). */
  onPin: (def: SettingDef) => void
}

function PinButton({ def, model, onPin }: RowProps) {
  const t = REPLAY_TEXT[model.locale].settings
  const label = settingText(model.locale, def.id).label
  const pinned = model.views.pins.includes(def.id)
  return (
    <button
      type="button"
      aria-pressed={pinned}
      aria-label={pinned ? t.pinRemoveFmt(label) : t.pinAddFmt(label)}
      title={pinned ? t.pinRemoveHint : t.pinAddHint}
      onClick={() => onPin(def)}
      className="flex h-5 w-5 shrink-0 cursor-pointer items-center justify-center rounded text-muted-foreground opacity-0 transition-opacity hover:bg-accent hover:text-accent-foreground focus-visible:opacity-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40 group-hover:opacity-100 aria-pressed:bg-accent aria-pressed:text-foreground aria-pressed:opacity-100"
    >
      <PinIcon />
    </button>
  )
}

/** Un oui/non : libellé, punaise si le réglage s'épingle, interrupteur. Cliquer le libellé bascule aussi. */
export function ToggleRow({ def, model, onPin }: RowProps) {
  const { label, hint } = settingText(model.locale, def.id)
  const on = model.values[def.id] === true
  const child = def.parent !== undefined
  // « Étage de la fiche ouverte » ne se commande que fiche ouverte : sans elle, grisé, la phrase le dit dessous.
  const disabled = def.id === 'floorFollow' && !model.floor.follow.available
  const change = () => model.set(def.id, !on)
  return (
    <div
      data-setting={def.id}
      className={`group flex min-h-7 items-center gap-1.5 rounded pr-1.5 text-xs hover:bg-muted ${child ? 'pl-6' : 'pl-2'} ${disabled ? 'opacity-55' : ''}`}
    >
      <span
        title={hint || undefined}
        onClick={disabled ? undefined : change}
        className={`min-w-0 flex-1 cursor-default truncate ${on && !child ? '' : 'text-muted-foreground'}`}
      >
        {label}
      </span>
      {def.pinnable && <PinButton def={def} model={model} onPin={onPin} />}
      <Switch checked={on} label={label} onChange={change} disabled={disabled} />
    </div>
  )
}

/** Un choix exclusif : le libellé au-dessus, les options en segmenté sur toute la largeur. */
export function ChoiceRow({ def, model }: RowProps) {
  const { label, hint } = settingText(model.locale, def.id)
  const options = optionsOf(def.id, model.ctx)
  const value = String(model.values[def.id])
  const child = def.parent !== undefined
  return (
    <div data-setting={def.id} className={`flex flex-col gap-0.5 px-2 pb-1.5 pt-0.5 text-xs ${child ? 'pl-6' : ''}`}>
      <div className="text-muted-foreground" title={hint || undefined}>{label}</div>
      <div role="radiogroup" aria-label={label} className="flex rounded-md border border-border p-px">
        {options.map((option) => {
          const text = choiceText(model.locale, def.id, option, model.meshPercent)
          const selected = value === option
          return (
            <button
              key={option}
              type="button"
              role="radio"
              aria-checked={selected}
              title={text.hint || undefined}
              onClick={() => model.set(def.id, option)}
              className={`flex min-h-6 flex-1 cursor-pointer items-center justify-center rounded-[5px] px-1.5 py-0.5 text-center text-[11px] font-medium leading-tight transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40 ${
                selected ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
              }`}
            >
              {text.label}
            </button>
          )
        })}
      </div>
    </div>
  )
}

/** Le filtre du fil : une liste, dont chaque entrée dit son nombre de lignes dans tout le match. */
export function SelectRow({ def, model }: RowProps) {
  const t = REPLAY_TEXT[model.locale]
  const { label } = settingText(model.locale, def.id)
  return (
    <div data-setting={def.id} className="flex flex-col gap-0.5 px-2 pb-1.5 pt-0.5 text-xs">
      <label htmlFor="replay-feed-filter-select" className="text-muted-foreground">{label}</label>
      <select
        id="replay-feed-filter-select"
        value={String(model.values.feedFilter)}
        onChange={(e) => {
          model.set('feedFilter', e.target.value)
          // LE FOCUS PART AVEC LE CHOIX : les raccourcis du lecteur se coupent sur un champ de liste (cf. `ReplayViewpointSelect`).
          e.currentTarget.blur()
        }}
        className="h-7 rounded-md border border-input bg-background px-1.5 text-xs focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40"
      >
        {model.feedOptions.map((o) => (
          <option key={o.id} value={o.id}>{`${t.feedFilterLabel[o.id]} (${o.count})`}</option>
        ))}
      </select>
    </div>
  )
}

/** Le volume : un curseur de 0 à 100, à zéro et grisé tant que le son est coupé (le niveau réglé revient avec le son). */
export function SliderRow({ def, model }: RowProps) {
  const t = REPLAY_TEXT[model.locale]
  const { label, hint } = settingText(model.locale, def.id)
  const on = model.values.sound === true
  const level = on ? Math.round(Number(model.values.volume) * 100) : 0
  return (
    <div data-setting={def.id} className="flex items-center gap-2.5 px-2 py-1 text-xs">
      <span className="w-[70px] shrink-0">{label}</span>
      <input
        type="range"
        min={0}
        max={100}
        step={5}
        value={level}
        disabled={!on}
        aria-label={hint}
        title={on ? hint : t.soundVolumeMutedHint}
        onChange={(e) => model.set(def.id, Number(e.currentTarget.value) / 100)}
        className="min-w-0 flex-1 accent-primary"
      />
      <span className="min-w-[4ch] shrink-0 text-right tabular-nums text-muted-foreground">{`${level} %`}</span>
    </div>
  )
}
