/**
 * IdentitiesSection — l'annuaire des joueurs (ADR 0035 D7).
 *
 * Une ligne par identité : ce que chacun des quatre registres (compte, profils
 * de suivi, identifiants, suivi live) sait d'un xuid, et les anomalies quand ils
 * se contredisent. C'est la vue qui aurait montré, le soir même, le compte Xbox
 * inconnu du 2026-07-23 — jusqu'ici la page Gestion listait les comptes et les
 * pages de monitoring les profils, donc un compte sans profil n'était nulle part.
 *
 * Table TanStack, tri par défaut « anomalies à regarder d'abord ». Couleurs :
 * uniquement des tokens sémantiques. Aucune logique ici — elle vit dans
 * identitiesDisplay.ts et useInstanceLock.ts.
 */
import { useMemo, useState } from 'react'
import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  getSortedRowModel,
  useReactTable,
  type SortingState,
} from '@tanstack/react-table'

import { Button } from '@/components/ui/button'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import { useCopyToClipboard } from '@/lib/clipboard/useCopyToClipboard'
import type {
  IdentityAnomaly,
  IdentityProfileRef,
  IdentityRecord,
  IdentityTokenRef,
} from '@/lib/api/types'
import { useAdminIdentities } from '../management/identitiesQueries'
import {
  anomalyLabelKey,
  anomalySeverityToken,
  identityRowKey,
  profileStateKeys,
  tokenState,
  tokenStateToken,
  TOKEN_STATE_KEY,
  warningCount,
} from '../management/identitiesDisplay'
import { LOCK_FORCED_CODE, useInstanceLock } from '../management/useInstanceLock'
import { useAdminT, type TAdmin } from '../useAdminText'

const EMPTY = '—'

const columnHelper = createColumnHelper<IdentityRecord>()

/** Le xuid, en chiffres fixes et copiable d'un clic (il se recopie souvent). */
function XuidCell({ xuid, tA }: { xuid?: string; tA: TAdmin }) {
  // Mécanique « copier + coche transitoire » partagée (lib/clipboard). Presse-papier
  // refusé (contexte non sécurisé, permission) : le xuid reste affiché et
  // sélectionnable à la main — pas d'état d'erreur à porter ici.
  const { copy: copyText, copied } = useCopyToClipboard()
  if (!xuid) return <span className="text-muted-foreground">{EMPTY}</span>

  const copy = () => copyText(xuid)

  return (
    <button
      type="button"
      onClick={() => void copy()}
      title={tA('admin.identities.copy_xuid')}
      className="font-mono text-xs tabular-nums text-muted-foreground hover:text-foreground"
    >
      {xuid}
      {copied && <span className="ml-1 not-italic">{tA('admin.identities.copy_done')}</span>}
    </button>
  )
}

function ProfileBadges({ profiles, tA }: { profiles?: IdentityProfileRef[] | null; tA: TAdmin }) {
  if (!profiles?.length) return <span className="text-muted-foreground">{EMPTY}</span>
  return (
    <div className="flex flex-wrap gap-1">
      {profiles.map((p) => (
        <span
          key={`${p.title_slug}/${p.key}`}
          className="rounded bg-muted px-2 py-0.5 text-xs text-muted-foreground"
          title={p.key}
        >
          <span className="font-medium text-foreground">{p.title_slug}</span>
          {profileStateKeys(p).map((key) => (
            <span key={key} className="ml-1 italic">
              {tA(key)}
            </span>
          ))}
        </span>
      ))}
    </div>
  )
}

function TokenCell({ token, tA }: { token?: IdentityTokenRef; tA: TAdmin }) {
  const state = tokenState(token)
  const color = tokenStateToken(state)
  return (
    <span
      className="rounded bg-muted px-2 py-0.5 text-xs text-muted-foreground"
      title={token?.last_auth_error || undefined}
    >
      <span
        className={color ? 'font-semibold' : undefined}
        style={color ? { color: tokenCssVar(color) } : undefined}
      >
        {tA(TOKEN_STATE_KEY[state])}
      </span>
    </span>
  )
}

function AnomalyBadges({ anomalies, tA }: { anomalies?: IdentityAnomaly[] | null; tA: TAdmin }) {
  if (!anomalies?.length) return <span className="text-muted-foreground">{EMPTY}</span>
  return (
    <div className="flex flex-wrap gap-1">
      {anomalies.map((a, i) => {
        const key = anomalyLabelKey(a.code)
        return (
          <span
            key={`${a.code}-${i}`}
            className="rounded bg-muted px-2 py-0.5 text-xs font-medium"
            style={{ color: tokenCssVar(anomalySeverityToken(a.severity)) }}
            title={a.detail || undefined}
          >
            {/* Code inconnu du web (serveur plus récent) : le montrer brut plutôt
                que de rendre une ligne muette. */}
            {key ? tA(key) : a.code}
          </span>
        )
      })}
    </div>
  )
}

/** Interrupteur « Instance fermée » — le verrou n'avait aucune interface. */
function InstanceLockToggle({ tA }: { tA: TAdmin }) {
  const { locked, isPending, isError, errorCode, setLocked } = useInstanceLock()
  // 409 instance_lock_forced : le verrou vient de l'environnement, pas du fichier —
  // le dire, plutôt qu'un échec générique et une case qui se recoche seule.
  const errorKey =
    errorCode === LOCK_FORCED_CODE
      ? 'admin.identities.lock_forced'
      : 'admin.identities.lock_failed'
  return (
    <div className="space-y-1">
      <label className="flex items-center gap-2 text-sm font-medium text-foreground">
        <input
          type="checkbox"
          checked={locked}
          disabled={isPending}
          onChange={(e) => setLocked(e.target.checked)}
        />
        {tA('admin.identities.lock_label')}
        {isPending && (
          <span className="text-xs font-normal text-muted-foreground">
            {tA('admin.identities.lock_saving')}
          </span>
        )}
      </label>
      <p className="max-w-xl text-xs text-muted-foreground">{tA('admin.identities.lock_help')}</p>
      {isError && <p className="text-xs text-destructive">{tA(errorKey)}</p>}
    </div>
  )
}

function CountsLine({ counts, tA }: { counts?: Record<string, number>; tA: TAdmin }) {
  const total = counts?.identities ?? 0
  const warnings = counts?.warnings ?? 0
  const infos = counts?.infos ?? 0
  return (
    <div className="flex flex-wrap items-center gap-3 text-xs text-muted-foreground">
      <span>
        <strong className="text-foreground">{total}</strong> {tA('admin.identities.count_identities')}
      </span>
      <span style={warnings > 0 ? { color: tokenCssVar('warning') } : undefined}>
        <strong>{warnings}</strong> {tA('admin.identities.count_warnings')}
      </span>
      <span>
        <strong>{infos}</strong> {tA('admin.identities.count_infos')}
      </span>
    </div>
  )
}

export function IdentitiesSection() {
  const { data, isLoading, isError, refetch, isFetching } = useAdminIdentities()
  const tA = useAdminT()
  // Ordre par défaut : les identités à regarder en premier. Le serveur rend déjà
  // cet ordre ; le rejouer ici garde le tri cohérent après un clic d'en-tête.
  const [sorting, setSorting] = useState<SortingState>([{ id: 'anomalies', desc: true }])

  const identities = useMemo(() => data?.identities ?? [], [data?.identities])

  const columns = useMemo(
    () => [
      columnHelper.accessor((rec) => rec.gamertag ?? '', {
        id: 'gamertag',
        header: () => tA('admin.identities.col_gamertag'),
        cell: (info) => (
          <span className="font-medium text-foreground">{info.getValue() || EMPTY}</span>
        ),
      }),
      columnHelper.accessor((rec) => rec.xuid ?? '', {
        id: 'xuid',
        header: () => tA('admin.identities.col_xuid'),
        cell: (info) => <XuidCell xuid={info.row.original.xuid} tA={tA} />,
      }),
      columnHelper.accessor((rec) => rec.account?.username ?? '', {
        id: 'account',
        header: () => tA('admin.identities.col_account'),
        cell: (info) => {
          const account = info.row.original.account
          if (!account) return <span className="text-muted-foreground">{EMPTY}</span>
          return (
            <span className="text-foreground">
              {account.username}
              <span className="ml-1 text-xs text-muted-foreground">{account.role}</span>
            </span>
          )
        },
      }),
      columnHelper.display({
        id: 'profiles',
        header: () => tA('admin.identities.col_profiles'),
        cell: (info) => <ProfileBadges profiles={info.row.original.profiles} tA={tA} />,
      }),
      columnHelper.display({
        id: 'token',
        header: () => tA('admin.identities.col_token'),
        cell: (info) => <TokenCell token={info.row.original.token} tA={tA} />,
      }),
      columnHelper.display({
        id: 'watched',
        header: () => tA('admin.identities.col_watched'),
        cell: (info) => {
          const watched = info.row.original.watched ?? []
          if (watched.length === 0) return <span className="text-muted-foreground">{EMPTY}</span>
          return <span className="text-xs text-muted-foreground">{watched.join(', ')}</span>
        },
      }),
      columnHelper.accessor((rec) => warningCount(rec), {
        id: 'anomalies',
        header: () => tA('admin.identities.col_anomalies'),
        sortDescFirst: true,
        cell: (info) => <AnomalyBadges anomalies={info.row.original.anomalies} tA={tA} />,
      }),
    ],
    [tA],
  )

  const table = useReactTable({
    data: identities,
    columns,
    state: { sorting },
    onSortingChange: setSorting,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
  })

  return (
    <section className="space-y-3">
      {/* Pas de titre ici : la page Gestion porte déjà le sien au-dessus de
          chaque section (un h2 par section), on ne le répète pas. */}
      <div className="flex flex-wrap items-start justify-between gap-2">
        <p className="max-w-2xl text-xs text-muted-foreground">{tA('admin.identities.subtitle')}</p>
        <Button size="sm" variant="outline" onClick={() => void refetch()} disabled={isFetching}>
          {isFetching ? tA('admin.identities.loading') : tA('admin.identities.refresh')}
        </Button>
      </div>

      <InstanceLockToggle tA={tA} />

      {isLoading ? (
        <p className="text-sm text-muted-foreground">{tA('admin.identities.loading')}</p>
      ) : isError ? (
        <p className="text-sm text-destructive">{tA('admin.identities.unavailable')}</p>
      ) : identities.length === 0 ? (
        <EmptyStateNotice title={tA('admin.identities.empty')} description="" />
      ) : (
        <>
          <CountsLine counts={data?.counts} tA={tA} />
          <div className="overflow-x-auto rounded-md border">
            <table className="w-full text-sm">
              <thead>
                {table.getHeaderGroups().map((hg) => (
                  <tr
                    key={hg.id}
                    className="border-b bg-muted/40 text-left text-xs uppercase tracking-wide text-muted-foreground"
                  >
                    {hg.headers.map((header) => (
                      <th
                        key={header.id}
                        className={`px-3 py-2 font-medium ${header.column.getCanSort() ? 'cursor-pointer select-none' : ''}`}
                        onClick={header.column.getToggleSortingHandler()}
                      >
                        <span className="inline-flex items-center gap-1">
                          {flexRender(header.column.columnDef.header, header.getContext())}
                          {{ asc: ' ↑', desc: ' ↓' }[header.column.getIsSorted() as string] ?? ''}
                        </span>
                      </th>
                    ))}
                  </tr>
                ))}
              </thead>
              <tbody>
                {table.getRowModel().rows.map((row, index) => (
                  <tr
                    key={identityRowKey(row.original, index)}
                    className="border-b last:border-0 align-top"
                  >
                    {row.getVisibleCells().map((cell) => (
                      <td key={cell.id} className="px-3 py-2">
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
    </section>
  )
}
