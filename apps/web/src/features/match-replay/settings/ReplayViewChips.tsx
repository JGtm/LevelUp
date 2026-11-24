/**
 * ReplayViewChips — EN TÊTE DU PANNEAU « VUE » : les préréglages en puces, Mes vues à côté, et la ligne d'état de la
 * vue active. Un préréglage est un point de départ : après un changement, la puce dit « <Nom> · modifié », avec
 * « Revenir à <Nom> », « Mettre à jour cette vue » (pour une vue enregistrée) et « Enregistrer comme ma vue ».
 *
 * « MODIFIÉ » SE DÉDUIT de la comparaison des réglages à la table de la vue, sur les seuls réglages que le match
 * propose (`useViewsController`) ; rien n'est écrit pour lui.
 */
import { useRef, useState } from 'react'

import { REPLAY_TEXT } from '../i18n/i18n'
import { MyViews } from './ReplayMyViews'
import { ChevronIcon, StarIcon } from './SettingIcons'
import { VIEW_LIMIT } from './savedViews'
import { activeViewLabel } from './viewLabel'
import { PRESET_IDS } from './viewPresets'
import type { ReplayViewPanelModel } from './viewPanelModel'

interface Props {
  model: ReplayViewPanelModel
  onStatus: (message: string) => void
}

const CHIP =
  'inline-flex h-[26px] cursor-pointer items-center gap-1 whitespace-nowrap rounded-full border border-border px-2.5 text-[11.5px] font-medium transition-colors hover:bg-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40 aria-pressed:border-primary aria-pressed:bg-primary aria-pressed:text-primary-foreground'
const BUTTON =
  'inline-flex h-6 cursor-pointer items-center rounded-md border border-border px-2 text-[11.5px] transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40'

export function ViewChips({ model, onStatus }: Props) {
  const t = REPLAY_TEXT[model.locale].settings
  const { views } = model
  const { state } = views
  const [saving, setSaving] = useState(false)
  const [name, setName] = useState('')
  const [managerOpen, setManagerOpen] = useState(false)
  const nameRef = useRef<HTMLInputElement>(null)

  const activeUser = views.activeUser
  const { name: activeName, hint, modified } = activeViewLabel(t, views)

  function chip(id: string, label: string, title: string, star: boolean) {
    const active = state.activeId === id
    return (
      // UNE PUCE ACTIVE ET MODIFIÉE passe en pointillé : la vue n'est plus tout à fait celle qu'on a choisie.
      <button key={id} type="button" aria-pressed={active} title={title} onClick={() => views.apply(id)} className={`${CHIP} ${active && modified ? 'border-dashed' : ''}`}>
        {star && <StarIcon filled />}
        {label}
        {active && modified && <span className="font-normal opacity-85">{`· ${t.modified}`}</span>}
      </button>
    )
  }

  function submit() {
    const result = views.saveAs(name)
    if (result === 'ok') {
      setSaving(false)
      onStatus(t.statusSavedFmt(name.trim()))
      setName('')
    } else {
      nameRef.current?.focus()
    }
  }

  return (
    <section className="space-y-1">
      <h3 className="text-xs font-medium text-muted-foreground">{t.presetsTitle}</h3>
      <div role="group" aria-label={t.presetsTitle} className="flex flex-wrap items-center gap-1.5">
        {PRESET_IDS.map((id) => chip(id, t.presets[id].label, t.presets[id].hint, false))}
        {state.views.length > 0 && <span aria-hidden className="mx-0.5 h-4 w-px bg-border" />}
        {state.views.map((v) => chip(v.id, v.name, t.myViewState, state.defaultId === v.id))}
      </div>
      <div className="flex min-h-6 flex-wrap items-center gap-x-2 gap-y-1 pt-1 text-xs">
        {modified ? (
          <>
            <span className="font-semibold">{`${activeName} · ${t.modified}`}</span>
            <span className="text-muted-foreground">{t.changedFmt(views.changed.length)}</span>
            <button
              type="button"
              className={BUTTON}
              onClick={() => {
                views.revert()
                onStatus(t.statusRevertedFmt(activeName))
              }}
            >
              {t.revertFmt(activeName)}
            </button>
            {activeUser && (
              <button
                type="button"
                className={BUTTON}
                onClick={() => {
                  views.updateActive()
                  onStatus(t.statusUpdatedFmt(activeName))
                }}
              >
                {t.updateView}
              </button>
            )}
            <button type="button" className={`${BUTTON} border-primary bg-primary text-primary-foreground hover:bg-primary/90 hover:text-primary-foreground`} onClick={() => setSaving(true)}>
              {t.saveMine}
            </button>
          </>
        ) : (
          <>
            <span className="text-muted-foreground">{`${hint}. ${t.startingPoint}`}</span>
            <button type="button" className={`${BUTTON} border-transparent text-muted-foreground`} onClick={() => setSaving(true)}>
              {t.saveMine}
            </button>
          </>
        )}
      </div>
      {saving &&
        (state.views.length >= VIEW_LIMIT ? (
          <p className="px-0.5 text-[11px] text-muted-foreground">{t.viewLimit}</p>
        ) : (
          <form
            // UNE COUCHE QU'ÉCHAP FERME AVANT LE PANNEAU, et avant le mode plein écran (cf. useReplayFullscreen).
            data-replay-escape-layer=""
            className="flex items-center gap-1.5"
            onSubmit={(e) => {
              e.preventDefault()
              submit()
            }}
            onKeyDown={(e) => {
              // ÉCHAP FERME LA COUCHE LA PLUS INTÉRIEURE : le formulaire d'abord, le panneau à la frappe suivante.
              if (e.key === 'Escape') {
                e.stopPropagation()
                setSaving(false)
              }
            }}
          >
            <label className="sr-only" htmlFor="replay-view-name">{t.viewNameLabel}</label>
            <input
              id="replay-view-name"
              ref={nameRef}
              autoFocus
              type="text"
              maxLength={24}
              value={name}
              placeholder={t.viewNameLabel}
              onChange={(e) => setName(e.target.value)}
              className="h-[26px] min-w-0 flex-1 rounded-md border border-input bg-background px-2 text-xs focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40"
            />
            <button type="submit" className={`${BUTTON} border-primary bg-primary text-primary-foreground hover:bg-primary/90 hover:text-primary-foreground`}>
              {t.save}
            </button>
            <button type="button" className={`${BUTTON} border-transparent text-muted-foreground`} onClick={() => setSaving(false)}>
              {t.cancel}
            </button>
          </form>
        ))}
      <button
        type="button"
        aria-expanded={managerOpen}
        onClick={() => setManagerOpen((v) => !v)}
        className={`${BUTTON} mt-1 gap-1.5 border-transparent text-muted-foreground`}
      >
        {t.myViewsFmt(state.views.length, VIEW_LIMIT)}
        <ChevronIcon up={managerOpen} />
      </button>
      {managerOpen && <MyViews model={model} onStatus={onStatus} />}
    </section>
  )
}
