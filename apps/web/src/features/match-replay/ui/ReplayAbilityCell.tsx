/**
 * ReplayAbilityCell.tsx — la CELLULE DE CAPACITÉ de la rangée d'inventaire : la vignette de
 * l'équipement porté, et les CHARGES qui lui restent, dites dans son infobulle.
 *
 * CE QUE L'INFOBULLE DIT DES CHARGES, et pourquoi :
 *  - le COMPTE, quand une lecture du canal i56 couvre la vie et l'équipement courants, avec
 *    l'âge de SA lecture ;
 *  - « PLEIN » QUALITATIF avant la première lecture : le film ne transmet RIEN au ramassage, un
 *    maximum déduit serait inventé ;
 *  - RIEN pour une famille que le canal ne mesure pas (camouflage, surbouclier, répulseur…) :
 *    ni compte, ni « plein » — rien d'affirmé.
 * Le tri entre ces trois états vit dans `abilityChargesAt` / `carriedUsesAt` (purs, testés sans rendu).
 *
 * LA CELLULE GARDE SA LARGEUR FIXE (`iconAbilityPx`) ET N'EN AJOUTE AUCUNE : la rangée n'a pas
 * d'espace souple après ses cellules fixes (elle rogne ce qui déborde), donc un compte posé à
 * côté de la vignette serait coupé. Rien n'est écrit sur la cellule : le compte et le « plein »
 * passent par l'infobulle de la vignette, qui pâlit avec l'âge de la lecture de la capacité.
 */
import type { ReactNode } from 'react'

import { WeaponIcon } from '@/components/ui/WeaponIcon'

import { abilityChargesAt, type AbilityChargeDisplay } from '../model/abilityChargeLogic'
import { carriedUsesAt } from '../model/carriedUsesLogic'
import type { CardGabarit } from '../model/cardGabarit'
import { catalogText, type CatalogLabel } from '../i18n/catalogLabel'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { formatSeconds, frameToMs, freshness, READING_FADE } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { abilityAt } from '../../../lib/replay/slotReadings'

// LA BOÎTE DE LA VIGNETTE DE CAPACITÉ (16 px, la hauteur de la ligne) VIENT DU GABARIT de la
// fiche (`model/cardGabarit.ts`, `iconAbilityPx`, 2026-09-06) — aucune cote locale ici.

type ReplayText = (typeof REPLAY_TEXT)[ReplayLocale]

/**
 * ReplayAbilityCell — la cellule FIXE de la vignette, puis les charges dans l'espace souple.
 *
 * LA CAPACITÉ A SA PROPRE LECTURE, et donc son propre âge : elle arrive surtout par le canal
 * i48 des paquets delta, qui ne tombe pas sur les images-clés de l'inventaire — l'estompage
 * de la rangée ne la décrit pas.
 */
export function ReplayAbilityCell({
  doc,
  slot,
  frame,
  readingFull,
  locale,
  gabarit,
}: {
  doc: ReplayDocumentReady
  slot: number
  frame: number
  readingFull: number
  locale: ReplayLocale
  /** Les cotes de la fiche : le côté de la vignette (`iconAbilityPx`). */
  gabarit: CardGabarit
}) {
  const t = REPLAY_TEXT[locale]
  const px = gabarit.iconAbilityPx
  const abilityRead = abilityAt(doc, slot, frame)
  const ability = abilityText(doc, abilityRead?.rank, t, locale)
  const charge =
    abilityRead !== null
      ? (abilityChargesAt(doc, slot, frame, abilityRead.rank) ?? carriedUsesAt(doc, slot, frame, abilityRead.rank))
      : null
  // LES CHARGES SE DISENT DANS L'INFOBULLE DE LA VIGNETTE.
  const chargeInTitle = charge ? chargeTitle(charge, doc, t) : null
  return (
    <span className="relative inline-flex shrink-0 items-center" style={{ width: px }}>
      {abilityRead ? (
        ability && (
          <span
            className="inline-flex items-center"
            style={{ opacity: freshness(abilityRead.age, readingFull, READING_FADE) }}
            title={
              chargeInTitle
                ? `${abilityAgeTitle(t, abilityRead.age, doc, ability.text)} · ${chargeInTitle}`
                : abilityAgeTitle(t, abilityRead.age, doc, ability.text)
            }
          >
            {ability.img ? (
              <WeaponIcon
                imageUrl={ability.img}
                tinted={ability.tinted}
                label={ability.text}
                width={px}
                height={px}
              />
            ) : ability.known ? (
              ability.text
            ) : (
              /* RANG NON RÉSOLU : un GLYPHE, pas un mot ni un caractère (planche du 16/08).
                 Le rang lu reste la seule chose vraie : il vit dans l'infobulle. */
              <AbilityUnknownMark label={ability.text} px={px} />
            )}
          </span>
        )
      ) : null /* AUCUNE LECTURE DANS LA VIE EN COURS : rien n'est rendu, ni glyphe ni texte — jamais
           la capacité d'une vie précédente du slot (correctif P0-2), jamais une lacune écrite. */}
    </span>
  )
}

/**
 * chargeTitle — ce que l'infobulle de la vignette dit des charges : le compte et l'âge de SA
 * lecture, ou d'où vient l'affirmation « plein » (rien transmis = rien consommé).
 */
function chargeTitle(charge: AbilityChargeDisplay, doc: ReplayDocumentReady, t: ReplayText): string {
  if (charge.kind === 'full') return t.abilityChargesFullHint
  return `${t.abilityChargesCount(charge.charges)} · ${t.abilityChargesAge} ${formatSeconds(frameToMs(charge.age, doc))}`
}

/**
 * StateMark — le socle d'un pictogramme muet : un dessin discret, UNE infobulle simple. Encre en
 * currentColor : la couleur vient du texte environnant, aucun littéral.
 */
function StateMark({ label, children }: { label: string; children: ReactNode }) {
  return (
    <span role="img" aria-label={label} title={label} className="inline-flex items-center opacity-70">
      {children}
    </span>
  )
}

/**
 * AbilityUnknownMark — capacité LUE mais NON IDENTIFIÉE : l'emplacement de la vignette,
 * vide, en pointillés. Le dessin dit exactement ce qu'on sait — « il y avait quelque chose
 * ici, on ne sait pas quoi ». Même gabarit que les vignettes de la ligne (`iconAbilityPx`).
 */
function AbilityUnknownMark({ label, px }: { label: string; px: number }) {
  return (
    <StateMark label={label}>
      <svg
        width={px}
        height={px}
        viewBox="0 0 16 16"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.2"
        strokeDasharray="2.4 1.8"
        aria-hidden="true"
      >
        <rect x="2.4" y="2.4" width="11.2" height="11.2" rx="2.4" />
      </svg>
    </StateMark>
  )
}

/**
 * abilityText nomme la capacité — et porte sa vignette de HUD quand le document en sert
 * une. Un index hors table garde son NUMÉRO dans le texte servi. Renvoie null quand rien
 * n'a été lu — l'absence de capacité et une capacité non identifiée sont deux états
 * différents.
 */
function abilityText(
  doc: ReplayDocumentReady,
  rank: number | undefined,
  t: ReplayText,
  locale: ReplayLocale,
): { text: string; known: boolean; img?: string; tinted?: boolean } | null {
  if (rank === undefined) return null
  const lbl: CatalogLabel | undefined = doc.abilityLabels?.[String(rank)]
  const name = catalogText(lbl, locale)
  if (name) return { text: name, known: true, img: lbl?.img, tinted: lbl?.tinted }
  return { text: t.abilityUnidentified(rank), known: false }
}

/**
 * abilityAgeTitle — l'infobulle de la capacité : son nom quand il est connu, et TOUJOURS
 * l'âge de sa lecture. Un âge négatif est une lecture À VENIR (début de vie), dit comme tel.
 */
function abilityAgeTitle(
  t: ReplayText,
  age: number,
  doc: ReplayDocumentReady,
  name: string | null,
): string {
  const ms = formatSeconds(frameToMs(Math.abs(age), doc))
  const when = age < 0 ? `${t.abilityAhead} ${ms}` : `${t.abilityAge} ${ms}`
  return name ? `${t.abilityLabel} — ${name} · ${when}` : when
}
