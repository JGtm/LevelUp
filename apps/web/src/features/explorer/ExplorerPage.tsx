/**
 * ExplorerPage — page Explorer (recherche + filtres).
 *
 * Orchestrateur slim. Le détail de chaque mode est dans son propre fichier
 * (voir audit #6 god-file split) :
 *   - ExplorerPage.matchesMode.tsx — filtres + tableau paginé
 *   - ExplorerPage.playerMode.tsx  — recherche gamertag + briefing + tables
 *
 * Mode Matchs : filtres (dropdowns + date range) + tableau paginé
 *               (ExplorerMatchesTable, repris du SquadMatchHistoryTable).
 * Mode Joueur : historique commun paginé (20/page) + badges encounter.
 *
 * État persistant : `mode`/`target` + tous les filtres de scope vivent dans
 * l'URL via `usePageScope` (+ miroir localStorage pour le cold-start). Ouvrir
 * un match puis revenir en arrière restaure donc tous les filtres — cf. plan
 * nav-context-unification (Phase 1) et `@/lib/page-scope`.
 */
import { useState, useEffect, useMemo } from 'react'
import { useParams, useNavigate, useSearch } from '@tanstack/react-router'
import { useTitleSlug } from '@/lib/title-routing'
import { Button } from '@/components/ui/button'
import { useDebounced } from '@/lib/hooks/useDebounced'
import { useExplorerMatches, useExplorerPlayer } from './queries'
import { DEFAULT_FILTER_CONTEXT } from '@/stores/createFilterStore'
import { useActiveSeason, seasonToPeriod } from '@/features/squad/useActiveSeason'
import type { ContextDescriptor } from '@/lib/match-nav/navContext'
import { formatMessage } from '@/lib/i18n/format'
import { explorerManifest, type ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import { useAppShellStore } from '@/stores/appShellStore'
import { useCapability } from '@/lib/capabilities/capabilities'
import { usePageScope } from '@/lib/page-scope/usePageScope'
import {
  EXPLORER_URL_KEYS,
  decodeExplorerScope,
  encodeExplorerScope,
  explorerScopeToFilterSpec,
  type EncodedExplorerScope,
  type ExplorerScope,
} from './explorerScope'

import { ExplorerMatchesMode } from './ExplorerPage.matchesMode'
import { ExplorerPlayerMode } from './ExplorerPage.playerMode'
import { buildExplorerFilterOptions } from './ExplorerPage.filterOptions'

type SearchMode = 'matches' | 'player'

/** Debounce de l'input match-ID avant de piloter la query (E2 revue 2026-07) :
 * 1 POST après la rafale de frappe, pas un recompute serveur par caractère. */
const MATCH_ID_DEBOUNCE_MS = 250

/** Clés à valeur Set<string> du scope — typage du toggle. */
type ExplorerSetKey =
  | 'expTypes'
  | 'playlists'
  | 'mapNames'
  | 'modeNames'
  | 'perfTiers'
  | 'skillTiers'
  | 'outcomeFilter'

export function ExplorerPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const navigate = useNavigate()
  const titleSlug = useTitleSlug()
  const search = useSearch({ from: '/{-$lang}/t/$titleSlug/players/$playerSlug/explorer/' }) as {
    mode?: SearchMode
    target?: string
    targetXuid?: string
  }

  const locale = useAppShellStore((s) => s.locale)

  const t = (key: ExplorerManifestKey, values?: Record<string, string | number>) =>
    formatMessage(explorerManifest, key, locale, values)

  const [mode, setMode] = useState<SearchMode>(search.mode ?? 'matches')
  const [targetGamertag, setTargetGamertag] = useState(search.target ?? '')
  // xuid optionnel (transmis par le Classement) : court-circuite la résolution
  // gamertag→xuid locale côté backend. Vidé dès qu'une recherche manuelle (par
  // gamertag) est lancée.
  const [targetXuid, setTargetXuid] = useState(search.targetXuid ?? '')

  // ─── Scope filtres (URL = source de vérité + miroir localStorage) ────────────
  const scopeParams = useMemo(() => ({ titleSlug, playerSlug }), [titleSlug, playerSlug])
  const { scope, setScope, reset: resetFilters } = usePageScope<
    ExplorerScope,
    EncodedExplorerScope
  >({
    to: '/{-$lang}/t/$titleSlug/players/$playerSlug/explorer',
    params: scopeParams,
    storageKey: `levelup-explorer-scope:${playerSlug}`,
    encode: encodeExplorerScope,
    decode: decodeExplorerScope,
    urlKeys: EXPLORER_URL_KEYS,
  })

  const {
    startDate,
    endDate,
    squadScope,
    replayScope: replayScopeMemorise,
    matchIDSearch,
    expTypes,
    playlists,
    mapNames,
    modeNames,
    perfTiers,
    skillTiers,
    outcomeFilter,
  } = scope

  // LE FILTRE « Avec rejeu / Sans rejeu » EST NEUTRALISE SUR UN TITRE SANS `replay`
  // (revue C-R1, constat C5). Le masquer ne suffisait pas : la portee est memorisee dans
  // `levelup-explorer-scope:{playerSlug}`, une cle scopee par JOUEUR et non par titre. Poser
  // « Avec rejeu » sur halo_infinite puis basculer sur halo_5 reinjectait donc `replay=with`
  // au chargement — liste filtree a zero match, contröle invisible, et rien pour le corriger
  // sinon tout effacer. Meme chose pour une URL portant `?replay=with`.
  //
  // La neutralisation est ICI, au point de LECTURE, et pas au montage du <select> : c'est le
  // seul endroit qui couvre a la fois le miroir, l'URL, la charge utile envoyee au backend et
  // le bandeau « filtres actifs ». La valeur memorisee n'est pas effacee — elle redevient
  // active telle quelle si le joueur repasse sur un titre qui a le rejeu.
  const hasReplayCapability = useCapability('replay')
  const replayScope = hasReplayCapability ? replayScopeMemorise : ''

  // saisonOpen reste local : pur état d'ouverture de dropdown (pas du scope).
  const [saisonOpen, setSaisonOpen] = useState(false)

  /** Toggle d'une valeur dans un Set du scope (immuable). */
  function toggleScopeSet(key: ExplorerSetKey, value: string) {
    const next = new Set(scope[key])
    if (next.has(value)) next.delete(value)
    else next.add(value)
    setScope({ [key]: next } as Partial<ExplorerScope>)
  }

  function handleStartDate(v: string) {
    const patch: Partial<ExplorerScope> = { startDate: v }
    // Si la borne haute devient antérieure à la borne basse, on la réinitialise.
    if (endDate && v && endDate < v) patch.endDate = ''
    setScope(patch)
  }

  // Saisons : dérivées du catalog du titre courant. activeSeason est calculée
  // depuis les inputs date locaux (Du/Au), pas du filterContext shell — Explorer
  // override ce dernier (vue tout-historique) donc la saison agit comme un
  // raccourci sur les dates locales.
  const { seasons, activeSeason } = useActiveSeason({
    start_date: startDate || null,
    end_date: endDate || null,
  })

  // ranked_context auto-déduit du multi-select Type d'expérience.
  // Sélection mono-valeur "PVP classé" → "ranked" (gate skill_tier sur CSR).
  // Sélection mono-valeur "PVP non classé" → "unranked" (gate skill_tier sur LUSR).
  // Toute autre combinaison (multi-valeurs, PVE seul, vide) → "" : skill_tier
  // resté désactivé pour éviter le mélange CSR/LUSR ambigu.
  // Cf. thought_log 2026-05-09 P3 — fusion du single-select "Expérience" dans
  // le multi-select "Type d'expérience" (Option A).
  // CONTRAT (GH6-1) : 'PVP classé'/'PVP non classé' sont les VALUES FR canoniques du
  // filtre expérience (identiques en FR et EN — seul le Label affiché est localisé côté
  // backend). Ne JAMAIS comparer sur un libellé localisé ici.
  const rankedContext: 'ranked' | 'unranked' | '' = (() => {
    if (expTypes.size !== 1) return ''
    if (expTypes.has('PVP classé')) return 'ranked'
    if (expTypes.has('PVP non classé')) return 'unranked'
    return ''
  })()

  // Quand la dérivation change, le skill_tier doit être réinitialisé pour
  // éviter de garder un tier CSR sélectionné après bascule en non-classé.
  // Guard sur la taille pour ne pas déclencher une navigation à vide en boucle.
  useEffect(() => {
    if (rankedContext === '' && skillTiers.size > 0) setScope({ skillTiers: new Set<string>() })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [rankedContext])

  // ─── URL sync mode/target ────────────────────────────────────────────────────
  // Init unique depuis l'URL au mount — légitime pour hydrater l'état initial.
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- hydratation unique de l'état depuis l'URL au montage (init légitime) (2026-07-22)
    if (search.mode) setMode(search.mode)
    if (search.target) setTargetGamertag(search.target)
    if (search.targetXuid) setTargetXuid(search.targetXuid)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // `replace: true` : ces deux navigate() ne font que refléter l'état local
  // (mode/target) dans l'URL — pas une navigation de page. Sans replace, chaque
  // sélection de cible ou changement de mode empilait l'historique et déclenchait
  // inutilement le pipeline de transition de route (TopProgressBar) (V72-24).
  function setModeAndUrl(m: SearchMode) {
    setMode(m)
    void navigate({
      to: '/{-$lang}/t/$titleSlug/players/$playerSlug/explorer',
      params: { titleSlug, playerSlug },
      search: (prev) => ({ ...prev, mode: m }),
      replace: true,
    })
  }

  function selectTarget(gamertag: string) {
    setTargetGamertag(gamertag)
    // Recherche manuelle par gamertag → pas de xuid connu : on repasse par la
    // résolution locale côté backend (et on purge un éventuel xuid hérité du Classement).
    setTargetXuid('')
    void navigate({
      to: '/{-$lang}/t/$titleSlug/players/$playerSlug/explorer',
      params: { titleSlug, playerSlug },
      search: (prev) => ({ ...prev, mode: 'player', target: gamertag, targetXuid: undefined }),
      replace: true,
    })
  }

  function openHeadToHead(gamertag: string) {
    void navigate({
      to: '/{-$lang}/t/$titleSlug/players/$playerSlug/community/compare',
      params: { titleSlug, playerSlug },
      search: { target: gamertag, from: 'explorer' },
    })
  }

  // ─── Queries ───────────────────────────────────────────────────────────────
  // Explorer = vue historique complète, 100% locale. On part de DEFAULT_FILTER_CONTEXT
  // (aucun héritage du store global solo/squad) et on pilote le scope via les
  // filtres locaux date/exp/playlist/etc. ci-dessus. pageSize=200 = max accepté
  // par maxPageSize backend ; pagination client gère le découpage 20/page.
  const explorerFilterContext = DEFAULT_FILTER_CONTEXT
  // Hash constant : le scope global n'influence plus la query. Les variations
  // locales (perfTiers/skillTiers/dates/etc.) sont déjà dans la queryKey.
  const filterContextHash = 'explorer-local'
  // E2 : l'input match-ID est débouncé avant d'entrer dans la query (1 POST après
  // la rafale, pas un par caractère). L'URL/scope, lui, reste mis à jour à chaque
  // frappe (input contrôlé) — seul le déclenchement réseau est retardé.
  const debouncedMatchIDSearch = useDebounced(matchIDSearch, MATCH_ID_DEBOUNCE_MS)
  const matchesQuery = useExplorerMatches(
    playerSlug,
    {
      filters: explorerFilterContext,
      pagination: { page: 1, page_size: 10000 },
      include_export_hint: true,
      // E2 : le briefing serveur n'est calculé qu'en mode Matchs (seul mode qui le
      // consomme). En mode Joueur la query est de toute façon désactivée ci-dessous.
      include_briefing: mode === 'matches',
      perf_tiers: perfTiers.size > 0 ? [...perfTiers].map(Number) : undefined,
      skill_tiers: skillTiers.size > 0 ? [...skillTiers] : undefined,
      ranked_context: rankedContext || undefined,
      outcome_filter: outcomeFilter.size > 0 ? [...outcomeFilter].map(Number) : undefined,
      // Tri désormais CLIENT (en-têtes du tableau) : on n'envoie plus sort_field/
      // sort_dir → le backend renvoie son ordre par défaut (les 10000 plus récents),
      // exactement le cap voulu. Cf. thought_log 2026-07-17.
      match_start_date: startDate || null,
      match_end_date: endDate || null,
      experience_types: expTypes.size > 0 ? [...expTypes] : undefined,
      playlists: playlists.size > 0 ? [...playlists] : undefined,
      map_names: mapNames.size > 0 ? [...mapNames] : undefined,
      mode_names: modeNames.size > 0 ? [...modeNames] : undefined,
      squad_scope: squadScope || undefined,
      replay_scope: replayScope || undefined,
      match_id_search: debouncedMatchIDSearch || undefined,
    },
    filterContextHash,
    // E2 : la query n'est lancée qu'en mode Matchs — en mode Joueur son résultat
    // (summary/briefing) n'est pas consommé, donc aucun recompute serveur inutile.
    mode === 'matches',
  )

  const playerQuery = useExplorerPlayer(playerSlug, {
    target_gamertag: targetGamertag,
    target_xuid: targetXuid || undefined,
  })

  // Mode Joueur : extraction des match_ids communs séparés par rôle
  // (ally vs enemy) depuis la réponse player-query, pour piloter les 2
  // tableaux scopés ci-dessous.
  const allyMatchIds = (playerQuery.data?.common_matches ?? [])
    .filter((m) => m.were_teammates)
    .map((m) => m.match_id)
  const enemyMatchIds = (playerQuery.data?.common_matches ?? [])
    .filter((m) => !m.were_teammates)
    .map((m) => m.match_id)

  // Requête tableau "matchs en allié" — réutilise le pipeline matches-query
  // avec un filtre match_ids (whitelist). Activée uniquement quand on a des
  // match_ids ET qu'on est en mode Joueur.
  const allyMatchesQuery = useExplorerMatches(
    playerSlug,
    {
      filters: explorerFilterContext,
      pagination: { page: 1, page_size: 10000 },
      sort_field: 'start_time',
      sort_dir: 'desc',
      match_ids: allyMatchIds,
    },
    filterContextHash,
    mode === 'player' && allyMatchIds.length > 0,
  )

  const enemyMatchesQuery = useExplorerMatches(
    playerSlug,
    {
      filters: explorerFilterContext,
      pagination: { page: 1, page_size: 10000 },
      sort_field: 'start_time',
      sort_dir: 'desc',
      match_ids: enemyMatchIds,
    },
    filterContextHash,
    mode === 'player' && enemyMatchIds.length > 0,
  )

  const summary = matchesQuery.data?.summary

  // Descriptor du contexte de navigation pour les matchs ouverts depuis le
  // tableau mode Matchs. Priorité au filtre le plus spécifique : 1 playlist >
  // 1 mode > période active > undefined (Q25 fallback générique côté match-view).
  const matchesContextDescriptor: ContextDescriptor | undefined = (() => {
    if (playlists.size === 1) {
      const [name] = [...playlists]
      return name ? { kind: 'playlist', name } : undefined
    }
    if (modeNames.size === 1) {
      const [category] = [...modeNames]
      return category ? { kind: 'mode', category } : undefined
    }
    if (startDate || endDate) {
      const toIso = (d: string) => (d ? new Date(d).toISOString() : undefined)
      return { kind: 'period', from: toIso(startDate), to: toIso(endDate) }
    }
    return undefined
  })()

  // Phase 4 : filterSpec dérivé des filtres Explorer locaux (scope) pour piloter
  // la nav contextuelle prev/next depuis Explorer (multi-playlist/mode supportés
  // par le backend Q25 depuis la Phase 3). Remplace la dérivation soloFilterStore.
  const matchesFilterSpec = explorerScopeToFilterSpec(scope)

  // ─── Options pour les MultiSelectFilter (extrait dans helper) ────────────
  const {
    expTypeOptions,
    playlistOptions,
    modeOptions,
    mapOptions,
    perfTierOptions,
    outcomeOptions,
    skillTierOptions,
    squadCountByValue,
  } = buildExplorerFilterOptions(summary, t)

  const hasActiveFilter =
    !!startDate ||
    !!endDate ||
    !!squadScope ||
    !!replayScope ||
    // `.trim()` : le backend ignore les blancs d'une recherche par match ID (un GUID n'en
    // porte aucun), donc une saisie qui s'y réduit ne filtre RIEN — annoncer « filtres actifs »
    // pour elle proposerait d'effacer un filtre qui n'existe pas.
    !!matchIDSearch.trim() ||
    expTypes.size > 0 ||
    playlists.size > 0 ||
    mapNames.size > 0 ||
    modeNames.size > 0 ||
    perfTiers.size > 0 ||
    skillTiers.size > 0 ||
    outcomeFilter.size > 0

  return (
    <div className="flex flex-col">
      <div className="p-6 space-y-6">
        {/* Onglets mode */}
        <div className="flex gap-2">
          <Button
            variant={mode === 'matches' ? 'default' : 'outline'}
            size="sm"
            onClick={() => setModeAndUrl('matches')}
          >
            {t('explorer.mode.matches')}
          </Button>
          <Button
            variant={mode === 'player' ? 'default' : 'outline'}
            size="sm"
            onClick={() => setModeAndUrl('player')}
          >
            {t('explorer.mode.player')}
          </Button>
        </div>

        {mode === 'player' && (
          <ExplorerPlayerMode
            playerSlug={playerSlug}
            targetGamertag={targetGamertag}
            locale={locale}
            t={t}
            playerQuery={playerQuery}
            allyMatchIds={allyMatchIds}
            enemyMatchIds={enemyMatchIds}
            allyMatchesData={allyMatchesQuery.data}
            enemyMatchesData={enemyMatchesQuery.data}
            onSelectTarget={selectTarget}
            onOpenHeadToHead={openHeadToHead}
          />
        )}

        {mode === 'matches' && (
          <ExplorerMatchesMode
            playerSlug={playerSlug}
            t={t}
            startDate={startDate}
            endDate={endDate}
            matchIDSearch={matchIDSearch}
            squadScope={squadScope}
            replayScope={replayScope}
            squadCountByValue={squadCountByValue}
            onStartDateChange={handleStartDate}
            onEndDateChange={(v) => setScope({ endDate: v })}
            onMatchIDSearchChange={(v) => setScope({ matchIDSearch: v })}
            onSquadScopeChange={(v) => setScope({ squadScope: v })}
            onReplayScopeChange={(v) => setScope({ replayScope: v })}
            seasons={seasons}
            activeSeason={activeSeason}
            saisonOpen={saisonOpen}
            onSaisonToggle={() => setSaisonOpen((o) => !o)}
            onSaisonClose={() => setSaisonOpen(false)}
            onSelectSeason={(s) => {
              const p = seasonToPeriod(s)
              setScope({ startDate: p.start_date ?? '', endDate: p.end_date ?? '' })
              setSaisonOpen(false)
            }}
            onClearPeriod={() => setScope({ startDate: '', endDate: '' })}
            expTypes={expTypes}
            playlists={playlists}
            modeNames={modeNames}
            mapNames={mapNames}
            outcomeFilter={outcomeFilter}
            perfTiers={perfTiers}
            skillTiers={skillTiers}
            expTypeOptions={expTypeOptions}
            playlistOptions={playlistOptions}
            modeOptions={modeOptions}
            mapOptions={mapOptions}
            outcomeOptions={outcomeOptions}
            perfTierOptions={perfTierOptions}
            skillTierOptions={skillTierOptions}
            rankedContext={rankedContext}
            onToggleExpType={(v) => toggleScopeSet('expTypes', v)}
            onTogglePlaylist={(v) => toggleScopeSet('playlists', v)}
            onToggleModeName={(v) => toggleScopeSet('modeNames', v)}
            onToggleMapName={(v) => toggleScopeSet('mapNames', v)}
            onToggleOutcome={(v) => toggleScopeSet('outcomeFilter', v)}
            onTogglePerfTier={(v) => toggleScopeSet('perfTiers', v)}
            onToggleSkillTier={(v) => toggleScopeSet('skillTiers', v)}
            hasActiveFilter={hasActiveFilter}
            onResetFilters={resetFilters}
            matchesQuery={matchesQuery}
            matchesContextDescriptor={matchesContextDescriptor}
            matchesFilterSpec={matchesFilterSpec}
          />
        )}
      </div>
    </div>
  )
}
