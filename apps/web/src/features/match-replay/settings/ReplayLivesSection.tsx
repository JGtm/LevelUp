/**
 * ReplayLivesSection — LE GROUPE « VIES » du panneau de la vue : trois calques (apparitions,
 * morts, tueurs) sous un en-tête qui SE REPLIE, puis les deux affinages COMMUNS (qui, quand).
 *
 * POURQUOI UN GROUPE QUI SE REPLIE : le panneau porte déjà une vingtaine de commandes, et ces trois
 * calques sont éteints à l'ouverture. Replié, l'en-tête dit seulement combien d'entre eux sont
 * allumés ; déplié, il rend les trois interrupteurs. L'état replié est une préférence du lecteur,
 * retenue dans CE navigateur.
 *
 * PAS DE COMMANDE QUI NE COMMANDE RIEN (règle du panneau) : un calque sans marque à poser n'est pas
 * proposé, le groupe entier disparaît quand aucun ne l'est, et les affinages ne s'affichent qu'avec
 * au moins une TRACE allumée — filtrer ce qu'on ne voit pas ne dit rien. « Joueur suivi » n'est
 * proposé que s'il y a un joueur suivi dans ce match.
 *
 * LA PROBABILITÉ DE RÉAPPARITION est un quatrième interrupteur du groupe, pas une trace : elle compte
 * dans le résumé, mais les affinages « Qui » et « Quand » ne la concernent pas (elle suit Seul et Masquer
 * du panneau de joueur, comme tout ce qui appartient à un joueur).
 */
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { LIVES_WHEN, LIVES_WHO, type LivesWhen, type LivesWho } from '../model/livesMarks'
import { FORECAST_DRAWS, type ForecastDraw } from '../model/respawnForecast.logic'
import { SettingsSegments, SettingsToggle } from './ReplaySettingsToggle'
import { usePersistedFlag } from './usePersistedFlag'

/** Un calque du groupe : si le film porte de quoi le poser, son état et sa bascule. */
export interface ReplayLivesLayerControls {
  available: boolean
  show: boolean
  onToggle: () => void
}

/** L'interrupteur de la probabilité de réapparition et son choix de dessin (zones et points, zones, points). */
export interface ReplayRespawnForecastControls extends ReplayLivesLayerControls {
  draw: ForecastDraw
  onSelectDraw: (draw: ForecastDraw) => void
}

/** Ce que le panneau sait du groupe « Vies ». */
export interface ReplayLivesControls {
  spawns: ReplayLivesLayerControls
  deaths: ReplayLivesLayerControls
  killers: ReplayLivesLayerControls
  /** La probabilité de réapparition : un interrupteur, deux calques (teinte, repères). */
  respawnForecast: ReplayRespawnForecastControls
  who: LivesWho
  onSelectWho: (who: LivesWho) => void
  /** Il y a un joueur suivi dans ce match : l'option « Joueur suivi » a un sens. */
  followedAvailable: boolean
  when: LivesWhen
  onSelectWhen: (when: LivesWhen) => void
}

const LIVES_OPEN_KEY = 'replay-lives-group-open'

/** L'en-tête qui replie : un bouton, son état en `aria-expanded`, et le résumé quand il est replié. */
function GroupHeader({
  locale, open, onToggle, summary,
}: { locale: ReplayLocale; open: boolean; onToggle: () => void; summary: string }) {
  const t = REPLAY_TEXT[locale].lives
  return (
    <button
      type="button"
      aria-expanded={open}
      onClick={onToggle}
      title={open ? t.collapseHint : t.expandHint}
      className="flex w-full items-center justify-between gap-2 rounded px-1 py-0.5 text-left transition-colors hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40"
    >
      <span className="flex items-center gap-1 text-xs font-medium text-muted-foreground" title={t.titleHint}>
        <svg
          width="10"
          height="10"
          viewBox="0 0 10 10"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.4"
          strokeLinecap="round"
          strokeLinejoin="round"
          aria-hidden="true"
          className={`shrink-0 transition-transform ${open ? 'rotate-90' : ''}`}
        >
          <path d="M3.5 1.5 7 5 3.5 8.5" />
        </svg>
        {t.title}
      </span>
      {!open && <span className="text-[11px] text-muted-foreground">{summary}</span>}
    </button>
  )
}

/** Les deux affinages communs, en segmentés d'une ligne (un choix exclusif chacun). */
function Refinements({ locale, c }: { locale: ReplayLocale; c: ReplayLivesControls }) {
  const t = REPLAY_TEXT[locale].lives
  const whoOptions = LIVES_WHO.filter((w) => w !== 'followed' || c.followedAvailable).map((w) => ({
    value: w,
    label: t.who[w],
    hint: t.whoHint[w],
  }))
  const whenOptions = LIVES_WHEN.map((w) => ({ value: w, label: t.when[w], hint: t.whenHint[w] }))
  return (
    <>
      <SettingsSegments label={t.whoLabel} value={c.who} options={whoOptions} onSelect={c.onSelectWho} />
      <SettingsSegments label={t.whenLabel} value={c.when} options={whenOptions} onSelect={c.onSelectWhen} />
    </>
  )
}

export function LivesSection({ locale, lives }: { locale: ReplayLocale; lives: ReplayLivesControls }) {
  const t = REPLAY_TEXT[locale].lives
  const forecast = REPLAY_TEXT[locale].respawnForecast
  const [open, toggleOpen] = usePersistedFlag(LIVES_OPEN_KEY, false)
  const layers = [
    { key: 'spawns', c: lives.spawns, label: t.layerSpawns, hint: t.layerSpawnsHint },
    { key: 'deaths', c: lives.deaths, label: t.layerDeaths, hint: t.layerDeathsHint },
    { key: 'killers', c: lives.killers, label: t.layerKillers, hint: t.layerKillersHint },
  ].filter((l) => l.c.available)
  const forecastShown = lives.respawnForecast.available
  const total = layers.length + (forecastShown ? 1 : 0)
  if (total === 0) return null
  const tracesOn = layers.filter((l) => l.c.show).length
  const on = tracesOn + (forecastShown && lives.respawnForecast.show ? 1 : 0)
  return (
    <section className="space-y-0.5">
      <GroupHeader locale={locale} open={open} onToggle={toggleOpen} summary={t.summaryFmt(on, total)} />
      {open && (
        <>
          {layers.length > 0 && (
            <div className="grid grid-cols-2 gap-x-3 gap-y-0.5">
              {layers.map((l) => (
                <SettingsToggle key={l.key} label={l.label} pressed={l.c.show} onToggle={l.c.onToggle} hint={l.hint} />
              ))}
            </div>
          )}
          {forecastShown && (
            <>
              <SettingsToggle
                label={forecast.title}
                pressed={lives.respawnForecast.show}
                onToggle={lives.respawnForecast.onToggle}
                hint={forecast.hint}
              />
              {lives.respawnForecast.show && (
                <SettingsSegments
                  label={forecast.drawLabel}
                  value={lives.respawnForecast.draw}
                  options={FORECAST_DRAWS.map((d) => ({ value: d, label: forecast.draw[d], hint: forecast.drawHint[d] }))}
                  onSelect={lives.respawnForecast.onSelectDraw}
                />
              )}
            </>
          )}
          {tracesOn > 0 && <Refinements locale={locale} c={lives} />}
        </>
      )}
    </section>
  )
}
