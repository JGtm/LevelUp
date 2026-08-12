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
 * UNE SEULE MARQUE, ET ELLE DIT « EN MAIN » : l'arme dégainée est soulignée et en pleine
 * encre, l'autre s'estompe. Sans sélecteur lu, les deux gardent la même encre — n'estomper
 * qu'une dirait un faux. (La notion de « primaire » du record n'est pas montrée : elle
 * vient du format, pas du jeu.)
 *
 * LES VISUELS SONT LES ICÔNES EXTRAITES DU JEU, servies par le document (`weaponLabels`,
 * champ `img`) et rendues par le composant commun `WeaponIcon`. Une arme sans visuel garde
 * son LIBELLÉ — jamais l'icône d'une arme voisine — et une arme sans libellé garde son
 * identifiant, souligné en pointillés.
 *
 * L'ÉCHANGE D'ARME EST ANIMÉ : quand le sélecteur bascule d'un emplacement à l'autre, les
 * deux vignettes permutent en se croisant. Sans mouvement, une permutation est
 * indiscernable d'un changement d'arme. L'animation reçoit un délai NÉGATIF égal à son
 * avancement réel : elle reste juste après un saut dans le temps de lecture.
 *
 * CE QUE CETTE RANGÉE NE DIT PAS : la CONTINUITÉ. Une image-clé toutes les ~20 s, un
 * ramassage entre deux est invisible. D'où l'estompage avec l'âge, l'âge exact dans
 * l'infobulle.
 */
import type { CSSProperties } from 'react'

import { WeaponIcon } from '@/components/ui/WeaponIcon'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'

import { catalogText } from './catalogLabel'
import { drawnSwapAt, loadoutSwapAt, type EquippedReading, type SwapReading } from './equippedLogic'
import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import { formatSeconds, frameToMs, freshness, msToFrames, READING_FADE } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'
import type { PlayerState } from './rosterLogic'

/** Durée de l'animation d'échange — celle du POC, calée sur la rémanence des lancers. */
const SWAP_ANIM_MS = 340

/** Boîte d'une vignette d'arme : hauteur de la ligne, largeur bornée. */
const ICON_H = 16
const ICON_W = 40

export function ReplayWeaponsRow({
  doc,
  state,
  read,
  frame,
  readingFull,
  locale,
}: {
  doc: ReplayDocumentReady
  state: PlayerState
  /** La lecture ORDONNÉE des armes — calculée par la fiche, partagée avec les munitions. */
  read: EquippedReading | null
  frame: number
  readingFull: number
  locale: ReplayLocale
}) {
  const t = REPLAY_TEXT[locale]
  if (!state.life) return null
  if (!read) {
    return <span className="font-mono text-[9.5px] italic text-muted-foreground/70">{t.loadoutUnread}</span>
  }
  const swapFrames = Math.max(1, msToFrames(SWAP_ANIM_MS, doc))
  const swapAge = drawnSwapAt(doc, state.life.slot, frame, swapFrames)
  const swap = loadoutSwapAt(doc, state.life.slot, frame)
  const drawnKnown = read.drawn !== null
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
      {read.weapons.map((w, k) => (
        <WeaponChip
          key={`${k}-${w.id}`}
          doc={doc}
          id={w.id}
          inHand={w.inHand}
          dimmed={drawnKnown && !w.inHand}
          swap={swapAge !== null ? { cls: k === 0 ? 'replay-wswap-l' : 'replay-wswap-r', age: swapAge, span: swapFrames } : null}
          hint={w.inHand ? t.weaponInHandHint : drawnKnown ? t.weaponSecondaryHint : undefined}
          locale={locale}
        />
      ))}
      {swap && <SwapMark swap={swap} doc={doc} locale={locale} />}
    </div>
  )
}

/**
 * WeaponChip — UNE vignette d'arme : l'icône extraite quand le document en pointe une, le
 * libellé sinon (jamais le visuel d'une arme voisine). La marque « en main » est un souligné
 * plein encre ; l'autre arme s'estompe quand le sélecteur est lu.
 */
function WeaponChip({
  doc,
  id,
  inHand,
  dimmed,
  swap,
  hint,
  locale,
}: {
  doc: ReplayDocumentReady
  id: string
  inHand: boolean
  dimmed: boolean
  swap: { cls: string; age: number; span: number } | null
  hint?: string
  locale: ReplayLocale
}) {
  const lbl = doc.weaponLabels?.[id]
  const name = catalogText(lbl, locale)
  const style: CSSProperties = {
    borderBottom: `2px solid ${inHand ? tokenCssVar('success') : 'transparent'}`,
    opacity: dimmed ? 0.45 : 1,
  }
  if (swap) {
    // Délai négatif = avancement réel : l'animation reprend où elle en était malgré les
    // re-rendus, et reste juste après un saut de lecture.
    style.animationDelay = `${(-(Math.min(swap.age, swap.span) / swap.span) * (SWAP_ANIM_MS / 1000)).toFixed(3)}s`
  }
  return (
    <span
      className={`inline-flex items-center px-0.5 ${swap ? swap.cls : ''}`}
      style={style}
      title={hint}
    >
      {lbl?.img ? (
        <WeaponIcon
          imageUrl={lbl.img}
          tinted={lbl.tinted}
          label={name ?? id}
          width={ICON_W}
          height={ICON_H}
          className={inHand ? 'text-foreground' : undefined}
        />
      ) : (
        <span
          className={[
            inHand ? 'font-semibold text-foreground' : '',
            name ? '' : 'border-b border-dashed border-border',
          ]
            .filter(Boolean)
            .join(' ')}
        >
          {name ?? id}
        </span>
      )}
    </span>
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
