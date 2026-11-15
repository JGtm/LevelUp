/**
 * SquadAppuiCard — « Appui », la première carte de la section « Appui et portée » de Synergies.
 *
 * DIAGRAMME DE FLUX « ASSISTANT -> FINISSEUR » : à gauche les ASSISTANTS (« Assistant : prépare
 * le frag »), à droite les FINISSEURS (« Finisseur : conclut le frag »), un ruban par couple dont
 * la largeur est le nombre d'assistances. Chaque joueur apparaît des deux côtés ; le nœud porte
 * la couleur du joueur et le ruban celle de l'assistant. C'est la même forme que sur la page Match
 * (`MatchAssistChart`, via `features/_shared/assists/AssistFlowChart`) : à question identique,
 * forme identique.
 *
 * Les deux grandeurs de l'ancien tableau survivent dans l'infobulle d'un ruban : le COMPTE (sa
 * largeur) ET LA PART dans les assistances de l'escouade (sur le total serveur — « 3 assistances
 * sur 6 n'est pas 3 sur 300 »), plus les assistances où l'assistant a fait plus de dégâts que le
 * finisseur quand il y en a. Phrases à la voix active, sujet = l'assistant.
 *
 * COULEURS PAR JOUEUR (`colorByPlayer`, palette de la page `useSquadPlayerPalette`) : un joueur
 * garde la même teinte partout sur la page.
 *
 * STYLE DE BLOC CANONIQUE (`SectionCard`) ET LECTURE EN INFOBULLE ⓘ : la phrase qui dit ce
 * qu'on lit vit dans l'aide du titre — ni bandeau de couverture ni description en texte gris.
 *
 * TITRE « APPUI » : la notion se dit « appui » ; le mot « assistance » désigne la statistique du
 * jeu qu'elle compte (garde-rail `vocabulaireCoordination.guard.test.ts`).
 */
import { useMemo } from 'react'

import { AssistFlowChart } from '@/features/_shared/assists/AssistFlowChart'
import type { AssistFlowChartOpts } from '@/features/_shared/assists/assistFlowOption'
import { ASSIST_FLOW_TEXT } from '@/features/_shared/assists/assistFlowStrings'
import { resolveToken } from '@/lib/accessibility'
import { titleWithInfo } from '@/components/ui/title-with-info'
import { SectionCard } from '@/components/ui/section-card'
import { intlLocale } from '@/lib/formatters'
import type { SquadAssistPairs } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { getSquadText } from './i18n'
import { squadAssistFlow } from './squadAssistPairs.logic'

export interface SquadAppuiCardProps {
  block: SquadAssistPairs
  /** Roster dans l'ordre de la page : joueur principal d'abord, puis les coéquipiers. */
  roster: string[]
  /** gamertag → couleur hex, palette de la page (`useSquadPlayerPalette`). */
  colorByPlayer: Record<string, string>
}

export function SquadAppuiCard({ block, roster, colorByPlayer }: SquadAppuiCardProps) {
  const locale = useAppShellStore((s) => s.locale)
  const labels = getSquadText(locale).assists
  const flowText = ASSIST_FLOW_TEXT[locale]
  const numLoc = intlLocale(locale)

  // Une part s'écrit sans décimale inutile (« 25 % », « 12,5 % »), séparateurs de la langue.
  const pctFmt = useMemo(
    () => new Intl.NumberFormat(numLoc, { style: 'percent', minimumFractionDigits: 0, maximumFractionDigits: 1 }),
    [numLoc],
  )
  const flow = useMemo(() => squadAssistFlow(block, roster), [block, roster])

  // Un joueur hors palette (absent de la sélection) prend l'encre achromatique (zone-neutral), jamais une teinte empruntée.
  const chart = useMemo<Omit<AssistFlowChartOpts, 'width'>>(
    () => ({
      colorOf: (name) => colorByPlayer[name] ?? resolveToken('zone-neutral'),
      fmtShare: (v) => pctFmt.format(v),
      texts: { ...flowText, share: labels.tooltipShare },
    }),
    [colorByPlayer, pctFmt, flowText, labels],
  )

  return (
    <SectionCard title={labels.title} titleAdornment={titleWithInfo(labels.description)} className="h-full">
      {/* GRAPHE CENTRÉ EN HAUTEUR DANS SON BLOC : `h-full` sur la carte, corps
          `flex flex-1 flex-col justify-center`, AUCUNE hauteur minimale — dans une rangée
          plus haute que lui, le graphe garde sa taille et le vide se répartit au-dessus et
          en dessous au lieu de tomber entièrement en bas. */}
      <div className="flex flex-1 flex-col justify-center px-3 py-2">
        <AssistFlowChart
          flow={flow}
          chart={chart}
          headers={{ assistant: flowText.assistantHeader, finisher: flowText.finisherHeader }}
          emptyMessage={labels.noPairs}
          testId="squad-appui"
        />
      </div>
    </SectionCard>
  )
}
