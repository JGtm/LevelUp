/**
 * ReplayWeaponsRow — les DEUX CELLULES D'ARME d'une fiche joueur, lues aux images-clés,
 * l'arme EN MAIN à gauche.
 *
 * LA RANGÉE EST UNE GRILLE À CELLULES FIXES depuis le 2026-08-24 (demande utilisateur :
 * « que les armes, grenades et équipements soient comme sur une grille pour faciliter la
 * lecture de plusieurs fiches ») : DEUX cellules d'arme, toujours rendues, toujours à la
 * même largeur — une arme absente laisse sa cellule vide plutôt que de décaler les
 * colonnes voisines. C'est ce qui aligne les fiches entre elles : les icônes d'armes ont
 * des largeurs naturelles différentes, et une rangée fluide décalait tout.
 *
 * L'ORDRE EST CELUI DU JEU : l'arme dégainée d'abord, la secondaire ensuite. Il vient du
 * sélecteur d'emplacement (`Inventory.D`, cf. equippedLogic) — quand il n'est pas lisible,
 * ou qu'il dit « rien de dégainé » (D=2, l'état d'avant le départ), l'ordre reste celui des
 * emplacements et rien n'est marqué.
 *
 * UNE SEULE MARQUE, ET ELLE DIT « EN MAIN » : l'arme dégainée est soulignée et en pleine
 * encre, l'autre s'estompe. Sans sélecteur lu, les deux gardent la même encre.
 *
 * L'ÉCHANGE D'ARME RESTE ANIMÉ (les deux vignettes permutent en se croisant) — c'est LUI
 * qui dit la permutation. Le libellé « échange » qui doublait cet effet est SUPPRIMÉ
 * (demande utilisateur du 2026-08-24), avec sa lecture (`loadoutSwapAt`).
 *
 * LE BADGE DE LANCER (le `.gic` du POC) SE SUPERPOSE à la cellule de la main : l'objet
 * réellement actif à cet instant n'est plus une arme, c'est la grenade — elle prend la
 * place de la main sans pousser les colonnes.
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
import { drawnSwapAt, type EquippedReading } from './equippedLogic'
import { GRENADE_THROW_HOLD_MS, grenadeThrowActive } from './grenadeFx'
import { grenadeIconOf } from './grenadeIcon'
import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import { formatSeconds, frameToMs, freshness, msToFrames, READING_FADE } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'
import type { PlayerState } from './rosterLogic'

/** Durée de l'animation d'échange — celle du POC, calée sur la rémanence des lancers. */
const SWAP_ANIM_MS = 340

/** Cellule d'arme : LARGEUR FIXE (la grille des fiches en dépend), hauteur de la ligne. */
export const WEAPON_CELL_W = 40
const ICON_H = 16
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
  // Le badge de lancer : un ÉVÉNEMENT daté, jamais estompé par l'âge de lecture.
  const throwHold = Math.max(1, msToFrames(GRENADE_THROW_HOLD_MS, doc))
  const activeThrow = filmIndex !== null ? grenadeThrowActive(doc, filmIndex, frame, throwHold) : null
  const gic = activeThrow ? (
    <GrenadeThrowBadge doc={doc} rank={activeThrow.rank} throwFrame={frame - activeThrow.age} locale={locale} />
  ) : null
  if (!read) {
    // Loadout non lu : les DEUX cellules restent, vides en pointillés — la grille des
    // fiches ne bouge pas, la lacune est dite en infobulle.
    return (
      <span className="inline-flex items-center gap-1" title={t.loadoutUnread}>
        <span className="relative inline-flex">
          <EmptyWeaponCell />
          {gic}
        </span>
        <EmptyWeaponCell />
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
  const cells: (typeof read.weapons[number] | null)[] = [read.weapons[0] ?? null, read.weapons[1] ?? null]
  return (
    <div
      className="flex items-center gap-1 font-mono text-[9.5px] text-muted-foreground"
      style={{ opacity: freshness(read.age, readingFull, READING_FADE) }}
      title={ageTxt}
    >
      {cells.map((w, k) => (
        <span key={k} className="relative inline-flex">
          {w ? (
            <WeaponChip
              doc={doc}
              id={w.id}
              inHand={w.inHand}
              dimmed={drawnKnown && !w.inHand}
              swap={swapAge !== null ? { cls: k === 0 ? 'replay-wswap-l' : 'replay-wswap-r', age: swapAge, span: swapFrames } : null}
              hint={!w.inHand && drawnKnown ? t.weaponSecondaryHint : undefined}
              locale={locale}
            />
          ) : (
            <EmptyWeaponCell />
          )}
          {/* Le badge de lancer COUVRE la cellule de la main (k = 0) : les opacités CSS se
              multiplient, le sortir du bloc estompé le garderait à pleine encre. */}
          {k === 0 && gic}
        </span>
      ))}
    </div>
  )
}

/** EmptyWeaponCell — une cellule d'arme sans arme : la place, rien d'autre. */
function EmptyWeaponCell() {
  return (
    <span
      aria-hidden
      className="inline-block rounded-sm border border-dashed border-border/50"
      style={{ width: WEAPON_CELL_W, height: ICON_H }}
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
  const titleSlug = useTitleSlug()
  const theme = useSettingsDraftStore((s) => s.localUiPrefs.theme)
  const lbl = doc.grenadeLabels[rank]
  const name = catalogText(lbl, locale) ?? `${rank}`
  const tooltip = `${t.grenadeThrown} — ${name}`
  // MÊME VIGNETTE QUE LA CELLULE DE GRENADE (grenadeIcon.ts) : le type qui part est le même
  // objet que celui qu'on compte dans la même rangée — deux dessins différents pour la même
  // grenade se liraient comme deux grenades.
  const icon = grenadeIconOf(lbl, titleSlug, theme)
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

/**
 * WeaponChip — UNE vignette d'arme dans sa cellule fixe : l'icône extraite quand le
 * document en pointe une, le libellé sinon (jamais le visuel d'une arme voisine, tronqué
 * plutôt que débordant — le nom complet vit dans l'aria-label). La marque « en main » est
 * un souligné plein encre ; l'autre arme s'estompe quand le sélecteur est lu.
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
    width: WEAPON_CELL_W,
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
      className={`inline-flex items-center justify-center overflow-hidden ${swap ? swap.cls : ''}`}
      style={style}
      title={hint}
    >
      {lbl?.img ? (
        <WeaponIcon
          imageUrl={lbl.img}
          tinted={lbl.tinted}
          label={name ?? id}
          width={WEAPON_CELL_W}
          height={ICON_H}
          className={inHand ? 'text-foreground' : undefined}
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
