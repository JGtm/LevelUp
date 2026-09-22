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
 * LE BLOC SE RETIRE DE LUI-MÊME quand la réponse ne le porte pas (titre sans film).
 *
 * PLUS DE TITRE DE SECTION NI DE BANDEAU DE COUVERTURE (décision D5 du 2026-09-21) : trois
 * intertitres se suivaient — « Les formes retenues », puis quatre tuiles de repères, puis le
 * vrai intertitre du premier bloc. Le nom de la section reste celui de l'ARIA ; les repères
 * du bandeau se relisent dans les formes elles-mêmes.
 *
 * UN BLOC SANS DONNÉE NOMME SA CAUSE (décision D8) : il reste affiché et dit POURQUOI il est
 * vide — aucun film décodé, aucun usage dans les modes retenus, aucun socle. Une section
 * entière sans objet (aucun match à objectif) se masque, intertitre compris : un intertitre
 * orphelin annonce un bloc qui n'arrive jamais.
 */
import { useMemo } from 'react'

import { SectionTitle } from '@/components/ui/detail-section'
import { EmptyStateNotice } from '@/components/ui/empty-state'
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
import { FORMES_TEXT, type FormesEmptyCause, type FormesText } from './i18n'
import { EQUIPMENT_AXES, axisValue, lobbyOf } from './model/access'
import { objectiveMatches } from './model/objectives'
import { namedPickups, unnamedOccupations } from './model/pads'
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

/**
 * LA CAUSE D'UN BLOC VIDE, DÉDUITE DES COMPTES (décision D8). Aucun match mesuré : le film
 * manque. Des matchs mesurés mais aucune mesure sur l'axe : les modes retenus ne portent pas
 * la chose. Rien de déductible : le message générique — jamais un écran muet.
 */
function equipmentCause(vm: FormesViewModel): FormesEmptyCause | null {
  if (vm.measured.length === 0) return 'noFilm'
  // Un match mesuré SANS AUCUN joueur de lobby n'est pas « zéro usage » : c'est une réponse
  // qu'on ne sait pas lire. On ne lui invente pas de cause, on le dit.
  const places = vm.measured.reduce((a, m) => a + lobbyOf(m).length, 0)
  if (places === 0) return 'generic'
  const total = vm.measured.reduce(
    (acc, m) =>
      acc +
      lobbyOf(m).reduce((a, p) => a + EQUIPMENT_AXES.reduce((x, k) => x + axisValue(p, k), 0), 0),
    0,
  )
  return total > 0 ? null : 'noEquipment'
}

function padsCause(vm: FormesViewModel): FormesEmptyCause | null {
  if (vm.measured.length === 0) return 'noFilm'
  if (namedPickups(vm.block) > 0) return null
  return unnamedOccupations(vm.block) > 0 ? 'padsUnnamedOnly' : 'noPads'
}

/**
 * Le bloc vide, à la place de ses cartes — RENDU PAR L'ÉTAT VIDE CANONIQUE DE L'APP
 * (`EmptyStateNotice`, 2026-09-22). Ce composant portait son propre cadre (`rounded-lg`,
 * pointillés, `py-6`) et une seule ligne grise : deux écarts au gabarit que toutes les
 * autres sections posent (titre en gras + description, `rounded-xl`, fond `muted`). La
 * typographie d'un état vide se décide en UN endroit, pas par section.
 */
function BlockEmpty({ cause, t }: { cause: FormesEmptyCause; t: FormesText }) {
  return (
    <div data-testid="formes-block-empty" data-formes-empty={cause}>
      <EmptyStateNotice title={t.emptyTitles[cause]} description={t.empty[cause]} />
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
  const equipmentEmpty = equipmentCause(vm)
  const padsEmpty = padsCause(vm)

  return (
    <section className="space-y-3" aria-label={t.sectionTitle}>
      <BlockTitle aide={t.blocks.equipment.aide}>{t.blocks.equipment.title}</BlockTitle>
      {equipmentEmpty != null ? (
        <BlockEmpty cause={equipmentEmpty} t={t} />
      ) : (
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
      )}

      <BlockTitle aide={t.blocks.weapons.aide}>{t.blocks.weapons.title}</BlockTitle>
      {padsEmpty != null ? (
        <BlockEmpty cause={padsEmpty} t={t} />
      ) : (
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
      )}

      {/* AUCUN MATCH À OBJECTIF = AUCUNE SECTION : l'intertitre partait avec (D8). */}
      {hasObjectives && (
        <>
          <BlockTitle aide={t.blocks.objectives.aide}>{t.blocks.objectives.title}</BlockTitle>
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
        </>
      )}
    </section>
  )
}
