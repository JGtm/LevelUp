/**
 * useSquadPresets — alimente le GamertagCombobox de la page Escouade en presets
 * (rosters nommés chargeables d'un clic) + un footer de gestion.
 *
 * Deux sources, clairement distinctes (anti-confusion) :
 *  - « Mes escouades » : compositions sauvegardées (entité Squad). Charger =
 *    appliquer le roster ; gérer = renommer / supprimer ; enregistrer la compo
 *    courante. Sous-titre = indice dérivé des playlists/modes habituels (si le
 *    backend le fournit ; jamais stocké).
 *  - « Mes groupes » : cercles d'accès familles/amis. Charger leurs membres dans
 *    la sélection (reprend la capacité de l'ancien SquadGroupLoader).
 *
 * Toute la logique métier vit ici pour garder GamertagCombobox (components/ui)
 * générique et SquadLayout lisible.
 */
import { useState } from 'react'
import { toast } from 'sonner'
import type { ComboboxPresetGroup } from '@/components/ui/GamertagCombobox'
import { apiErrorMessage } from '@/lib/api/client'
import {
  useMySquads,
  useCreateSquad,
  useRenameSquad,
  useDeleteSquad,
} from '@/features/prestige/hooks/useSquads'
import { useMyGroups } from '@/features/groups/queries'
import type { TeammateRow } from '@/lib/api/types'
import { findSquadByRoster, squadTeammates } from './squadRoster'
import { SQUAD_PRESETS_STRINGS, type SquadPresetsStrings } from './squadPresets.i18n'

function buildUsualSubtitle(
  playlists: string[] | undefined,
  modes: string[] | undefined,
  t: SquadPresetsStrings,
): string | undefined {
  const parts = [...(playlists ?? []).slice(0, 2), ...(modes ?? []).slice(0, 1)]
  if (parts.length === 0) return undefined
  return `${t.usualPrefix} ${parts.join(' · ')}`
}

export interface UseSquadPresetsOptions {
  playerSlug: string
  /** XUID absolu du joueur courant — exclut le viewer du roster (player-agnostic).
   *  Vide tant que le header n'est pas résolu (dégradation gracieuse transitoire). */
  currentPlayerXuid: string
  hasLinkedIdentity: boolean
  locale: string
  /** Coéquipiers actuellement sélectionnés (pour « enregistrer la compo »). */
  selectedRows: TeammateRow[]
  /** Labels (playlists/modes) du filtre courant → trie en tête les escouades
   *  dont les contextes habituels matchent (indice souple, jamais un verrou). */
  activeContextLabels?: string[]
}

export interface UseSquadPresetsResult {
  presetGroups: ComboboxPresetGroup[]
  footer: React.ReactNode
  /** À brancher sur GamertagCombobox.onClose : réinitialise l'état de gestion
   *  (mode Gérer / renommage / confirmation de suppression) à la fermeture. */
  onClose: () => void
}

export function useSquadPresets({
  playerSlug,
  currentPlayerXuid,
  hasLinkedIdentity,
  locale,
  selectedRows,
  activeContextLabels,
}: UseSquadPresetsOptions): UseSquadPresetsResult {
  const t = SQUAD_PRESETS_STRINGS[locale === 'en' ? 'en' : 'fr']
  const [manageMode, setManageMode] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [draft, setDraft] = useState('')
  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null)

  const { data: mySquads } = useMySquads(playerSlug)
  const { data: myGroups } = useMyGroups(hasLinkedIdentity)
  const createSquad = useCreateSquad(playerSlug)
  const renameSquad = useRenameSquad(playerSlug)
  const deleteSquad = useDeleteSquad(playerSlug)

  const slugLower = playerSlug.toLowerCase()

  const commitRename = (id: string) => {
    const name = draft.trim()
    if (name) renameSquad.mutate({ squadId: id, name })
    setEditingId(null)
  }

  const manageControls = (id: string, name: string): React.ReactNode => {
    if (editingId === id) {
      return (
        <>
          <input
            value={draft}
            autoFocus
            onChange={(e) => setDraft(e.target.value)}
            onClick={(e) => e.stopPropagation()}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                e.preventDefault()
                commitRename(id)
              } else if (e.key === 'Escape') {
                setEditingId(null)
              }
            }}
            className="w-28 rounded border border-input bg-background px-1.5 py-0.5 text-xs"
          />
          <button
            type="button"
            onClick={() => commitRename(id)}
            className="px-1 text-xs font-medium text-primary"
          >
            {t.ok}
          </button>
        </>
      )
    }
    const confirming = confirmDeleteId === id
    return (
      <>
        <button
          type="button"
          title={t.rename}
          onClick={() => {
            setEditingId(id)
            setDraft(name)
            setConfirmDeleteId(null)
          }}
          className="px-1 text-2xs text-muted-foreground hover:text-foreground"
        >
          {t.rename}
        </button>
        <button
          type="button"
          title={confirming ? t.confirmDelete : t.del}
          onClick={() => {
            if (confirming) {
              deleteSquad.mutate({ squadId: id })
              setConfirmDeleteId(null)
            } else {
              setConfirmDeleteId(id)
            }
          }}
          className={
            confirming
              ? 'px-1 text-2xs font-semibold text-destructive'
              : 'px-1 text-2xs text-muted-foreground hover:text-destructive'
          }
        >
          {confirming ? t.confirmDelete : t.del}
        </button>
      </>
    )
  }

  // Tri souple : escouades dont les playlists/modes habituels matchent le filtre
  // courant remontent en tête. Score 0 si aucun filtre → ordre d'origine (stable).
  const activeSet = new Set((activeContextLabels ?? []).map((l) => l.toLowerCase()))
  const scoreSquad = (sm: { usual_playlists?: string[]; usual_modes?: string[] }): number => {
    if (activeSet.size === 0) return 0
    return [...(sm.usual_playlists ?? []), ...(sm.usual_modes ?? [])].reduce(
      (n, l) => (activeSet.has(l.toLowerCase()) ? n + 1 : n),
      0,
    )
  }
  const sortedSquads = [...(mySquads?.squads ?? [])].sort((a, b) => scoreSquad(b) - scoreSquad(a))

  const squadOptions = sortedSquads.map((sm) => {
    // Player-agnostic : on exclut le viewer par SON xuid (pas par le slug), puis on
    // mappe vers le gamertag. Charger le preset ne réinjecte donc jamais le viewer
    // (bug « lui-même en double, créateur absent » quand le créateur ≠ viewer).
    const gamertags = squadTeammates(sm.members, currentPlayerXuid)
      .map((m) => m.gamertag ?? '')
      .filter((g) => g)
    return {
      id: sm.squad.id,
      name: sm.squad.name,
      subtitle: buildUsualSubtitle(sm.usual_playlists, sm.usual_modes, t),
      gamertags,
      trailing: manageMode ? manageControls(sm.squad.id, sm.squad.name) : undefined,
    }
  })

  const groupOptions = (myGroups ?? []).map((g) => ({
    id: g.id,
    name: g.name,
    gamertags: g.members
      .map((m) => m.gamertag)
      .filter((gt) => gt && gt.toLowerCase() !== slugLower),
  }))

  const presetGroups: ComboboxPresetGroup[] = []
  if (squadOptions.length > 0) {
    presetGroups.push({ key: 'squads', label: t.squadsHeader, options: squadOptions })
  }
  if (hasLinkedIdentity && groupOptions.length > 0) {
    presetGroups.push({ key: 'groups', label: t.groupsHeader, options: groupOptions })
  }

  // Anti-doublon : si la sélection courante correspond déjà au roster d'une
  // escouade enregistrée, on n'autorise pas un nouvel enregistrement (les tables
  // shared_social sont append-only → un doublon serait persistant). Même garde
  // que SquadFocusStrip, via le helper partagé.
  const selectionXuids = selectedRows.map((r) => r.xuid).filter((x): x is string => !!x)
  const alreadySaved = findSquadByRoster(mySquads?.squads, selectionXuids, currentPlayerXuid) != null
  const canSave = selectionXuids.length > 0 && !alreadySaved
  const handleSave = () => {
    if (!canSave) return
    const members = selectedRows
      .filter((r) => r.xuid)
      .map((r) => ({ xuid: r.xuid as string, gamertag: r.gamertag }))
    if (members.length === 0) return
    const name = selectedRows.map((r) => r.gamertag).join(' + ')
    createSquad.mutate(
      { name, created_by: playerSlug, members },
      {
        onSuccess: () => toast.success(t.saveSuccess),
        onError: (err) => toast.error(apiErrorMessage(err) ?? t.saveError),
      },
    )
  }
  const saveLabel = createSquad.isPending ? t.saving : alreadySaved ? t.saved : t.save

  const footer = (
    <div className="flex items-center justify-between gap-2 px-3 py-2">
      <button
        type="button"
        disabled={!canSave || createSquad.isPending}
        onClick={handleSave}
        className="rounded-md px-2 py-1 text-xs font-medium text-primary hover:bg-accent disabled:opacity-40"
      >
        {saveLabel}
      </button>
      {squadOptions.length > 0 && (
        <button
          type="button"
          onClick={() => {
            setManageMode((v) => !v)
            setEditingId(null)
            setConfirmDeleteId(null)
          }}
          className="rounded-md px-2 py-1 text-xs text-muted-foreground hover:text-foreground"
        >
          {manageMode ? t.done : t.manage}
        </button>
      )}
    </div>
  )

  const onClose = () => {
    setManageMode(false)
    setEditingId(null)
    setConfirmDeleteId(null)
  }

  return { presetGroups, footer, onClose }
}
