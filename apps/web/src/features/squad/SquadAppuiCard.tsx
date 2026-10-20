/**
 * SquadAppuiCard — « Appui », la première carte de la section « Appui et portée » de Synergies.
 *
 * BARRES HORIZONTALES EMPILÉES, une par ASSISTANT — le LARBIN —, segments = les
 * BÉNÉFICIAIRES qu'il a servis — les PATRONS (vocabulaire de l'écran, décision utilisateur
 * du 2026-09-17 : titre d'axe, infobulle et description nomment les deux rôles). C'est la
 * forme déjà retenue pour le même fait sur la page Match (`MatchAssistChart`) : à question
 * identique, forme identique — un tableau ici et un graphe là obligeaient à réapprendre la
 * lecture d'un écran à l'autre.
 *
 * Remplace `SquadAssistPairsTable` (retiré le 2026-09-13, décision utilisateur). Les deux
 * grandeurs du tableau survivent, dans l'infobulle : le COMPTE (hauteur du segment) ET LA
 * PART dans les assistances mesurées de l'escouade — « 3 assistances sur 6 n'est pas 3 sur
 * 300 » —, plus les éliminations volées quand il y en a.
 *
 * COULEURS PAR JOUEUR (`colorByPlayer`, palette de la page `useSquadPlayerPalette`) : un joueur
 * garde la même teinte partout sur la page. Un segment désigne donc le bénéficiaire par sa couleur, sans lecture de légende.
 *
 * STYLE DE BLOC CANONIQUE (`SectionCard`) ET LECTURE EN INFOBULLE ⓘ (2026-09-19) : le bloc
 * se rend comme les autres de la page, et la phrase qui dit ce qu'on lit vit dans l'aide du
 * titre — plus de bandeau de couverture ni de description en texte gris.
 *
 * TITRE « APPUI » : la notion se dit « appui » ; le mot « assistance » désigne la
 * statistique du jeu qu'elle compte (garde-rail `vocabulaireCoordination.guard.test.ts`).
 *
 * PLUS DE PHRASE DE LECTEUR (D22-verbosité, LOI du 2026-09-21) : graphe et légende
 * seulement, l'explication tient dans l'infobulle (i) du titre.
 */
import { useMemo } from 'react'

import { BarStackedChart } from '@/components/charts/BarStackedChart'
import { titleWithInfo } from '@/components/ui/title-with-info'
import { SectionCard } from '@/components/ui/section-card'
import { intlLocale } from '@/lib/formatters'
import type { SquadAssistPairs } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { getSquadText } from './i18n'
import {
  assistBeneficiaires,
  assistCle,
  assistPairsSeries,
  assistPartParCouple,
  assistVoleesParCouple,
} from './squadAssistPairs.logic'

export interface SquadAppuiCardProps {
  block: SquadAssistPairs
  /** Roster dans l'ordre de la page : joueur principal d'abord, puis les coéquipiers. */
  roster: string[]
  /** gamertag → couleur hex, palette de la page (`useSquadPlayerPalette`). */
  colorByPlayer: Record<string, string>
}

export function SquadAppuiCard({ block, roster, colorByPlayer }: SquadAppuiCardProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadText(locale)
  const labels = t.assists
  const numLoc = intlLocale(locale)

  // `pairs` est un tableau NULLABLE au contrat (toute tranche Go sort ainsi) : comblé ici,
  // à la frontière, une seule fois.
  const pairs = useMemo(() => block.pairs ?? [], [block.pairs])

  const pctFmt = useMemo(
    () =>
      new Intl.NumberFormat(numLoc, {
        style: 'percent',
        minimumFractionDigits: 1,
        maximumFractionDigits: 1,
      }),
    [numLoc],
  )

  const series = useMemo(() => assistPairsSeries(pairs, roster), [pairs, roster])
  const beneficiaires = useMemo(() => assistBeneficiaires(pairs, roster), [pairs, roster])
  const volees = useMemo(() => assistVoleesParCouple(pairs), [pairs])
  const parts = useMemo(() => assistPartParCouple(block), [block])

  // Couleurs par joueur : celles de la page, restreintes aux bénéficiaires (les segments).
  const componentHexColors = useMemo(() => {
    const out: Record<string, string> = {}
    for (const gt of beneficiaires) {
      const couleur = colorByPlayer[gt]
      if (couleur) out[gt] = couleur
    }
    return out
  }, [colorByPlayer, beneficiaires])

  // Les deux rôles du graphe, nommés (décision utilisateur 2026-09-17) : la barre est le
  // LARBIN (il a assisté), le segment le PATRON (il a eu le frag crédité). Les titres
  // d'axes les portent (catégories = larbin, valeurs = assistances par patron), l'infobulle
  // aussi.
  const tooltipRoles = useMemo(
    () => ({ category: labels.roleAssistant, component: labels.roleBeneficiary }),
    [labels],
  )

  const tooltipComponentNote = useMemo(
    () => (assistant: string, beneficiaire: string) => {
      const cle = assistCle(assistant, beneficiaire)
      const notes: string[] = []
      const part = parts.get(cle)
      if (part != null) notes.push(labels.tooltipShare(pctFmt.format(part)))
      const volee = volees.get(cle)
      if (volee) notes.push(labels.tooltipAssistantOutdamaged(volee))
      return notes.length > 0 ? notes.join(' · ') : undefined
    },
    [parts, volees, labels, pctFmt],
  )

  return (
    <SectionCard
      title={labels.title}
      titleAdornment={titleWithInfo(labels.description)}
      className="h-full"
    >
      {/* GRAPHE CENTRÉ EN HAUTEUR DANS SON BLOC : `h-full` sur la carte, corps
          `flex flex-1 flex-col justify-center`, AUCUNE hauteur minimale — dans une rangée
          plus haute que lui, le graphe garde sa taille et le vide se répartit au-dessus et
          en dessous au lieu de tomber entièrement en bas. */}
      <div
        className="flex flex-1 flex-col justify-center px-3 py-2"
        data-testid="squad-appui"
      >
        <BarStackedChart
          series={series}
          height={320}
          orientation="horizontal"
          emptyMessage={labels.noPairs}
          componentOrder={beneficiaires}
          componentHexColors={componentHexColors}
          tooltipHideZero
          tooltipComponentNote={tooltipComponentNote}
          categoryAxisName={labels.roleAssistant}
          valueAxisName={labels.valueAxis}
          tooltipRoles={tooltipRoles}
          frameless
        />
      </div>
    </SectionCard>
  )
}
