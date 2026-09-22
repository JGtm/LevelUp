/**
 * SessionUsageSection — LES BLOCS « usages d'équipement, armes spéciales et objectifs »
 * de la page Sessions (chantier session-usage S3, GO utilisateur 2026-09-05 sur la
 * voie de repli : grammaire du handoff §1 + modèle de la vue match livrée).
 *
 * LA DONNÉE ARRIVE DANS LA RÉPONSE EXISTANTE (`SessionPageResponse.usage`, attaché
 * par le service Go — aucune query nouvelle, aucun appel de plus). Le contexte
 * Solo/Escouade est résolu EN AMONT par le serveur : le bloc porte `squad_players`
 * et des lignes `squad` par grandeur — vides en solo.
 *
 * HUIT CARTES DEPUIS LE 2026-09-21 (le gabarit `SectionCard`) : l'équipement en occupe
 * TROIS (`SessionUsageEquipmentCards`, découpage du 2026-09-13), les armes spéciales
 * QUATRE (`SessionPadControlCards`, D6 — le même découpage que Synthèse et Escouade),
 * et les objectifs UNE. Les deux fichiers de cartes portent le détail de leur rangée.
 *
 * CE QUI N'EST PLUS ÉCRIT SOUS LES GRAPHES : la lecture des barres, le trait de parité,
 * les cases de régularité et les réserves de mesure vivent dans UNE infobulle (i) par
 * carte, à droite du titre (`usageCardTitle`). Ne pas re-déverser de méthode dans le
 * corps : c'est exactement le pavé qui a été retiré.
 *
 * TROIS RETRAITS DU 2026-09-21 (lot A2, retours utilisateur) : le compteur « Matchs
 * mesurés N/M » des bandeaux, le PIED du contrôle des armes spéciales (ramassages non
 * attribués, bonus anonymes, cadence) et les cinq phrases explicatives des prises nettes
 * de drapeau (D1) — la vue garde son titre, la grandeur est dite par la jauge du rôle
 * « prendre ».
 *
 * Aucun calcul ici : projections dans `@/features/_shared/usage/` (déménagées de ce
 * dossier le 2026-09-09, étape E5.1), formes dans `UsageForms.tsx` / `UsageLobbyTrack.tsx`
 * / `UsageRegularityBand.tsx`, chrome dans `SessionUsageShared.tsx`.
 */
import { useMemo } from 'react'

import { ValueGrid } from '@/components/charts/ValueGrid'
import { SectionCard } from '@/components/ui/section-card'
import { tokenCssVar } from '@/lib/accessibility'
import type { SessionFlagGrabsNetBlock, SessionUsageBlock } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { UsageEmptyNotice } from '@/features/_shared/usage/UsageEmptyNotice'
import { UsageGaugeGrid } from '@/features/_shared/usage/UsageForms'
import { usageAvailability } from '@/features/_shared/usage/usageAvailability'
import { usageCardTitle } from '@/features/_shared/usage/usageCardTitle'
import { buildGaugeRow } from '@/features/_shared/usage/usageGaugeModel'
import { buildObjectiveFamilyGrid, buildSquadRoleGrid } from '@/features/_shared/usage/usageGrids'
import { USAGE_TEXT, roleLabel, type UsageText } from '@/features/_shared/usage/usageI18n'
import { roleToken } from '@/features/_shared/usage/usageMetricKinds'
import { sortRoles } from '@/features/_shared/usage/usageObjectives'
import { teamOfLobbyParityPct } from '@/features/_shared/usage/usageParity'

import { PadControlCards } from './SessionPadControlCards'
import { EquipmentCards } from './SessionUsageEquipmentCards'
import { useGridInks, type CardProps } from './SessionUsageShared'
import { sessionUsageCardsShown } from './sessionSectionVisibility'

/** Le titre d'une vue à l'intérieur d'une carte (même gabarit que la vue match). */
function ViewTitle({ children }: { children: string }) {
  return (
    <h4 className="mb-2 text-3xs font-semibold uppercase tracking-wider text-muted-foreground">
      {children}
    </h4>
  )
}

interface Props {
  /** Le bloc `usage` de la réponse — absent (vieux serveur) : rien ne se rend. */
  usage: SessionUsageBlock | null | undefined
  /** Libellé du joueur de la page (slug de la route — l'identité des lignes « moi »). */
  meLabel: string
  /** Colonne divisée : tout reste rendu, en plus serré (cf. `CardProps.compact`). */
  compact?: boolean
}

export function SessionUsageSection({ usage, meLabel, compact = false }: Props) {
  const locale = useAppShellStore((s) => s.locale)
  const t = USAGE_TEXT[locale]
  const availability = usageAvailability(usage)

  if (availability.kind === 'hidden' || usage == null) return null
  if (availability.kind === 'empty') {
    return (
      <>
        <SectionCard title={t.blockUnavailableTitle} label={t.blockUnavailableTitle}>
          <div className="px-3 pb-3 pt-3">
            <UsageEmptyNotice reason={usage.available ? 'no-film' : 'load-failed'} t={t} />
          </div>
        </SectionCard>
        {/* Les objectifs vivent HORS films : ils s'affichent même sans match mesuré. */}
        {usage.available && (
          <ObjectivesCard usage={usage} meLabel={meLabel} t={t} locale={locale} compact={compact} />
        )}
      </>
    )
  }

  return (
    <>
      <EquipmentCards usage={usage} meLabel={meLabel} t={t} locale={locale} compact={compact} />
      <PadControlCards usage={usage} meLabel={meLabel} t={t} locale={locale} compact={compact} />
      <ObjectivesCard usage={usage} meLabel={meLabel} t={t} locale={locale} compact={compact} />
    </>
  )
}

/** Bloc « objectifs » — par rôle et famille (scope indépendant des films). */
function ObjectivesCard({ usage, meLabel, t, locale, compact }: CardProps) {
  const inks = useGridInks()
  const obj = usage.objectives
  const roles = useMemo(() => sortRoles(obj?.roles), [obj])
  const squadPlayers = useMemo(() => usage.squad_players ?? [], [usage.squad_players])
  const roleInks = useMemo(
    () => ({ ...inks, columnColor: (key: string) => tokenCssVar(roleToken(key)) }),
    [inks],
  )
  const gaugeRows = useMemo(() => {
    if (obj == null) return []
    const teamOfLobby = teamOfLobbyParityPct(obj.team_size_avg, obj.lobby_size_avg)
    return roles.map((r) =>
      buildGaugeRow({
        key: r.role,
        label: roleLabel(r.role, t),
        shares: r,
        isDuration: r.is_duration === true,
        teamParityPct: obj.team_parity_pct,
        lobbyParityPct: obj.lobby_parity_pct,
        teamOfLobbyParityPct: teamOfLobby,
        t,
        locale,
      }),
    )
  }, [obj, roles, t, locale])
  const familyGrid = useMemo(
    () => buildObjectiveFamilyGrid({ families: obj?.families ?? [], t, locale, ...roleInks }),
    [obj, t, locale, roleInks],
  )
  const squadGrid = useMemo(
    () => buildSquadRoleGrid({ roles, squadPlayers, meLabel, t, locale, ...roleInks }),
    [roles, squadPlayers, meLabel, t, locale, roleInks],
  )
  // SECTION ENTIÈRE SANS OBJET = MASQUÉE (D8) : aucun bloc d'objectifs servi, il n'y a pas
  // de rangée à laisser bancale. En revanche un bloc SERVI mais sans rôle mesuré garde sa
  // carte et dit pourquoi — c'est une sélection sans mode à objectif, pas une absence de
  // mesure. LA CONDITION S'ÉCRIT DANS `sessionSectionVisibility` : le titre de section
  // « Frags et usages » la lit aussi. Le `obj == null` est REDONDANT à l'exécution (le
  // prédicat l'inclut) — il est là pour que le compilateur affine `obj` sur la suite.
  if (obj == null || !sessionUsageCardsShown(usage).objectives) return null
  const vide = roles.length === 0

  return (
    <SectionCard
      title={t.blockObjectives}
      label={t.blockObjectives}
      titleAdornment={usageCardTitle(
        t.cardHintObjectives,
        t.objectivesScopeFmt(obj.matches_with_objectives, usage.matches_total),
      )}
    >
      <div className="space-y-5 px-3 pb-3 pt-3">
        {vide ? (
          <UsageEmptyNotice reason="no-objectives" t={t} />
        ) : (
          <>
            <section aria-label={t.viewRoles}>
              <UsageGaugeGrid rows={gaugeRows} t={t} dense={compact} />
            </section>
            {familyGrid && (
              <section aria-label={t.viewFamilies}>
                <ViewTitle>{t.viewFamilies}</ViewTitle>
                <ValueGrid model={familyGrid} dense={compact} />
              </section>
            )}
            {squadGrid && (
              <section aria-label={t.viewSquadRoles}>
                <ViewTitle>{t.viewSquadRoles}</ViewTitle>
                <ValueGrid model={squadGrid} dense={compact} />
              </section>
            )}
            <FlagGrabsNetView block={obj.flag_grabs_net} t={t} />
          </>
        )}
      </div>
    </SectionCard>
  )
}

/**
 * FlagGrabsNetView — les PRISES NETTES de drapeau, sous les rôles.
 *
 * ELLE NE PORTE PLUS QUE SON TITRE (D1, 2026-09-21). Ses cinq paragraphes — l'écart
 * brut/net, la fenêtre de jonglage repliée, les ouvertures lues par le film, la couverture
 * en matchs de Capture du drapeau et le périmètre d'équipe — ont été retirés sur demande
 * utilisateur : cinq phrases de méthode pour deux chiffres, sous une carte qui en portait
 * déjà trois vues. La grandeur est dite par la jauge du rôle « prendre », qui reste.
 *
 * LA VUE RESTE CONDITIONNÉE À UNE MESURE : sans prise lue sur le scope, pas de titre — un
 * intertitre seul, sans rien dessous, serait un bloc mort.
 */
function FlagGrabsNetView({
  block,
  t,
}: {
  block: SessionFlagGrabsNetBlock | null | undefined
  t: UsageText
}) {
  if (block == null || block.lobby_raw_total <= 0) return null
  return (
    <section aria-label={t.viewFlagGrabsNet}>
      <ViewTitle>{t.viewFlagGrabsNet}</ViewTitle>
    </section>
  )
}
