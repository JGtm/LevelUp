/**
 * HomePage — Accueil Mission Control (Slice 5).
 */
import { useParams } from '@tanstack/react-router'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { EmptyStateCard, EmptyStateNotice } from '@/components/ui/empty-state'
import { PrivacyBanner } from '@/components/ui/privacy-banner'
import { Spinner } from '@/components/ui/spinner'
import { MatchCard } from '@/components/ui/match-card'
import { RecentMediaRail } from './RecentMediaRail'
import { useHomePage, useBattlePass, useChallenges } from './queries'

function KPICard({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="rounded-lg border border-border bg-muted px-4 py-3 text-center">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="text-xl font-bold text-primary">{value}</p>
    </div>
  )
}

export function HomePage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const { data, isLoading, isError, refetch } = useHomePage(playerSlug)
  const { data: bp } = useBattlePass(playerSlug)
  const { data: challenges } = useChallenges(playerSlug)

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
  const recentMedia = data.recent_media ?? []
  const soloSession = data.solo_session ?? null
  const squadSession = data.squad_session ?? null

  return (
    <div className="flex flex-col">
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
              <KPICard label="Taux de victoire" value={`${(hero.kpis.win_rate * 100).toFixed(0)}%`} />
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
                  <p className="text-sm text-foreground">Rang <strong className="text-primary">{bp.rank}</strong></p>
                  {bp.progress != null && (
                    <div className="h-2 w-full rounded-full bg-muted">
                      <div className="h-full rounded-full bg-primary" style={{ width: `${bp.progress}%` }} />
                    </div>
                  )}
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">Non disponible ({bp?.error_hint ?? 'live API non configurée'})</p>
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
                  <p className="text-sm text-foreground">
                    <strong className="text-primary">{challenges.completed}</strong> / {challenges.total} complétés
                  </p>
                  {challenges.xp_available && (
                    <p className="text-xs text-muted-foreground">{challenges.xp_available.toLocaleString('fr-FR')} XP disponibles</p>
                  )}
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">Non disponible</p>
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
                  <div className="rounded-md bg-muted p-3">
                    <Badge variant="secondary" className="mb-2">Solo</Badge>
                    <p className="text-sm font-medium">{soloSession.session_label}</p>
                    <p className="text-xs text-muted-foreground">{soloSession.match_count} parties · {(soloSession.win_rate * 100).toFixed(0)}% W</p>
                  </div>
                )}
                {squadSession && (
                  <div className="rounded-md bg-primary/10 p-3">
                    <Badge variant="default" className="mb-2">Escouade</Badge>
                    <p className="text-sm font-medium">{squadSession.session_label}</p>
                    <p className="text-xs text-muted-foreground">{squadSession.match_count} parties · {(squadSession.win_rate * 100).toFixed(0)}% W</p>
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
                  <div key={i} className="rounded-md border border-border p-3">
                    <p className="text-xs font-medium text-muted-foreground">{h.title}</p>
                    <p className="text-base font-bold text-primary">{h.value}</p>
                    <p className="text-xs text-muted-foreground">{h.detail}</p>
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

        {/* Matchs récents — 4 tuiles MatchCard */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Matchs récents</CardTitle>
          </CardHeader>
          <CardContent>
            {recentMatches.length > 0 ? (
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
                {recentMatches.slice(0, 4).map((m) => (
                  <MatchCard key={m.match_id} match={m} />
                ))}
              </div>
            ) : (
              <EmptyStateNotice
                title="Aucun match récent disponible"
                description="Les matchs récents apparaîtront ici après une synchronisation."
              />
            )}
          </CardContent>
        </Card>

        <RecentMediaRail playerSlug={playerSlug} items={recentMedia} />
      </div>
    </div>
  )
}
