/**
 * ReplayQuickAccess — L'ACCÈS RAPIDE au bord de la carte, à gauche du bouton « Vue » : jusqu'à cinq interrupteurs que
 * le lecteur a épinglés depuis le panneau (la punaise d'une ligne), allumés ou éteints d'un clic sans ouvrir le
 * panneau. Par défaut : la visée, la traînée, la carte de chaleur et le fil posé sur le plan.
 *
 * C'EST LE MÊME ÉTAT QUE LE PANNEAU : un bouton épinglé et sa ligne bougent ensemble, rien n'est copié. Seuls les
 * interrupteurs s'épinglent (la table des réglages dit lesquels, `SettingDef.pinnable`).
 *
 * UN RÉGLAGE ÉPINGLÉ SANS OBJET reste affiché, grisé, avec la raison en infobulle (« Sans objet sur ce match », ou
 * « Disponible au Cinéma et au Plein cadre » pour le fil posé sur le plan) : la place épinglée ne bouge pas d'un match
 * à l'autre, et rien d'impossible n'est cliquable.
 *
 * COULEURS : jetons sémantiques seulement ; le bouton allumé prend le fond d'accent, comme un bouton de barre pressé.
 */
import { REPLAY_TEXT } from '../i18n/i18n'
import { pinStatus } from './settingsAvailability'
import { SettingIcon } from './SettingIcons'
import { settingText } from './settingsText'
import type { ReplayViewPanelModel } from './viewPanelModel'

export function ReplayQuickAccess({ model }: { model: ReplayViewPanelModel }) {
  const t = REPLAY_TEXT[model.locale].settings
  const pins = model.views.pins
  if (pins.length === 0) return null
  return (
    <div
      role="toolbar"
      aria-label={t.quickAria}
      className="flex h-8 items-center gap-0.5 rounded-md border border-border bg-card/85 p-0.5 backdrop-blur-sm"
    >
      {pins.map((id) => {
        const { label } = settingText(model.locale, id)
        const on = model.values[id] === true
        const status = pinStatus(id, model.ctx)
        const reason = status.reason === 'frame' ? t.quickNoFrame : status.reason === 'match' ? t.quickNoMatch : null
        const title = reason !== null ? t.quickDisabledFmt(label, reason) : on ? t.quickOnFmt(label) : t.quickOffFmt(label)
        return (
          <button
            key={id}
            type="button"
            aria-pressed={on}
            aria-label={label}
            title={title}
            disabled={!status.enabled}
            onClick={() => model.set(id, !on)}
            className="flex h-[26px] w-[26px] cursor-pointer items-center justify-center rounded text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40 disabled:cursor-not-allowed disabled:opacity-35 disabled:hover:bg-transparent aria-pressed:bg-accent aria-pressed:text-accent-foreground"
          >
            <SettingIcon id={id} />
          </button>
        )
      })}
    </div>
  )
}
