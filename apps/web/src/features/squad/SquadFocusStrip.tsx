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
import { Button } from '@/components/ui/button'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSquadContext } from './SquadContext'
import {
  useMySquads,
  useSquadChallenges,
  useCreateSquad,
  useEvaluateSquadChallenge,
  useRefreshSquadPool,
  useCreateSquadChallenge,
  useSquadOrientation,
} from '@/features/prestige/hooks/useSquads'
import { useJoinSquadChallenge } from '@/features/prestige/hooks'
import type { SquadWithMembers } from '@/lib/prestige'

const STRINGS = {
  fr: {
    title: "Cap d'escouade",
    saved: (name: string) => `Escouade « ${name} »`,
    objectives: (n: number) => `${n} objectif${n > 1 ? 's' : ''}`,
    manage: 'Gérer les objectifs',
    hide: 'Masquer',
    saveCta: 'Enregistrer cette compo comme escouade',
    saving: 'Enregistrement…',
    noObjectives: 'Aucun objectif actif pour cette escouade.',
    join: 'Rejoindre',
    joined: 'Rejoint',
    evaluate: 'Réévaluer',
    challenge: 'Défi',
    propose: 'Proposer des défis',
    proposing: 'Génération…',
    create: 'Créer',
    creating: 'Création…',
    poolEmpty: 'Aucun défi proposé.',
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
    manage: 'Manage objectives',
    hide: 'Hide',
    saveCta: 'Save this lineup as a squad',
    saving: 'Saving…',
    noObjectives: 'No active objective for this squad.',
    join: 'Join',
    joined: 'Joined',
    evaluate: 'Re-evaluate',
    challenge: 'Challenge',
    propose: 'Suggest challenges',
    proposing: 'Generating…',
    create: 'Create',
    creating: 'Creating…',
    poolEmpty: 'No challenge suggested.',
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
  const matched = useMemo<SquadWithMembers | null>(() => {
    if (!mySquads?.squads || selectionXuids.length === 0) return null
    const sel = new Set(selectionXuids)
    return (
      mySquads.squads.find((sm) => {
        const others = sm.members.filter((m) => m.user_id !== playerSlug).map((m) => m.xuid)
        return others.length === sel.size && others.every((x) => sel.has(x))
      }) ?? null
    )
  }, [mySquads, selectionXuids, playerSlug])

  // Pas de composition exploitable → rien à afficher.
  if (selectionXuids.length === 0) return null

  const onSave = () => {
    const members = selectedRows
      .filter((r) => r.xuid)
      .map((r) => ({ xuid: r.xuid as string, gamertag: r.gamertag }))
    const name = selectedRows.map((r) => r.gamertag).join(' + ')
    createSquad.mutate({ name, created_by: playerSlug, members })
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
        <SquadObjectivesPanel squadId={matched.squad.id} playerSlug={playerSlug} t={t} />
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

// SQUAD_TITLE_SLUG : titre courant pour les défis d'escouade (V1 mono-titre).
const SQUAD_TITLE_SLUG = 'halo_infinite'

function SquadObjectivesPanel({
  squadId,
  playerSlug,
  t,
}: {
  squadId: string
  playerSlug: string
  t: (typeof STRINGS)['fr']
}) {
  const locale = useAppShellStore((s) => s.locale)
  const { data } = useSquadChallenges(squadId)
  const join = useJoinSquadChallenge()
  const evaluate = useEvaluateSquadChallenge(squadId)
  const refreshPool = useRefreshSquadPool()
  const createChallenge = useCreateSquadChallenge(squadId)
  const challenges = data?.squad_challenges ?? []
  const pool = refreshPool.data?.pool ?? []

  return (
    <div className="mt-3 space-y-3">
      {/* Défis actifs */}
      {challenges.length === 0 ? (
        <p className="text-xs text-muted-foreground">{t.noObjectives}</p>
      ) : (
        <ul className="space-y-2">
          {challenges.map((c) => (
            <li
              key={c.id}
              className="flex items-center justify-between gap-2 rounded-md border border-border bg-background px-3 py-2"
            >
              <span className="truncate text-sm">{c.template_id || t.challenge}</span>
              <span className="flex shrink-0 gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => evaluate.mutate({ id: c.id, requestedBy: playerSlug })}
                  disabled={evaluate.isPending}
                >
                  {t.evaluate}
                </Button>
                <Button
                  size="sm"
                  onClick={() => join.mutate({ challengeId: c.id, userId: playerSlug })}
                  disabled={join.isPending}
                >
                  {t.join}
                </Button>
              </span>
            </li>
          ))}
        </ul>
      )}

      {/* Proposer puis créer un défi depuis le pool (biaisé coach). */}
      <div className="border-t border-border pt-3">
        <Button
          variant="outline"
          size="sm"
          onClick={() =>
            refreshPool.mutate({ squadId, titleSlug: SQUAD_TITLE_SLUG, requestedBy: playerSlug })
          }
          disabled={refreshPool.isPending}
        >
          {refreshPool.isPending ? t.proposing : t.propose}
        </Button>

        {refreshPool.isSuccess && pool.length === 0 ? (
          <p className="mt-2 text-xs text-muted-foreground">{t.poolEmpty}</p>
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
                    createChallenge.mutate({
                      template_id: tpl.id,
                      title_slug: tpl.title_slug,
                      mode: 'collective',
                      eval_type: tpl.eval_type,
                      window_type: tpl.window_type,
                      window_value: tpl.window_value,
                      target_per_member: tpl.normal_target,
                      created_by: playerSlug,
                    })
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
