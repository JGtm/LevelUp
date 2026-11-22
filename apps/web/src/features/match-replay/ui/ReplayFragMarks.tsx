/**
 * ReplayFragMarks — LES PICTOGRAMMES DE TIR À LA TÊTE ET DE FRAG PARFAIT d'une ligne d'élimination, en tête de la
 * dernière colonne, avant les médailles.
 *
 * LA SOURCE est l'élimination du tableau des scores que la ligne désigne (`model/eliminationMarks`) : les marques que
 * le jeu écrit lui-même dans ses compteurs, les mêmes que les totaux de la fiche du joueur. Seule une marque VRAIE a
 * une icône ; une marque fausse ou inconnue n'écrit rien, ni zéro ni « non mesuré ».
 *
 * LES VIGNETTES sont celles du kill feed du jeu (atlas `killfeed`, index 64 « headshot » et 83 « perfect »), en masque
 * teint comme l'icône d'assistance (`AssistMark`) : l'encre est celle du repère neutre, aucune couleur d'équipe ne
 * dit un sous-type de frag. Le garde-rail `fragMarksIcon.guard.test.ts` tient les deux fichiers et leur index.
 */
import { WeaponIcon } from '@/components/ui/WeaponIcon'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import { staticAssetURL } from '@/lib/staticAssets'
import { useTitleSlug } from '@/lib/title-routing/useTitleSlug'

import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { COTES_LIGNE_FIL } from '../model/cotesFeedLine'
import type { FragMarks } from '../model/eliminationMarks'

/** Stems (sans dossier ni extension) des vignettes du kill feed du jeu : tir à la tête et frag parfait. */
export const HEADSHOT_ICON_STEM = 'killfeed-64'
export const PERFECT_ICON_STEM = 'killfeed-83'

const PICTO_PX = COTES_LIGNE_FIL.medailles.pictoPx

/**
 * FragMarkIcons rend une icône par marque vraie, dans l'ordre tir à la tête puis frag parfait. `perfect: false`
 * retire l'icône du frag parfait (la ligne porte déjà la médaille « Parfait », qui le dit).
 */
export function FragMarkIcons({
  marks,
  perfect,
  locale,
}: {
  marks: FragMarks | undefined
  perfect: boolean
  locale: ReplayLocale
}) {
  const titleSlug = useTitleSlug()
  const t = REPLAY_TEXT[locale].feedLines
  const shown: { stem: string; label: string }[] = []
  if (marks?.headshot) shown.push({ stem: HEADSHOT_ICON_STEM, label: t.headshotMark })
  if (marks?.perfect && perfect) shown.push({ stem: PERFECT_ICON_STEM, label: t.perfectMark })
  return (
    <>
      {shown.map((m) => (
        <span key={m.stem} data-frag-mark={m.stem} title={m.label} className="inline-flex shrink-0 items-center">
          <WeaponIcon
            imageUrl={staticAssetURL('weapon', `jeu/${m.stem}`, '.png', titleSlug)}
            tinted
            label={m.label}
            width={PICTO_PX}
            height={PICTO_PX}
            style={{ color: tokenCssVar('divergent-neutral') }}
          />
        </span>
      ))}
    </>
  )
}
