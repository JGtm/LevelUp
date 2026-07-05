/**
 * InvariantsSection — Intégrité des données (invariants sync, plan
 * SYNC_INVARIANTS_GATE). Extraction 1:1 depuis l'ancienne AdminPage.
 */
import { useEffect, useRef, useState } from 'react'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useAppShellStore } from '@/stores/appShellStore'
import { intlLocale } from '@/lib/formatters'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import { useAdminInvariants } from '../queries'
import type { components } from '@/lib/api/generated'

// Le contrat (struct Go) est la source de vérité : InvariantViolation.severity
// est un string et sample est nullable. On type le sous-composant sur le
// contrat plutôt que sur le mirror hand-écrit AdminInvariantViolation (périmé).
type InvariantViolation = components['schemas']['InvariantViolation']
import {
  SHARED_SCOPE_KEY,
  buildInvariantsSnapshot,
  invariantDelta,
  readInvariantsSnapshot,
  writeInvariantsSnapshot,
  type InvariantsSnapshot,
} from '../invariantsTrend'

export function InvariantsSection() {
  const { data, isLoading, isError, refetch, isFetching } = useAdminInvariants()
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  // Baseline ROULANTE : au 1er run de la session, comparaison au snapshot
  // localStorage (inter-sessions) ; ensuite chaque nouveau run (generated_at
  // différent) compare au run PRÉCÉDENT — pas au snapshot figé au mount
  // (sinon un refetch intra-session masquerait une régression revenue au
  // niveau pré-mount).
  const [previous, setPrevious] = useState<InvariantsSnapshot>(() => readInvariantsSnapshot())
  const lastRunRef = useRef<{ generatedAt: string; snapshot: InvariantsSnapshot } | null>(null)
  useEffect(() => {
    if (!data) return
    const snap = buildInvariantsSnapshot(data)
    const last = lastRunRef.current
    if (last && last.generatedAt !== data.generated_at) {
      setPrevious(last.snapshot)
    }
    lastRunRef.current = { generatedAt: data.generated_at, snapshot: snap }
    writeInvariantsSnapshot(snap)
  }, [data])

  return (
    <Card>
      <CardContent className="pt-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h2 className="text-lg font-semibold text-foreground">
              {t('common.admin.invariants_section')}
            </h2>
            {data?.generated_at && (
              <p className="text-xs text-muted-foreground">
                {t('common.admin.invariants_generated_at')}{' '}
                {new Date(data.generated_at).toLocaleString(intlLocale(locale))}
              </p>
            )}
          </div>
          <Button size="sm" variant="outline" onClick={() => refetch()} disabled={isFetching}>
            {isFetching ? t('common.admin.invariants_loading') : t('common.admin.invariants_refresh')}
          </Button>
        </div>

        {isLoading ? (
          <p className="text-sm text-muted-foreground">{t('common.admin.invariants_loading')}</p>
        ) : isError ? (
          <p className="text-sm text-destructive">{t('common.admin.invariants_load_failed')}</p>
        ) : !data?.reports?.length ? (
          <p className="text-sm text-muted-foreground">{t('common.admin.invariants_empty')}</p>
        ) : (
          <div className="space-y-3">
            <InvariantsCard
              title={t('common.admin.invariants_shared_scope')}
              scope={SHARED_SCOPE_KEY}
              checkError={data.shared_check_error}
              failCount={data.shared_fail_count}
              warnCount={data.shared_warn_count}
              violations={data.shared_violations ?? []}
              previous={previous}
              t={t}
            />
            {data.reports.map((r) => (
              <InvariantsCard
                key={r.player_slug || r.gamertag}
                title={r.gamertag}
                scope={r.player_slug}
                checkError={r.check_error}
                failCount={r.fail_count}
                warnCount={r.warn_count}
                violations={r.violations ?? []}
                previous={previous}
                t={t}
              />
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function InvariantsCard({
  title,
  scope,
  checkError,
  failCount,
  warnCount,
  violations,
  previous,
  t,
}: {
  title: string
  scope: string
  checkError?: string
  failCount: number
  warnCount: number
  violations: InvariantViolation[]
  previous: InvariantsSnapshot
  t: (key: CommonManifestKey) => string
}) {
  const healthy = !checkError && failCount === 0 && warnCount === 0
  return (
    <div className="rounded-md border px-4 py-3">
      <div className="flex items-center justify-between">
        <span className="font-medium text-foreground">{title}</span>
        <div className="flex items-center gap-2 text-xs">
          {checkError ? (
            <span className="rounded bg-muted px-2 py-0.5 text-destructive">
              {t('common.admin.invariants_check_error')}
            </span>
          ) : (
            <>
              {failCount > 0 && (
                <span className="rounded bg-muted px-2 py-0.5 font-semibold text-destructive">
                  {failCount} FAIL
                </span>
              )}
              {warnCount > 0 && (
                <span className="rounded bg-muted px-2 py-0.5 text-muted-foreground">
                  {warnCount} WARN
                </span>
              )}
              {healthy && (
                <span className="rounded bg-muted px-2 py-0.5 text-muted-foreground">OK</span>
              )}
            </>
          )}
        </div>
      </div>

      {checkError && <p className="mt-2 text-xs text-muted-foreground">{checkError}</p>}

      {!checkError && violations.length === 0 && (
        <p className="mt-2 text-xs text-muted-foreground">{t('common.admin.invariants_all_ok')}</p>
      )}

      {violations.length > 0 && (
        <ul className="mt-2 space-y-1.5">
          {violations.map((v) => {
            const delta = invariantDelta(previous, scope, v.key, v.count)
            return (
              <li key={v.key} className="text-xs">
                <span
                  className={
                    v.severity === 'fail'
                      ? 'font-mono font-semibold text-destructive'
                      : 'font-mono text-muted-foreground'
                  }
                >
                  [{v.severity}] {v.key}
                </span>{' '}
                <span className="text-foreground">×{v.count}</span>
                {delta !== undefined && (
                  <span
                    className={
                      delta > 0
                        ? 'ml-1 font-semibold text-destructive'
                        : 'ml-1 text-muted-foreground'
                    }
                  >
                    ({delta > 0 ? '+' : ''}
                    {delta})
                  </span>
                )}
                <span className="ml-1 text-muted-foreground">— {v.description}</span>
                {(v.sample ?? []).length > 0 && (
                  <div className="mt-0.5 truncate font-mono text-muted-foreground/70">
                    {(v.sample ?? []).join(', ')}
                  </div>
                )}
              </li>
            )
          })}
        </ul>
      )}
    </div>
  )
}
