/**
 * IssueSections — les quatre sections de listes d'inconnus de la page
 * Qualité données : modes sans traduction FR (form mode_name_tr), assets UUID
 * bruts + playlists hors catalogue (form asset_translations), xuids orphelins
 * (lecture seule). Une seule ligne de formulaire ouverte à la fois par section.
 */
import { useState } from 'react'
import { toast } from 'sonner'

import { EmptyStateNotice } from '@/components/ui/empty-state'
import { apiErrorMessage } from '@/lib/api/client'
import type { AdminDataQualityIssue, DataQualityIssueKind } from '@/lib/api/types'
import { DATA_QUALITY_LOCALE, useDataQualityIssues } from './queries'
import { useResolveAssetTranslation, useResolveModeTranslation } from './mutations'
import { InlineResolveForm } from './InlineResolveForm'
import { IssueTable, type IssueColumn } from './IssueTable'
import { useAdminT, type TAdmin } from '../useAdminText'
import { SectionHeader } from '../components/SectionHeader'

function rowKeyOf(issue: AdminDataQualityIssue): string {
  return `${issue.kind}:${issue.asset_kind ?? ''}:${issue.id}`
}

function toastResolve(action: string | undefined, tA: TAdmin) {
  toast.success(action === 'updated' ? tA('admin.dq.saved_updated') : tA('admin.dq.saved_created'))
}

/** Section générique : header + états vides + table. */
function SectionShell({
  title,
  hint,
  kind,
  emptyTitle,
  emptyDesc,
  children,
  count,
}: {
  title: string
  hint?: string
  kind: DataQualityIssueKind
  emptyTitle: string
  emptyDesc: string
  count: number | undefined
  children: React.ReactNode
}) {
  return (
    <section className="space-y-3" data-section={kind}>
      <div>
        <SectionHeader title={title} />
        {hint && <p className="mt-0.5 max-w-2xl text-xs text-muted-foreground">{hint}</p>}
      </div>
      {/* count === 0 = plus aucune inconnue dans cette catégorie : c'est un
          succès (catalogue complet / tout traduit / zéro orphelin), rendu en
          vert perceptible plutôt qu'en placeholder gris. */}
      {count === 0 ? (
        <EmptyStateNotice title={emptyTitle} description={emptyDesc} tone="success" />
      ) : (
        children
      )}
    </section>
  )
}

// ─── Modes sans traduction FR ─────────────────────────────────────────────────

export function UntranslatedModesSection() {
  const tA = useAdminT()
  const { data } = useDataQualityIssues('untranslated_modes')
  const resolve = useResolveModeTranslation()
  const [openID, setOpenID] = useState<string | null>(null)

  const columns: IssueColumn[] = [
    {
      header: tA('admin.dq.col_mode'),
      cell: (i) => <span className="font-medium text-foreground">{i.id}</span>,
      sortValue: (i) => i.id,
    },
    {
      header: tA('admin.dq.col_sample'),
      cell: (i) => (
        <span className="font-mono text-xs text-muted-foreground" title={i.label}>
          {i.label}
        </span>
      ),
      sortValue: (i) => i.label ?? null,
    },
  ]

  // Libellé honnête : la locale visée (data.locale, échotée par le serveur ;
  // défaut fr) apparaît dans le titre — « Modes sans traduction (fr) ».
  return (
    <SectionShell
      title={`${tA('admin.dq.modes_section')} (${data?.locale ?? DATA_QUALITY_LOCALE})`}
      kind="untranslated_modes"
      emptyTitle={tA('admin.dq.modes_empty_title')}
      emptyDesc={tA('admin.dq.modes_empty_desc')}
      count={data?.items?.length}
    >
      <IssueTable
        issues={data?.items ?? []}
        matchLinkTitleSlug={data?.title_slug}
        columns={columns}
        sortable
        actionLabel={tA('admin.dq.translate_btn')}
        openID={openID}
        onAction={(i) => setOpenID(openID === rowKeyOf(i) ? null : rowKeyOf(i))}
        renderForm={(i) => (
          <InlineResolveForm
            subject={i.id}
            fields={[{ key: 'name_fr', label: tA('admin.dq.form_name_fr'), placeholder: i.id }]}
            busy={resolve.isPending}
            onCancel={() => setOpenID(null)}
            onSubmit={(values) => {
              const nameFR = values.name_fr?.trim()
              if (!nameFR) return
              resolve.mutate(
                { mode_en: i.id, name_fr: nameFR },
                {
                  onSuccess: (res) => {
                    toastResolve(res.action, tA)
                    setOpenID(null)
                  },
                  onError: (err) => toast.error(apiErrorMessage(err) ?? tA('admin.dq.save_failed')),
                },
              )
            }}
          />
        )}
      />
    </SectionShell>
  )
}

// ─── Assets UUID bruts ────────────────────────────────────────────────────────

export function RawAssetsSection() {
  const tA = useAdminT()
  const { data } = useDataQualityIssues('raw_uuids')
  const [openID, setOpenID] = useState<string | null>(null)

  const columns: IssueColumn[] = [
    {
      header: tA('admin.dq.col_asset_kind'),
      cell: (i) => <span className="text-xs uppercase tracking-wide text-muted-foreground">{i.asset_kind}</span>,
      sortValue: (i) => i.asset_kind ?? null,
    },
    {
      header: tA('admin.dq.col_id'),
      cell: (i) => (
        <span className="font-mono text-xs text-foreground" title={i.id}>
          {i.id}
        </span>
      ),
      sortValue: (i) => i.id,
    },
  ]

  return (
    <SectionShell
      title={tA('admin.dq.raw_uuids_section')}
      hint={tA('admin.dq.raw_uuids_hint')}
      kind="raw_uuids"
      emptyTitle={tA('admin.dq.raw_uuids_empty_title')}
      emptyDesc={tA('admin.dq.raw_uuids_empty_desc')}
      count={data?.items?.length}
    >
      <IssueTable
        issues={data?.items ?? []}
        matchLinkTitleSlug={data?.title_slug}
        columns={columns}
        sortable
        actionLabel={tA('admin.dq.resolve_btn')}
        openID={openID}
        onAction={(i) => setOpenID(openID === rowKeyOf(i) ? null : rowKeyOf(i))}
        renderForm={(i) => <AssetResolveForm issue={i} onDone={() => setOpenID(null)} />}
      />
    </SectionShell>
  )
}

/** Formulaire asset (EN + FR) partagé entre raw_uuids et orphan_playlists. */
function AssetResolveForm({
  issue,
  assetKind,
  onDone,
}: {
  issue: AdminDataQualityIssue
  assetKind?: string
  onDone: () => void
}) {
  const tA = useAdminT()
  const resolve = useResolveAssetTranslation()
  const kind = assetKind ?? issue.asset_kind ?? ''
  return (
    <InlineResolveForm
      subject={`${kind} · ${issue.id}`}
      fields={[
        { key: 'name_en', label: tA('admin.dq.form_name_en'), initial: '' },
        { key: 'name_fr', label: tA('admin.dq.form_name_fr'), initial: '' },
      ]}
      busy={resolve.isPending}
      onCancel={onDone}
      onSubmit={(values) => {
        const nameEN = values.name_en?.trim() ?? ''
        const nameFR = values.name_fr?.trim() ?? ''
        if (!nameEN && !nameFR) return
        resolve.mutate(
          { asset_kind: kind, asset_id: issue.id, name_en: nameEN || undefined, name_fr: nameFR || undefined },
          {
            onSuccess: (res) => {
              toastResolve(res.action, tA)
              onDone()
            },
            onError: (err) => toast.error(apiErrorMessage(err) ?? tA('admin.dq.save_failed')),
          },
        )
      }}
    />
  )
}

// ─── Playlists hors catalogue ─────────────────────────────────────────────────

export function OrphanPlaylistsSection() {
  const tA = useAdminT()
  const { data } = useDataQualityIssues('orphan_playlists')
  const [openID, setOpenID] = useState<string | null>(null)

  const columns: IssueColumn[] = [
    {
      header: tA('admin.dq.col_id'),
      cell: (i) => (
        <span className="font-mono text-xs text-foreground" title={i.id}>
          {i.id}
        </span>
      ),
      sortValue: (i) => i.id,
    },
    {
      header: tA('admin.dq.col_label'),
      cell: (i) => <span className="text-foreground">{i.label || '—'}</span>,
      sortValue: (i) => i.label ?? null,
    },
  ]

  return (
    <SectionShell
      title={tA('admin.dq.orphan_playlists_section')}
      kind="orphan_playlists"
      emptyTitle={tA('admin.dq.orphan_playlists_empty_title')}
      emptyDesc={tA('admin.dq.orphan_playlists_empty_desc')}
      count={data?.items?.length}
    >
      <IssueTable
        issues={data?.items ?? []}
        matchLinkTitleSlug={data?.title_slug}
        columns={columns}
        sortable
        actionLabel={tA('admin.dq.resolve_btn')}
        openID={openID}
        onAction={(i) => setOpenID(openID === rowKeyOf(i) ? null : rowKeyOf(i))}
        renderForm={(i) => (
          <AssetResolveForm issue={i} assetKind="playlist" onDone={() => setOpenID(null)} />
        )}
      />
    </SectionShell>
  )
}

// ─── XUIDs orphelins (lecture seule, liste longue → pagination serveur) ────────

const ORPHAN_XUIDS_PAGE_SIZE = 25

export function OrphanXuidsSection() {
  const tA = useAdminT()
  const [pageIndex, setPageIndex] = useState(0)
  // Pagination serveur : la fenêtre [offset, offset+limit) est calculée côté Go
  // (limit/offset) et le total pilote le nombre de pages TanStack (règle 13).
  const { data } = useDataQualityIssues(
    'orphan_xuids',
    ORPHAN_XUIDS_PAGE_SIZE,
    pageIndex * ORPHAN_XUIDS_PAGE_SIZE,
  )

  const columns: IssueColumn[] = [
    {
      header: tA('admin.dq.col_id'),
      cell: (i) => <span className="font-mono text-xs text-foreground">{i.id}</span>,
    },
  ]

  return (
    <SectionShell
      title={tA('admin.dq.orphan_xuids_section')}
      hint={tA('admin.dq.orphan_xuids_hint')}
      kind="orphan_xuids"
      emptyTitle={tA('admin.dq.orphan_xuids_empty_title')}
      emptyDesc={tA('admin.dq.orphan_xuids_empty_desc')}
      count={data?.total}
    >
      {/* I16 : PAS de `sortable` ici — pagination SERVEUR (limit/offset), un tri
          client ne porterait que sur la page visible (25 lignes), pas le total
          (potentiellement des milliers) → ordre FAUX. IssueTable l'ignorerait de
          toute façon (garde défensive `pagination` prime), mais on ne le passe
          pas pour rester honnête sur l'intention. */}
      <IssueTable
        issues={data?.items ?? []}
        matchLinkTitleSlug={data?.title_slug}
        columns={columns}
        pagination={{
          pageIndex,
          pageSize: ORPHAN_XUIDS_PAGE_SIZE,
          total: data?.total ?? 0,
          onPageChange: setPageIndex,
        }}
      />
    </SectionShell>
  )
}
