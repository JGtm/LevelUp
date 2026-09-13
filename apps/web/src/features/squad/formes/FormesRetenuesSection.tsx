/**
 * FormesRetenuesSection.tsx — LA SECTION ENTIÈRE de l'artefact 2ec1b8eb sur
 * l'onglet Synergies : trois blocs, deux contextes par bloc, dix-neuf cartes.
 *
 * L'ORDRE EST CELUI DE L'ARTEFACT, et il porte du sens : chaque bloc s'ouvre sur
 * son CONSTAT (ce que la période dit), puis son LEXIQUE (ce que les mots
 * désignent et ce que la mesure ne couvre pas), puis les cartes — d'abord
 * « moi dans mon équipe et dans le lobby », ensuite « mon camp contre le leur ».
 * Les deux contextes ne répondent pas à la même question et ne se mélangent pas.
 *
 * UNE SEULE COLONNE : chaque forme a besoin de toute la largeur (une bande de
 * vingt matchs, une grille de cinq colonnes graduées). Deux cartes côte à côte
 * les rendraient illisibles.
 *
 * LE BLOC SE RETIRE DE LUI-MÊME quand la réponse ne le porte pas (titre sans
 * film), et affiche « 0 sur N » quand aucun match du scope n'est mesuré : une
 * couverture nulle est un état légitime, pas une page vide.
 */
import { useMemo } from 'react'

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
import { RichText } from './forms/RichText'
import { FORMES_TEXT } from './i18n'
import { average, matchSizes, parityOf } from './model/access'
import { aggregateAxis } from './model/aggregates'
import { OBJECTIVE_ROLES, aggregateRole, objectiveFamilies, objectiveMatches } from './model/objectives'
import { WEAPON_CLASSES, aggregateWeaponClass, namedPickups } from './model/pads'
import { buildFormesViewModel, type FormesViewModel } from './viewModel'

export interface FormesRetenuesSectionProps {
  /** Le bloc `formes_retenues` de la réponse — absent : rien ne se rend. */
  block: SquadFormesBlock | null | undefined
  locale: Locale
}

/** Un intertitre de contexte : un petit titre suivi d'un filet. */
function ContextTitle({ children }: { children: string }) {
  return (
    <h4 className="mt-7 flex items-center gap-2.5 text-3xs font-bold uppercase tracking-widest text-foreground">
      {children}
      <span className="h-px flex-1 bg-border" aria-hidden="true" />
    </h4>
  )
}

/** Le constat d'un bloc : un paragraphe, avec ses passages en gras. */
function Constat({ text }: { text: string }) {
  return (
    <p className="mt-2.5 max-w-[74ch] text-xs text-muted-foreground">
      <RichText text={text} />
    </p>
  )
}

/** Le lexique d'un bloc : l'encadré bordé qui dit les mots et les réserves. */
function Lexique({ text }: { text: string }) {
  return (
    <div className="mt-3 max-w-[74ch] border-l-[3px] border-warning bg-card px-3 py-2 text-3xs text-muted-foreground">
      <RichText text={text} />
    </div>
  )
}

/** Le titre d'un bloc. */
function BlockTitle({ children }: { children: string }) {
  return (
    <h3 className="mt-10 border-b border-border pb-2 text-base font-semibold text-foreground">
      {children}
    </h3>
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

/** Le constat du bloc 2, alimenté par les mesures de la période. */
function weaponsConstat(vm: FormesViewModel): string {
  if (namedPickups(vm.block) === 0) return vm.ct.weaponsConstatEmpty
  const all = aggregateAxis(vm.block, 'pad_pickups')
  const heavy = aggregateWeaponClass(vm.block, vm.weapons, WEAPON_CLASSES[0])
  const precision = aggregateWeaponClass(vm.block, vm.weapons, WEAPON_CLASSES[1])
  return vm.ct.weaponsConstatFmt({
    teamShare: vm.fmtPct(all.teamShareOfLobbyPct ?? 0),
    heavyTeamShare: vm.fmtPct(heavy.teamShareOfLobbyPct ?? 0),
    precisionTeamShare: vm.fmtPct(precision.teamShareOfLobbyPct ?? 0),
    myHeavyShare: vm.fmtPct(heavy.myShareOfLobbyPct ?? 0),
    lobbyParity: vm.fmtPct(all.lobby.parity ?? 0),
  })
}

/** Le constat du bloc 3, alimenté par les mesures de la période. */
function objectivesConstat(vm: FormesViewModel): string {
  const withObjective = objectiveMatches(vm.block).length
  if (withObjective === 0) return vm.ct.objectivesConstatEmpty
  let measured = 0
  let above = 0
  for (const role of OBJECTIVE_ROLES) {
    const agg = aggregateRole(vm.block, role)
    if (agg.lobby <= 0) continue
    measured += 1
    if ((agg.teamShareOfLobbyPct ?? 0) > 50) above += 1
  }
  return vm.ct.objectivesConstatFmt({
    matchesWithObjective: withObjective,
    matchesTotal: vm.block.matches_total,
    rolesAboveParity: above,
    rolesMeasured: measured,
  })
}

export function FormesRetenuesSection({ block, locale }: FormesRetenuesSectionProps) {
  const t = FORMES_TEXT[locale]
  const ct = FORMES_CARDS_TEXT[locale]
  const vm = useMemo(
    () => (block != null && block.available ? buildFormesViewModel(block, t, ct, locale) : null),
    [block, t, ct, locale],
  )
  // Bloc absent du contrat (scope vide) ou indisponible (titre sans film) : la
  // section se retire, elle n'affiche pas une coquille vide.
  if (vm == null) return null

  const hasObjectives = objectiveMatches(vm.block).length > 0

  return (
    <section className="space-y-3" aria-label={t.sectionTitle}>
      <h3 className="text-base font-semibold text-foreground">{t.sectionTitle}</h3>
      <p className="max-w-[74ch] text-xs text-muted-foreground">
        <RichText text={t.intro} />
      </p>
      <HeaderStrip vm={vm} />

      <BlockTitle>{t.blocks.equipment.title}</BlockTitle>
      <Constat text={t.blocks.equipment.constat} />
      <Lexique text={t.blocks.equipment.lexique} />
      <ContextTitle>{t.contexts.solo}</ContextTitle>
      <div className="space-y-4">
        <EquipmentSharesCard vm={vm} />
        <EquipmentByMatchCard vm={vm} />
        <EquipmentSpreadCard vm={vm} />
      </div>
      <ContextTitle>{t.contexts.squad}</ContextTitle>
      <div className="space-y-4">
        <EquipmentRegularityCard vm={vm} />
        <EquipmentLobbyTrackCard vm={vm} />
        <EquipmentSquadGridCard vm={vm} />
        <EquipmentSquadTrackCard vm={vm} />
      </div>

      <BlockTitle>{t.blocks.weapons.title}</BlockTitle>
      <Constat text={weaponsConstat(vm)} />
      <Lexique text={t.blocks.weapons.lexique} />
      <ContextTitle>{t.contexts.solo}</ContextTitle>
      <div className="space-y-4">
        <PadsGapSoloCard vm={vm} />
        <PadsShareSoloCard vm={vm} />
        <PadsWeaponGridCard vm={vm} />
      </div>
      <ContextTitle>{t.contexts.squad}</ContextTitle>
      <div className="space-y-4">
        <PadsGapSquadCard vm={vm} />
        <PadsTwoFriezesCard vm={vm} />
        <PadsSquadByMatchCard vm={vm} />
        <PadsSquadWeaponGridCard vm={vm} />
      </div>

      <BlockTitle>{t.blocks.objectives.title}</BlockTitle>
      <Constat text={objectivesConstat(vm)} />
      <Lexique text={t.blocks.objectives.lexique} />
      {hasObjectives && (
        <>
          <ContextTitle>{t.contexts.solo}</ContextTitle>
          <div className="space-y-4">
            <ObjectivesGapRoleCard vm={vm} />
            <ObjectivesSharesByFamilyCard vm={vm} />
            <ObjectivesRawGridCard vm={vm} />
          </div>
          <ContextTitle>{t.contexts.squad}</ContextTitle>
          <div className="space-y-4">
            <ObjectivesGapSquadCard vm={vm} />
            <ObjectivesLobbyTrackCard vm={vm} />
          </div>
        </>
      )}
    </section>
  )
}
