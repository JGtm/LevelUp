/**
 * ReplayRubricTabs — LES CINQ RUBRIQUES du panneau « Vue » (Joueurs, Carte, Objets, Fil, Son) en onglets verticaux :
 * une rubrique visible à la fois, avec son interrupteur maître à trois états (tout, une partie, rien) et son compte
 * « N sur M ». Tout se déduit de la table des réglages : les lignes d'une rubrique, leur ordre, leurs sous-titres,
 * leurs dépendances, ce que le match propose.
 *
 * ACCESSIBILITÉ DES ONGLETS : `role="tablist"` vertical, un onglet sélectionné qui porte seul le focus de tabulation
 * (les autres s'atteignent aux flèches), Début et Fin, et le panneau d'onglet relié à son onglet. Une rubrique sans
 * aucune ligne pour ce match n'a pas d'onglet (« pas de commande qui ne commande rien ») ; si la rubrique retenue
 * disparaît, la première prend sa place.
 */
import { useRef, type KeyboardEvent } from 'react'

import { EmptyStateNotice } from '@/components/ui/empty-state'

import { REPLAY_TEXT } from '../i18n/i18n'
import { FloorRow } from './ReplayFloorRow'
import { ChoiceRow, SelectRow, SliderRow, Switch, ToggleRow } from './ReplaySettingRows'
import { effectiveRubric, isProposed, masterState, masterWrites, shownOfRubric, visibleRubrics } from './settingsAvailability'
import { followNote } from './settingsText'
import type { RubricId, SettingDef } from './settingsTable'
import type { ReplayViewPanelModel } from './viewPanelModel'

const tabId = (r: RubricId) => `replay-view-tab-${r}`
const PANEL_ID = 'replay-view-tabpanel'

interface Env {
  model: ReplayViewPanelModel
  onPin: (def: SettingDef) => void
}

function Row({ def, model, onPin }: Env & { def: SettingDef }) {
  const rowProps = { def, model, onPin }
  switch (def.control) {
    case 'toggle': {
      const note = def.id === 'floorFollow' ? followNote(REPLAY_TEXT[model.locale].relief, model.floor) : null
      return (
        <>
          <ToggleRow {...rowProps} />
          {note && <p className="pl-6 pr-2 text-[11px] text-muted-foreground">{note}</p>}
        </>
      )
    }
    case 'choice': return <ChoiceRow {...rowProps} />
    case 'select': return <SelectRow {...rowProps} />
    case 'slider': return <SliderRow {...rowProps} />
    case 'floor': return <FloorRow model={model} />
  }
}

/** Les notes d'une rubrique : ce qu'une ligne absente doit dire (la probabilité refusée, le fil qui ne se pose pas ici). */
function RubricNotes({ rubric, model }: { rubric: RubricId; model: ReplayViewPanelModel }) {
  const t = REPLAY_TEXT[model.locale]
  const refusal = model.ctx.film.respawnForecastRefusal ?? null
  if (rubric === 'joueurs' && refusal !== null && !isProposed('forecast', model.ctx)) {
    return <EmptyStateNotice size="compact" title={t.respawnForecast.title} description={refusal} className="mx-2 mt-1" />
  }
  if (rubric === 'fil' && !model.ctx.planFeedAvailable) {
    return (
      <p className="mx-2 mt-1 rounded-md border border-dashed border-placeholder-border px-2 py-1.5 text-[11px] text-muted-foreground">
        {t.settings.feedStandardNote}
      </p>
    )
  }
  return null
}

function RubricContent({ rubric, model, onPin }: Env & { rubric: RubricId }) {
  const t = REPLAY_TEXT[model.locale].settings
  const { ctx, values } = model
  const rows = shownOfRubric(rubric, ctx, values)
  const master = masterState(rubric, ctx, values)
  const text = t.rubrics[rubric]
  return (
    <div role="tabpanel" id={PANEL_ID} aria-labelledby={tabId(rubric)} className="min-w-0">
      <div className="mb-0.5 flex items-center gap-2">
        <h3 className="flex-1 text-[13px] font-semibold">{text.label}</h3>
        {master.total > 0 && (
          <>
            <span className="text-[11px] tabular-nums text-muted-foreground">{t.countFmt(master.on, master.total)}</span>
            <Switch
              checked={master.state === 'all' ? true : master.state === 'some' ? 'mixed' : false}
              label={t.masterAriaFmt(text.label)}
              title={t.masterAriaFmt(text.label)}
              onChange={() => {
                for (const [id, value] of masterWrites(rubric, ctx, master.state !== 'all')) model.set(id, value)
              }}
            />
          </>
        )}
      </div>
      <p className="mb-1.5 text-[11px] text-muted-foreground">{text.hint}</p>
      {rows.map((def, at) => {
        // Le sous-titre s'écrit une fois, avant la première ligne de sa pile.
        const sub = def.sub !== undefined && def.sub !== rows[at - 1]?.sub ? def.sub : null
        return (
          <div key={def.id}>
            {sub && (
              <div className="mb-0.5 mt-2 px-2 text-[11px] font-medium uppercase tracking-[0.06em] text-muted-foreground/70">
                {t.subs[sub]}
              </div>
            )}
            <Row def={def} model={model} onPin={onPin} />
          </div>
        )
      })}
      <RubricNotes rubric={rubric} model={model} />
    </div>
  )
}

export function RubricTabs({ model, tab, onTab, onPin }: Env & { tab: RubricId; onTab: (r: RubricId) => void }) {
  const t = REPLAY_TEXT[model.locale].settings
  const visible = visibleRubrics(model.ctx, model.values)
  const active = effectiveRubric(tab, visible)
  const refs = useRef(new Map<RubricId, HTMLButtonElement>())
  if (active === null) return null

  function onKeyDown(e: KeyboardEvent<HTMLDivElement>) {
    const at = visible.indexOf(active as RubricId)
    const step: Record<string, number | undefined> = { ArrowDown: 1, ArrowRight: 1, ArrowUp: -1, ArrowLeft: -1 }
    let next: number | null = null
    if (e.key === 'Home') next = 0
    else if (e.key === 'End') next = visible.length - 1
    else if (step[e.key] !== undefined) next = (at + (step[e.key] as number) + visible.length) % visible.length
    if (next === null) return
    e.preventDefault()
    const target = visible[next]
    onTab(target)
    refs.current.get(target)?.focus()
  }

  return (
    <div className="grid min-h-[300px] grid-cols-[104px_minmax(0,1fr)] gap-2.5">
      <div role="tablist" aria-orientation="vertical" aria-label={t.rubricsAria} onKeyDown={onKeyDown} className="flex flex-col gap-0.5">
        {visible.map((r) => {
          const m = masterState(r, model.ctx, model.values)
          return (
            <button
              key={r}
              ref={(el) => {
                if (el) refs.current.set(r, el)
                else refs.current.delete(r)
              }}
              type="button"
              role="tab"
              id={tabId(r)}
              aria-selected={r === active}
              aria-controls={PANEL_ID}
              tabIndex={r === active ? 0 : -1}
              onClick={() => onTab(r)}
              className="flex cursor-pointer flex-col items-start rounded-md px-2 py-1.5 text-left text-[12.5px] font-medium text-muted-foreground transition-colors hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40 aria-selected:bg-accent aria-selected:text-accent-foreground"
            >
              {t.rubrics[r].label}
              {m.total > 0 && <small className="text-[10.5px] font-normal tabular-nums">{t.countFmt(m.on, m.total)}</small>}
            </button>
          )
        })}
      </div>
      <RubricContent rubric={active} model={model} onPin={onPin} />
    </div>
  )
}
