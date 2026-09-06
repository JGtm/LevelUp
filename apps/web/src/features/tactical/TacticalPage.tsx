/**
 * TacticalPage — L'ÉCRAN D'ENTRÉE de l'onglet Tactique : la grille des cartes JOUÉES.
 *
 * Ce que la page décide : RIEN. Le tri, le verdict de plancher, la barre et la couverture
 * viennent de `tacticalLogic` (pur, testé seul) ; les libellés du manifeste `tactical.toml`.
 *
 * LE CLIC SÉLECTIONNE, IL N'OUVRE PAS ENCORE (décision de la phase 4, consignée au plan) :
 * la vue d'analyse d'une carte est la phase 5, GELÉE jusqu'au lot D de l'audit du rejeu, et
 * sa route n'existe pas. Un `Link` vers une route inexistante ne compilerait pas ; un
 * `navigate` vers une chaîne construite mènerait à un 404 — un lien MORT. La carte choisie
 * est donc écrite dans l'URL (`?carte=<map_id>`), ce qui est à la fois honnête (le bouton
 * SÉLECTIONNE, son nom accessible le dit, et la sélection se voit), durable (elle survit au
 * retour navigateur et se partage) et exactement l'état que la phase 5 consommera.
 *
 * ─── LE PÉRIMÈTRE EST RÉSOLU AVANT D'ÊTRE LU (phase 4 bis) ──────────────────
 *
 * L'onglet a sa PROPRE barre L2 (`TacticalFilterBar`), dont l'état vit dans l'URL. Elle
 * produit un contexte de filtre ; `/filters/match-ids` le résout en `match_id` sur la base
 * JOUEUR ; la grille poste cette liste blanche. C'est ce chemin-là — et lui seul — qui sait
 * lire les SESSIONS, que les requêtes shared du lecteur tactique ne joignent pas.
 */
import { useMemo } from 'react'
import { useParams } from '@tanstack/react-router'

import { EmptyStateNotice } from '@/components/ui/empty-state'
import { SectionCard } from '@/components/ui/section-card'
import { usePageScope } from '@/lib/page-scope/usePageScope'
import { useAppShellStore } from '@/stores/appShellStore'

import { getTacticalText } from './i18n'
import { TacticalFilterBar } from './TacticalFilterBar'
import { TacticalMapTile } from './TacticalMapTile'
import { useCoequipierOptions, useTacticalMaps, useTacticalMatchIDs } from './queries'
import { contexteFiltre, couvertureGrille, resoudreComposition, trierCartes } from './tacticalLogic'
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

  // Params de navigation : `lang` est OPTIONNEL dans la route (`{-$lang}`), donc on
  // ne le pose que s'il est présent — le poser à vide fabriquerait une URL `//`.
  const routeParams = useMemo(() => {
    const p: Record<string, string> = { playerSlug, titleSlug: params.titleSlug ?? '' }
    if (params.lang) p.lang = params.lang
    return p
  }, [playerSlug, params.titleSlug, params.lang])

  const { scope, setScope } = usePageScope<TacticalScope, ReturnType<typeof encodeTacticalScope>>({
    to: '/{-$lang}/t/$titleSlug/players/$playerSlug/ascension/tactique',
    params: routeParams,
    storageKey: `levelup-tactical-scope:${playerSlug}`,
    encode: encodeTacticalScope,
    decode: decodeTacticalScope,
    urlKeys: TACTICAL_URL_KEYS,
  })

  // ── Périmètre ──────────────────────────────────────────────────────────────
  const contexte = useMemo(() => contexteFiltre(scope), [scope])
  const {
    data: perimetre,
    isPending: perimetreEnCours,
    isError: perimetreEnEchec,
  } = useTacticalMatchIDs(playerSlug, contexte)

  const { options: coequipierOptions, chargees } = useCoequipierOptions(playerSlug)
  const composition = useMemo(
    () => resoudreComposition(scope.coequipiers, coequipierOptions),
    [scope.coequipiers, coequipierOptions],
  )
  // Un coéquipier non traduisible ARRÊTE la lecture : l'ignorer élargirait le
  // périmètre sans le dire. Tant que la liste n'est pas chargée, c'est un état
  // d'attente ; une fois chargée, c'est une composition impossible, et on le dit.
  const compositionImpossible = chargees && composition.inconnus.length > 0
  const matchIDs =
    perimetre && composition.inconnus.length === 0 ? perimetre.match_ids : null

  const grille = useTacticalMaps(playerSlug, matchIDs, composition.xuids)
  const data = grille.data

  const cartes = useMemo(() => trierCartes(data?.cartes ?? []), [data])
  const couverture = useMemo(() => couvertureGrille(cartes), [cartes])
  const plancher = data?.plancher_matchs ?? 0

  // ─── TROIS ÉTATS, ET L'ÉTAT VIDE EST LE DERNIER ───────────────────────────
  //
  // La grille est SUSPENDUE tant que le périmètre n'est pas résolu, et une requête
  // suspendue n'est pas « chargée » : en TanStack v5, `isLoading` vaut
  // `isPending && isFetching`, donc FAUX sur une requête désactivée. S'y fier faisait
  // rendre « Aucune carte jouée » au premier montage, à chaque clic sur « Analyser »
  // (nouvelle clé, aucune donnée en cache) et DÉFINITIVEMENT si la résolution
  // échouait. On lit donc `isPending`, qui reste vrai tant qu'aucune donnée n'existe.
  //
  // L'ORDRE COMPTE : composition impossible, puis échec, puis attente, puis état vide.
  // « Aucune carte jouée » est une RÉPONSE — elle exige une liste de matchs résolue et
  // une grille servie, jamais l'absence de l'une des deux.
  const enEchec = perimetreEnEchec || grille.isError
  const enChargement = !compositionImpossible && !enEchec && (perimetreEnCours || grille.isPending)

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
      <SectionCard
        title={t.mapsTitle}
        label={t.mapsLabel}
        footer={
          cartes.length > 0 ? (
            <div className="border-t border-border px-3 py-2 text-xs text-muted-foreground">
              <p data-testid="tactical-couverture">
                {t.coverage(couverture.cartes, couverture.matchs)}
              </p>
              {plancher > 0 && <p className="mt-1">{t.floorNote(plancher)}</p>}
            </div>
          ) : undefined
        }
      >
        <div className="p-3">
          {compositionImpossible && (
            <EmptyStateNotice
              title={t.unknownTeammateTitle}
              description={t.unknownTeammateDescription(composition.inconnus.join(', '))}
            />
          )}
          {!compositionImpossible && enChargement && (
            <p className="text-sm text-muted-foreground">{t.loading}</p>
          )}
          {!compositionImpossible && !enChargement && enEchec && (
            <p className="text-sm text-muted-foreground" data-testid="tactical-erreur">
              {t.error}
            </p>
          )}
          {!compositionImpossible && !enChargement && !enEchec && cartes.length === 0 && (
            <EmptyStateNotice title={t.emptyTitle} description={t.emptyDescription} />
          )}
          {!compositionImpossible && !enChargement && !enEchec && cartes.length > 0 && (
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
              {cartes.map((carte) => (
                <TacticalMapTile
                  key={carte.map_id}
                  carte={carte}
                  plancher={plancher}
                  playerSlug={playerSlug}
                  locale={locale}
                  t={t}
                  selectionnee={scope.carte === carte.map_id}
                  onSelect={(mapId) => setScope({ carte: mapId })}
                />
              ))}
            </div>
          )}
        </div>
      </SectionCard>
    </>
  )
}
