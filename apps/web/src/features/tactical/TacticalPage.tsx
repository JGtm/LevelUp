/**
 * TacticalPage — L'ÉCRAN UNIQUE de l'onglet Tactique : sous la barre de filtres, la grille
 * « cockpit » — à gauche les cartes jouées, au centre et à droite la lecture de la carte affichée.
 *
 * Ce que la page décide : RIEN. La carte affichée (`carteEffective`), la colonne des cartes
 * (`colonneDesCartes`) et les mesures de la grille viennent de `cockpit.logic` (pur, testé seul) ;
 * les libellés du manifeste `tactical.toml`.
 *
 * LA CARTE AFFICHÉE (D11) : celle de l'URL (`?carte=`) si elle est ouvrable dans le filtre ; sans
 * carte dans l'URL, la plus jouée des ouvrables — choisie d'office SANS réécrire l'URL (aucune
 * entrée d'historique que l'utilisateur n'a pas demandée). Une carte d'URL absente du filtre ou
 * sous le plancher reste nommée, sans requête de lecture. Le clic sur une vignette, lui, écrit la
 * carte dans l'URL : c'est un choix de l'utilisateur, partageable.
 *
 * ─── LE PÉRIMÈTRE EST RÉSOLU AVANT D'ÊTRE LU ─────────────────────────────────────────────────
 *
 * L'onglet a sa PROPRE barre L2 (`TacticalFilterBar`), dont l'état vit dans l'URL. Elle produit un
 * contexte de filtre ; `/filters/match-ids` le résout en `match_id` sur la base JOUEUR ; les
 * lectures postent cette liste blanche. C'est ce chemin-là — et lui seul — qui sait lire les
 * SESSIONS, que les requêtes shared du lecteur tactique ne joignent pas.
 *
 * ─── QUI DIT QUOI ───────────────────────────────────────────────────────────────────────────
 *
 * LA CARTE DU PLAN EST TOUJOURS MONTÉE (retours rejeu L2, 2026-09-23) : avec la carte de l'URL, ou la
 * plus jouée dès que la liste des cartes répond — et, avant, sans carte, son cadre au rapport par
 * défaut sous l'indicateur. L'échec de la lecture ou de son périmètre et la composition impossible
 * se disent SUR le fond, qui reste. La colonne des cartes dit ses propres états : liste en échec,
 * attente, aucune carte ; quand le périmètre ne peut pas être appliqué, elle se tait (le plan dit
 * pourquoi). « Aucune carte jouée » est une RÉPONSE : elle exige une liste de matchs résolue et une
 * liste de cartes servie. En RELECTURE (réponse précédente gardée par `placeholderData`), rien ne se
 * démonte.
 */
import { useMemo, type ReactNode } from 'react'
import { useParams } from '@tanstack/react-router'

import { usePageScope } from '@/lib/page-scope/usePageScope'
import { useAppShellStore } from '@/stores/appShellStore'

import { carteEffective, VARIABLES_COCKPIT } from './cockpit.logic'
import { getTacticalText, type TacticalText } from './i18n'
import { TacticalAnalysisView } from './TacticalAnalysisView'
import { TacticalFilterBar } from './TacticalFilterBar'
import { TacticalMapsColumn } from './TacticalMapsColumn'
import { useCoequipierOptions, useTacticalMaps, useTacticalMatchIDs } from './queries'
import { contexteFiltre, nomCarte, resoudreComposition } from './tacticalLogic'
import {
  decodeTacticalScope,
  encodeTacticalScope,
  TACTICAL_URL_KEYS,
  type TacticalScope,
} from './tacticalScope'

export function TacticalPage() {
  const params = useParams({ strict: false }) as {
    playerSlug?: string
    titleSlug?: string
    lang?: string
  }
  const playerSlug = params.playerSlug ?? ''
  const locale = useAppShellStore((s) => s.locale)
  const t = getTacticalText(locale)
  const { scope, setScope } = useScopeTactique(playerSlug, params.titleSlug ?? '', params.lang)
  const p = useCartesDuPerimetre(playerSlug, scope)

  const etatColonne = etatDeLaColonne(t, p)
  // Les cartes servies ; `undefined` tant qu'aucune réponse n'existe ; `null` quand la liste a échoué.
  let listeDesCartes: typeof p.cartes | null | undefined = undefined
  if (p.cartesConnues) listeDesCartes = p.cartes
  else if (p.grilleEnEchec) listeDesCartes = null
  const effective = useMemo(() => carteEffective(scope.carte, listeDesCartes), [scope.carte, listeDesCartes])
  const carte = p.cartes.find((c) => c.map_id === effective.mapId)
  const nom = carte ? nomCarte(carte, locale) : effective.mapId

  return (
    <>
      <TacticalFilterBar
        playerSlug={playerSlug}
        locale={locale}
        t={t}
        scope={scope}
        setScope={setScope}
        coequipierOptions={p.coequipierOptions}
      />
      <div
        className="grid grid-cols-1 items-start gap-3 min-[1400px]:grid-cols-[var(--tac-cartes-l)_minmax(0,1fr)]"
        style={VARIABLES_COCKPIT}
        data-testid="tactical-cockpit"
      >
        <TacticalMapsColumn
          cartes={p.cartes}
          plancher={p.plancher}
          carteActive={etatColonne ? '' : effective.mapId}
          playerSlug={playerSlug}
          locale={locale}
          t={t}
          onSelect={(mapId) => setScope({ carte: mapId })}
          enRelecture={p.enRelecture}
          etat={etatColonne}
        />
        {/* `key` = LA CARTE : changer de carte remet à zéro la vue (lecture, joueurs, réapparition,
            cellule) ET l'observateur du raster — aucune réponse d'une carte ne sert de
            placeholder à une autre. */}
        <TacticalAnalysisView
          key={effective.mapId}
          playerSlug={playerSlug}
          carte={effective}
          mapName={nom}
          locale={locale}
          t={t}
          matchIds={p.matchIDs}
          coequipiers={p.composition.xuids}
          perimetreEnRelecture={p.perimetreEnRelecture}
          perimetreEnEchec={p.perimetreEnEchec}
          coequipiersInconnus={p.compositionImpossible ? p.composition.inconnus : null}
        />
      </div>
    </>
  )
}

/** useScopeTactique — l'état de la barre et la carte choisie, persistés dans l'URL. */
function useScopeTactique(playerSlug: string, titleSlug: string, lang: string | undefined) {
  // Params de navigation : `lang` est OPTIONNEL dans la route (`{-$lang}`), donc on ne le pose que
  // s'il est présent — le poser à vide fabriquerait une URL `//`.
  const routeParams = useMemo(() => {
    const r: Record<string, string> = { playerSlug, titleSlug }
    if (lang) r.lang = lang
    return r
  }, [playerSlug, titleSlug, lang])

  return usePageScope<TacticalScope, ReturnType<typeof encodeTacticalScope>>({
    to: '/{-$lang}/t/$titleSlug/players/$playerSlug/ascension/tactique',
    params: routeParams,
    storageKey: `levelup-tactical-scope:${playerSlug}`,
    encode: encodeTacticalScope,
    decode: decodeTacticalScope,
    urlKeys: TACTICAL_URL_KEYS,
  })
}

/**
 * useCartesDuPerimetre — le périmètre de la barre résolu en `match_id`, la composition traduite en
 * xuids, et les cartes jouées que ce périmètre contient, avec leurs états.
 */
function useCartesDuPerimetre(playerSlug: string, scope: TacticalScope) {
  const contexte = useMemo(() => contexteFiltre(scope), [scope])
  const {
    data: perimetre,
    isError: perimetreEnEchec,
    isPlaceholderData: perimetreEnRelecture,
  } = useTacticalMatchIDs(playerSlug, contexte)

  const { options: coequipierOptions, chargees } = useCoequipierOptions(playerSlug)
  const composition = useMemo(
    () => resoudreComposition(scope.coequipiers, coequipierOptions),
    [scope.coequipiers, coequipierOptions],
  )
  // Un coéquipier non traduisible ARRÊTE la lecture : l'ignorer élargirait le périmètre sans le
  // dire. Tant que la liste n'est pas chargée, c'est un état d'attente ; une fois chargée, c'est
  // une composition impossible, et on le dit.
  const compositionImpossible = chargees && composition.inconnus.length > 0
  const matchIDs = perimetre && composition.inconnus.length === 0 ? perimetre.match_ids : null

  const grille = useTacticalMaps(playerSlug, matchIDs, composition.xuids)
  const data = grille.data
  const cartes = useMemo(() => data?.cartes ?? [], [data])

  return {
    coequipierOptions,
    composition,
    compositionImpossible,
    matchIDs,
    cartes,
    // « CONNUES » VEUT DIRE « UNE RÉPONSE EXISTE » : la liste des cartes est SUSPENDUE tant que le
    // périmètre n'est pas résolu, et une requête suspendue n'est pas « chargée » (en TanStack v5,
    // `isLoading` vaut `isPending && isFetching`, donc FAUX sur une requête désactivée).
    cartesConnues: data !== undefined,
    plancher: data?.plancher_matchs ?? 0,
    perimetreEnEchec,
    grilleEnEchec: grille.isError,
    enRelecture: perimetreEnRelecture || grille.isPlaceholderData,
    perimetreEnRelecture,
  }
}

/**
 * etatDeLaColonne — le corps de la colonne quand elle n'a pas de cartes à montrer : la liste en
 * échec, un périmètre qu'on ne peut pas appliquer (rien : le plan dit pourquoi), l'attente du
 * premier chargement, aucune carte. `undefined` : la colonne montre ses cartes.
 */
function etatDeLaColonne(
  t: TacticalText,
  p: {
    grilleEnEchec: boolean
    perimetreEnEchec: boolean
    compositionImpossible: boolean
    cartesConnues: boolean
    cartes: readonly unknown[]
  },
): ReactNode | undefined {
  if (p.grilleEnEchec) {
    return (
      <p className="p-3 text-sm text-muted-foreground" data-testid="tactical-erreur">
        {t.error}
      </p>
    )
  }
  if (p.perimetreEnEchec || p.compositionImpossible) return <></>
  if (!p.cartesConnues) return <p className="p-3 text-sm text-muted-foreground">{t.loading}</p>
  if (p.cartes.length === 0) {
    return (
      <p className="p-3 text-sm font-medium text-muted-foreground" data-testid="tactical-maps-vide">
        {t.emptyTitle}
      </p>
    )
  }
  return undefined
}
