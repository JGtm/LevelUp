/**
 * FormesRetenuesSection.tsx — LA SECTION « Les formes retenues » (artefact 2ec1b8eb) :
 * trois blocs, et les cartes DU SEUL CONTEXTE DEMANDÉ.
 *
 * UNE PAGE = UN CONTEXTE (2026-09-19, PLAN_AJUSTEMENTS_PRE_V75 item 1.E). Les deux
 * contextes cohabitaient sur l'Escouade, séparés par deux intertitres qui les nommaient :
 * neuf cartes y parlaient de MOI sur une page qui parle de NOUS.
 * Le contexte solo vit désormais sur Timeseries (onglet Progression), l'escouade sur la
 * page Escouade — même bloc de contrat, même modèle de vue, même composant.
 *
 * L'ORDRE DES BLOCS EST CELUI DE L'ARTEFACT : équipement, armes spéciales, objectifs.
 * Chaque titre de bloc porte son AIDE ⓘ — ce que le bloc mesure et ce qu'il ne mesure pas,
 * trois phrases au plus. Les pavés de constat et de lexique qui les précédaient ont été
 * résumés là (décision 9) : un lexique se lit une fois, puis n'est qu'un mur.
 *
 * UNE SEULE COLONNE : chaque forme a besoin de toute la largeur (une bande de vingt
 * matchs, une grille de cinq colonnes graduées). Deux cartes côte à côte les rendraient
 * illisibles.
 *
 * LE BLOC SE RETIRE DE LUI-MÊME quand la réponse ne le porte pas (titre sans film), et
 * affiche « 0 sur N » quand aucun match du scope n'est mesuré : une couverture nulle est un
 * état légitime, pas une page vide.
 */
import { useMemo } from 'react'

import { SectionTitle } from '@/components/ui/detail-section'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import type { SquadFormesBlock } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { FORMES_CARDS_TEXT } from './cardsI18n'
import {
  EquipmentByMatchCard,
  EquipmentLobbyTrackCard,
  EquipmentRegularityCard,
  EquipmentSharesCard,
  EquipmentSpreadCard,
  EquipmentSquadGridCard,
  EquipmentSquadTrackCard,
} from './cards/EquipmentCards'
import {
  ObjectivesGapRoleCard,
  ObjectivesGapSquadCard,
  ObjectivesLobbyTrackCard,
  ObjectivesRawGridCard,
  ObjectivesSharesByFamilyCard,
} from './cards/ObjectiveCards'
import {
  PadsGapSoloCard,
  PadsGapSquadCard,
  PadsShareSoloCard,
  PadsSquadByMatchCard,
  PadsSquadWeaponGridCard,
  PadsTwoFriezesCard,
  PadsWeaponGridCard,
} from './cards/PadCards'
import { measuredPlayersCount } from './cards/shared'
import { FORMES_TEXT } from './i18n'
import { average, matchSizes, parityOf } from './model/access'
import { objectiveFamilies, objectiveMatches } from './model/objectives'
import { buildFormesViewModel, type FormesViewModel } from './viewModel'

/** Le contexte de lecture : « moi dans mon équipe et dans le lobby », ou « mon camp
 *  contre le leur ». Une page n'en montre qu'un. */
export type FormesContexte = 'solo' | 'squad'

export interface FormesRetenuesSectionProps {
  /** Le bloc `formes_retenues` de la réponse — absent : rien ne se rend. */
  block: SquadFormesBlock | null | undefined
  locale: Locale
  /** Le contexte des cartes montées. */
  contexte: FormesContexte
  /** Le nom du joueur de la page (`main_player` de la réponse) — voir viewModel. */
  mainPlayerLabel?: string
}

/** Le titre d'un bloc, avec l'aide ⓘ qui dit ce qu'il mesure. */
function BlockTitle({ children, aide }: { children: string; aide: string }) {
  return (
    <SectionTitle className="mt-10 flex items-center gap-1.5 border-b border-border pb-2">
      {children}
      <InfoTooltip content={aide} />
    </SectionTitle>
  )
}

/** Une tuile du bandeau : une clé, une valeur, une précision. */
function HeaderTile({ label, value, hint }: { label: string; value: string; hint: string }) {
  return (
    <div className="bg-card px-2.5 py-2">
      <div className="text-3xs uppercase tracking-wider text-muted-foreground">{label}</div>
      <div className="text-sm font-semibold tabular-nums text-foreground">{value}</div>
      <div className="text-3xs text-muted-foreground">{hint}</div>
    </div>
  )
}

/** Les quatre repères de couverture, calculés sur la période affichée. */
function HeaderStrip({ vm }: { vm: FormesViewModel }) {
  const { t, block } = vm
  const teamParity = parityOf(average(vm.measured.map((m) => matchSizes(m).team)) ?? 0)
  const lobbyParity = parityOf(average(vm.measured.map((m) => matchSizes(m).lobby)) ?? 0)
  const families = objectiveFamilies(block).length
  return (
    <div className="mt-4 grid gap-px border border-border bg-border [grid-template-columns:repeat(auto-fit,minmax(150px,1fr))]">
      <HeaderTile
        label={t.header.scope}
        value={t.header.matchesFmt(block.matches_total)}
        hint={t.header.scopeMeasuredFmt(block.matches_measured, block.matches_total)}
      />
      <HeaderTile
        label={t.header.lobbies}
        value={String(measuredPlayersCount(vm))}
        hint={t.header.lobbiesPlacesFmt(measuredPlayersCount(vm), vm.squad.length)}
      />
      <HeaderTile
        label={t.header.parity}
        value={`${vm.fmtPct(teamParity ?? 0)} / ${vm.fmtPct(lobbyParity ?? 0)}`}
        hint={t.header.parityHint}
      />
      <HeaderTile
        label={t.header.modes}
        value={t.header.familiesFmt(families)}
        hint={t.header.modesHint}
      />
    </div>
  )
}

export function FormesRetenuesSection({
  block,
  locale,
  contexte,
  mainPlayerLabel,
}: FormesRetenuesSectionProps) {
  const t = FORMES_TEXT[locale]
  const ct = FORMES_CARDS_TEXT[locale]
  const vm = useMemo(
    () =>
      block != null && block.available
        ? buildFormesViewModel(block, t, ct, locale, mainPlayerLabel)
        : null,
    [block, t, ct, locale, mainPlayerLabel],
  )
  // Bloc absent du contrat (scope vide) ou indisponible (titre sans film) : la
  // section se retire, elle n'affiche pas une coquille vide.
  if (vm == null) return null

  const solo = contexte === 'solo'
  const hasObjectives = objectiveMatches(vm.block).length > 0

  return (
    <section className="space-y-3" aria-label={t.sectionTitle}>
      <SectionTitle>{t.sectionTitle}</SectionTitle>
      <HeaderStrip vm={vm} />

      <BlockTitle aide={t.blocks.equipment.aide}>{t.blocks.equipment.title}</BlockTitle>
      <div className="space-y-4">
        {solo ? (
          <>
            <EquipmentSharesCard vm={vm} />
            <EquipmentByMatchCard vm={vm} />
            <EquipmentSpreadCard vm={vm} />
          </>
        ) : (
          <>
            <EquipmentRegularityCard vm={vm} />
            <EquipmentLobbyTrackCard vm={vm} />
            <EquipmentSquadGridCard vm={vm} />
            <EquipmentSquadTrackCard vm={vm} />
          </>
        )}
      </div>

      <BlockTitle aide={t.blocks.weapons.aide}>{t.blocks.weapons.title}</BlockTitle>
      <div className="space-y-4">
        {solo ? (
          <>
            <PadsGapSoloCard vm={vm} />
            <PadsShareSoloCard vm={vm} />
            <PadsWeaponGridCard vm={vm} />
          </>
        ) : (
          <>
            <PadsGapSquadCard vm={vm} />
            <PadsTwoFriezesCard vm={vm} />
            <PadsSquadByMatchCard vm={vm} />
            <PadsSquadWeaponGridCard vm={vm} />
          </>
        )}
      </div>

      <BlockTitle aide={t.blocks.objectives.aide}>{t.blocks.objectives.title}</BlockTitle>
      {hasObjectives && (
        <div className="space-y-4">
          {solo ? (
            <>
              <ObjectivesGapRoleCard vm={vm} />
              <ObjectivesSharesByFamilyCard vm={vm} />
              <ObjectivesRawGridCard vm={vm} />
            </>
          ) : (
            <>
              <ObjectivesGapSquadCard vm={vm} />
              <ObjectivesLobbyTrackCard vm={vm} />
            </>
          )}
        </div>
      )}
    </section>
  )
}
