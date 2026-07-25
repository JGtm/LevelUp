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
import { normalizeModeLabel } from '@/lib/halo/modeLabel'
import { findSquadByRoster, squadTeammates } from './squadRoster'
import { SQUAD_PRESETS_STRINGS, type SquadPresetsStrings } from './squadPresets.i18n'

/**
 * buildUsualSubtitle — indice « surtout <playlists> · <mode> » de la liste des
 * compositions enregistrées (V72-10). Exporté pour test.
 *
 * `modes` arrive du backend (`GET /squads` → `usual_modes`, cf.
 * `prestige_squad_match_provider.go SquadUsualContexts`). Depuis V72-10.1 le serveur
 * sert des modes NORMALISÉS et TRADUITS en FR (mode_name_tr : « Team Slayer » →
 * « Assassin en équipe ») — la forme `pair_name` BRUTE des origines n'est plus le cas
 * nominal. Le strip canonique reste appliqué ici pour la DÉGRADATION GRACIEUSE de
 * `SquadUsualContexts` (traducteur absent / en erreur), qui peut encore servir un
 * libellé composé « Assassin en équipe sur Bazaar » ou « Arena:Slayer on Bazaar »
 * (mode ET carte collés, cf. `reference_mode_label_cross_lang_strip`). Même strip que
 * `match-card.tsx`/`format.ts` (`normalizeModeLabel`, pas de mapLabel connu ici → seul
 * le filet générique "on/sur <reste>" s'applique, mais il est TOUJOURS déclenché
 * quelle que soit la langue) :
 *  - retire la carte collée au mode (décision utilisateur V72-10 : la carte
 *    n'apporte rien dans cet indice, playlist + mode suffisent) ;
 *  - extrait le sous-mode propre du préfixe technique ("Arena:Slayer" → "Slayer").
 * Sur un libellé déjà propre, `normalizeModeLabel` est idempotent — appliquer le strip
 * au cas nominal ne coûte rien (même choix que `contextMatchKey`, juste en dessous).
 *
 * `playlists` n'a pas ce problème (pas de suffixe carte) — laissé tel quel. Le
 * FR-d'abord est déjà appliqué côté SQL (COALESCE playlist_name_fr/pair_name_fr
 * avant repli EN) ; un résidu anglais au-delà de ce point est un trou de
 * traduction côté données (asset_translations / mode_name_tr), pas un bug front.
 */
export function buildUsualSubtitle(
  playlists: string[] | undefined,
  modes: string[] | undefined,
  t: SquadPresetsStrings,
): string | undefined {
  const cleanModes = (modes ?? [])
    .map((m) => normalizeModeLabel(m, undefined))
    .filter((m): m is string => !!m)
  const parts = [...(playlists ?? []).slice(0, 2), ...cleanModes.slice(0, 1)]
  if (parts.length === 0) return undefined
  return `${t.usualPrefix} ${parts.join(' · ')}`
}

/**
 * contextMatchKey — forme COMMUNE de comparaison d'un libellé de contexte
 * (playlist ou mode), pour le tri souple des compositions.
 *
 * Pourquoi (constat 2026-07-25) : les deux côtés de la comparaison ne sont pas
 * garantis dans la même forme.
 *  - Côté filtre actif (`activeContextLabels`, cf. SquadLayout) : libellés
 *    d'affichage des options de cascade — pour les modes, `service.modeUI` =
 *    `analysis.ResolveModeUIWithVariant` → déjà normalisé et FR-d'abord.
 *  - Côté composition (`usual_modes` de `GET /squads`) : depuis V72-10.1 le
 *    backend normalise ET traduit en FR (mode_name_tr, « Team Slayer » →
 *    « Assassin en équipe »), mais la dégradation gracieuse de
 *    `SquadUsualContexts` (traducteur absent / en erreur) peut encore servir la
 *    forme brute composée « Assassin en équipe sur Bazaar ».
 *
 * Une comparaison sur les chaînes brutes échouait donc silencieusement dès qu'un
 * suffixe de carte ou un préfixe technique subsistait d'UN SEUL côté — alors que
 * `buildUsualSubtitle`, juste au-dessus, normalise déjà. On applique le MÊME
 * strip canonique (`normalizeModeLabel`, idempotent sur un libellé propre) aux
 * DEUX côtés, puis on compare sans tenir compte de la casse.
 *
 * Appliqué aussi aux playlists : la transformation est symétrique (même fonction
 * des deux côtés), donc elle ne peut pas créer de faux négatif ; au pire deux
 * libellés distincts se replient sur la même clé, ce qui reste acceptable pour un
 * indice de tri souple (jamais un verrou).
 *
 * Limite connue, NON traitée ici : si la traduction FR n'existe que d'un côté
 * (mode_name_tr couvre le mode mais pas l'asset du filtre, ou l'inverse), les
 * clés restent FR vs EN et ne matchent pas. C'est un trou de données
 * (asset_translations / mode_name_tr), pas un défaut de ce comparateur.
 */
function contextMatchKey(label: string): string {
  return (normalizeModeLabel(label, undefined) ?? label).trim().toLowerCase()
}

/** Clés de comparaison des libellés du filtre courant. Exporté pour test. */
export function buildActiveContextKeys(labels: string[] | undefined): Set<string> {
  const out = new Set<string>()
  for (const l of labels ?? []) {
    const key = contextMatchKey(l)
    if (key) out.add(key)
  }
  return out
}

/**
 * scoreSquadContext — nombre de contextes habituels de la composition qui
 * matchent le filtre courant. 0 si aucun filtre actif → ordre d'origine (stable).
 * Exporté pour test.
 */
export function scoreSquadContext(
  sm: { usual_playlists?: string[]; usual_modes?: string[] },
  activeKeys: Set<string>,
): number {
  if (activeKeys.size === 0) return 0
  return [...(sm.usual_playlists ?? []), ...(sm.usual_modes ?? [])].reduce(
    (n, l) => (activeKeys.has(contextMatchKey(l)) ? n + 1 : n),
    0,
  )
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
  // courant remontent en tête. Comparaison sur la forme commune normalisée +
  // insensible à la casse (cf. contextMatchKey).
  const activeKeys = buildActiveContextKeys(activeContextLabels)
  const sortedSquads = [...(mySquads?.squads ?? [])].sort(
    (a, b) => scoreSquadContext(b, activeKeys) - scoreSquadContext(a, activeKeys),
  )

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
