/**
 * ReplayWeaponsRow — les CELLULES D'ARME d'une fiche joueur, lues aux images-clés, l'arme EN
 * MAIN à gauche : DEUX cellules sur la fiche normale, UNE SEULE sur la tuile compacte.
 *
 * LA RANGÉE EST UNE GRILLE À CELLULES FIXES depuis le 2026-08-24 (demande utilisateur :
 * « que les armes, grenades et équipements soient comme sur une grille pour faciliter la
 * lecture de plusieurs fiches ») : les cellules d'arme sont toujours rendues, toujours à la
 * même largeur — une arme absente laisse sa cellule vide plutôt que de décaler les
 * colonnes voisines. C'est ce qui aligne les fiches entre elles : les icônes d'armes ont
 * des largeurs naturelles différentes, et une rangée fluide décalait tout.
 *
 * L'ORDRE EST CELUI DU JEU : l'arme dégainée d'abord, la secondaire ensuite. Il vient du
 * sélecteur d'emplacement (`Inventory.D`, cf. equippedLogic) — quand il n'est pas lisible,
 * ou qu'il dit « rien de dégainé » (D=2, l'état d'avant le départ), l'ordre reste celui des
 * emplacements et rien n'est marqué.
 *
 * UNE SEULE MARQUE, ET ELLE DIT « EN MAIN » (option 2a du handoff 2026-08-27) : la cellule
 * de l'arme dégainée s'ALLUME — un fond à l'encre du thème très diluée, coins 2 px — et
 * l'icône reste en pleine encre ; l'arme rangée s'estompe, sans fond. Le souligné vert
 * d'avant empruntait le token `success` à la jauge de santé pour dire autre chose.
 * Sans sélecteur lu, les deux cellules gardent la même encre, aucune ne s'allume.
 *
 * UNE SEULE CELLULE SUR LA TUILE COMPACTE (`weaponCells: 1`, plan fiches compactes 2026-09-06,
 * étape 3) : celle de la main. L'arme rangée est NOMMÉE dans l'infobulle (`weaponStowedFmt`),
 * avec les munitions et les marques de la main que la fiche compose (`handHint`, cf.
 * `model/handCellHint.ts`) et l'âge de la lecture — le tout sur la RANGÉE, dont la boîte est
 * alors exactement la cellule : la vignette ne porte pas d'infobulle propre, pour ne pas
 * masquer celle-là.
 *
 * L'ÉCHANGE D'ARME RESTE ANIMÉ — c'est LUI qui dit la permutation. À deux cellules, les deux
 * vignettes permutent en se croisant (`--replay-dx` par défaut : la place de l'autre). À une
 * cellule, seule la vignette ENTRANTE glisse, de la largeur de la cellule (`--replay-dx` posé
 * par la vignette) : l'échange reste visible, il ne croise plus rien (I6). Le libellé
 * « échange » qui doublait cet effet est SUPPRIMÉ (demande utilisateur du 2026-08-24), avec sa
 * lecture (`loadoutSwapAt`).
 *
 * LE BADGE DE LANCER (le `.gic` du POC) SE SUPERPOSE à la cellule de la main : l'objet
 * réellement actif à cet instant n'est plus une arme, c'est la grenade — elle prend la
 * place de la main sans pousser les colonnes. Sur une cellule de 48 px, il fait 48 px.
 *
 * CE QUE CETTE RANGÉE NE DIT PAS : la CONTINUITÉ. Une image-clé toutes les ~20 s, un
 * ramassage entre deux est invisible. D'où l'estompage avec l'âge, l'âge exact dans
 * l'infobulle.
 */
import type { CSSProperties } from 'react'

import { WeaponIcon } from '@/components/ui/WeaponIcon'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'

import { catalogText } from '../i18n/catalogLabel'
import type { CardGabarit } from '../model/cardGabarit'
import { drawnSwapAt, type EquippedReading } from '../model/equippedLogic'
import { GRENADE_THROW_HOLD_MS, grenadeThrowActive } from '../model/grenadeFx'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { formatSeconds, frameToMs, freshness, msToFrames, READING_FADE } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { PlayerState } from '../../../lib/replay/rosterLogic'
import { MIRROR_STYLE, weaponFullIcon } from '../model/weaponFullIcon'

/** Durée de l'animation d'échange — celle du POC, calée sur la rémanence des lancers. */
const SWAP_ANIM_MS = 340

/**
 * Cellule d'arme : GABARIT FIXE (la grille des fiches en dépend). La LARGEUR vient du gabarit
 * (`weaponCellW` : 40 en normal, 48 en compact — `model/cardGabarit.ts`, 2026-09-06) ; la
 * hauteur est celle de la ligne.
 */
const CELL_H = 16
/** L'icône dans sa cellule : 38 × 13, centrée — la cellule allumée garde une marge d'encre. */
const ICON_W = 38
const ICON_H = 13
/** Fond de la cellule « en main » : l'encre du thème diluée (remplace le souligné vert). */
const HAND_BG_PCT = 7
/** Estompage de l'arme RANGÉE quand le sélecteur est lu (option 2a : .35). */
const STOWED_OPACITY = 0.35
/** Hauteur de la vignette du badge de lancer (POC .gic : 15 px). */
const GIC_H = 15

type ReplayText = (typeof REPLAY_TEXT)[ReplayLocale]

export function ReplayWeaponsRow({
  doc,
  state,
  read,
  frame,
  readingFull,
  filmIndex,
  locale,
  gabarit,
  handHint,
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
  /** Les cotes de la fiche : le nombre de cellules d'arme et leur largeur. */
  gabarit: CardGabarit
  /**
   * Ce que la MAIN a à dire quand la rangée n'a qu'une cellule (munitions, marques de la
   * lecture d'inventaire — `model/handCellHint.ts`) ; jamais donné à deux cellules.
   */
  handHint?: string
}) {
  const t = REPLAY_TEXT[locale]
  const cellW = gabarit.weaponCellW
  const seule = gabarit.weaponCells === 1
  if (!state.life) return null
  // Le badge de lancer : un ÉVÉNEMENT daté, jamais estompé par l'âge de lecture.
  const throwHold = Math.max(1, msToFrames(GRENADE_THROW_HOLD_MS, doc))
  const activeThrow = filmIndex !== null ? grenadeThrowActive(doc, filmIndex, frame, throwHold) : null
  const gic = activeThrow ? (
    <GrenadeThrowBadge doc={doc} rank={activeThrow.rank} throwFrame={frame - activeThrow.age} locale={locale} />
  ) : null
  if (!read) {
    // Loadout non lu : les cellules restent, vides en pointillés — la grille des fiches ne
    // bouge pas, la lacune est dite en infobulle (avec ce que la main sait encore dire).
    return (
      <span className="inline-flex items-center gap-[5px]" title={[t.loadoutUnread, handHint].filter(Boolean).join(' · ')}>
        <span className="relative inline-flex">
          <EmptyWeaponCell width={cellW} />
          {gic}
        </span>
        {!seule && <EmptyWeaponCell width={cellW} />}
      </span>
    )
  }
  const swapFrames = Math.max(1, msToFrames(SWAP_ANIM_MS, doc))
  const swapAge = drawnSwapAt(doc, state.life.slot, frame, swapFrames)
  const drawnKnown = read.drawn !== null
  // Âge NÉGATIF = lecture d'une image-clé À VENIR (début de vie, cf. loadoutAt) : l'infobulle
  // le dit — l'estompage, lui, porte déjà sur la valeur absolue.
  const ageMs = frameToMs(Math.abs(read.age), doc)
  const ageTxt =
    read.age < 0
      ? `${t.loadoutAhead} ${formatSeconds(ageMs)}`
      : `${t.loadoutAge} ${formatSeconds(ageMs)}`
  const cells: (typeof read.weapons[number] | null)[] = seule
    ? [read.weapons[0] ?? null]
    : [read.weapons[0] ?? null, read.weapons[1] ?? null]
  return (
    <div
      className="flex items-center gap-[5px] font-mono text-[9.5px] text-muted-foreground"
      style={{ opacity: freshness(read.age, readingFull, READING_FADE) }}
      title={seule ? handCellTitle({ t, doc, read, locale, handHint, ageTxt }) : ageTxt}
    >
      {cells.map((w, k) => (
        <span key={k} className="relative inline-flex">
          {w ? (
            <WeaponChip
              doc={doc}
              id={w.id}
              inHand={w.inHand}
              dimmed={drawnKnown && !w.inHand}
              swap={
                swapAge !== null
                  ? { cls: k === 0 ? 'replay-wswap-l' : 'replay-wswap-r', age: swapAge, span: swapFrames, dx: seule ? cellW : null }
                  : null
              }
              hint={!seule && !w.inHand && drawnKnown ? t.weaponSecondaryHint : undefined}
              locale={locale}
              cellW={cellW}
            />
          ) : (
            <EmptyWeaponCell width={cellW} />
          )}
          {/* Le badge de lancer COUVRE la cellule de la main (k = 0) : les opacités CSS se
              multiplient, le sortir du bloc estompé le garderait à pleine encre. */}
          {k === 0 && gic}
        </span>
      ))}
    </div>
  )
}

/**
 * handCellTitle — l'infobulle de la cellule SEULE : l'arme rangée NOMMÉE (quand le sélecteur
 * est lu, c'est bien une arme rangée ; sinon la seconde arme est nommée sans le dire), puis ce
 * que la main dit (munitions, marques), puis l'âge de la lecture.
 */
function handCellTitle({
  t, doc, read, locale, handHint, ageTxt,
}: {
  t: ReplayText
  doc: ReplayDocumentReady
  read: EquippedReading
  locale: ReplayLocale
  handHint: string | undefined
  ageTxt: string
}): string {
  const stowed = read.weapons[1]
  const stowedName = stowed ? (catalogText(doc.weaponLabels?.[stowed.id], locale) ?? stowed.id) : null
  return [
    stowedName === null ? null : read.drawn !== null ? t.weaponStowedFmt(stowedName) : stowedName,
    handHint,
    ageTxt,
  ]
    .filter((p): p is string => !!p)
    .join(' · ')
}

/** EmptyWeaponCell — une cellule d'arme sans arme : la place, rien d'autre. */
function EmptyWeaponCell({ width }: { width: number }) {
  return (
    <span
      aria-hidden
      className="inline-block rounded-[2px] border border-dashed border-border/50"
      style={{ width, height: CELL_H }}
    />
  )
}

/**
 * GrenadeThrowBadge — le `.gic` du POC : la vignette du TYPE lancé, allumée à l'accent,
 * avec un POP d'apparition, posée SUR la cellule de la main. La clé porte la frame du
 * lancer : React garde l'identité du nœud pendant la rémanence, l'animation ne repart pas
 * à chaque image publiée.
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
  const lbl = doc.grenadeLabels[rank]
  const name = catalogText(lbl, locale) ?? `${rank}`
  const tooltip = `${t.grenadeThrown} — ${name}`
  // MÊME VIGNETTE QUE LA CELLULE DE GRENADE : le masque de HUD cuit dans l'artefact
  // (option 2a du handoff 2026-08-27 — la version PLEINE, teinte par currentColor, a
  // remplacé l'image versionnée du 16/08). Deux dessins différents pour la même grenade
  // se liraient comme deux grenades.
  const icon = lbl?.img ? { url: lbl.img, tinted: !!lbl.tinted } : null
  return (
    <span
      key={`gic-${throwFrame}`}
      className="replay-gpop absolute inset-0 z-10 inline-flex items-center justify-center rounded-sm"
      style={{
        outline: `1px solid ${tokenCssVar('warning')}`,
        background: `color-mix(in srgb, ${tokenCssVar('warning')} 13%, var(--card))`,
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

/** Le style d'une vignette, avec la variable de distance de l'échange quand elle est posée. */
type ChipStyle = CSSProperties & { '--replay-dx'?: string }

/**
 * WeaponChip — UNE vignette d'arme dans sa cellule fixe : l'icône extraite quand le
 * document en pointe une, le libellé sinon (jamais le visuel d'une arme voisine, tronqué
 * plutôt que débordant — le nom complet vit dans l'aria-label). La marque « en main » est
 * la cellule ALLUMÉE (fond à l'encre diluée, icône en pleine encre) ; l'arme rangée
 * s'estompe quand le sélecteur est lu.
 */
function WeaponChip({
  doc,
  id,
  inHand,
  dimmed,
  swap,
  hint,
  locale,
  cellW,
}: {
  doc: ReplayDocumentReady
  id: string
  inHand: boolean
  dimmed: boolean
  /**
   * L'échange en cours : sa classe (entrante / sortante), son avancement, et `dx` — la
   * distance posée sur la vignette (la largeur de la cellule, à une cellule) ou null pour la
   * distance par défaut de la feuille de style (la place de l'autre cellule).
   */
  swap: { cls: string; age: number; span: number; dx: number | null } | null
  hint?: string
  locale: ReplayLocale
  /** Largeur de la cellule, du gabarit de la fiche. */
  cellW: number
}) {
  const lbl = doc.weaponLabels?.[id]
  const name = catalogText(lbl, locale)
  // LA VERSION PLEINE, RETOURNÉE (option 2a) : l'atlas silhouette à la place du contour
  // cuit dans l'artefact, rendue dans le sens du kill feed du jeu (cf. weaponFullIcon.ts).
  const icon = lbl?.img ? weaponFullIcon(lbl.img) : null
  const style: ChipStyle = {
    width: cellW,
    height: CELL_H,
    background: inHand
      ? `color-mix(in srgb, var(--foreground) ${HAND_BG_PCT}%, transparent)`
      : undefined,
    opacity: dimmed ? STOWED_OPACITY : 1,
  }
  if (swap) {
    // Délai négatif = avancement réel : l'animation reprend où elle en était malgré les
    // re-rendus, et reste juste après un saut de lecture.
    style.animationDelay = `${(-(Math.min(swap.age, swap.span) / swap.span) * (SWAP_ANIM_MS / 1000)).toFixed(3)}s`
    if (swap.dx !== null) style['--replay-dx'] = `${swap.dx}px`
  }
  return (
    <span
      className={`inline-flex items-center justify-center overflow-hidden rounded-[2px] ${swap ? swap.cls : ''}`}
      style={style}
      title={hint}
    >
      {icon ? (
        <WeaponIcon
          imageUrl={icon.url}
          tinted={lbl?.tinted}
          label={name ?? id}
          width={ICON_W}
          height={ICON_H}
          className={inHand ? 'text-foreground' : undefined}
          style={icon.mirrored ? MIRROR_STYLE : undefined}
        />
      ) : (
        <span
          className={[
            'truncate',
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
