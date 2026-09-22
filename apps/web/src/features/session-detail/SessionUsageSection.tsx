/**
 * SessionUsageSection — LES BLOCS « usages d'équipement, armes spéciales et objectifs »
 * de la page Sessions (chantier session-usage S3, GO utilisateur 2026-09-05 sur la
 * voie de repli : grammaire du handoff §1 + modèle de la vue match livrée).
 *
 * LA DONNÉE ARRIVE DANS LA RÉPONSE EXISTANTE (`SessionPageResponse.usage`, attaché
 * par le service Go — aucune query nouvelle, aucun appel de plus). Le contexte
 * Solo/Escouade est résolu EN AMONT par le serveur : le bloc porte `squad_players`
 * et des lignes `squad` par grandeur — vides en solo. À l'écran, l'escouade ajoute
 * une ligne par coéquipier dans les grilles et découpe la piste du lobby par
 * joueur ; le solo garde moi + les agrégats.
 *
 * CINQ CARTES (le gabarit `SectionCard` de la vue match) — l'équipement en occupe
 * TROIS depuis le 2026-09-13, une par vue (`SessionUsageEquipmentCards`) :
 *   1. « Cadences par match » — grille alignée moi / coéquipiers / équipe / lobby ;
 *   2. « Parts et parités » — les TROIS dénominateurs de jauge, toujours rendus ;
 *   3. « Régularité match par match » — une case par match, teintée par l'écart ;
 *   4. « Contrôle des armes spéciales » — jauges par famille d'arme puis leur TOTAL,
 *      PISTE DU LOBBY découpée par joueur (hachuré = eux, anonyme), bande de
 *      régularité ; ramassages non attribués et bonus ANONYMES en pied ;
 *   5. « Objectifs par rôle et par famille » — jauges par rôle, grille famille ×
 *      rôle, et (escouade) grille joueur × rôle. Son scope est INDÉPENDANT de la
 *      couverture des films.
 *
 * « Matchs mesurés N/M » est TOUJOURS visible (bandeau de titre de la première carte
 * de chaque bloc). Bloc indisponible → carte d'état vide avec la raison, jamais un
 * crash ni un bloc fantôme.
 *
 * CE QUI N'EST PLUS ÉCRIT SOUS LES GRAPHES (revue de lisibilité 2026-09-09,
 * `.ai/PLAN_SESSION_USAGE_LISIBILITE_2026-09-09.md`) : la lecture des barres, le trait
 * de parité, les cases de régularité et le corollaire du §4 (épisode actif ≠ bonus
 * ramassé, jamais additionnés) vivent dans UNE aide par carte, portée par le libellé du
 * titre (`cardTitleAdornment`). Le pied ne garde que des CHIFFRES — un chiffre ne se
 * survole pas. Ne pas re-déverser de méthode dans le corps : c'est exactement le pavé
 * qui a été retiré.
 *
 * Aucun calcul ici : projections dans `@/features/_shared/usage/` (déménagées de ce
 * dossier le 2026-09-09, étape E5.1 — `usageGaugeModel.ts`, `usageGrids.ts`, etc.), formes
 * dans `@/features/_shared/usage/UsageForms.tsx`, chrome dans `SessionUsageShared.tsx`.
 */
import { useMemo } from 'react'

import { ValueGrid } from '@/components/charts/ValueGrid'
import { SectionCard } from '@/components/ui/section-card'
import { tokenCssVar } from '@/lib/accessibility'
import type {
  SessionFlagGrabsNetBlock,
  SessionUsageBlock,
  SessionUsageMetric,
} from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import { useAppShellStore } from '@/stores/appShellStore'

import { usageAvailability } from '@/features/_shared/usage/usageAvailability'
import { buildGaugeRow } from '@/features/_shared/usage/usageGaugeModel'
import { buildObjectiveFamilyGrid, buildSquadRoleGrid } from '@/features/_shared/usage/usageGrids'
import { USAGE_TEXT, powerupLabel, roleLabel, type UsageText } from '@/features/_shared/usage/usageI18n'
import { buildLobbyTrack } from '@/features/_shared/usage/usageLobbyTrackModel'
import { buildPadTierGaugeRows, padTiersNotes } from '@/features/_shared/usage/usagePadTiersModel'
import { metricLabel, padMetric, roleToken } from '@/features/_shared/usage/usageMetricKinds'
import {
  formatUsageCount,
  formatUsagePct,
  formatUsageRate,
} from '@/features/_shared/usage/usageFormat'
import { sortRoles } from '@/features/_shared/usage/usageObjectives'
import { teamOfLobbyParityPct } from '@/features/_shared/usage/usageParity'
import { buildRegularityBand } from '@/features/_shared/usage/usageRegularityBandModel'
import { UsageGaugeGrid, UsageLobbyTrack, UsageRegularityBand } from '@/features/_shared/usage/UsageForms'

import { EquipmentCards } from './SessionUsageEquipmentCards'
import {
  bandAboveCaption,
  cardTitleAdornment,
  metricGaugeRows,
  useGridInks,
  type CardProps,
} from './SessionUsageShared'
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
  const availability = usageAvailability(usage, t)

  if (availability.kind === 'hidden' || usage == null) return null
  if (availability.kind === 'empty') {
    return (
      <>
        <SectionCard
          title={t.blockUnavailableTitle}
          label={t.blockUnavailableTitle}
          titleAdornment={
            usage.available
              ? cardTitleAdornment(
                  t.measuredFmt(usage.matches_measured, usage.matches_total),
                  t.cardHintShares,
                )
              : undefined
          }
        >
          <p className="px-3 pb-3 pt-3 text-sm text-muted-foreground">{availability.message}</p>
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
      <PadControlCard usage={usage} meLabel={meLabel} t={t} locale={locale} compact={compact} />
      <ObjectivesCard usage={usage} meLabel={meLabel} t={t} locale={locale} compact={compact} />
    </>
  )
}

/**
 * Bloc « armes spéciales » — familles nommées, leur total, bonus anonymes en pied.
 *
 * PAS DÉCOUPÉ EN TROIS comme l'équipement, et c'est mesuré : ses vues ne sont pas les
 * mêmes. Il n'a PAS de vue « cadences » (sa cadence tient en une ligne de pied, avec
 * les ramassages non attribués), et sa vue propre — « Qui ramasse les armes spéciales »,
 * la piste du lobby — n'existe nulle part ailleurs. Trois cartes pour deux formes et un
 * pied de chiffres auraient fabriqué des bandeaux vides.
 */
function PadControlCard({ usage, meLabel, t, locale, compact }: CardProps) {
  const pad = useMemo(() => padMetric(usage.metrics), [usage.metrics])
  const squadPlayers = useMemo(() => usage.squad_players ?? [], [usage.squad_players])
  // LES FAMILLES D'ABORD, LE TOTAL ENSUITE (D6). `pad_pickups` est exactement la somme
  // des familles : posé en tête et à leur niveau, il se lisait comme une grandeur de
  // plus. En pied et marqué `isTotal`, il redevient ce qu'il est — un total.
  const gaugeRows = useMemo(() => {
    const teamOfLobby = teamOfLobbyParityPct(usage.team_size_avg, usage.lobby_size_avg)
    const rows = (usage.pad_families ?? []).map((fam) =>
      buildGaugeRow({
        key: `family-${fam.family_key}`,
        // Le NOM vient du serveur (catalogue d'armes du titre, résolu à la requête).
        // Absent = famille hors catalogue : la CLÉ s'affiche, jamais un nom approchant
        // (même règle que le catalogue du rejeu — D7).
        label: fam.family_label || t.padFamilyFmt(fam.family_key),
        shares: fam,
        teamParityPct: usage.team_parity_pct,
        lobbyParityPct: usage.lobby_parity_pct,
        teamOfLobbyParityPct: teamOfLobby,
        t,
        locale,
      }),
    )
    if (pad) {
      rows.push(
        ...metricGaugeRows([pad], usage, t, locale).map((row) => ({ ...row, isTotal: true })),
      )
    }
    return rows
  }, [pad, usage, t, locale])
  const track = useMemo(
    () =>
      pad
        ? buildLobbyTrack({ meLabel, shares: pad, squadPlayers, squadShares: pad.squad, t, locale })
        : null,
    [pad, meLabel, squadPlayers, t, locale],
  )
  // LES NIVEAUX D'ARME (2026-09-14) : les MÊMES prises, rangées par niveau — base, terrain,
  // puissance. Section à part DANS LA MÊME CARTE : c'est une seconde lecture des mêmes socles,
  // pas une seconde grandeur ; une carte de plus l'aurait fait passer pour un autre sujet.
  const tierRows = useMemo(
    // LES PARITÉS DU BLOC viennent du bloc lui-même : celles de la carte portent sur le
    // périmètre du résumé d'usage, qui n'est pas le sien (revue du 2026-09-14).
    () => buildPadTierGaugeRows(usage.pad_tiers, { t, locale }),
    [usage.pad_tiers, t, locale],
  )
  const tierNotes = useMemo(() => padTiersNotes(usage.pad_tiers, t), [usage.pad_tiers, t])
  // LES NIVEAUX COMPTENT DANS LA PORTE (revue du 2026-09-14) : ils viennent d'une AUTRE passe,
  // sur d'autres matchs. Sans eux dans la condition, une session dont seuls les niveaux sont
  // mesures ne rendait RIEN — une mesure existante avalee par la porte de sa voisine. La
  // condition vit dans `sessionSectionVisibility` : le titre de section l'interroge aussi.
  if (!sessionUsageCardsShown(usage).padControl) return null

  return (
    <SectionCard
      title={t.blockPadControl}
      label={t.blockPadControl}
      titleAdornment={cardTitleAdornment(
        t.measuredFmt(usage.matches_measured, usage.matches_total),
        t.cardHintPadControl,
      )}
      footer={<PadControlFootnotes usage={usage} pad={pad} t={t} locale={locale} />}
    >
      <div className="space-y-5 px-3 pb-3 pt-3">
        {gaugeRows.length > 0 && (
          <section aria-label={t.viewShares}>
            <UsageGaugeGrid rows={gaugeRows} t={t} dense={compact} />
          </section>
        )}
        {tierRows.length > 0 && (
          <section aria-label={t.blockPadTiers}>
            <ViewTitle>{t.blockPadTiers}</ViewTitle>
            <UsageGaugeGrid rows={tierRows} t={t} dense={compact} />
            {tierNotes.map((note) => (
              <p key={note} className="pt-2 text-3xs text-muted-foreground">
                {note}
              </p>
            ))}
          </section>
        )}
        {track && (
          <section aria-label={t.viewLobbyTrack}>
            <ViewTitle>{t.viewLobbyTrack}</ViewTitle>
            <UsageLobbyTrack segments={track} label={t.viewLobbyTrack} dense={compact} />
          </section>
        )}
        {pad?.per_match != null && pad.per_match.length > 0 && (
          <section aria-label={t.viewRegularity}>
            <ViewTitle>{t.viewRegularity}</ViewTitle>
            <UsageRegularityBand
              label={metricLabel(pad.key, t)}
              cells={buildRegularityBand(pad.per_match, usage.team_parity_pct, t, locale)}
              caption={bandAboveCaption(pad, usage.matches_measured, t)}
              dense={compact}
            />
          </section>
        )}
      </div>
    </SectionCard>
  )
}

/**
 * Le pied du bloc « armes spéciales » : ce que les formes ne PEUVENT PAS porter — les
 * ramassages que la mesure n'attribue à personne, et les bonus qui ne sont attribuables
 * à personne.
 *
 * LA RÈGLE DE TRI EST « CHIFFRE OU MÉTHODE » (D5) : ce qui est un CHIFFRE reste au pied
 * — un chiffre ne se survole pas ; ce qui était de la MÉTHODE (le corollaire « épisode
 * actif ≠ bonus ramassé ») est parti dans l'aide d'en-tête. Le paragraphe de quatre
 * lignes devient au plus trois lignes de comptes. Rien à dire (session entièrement
 * attribuée, sans bonus, sans cadence) → pas de pied du tout.
 */
function PadControlFootnotes({
  usage,
  pad,
  t,
  locale,
}: {
  usage: SessionUsageBlock
  pad: SessionUsageMetric | null
  t: UsageText
  locale: Locale
}) {
  const powerups = usage.powerup_pickups ?? []
  const unnamed = usage.pad_unnamed_total ?? 0
  if (unnamed <= 0 && powerups.length === 0 && pad == null) return undefined
  const detail = powerups
    .map((p) => t.powerupDetailFmt(powerupLabel(p.family_key, t), p.occupations))
    .join(' · ')
  return (
    <div className="space-y-1 border-t border-border px-3 pb-2 pt-2 text-[11px] text-muted-foreground">
      {unnamed > 0 && <p>{t.padUnnamedFmt(unnamed)}</p>}
      {powerups.length > 0 && <p>{`${t.powerupLine} : ${detail}`}</p>}
      {pad != null && (
        <p>
          {t.padCadenceFmt(
            formatUsageRate(pad.player_per_match, locale),
            formatUsageRate(pad.team_per_match, locale),
            formatUsageRate(pad.lobby_per_match, locale),
          )}
        </p>
      )}
    </div>
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
  // Même source de vérité que le titre de section (cf. `sessionSectionVisibility`). Le
  // `obj == null` est REDONDANT à l'exécution (le prédicat l'inclut) : il est là pour que le
  // compilateur affine `obj` sur la suite, ce qu'un appel de fonction ne fait pas.
  if (obj == null || !sessionUsageCardsShown(usage).objectives) return null

  return (
    <SectionCard
      title={t.blockObjectives}
      label={t.blockObjectives}
      titleAdornment={cardTitleAdornment(
        t.objectivesScopeFmt(obj.matches_with_objectives, usage.matches_total),
        t.cardHintObjectives,
      )}
    >
      <div className="space-y-5 px-3 pb-3 pt-3">
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
        <FlagGrabsNetView block={obj.flag_grabs_net} t={t} locale={locale} />
      </div>
    </SectionCard>
  )
}

/**
 * FlagGrabsNetView — les PRISES NETTES de drapeau, sous les rôles.
 *
 * ELLE VIT À PART DES TROIS RÔLES, ET C'EST UNE CONSÉQUENCE DE LA MESURE. La
 * grandeur est lue du FILM : elle n'existe que pour les matchs dont l'artefact a
 * été lu, alors que les rôles se mesurent sur tous les matchs à objectif. La
 * verser dans « prendre » aurait compté un match sans film comme un match sans
 * prise. Elle porte donc ses propres dénominateurs, écrits en toutes lettres.
 *
 * LE COMPTEUR BRUT EST AFFICHÉ À CÔTÉ, et c'est le cœur du bloc : c'est l'écart
 * entre les deux qui dit ce que le compteur officiel compte en trop.
 */
function FlagGrabsNetView({
  block,
  t,
  locale,
}: {
  block: SessionFlagGrabsNetBlock | null | undefined
  t: UsageText
  locale: Locale
}) {
  if (block == null || block.lobby_raw_total <= 0) return null
  const n = (v: number) => formatUsageCount(v, locale)
  return (
    <section aria-label={t.viewFlagGrabsNet}>
      <ViewTitle>{t.viewFlagGrabsNet}</ViewTitle>
      {/* LE COUPLE COMPARABLE, et lui seul : le joueur ET son camp sur les matchs à camp
          connu. Comparer un total joueur « tout le scope » à un total d'équipe restreint
          ferait une part qui dépasse 100 %. */}
      <p className="text-2xs leading-relaxed text-muted-foreground">
        {t.flagGrabsNetFmt(
          n(block.player_team_scope_total),
          n(block.team_total),
          n(block.team_raw_total),
        )}{' '}
        {block.player_share_of_team_pct != null &&
          t.flagGrabsNetShareFmt(formatUsagePct(block.player_share_of_team_pct, locale))}
      </p>
      {block.window_seconds != null && block.window_seconds > 0 && (
        <p className="mt-1 text-3xs leading-relaxed text-muted-foreground">
          {t.flagGrabsNetRuleFmt(n(block.window_seconds))}
        </p>
      )}
      {/* CE QUE LE FILM A LU, avec son dénominateur : sans les ouvertures, « 25 ramassages »
          se lirait comme une exhaustivité. */}
      {block.openings_total > 0 && (
        <p className="mt-1 text-3xs leading-relaxed text-muted-foreground">
          {t.flagGrabsNetOpeningsFmt(n(block.lobby_raw_total), n(block.openings_total))}
        </p>
      )}
      <p className="mt-1 text-3xs leading-relaxed text-muted-foreground">
        {t.flagGrabsNetScopeFmt(block.matches_measured, block.matches_with_flag_family)}
      </p>
      {block.matches_team_known < block.matches_measured && (
        <p className="mt-1 text-3xs leading-relaxed text-muted-foreground">
          {t.flagGrabsNetTeamScopeFmt(block.matches_team_known, block.matches_measured)}
        </p>
      )}
    </section>
  )
}
