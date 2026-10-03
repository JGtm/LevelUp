/* eslint-disable max-lines -- 2026-09-23 (retours rejeu L2, controle de parc) : instrument de MESURE autonome, ignore par defaut (porte TACTIQUE_MESURE) ; un seul fichier par consigne du controle pour rester rejouable tel quel sur la base et sur le lot (le decouper imposerait de recopier ses modules annexes dans chaque arbre compare). Aucune logique de production. */
/// <reference types="node" />
/**
 * TacticalFond.mesure.test.ts — CONTRÔLE DE PARC du lot L2 des retours rejeu (2026-09-23) :
 * LE FOND DU PLAN NE SE DÉMONTE PLUS, SUR TOUTES LES CARTES ET TOUS LES FILMS.
 *
 * ## Pourquoi cette mesure existe
 *
 * Les tests du lot cadenassent le comportement sur UNE carte fictive (`streets`). L'exigence du
 * 23/09 (§3.0 du plan) est plus large : le correctif doit valoir pour les autres cartes et les
 * films futurs, sans régression. Cette mesure rejoue donc, avec les VRAIS modules de l'onglet
 * (`TacticalPage` et tout ce qu'il monte — seuls `api` et le routeur sont moqués), les
 * transitions de l'onglet sur CHAQUE carte du catalogue des fonds figés et avec la géométrie de
 * CHAQUE document de rejeu cuit, et rend un relevé par transition :
 *
 *   - PENDANT l'attente de la relecture : le `<img>` du fond est-il le MÊME nœud qu'avant ?
 *     l'indicateur de premier chargement REMPLACE-t-il le plan ? la vue dit-elle « Mise à
 *     jour… » ? le titre garde-t-il le NOM de la carte ? une donnée d'une autre carte ou d'un
 *     autre joueur est-elle affichée ?
 *   - APRÈS la réponse : le texte rendu (titre, KPI, légende, pied, cadre) — ce qui ne doit PAS
 *     bouger entre la base et le lot.
 *
 * Le fichier ne recopie AUCUNE logique de l'onglet : il ne fait que monter la page et lire le DOM.
 * Il tourne tel quel sur la base (sans le lot) et sur le lot : comparer les deux sorties JSON
 * donne l'avant/après.
 *
 * ## Données (lecture seule)
 *
 *   - catalogue des fonds : `data/titles/halo_infinite/reference/map_backgrounds/*.json` — le
 *     calage de chaque carte, servi tel quel comme réponse de `/tactical/{map}/background` ;
 *   - documents de rejeu cuits : `data/cache/replays/halo_infinite/<short8>.json` — lus UN À LA
 *     FOIS ; seule une empreinte normalisée de leurs trajectoires (≤ 3 000 points) est gardée pour
 *     fabriquer la réponse du raster (la répartition spatiale réelle d'un match, recalée dans le
 *     cadre de la carte). Le contenu du raster n'est pas l'objet de la mesure ; sa FORME l'est
 *     (bornes, pas, grappes, échelle, cellules hors cadre) et elle varie avec les films.
 *
 * ## Régime
 *
 *     TACTIQUE_MESURE=1 \
 *     TACTIQUE_MESURE_FONDS=<abs>/data/titles/halo_infinite/reference/map_backgrounds \
 *     TACTIQUE_MESURE_DOCS=<abs>/data/cache/replays/halo_infinite \
 *     TACTIQUE_MESURE_SORTIE=<abs>/sortie.json \
 *     npx vitest run src/features/tactical/TacticalFond.mesure.test.ts
 *
 * Défauts des dossiers : ceux du dépôt courant. Sans `TACTIQUE_MESURE`, la suite est ignorée
 * (même porte que `tourelleVisee.mesure.test.ts`). `TACTIQUE_MESURE_EXIGE=1` fait échouer la
 * suite si un invariant du lot est violé (à utiliser sur le lot, pas sur la base).
 */
import { existsSync, readdirSync, readFileSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { createElement, type ReactNode } from 'react'
import { afterAll, beforeAll, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

import type {
  ReplayMapBackground,
  TacticalMapCard,
  TacticalMapsPage,
  TacticalRaster,
} from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { racineDuDepot } from '../match-replay/test/featureFiles'
import { getTacticalText } from './i18n'
import { TacticalPage } from './TacticalPage'

// ─── Routeur et API moqués (les seuls) ─────────────────────────────────────────

let searchCourant: Record<string, unknown> = {}
let paramsCourants: Record<string, string> = {}
vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => () => undefined,
    useParams: () => paramsCourants,
    useSearch: () => searchCourant,
  }
})

interface Appel {
  path: string
  corps: unknown
  resoudre: () => void
  rejeter: () => void
  regle: boolean
}
const appels: Appel[] = []
let auto = true
let repondre: (path: string, corps: unknown) => unknown = () => {
  throw new Error('répondeur non posé')
}

function post(path: string, corps: unknown): Promise<unknown> {
  return new Promise((resolve, reject) => {
    const appel: Appel = {
      path,
      corps,
      regle: false,
      resoudre: () => {
        appel.regle = true
        try {
          resolve(repondre(path, corps))
        } catch (e) {
          reject(e)
        }
      },
      rejeter: () => {
        appel.regle = true
        reject(new Error(`échec injecté : ${path}`))
      },
    }
    appels.push(appel)
    if (auto) appel.resoudre()
  })
}

let fondDe: (mapId: string) => ReplayMapBackground | null = () => null
function get(path: string): Promise<unknown> {
  const m = /\/tactical\/([^/]+)\/background$/.exec(path)
  if (m) {
    const fond = fondDe(decodeURIComponent(m[1]))
    return fond ? Promise.resolve(fond) : Promise.reject(new Error('404'))
  }
  return Promise.resolve({ teammates: [], enemies: [], total: 0 })
}
const blobs = new WeakMap<Blob, string>()
function getBlob(path: string): Promise<Blob> {
  const m = /\/tactical\/([^/]+)\/background\.png$/.exec(path)
  const mapId = m ? decodeURIComponent(m[1]) : ''
  if (!fondDe(mapId)) return Promise.reject(new Error('404'))
  const b = new Blob([mapId])
  blobs.set(b, mapId)
  return Promise.resolve(b)
}

vi.mock('@/lib/api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api/client')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      get: (path: string) => get(path),
      post: (path: string, corps: unknown) => post(path, corps),
      getBlob: (path: string) => getBlob(path),
    },
  }
})

// ─── Données du parc ───────────────────────────────────────────────────────────

const PORTE = process.env.TACTIQUE_MESURE === '1'
const EXIGE = process.env.TACTIQUE_MESURE_EXIGE === '1'
/** Essai à blanc : ne rejoue que les N premiers films (et saute la couverture du catalogue). */
const LIMITE = Number(process.env.TACTIQUE_MESURE_LIMITE ?? '0')
const DOSSIER_FONDS =
  process.env.TACTIQUE_MESURE_FONDS ??
  join(racineDuDepot(), 'data', 'titles', 'halo_infinite', 'reference', 'map_backgrounds')
const DOSSIER_DOCS =
  process.env.TACTIQUE_MESURE_DOCS ??
  join(racineDuDepot(), 'data', 'cache', 'replays', 'halo_infinite')

const t = getTacticalText('fr')
const JOUEUR = 'JoueurA'
const AUTRE = 'JoueurB'
/**
 * Le périmètre servi : 20 matchs sans borne pour A, 7 pour B ; avec une borne de début, un
 * compte qui dépend du MOIS de la borne (deux périodes différentes = deux listes différentes,
 * donc deux lectures de grille — jamais une réponse resservie par le cache).
 */
function perimetreServi(joueur: string, debut: string | null | undefined): string[] {
  const prefixe = joueur === JOUEUR ? 'a' : 'b'
  const n = debut ? (joueur === JOUEUR ? 4 : 1) + Number(debut.slice(5, 7)) : joueur === JOUEUR ? 20 : 7
  return Array.from({ length: n }, (_, i) => `${prefixe}-${i}`)
}

interface Carte {
  id: string
  nom: string
  fond: ReplayMapBackground
}
interface Film {
  id: string
  pts: { u: number; v: number; equipe: number }[]
  equipes: number[]
}

function catalogue(): Carte[] {
  return readdirSync(DOSSIER_FONDS)
    .filter((f) => f.endsWith('.json'))
    .sort()
    .map((f) => {
      const fond = JSON.parse(readFileSync(join(DOSSIER_FONDS, f), 'utf8')) as ReplayMapBackground
      return { id: fond.module, nom: fond.mapNames?.[0] ?? fond.module, fond }
    })
}

function listeDesFilms(): string[] {
  return readdirSync(DOSSIER_DOCS)
    .filter((f) => /^[0-9a-f]{8}\.json$/.test(f))
    .sort()
}

/** L'empreinte d'un document : ses trajectoires normalisées dans [0,1]², ≤ 3 000 points. */
function empreinte(fichier: string): Film {
  const doc = JSON.parse(readFileSync(join(DOSSIER_DOCS, fichier), 'utf8')) as {
    matchId: string
    bounds: { minX: number; minY: number; maxX: number; maxY: number }
    tracks: { team: number; points: { x: number; y: number }[] }[] | null
  }
  const b = doc.bounds
  const w = b.maxX - b.minX || 1
  const h = b.maxY - b.minY || 1
  const pistes = doc.tracks ?? []
  const total = pistes.reduce((s, p) => s + (p.points?.length ?? 0), 0)
  const pas = Math.max(1, Math.ceil(total / 3000))
  const pts: Film['pts'] = []
  let i = 0
  for (const piste of pistes) {
    for (const p of piste.points ?? []) {
      if (i++ % pas === 0) pts.push({ u: (p.x - b.minX) / w, v: (p.y - b.minY) / h, equipe: piste.team })
    }
  }
  const equipes = [...new Set(pistes.map((p) => p.team))].sort()
  return { id: doc.matchId.slice(0, 8), pts, equipes }
}

const QUESTIONS = ['morts', 'kills', 'gagne', 'temps', 'routes', 'isole']

/** La réponse du raster : la répartition du film recalée dans le cadre de la carte. */
function raster(carte: Carte | null, film: Film, corps: Record<string, unknown>): TacticalRaster {
  const cal = carte?.fond.calibration
  const x0 = cal ? cal.originX : -50
  const w = cal ? cal.widthPx * cal.metersPerPixel : 100
  const yHaut = cal ? cal.originY : 40
  const h = cal ? cal.heightPx * cal.metersPerPixel : 80
  const pas = w < 60 ? 0.5 : w < 150 ? 1 : 2
  const question = String(corps.question ?? 'morts')
  const qui = String(corps.qui ?? 'moi')
  const spawn = corps.spawn as string | undefined
  const n = (corps.match_ids as string[]).length
  const q = QUESTIONS.indexOf(question)
  const cumul = new Map<string, { col: number; lig: number; c: number }>()
  film.pts.forEach((p, i) => {
    if (i % 6 === q) return
    if (qui === 'adv' && p.equipe === film.equipes[0]) return
    if (qui === 'escouade' && p.equipe !== film.equipes[0]) return
    if (spawn === 'g1' && p.u >= 0.5) return
    if (spawn === 'g2' && p.u < 0.5) return
    if (i % 20 >= n) return
    // 3 % des points volontairement HORS du cadre : la note « hors cadre » est servie.
    const x = x0 + (i % 33 === 0 ? 1.1 : p.u) * w
    const y = yHaut - h + p.v * h
    const col = Math.floor(x / pas)
    const lig = Math.floor(y / pas)
    const k = `${col}:${lig}`
    const c = cumul.get(k) ?? { col, lig, c: 0 }
    c.c++
    cumul.set(k, c)
  })
  const signe = question === 'gagne'
  const cellules = [...cumul.values()].map((c) => {
    const matchs = Math.min(c.c, 8)
    const v = signe ? ((c.col + c.lig) % 2 ? c.c : -c.c) : c.c
    return {
      col: c.col,
      lig: c.lig,
      valeur: v,
      brut: c.c,
      matchs,
      matchs_victoire: Math.floor(matchs / 2),
      matchs_defaite: matchs - Math.floor(matchs / 2),
      centre_x: (c.col + 0.5) * pas,
      centre_y: (c.lig + 0.5) * pas,
    }
  })
  const valeurs = cellules.map((c) => Math.abs(c.valeur)).sort((a, b) => a - b)
  const quantile = (f: number) => valeurs[Math.min(valeurs.length - 1, Math.floor(f * valeurs.length))] ?? 0
  const retenus = n > 3 ? n - 1 : n
  const couverture = (num: number) => ({
    brut: num,
    n: Math.max(1, cellules.length),
    taux: num / Math.max(1, cellules.length),
    par_match: num / Math.max(1, retenus),
    echantillon_faible: retenus < 5,
  })
  return {
    map_id: carte?.id ?? 'sans-fond',
    question,
    qui,
    bornes: { min_x: x0, max_x: x0 + w, min_y: yHaut - h, max_y: yHaut, valide: true },
    pas_m: pas,
    echelle: { p50: quantile(0.5), p95: quantile(0.95), borne: quantile(0.95), n_cellules: cellules.length, symetrique: signe },
    cellules,
    grappes: [
      { id: 'g1', nom_fr: 'Ouest', nom_en: 'West', matchs: n, x: x0 + w / 4, y: yHaut - h / 2 },
      { id: 'g2', nom_fr: 'Est', nom_en: 'East', matchs: n, x: x0 + (3 * w) / 4, y: yHaut - h / 2 },
    ],
    isolement: couverture(Math.floor(cellules.length / 3)),
    echange: couverture(Math.floor(cellules.length / 4)),
    matchs_filtres: n,
    matchs_retenus: retenus,
    matchs_victoire: Math.floor(retenus / 2),
    matchs_defaite: retenus - Math.floor(retenus / 2),
    evenements_journal: film.pts.length,
    evenements_localises: film.pts.length,
    points_ignores: 0,
  }
}

function pageDesCartes(cartes: Carte[], n: number): TacticalMapsPage {
  return {
    plancher_matchs: 3,
    cartes: cartes.map<TacticalMapCard>((c, i) => ({
      map_id: c.id,
      map_name: c.nom,
      map_name_fr: c.nom,
      matchs: n + (i % 7),
      victoires: Math.floor((n + (i % 7)) / 2),
      defaites: Math.ceil((n + (i % 7)) / 2),
      sous_plancher: false,
    })),
  }
}

// ─── Lecture du DOM ────────────────────────────────────────────────────────────

function imgDuPlan(): HTMLImageElement | null {
  return (screen.queryByTestId('tactical-plan-frame')?.querySelector('img') as HTMLImageElement | null) ?? null
}
function texte(id: string): string | null {
  return screen.queryByTestId(id)?.textContent ?? null
}

interface Instant {
  imgPresent: boolean
  imgMemeNoeud: boolean
  imgCarte: string | null
  indicateurPremierChargement: boolean
  miseAJour: boolean
  kpi: string | null
  erreur: boolean
  titre: string | null
  titreAvecNom: boolean
  titreSurIdentifiant: boolean
  question: string | null
}

function instant(ref: HTMLImageElement | null, carte: { id: string; nom: string }): Instant {
  const img = imgDuPlan()
  const titre = texte('tactical-analysis-title')
  const select = screen.queryByRole('combobox', { name: t.questionLabel }) as HTMLSelectElement | null
  return {
    imgPresent: img !== null,
    imgMemeNoeud: ref !== null && img === ref && ref.isConnected,
    imgCarte: img ? (img.getAttribute('src') ?? '').replace('blob:fond/', '') : null,
    indicateurPremierChargement: screen.queryByTestId('tactical-analysis-pending') !== null,
    miseAJour: screen.queryByTestId('tactical-analysis-updating') !== null,
    kpi: texte('kpi-strip'),
    erreur: screen.queryByText(t.analysisErrorTitle) !== null,
    titre,
    titreAvecNom: titre !== null && titre.includes(carte.nom),
    // Une carte dont le catalogue n'a pas de nom (nom = identifiant) ne compte pas.
    titreSurIdentifiant: titre !== null && carte.nom !== carte.id && titre.includes(carte.id),
    question: select?.value ?? null,
  }
}

/** Ce qui ne doit PAS bouger entre la base et le lot, une fois la réponse servie. */
function rendu(): Record<string, string | null> {
  const cadre = screen.queryByTestId('tactical-plan-frame') as HTMLElement | null
  return {
    titre: texte('tactical-analysis-title'),
    kpi: texte('kpi-strip'),
    legende: texte('tactical-plan-legend'),
    echelle: texte('tactical-plan-scale-note'),
    retenus: texte('tactical-plan-retained'),
    pas: texte('tactical-plan-grid-step'),
    horsCadre: texte('tactical-plan-off-frame'),
    canvas: screen.queryByTestId('tactical-plan-canvas') ? 'oui' : null,
    cadre: cadre ? `${cadre.style.aspectRatio}|${cadre.style.maxWidth}` : null,
    img: imgDuPlan()?.getAttribute('src') ?? null,
    erreur: screen.queryByText(t.analysisErrorTitle) ? 'oui' : null,
  }
}

async function stable(): Promise<void> {
  await waitFor(
    () => {
      expect(screen.queryByTestId('tactical-analysis-pending')).toBeNull()
      expect(screen.queryByTestId('tactical-analysis-updating')).toBeNull()
      const pret = screen.queryByTestId('kpi-strip') !== null || screen.queryByText(t.analysisErrorTitle) !== null
      expect(pret).toBe(true)
      expect(appels.every((a) => a.regle)).toBe(true)
    },
    { timeout: 3000 },
  )
}

async function attendreAppel(pred: (a: Appel) => boolean): Promise<void> {
  await waitFor(() => expect(appels.some((a) => !a.regle && pred(a))).toBe(true), { timeout: 3000 })
}

function reglerTout(echec: (a: Appel) => boolean = () => false): void {
  for (const a of appels) if (!a.regle) regler(a, echec(a))
}

function regler(a: Appel, echoue: boolean): void {
  if (echoue) a.rejeter()
  else a.resoudre()
}

const estRaster = (a: Appel) => a.path.endsWith('/raster')
const estPerimetre = (a: Appel) => a.path.endsWith('/filters/match-ids')

function repondeur(cartes: Carte[], film: Film, carteDe: () => Carte | null, extra: Carte[] = []) {
  return (path: string, corps: unknown): unknown => {
    const joueur = /^\/players\/([^/]+)\//.exec(path)?.[1] ?? ''
    if (path.endsWith('/filters/match-ids')) {
      return { match_ids: perimetreServi(joueur, (corps as { period?: { start_date?: string | null } }).period?.start_date) }
    }
    if (path.endsWith('/tactical/maps')) return pageDesCartes([...cartes, ...extra], (corps as { match_ids: string[] }).match_ids.length)
    if (path.endsWith('/raster')) return raster(carteDe(), film, corps as Record<string, unknown>)
    throw new Error(`appel inattendu : ${path}`)
  }
}

function monter() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const wrapper = ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client }, children)
  const r = render(createElement(TacticalPage), { wrapper })
  return {
    rerendre: () => r.rerender(createElement(TacticalPage)),
    demonter: () => {
      r.unmount()
      client.clear()
    },
  }
}

// ─── Scénarios ─────────────────────────────────────────────────────────────────

interface Ligne {
  scenario: string
  carte: string
  film: string
  transition: string
  pendant: Instant[]
  apres: Record<string, string | null> | null
  apresInstant: Instant | null
  exception?: string
}

function reinitialiser(carte: string) {
  useAppShellStore.setState({ locale: 'fr', currentTitleSlug: 'halo_infinite' })
  searchCourant = carte ? { carte } : {}
  paramsCourants = { playerSlug: JOUEUR, titleSlug: 'halo_infinite' }
  localStorage.clear()
  appels.length = 0
  auto = true
}

/** L'écran d'analyse d'UNE carte (avec ou sans fond), rejoué sur toutes ses transitions. */
async function scenarioAnalyse(
  nom: string,
  cartes: Carte[],
  carte: Carte | null,
  suivante: Carte,
  film: Film,
): Promise<Ligne[]> {
  const lignes: Ligne[] = []
  const id = carte?.id ?? `sans-fond-${film.id}`
  let courante: { id: string; nom: string } = carte ?? { id, nom: `Carte ${film.id}` }
  let carteServie: Carte | null = carte
  fondDe = (m) => cartes.find((c) => c.id === m)?.fond ?? null
  // La carte sans fond figé est dans la grille (elle a été jouée), sans calage ni image.
  const sansFond: Carte[] = carte ? [] : [{ id, nom: `Carte ${film.id}`, fond: null as unknown as ReplayMapBackground }]
  repondre = repondeur(cartes, film, () => carteServie, sansFond)
  reinitialiser(id)
  const page = monter()
  let ref: HTMLImageElement | null = null
  const ligne = (transition: string): Ligne => ({ scenario: nom, carte: courante.id, film: film.id, transition, pendant: [], apres: null, apresInstant: null })

  /** Joue une transition : déclenche, observe à chaque appel attendu, puis règle et relève. */
  async function jouer(
    transition: string,
    declencher: () => void,
    etapes: ((a: Appel) => boolean)[],
    echec: (a: Appel) => boolean = () => false,
  ) {
    const l = ligne(transition)
    lignes.push(l)
    try {
      auto = false
      declencher()
      for (const etape of etapes) {
        await attendreAppel(etape)
        l.pendant.push(instant(ref, courante))
        for (const a of appels) if (!a.regle && etape(a)) regler(a, echec(a))
      }
      auto = true
      reglerTout(echec)
      await stable()
      // Une réponse servie sur une carte à fond : on attend aussi SON image (le blob est servi
      // sans délai, mais son rendu peut suivre celui des KPI d'un tour).
      if (fondDe(courante.id) && !screen.queryByText(t.analysisErrorTitle)) {
        await waitFor(() => expect(imgDuPlan()?.getAttribute('src')).toBe(`blob:fond/${courante.id}`), { timeout: 3000 })
      }
      l.apres = rendu()
      l.apresInstant = instant(ref, courante)
      ref = imgDuPlan() ?? ref
    } catch (e) {
      auto = true
      reglerTout()
      l.exception = String(e).slice(0, 300)
    }
  }

  try {
    // T0 — premier chargement.
    const l0 = ligne('premier-chargement')
    lignes.push(l0)
    try {
      await stable()
      if (carte) await waitFor(() => expect(imgDuPlan()).not.toBeNull())
      l0.apres = rendu()
      ref = imgDuPlan()
      l0.apresInstant = instant(ref, courante)
    } catch (e) {
      l0.exception = String(e).slice(0, 300)
    }

    const select = (label: string) => screen.getByRole('combobox', { name: label }) as HTMLSelectElement
    await jouer('question', () => fireEvent.change(select(t.questionLabel), { target: { value: 'kills' } }), [estRaster])
    await jouer('qui', () => fireEvent.click(screen.getByRole('button', { name: t.whoOpponents })), [estRaster])
    await jouer('spawn', () => fireEvent.change(select(t.spawnLabel), { target: { value: 'g1' } }), [estRaster])
    await jouer(
      'filtre-periode',
      () => {
        searchCourant = { ...searchCourant, de: '2026-01-01' }
        page.rerendre()
      },
      [estPerimetre, estRaster],
    )
    // Échec d'une relecture, puis retour : l'écran ne doit pas rester figé.
    await jouer('echec-relecture', () => fireEvent.change(select(t.questionLabel), { target: { value: 'temps' } }), [estRaster], estRaster)
    await jouer('reprise-apres-echec', () => fireEvent.change(select(t.questionLabel), { target: { value: 'routes' } }), [estRaster])
    // Changement de joueur, même carte.
    await jouer(
      'joueur',
      () => {
        paramsCourants = { ...paramsCourants, playerSlug: AUTRE }
        page.rerendre()
      },
      [estPerimetre, estRaster],
    )
    // Changement de carte.
    await jouer(
      'carte',
      () => {
        carteServie = suivante
        courante = suivante
        searchCourant = { ...searchCourant, carte: suivante.id }
        page.rerendre()
      },
      [estRaster],
    )
  } finally {
    page.demonter()
    cleanup()
  }
  return lignes
}

interface LigneGrille {
  transition: string
  vignettesMemeNoeud: number
  vignettesMonteesPendant: number
  fondsMemeNoeud: number
  chargementAffiche: boolean
  miseAJour: boolean
  erreur: boolean
  apres: string[]
  vignettesDAutreJoueur: number
  exception?: string
}

/** L'écran grille : les vignettes de TOUT le catalogue, sur un filtre, un échec, un joueur. */
async function scenarioGrille(cartes: Carte[], film: Film): Promise<LigneGrille[]> {
  fondDe = (m) => cartes.find((c) => c.id === m)?.fond ?? null
  repondre = repondeur(cartes, film, () => null)
  reinitialiser('')
  const page = monter()
  const lignes: LigneGrille[] = []
  const vignette = (id: string) => screen.queryByTestId(`tactical-map-${id}`)
  const fond = (id: string) => screen.queryByTestId(`tactical-map-fond-${id}`)
  const photo = () => cartes.map((c) => ({ v: vignette(c.id), f: fond(c.id) }))
  const textes = () => cartes.map((c) => vignette(c.id)?.textContent ?? '∅')
  try {
    await waitFor(() => expect(cartes.every((c) => fond(c.id) !== null)).toBe(true), { timeout: 5000 })
    lignes.push({ transition: 'premier-chargement', vignettesMemeNoeud: 0, vignettesMonteesPendant: 0, fondsMemeNoeud: 0, chargementAffiche: false, miseAJour: false, erreur: false, apres: textes(), vignettesDAutreJoueur: 0 })

    async function jouer(transition: string, declencher: () => void, echec: boolean) {
      const avant = photo()
      const l: LigneGrille = { transition, vignettesMemeNoeud: 0, vignettesMonteesPendant: 0, fondsMemeNoeud: 0, chargementAffiche: false, miseAJour: false, erreur: false, apres: [], vignettesDAutreJoueur: 0 }
      lignes.push(l)
      const nAvant = appels.length
      try {
        auto = false
        declencher()
        await attendreAppel(estPerimetre)
        l.vignettesMonteesPendant = cartes.filter((c) => vignette(c.id) !== null).length
        l.vignettesMemeNoeud = avant.filter((p, i) => p.v !== null && p.v === vignette(cartes[i].id)).length
        l.fondsMemeNoeud = avant.filter((p, i) => p.f !== null && p.f === fond(cartes[i].id)).length
        l.chargementAffiche = screen.queryByText(t.loading) !== null
        l.miseAJour = screen.queryByTestId('tactical-grille-updating') !== null
        auto = true
        reglerTout((a) => echec && estPerimetre(a))
        if (echec) await screen.findByTestId('tactical-erreur')
        else {
          // La relecture est FINIE : la grille du nouveau périmètre a été demandée ET servie,
          // plus rien n'est « en chargement » ni « mise à jour ».
          await waitFor(
            () => {
              expect(appels.slice(nAvant).some((a) => a.path.endsWith('/tactical/maps'))).toBe(true)
              expect(appels.every((a) => a.regle)).toBe(true)
              expect(screen.queryByText(t.loading)).toBeNull()
              expect(screen.queryByTestId('tactical-grille-updating')).toBeNull()
              expect(cartes.every((c) => fond(c.id) !== null)).toBe(true)
            },
            { timeout: 5000 },
          )
        }
        await waitFor(() => expect(appels.every((a) => a.regle)).toBe(true))
        l.erreur = screen.queryByTestId('tactical-erreur') !== null
        l.apres = textes()
      } catch (e) {
        auto = true
        reglerTout()
        l.exception = String(e).slice(0, 300)
      }
    }

    await jouer('filtre-periode', () => {
      searchCourant = { de: '2026-01-01' }
      page.rerendre()
    }, false)
    await jouer('echec-relecture', () => {
      searchCourant = { de: '2026-02-01', a: '2026-03-01' }
      page.rerendre()
    }, true)
    await jouer('reprise-apres-echec', () => {
      searchCourant = { de: '2026-04-01' }
      page.rerendre()
    }, false)
    await jouer('joueur', () => {
      paramsCourants = { ...paramsCourants, playerSlug: AUTRE }
      page.rerendre()
    }, false)
    // Aucune lecture de l'autre joueur ne porte un match_id du premier.
    const fuites = appels.filter(
      (a) => a.path.startsWith(`/players/${AUTRE}/tactical/`) && ((a.corps as { match_ids?: string[] }).match_ids ?? []).some((m) => m.startsWith('a-')),
    ).length
    lignes[lignes.length - 1].vignettesDAutreJoueur = fuites
  } finally {
    page.demonter()
    cleanup()
  }
  return lignes
}

// ─── La mesure ─────────────────────────────────────────────────────────────────

describe.skipIf(!PORTE)('mesure — le fond du plan tactique ne se démonte plus (L2, parc)', () => {
  let cartes: Carte[] = []
  let films: string[] = []
  let urlSpy: ReturnType<typeof vi.spyOn>
  const analyses: Ligne[] = []
  let grille: LigneGrille[] = []

  beforeAll(() => {
    if (!existsSync(DOSSIER_FONDS) || !existsSync(DOSSIER_DOCS)) {
      throw new Error(`dossiers absents : ${DOSSIER_FONDS} / ${DOSSIER_DOCS}`)
    }
    cartes = catalogue()
    films = listeDesFilms()
    if (LIMITE > 0) films = films.slice(0, LIMITE)
    urlSpy = vi.spyOn(URL, 'createObjectURL').mockImplementation((b) => `blob:fond/${blobs.get(b as Blob) ?? '?'}`)
  })

  afterAll(() => {
    urlSpy?.mockRestore()
    const sortie = process.env.TACTIQUE_MESURE_SORTIE
    if (sortie) writeFileSync(sortie, JSON.stringify({ cartes: cartes.length, films: films.length, analyses, grille }, null, 1))
  })

  it(
    'chaque film du parc, sur une carte du catalogue (toutes couvertes), et un film sur dix sans fond',
    async () => {
      for (let j = 0; j < films.length; j++) {
        const film = empreinte(films[j]) // un document à la fois
        const carte = cartes[j % cartes.length]
        const suivante = cartes[(j + 1) % cartes.length]
        analyses.push(...(await scenarioAnalyse('avec-fond', cartes, carte, suivante, film)))
        if (j % 10 === 0) analyses.push(...(await scenarioAnalyse('sans-fond', cartes, null, suivante, film)))
      }
      // Toute carte du catalogue non encore couverte (moins de films que de cartes) l'est ici.
      for (let k = films.length; LIMITE === 0 && k < cartes.length; k++) {
        const film = empreinte(films[k % films.length])
        analyses.push(...(await scenarioAnalyse('avec-fond', cartes, cartes[k], cartes[(k + 1) % cartes.length], film)))
      }
      grille = await scenarioGrille(cartes, empreinte(films[0]))

      const exceptions = analyses.filter((l) => l.exception).length + grille.filter((l) => l.exception).length
      console.log(`[mesure L2] ${cartes.length} cartes, ${films.length} films, ${analyses.length} transitions, ${exceptions} exceptions`)
      if (EXIGE) expect(exceptions).toBe(0)
    },
    60 * 60 * 1000,
  )

  it.skipIf(!EXIGE)('invariants du lot sur tout le parc', () => {
    const relectures = ['question', 'qui', 'spawn', 'filtre-periode', 'reprise-apres-echec']
    for (const l of analyses) {
      const avecFond = l.scenario === 'avec-fond'
      for (const p of l.pendant) {
        // (2) jamais le fond ni la réponse d'une autre carte.
        if (p.imgCarte !== null) expect(p.imgCarte, `${l.carte} ${l.transition}`).toBe(l.carte)
        if (relectures.includes(l.transition)) {
          // (1) le fond reste le même nœud ; aucun indicateur ne remplace le plan.
          if (avecFond) expect(p.imgMemeNoeud, `${l.carte} ${l.transition}`).toBe(true)
          expect(p.indicateurPremierChargement, `${l.carte} ${l.transition}`).toBe(false)
          expect(p.miseAJour, `${l.carte} ${l.transition}`).toBe(true)
          expect(p.titreSurIdentifiant, `${l.carte} ${l.transition}`).toBe(false)
        }
        if (l.transition === 'joueur' || l.transition === 'carte') expect(p.kpi, `${l.carte} ${l.transition}`).toBeNull()
        if (l.transition === 'joueur' && avecFond) expect(p.imgMemeNoeud, `${l.carte} joueur`).toBe(true)
      }
      // (3) l'échec se dit, et ne fige pas : la transition suivante revient à une réponse.
      if (l.transition === 'echec-relecture') {
        expect(l.apresInstant?.erreur, l.carte).toBe(true)
        expect(l.apresInstant?.miseAJour, l.carte).toBe(false)
        if (avecFond) expect(l.apresInstant?.imgPresent, l.carte).toBe(true)
      }
      if (l.transition === 'reprise-apres-echec') {
        expect(l.apresInstant?.erreur, l.carte).toBe(false)
        expect(l.apresInstant?.kpi, l.carte).not.toBeNull()
      }
      if (l.transition === 'carte') expect(l.apresInstant?.question, l.carte).toBe('morts')
      if (l.apres && !l.exception) expect(l.apresInstant?.titreSurIdentifiant, `${l.carte} ${l.transition}`).toBe(false)
    }
    for (const g of grille) expect(g.vignettesDAutreJoueur).toBe(0)
  })
})
