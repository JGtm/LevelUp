/**
 * CareerTopEncountersSection — "Joueurs les plus croisés (hors amis)".
 *
 * Réutilise MatchEncountersTable (même tableau que Match View > "Historique
 * de rencontre") avec deux overrides :
 *  - hideCardWrapper : pas de barre de titre dupliquée (le h2 de section suffit)
 *  - onPlayerClick : navigation vers Explorer mode joueur (au lieu du
 *    fallback navigation interne qui n'a pas de sens hors match-view)
 *
 * Limit 10 résultats, amis configurés (FriendGamertags) exclus côté backend.
 */
import { useNavigate, useParams } from '@tanstack/react-router'
import { useTitleSlug } from '@/lib/title-routing'
import { MatchEncountersTable } from '@/features/match-view/MatchEncountersTable'
import { Spinner } from '@/components/ui/spinner'
import { useCareerTopEncounters } from './queries'
import { careerManifest } from '@/lib/i18n/generated/career'
import type { ManifestLocale } from '@/lib/i18n/format'
import { useAppShellStore } from '@/stores/appShellStore'

export function CareerTopEncountersSection() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const locale = useAppShellStore((s) => s.locale) as ManifestLocale
  const t = (key: keyof typeof careerManifest) => careerManifest[key][locale]
  const navigate = useNavigate()
  const titleSlug = useTitleSlug()

  const { data, isLoading, isError } = useCareerTopEncounters(playerSlug)

  const handlePlayerClick = (gamertag: string) => {
    void navigate({
      to: '/{-$lang}/t/$titleSlug/players/$playerSlug/explorer',
      params: { titleSlug, playerSlug },
      search: { mode: 'player', target: gamertag },
    })
  }

  return (
    <section className="space-y-3">
      <div className="flex items-baseline gap-3">
        <h2 className="text-sm font-semibold">{t('career.top_encounters.section_title')}</h2>
        <span className="text-xs text-muted-foreground">{t('career.top_encounters.section_hint')}</span>
      </div>

      {isLoading && (
        <div className="flex h-24 items-center justify-center">
          <Spinner size="md" />
        </div>
      )}
      {isError && <p className="text-sm text-destructive">{t('career.errors.load_progression_failed')}</p>}
      {!isLoading && !isError && (
        <MatchEncountersTable
          rows={data?.items ?? []}
          locale={locale === 'en' ? 'en' : 'fr'}
          onPlayerClick={handlePlayerClick}
          hideCardWrapper
        />
      )}
    </section>
  )
}
