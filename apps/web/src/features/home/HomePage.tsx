/**
 * HomePage — Accueil Mission Control (Slice 5).
 */
import { useParams } from '@tanstack/react-router'
import { Link } from '@tanstack/react-router'
import { PageHeader } from '@/components/shell/PageHeader'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { EmptyStateCard, EmptyStateNotice } from '@/components/ui/empty-state'
import { PrivacyBanner } from '@/components/ui/privacy-banner'
import { Spinner } from '@/components/ui/spinner'
import { RecentMediaRail } from './RecentMediaRail'
import { useHomePage, useBattlePass, useChallenges } from './queries'

function KPICard({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="rounded-lg border border-gray-100 bg-gray-50 px-4 py-3 text-center">
      <p className="text-xs text-gray-500">{label}</p>
      <p className="text-xl font-bold text-purple-700">{value}</p>
    </div>
  )
}

export function HomePage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const { data, isLoading, isError, refetch } = useHomePage(playerSlug)
  const { data: bp } = useBattlePass(playerSlug)
  const { data: challenges } = useChallenges(playerSlug)

  const quickActions = [
    {
      to: '/players/$playerSlug/stats/timeseries',
      label: 'Statistiques',
      icon: '📈',
    },
    {
      to: '/players/$playerSlug/squad',
      label: 'Escouade',
      icon: '👥',
    },
    {
      to: '/players/$playerSlug/explorer',
      label: 'Explorer',
      icon: '🔍',
    },
    {
      to: '/players/$playerSlug/media',
      label: 'Médias',
      icon: '🖼',
    },
  ] as const

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <Spinner size="lg" label="Chargement de l'accueil…" />
      </div>
    )
  }

  if (isError) {
    return (
      <div className="flex flex-col">
        <PageHeader title="Mission Control" subtitle="Accueil joueur" />
        <div className="p-6">
          <EmptyStateCard
            title="Accueil indisponible"
            description="La page d'accueil n'a pas pu être chargée pour ce joueur. Vérifie la session ou relance la requête."
            actionLabel="Réessayer"
            onAction={() => refetch()}
          />
        </div>
      </div>
    )
  }

  if (!data) {
    return (
      <div className="flex flex-col">
        <PageHeader title="Mission Control" subtitle="Accueil joueur" />
        <div className="p-6">
          <EmptyStateCard
            title="Accueil vide"
            description="Aucune donnée d'accueil n'a été renvoyée pour ce joueur. Vérifie le bootstrap ou les données locales avant de continuer."
            actionLabel="Relancer"
            onAction={() => refetch()}
          />
        </div>
      </div>
    )
  }

  const { hero } = data
  const highlights = data.highlights ?? []
  const recentMatches = data.recent_matches ?? []
  const soloSession = data.solo_session ?? null
  const squadSession = data.squad_session ?? null

  return (
    <div className="flex flex-col">
      <PageHeader title={`Bienvenue, ${hero.player_name}`} subtitle="Mission Control" />

      <div className="space-y-6 p-6">
        {/* B9 : signal discret si données partielles (compte privé) */}
        <PrivacyBanner warning={data.privacy_warning} />

        {/* Hero KPIs */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Performance globale</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
              <KPICard label="Parties" value={hero.kpis.total_matches.toLocaleString('fr-FR')} />
              <KPICard label="Win Rate" value={`${(hero.kpis.win_rate * 100).toFixed(0)}%`} />
              <KPICard label="K/D" value={hero.kpis.global_ratio?.toFixed(2) ?? '—'} />
              <KPICard label="Victoires" value={hero.kpis.wins.toLocaleString('fr-FR')} />
              <KPICard label="Défaites" value={hero.kpis.losses.toLocaleString('fr-FR')} />
              <KPICard label="Précision" value={hero.kpis.avg_accuracy != null ? `${hero.kpis.avg_accuracy.toFixed(0)}%` : '—'} />
            </div>
          </CardContent>
        </Card>

        {/* Battle Pass + Défis */}
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Battle Pass</CardTitle>
            </CardHeader>
            <CardContent>
              {bp?.available ? (
                <div className="space-y-1">
                  <p className="text-sm text-gray-700">Rang <strong className="text-purple-700">{bp.rank}</strong></p>
                  {bp.progress != null && (
                    <div className="h-2 w-full rounded-full bg-gray-100">
                      <div className="h-full rounded-full bg-purple-500" style={{ width: `${bp.progress}%` }} />
                    </div>
                  )}
                </div>
              ) : (
                <p className="text-sm text-gray-400">Non disponible ({bp?.error_hint ?? 'live API non configurée'})</p>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Défis actifs</CardTitle>
            </CardHeader>
            <CardContent>
              {challenges?.available ? (
                <div className="space-y-1">
                  <p className="text-sm text-gray-700">
                    <strong className="text-purple-700">{challenges.completed}</strong> / {challenges.total} complétés
                  </p>
                  {challenges.xp_available && (
                    <p className="text-xs text-gray-400">{challenges.xp_available.toLocaleString('fr-FR')} XP disponibles</p>
                  )}
                </div>
              ) : (
                <p className="text-sm text-gray-400">Non disponible</p>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Sessions récentes */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Sessions récentes</CardTitle>
          </CardHeader>
          <CardContent>
            {soloSession || squadSession ? (
              <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                {soloSession && (
                  <div className="rounded-md bg-gray-50 p-3">
                    <Badge variant="secondary" className="mb-2">Solo</Badge>
                    <p className="text-sm font-medium">{soloSession.session_label}</p>
                    <p className="text-xs text-gray-500">{soloSession.match_count} parties · {(soloSession.win_rate * 100).toFixed(0)}% W</p>
                  </div>
                )}
                {squadSession && (
                  <div className="rounded-md bg-purple-50 p-3">
                    <Badge variant="default" className="mb-2">Escouade</Badge>
                    <p className="text-sm font-medium">{squadSession.session_label}</p>
                    <p className="text-xs text-gray-500">{squadSession.match_count} parties · {(squadSession.win_rate * 100).toFixed(0)}% W</p>
                  </div>
                )}
              </div>
            ) : (
              <EmptyStateNotice
                title="Aucune session récente disponible"
                description="Aucune session solo ou escouade n'a été calculée pour le scope actuel."
              />
            )}
          </CardContent>
        </Card>

        {/* Highlights */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Points saillants</CardTitle>
          </CardHeader>
          <CardContent>
            {highlights.length > 0 ? (
              <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
                {highlights.map((h, i) => (
                  <div key={i} className="rounded-md border border-gray-100 p-3">
                    <p className="text-xs font-medium text-gray-500">{h.title}</p>
                    <p className="text-base font-bold text-purple-700">{h.value}</p>
                    <p className="text-xs text-gray-400">{h.detail}</p>
                  </div>
                ))}
              </div>
            ) : (
              <EmptyStateNotice
                title="Aucun point saillant disponible"
                description="Le backend n'a renvoyé aucun highlight exploitable pour cette période."
              />
            )}
          </CardContent>
        </Card>

        {/* Matchs récents */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Matchs récents</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-1">
              {recentMatches.slice(0, 8).map((m) => (
                <div key={m.match_id} className="flex items-center justify-between rounded-md px-3 py-2 hover:bg-gray-50">
                  <div>
                    <p className="text-sm font-medium text-gray-800">{m.title}</p>
                    <p className="text-xs text-gray-400">{m.detail}</p>
                  </div>
                  <Badge
                    variant={
                      m.outcome_tone === 'win' ? 'success' :
                      m.outcome_tone === 'loss' ? 'destructive' : 'secondary'
                    }
                  >
                    {m.outcome_label}
                  </Badge>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Quick Actions */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Accès rapide</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
              {quickActions.map((item) => (
                <Link
                  key={item.to}
                  to={item.to}
                  params={{ playerSlug }}
                  className="flex flex-col items-center justify-center gap-1 rounded-lg border border-gray-200 bg-gray-50 px-4 py-4 text-center hover:bg-purple-50 hover:border-purple-300 transition-colors"
                >
                  <span className="text-2xl">{item.icon}</span>
                  <span className="text-xs font-medium text-gray-700">{item.label}</span>
                </Link>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Dernier match — lien rapide */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Dernier match</CardTitle>
          </CardHeader>
          <CardContent>
            {recentMatches.length > 0 ? (
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-gray-800">{recentMatches[0].title}</p>
                  <p className="text-xs text-gray-400">{recentMatches[0].detail}</p>
                </div>
                <Link
                  to="/players/$playerSlug/last-match"
                  params={{ playerSlug }}
                  className="ml-4 inline-flex items-center gap-1 rounded-md bg-purple-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-purple-700 transition-colors"
                >
                  Voir →
                </Link>
              </div>
            ) : (
              <p className="text-sm text-gray-400">Aucun match récent.</p>
            )}
          </CardContent>
        </Card>

        <RecentMediaRail playerSlug={playerSlug} />
      </div>
    </div>
  )
}
