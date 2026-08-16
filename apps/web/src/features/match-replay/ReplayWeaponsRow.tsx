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

import { useTitleSlug } from '@/lib/title-routing/useTitleSlug'
import { useSettingsDraftStore } from '@/stores/settingsDraftStore'

import { catalogText } from './catalogLabel'
import { drawnSwapAt, loadoutSwapAt, type EquippedReading, type SwapReading } from './equippedLogic'
import { GRENADE_THROW_HOLD_MS, grenadeThrowActive } from './grenadeFx'
import { grenadeIconOf } from './grenadeIcon'
import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import { formatSeconds, frameToMs, freshness, msToFrames, READING_FADE } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'
import type { PlayerState } from './rosterLogic'

/** Durée de l'animation d'échange — celle du POC, calée sur la rémanence des lancers. */
const SWAP_ANIM_MS = 340

/** Boîte d'une vignette d'arme : hauteur de la ligne, largeur bornée. */
const ICON_H = 16
const ICON_W = 40
/** Hauteur de la vignette du badge de lancer (POC .gic : 15 px). */
const GIC_H = 15

export function ReplayWeaponsRow({
  doc,
  state,
  read,
  frame,
  readingFull,
  filmIndex,
  locale,
}: {
  doc: ReplayDocumentReady
  state: PlayerState
  /** La lecture ORDONNÉE des armes — calculée par la fiche, partagée avec les munitions. */
  read: EquippedReading | null
  frame: number
  readingFull: number
  /** Index de film du joueur (roster) — la clé des LANCERS ; null si le roster le tait. */
  filmIndex: number | null
  locale: ReplayLocale
}) {
  const t = REPLAY_TEXT[locale]
  if (!state.life) return null
  // LE BADGE DE LANCER (le `.gic` du POC) : l'objet réellement actif à cet instant n'est
  // plus une arme, c'est la grenade — elle prend la PREMIÈRE place, celle de la main.
  // L'auteur est ÉCRIT dans le film (pas deviné) ; le badge n'est PAS estompé par l'âge
  // de lecture : c'est un ÉVÉNEMENT daté, pas une lecture d'image-clé.
  const throwHold = Math.max(1, msToFrames(GRENADE_THROW_HOLD_MS, doc))
  const activeThrow = filmIndex !== null ? grenadeThrowActive(doc, filmIndex, frame, throwHold) : null
  const gic = activeThrow ? (
    <GrenadeThrowBadge doc={doc} rank={activeThrow.rank} throwFrame={frame - activeThrow.age} locale={locale} />
  ) : null
  if (!read) {
    return (
      <span className="inline-flex items-center gap-1.5">
        {gic}
        <span className="font-mono text-[9.5px] italic text-muted-foreground/70">{t.loadoutUnread}</span>
      </span>
    )
  }
  const swapFrames = Math.max(1, msToFrames(SWAP_ANIM_MS, doc))
  const swapAge = drawnSwapAt(doc, state.life.slot, frame, swapFrames)
  const swap = loadoutSwapAt(doc, state.life.slot, frame)
  const drawnKnown = read.drawn !== null
  // Âge NÉGATIF = lecture d'une image-clé À VENIR (début de vie, cf. loadoutAt) : l'infobulle
  // le dit — l'estompage, lui, porte déjà sur la valeur absolue.
  const ageMs = frameToMs(Math.abs(read.age), doc)
  const ageTxt =
    read.age < 0
      ? `${t.loadoutAhead} ${formatSeconds(ageMs)}`
      : `${t.loadoutAge} ${formatSeconds(ageMs)}`
  // L'état « armes rangées » n'est plus dit ici : son pictogramme (et son unique
  // infobulle) vit dans la rangée d'inventaire — décision produit 4 du plan parité.
  return (
    <div className="flex flex-wrap items-center gap-x-1.5 gap-y-0.5 font-mono text-[9.5px] text-muted-foreground">
      {/* Le badge de lancer vit HORS du bloc estompé : les opacités CSS se MULTIPLIENT,
          le poser dedans le ferait pâlir avec la lecture d'armes alors qu'il est un
          événement daté, sans âge à montrer (règle du POC). */}
      {gic}
      <div
        className="flex flex-wrap items-center gap-x-1.5 gap-y-0.5"
        style={{ opacity: freshness(read.age, readingFull, READING_FADE) }}
        title={ageTxt}
      >
        {read.weapons.map((w, k) => (
          <WeaponChip
            key={`${k}-${w.id}`}
            doc={doc}
            id={w.id}
            inHand={w.inHand}
            dimmed={drawnKnown && !w.inHand}
            swap={swapAge !== null ? { cls: k === 0 ? 'replay-wswap-l' : 'replay-wswap-r', age: swapAge, span: swapFrames } : null}
            hint={!w.inHand && drawnKnown ? t.weaponSecondaryHint : undefined}
            locale={locale}
          />
        ))}
        {swap && <SwapMark swap={swap} doc={doc} locale={locale} />}
      </div>
    </div>
  )
}

/**
 * GrenadeThrowBadge — le `.gic` du POC : la vignette du TYPE lancé, allumée à l'accent,
 * avec un POP d'apparition (sans mouvement, une permutation d'objet actif ne se voit
 * pas). La clé porte la frame du lancer : React garde l'identité du nœud pendant la
 * rémanence, l'animation ne repart pas à chaque image publiée.
 */
function GrenadeThrowBadge({
  doc,
  rank,
  throwFrame,
  locale,
}: {
  doc: ReplayDocumentReady
  rank: number
  throwFrame: number
  locale: ReplayLocale
}) {
  const t = REPLAY_TEXT[locale]
  const titleSlug = useTitleSlug()
  const theme = useSettingsDraftStore((s) => s.localUiPrefs.theme)
  const lbl = doc.grenadeLabels[rank]
  const name = catalogText(lbl, locale) ?? `${rank}`
  const tooltip = `${t.grenadeThrown} — ${name}`
  // MÊME VIGNETTE QUE LA LIGNE D'INVENTAIRE (grenadeIcon.ts) : le type qui part est le même
  // objet que celui qu'on compte deux lignes plus bas — deux dessins différents pour la même
  // grenade se liraient comme deux grenades.
  const icon = grenadeIconOf(lbl, titleSlug, theme)
  return (
    <span
      key={`gic-${throwFrame}`}
      className="replay-gpop inline-flex items-center rounded-sm px-0.5"
      style={{
        outline: `1px solid ${tokenCssVar('warning')}`,
        background: `color-mix(in srgb, ${tokenCssVar('warning')} 13%, transparent)`,
      }}
      title={tooltip}
    >
      {icon ? (
        <WeaponIcon
          imageUrl={icon.url}
          tinted={icon.tinted}
          label={tooltip}
          width={GIC_H}
          height={GIC_H}
          className="text-foreground"
        />
      ) : (
        <span role="img" aria-label={tooltip} className="font-semibold text-foreground">
          {name}
        </span>
      )}
    </span>
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
  // L'infobulle garde le DÉTAIL (entrées, sorties, âge) et perd la phrase de méthode sur
  // les images-clés : l'information reste, le jargon de flux sort de l'écran.
  return (
    <span
      className="text-[8px] uppercase tracking-wide opacity-70"
      title={`${parts.join(' ; ')}. ${when}.`}
    >
      {t.weaponSwap}
    </span>
  )
}
