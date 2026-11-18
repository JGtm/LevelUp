/**
 * ReplayInventoryRow — la grenade sélectionnée et la capacité d'armure d'une fiche joueur.
 *
 * LA RANGÉE EST UNE GRILLE À CELLULES FIXES (même règle que la cellule d'arme) : une cellule de
 * GRENADE, une cellule de CAPACITÉ — toujours rendues, toujours aux mêmes largeurs. Une donnée
 * absente laisse sa cellule vide plutôt que de décaler les colonnes des fiches voisines.
 *
 * TOUT CE QUI N'EST PAS LU S'AFFICHE COMME LACUNE, jamais comme une valeur par défaut : une
 * capacité hors table reçoit un GLYPHE NEUTRE (vignette vide en pointillés) et garde son NUMÉRO
 * en infobulle.
 *
 * LA CELLULE DE GRENADE ne rend que la grenade SÉLECTIONNÉE (vignette de `iconGrenadePx`) ; le
 * stock des types portés, la provenance de la sélection et « grenade en main : inconnue » passent
 * dans son infobulle (`grenadeBoxHint`). Aucune marque n'est écrite en texte sur la rangée : les
 * munitions de la main, « dégainée ? » et l'état vide (« Mort » / « Inventaire indisponible »)
 * se disent dans l'infobulle de l'arme (`model/handCellHint.ts`), les charges dans celle de la
 * capacité.
 *
 * CHAQUE CELLULE PÂLIT AVEC L'ÂGE DE LA LECTURE QUI LA DÉCRIT, et non avec celui de la rangée :
 * les grenades viennent de l'axe `grenadeReads` (deux canaux, dont les paquets delta), la capacité
 * du canal i48. Un estompage unique posé sur le conteneur multiplierait l'opacité propre de
 * chaque cellule par celle de l'inventaire et rendrait toute cellule plus fraîche PLUS PÂLE que
 * la rangée — l'inverse de ce qu'un estompage dit.
 *
 * LA CELLULE DE CAPACITÉ VIT DANS `ReplayAbilityCell.tsx` : l'affichage des CHARGES restantes
 * s'y branche (compte lu, « plein » qualitatif, ou rien — cf. `abilityChargeLogic.ts`).
 */
import { WeaponIcon } from '@/components/ui/WeaponIcon'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'

import type { CardGabarit } from '../model/cardGabarit'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import {
  grenadeBoxAt,
  grenadeBoxHint,
  grenadesCarriedFrom,
  type GrenadeBox,
  type GrenadeSelection,
  inventoryAt,
  selectedGrenadeFrom,
} from '../model/inventoryReading'
import { ReplayAbilityCell } from './ReplayAbilityCell'
import { grenadeMaskOf } from './grenadeMask'
import { formatSeconds, frameToMs, freshness, READING_FADE } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'

// LA VIGNETTE DE GRENADE (`iconGrenadePx`) VIENT DU GABARIT de la fiche (`model/cardGabarit.ts`) :
// la grille des fiches en dépend, aucune cote locale ici.

type ReplayText = (typeof REPLAY_TEXT)[ReplayLocale]

export function ReplayInventoryRow({
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
  /** Les cotes de la fiche : côté de la vignette de grenade et de capacité. */
  gabarit: CardGabarit
}) {
  const t = REPLAY_TEXT[locale]
  const read = inventoryAt(doc, slot, frame)
  // LES GRENADES ONT LEUR PROPRE LECTURE, et donc leur propre age (schema 20, lot 4.4 du suivi
  // delta). L axe `grenadeReads` porte DEUX canaux sur la meme grandeur : les images-cles
  // (~20 s) et les paquets delta, transmis AU CHANGEMENT — donc la ou l etat bouge. La plus
  // recente gagne, et l age median de ce qui s affiche tombe de 10,00 s a 8,09 s.
  //
  // LE REPLI EST LE POINT : un artefact anterieur au schema 20 ne porte pas cet axe, et la
  // boite retombe alors sur la lecture d inventaire — exactement l affichage d avant. Un
  // artefact ancien ne se vide pas, il est seulement moins frais.
  //
  // LE DEPARTAGE VIT DANS `grenadeBoxAt`, PAS ICI : une lecture A VENIR ne prime jamais une
  // information passee (meme doctrine que la « lecture vide A VENIR »), et le composant ne
  // fait qu afficher ce que la fonction pure a tranche.
  const box = grenadeBoxAt(doc, slot, frame, read)
  const grenades = box ? grenadesCarriedFrom(box.g, doc.grenadeLabels, locale) : []
  const selected = box ? selectedGrenadeFrom(box.g, box.gs) : null
  // Âge NÉGATIF = lecture d'une image-clé À VENIR (début de vie, cf. inventoryAt) :
  // l'infobulle le dit — l'estompage porte déjà sur la valeur absolue.
  const ageMs = read ? formatSeconds(frameToMs(Math.abs(read.age), doc)) : ''
  const ageTitle = !read
    ? undefined
    : read.age < 0
      ? `${t.inventoryAhead} ${ageMs}`
      : `${t.inventoryAge} ${ageMs}`

  return (
    <div
      // `min-w-0` N'EST PAS DÉCORATIF : sans lui, cette rangée est un élément flex à
      // `min-width: auto`, elle prend sa largeur hypothétique et déborde la rangée parente,
      // qui la CLIPPE.
      className="flex min-w-0 items-center gap-[5px] font-mono text-[9.5px] text-muted-foreground"
      // LA RANGÉE N'IMPOSE PLUS SON ÂGE À TOUTE LA GRILLE. L'estompage vivait ici tant que
      // l'inventaire était la seule lecture de la ligne ; il ne l'est plus (capacité par i48,
      // grenades par l'axe `grenadeReads`). Laissé au conteneur, il MULTIPLIAIT l'opacité
      // propre de chaque cellule par celle de l'inventaire : une lecture de grenades de 8,1 s
      // s'affichait plus PÂLE qu'avant le lot, et le gain de fraîcheur devenait invisible.
      // Chaque cellule porte donc l'âge de la lecture qui la décrit, et rien d'autre.
      title={ageTitle}
    >
      <GrenadeSelectedCell
        box={box}
        grenades={grenades}
        selected={selected}
        doc={doc}
        readingFull={readingFull}
        iconPx={gabarit.iconGrenadePx}
        t={t}
      />
      {/* LA CELLULE DE CAPACITÉ vit dans son propre fichier depuis le lot P6 (charges) :
          vignette à largeur fixe, puis compte de charges ou « plein » dans l'espace souple —
          elle garde son âge propre, sa lecture ne tombant pas sur les images-clés. */}
      <ReplayAbilityCell
        doc={doc}
        slot={slot}
        frame={frame}
        readingFull={readingFull}
        locale={locale}
        gabarit={gabarit}
      />
    </div>
  )
}

/**
 * GrenadeSelectedCell — la cellule FIXE de grenade : la vignette du SEUL type qui
 * partira au prochain lancer, à l'encre `warning` du type équipé, sans compteur ni anneau (la
 * place). Tout le reste passe dans l'infobulle : la provenance de la sélection (LUE dans le
 * film, ou DÉDUITE d'un seul type porté — deux affirmations qui ne se confondent pas), le
 * STOCK des types portés (`grenadeBoxHint`, réemployé tel quel) et son âge, et « grenade en main :
 * inconnue » quand plusieurs types sont portés sans sélecteur lu — la cellule reste alors VIDE, on ne
 * devine pas laquelle partira. Sans lecture : vide, sans infobulle. Elle pâlit avec l'âge de la
 * lecture de grenades.
 */
function GrenadeSelectedCell({
  box,
  grenades,
  selected,
  doc,
  readingFull,
  iconPx,
  t,
}: {
  box: GrenadeBox | null
  grenades: readonly { rank: number; name: string; count: number }[]
  selected: GrenadeSelection
  doc: ReplayDocumentReady
  readingFull: number
  iconPx: number
  t: ReplayText
}) {
  const sel = typeof selected === 'object' && selected !== null ? selected : null
  const chosen = sel ? (grenades.find((g) => g.rank === sel.rank) ?? null) : null
  const title = !box
    ? undefined
    : [
        chosen && sel ? `${chosen.name} — ${sel.read ? t.grenadeSelectedRead : t.grenadeSelected}` : null,
        selected === 'indeterminate' ? t.grenadeSelUnknown : null,
        grenadeBoxHint(t, box, grenades, doc),
      ]
        .filter((p): p is string => !!p)
        .join(' · ')
  const icon = chosen ? grenadeMaskOf(doc.grenadeLabels?.[chosen.rank]) : null
  return (
    <span
      className="inline-flex shrink-0 items-center overflow-hidden"
      style={{
        width: iconPx,
        opacity: box ? freshness(box.age, readingFull, READING_FADE) : 1,
        color: chosen ? tokenCssVar('warning') : undefined,
      }}
      title={title}
    >
      {chosen &&
        (icon ? (
          <WeaponIcon imageUrl={icon.url} tinted={icon.tinted} label={chosen.name} width={iconPx} height={iconPx} />
        ) : (
          <span className="truncate">{chosen.name}</span>
        ))}
    </span>
  )
}
