/**
 * ReplayWeaponsRow — les armes portées d'une fiche joueur, lues aux images-clés, l'arme
 * EN MAIN à gauche. Extrait de ReplayTeams.tsx (seuil de taille du dépôt) : la rangée
 * d'armes et son indicateur de swap forment une unité, adossée à `equippedLogic`.
 *
 * L'ORDRE EST CELUI DU JEU : l'arme dégainée d'abord, la secondaire ensuite. Il vient du
 * sélecteur d'emplacement (`Inventory.D`, cf. equippedLogic) — quand il n'est pas lisible,
 * ou qu'il dit « rien de dégainé » (D=2, l'état d'avant le départ), l'ordre reste celui des
 * emplacements et rien n'est marqué.
 *
 * CE QUE CETTE RANGÉE NE DIT PAS : la CONTINUITÉ. Une image-clé toutes les ~20 s, un
 * ramassage entre deux est invisible. D'où l'estompage avec l'âge, l'âge exact dans
 * l'infobulle — et l'indicateur de swap, qui date un CHANGEMENT entre les deux dernières
 * lectures sans prétendre le suivre.
 *
 * UNE ARME NON CATALOGUÉE GARDE SON IDENTIFIANT plutôt que d'emprunter un nom voisin.
 */
import { catalogText } from './catalogLabel'
import { equippedWeapons, loadoutSwapAt, type SwapReading } from './equippedLogic'
import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import { formatSeconds, frameToMs, freshness, READING_FADE } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'
import type { PlayerState } from './rosterLogic'

export function ReplayWeaponsRow({
  doc,
  state,
  frame,
  readingFull,
  locale,
}: {
  doc: ReplayDocumentReady
  state: PlayerState
  frame: number
  readingFull: number
  locale: ReplayLocale
}) {
  const t = REPLAY_TEXT[locale]
  if (!state.life) return null
  const read = equippedWeapons(doc, state.life.slot, frame)
  if (!read) {
    return <span className="font-mono text-[9.5px] italic text-muted-foreground/70">{t.loadoutUnread}</span>
  }
  const swap = loadoutSwapAt(doc, state.life.slot, frame)
  // « Secondaire » n'a de sens que si une arme est EN MAIN : sans sélecteur lu, aucune des
  // deux n'est désignée, et on ne baptise pas l'ordre d'emplacement.
  const drawnKnown = read.weapons.some((w) => w.inHand)
  const ageMs = frameToMs(read.age, doc)
  const hint = read.holstered
    ? `${t.weaponsHolstered} — ${t.loadoutAge} ${formatSeconds(ageMs)}`
    : `${t.loadoutAge} ${formatSeconds(ageMs)}`
  return (
    <div
      className="flex flex-wrap items-center gap-x-1.5 gap-y-0.5 font-mono text-[9.5px] text-muted-foreground"
      style={{ opacity: freshness(read.age, readingFull, READING_FADE) }}
      title={hint}
    >
      {read.weapons.map((w, i) => {
        // Le tag brut reste À CÔTÉ du libellé, jamais à sa place : une arme hors
        // catalogue garde son hexadécimal, souligné en pointillés pour le dire.
        const name = catalogText(doc.weaponLabels?.[w.id], locale)
        const classes = [
          w.inHand ? 'font-semibold text-foreground' : '',
          name ? '' : 'border-b border-dashed border-border',
        ]
          .filter(Boolean)
          .join(' ')
        return (
          <span
            key={`${i}-${w.id}`}
            className={classes || undefined}
            title={drawnKnown && !w.inHand ? t.weaponSecondaryHint : undefined}
          >
            {name ?? w.id}
            {w.inHand && (
              <i
                className="ml-0.5 align-middle text-[8px] not-italic uppercase tracking-wide text-muted-foreground"
                title={t.weaponInHandHint}
              >
                {t.weaponInHand}
              </i>
            )}
          </span>
        )
      })}
      {swap && <SwapMark swap={swap} doc={doc} locale={locale} />}
    </div>
  )
}

/**
 * SwapMark — le CHANGEMENT de dotation entre les deux dernières lectures, daté.
 *
 * L'indicateur reste discret parce que l'information est GROSSIÈRE : une image-clé toutes
 * les ~20 s, donc « échange » veut dire « quelque part dans cet intervalle », jamais « à
 * l'instant ». Le détail — ce qui est entré, ce qui est sorti, l'âge de la lecture — vit
 * dans l'infobulle, avec les noms du catalogue quand ils existent.
 */
function SwapMark({
  swap,
  doc,
  locale,
}: {
  swap: SwapReading
  doc: ReplayDocumentReady
  locale: ReplayLocale
}) {
  const t = REPLAY_TEXT[locale]
  const name = (id: string) => catalogText(doc.weaponLabels?.[id], locale) ?? id
  const parts: string[] = []
  if (swap.picked.length > 0) parts.push(`+ ${swap.picked.map(name).join(', ')}`)
  if (swap.dropped.length > 0) parts.push(`− ${swap.dropped.map(name).join(', ')}`)
  const when = `${t.loadoutAge} ${formatSeconds(frameToMs(swap.age, doc))}`
  return (
    <span
      className="text-[8px] uppercase tracking-wide opacity-70"
      title={`${t.weaponSwapHint} ${parts.join(' ; ')}. ${when}.`}
    >
      {t.weaponSwap}
    </span>
  )
}
