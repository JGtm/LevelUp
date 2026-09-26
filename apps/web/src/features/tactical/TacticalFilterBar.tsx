/**
 * TacticalFilterBar — la barre L2 de l'onglet Tactique.
 *
 * ELLE N'INVENTE AUCUN CONTRÔLE : elle ASSEMBLE ce qui existe (décision produit du
 * 2026-09-06, « un mix de l'Explorateur et de l'Escouade, sans rien inventer ») —
 *   période / saison / expérience / playlists / modes  `useLocalFilterBar`
 *   segmentation solo / escouade / mixte                `viewLabels` du même hook
 *   sessions épinglées                                  `SessionMultiSelect`
 *   composition (0 à 3 coéquipiers)                     `GamertagCombobox`
 *
 * Les deux derniers sont rendus DANS la barre du hook (`extras`), pas en dessous :
 * une seconde ligne de filtres donnerait deux zones pour un seul scope.
 *
 * ─── CE QUI S'APPLIQUE QUAND ────────────────────────────────────────────────
 *
 * Sessions et composition s'appliquent EN DIRECT, période et cascade au clic sur
 * « Analyser » — exactement le partage de la barre de l'Escouade. La raison est la
 * même : une session ou un coéquipier sont des choix ponctuels et lisibles, une
 * cascade se règle en plusieurs gestes et ne doit pas relancer une requête à chacun.
 *
 * Tout l'état committed vit dans l'URL (`usePageScope`) : le retour navigateur, le
 * rechargement et le lien partagé restaurent la même lecture.
 */
import { useEffect, useMemo } from 'react'

import {
  GamertagCombobox,
  type GamertagSuggestionSource,
} from '@/components/ui/GamertagCombobox'
import { SessionMultiSelect } from '@/components/ui/SessionMultiSelect'
import { useLocalFilterBar, type LocalFilterBarState } from '@/features/_shared/useLocalFilterBar'
import type { SessionOption, TeammateOption } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import { reconcileSquadSessionLabels } from '@/lib/sessions/sessionLabels'

import type { TacticalText } from './i18n'
import { sessionsHorsListe, sessionsProposees } from './tacticalLogic'
import { MAX_COEQUIPIERS, type TacticalScope } from './tacticalScope'

/** La SEULE source de composition que cette page sait traduire en XUID. */
const SOURCES_COEQUIPIERS: readonly GamertagSuggestionSource[] = ['frequent']

export interface TacticalFilterBarProps {
  playerSlug: string
  locale: Locale
  t: TacticalText
  scope: TacticalScope
  setScope: (patch: Partial<TacticalScope>) => void
  /** Coéquipiers proposés au sélecteur de composition (avec leur xuid). */
  coequipierOptions: TeammateOption[]
}

export function TacticalFilterBar({
  playerSlug,
  locale,
  t,
  scope,
  setScope,
  coequipierOptions,
}: TacticalFilterBarProps) {
  // L'état committed du hook EST le scope d'URL : une seule vérité, donc le retour
  // navigateur remet les pills ET les requêtes dans le même mouvement.
  const committed = useMemo(
    () => ({
      value: {
        period: { start_date: scope.debut || null, end_date: scope.fin || null },
        experience: scope.experience,
        playlists: scope.playlists,
        modes: scope.modes,
        view: scope.vue,
      } satisfies LocalFilterBarState,
      onCommit: (next: LocalFilterBarState) =>
        setScope({
          debut: next.period.start_date ?? '',
          fin: next.period.end_date ?? '',
          experience: next.experience,
          playlists: next.playlists,
          modes: next.modes,
          vue: next.view,
        }),
    }),
    [scope.debut, scope.fin, scope.experience, scope.playlists, scope.modes, scope.vue, setScope],
  )

  const avecComposition = scope.coequipiers.length > 0

  const { bar, sessionOptions } = useLocalFilterBar({
    playerSlug,
    labels: t.filterLabels,
    viewLabels: t.viewLabels,
    committed,
    // Sessions et composition sont des filtres A PART ENTIERE : ils comptent dans
    // « y a-t-il un filtre actif ? » (sinon le bouton ↺ n'apparaît même pas quand
    // seule une session filtre) et ils sont vidés par le même ↺ — `carte` ne l'est
    // PAS : c'est une SÉLECTION dans la grille, pas un filtre.
    extrasActifs: scope.sessions.length > 0 || scope.coequipiers.length > 0,
    onResetExtras: () => setScope({ sessions: [], coequipiers: [] }),
    // Fonction, et pas un nœud : les sessions proposées viennent de ce que le hook
    // charge pour ses propres counts — donc elles n'existent pas encore ici.
    extras: ({ sessionOptions: dispo }) => (
      <BarreExtras
        locale={locale}
        t={t}
        scope={scope}
        setScope={setScope}
        sessions={dispo}
        avecComposition={avecComposition}
        coequipierOptions={coequipierOptions}
        playerSlug={playerSlug}
      />
    ),
  })

  // Les sessions épinglées que la liste COURANTE ne propose pas — typiquement après
  // l'ajout d'un coéquipier, qui bascule la liste de « solo » à « escouade ».
  const proposees = useMemo(
    () => sessionsProposees(sessionOptions, avecComposition),
    [sessionOptions, avecComposition],
  )
  const horsListe = useMemo(
    () => sessionsHorsListe(scope.sessions, proposees),
    [scope.sessions, proposees],
  )

  return (
    <>
      {bar}
      {/* LA SITUATION SE VOIT, ELLE NE SE DEVINE PAS. Le filtre RESTE appliqué (le
          retirer ferait passer la lecture d'une soirée à l'historique entier) ; on dit
          donc ce qui est vrai : ces sessions ne sont pas dans la liste courante.
          Gabarit de note du dépôt (`SquadLayout`, bandeau des dégradations) — `role`
          status, tint `warning` et `text-warning` (et non `text-warning-foreground`,
          illisible sur ce tint en thème sombre : piège documenté dans
          `components/ui/privacy-banner.tsx`). */}
      {horsListe.length > 0 && (
        <div
          role="status"
          data-testid="tactical-sessions-hors-liste"
          className="mx-6 mt-2 rounded-md border border-warning/40 bg-warning/10 px-3 py-2 text-xs text-warning"
        >
          {t.sessionsHorsListe(horsListe.length, horsListe.join(', '))}
        </div>
      )}
    </>
  )
}

/** Les deux contrôles que le hook ne porte pas : sessions et composition. */
function BarreExtras({
  playerSlug,
  locale,
  t,
  scope,
  setScope,
  avecComposition,
  sessions,
  coequipierOptions,
}: {
  playerSlug: string
  locale: Locale
  t: TacticalText
  scope: TacticalScope
  setScope: (patch: Partial<TacticalScope>) => void
  avecComposition: boolean
  sessions: readonly SessionOption[]
  coequipierOptions: TeammateOption[]
}) {
  // Les sessions proposées SUIVENT la composition : escouade dès qu'un coéquipier
  // est choisi, solo sinon (même mécanique que la barre de l'Escouade).
  const proposees = useMemo(
    () => sessionsProposees(sessions, avecComposition),
    [sessions, avecComposition],
  )

  // Le COMPTE de chaque session SOUS LES FILTRES COURANTS (`match_count_filtered`),
  // comme la barre de l'Escouade : sans lui, `SessionMultiSelect` affiche le compte
  // FIGÉ du label et propose des sessions qui ne portent aucun match sous la sélection.
  const comptes = useMemo(() => {
    const m = new Map<string, number>()
    for (const s of sessions) m.set(s.label, s.match_count_filtered)
    return m
  }, [sessions])

  // ─── LE LABEL ZOMBIE ────────────────────────────────────────────────────────
  //
  // Un label de session embarque son compte de matchs et le backend filtre par égalité
  // stricte : deux matchs de plus à la prochaine synchronisation, et le label épinglé
  // dans l'URL (ou dans le miroir localStorage) ne désigne plus rien — la case à cocher
  // disparaît et la grille revient vide sans rien dire. On le remappe sur sa forme
  // courante dès que la liste arrive, et on retire ce qui ne se remappe pas.
  //
  // MÊME MÉCANIQUE ET MÊME JUSTIFICATION QUE `SquadLayout` (ré-ancrage des labels à
  // l'arrivée asynchrone des sessions) : l'écriture ne peut pas se faire au rendu, elle
  // dépend d'une donnée qui arrive après.
  useEffect(() => {
    if (scope.sessions.length === 0 || proposees.length === 0) return
    const reconcilies = reconcileSquadSessionLabels(scope.sessions, proposees)
    // LA GARDE DE `SquadLayout` (:452), et la copie l'avait perdue : quand AUCUN label
    // n'est retrouvé, on ne touche à RIEN. Le cas n'est pas un zombie de
    // synchronisation, c'est un changement de contexte — ajouter un coéquipier bascule
    // la liste de « solo » à « escouade », et la session solo épinglée en disparaît sans
    // avoir rien perdu de sa validité. L'écrire à vide faisait retomber `filter_mode` en
    // `period` SANS DATES : la lecture passait d'une soirée à l'HISTORIQUE ENTIER, pour
    // seul signal un avertissement de console. La note sous la barre le dit à la place.
    if (reconcilies.length === 0) return
    const inchange =
      reconcilies.length === scope.sessions.length &&
      reconcilies.every((l, i) => l === scope.sessions[i])
    if (inchange) return
    if (reconcilies.length < scope.sessions.length) {
      console.warn(
        '[tactique] session épinglée introuvable dans les sessions courantes — retirée du filtre',
        { epinglees: scope.sessions, retenues: reconcilies },
      )
    }
    setScope({ sessions: reconcilies })
    // eslint-disable-next-line react-hooks/exhaustive-deps -- déclencheur unique : l'arrivée (ou le changement) de la liste de sessions
  }, [proposees])

  return (
    <>
      <SessionMultiSelect
        sessions={proposees}
        selected={scope.sessions}
        onChange={(labels) => setScope({ sessions: labels })}
        locale={locale}
        placeholder={t.sessions}
        getMatchCount={(label) => comptes.get(label)}
        triggerClassName="flex items-center gap-1.5 rounded-md border border-input bg-background px-2.5 py-1 text-xs font-medium hover:bg-muted whitespace-nowrap transition-colors"
      />
      {/* LE SÉLECTEUR NE PROPOSE QUE CE QUE LA PAGE SAIT RÉSOUDRE : la composition
          part au serveur en XUIDS, et le seul endroit où cette page les connaît est
          la liste des coéquipiers fréquents. Laisser le popover offrir l'annuaire, le
          repli Xbox ou la saisie libre revenait à accepter un nom pour le refuser une
          seconde plus tard, la grille bloquée sur « Coéquipier introuvable ». */}
      <GamertagCombobox
        compact
        selected={scope.coequipiers}
        onChange={(gts) => setScope({ coequipiers: gts.slice(0, MAX_COEQUIPIERS) })}
        max={MAX_COEQUIPIERS}
        frequentOptions={coequipierOptions}
        sources={SOURCES_COEQUIPIERS}
        allowFreeInput={false}
        excludeGamertag={playerSlug}
        placeholder={t.squadPlaceholder(coequipierOptions.length)}
      />
    </>
  )
}
