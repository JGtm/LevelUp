/**
 * ReplayAbilityCell.tsx — la CELLULE DE CAPACITÉ de la rangée d'inventaire : la vignette de
 * l'équipement porté, et — depuis le lot P6 — les CHARGES qui lui restent.
 *
 * EXTRAITE DE `ReplayInventoryRow.tsx` LE 2026-09-04, et l'extraction PAIE l'addition (la
 * règle du cliquet du canvas, appliquée à la rangée) : le fichier était à 496 lignes pour un
 * plafond de 500, et le lot P6 devait y brancher l'affichage des charges. La cellule part
 * donc entière — vignette, glyphe de rang non résolu, infobulle d'âge — et les charges
 * naissent ici, jamais une ligne de logique au composant de rangée.
 *
 * CE QUE LA CELLULE AFFICHE DES CHARGES, et pourquoi (décisions P6.1, arbitrage utilisateur
 * du 04/09) :
 *  - un COMPTE discret à côté de la vignette (même convention « ×N » que les grenades) quand
 *    une lecture du canal i56 couvre la vie et l'équipement courants ;
 *  - « PLEIN » QUALITATIF, sans chiffre, avant la première lecture : le film ne transmet
 *    RIEN au ramassage, un maximum déduit serait inventé ;
 *  - RIEN pour une famille que le canal ne mesure pas (camouflage, surbouclier,
 *    répulseur…) : ni chiffre, ni « plein » — rien d'affirmé.
 * Le tri entre ces trois états vit dans `abilityChargeLogic.ts` (pur, testé sans rendu).
 *
 * LA CELLULE GARDE SA LARGEUR FIXE (la grille des fiches en dépend) : le compte et le
 * « plein » se posent APRÈS elle, dans l'espace souple de la rangée — comme le badge d'état
 * vide, ils ne décalent aucune colonne. Chaque marque pâlit avec l'âge de SA lecture (la
 * doctrine de la rangée) ; « plein » n'a pas de lecture, il ne pâlit pas.
 */
import type { ReactNode } from 'react'

import { WeaponIcon } from '@/components/ui/WeaponIcon'

import { abilityChargesAt, type AbilityChargeDisplay } from '../abilityChargeLogic'
import { catalogText, type CatalogLabel } from '../i18n/catalogLabel'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { formatSeconds, frameToMs, freshness, READING_FADE } from '../replayLogic'
import type { ReplayDocumentReady } from '../replayNormalize'
import { abilityAt } from '../rosterLogic'

/** Boîte de la vignette de CAPACITÉ : la hauteur de la ligne. */
const HUD_ICON_PX = 16

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
}: {
  doc: ReplayDocumentReady
  slot: number
  frame: number
  readingFull: number
  locale: ReplayLocale
}) {
  const t = REPLAY_TEXT[locale]
  const abilityRead = abilityAt(doc, slot, frame)
  const ability = abilityText(doc, abilityRead?.rank, t, locale)
  const charge =
    abilityRead !== null ? abilityChargesAt(doc, slot, frame, abilityRead.rank) : null
  return (
    <>
      <span className="inline-flex shrink-0 items-center" style={{ width: HUD_ICON_PX }}>
        {ability && abilityRead && (
          <span
            className="inline-flex items-center"
            style={{ opacity: freshness(abilityRead.age, readingFull, READING_FADE) }}
            title={abilityAgeTitle(t, abilityRead.age, doc, ability.text)}
          >
            {ability.img ? (
              <WeaponIcon
                imageUrl={ability.img}
                tinted={ability.tinted}
                label={ability.text}
                width={HUD_ICON_PX}
                height={HUD_ICON_PX}
              />
            ) : ability.known ? (
              ability.text
            ) : (
              /* RANG NON RÉSOLU : un GLYPHE, pas un mot ni un caractère (planche du 16/08).
                 Le rang lu reste la seule chose vraie : il vit dans l'infobulle. */
              <AbilityUnknownMark label={ability.text} />
            )}
          </span>
        )}
      </span>
      {ability && abilityRead && charge && (
        <AbilityChargeMark charge={charge} doc={doc} readingFull={readingFull} t={t} />
      )}
    </>
  )
}

/**
 * AbilityChargeMark — le compte de charges, ou « plein » qualitatif.
 *
 * Le compte pâlit avec l'âge de SA lecture (comme toute cellule de la rangée) et son
 * infobulle date la lecture. « Plein » n'est pas une lecture : un libellé discret, et
 * l'infobulle dit d'où vient l'affirmation (rien transmis = rien consommé).
 */
function AbilityChargeMark({
  charge,
  doc,
  readingFull,
  t,
}: {
  charge: AbilityChargeDisplay
  doc: ReplayDocumentReady
  readingFull: number
  t: ReplayText
}) {
  if (charge.kind === 'full') {
    return (
      <span className="opacity-70" title={t.abilityChargesFullHint}>
        {t.abilityChargesFull}
      </span>
    )
  }
  return (
    <span
      className="tabular-nums text-foreground"
      style={{ opacity: freshness(charge.age, readingFull, READING_FADE) }}
      title={`${t.abilityChargesCount(charge.charges)} · ${t.abilityChargesAge} ${formatSeconds(frameToMs(charge.age, doc))}`}
    >
      ×{charge.charges}
    </span>
  )
}

/**
 * StateMark — le socle des pictogrammes d'état MESURÉ rendus muets (décision produit 4 du
 * plan parité) : « munitions pleines » (emplacement jamais écrit — flux différentiel, le
 * plein est la valeur par défaut). Un dessin discret, UNE infobulle simple. Encre en
 * currentColor : la couleur vient du texte environnant, aucun littéral.
 */
export function StateMark({ label, children }: { label: string; children: ReactNode }) {
  return (
    <span role="img" aria-label={label} title={label} className="inline-flex items-center opacity-70">
      {children}
    </span>
  )
}

/**
 * AbilityUnknownMark — capacité LUE mais NON IDENTIFIÉE : l'emplacement de la vignette,
 * vide, en pointillés. Le dessin dit exactement ce qu'on sait — « il y avait quelque chose
 * ici, on ne sait pas quoi ». Même gabarit que les vignettes de la ligne (16 px).
 */
function AbilityUnknownMark({ label }: { label: string }) {
  return (
    <StateMark label={label}>
      <svg
        width={HUD_ICON_PX}
        height={HUD_ICON_PX}
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
