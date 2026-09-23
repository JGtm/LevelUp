/**
 * queries — les lectures TanStack Query de l'onglet Tactique.
 *
 * TROIS LECTURES, TROIS CLÉS, et elles ne s'invalident pas ensemble :
 *   - le PÉRIMÈTRE (les `match_id` de la sélection) dépend de la barre L2 ;
 *   - la GRILLE dépend du périmètre et de la composition ;
 *   - le FOND d'une carte ne dépend de rien : il est figé entre deux cuissons.
 *
 * ─── LE PÉRIMÈTRE EST RÉSOLU CÔTÉ CLIENT, COMME L'EXPLORATEUR ────────────────
 *
 * La barre produit un `FilterContextInput` ; `/filters/match-ids` le résout sur la
 * base JOUEUR (période OU sessions épinglées, contexte solo/escouade, cascade) et
 * rend les `match_id` ; l'onglet les poste en LISTE BLANCHE. Une seule définition du
 * périmètre dans l'app — et c'est la seule qui sache lire les sessions, que les
 * requêtes shared du lecteur tactique ne joignent pas.
 *
 * ─── UNE RELECTURE GARDE LA RÉPONSE PRÉCÉDENTE (retours rejeu L2, 2026-09-23) ──────
 *
 * Les trois lectures dont la clé porte l'empreinte d'un filtre (`hashFiltre(`) — périmètre,
 * grille, raster — déclarent `placeholderData: precedenteDuMemeJoueur(…)`. Sans lui, changer
 * de question ou cocher une session créait une clé SANS DONNÉE : la vue repassait par
 * « en attente » et démontait tout, fond de carte compris, avant de le reconstruire. Avec
 * lui, la réponse précédente reste affichée (`isPlaceholderData`) pendant la relecture, et
 * la vue dit qu'elle se met à jour. Garde-rail : `queriesPlaceholder.guard.test.ts`.
 * SEULE EXCEPTION : `useTacticalCellule` — les contributions d'une AUTRE cellule mentiraient.
 *
 * AU MÊME JOUEUR SEULEMENT (revue L2-R2) : la page reste montée quand le joueur change, et un
 * `keepPreviousData` nu gardait le périmètre du joueur A pendant la résolution de celui de B
 * — la liste de `match_id` de A partait alors sur les lectures de B, et la réponse
 * s'affichait comme celle de B.
 */
import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import type {
  CareerEncountersResponse,
  FilterContextInput,
  FilterMatchIdsResponse,
  TacticalCelluleReponse,
  TacticalMapsBody,
  TacticalMapsPage,
  TacticalRaster,
  TeammateOption,
  ReplayMapBackground,
} from '@/lib/api/types'
import { queryKeys } from '@/lib/query/keys'
import { mapFrame, type MapFrame } from '@/lib/replay/heatPaint'
import { useAppShellStore } from '@/stores/appShellStore'

/** FNV-1a 32 bits — même algorithme que `useFiltersPreview` et `computeHash`. */
export function hashFiltre(valeur: unknown): string {
  const s = JSON.stringify(valeur) ?? ''
  let h = 0x811c9dc5
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i)
    h = Math.imul(h, 0x01000193) >>> 0
  }
  return h.toString(16).padStart(8, '0')
}

/**
 * precedenteDuMemeJoueur — `keepPreviousData`, BORNÉ à la même portée : la réponse
 * précédente n'est gardée que si la requête précédente visait le même joueur, le même titre
 * (et, pour le raster, la même carte). Sinon, aucune donnée : la vue repasse par le premier
 * chargement, ce qui est vrai.
 *
 * `portee` = les éléments de la clé qui SUIVENT son nom, dans l'ordre des fabriques
 * `queryKeys.tactical*` : `[playerSlug, titleSlug]`, puis `mapId` pour le raster. Seule
 * l'empreinte du filtre (dernier élément) peut différer.
 */
function precedenteDuMemeJoueur(...portee: string[]) {
  return <T>(
    precedente: T | undefined,
    requete: { queryKey: readonly unknown[] } | undefined,
  ): T | undefined =>
    requete && portee.every((v, i) => requete.queryKey[i + 1] === v) ? precedente : undefined
}

/**
 * useTacticalMatchIDs — le PÉRIMÈTRE : les `match_id` de la sélection courante.
 *
 * Même endpoint et même pipeline que le bouton « Voir les matchs » de l'omnibar :
 * `match_context`, sessions, période et cascade y sont tous honorés. Une liste VIDE
 * est une réponse légitime (le filtre ne retient rien) et NON une absence de
 * réponse : les lectures qui la consomment servent alors une grille vide.
 *
 * UN ÉCHEC N'EST PAS AVALÉ. Il est journalisé ici puis propagé : l'appelant doit
 * pouvoir DIRE que la lecture a échoué, là où une résolution muette laissait la page
 * afficher « aucune carte » pour toujours.
 */
export function useTacticalMatchIDs(playerSlug: string, contexte: FilterContextInput) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  return useQuery({
    queryKey: queryKeys.tacticalMatchIDs(playerSlug, titleSlug, hashFiltre(contexte)),
    queryFn: async () => {
      try {
        return await api.post<FilterMatchIdsResponse>(
          `/players/${playerSlug}/filters/match-ids`,
          contexte,
        )
      } catch (err) {
        console.error('[tactique] résolution du périmètre en échec', err)
        throw err
      }
    },
    enabled: !!playerSlug,
    staleTime: 2 * 60 * 1000,
    // Un nouveau filtre garde l'ancien périmètre jusqu'à la réponse : sans lui, la liste
    // retombait à `null`, et la grille comme le raster repassaient par « en attente ».
    // Jamais celui d'un AUTRE joueur (cf. `precedenteDuMemeJoueur`).
    placeholderData: precedenteDuMemeJoueur(playerSlug, titleSlug),
  })
}

/**
 * useTacticalMaps — la grille des cartes jouées DANS LE PÉRIMÈTRE.
 *
 * POST : la liste de `match_id` ne tient pas dans une query string. La clé de cache
 * porte l'empreinte du périmètre ET de la composition — deux périmètres différents
 * ne doivent jamais se resservir l'un l'autre.
 *
 * `matchIDs` à `null` = le périmètre n'est pas encore résolu (ou une composition
 * n'est pas traduisible) : la requête N'EST PAS lancée, et l'appelant rend un état
 * d'ATTENTE — jamais un état vide, qui se lirait comme un résultat.
 *
 * Le corps est typé par le CONTRAT GÉNÉRÉ (`TacticalMapsBody`) : renommer un champ
 * côté Go doit casser `tsc` ici, pas se découvrir à l'exécution.
 */
export function useTacticalMaps(
  playerSlug: string,
  matchIDs: string[] | null,
  coequipiers: string[],
) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const corps: TacticalMapsBody = { match_ids: matchIDs ?? [], coequipiers }
  return useQuery({
    queryKey: queryKeys.tacticalMaps(playerSlug, titleSlug, hashFiltre(corps)),
    queryFn: () => api.post<TacticalMapsPage>(`/players/${playerSlug}/tactical/maps`, corps),
    enabled: !!playerSlug && matchIDs !== null,
    staleTime: 2 * 60 * 1000,
    // La grille précédente reste affichée pendant la relecture : les vignettes (fond et
    // mini-plan) ne sont plus démontées, et le titre de l'écran d'analyse, qui lit le NOM
    // de la carte dans cette grille, ne retombe plus sur l'identifiant brut.
    placeholderData: precedenteDuMemeJoueur(playerSlug, titleSlug),
  })
}

/**
 * useCoequipierOptions — les coéquipiers proposés au sélecteur de composition,
 * AVEC LEUR XUID (le serveur ne connaît que celui-là).
 *
 * Source : `/pages/career/encounters`, la liste des joueurs croisés le plus souvent
 * COMME COÉQUIPIERS — amis compris, contrairement à `top-encounters` qui les exclut
 * par construction et qui serait donc la mauvaise liste pour choisir une escouade.
 *
 * MÊME CLÉ DE CACHE QUE LA PAGE CARRIÈRE (`queryKeys.careerEncounters`) : c'est le
 * même endpoint et la même réponse, donc UNE seule entrée de cache et jamais deux
 * requêtes pour une seule liste. Le hook de la carrière n'est pas importé — ce
 * serait une dépendance croisée entre features pour trois lignes de projection.
 */
export function useCoequipierOptions(playerSlug: string): {
  options: TeammateOption[]
  chargees: boolean
} {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const { data, isSuccess } = useQuery({
    queryKey: queryKeys.careerEncounters(playerSlug, titleSlug),
    queryFn: () =>
      api.get<CareerEncountersResponse>(`/players/${playerSlug}/pages/career/encounters`),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })
  // MÉMOÏSÉ sur la réponse : un tableau neuf à chaque rendu recalculait la composition, puis
  // les paramètres du raster et leur empreinte (`hashFiltre` sur toute la liste de
  // `match_id`), à chaque rendu de la page.
  const options = useMemo(
    () =>
      (data?.teammates ?? []).map((t) => ({
        gamertag: t.gamertag,
        xuid: t.xuid,
        encounter_count: t.match_count,
      })),
    [data],
  )
  return { options, chargees: isSuccess }
}

/**
 * useTacticalMapBackgroundUrl — l'URL affichable du fond d'une carte, ou `null`.
 *
 * L'URL D'OBJET **EST** L'ENTRÉE DE CACHE, et c'est un choix, pas une facilité. Elle est
 * créée une fois dans la requête et gardée pour la session (`staleTime` et `gcTime`
 * infinis) : une image de carte ne change qu'à une re-cuisson, et revenir sur l'onglet doit
 * réafficher la grille sans re-télécharger. La contrepartie assumée est qu'aucune URL n'est
 * révoquée avant la fermeture de l'onglet ; créer l'URL au montage de chaque vignette
 * aurait imposé un `setState` dans un effet — cascade de rendus que le lint du dépôt
 * signale, pour un gain nul à cette échelle.
 *
 * LA BORNE EST LE NOMBRE DE CARTES DU TITRE, quel que soit le nombre de joueurs consultés
 * (revue R1, W4) : la clé de cache ne porte PAS le joueur. Une image de carte est une donnée
 * de référence du titre, identique pour tout le monde ; la mettre en cache par joueur
 * retenait N fois le même contenu. Le joueur reste dans l'URL de fetch, parce que la route
 * est derrière l'ownership.
 *
 * 404 = la carte n'a pas de fond figé : cas NOMINAL (toutes les cartes n'en ont pas). Pas
 * de nouvelle tentative, rien à dire à l'utilisateur — la vignette s'affiche sans image.
 */
export function useTacticalMapBackgroundUrl(playerSlug: string, mapId: string): string | null {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const { data } = useQuery({
    queryKey: queryKeys.tacticalMapBackground(titleSlug, mapId),
    queryFn: async () => {
      const blob = await api.getBlob(
        `/players/${playerSlug}/tactical/${encodeURIComponent(mapId)}/background.png`,
      )
      // `createObjectURL` manque dans certains environnements de test : l'absence de fond
      // n'est pas une panne de grille, la vignette s'affiche sans image.
      if (typeof URL.createObjectURL !== 'function') return null
      return URL.createObjectURL(blob)
    },
    staleTime: Infinity,
    gcTime: Infinity,
    enabled: !!playerSlug && !!mapId,
    retry: false,
  })
  return data ?? null
}

/**
 * useTacticalMapBackgroundFrame — LE CALAGE du fond d'une carte : le rectangle MONDE que
 * l'image couvre (`/tactical/{map_id}/background`, déjà servi par l'API depuis la phase 5).
 *
 * C'EST LE SEUL REPÈRE DANS LEQUEL LE CALQUE ET SON FOND COÏNCIDENT, et il n'était pas lu :
 * le plan projetait sa chaleur sur la boîte englobante de ses propres cellules, c'est-à-dire
 * dans un repère sans rapport avec l'image posée dessous (sur Illusion, 30 x 36 m contre
 * 53 x 69 m). Les zones chaudes tombaient donc à côté du bâtiment.
 *
 * MÊME RÉGIME DE CACHE QUE L'IMAGE : le calage ne change qu'à une re-cuisson, et il ne dépend
 * pas du joueur — la clé ne le porte donc pas. 404 = carte sans fond figé : cas NOMINAL
 * (toutes les cartes n'en ont pas), aucune nouvelle tentative, le plan retombe alors sur ses
 * propres bornes.
 */
export function useTacticalMapBackgroundFrame(playerSlug: string, mapId: string): MapFrame | null {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const { data } = useQuery({
    queryKey: queryKeys.tacticalMapBackgroundFrame(titleSlug, mapId),
    queryFn: () =>
      api.get<ReplayMapBackground>(
        `/players/${playerSlug}/tactical/${encodeURIComponent(mapId)}/background`,
      ),
    staleTime: Infinity,
    gcTime: Infinity,
    enabled: !!playerSlug && !!mapId,
    retry: false,
  })
  return data?.calibration ? mapFrame(data.calibration) : null
}

/**
 * useTacticalRaster — la grille de placement pour UNE carte et une question.
 *
 * POST : les paramètres de la requête sont envoyés dans le corps (liste de match_id,
 * composition, question, qui, spawn). La clé de cache porte l'empreinte de tous les
 * paramètres — changer de question donne une nouvelle requête, jamais un cache croisé.
 *
 * `matchIds` à `null` = le périmètre n'est pas encore résolu (même contrat que
 * `useTacticalMaps`) : la requête N'EST PAS lancée. Une liste VIDE une fois le périmètre
 * résolu est une réponse légitime (aucun match ne correspond) et part normalement.
 */
export function useTacticalRaster(
  playerSlug: string,
  mapId: string,
  params: {
    match_ids: string[] | null
    coequipiers?: string[]
    question?: string
    qui?: string
    spawn?: string
  },
) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const corps = { ...params, match_ids: params.match_ids ?? [] }
  return useQuery({
    queryKey: queryKeys.tacticalRaster(playerSlug, titleSlug, mapId, hashFiltre(corps)),
    queryFn: () =>
      api.post<TacticalRaster>(
        `/players/${playerSlug}/tactical/${encodeURIComponent(mapId)}/raster`,
        corps,
      ),
    enabled: !!playerSlug && !!mapId && params.match_ids !== null,
    staleTime: 2 * 60 * 1000,
    // La réponse précédente reste servie pendant la relecture (`isPlaceholderData`) : la
    // vue la montre ESTOMPÉE sous « Mise à jour… » (décision Q26 du 2026-09-23) et ne
    // démonte jamais le fond. Elle ne passe jamais d'une carte à l'autre (la vue est
    // remontée par carte, `key={scope.carte}` dans `TacticalPage`, et la portée compare la
    // carte), ni d'un joueur à l'autre.
    placeholderData: precedenteDuMemeJoueur(playerSlug, titleSlug, mapId),
  })
}

/**
 * useTacticalCellule — le DÉTAIL d'UNE cellule cliquée : ses contributions ouvrables
 * (ADR 0029) et le compte de celles écartées (lot M1, Tactique S.1).
 *
 * MÊME PÉRIMÈTRE QUE `useTacticalRaster`, PLUS L'ADRESSE DE LA CELLULE. La clé de cache en
 * porte l'empreinte : deux cellules différentes ne doivent jamais se resservir l'une
 * l'autre, et changer de question/qui/spawn doit redemander le détail.
 *
 * `cellule` à `null` = AUCUNE CELLULE SÉLECTIONNÉE : la requête N'EST PAS lancée. Ce n'est
 * pas un cas d'attente comme `matchIds === null` (le périmètre non résolu) — c'est l'état
 * NORMAL avant tout clic, et il ne doit déclencher aucun appel réseau.
 *
 * `pas_m` VOYAGE AVEC L'ADRESSE : une adresse de cellule n'a de sens qu'à un pas donné, et
 * depuis le pas adaptatif (lot 3.2) la lecture peut avoir retenu 0,5, 1 ou 2 m. On renvoie
 * donc le `pas_m` que la lecture agrégée a publié — jamais une constante.
 */
export function useTacticalCellule(
  playerSlug: string,
  mapId: string,
  cellule: { col: number; lig: number; pas_m: number } | null,
  params: {
    match_ids: string[] | null
    coequipiers?: string[]
    question?: string
    qui?: string
    spawn?: string
  },
) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const corps = cellule ? { ...params, match_ids: params.match_ids ?? [], cellule } : null
  return useQuery({
    queryKey: queryKeys.tacticalCellule(playerSlug, titleSlug, mapId, hashFiltre(corps)),
    queryFn: () =>
      api.post<TacticalCelluleReponse>(
        `/players/${playerSlug}/tactical/${encodeURIComponent(mapId)}/cellule`,
        corps,
      ),
    enabled: !!playerSlug && !!mapId && !!corps && params.match_ids !== null,
    staleTime: 2 * 60 * 1000,
  })
}
