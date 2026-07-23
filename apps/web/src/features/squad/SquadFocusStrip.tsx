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
  useSquadChallenges,
  useCreateSquad,
  useRenameSquad,
  useDeleteSquad,
  useEvaluateSquadChallenge,
  useRefreshSquadPool,
  useCreateSquadChallenge,
  useAbandonSquadChallenge,
  useSquadOrientation,
} from '@/features/prestige/hooks/useSquads'
import { useJoinSquadChallenge } from '@/features/prestige/hooks'
import type {
  Squad,
  SquadWithMembers,
  SquadMember,
  SquadParticipantProgress,
} from '@/lib/prestige'
import { findSquadByRoster } from './squadRoster'

const STRINGS = {
  fr: {
    title: "Cap d'escouade",
    saved: (name: string) => `Escouade « ${name} »`,
    objectives: (n: number) => `${n} objectif${n > 1 ? 's' : ''}`,
    manage: 'Gérer',
    hide: 'Masquer',
    saveCta: 'Enregistrer cette compo comme escouade',
    saving: 'Enregistrement…',
    saveSuccess: 'Escouade enregistrée',
    saveError: "Échec de l'enregistrement",
    renameCta: 'Renommer',
    deleteCta: 'Supprimer',
    confirmDelete: 'Confirmer ?',
    ok: 'OK',
    renamed: 'Escouade renommée',
    deleted: 'Escouade supprimée',
    noObjectives: 'Aucun objectif actif pour cette escouade.',
    join: 'Rejoindre',
    joined: 'Rejoint',
    joinedToast: 'Défi rejoint',
    joinError: 'Impossible de rejoindre le défi',
    evaluate: 'Réévaluer',
    evaluated: 'Progression recalculée',
    evalError: 'Échec du recalcul',
    created: 'Défi créé',
    createError: 'Échec de la création du défi',
    challengeDeleted: 'Défi supprimé',
    challengeDeleteError: 'Échec de la suppression du défi',
    reached: 'Atteint',
    expired: 'Expiré',
    target: (n: number) => `Cible ${n} / membre`,
    matchesN: (n: number) => `${n} match${n > 1 ? 's' : ''}`,
    challenge: 'Défi',
    propose: 'Proposer des défis',
    proposing: 'Génération…',
    create: 'Créer',
    creating: 'Création…',
    poolEmpty: 'Aucun défi proposé.',
    poolCaption: 'Défi collectif : chaque membre cumule la métrique jusqu’à la cible, sur les matchs de l’escouade postérieurs à la création.',
    poolError: 'Échec de la génération des défis',
    orientation: (axis: string) => `Axe à renforcer : ${axis}`,
    axisLabels: {
      combat: 'Combat',
      survival: 'Survie',
      support: 'Support',
      score: 'Score',
      objective: 'Objectif',
      impact: 'Impact',
    },
  },
  en: {
    title: 'Squad focus',
    saved: (name: string) => `Squad “${name}”`,
    objectives: (n: number) => `${n} objective${n > 1 ? 's' : ''}`,
    manage: 'Manage',
    hide: 'Hide',
    saveCta: 'Save this lineup as a squad',
    saving: 'Saving…',
    saveSuccess: 'Squad saved',
    saveError: 'Save failed',
    renameCta: 'Rename',
    deleteCta: 'Delete',
    confirmDelete: 'Confirm?',
    ok: 'OK',
    renamed: 'Squad renamed',
    deleted: 'Squad deleted',
    noObjectives: 'No active objective for this squad.',
    join: 'Join',
    joined: 'Joined',
    joinedToast: 'Challenge joined',
    joinError: 'Could not join the challenge',
    evaluate: 'Re-evaluate',
    evaluated: 'Progress recalculated',
    evalError: 'Recalculation failed',
    created: 'Challenge created',
    createError: 'Failed to create challenge',
    challengeDeleted: 'Challenge deleted',
    challengeDeleteError: 'Failed to delete challenge',
    reached: 'Reached',
    expired: 'Expired',
    target: (n: number) => `Target ${n} / member`,
    matchesN: (n: number) => `${n} match${n > 1 ? 'es' : ''}`,
    challenge: 'Challenge',
    propose: 'Suggest challenges',
    proposing: 'Generating…',
    create: 'Create',
    creating: 'Creating…',
    poolEmpty: 'No challenge suggested.',
    poolCaption: 'Team challenge: each member accumulates the metric toward the target, over squad matches after creation.',
    poolError: 'Failed to generate challenges',
    orientation: (axis: string) => `Focus to improve: ${axis}`,
    axisLabels: {
      combat: 'Combat',
      survival: 'Survival',
      support: 'Support',
      score: 'Score',
      objective: 'Objective',
      impact: 'Impact',
    },
  },
}

export function SquadFocusStrip() {
  const { selectedRows, playerSlug } = useSquadContext()
  const locale = useAppShellStore((s) => s.locale)
  const t = STRINGS[locale === 'en' ? 'en' : 'fr']
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
  t: (typeof STRINGS)['fr']
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
  t: (typeof STRINGS)['fr']
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

function SquadObjectivesPanel({
  squadId,
  playerSlug,
  members,
  t,
}: {
  squadId: string
  playerSlug: string
  members: SquadMember[]
  t: (typeof STRINGS)['fr']
}) {
  const locale = useAppShellStore((s) => s.locale)
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const { data } = useSquadChallenges(squadId, playerSlug)
  const join = useJoinSquadChallenge()
  const evaluate = useEvaluateSquadChallenge(squadId)
  const refreshPool = useRefreshSquadPool()
  const createChallenge = useCreateSquadChallenge(squadId)
  const abandon = useAbandonSquadChallenge(squadId)
  const challenges = data?.squad_challenges ?? []
  const pool = refreshPool.data?.pool ?? []

  // Progression par défi renvoyée par « Réévaluer » (transitoire : la réponse
  // d'évaluation n'est pas dans le cache liste — on la garde le temps de l'afficher).
  const [progressByChallenge, setProgressByChallenge] = useState<
    Record<string, SquadParticipantProgress[]>
  >({})
  // Défi en attente de confirmation de suppression (2e clic), pattern du footer squad.
  const [confirmingDelete, setConfirmingDelete] = useState<string | null>(null)

  const onEvaluate = (challengeId: string) =>
    evaluate.mutate(
      { id: challengeId, requestedBy: playerSlug },
      {
        onSuccess: (res) => {
          setProgressByChallenge((prev) => ({ ...prev, [challengeId]: res.progress }))
          toast.success(t.evaluated)
        },
        onError: (err) => toast.error(apiErrorMessage(err) ?? t.evalError),
      },
    )

  const onJoin = (challengeId: string) =>
    join.mutate(
      { challengeId, userId: playerSlug, squadId },
      {
        onSuccess: () => toast.success(t.joinedToast),
        onError: (err) => toast.error(apiErrorMessage(err) ?? t.joinError),
      },
    )

  const onDelete = (challengeId: string) => {
    if (confirmingDelete !== challengeId) {
      setConfirmingDelete(challengeId)
      return
    }
    setConfirmingDelete(null)
    abandon.mutate(
      { id: challengeId, requestedBy: playerSlug },
      {
        onSuccess: () => toast.success(t.challengeDeleted),
        onError: (err) => toast.error(apiErrorMessage(err) ?? t.challengeDeleteError),
      },
    )
  }

  return (
    <div className="mt-3 space-y-3">
      {/* Défis actifs */}
      {challenges.length === 0 ? (
        <p className="text-xs text-muted-foreground">{t.noObjectives}</p>
      ) : (
        <ul className="space-y-2">
          {challenges.map((c) => {
            const label = (locale === 'en' ? c.label_en : c.label_fr) || t.challenge
            const isParticipant = c.participants.some((p) => p.user_id === playerSlug)
            return (
              <li
                key={c.id}
                className="rounded-md border border-border bg-background px-3 py-2"
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="min-w-0">
                    <span className="flex items-center gap-2">
                      <span className="truncate text-sm">{label}</span>
                      {c.expired ? (
                        <span className="shrink-0 text-2xs font-semibold text-muted-foreground">
                          {t.expired}
                        </span>
                      ) : null}
                    </span>
                    {c.target_per_member ? (
                      <span className="text-2xs text-muted-foreground">
                        {t.target(c.target_per_member)}
                      </span>
                    ) : null}
                  </span>
                  <span className="flex shrink-0 gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => onEvaluate(c.id)}
                      disabled={evaluate.isPending}
                    >
                      {t.evaluate}
                    </Button>
                    <Button
                      size="sm"
                      variant={isParticipant ? 'outline' : 'default'}
                      onClick={() => onJoin(c.id)}
                      disabled={isParticipant || c.expired || join.isPending}
                    >
                      {isParticipant ? t.joined : t.join}
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      className={confirmingDelete === c.id ? 'text-destructive' : undefined}
                      onClick={() => onDelete(c.id)}
                      disabled={abandon.isPending}
                    >
                      {confirmingDelete === c.id ? t.confirmDelete : t.deleteCta}
                    </Button>
                  </span>
                </div>
                <SquadProgressList
                  progress={progressByChallenge[c.id]}
                  targetPerMember={c.target_per_member}
                  members={members}
                  t={t}
                />
              </li>
            )
          })}
        </ul>
      )}

      {/* Proposer puis créer un défi depuis le pool (biaisé coach). */}
      <div className="border-t border-border pt-3">
        <Button
          variant="outline"
          size="sm"
          onClick={() =>
            refreshPool.mutate(
              { squadId, titleSlug, requestedBy: playerSlug },
              { onError: (err) => toast.error(apiErrorMessage(err) ?? t.poolError) },
            )
          }
          disabled={refreshPool.isPending}
        >
          {refreshPool.isPending ? t.proposing : t.propose}
        </Button>

        {refreshPool.isSuccess && pool.length === 0 ? (
          <p className="mt-2 text-xs text-muted-foreground">{t.poolEmpty}</p>
        ) : null}

        {pool.length > 0 ? (
          <p className="mt-2 text-2xs text-muted-foreground">{t.poolCaption}</p>
        ) : null}

        {pool.length > 0 ? (
          <ul className="mt-2 space-y-2">
            {pool.map((tpl) => (
              <li
                key={tpl.id}
                className="flex items-center justify-between gap-2 rounded-md border border-border bg-background px-3 py-2"
              >
                <span className="truncate text-sm">
                  {locale === 'en' ? tpl.label_en : tpl.label_fr}
                </span>
                <Button
                  size="sm"
                  onClick={() =>
                    createChallenge.mutate(
                      {
                        template_id: tpl.id,
                        title_slug: tpl.title_slug,
                        mode: 'collective',
                        eval_type: tpl.eval_type,
                        window_type: tpl.window_type,
                        window_value: tpl.window_value,
                        target_per_member: tpl.normal_target,
                        created_by: playerSlug,
                      },
                      {
                        onSuccess: () => toast.success(t.created),
                        onError: (err) => toast.error(apiErrorMessage(err) ?? t.createError),
                      },
                    )
                  }
                  disabled={createChallenge.isPending}
                >
                  {createChallenge.isPending ? t.creating : t.create}
                </Button>
              </li>
            ))}
          </ul>
        ) : null}
      </div>
    </div>
  )
}

// SquadProgressList affiche la progression par membre d'un défi, telle que
// renvoyée par « Réévaluer » (valeur cumulée, nb de matchs comptés, atteinte de
// la cible). Le gamertag est résolu depuis le roster (fallback xuid tronqué).
// Rien tant qu'aucune évaluation n'a été lancée pour ce défi.
function SquadProgressList({
  progress,
  targetPerMember,
  members,
  t,
}: {
  progress: SquadParticipantProgress[] | undefined
  targetPerMember?: number
  members: SquadMember[]
  t: (typeof STRINGS)['fr']
}) {
  if (!progress || progress.length === 0) return null
  const gamertagByXuid = new Map(members.map((m) => [m.xuid, m.gamertag]))
  return (
    <ul className="mt-2 space-y-1 border-t border-border pt-2">
      {progress.map((p) => {
        const name = gamertagByXuid.get(p.xuid) || p.xuid.slice(0, 8)
        return (
          <li key={p.xuid} className="flex items-center justify-between gap-2 text-2xs">
            <span className="truncate text-muted-foreground">{name}</span>
            <span className="flex shrink-0 items-center gap-2">
              <span className="tabular-nums">
                {Math.round(p.value)}
                {targetPerMember ? ` / ${targetPerMember}` : ''}
              </span>
              <span className="text-muted-foreground">{t.matchesN(p.matches)}</span>
              {p.completed ? (
                <span className="font-semibold text-success">{t.reached}</span>
              ) : null}
            </span>
          </li>
        )
      })}
    </ul>
  )
}
