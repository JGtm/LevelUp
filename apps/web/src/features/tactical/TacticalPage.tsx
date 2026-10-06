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
 * ─── LES ÉTATS, DANS L'ORDRE ────────────────────────────────────────────────────────────────
 *
 * Composition impossible, échec, attente (aucune donnée encore), aucune carte : chacun est dit UNE
 * fois, dans la colonne des cartes, et aucune lecture n'est montée à côté — un plan sur un
 * périmètre qu'on ne sait pas appliquer serait faux, et une « Mise à jour… » sans fin aussi.
 * « Aucune carte jouée » est une RÉPONSE : elle exige une liste de matchs résolue et une liste de
 * cartes servie. En RELECTURE (nouveau filtre en cours, réponse précédente gardée par
 * `placeholderData`), rien ne se démonte : les vignettes restent, estompées, et la lecture dit
 * « Mise à jour… ».
 */
import { useMemo, type ReactNode } from 'react'
import { useParams } from '@tanstack/react-router'

import { EmptyStateNotice } from '@/components/ui/empty-state'
import { SectionCard } from '@/components/ui/section-card'
import type { TacticalMapCard } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import { usePageScope } from '@/lib/page-scope/usePageScope'
import { useAppShellStore } from '@/stores/appShellStore'

import { carteEffective, carteLue, VARIABLES_COCKPIT, type CarteEffective } from './cockpit.logic'
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

  const {
    coequipierOptions,
    composition,
    compositionImpossible,
    matchIDs,
    cartes,
    plancher,
    enEchec,
    enChargement,
    enRelecture,
    perimetreEnRelecture,
  } = useCartesDuPerimetre(playerSlug, scope)

  const etat = etatDeLaPage(t, {
    inconnus: compositionImpossible ? composition.inconnus : null,
    enEchec,
    enChargement,
    aucuneCarte: cartes.length === 0,
  })
  const effective = useMemo(() => carteEffective(scope.carte, cartes), [scope.carte, cartes])

  return (
    <>
      <TacticalFilterBar
        playerSlug={playerSlug}
        locale={locale}
        t={t}
        scope={scope}
        setScope={setScope}
        coequipierOptions={coequipierOptions}
      />
      <div
        className="grid grid-cols-1 items-start gap-3 min-[1400px]:grid-cols-[var(--tac-cartes-l)_minmax(0,1fr)]"
        style={VARIABLES_COCKPIT}
        data-testid="tactical-cockpit"
      >
        <TacticalMapsColumn
          cartes={cartes}
          plancher={plancher}
          carteActive={etat ? '' : effective.mapId}
          playerSlug={playerSlug}
          locale={locale}
          t={t}
          onSelect={(mapId) => setScope({ carte: mapId })}
          enRelecture={enRelecture}
          etat={etat}
        />
        {!etat && (
          <LectureDeLaCarte
            effective={effective}
            cartes={cartes}
            playerSlug={playerSlug}
            locale={locale}
            t={t}
            matchIds={matchIDs}
            coequipiers={composition.xuids}
            perimetreEnRelecture={perimetreEnRelecture}
          />
        )}
      </div>
    </>
  )
}

/** useScopeTactique — l'état de la barre et la carte choisie, persistés dans l'URL. */
function useScopeTactique(playerSlug: string, titleSlug: string, lang: string | undefined) {
  // Params de navigation : `lang` est OPTIONNEL dans la route (`{-$lang}`), donc on ne le pose que
  // s'il est présent — le poser à vide fabriquerait une URL `//`.
  const routeParams = useMemo(() => {
    const p: Record<string, string> = { playerSlug, titleSlug }
    if (lang) p.lang = lang
    return p
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
 * xuids, et les cartes jouées que ce périmètre contient, avec les états de la page.
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

  // « EN CHARGEMENT » VEUT DIRE « AUCUNE DONNÉE ENCORE » : la liste des cartes est SUSPENDUE tant
  // que le périmètre n'est pas résolu, et une requête suspendue n'est pas « chargée » (en TanStack
  // v5, `isLoading` vaut `isPending && isFetching`, donc FAUX sur une requête désactivée). On lit
  // donc la PRÉSENCE des données.
  const enEchec = perimetreEnEchec || grille.isError
  return {
    coequipierOptions,
    composition,
    compositionImpossible,
    matchIDs,
    cartes,
    plancher: data?.plancher_matchs ?? 0,
    enEchec,
    enChargement: !compositionImpossible && !enEchec && (perimetre === undefined || data === undefined),
    enRelecture: perimetreEnRelecture || grille.isPlaceholderData,
    perimetreEnRelecture,
  }
}

/**
 * etatDeLaPage — le corps de la colonne quand la page n'a pas de cartes à montrer, dans l'ORDRE :
 * composition impossible, échec, attente (premier chargement), aucune carte. `undefined` : la
 * colonne montre ses cartes et la lecture est montée.
 */
function etatDeLaPage(
  t: TacticalText,
  e: { inconnus: string[] | null; enEchec: boolean; enChargement: boolean; aucuneCarte: boolean },
): ReactNode | undefined {
  if (e.inconnus) {
    return (
      <div className="p-3">
        <EmptyStateNotice
          title={t.unknownTeammateTitle}
          description={t.unknownTeammateDescription(e.inconnus.join(', '))}
        />
      </div>
    )
  }
  if (e.enEchec) {
    return (
      <p className="p-3 text-sm text-muted-foreground" data-testid="tactical-erreur">
        {t.error}
      </p>
    )
  }
  if (e.enChargement) return <p className="p-3 text-sm text-muted-foreground">{t.loading}</p>
  if (e.aucuneCarte) {
    return (
      <div className="p-3">
        <EmptyStateNotice title={t.emptyTitle} description={t.emptyDescription} />
      </div>
    )
  }
  return undefined
}

/**
 * LectureDeLaCarte — le centre et la droite du cockpit pour la carte affichée : la vue d'analyse
 * quand la carte est LUE ; pour une carte d'URL hors du filtre ou sous le plancher, son nom et
 * « Aucun match sur cette carte dans ce filtre », sans requête ; rien quand aucune carte n'est
 * ouvrable (la colonne le dit).
 */
function LectureDeLaCarte({
  effective,
  cartes,
  playerSlug,
  locale,
  t,
  matchIds,
  coequipiers,
  perimetreEnRelecture,
}: {
  effective: CarteEffective
  cartes: readonly TacticalMapCard[]
  playerSlug: string
  locale: Locale
  t: TacticalText
  matchIds: string[] | null
  coequipiers: string[]
  perimetreEnRelecture: boolean
}) {
  if (effective.origine === 'aucune') return null
  const carte = cartes.find((c) => c.map_id === effective.mapId)
  const nom = carte ? nomCarte(carte, locale) : effective.mapId
  if (!carteLue(effective)) {
    return (
      <SectionCard title={nom} label={nom}>
        <p className="p-6 text-center text-sm font-medium" data-testid="tactical-carte-hors-filtre">
          {t.planEmptyNoMatchTitle}
        </p>
      </SectionCard>
    )
  }
  // `key` = LA CARTE : changer de carte remet à zéro la vue (question, qui, spawn, cellule) ET
  // l'observateur du raster, donc aucune réponse d'une carte ne sert de placeholder à une autre.
  return (
    <div className="min-w-0">
      <TacticalAnalysisView
        key={effective.mapId}
        playerSlug={playerSlug}
        mapId={effective.mapId}
        mapName={nom}
        locale={locale}
        t={t}
        matchIds={matchIds}
        coequipiers={coequipiers}
        perimetreEnRelecture={perimetreEnRelecture}
      />
    </div>
  )
}
