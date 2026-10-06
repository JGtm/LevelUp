/**
 * TendancesTab — l'onglet « Tendances » d'Ascension (vues Solo et Escouade).
 *
 * Ordre de la page : la bascule de vue et le filtre « Type de partie » (plus, en vue Escouade,
 * le sélecteur d'escouade), la matrice par mois et par horizon, la barre « Horizon », la section
 * « Évolution », puis, en vue Solo seulement, le calendrier, « Victoires et défaites » et, côte
 * à côte, les médailles et les types de partie. L'horizon et le pas vivent ici : le pas d'une
 * série revient à celui de l'horizon dès que l'horizon change. Changer d'horizon ou de pas ne
 * relit rien ; changer de vue, de type de partie ou d'escouade refait la requête (ils sont dans
 * la clé). Changer de vue GARDE le type de partie, l'horizon et le pas.
 *
 * `TendancesTab` est une enveloppe : elle remonte le composant intérieur pour chaque couple
 * (titre, joueur), de sorte que le type de partie, la vue, l'horizon et le pas repartent à zéro
 * pour un autre joueur ou un autre titre. La sélection d'escouade, elle, est celle de la page
 * Escouade (même stockage par joueur, cf. `useTendancesSquadSelection`).
 */
import { useMemo, useState } from 'react'
import { useParams } from '@tanstack/react-router'

import type { TrendsMember, TrendsQueryRequest } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import { useAppShellStore } from '@/stores/appShellStore'

import { useTrendsPage } from './queries'
import {
  DEFAULT_HORIZON,
  defaultStep,
  gameTypeOptions,
  type Horizon,
  type Step,
  type TendancesView,
} from './tendances.logic'
import { TendancesBody } from './TendancesBody'
import { TendancesControls } from './TendancesControls'
import { useTendancesSquadSelection } from './useTendancesSquadSelection'

/** Aucun membre : référence stable tant que la réponse n'est pas là. */
const NO_MEMBERS: readonly TrendsMember[] = []

/** Le corps de la requête d'une vue (le corps Solo ne porte ni composition ni option). */
function buildRequest(
  view: TendancesView,
  gameType: string,
  locale: Locale,
  squadGamertags: string[],
  exactComposition: boolean,
): TrendsQueryRequest {
  if (view === 'solo') return { view: 'solo', game_type: gameType, locale }
  return {
    view: 'squad',
    game_type: gameType,
    selected_gamertags: squadGamertags,
    exact_composition: exactComposition,
    locale,
  }
}

function TendancesPage({ playerSlug }: { playerSlug: string }) {
  const locale = useAppShellStore((s) => s.locale)
  const [view, setView] = useState<TendancesView>('solo')
  const [gameType, setGameType] = useState('')
  const [horizon, setHorizon] = useState<Horizon>(DEFAULT_HORIZON)
  const [step, setStep] = useState<Step>(defaultStep(DEFAULT_HORIZON))
  const squad = useTendancesSquadSelection(playerSlug)
  const { squadGamertags, exactComposition } = squad

  const request = useMemo(
    () => buildRequest(view, gameType, locale, squadGamertags, exactComposition),
    [view, gameType, locale, squadGamertags, exactComposition],
  )
  const { data, isLoading, isError, refetch } = useTrendsPage(playerSlug, request)

  const onHorizonChange = (next: Horizon) => {
    setHorizon(next)
    setStep(defaultStep(next))
  }

  const squadWithoutTeammate = view === 'squad' && squadGamertags.length === 0
  const offered = (squadWithoutTeammate ? undefined : data?.game_types) ?? []

  return (
    <div className="space-y-6">
      <TendancesControls
        playerSlug={playerSlug}
        locale={locale}
        view={view}
        onViewChange={setView}
        gameType={gameType}
        gameTypeKeys={gameTypeOptions(
          offered.map((g) => g.key),
          gameType,
        )}
        onGameTypeChange={setGameType}
        squad={squad}
        members={data?.members ?? NO_MEMBERS}
      />
      <TendancesBody
        locale={locale}
        view={view}
        squadWithoutTeammate={squadWithoutTeammate}
        data={data}
        isLoading={isLoading}
        isError={isError}
        onRetry={() => refetch()}
        horizon={horizon}
        step={step}
        onHorizonChange={onHorizonChange}
        onStepChange={setStep}
      />
    </div>
  )
}

export function TendancesTab() {
  const params = useParams({ strict: false }) as { playerSlug?: string }
  const playerSlug = params.playerSlug ?? ''
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  if (!playerSlug) return null
  return <TendancesPage key={`${titleSlug}:${playerSlug}`} playerSlug={playerSlug} />
}
