/**
 * SquadFocusStrip — « Cap d'escouade » (Phase C).
 *
 * Bandeau prospectif en tête de l'onglet Synergies : si la composition courante
 * correspond à une escouade enregistrée, affiche ses objectifs (défis) et permet
 * de les gérer (rejoindre / réévaluer). Sinon, propose d'enregistrer la compo
 * comme escouade. Élément forward-looking, isolé du contenu rétrospectif.
 *
 * Identité d'escouade : roster clé xuid (cf. backend Phase C). La sélection de
 * coéquipiers (hors joueur courant) est matchée au roster exact d'une escouade.
 */
import { useMemo, useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { apiErrorMessage } from '@/lib/api/client'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSquadContext } from './SquadContext'
import {
  useMySquads,
  useCreateSquad,
  useRenameSquad,
  useDeleteSquad,
  useSquadOrientation,
} from '@/features/prestige/hooks/useSquads'
import type { Squad, SquadWithMembers } from '@/lib/prestige'
import { findSquadByRoster } from './squadRoster'
import { SQUAD_FOCUS_STRINGS, type SquadFocusText } from './squadFocusStrings'
import { SquadObjectivesPanel } from './SquadObjectivesPanel'

export function SquadFocusStrip() {
  const { selectedRows, playerSlug } = useSquadContext()
  const locale = useAppShellStore((s) => s.locale)
  const t = SQUAD_FOCUS_STRINGS[locale === 'en' ? 'en' : 'fr']
  const [open, setOpen] = useState(false)

  const selectionXuids = useMemo(
    () => selectedRows.map((r) => r.xuid).filter((x): x is string => !!x),
    [selectedRows],
  )

  const { data: mySquads } = useMySquads(playerSlug)
  const createSquad = useCreateSquad(playerSlug)

  // Match : escouade dont le roster hors joueur courant == sélection (exact).
  // Helper partagé avec useSquadPresets (source unique du matching de roster).
  const matched = useMemo<SquadWithMembers | null>(
    () => findSquadByRoster(mySquads?.squads, selectionXuids, playerSlug),
    [mySquads, selectionXuids, playerSlug],
  )

  // Pas de composition exploitable → rien à afficher.
  if (selectionXuids.length === 0) return null

  const onSave = () => {
    if (matched) return // garde anti-doublon (tables shared_social append-only)
    const members = selectedRows
      .filter((r) => r.xuid)
      .map((r) => ({ xuid: r.xuid as string, gamertag: r.gamertag }))
    const name = selectedRows.map((r) => r.gamertag).join(' + ')
    createSquad.mutate(
      { name, created_by: playerSlug, members },
      {
        onSuccess: () => toast.success(t.saveSuccess),
        onError: (err) => toast.error(apiErrorMessage(err) ?? t.saveError),
      },
    )
  }

  return (
    <section className="rounded-lg border border-border bg-card p-3">
      <div className="flex items-center justify-between gap-3">
        <div className="min-w-0">
          <p className="text-2xs uppercase tracking-wide text-muted-foreground">{t.title}</p>
          {matched ? (
            <>
              <p className="truncate text-sm font-semibold text-foreground">
                {t.saved(matched.squad.name)}
              </p>
              <SquadOrientationLine squadId={matched.squad.id} playerSlug={playerSlug} t={t} />
            </>
          ) : (
            <p className="text-sm text-muted-foreground">{t.saveCta}</p>
          )}
        </div>

        {matched ? (
          <Button variant="outline" size="sm" onClick={() => setOpen((v) => !v)}>
            {open ? t.hide : t.manage}
          </Button>
        ) : (
          <Button size="sm" onClick={onSave} disabled={createSquad.isPending}>
            {createSquad.isPending ? t.saving : t.saveCta}
          </Button>
        )}
      </div>

      {matched && open ? (
        <>
          <SquadObjectivesPanel
            squadId={matched.squad.id}
            playerSlug={playerSlug}
            members={matched.members}
            t={t}
          />
          <SquadManageActions
            squad={matched.squad}
            playerSlug={playerSlug}
            t={t}
            onDeleted={() => setOpen(false)}
          />
        </>
      ) : null}
    </section>
  )
}

// SquadOrientationLine affiche l'axe focal (à renforcer) de l'escouade — la
// lecture du coach, dérivée du profil de perf agrégé. Rien si indisponible.
function SquadOrientationLine({
  squadId,
  playerSlug,
  t,
}: {
  squadId: string
  playerSlug: string
  t: SquadFocusText
}) {
  const { data } = useSquadOrientation(squadId, playerSlug)
  const axis = data?.axis
  if (!axis) return null
  const label = (t.axisLabels as Record<string, string>)[axis] ?? axis
  return <p className="truncate text-2xs text-muted-foreground">{t.orientation(label)}</p>
}

// SquadManageActions — renommer / supprimer l'escouade courante depuis le
// panneau « Gérer » de l'encart (guichet unique du cycle de vie). Réutilise les
// mutations useRenameSquad / useDeleteSquad ; même pattern confirm-au-2e-clic que
// le footer du sélecteur (useSquadPresets). Suppression → l'encart repasse à
// « Enregistrer » (mySquads invalidé → matched redevient null).
function SquadManageActions({
  squad,
  playerSlug,
  t,
  onDeleted,
}: {
  squad: Squad
  playerSlug: string
  t: SquadFocusText
  onDeleted: () => void
}) {
  const rename = useRenameSquad(playerSlug)
  const del = useDeleteSquad(playerSlug)
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState(squad.name)
  const [confirming, setConfirming] = useState(false)

  const commitRename = () => {
    const name = draft.trim()
    if (!name || name === squad.name) {
      setEditing(false)
      return
    }
    rename.mutate(
      { squadId: squad.id, name },
      {
        onSuccess: () => {
          toast.success(t.renamed)
          setEditing(false)
        },
        onError: (err) => toast.error(apiErrorMessage(err) ?? t.saveError),
      },
    )
  }

  const onDelete = () => {
    if (!confirming) {
      setConfirming(true)
      return
    }
    del.mutate(
      { squadId: squad.id },
      {
        onSuccess: () => {
          toast.success(t.deleted)
          onDeleted()
        },
        onError: (err) => toast.error(apiErrorMessage(err) ?? t.saveError),
      },
    )
    setConfirming(false)
  }

  return (
    <div className="mt-3 flex items-center gap-2 border-t border-border pt-3">
      {editing ? (
        <>
          <input
            value={draft}
            autoFocus
            maxLength={60}
            onChange={(e) => setDraft(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                e.preventDefault()
                commitRename()
              } else if (e.key === 'Escape') {
                setEditing(false)
              }
            }}
            className="w-40 rounded border border-input bg-background px-2 py-1 text-xs"
          />
          <Button size="sm" variant="outline" onClick={commitRename} disabled={rename.isPending}>
            {t.ok}
          </Button>
        </>
      ) : (
        <Button
          size="sm"
          variant="outline"
          onClick={() => {
            setDraft(squad.name)
            setEditing(true)
            setConfirming(false)
          }}
        >
          {t.renameCta}
        </Button>
      )}
      <Button
        size="sm"
        variant="outline"
        className={confirming ? 'text-destructive' : undefined}
        onClick={onDelete}
        disabled={del.isPending}
      >
        {confirming ? t.confirmDelete : t.deleteCta}
      </Button>
    </div>
  )
}
