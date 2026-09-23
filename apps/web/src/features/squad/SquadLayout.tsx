/**
 * SquadLayout — layout partagé de la section Escouade.
 *
 * Gère la sélection des coéquipiers (via data.options), les KPI cards et la
 * navigation par onglets (Synergies / Contributions / Dynamique / Usages). Expose les données
 * sélectionnées via SquadContext pour les onglets enfants.
 *
 * Multi-titres : tous les libellés métier passent par useFieldMappings
 * (fields.toml du titre courant) ; les strings UI passent par getSquadText
 * (FR/EN). La liste des KPIs (SQUAD_KPI_METRICS) dégrade gracefully quand
 * un FieldKey est absent du titre courant.
 *
 * Barre de filtres unifiée : rendue par `SquadFilterBar` (ENFANT de ce layout).
 * Ce découpage est la correction du symptôme du 2026-09-20 (« dès que je touche
 * à un filtre, la page a l'air de recharger sans rien changer ») : tant que
 * l'état en attente de la barre vivait ICI, chaque case cochée re-rendait
 * `<Outlet />` et rejouait l'animation de tous les graphes. Ce layout ne
 * connaît plus que le COMMITÉ (filterContext) et l'IMMÉDIAT (coéquipiers,
 * sessions pickées, composition stricte) — ce dont sa requête a besoin.
 *
 * Route parente : /players/$playerSlug/squad
 * Routes enfants : /squad/synergies · /squad/contributions · /squad/dynamique · /squad/usages
 */
import { useState, useMemo } from 'react'
import { Outlet, useParams, Link, useMatchRoute } from '@tanstack/react-router'
import { useTitleSlug } from '@/lib/title-routing'
import { useSquadFilterStore } from '@/stores/squadFilterStore'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSquadPageRequests } from './useSquadPageRequests'
import { useSquadSessionSelection } from './useSquadSessionSelection'
import { useFiltersResolve } from '@/features/filters/queries'
import { EmptyStateCard } from '@/components/ui/empty-state'
import { AddFriendModal } from '@/features/friends/AddFriendFlow'
import { getSquadText } from './i18n'
import { SquadObjectiveStatsPanel } from './SquadObjectiveStatsPanel'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import { log } from './_logger'
import { SquadContext, type SquadContextValue } from './SquadContext'
import { SquadFilterBar } from './SquadFilterBar'
import { SquadFocusStrip } from './SquadFocusStrip'
import type { KPIStats, TeammateRow, TeammatesQueryRequest } from '@/lib/api/types'
import type { KPIStats as V2KPIStats } from './v2/types'
import { SessionBriefing } from '@/features/_shared/SessionBriefing'
import { formatDataIssues } from './squadDataIssues'
import { exactCompositionDefault } from './exactComposition'

import { useNavigateToMatch } from '@/lib/match-nav/useNavigateToMatch'
import { filterContextToMatchFilterSpec } from '@/lib/match-nav/fromFilterContext'

// ─── Constantes ───────────────────────────────────────────────────────────────

/**
 * Le DTO local v2/types.ts::KPIStats déclare avg_offensive_conversion /
 * avg_defensive_resistance / combat_profile en `T | null` (le Go peut renvoyer
 * null), alors que le contrat OpenAPI (KPIStats) les expose en `T | undefined`.
 * SessionBriefing consomme le type du contrat → on normalise null→undefined au
 * passage (le consommateur traite déjà null et undefined comme « absent »).
 */
function toContractKpis(k: V2KPIStats): KPIStats {
  return {
    ...k,
    avg_offensive_conversion: k.avg_offensive_conversion ?? undefined,
    avg_defensive_resistance: k.avg_defensive_resistance ?? undefined,
    combat_profile: k.combat_profile ?? undefined,
  }
}

function formatError(err: unknown): string {
  if (err == null) return 'Erreur inconnue'
  if (err instanceof Error) return err.message
  if (typeof err === 'string') return err
  if (typeof err === 'object') {
    const e = err as { message?: unknown; status?: unknown; statusText?: unknown }
    if (typeof e.message === 'string') return e.message
    if (typeof e.statusText === 'string') {
      return typeof e.status === 'number' ? `${e.status} ${e.statusText}` : e.statusText
    }
    try { return JSON.stringify(err) } catch { return 'Erreur non sérialisable' }
  }
  return String(err)
}

// ─── Composant principal ──────────────────────────────────────────────────────

export function SquadLayout() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const titleSlug = useTitleSlug()
  const {
    filterContext,
    filterContextHash,
    resetFilters,
  } = useSquadFilterStore()
  // Composition (lien profond de l'accueil, amis, choix), session pickée — source
  // UNIQUE : le store escouade — et moment où la requête lourde peut partir.
  const {
    selectedGts,
    setSelectedGts,
    pickedSquadSessionLabels,
    applySessionLabels,
    mountApplied,
    teammatesReady,
  } = useSquadSessionSelection(playerSlug)
  // Résout le filterContext squad côté backend → alimente `resolvedContext`
  // (options de session, cascade disponible, presets de période) pour les pills et le rail.
  // match_context='squad' : même population que l'aperçu, qui ne tourne plus que
  // filtres en attente (D4.3). Attend l'état de montage (lien profond, migration).
  useFiltersResolve(playerSlug, useSquadFilterStore, { matchContext: 'squad', enabled: mountApplied })
  // L'ancrage de session est piloté par la COMPOSITION (cf. effet de ré-ancrage
  // plus bas) — pas via useFollowLatestSession, qui snappait sur la dernière
  // session squad du joueur principal (composition-agnostique → ajoutait un
  // coéquipier à une session qu'il n'avait pas jouée).
  const locale = useAppShellStore((s) => s.locale)
  const hasLinkedIdentity = useAppShellStore((s) => !!s.linkedHaloIdentity)
  const t = getSquadText(locale)
  const tCommon = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  const confirmedGts = selectedGts
  const [addFriendGamertag, setAddFriendGamertag] = useState<string | null>(null)

  // ── Option « composition stricte » (cochée par défaut) ───────────────────
  // Par défaut, seuls les matchs joués avec exactement cette composition sont
  // comptés. La décocher élargit à la règle « matchs commencés ensemble »
  // (intersection du roster), même si un autre joueur connu accompagnait
  // l'équipe. Choix persisté par joueur (cf. exactCompositionDefault),
  // appliqué en direct (pas de passage par Analyser, comme les
  // coéquipiers/sessions).
  const exactCompositionStorageKey = `squad-exact-composition-${playerSlug}`
  const [exactComposition, setExactCompositionRaw] = useState<boolean>(() => {
    try {
      return exactCompositionDefault(localStorage.getItem(exactCompositionStorageKey))
    } catch { return exactCompositionDefault(null) }
  })
  const setExactComposition = (value: boolean) => {
    setExactCompositionRaw(value)
    try { localStorage.setItem(exactCompositionStorageKey, String(value)) } catch { /* ignore */ }
  }

  const matchRoute = useMatchRoute()

  // ── Requête TeammatesService ─────────────────────────────────────────────
  // match_context="squad" : le backend ne considère que les matchs is_with_friends=true.
  // `picked_squad_session_labels` (contrat serveur inchangé) = les sessions du store,
  // les mêmes que `filters.sessions` : une seule source, une seule clé (D4.1).
  const squadFilterContext = useMemo(() => ({ ...filterContext, match_context: 'squad' as const }), [filterContext])
  const request: TeammatesQueryRequest = {
    filters: squadFilterContext,
    selected_gamertags: confirmedGts.length > 0 ? confirmedGts : undefined,
    picked_squad_session_labels: pickedSquadSessionLabels.length > 0 ? pickedSquadSessionLabels : undefined,
    locale,
    filter_exact_composition: exactComposition,
  }
  // Deux requêtes (lot perf L4b) : la LÉGÈRE (sessions de la composition) décide de
  // l'ancrage, la LOURDE ne part qu'ensuite, déjà sur la bonne session — cf.
  // useSquadPageRequests, qui porte aussi le ré-ancrage et la réconciliation des
  // sessions pickées. `isPending` et non `isLoading` : tant qu'elle attend (composition
  // initiale inconnue, D4.2 ; ancrage pas encore décidé, L4b), la requête lourde est
  // DÉSACTIVÉE — « Chargement… », jamais l'état vide.
  const {
    teammates: { data, isPending, isError, error },
    compositionSessions,
  } = useSquadPageRequests({
    playerSlug,
    request,
    filterContextHash,
    selectedGts: confirmedGts,
    exactComposition,
    teammatesReady,
    pickedSquadSessionLabels,
    applySessionLabels,
  })
  // `compositionSessions` (sélecteur, rail, état vide) : avec coéquipier(s), les sessions
  // de la COMPOSITION (intersection, historique complet) — jamais celles du joueur
  // principal quand elle est vide, on afficherait des sessions non jouées par la
  // composition ; sans coéquipier, les sessions escouade du joueur principal.
  const hasTeammates = confirmedGts.length > 0

  // Dégradations remontées par l'API (chargements best-effort en échec) :
  // affichées telles quelles — un chiffre partiel doit se voir, pas se deviner.
  const dataIssueMessages = useMemo(() => formatDataIssues(data?.data_issues, t), [data?.data_issues, t])

  // Bouton "Voir les matchs" (L2) : la liste de matchs de la composition est
  // déjà chargée ici (data.match_history, DESC), donc zéro aller-retour serveur.
  // On reproduit exactement la logique de SquadMatchHistoryTable : matchIds en
  // ordre chronologique (oldest-first), point d'entrée = match le plus récent.
  const navigateToMatch = useNavigateToMatch(playerSlug)
  const squadMatchIds = useMemo(
    () => [...(data?.match_history ?? [])].reverse().map((m) => m.match_id),
    [data?.match_history],
  )
  const squadEntryMatchId = data?.match_history?.[0]?.match_id
  const handleBrowseMatches = () => {
    if (!squadEntryMatchId) return
    const filterSpec = filterContextToMatchFilterSpec(squadFilterContext)
    navigateToMatch(squadEntryMatchId, {
      source: 'session',
      matchIds: squadMatchIds,
      filterSpec: filterSpec ?? undefined,
    })
  }
  // Bouton « Voir les matchs » — rendu dans le rail (zone centrale, après le
  // compteur de matchs) pour décharger la barre de filtres. squadEntryMatchId
  // (1er match de match_history, population escouade) suffit à prouver qu'il
  // existe au moins un match à parcourir — l'ancien garde-fou additionnel (le
  // total post-filtres du joueur PRINCIPAL via /filters/resolve, pas celui de
  // l'escouade affichée — ADR 0033, garde-rail singleCountSource.guard.test.ts)
  // retiré : redondant de toute façon.
  const browseButton =
    squadEntryMatchId ? (
      <button
        type="button"
        onClick={handleBrowseMatches}
        className="shrink-0 inline-flex items-center gap-1 rounded-md border border-input bg-background px-2.5 py-1 text-xs font-medium text-foreground transition-colors hover:bg-muted"
        title={tCommon('common.filters.browse_title')}
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 16 16"
          fill="currentColor"
          className="h-3.5 w-3.5 opacity-70"
          aria-hidden="true"
        >
          <path d="M6.22 8.72a.75.75 0 0 0 1.06 1.06l5.22-5.22v1.69a.75.75 0 0 0 1.5 0v-3.5a.75.75 0 0 0-.75-.75h-3.5a.75.75 0 0 0 0 1.5h1.69L6.22 8.72Z" />
          <path d="M3.5 6.75c0-.69.56-1.25 1.25-1.25H7A.75.75 0 0 0 7 4H4.75A2.75 2.75 0 0 0 2 6.75v4.5A2.75 2.75 0 0 0 4.75 14h4.5A2.75 2.75 0 0 0 12 11.25V9a.75.75 0 0 0-1.5 0v2.25c0 .69-.56 1.25-1.25 1.25h-4.5c-.69 0-1.25-.56-1.25-1.25v-4.5Z" />
        </svg>
        {tCommon('common.filters.browse_label')}
      </button>
    ) : null

  // ── Routes actives ───────────────────────────────────────────────────────
  const synergiesRoute = '/{-$lang}/t/$titleSlug/players/$playerSlug/squad/synergies' as const
  const contributionsRoute = '/{-$lang}/t/$titleSlug/players/$playerSlug/squad/contributions' as const
  const dynamiqueRoute = '/{-$lang}/t/$titleSlug/players/$playerSlug/squad/dynamique' as const
  const usagesRoute = '/{-$lang}/t/$titleSlug/players/$playerSlug/squad/usages' as const
  const isSynergies = !!matchRoute({ to: synergiesRoute, fuzzy: true })
  const isContributions = !!matchRoute({ to: contributionsRoute, fuzzy: true })
  const isDynamique = !!matchRoute({ to: dynamiqueRoute, fuzzy: true })
  const isUsages = !!matchRoute({ to: usagesRoute, fuzzy: true })

  // ── Gestion chargement / erreur ──────────────────────────────────────────
  // La barre de filtres (sticky) est toujours rendue; seul le contenu est
  // conditionnel pour ne pas faire disparaître les contrôles lors d'un refetch.
  const availableOptions = data?.options ?? []
  // Mémoïsé : `selectedRows` est publié par SquadContext. Sans mémo, un rendu du
  // layout SANS nouvelle donnée (rafraîchissement du store, saisie ailleurs) en
  // recréait un tableau neuf → identité de la valeur de contexte changée → tous
  // les consommateurs (Synergies / Contributions / Dynamique / FocusStrip) et
  // leurs graphes ECharts re-rendaient pour rien.
  const selectedRows = useMemo(() => {
    const teammates = data?.teammates ?? []
    return confirmedGts
      .map((gt) => teammates.find((r) => r.gamertag.toLowerCase() === gt.toLowerCase()))
      .filter(Boolean) as TeammateRow[]
  }, [data?.teammates, confirmedGts])

  if (confirmedGts.length > 0 && !isPending && selectedRows.length === 0) {
    log.warn(
      `invalid_selection:${playerSlug}`,
      `Aucun gamertag confirmé n'a matché un teammate côté backend (player=${playerSlug}).`,
      { confirmedGts },
    )
  }

  // XUID absolu du joueur courant — résolu depuis la card "moi" du header
  // (player_cards filtré sur main_player). Sert à exclure le viewer du roster de
  // façon player-agnostic (jamais par le slug d'URL). Vide tant que le header
  // squad n'est pas chargé (dégradation gracieuse transitoire).
  const currentPlayerXuid = useMemo(() => {
    const mainGT = data?.main_player ?? ''
    return data?.header?.player_cards?.find((c) => c.gamertag === mainGT)?.xuid ?? ''
  }, [data])

  // Valeur de contexte mémoïsée — cf. `selectedRows` ci-dessus : un objet
  // littéral recréé à chaque rendu rendait la mémoïsation de `<Outlet />`
  // (React.memo côté routeur) sans effet.
  const squadContextValue = useMemo<SquadContextValue>(
    () => ({
      selectedRows,
      confirmedGamertags: confirmedGts,
      pageData: data ?? null,
      playerSlug,
      currentPlayerXuid,
    }),
    [selectedRows, confirmedGts, data, playerSlug, currentPlayerXuid],
  )

  return (
    <SquadContext.Provider value={squadContextValue}>
      <SquadFilterBar
        playerSlug={playerSlug}
        locale={locale}
        hasLinkedIdentity={hasLinkedIdentity}
        selectedGts={selectedGts}
        setSelectedGts={setSelectedGts}
        availableOptions={availableOptions}
        selectedRows={selectedRows}
        currentPlayerXuid={currentPlayerXuid}
        onAddFriendGamertag={setAddFriendGamertag}
        compositionSessions={compositionSessions}
        pickedSquadSessionLabels={pickedSquadSessionLabels}
        applySessionLabels={applySessionLabels}
        hasTeammates={hasTeammates}
        exactComposition={exactComposition}
        setExactComposition={setExactComposition}
        browseButton={browseButton}
        // Les sessions pickées vivent dans le store : le reset les vide avec le reste.
        onReset={resetFilters}
      />

      {/* ─── Contenu ─────────────────────────────────────────────────────────── */}
      {isPending && (
        <div className="flex items-center justify-center p-12 text-sm text-muted-foreground">
          Chargement…
        </div>
      )}

      {!isPending && isError && (
        <div className="p-6 text-center text-destructive">
          {t.errors.loadError(formatError(error))}
        </div>
      )}

      {/* Données partielles : un chargement best-effort a échoué côté serveur.
          Visible plutôt que silencieux — sinon les compteurs varient d'une
          requête à l'autre sans explication (cf. domain.DataIssue).
          text-warning (et non text-warning-foreground, pensé pour un fond plein) :
          sur le tint bg-warning/10 ce dernier est illisible en thème sombre —
          piège documenté dans components/ui/privacy-banner.tsx. */}
      {dataIssueMessages.length > 0 && (
        <div
          role="status"
          className="mx-4 mt-3 rounded-md border border-warning/40 bg-warning/10 px-3 py-2 text-xs text-warning"
        >
          <p className="font-medium">{t.dataIssues.title}</p>
          <ul className="mt-1 list-disc pl-4">
            {dataIssueMessages.map((msg) => (
              <li key={msg}>{msg}</li>
            ))}
          </ul>
        </div>
      )}

      {!isPending && !isError && !data && (
        <div className="p-6">
          <EmptyStateCard title={t.empty.noDataTitle} description={t.empty.noDataDescription} />
        </div>
      )}

      {/* Coéquipier(s) sélectionné(s) mais aucune session commune : la composition
          exacte n'a jamais joué ensemble (sur le scope filtré) → état vide clair,
          pas les stats du joueur principal. */}
      {!isPending && !isError && data && hasTeammates && compositionSessions.length === 0 && (
        <div className="p-6">
          <EmptyStateCard
            title={t.empty.invalidSelectionTitle}
            description={t.empty.invalidSelectionDescription}
          />
        </div>
      )}

      {!isPending && !isError && data && !(hasTeammates && compositionSessions.length === 0) && (
        <div className="flex flex-col gap-6 p-6">
          {/* SessionBriefing — KPIs + verdict squad + drill-down click */}
          {/* Remplace l'ancienne section "Synergies avec les coéquipiers sélectionnés"
             (KPIBlock par teammate) — meme info accessible via le drill-down click sur
             la card du joueur dans la bande verdict. */}
          {data?.header?.solo_kpis && (() => {
            const header = data.header
            const soloKpis = header.solo_kpis
            if (!soloKpis) return null
            // Réutilise le XUID courant déjà mémoïsé (source unique, cf.
            // currentPlayerXuid) plutôt que de le recalculer inline.
            const briefingSquad =
              header?.squad_score &&
              header?.player_cards &&
              header?.team_avg_kpis &&
              header?.kpis_by_xuid &&
              currentPlayerXuid
                ? {
                    score: header.squad_score,
                    players: header.player_cards,
                    kpisByXuid: Object.fromEntries(
                      Object.entries(header.kpis_by_xuid).map(([x, k]) => [x, toContractKpis(k)]),
                    ),
                    teamAvgKpis: toContractKpis(header.team_avg_kpis),
                    activeXuid: currentPlayerXuid,
                  }
                : undefined
            return <SessionBriefing kpis={toContractKpis(soloKpis)} squad={briefingSquad} />
          })()}

          {/* Objectifs de l'escouade (CTF/Zones/Oddball) — capability-gated + data-driven. */}
          <SquadObjectiveStatsPanel
            statsByXuid={data?.header?.objective_stats_by_xuid}
            texts={t}
            numLoc={t.intlLocale}
          />

          {/* « Cap d'escouade » (Enregistrer cette compo) — remonté AU-DESSUS de la
              barre d'onglets L3, commun aux deux onglets (Synergies / Contributions). */}
          <SquadFocusStrip />

          {/* Navigation onglets */}
          <div className="border-b">
            <nav className="flex gap-0">
              <Link
                to="/{-$lang}/t/$titleSlug/players/$playerSlug/squad/synergies"
                params={{ titleSlug, playerSlug }}
                className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${isSynergies ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'}`}
              >
                {t.nav.synergies}
              </Link>
              <Link
                to="/{-$lang}/t/$titleSlug/players/$playerSlug/squad/contributions"
                params={{ titleSlug, playerSlug }}
                className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${isContributions ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'}`}
              >
                {t.nav.contributions}
              </Link>
              <Link
                to="/{-$lang}/t/$titleSlug/players/$playerSlug/squad/dynamique"
                params={{ titleSlug, playerSlug }}
                className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${isDynamique ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'}`}
              >
                {t.nav.dynamique}
              </Link>
              <Link
                to="/{-$lang}/t/$titleSlug/players/$playerSlug/squad/usages"
                params={{ titleSlug, playerSlug }}
                className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${isUsages ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'}`}
              >
                {t.nav.usages}
              </Link>
            </nav>
          </div>

          <Outlet />
        </div>
      )}

      {addFriendGamertag && (
        <AddFriendModal
          playerSlug={playerSlug}
          gamertag={addFriendGamertag}
          open={!!addFriendGamertag}
          onClose={() => setAddFriendGamertag(null)}
          locale={locale}
        />
      )}
    </SquadContext.Provider>
  )
}
