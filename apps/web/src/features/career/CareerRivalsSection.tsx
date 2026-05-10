/**
 * CareerRivalsSection — Top némésis + Top souffre-douleur côte à côte.
 *
 * Format : tableau custom 6 colonnes (#, Joueur, Frags, Morts, Ratio, Matchs).
 * Données issues de shared.killer_victim_pairs au niveau global :
 *  - Némésis  : tri par deaths DESC (joueurs qui m'ont le plus tué)
 *  - Souffre-douleur : tri par frags DESC (joueurs que j'ai le plus tué)
 *
 * Pas de seuil min — TOP 10 brut. Clic sur le gamertag → Explorer mode joueur.
 * Style aligné sur le pattern table de MatchEncountersTable (border-collapse,
 * border-2 + cells border, header text-xs muted).
 */
import { useNavigate, useParams } from '@tanstack/react-router'
import { Spinner } from '@/components/ui/spinner'
import { useCareerRivals } from './queries'
import { careerManifest } from '@/lib/i18n/generated/career'
import type { ManifestLocale } from '@/lib/i18n/format'
import { useAppShellStore } from '@/stores/appShellStore'
import { tokenCssVar } from '@/lib/accessibility'
import type { CareerRival } from '@/lib/api/types'

function formatRatio(r: number): string {
  if (!Number.isFinite(r)) return '∞'
  return r.toFixed(2)
}

function ratioColor(deaths: number, ratio: number): string | undefined {
  if (deaths === 0) return ratio > 0 ? tokenCssVar('outcome-win') : undefined
  if (ratio > 1) return tokenCssVar('outcome-win')
  if (ratio < 1) return tokenCssVar('outcome-loss')
  return tokenCssVar('outcome-draw')
}

interface RivalsTableProps {
  title: string
  rows: CareerRival[]
  emptyLabel: string
  labels: {
    rank: string
    player: string
    frags: string
    deaths: string
    ratio: string
    matches: string
  }
  onPlayerClick: (gamertag: string) => void
}

function RivalsTable({ title, rows, emptyLabel, labels, onPlayerClick }: RivalsTableProps) {
  return (
    <div className="space-y-2">
      <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">{title}</h3>
      {rows.length === 0 ? (
        <p className="text-xs text-muted-foreground">{emptyLabel}</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full border-2 border-border border-collapse text-xs">
            <thead>
              <tr className="text-muted-foreground">
                <th className="border border-border border-b-2 px-2 py-1 text-right">{labels.rank}</th>
                <th className="border border-border border-b-2 px-2 py-1 text-left">{labels.player}</th>
                <th className="border border-border border-b-2 px-2 py-1 text-right">{labels.frags}</th>
                <th className="border border-border border-b-2 px-2 py-1 text-right">{labels.deaths}</th>
                <th className="border border-border border-b-2 px-2 py-1 text-right">{labels.ratio}</th>
                <th className="border border-border border-b-2 px-2 py-1 text-right">{labels.matches}</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((rival, idx) => {
                const color = ratioColor(rival.deaths, rival.ratio)
                return (
                  <tr key={`${rival.gamertag}-${idx}`} className="hover:bg-accent/40 transition-colors">
                    <td className="border border-border px-2 py-1.5 text-right font-mono text-muted-foreground">
                      {idx + 1}
                    </td>
                    <td className="border border-border px-2 py-1.5 text-left">
                      <button
                        type="button"
                        className="font-semibold text-info hover:underline whitespace-nowrap"
                        onClick={() => onPlayerClick(rival.gamertag)}
                      >
                        {rival.gamertag}
                      </button>
                    </td>
                    <td className="border border-border px-2 py-1.5 text-right font-mono tabular-nums">{rival.frags}</td>
                    <td className="border border-border px-2 py-1.5 text-right font-mono tabular-nums">{rival.deaths}</td>
                    <td
                      className="border border-border px-2 py-1.5 text-right font-mono font-bold tabular-nums"
                      style={color ? { color } : undefined}
                    >
                      {formatRatio(rival.ratio)}
                    </td>
                    <td className="border border-border px-2 py-1.5 text-right font-mono tabular-nums">{rival.match_count}</td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

export function CareerRivalsSection() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const locale = useAppShellStore((s) => s.locale) as ManifestLocale
  const t = (key: keyof typeof careerManifest) => careerManifest[key][locale]
  const navigate = useNavigate()

  const { data, isLoading, isError } = useCareerRivals(playerSlug)

  const onPlayerClick = (gamertag: string) => {
    void navigate({
      to: '/players/$playerSlug/explorer',
      params: { playerSlug },
      search: { mode: 'player', target: gamertag },
    })
  }

  const labels = {
    rank: t('career.rivals.col_rank'),
    player: t('career.rivals.col_player'),
    frags: t('career.rivals.col_frags'),
    deaths: t('career.rivals.col_deaths'),
    ratio: t('career.rivals.col_ratio'),
    matches: t('career.rivals.col_matches'),
  }

  return (
    <section className="space-y-3">
      <h2 className="text-sm font-semibold">{t('career.rivals.section_title')}</h2>

      {isLoading && (
        <div className="flex h-24 items-center justify-center">
          <Spinner size="md" />
        </div>
      )}
      {isError && <p className="text-sm text-destructive">{t('career.errors.load_progression_failed')}</p>}
      {!isLoading && !isError && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <RivalsTable
            title={t('career.rivals.nemesis_title')}
            rows={data?.nemeses ?? []}
            emptyLabel={t('career.rivals.empty')}
            labels={labels}
            onPlayerClick={onPlayerClick}
          />
          <RivalsTable
            title={t('career.rivals.victims_title')}
            rows={data?.victims ?? []}
            emptyLabel={t('career.rivals.empty')}
            labels={labels}
            onPlayerClick={onPlayerClick}
          />
        </div>
      )}
    </section>
  )
}
