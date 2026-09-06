/**
 * GARDE-RAIL i18n de l'échange — parité FR/EN et vocabulaire FR.
 *
 * Le build des manifests vérifie déjà que chaque clé porte ses deux langues ; ce
 * test-ci garde ce que le build ne peut pas voir :
 *
 *   1. aucune des deux langues n'est VIDE (une clé `en = ""` passe le build et rend
 *      une chaîne vide à l'écran) ;
 *   2. le vocabulaire FR de CE chantier est tenu : « échange », « vengeance »,
 *      « riposte ». Le garde-rail global anti-anglicismes couvre `squad.toml` mais
 *      pas les mots propres à ce lot — `trade` et le mot anglais des cartes de
 *      chaleur, celui-ci parce que le plan (§1) l'inscrit au garde global via le
 *      lot C et qu'aucune chaîne FR d'ici ne doit le contenir d'ici là ;
 *   3. chaque clé déclarée est bien SERVIE par `getSquadEchangeText` — une clé
 *      orpheline dans le manifest est une chaîne que personne ne verra jamais, et
 *      une clé manquante s'afficherait telle quelle (formatMessage rend la clé).
 */
import { describe, expect, it } from 'vitest'

import { squadManifest } from '@/lib/i18n/generated/squad'

import capCardSource from './SquadEchangeCapCard?raw'
import delaiCardSource from './SquadEchangeDelaiCard?raw'
import kpiSource from './SquadEchangeKpi?raw'
import matrixCardSource from './SquadEchangeMatrixCard?raw'
import accesseurSource from './squadEchangeStrings?raw'
import { getSquadEchangeText } from './squadEchangeStrings'

const PREFIXE = 'squad.echange.'

/** Anglicismes propres à ce chantier, interdits dans les chaînes FR. */
const ANGLICISMES_FR = [/\btrades?\b/i, /\bheat\s?maps?\b/i, /\bkills?\b/i]

const entrees = Object.entries(squadManifest).filter(([k]) => k.startsWith(PREFIXE))

describe('manifest squad.echange.*', () => {
  it('déclare des clés (sentinelle anti-vacuité)', () => {
    expect(entrees.length).toBeGreaterThan(20)
  })

  it('porte FR ET EN non vides sur chaque clé', () => {
    const vides = entrees.filter(([, v]) => !v.fr?.trim() || !v.en?.trim()).map(([k]) => k)
    expect(vides, `clés sans FR ou sans EN : ${vides.join(', ')}`).toEqual([])
  })

  it('ne laisse aucun anglicisme de ce chantier dans les chaînes FR', () => {
    const fautifs = entrees
      .filter(([, v]) => ANGLICISMES_FR.some((re) => re.test(v.fr)))
      .map(([k, v]) => `${k} = « ${v.fr} »`)
    expect(
      fautifs,
      `FR sans anglicisme : « échange », « vengeance », « riposte » — ${fautifs.join(' ; ')}`,
    ).toEqual([])
  })

  it('résout toutes les clés qu’il sert : aucune ne s’affiche telle quelle', () => {
    // On résout tout l'objet `t` (les fonctions sont appelées avec des valeurs
    // neutres) et on vérifie qu'aucune valeur rendue n'est restée une CLÉ — c'est
    // ce que `formatMessage` renvoie quand la clé est absente du manifest, et cela
    // s'afficherait tel quel à l'écran.
    const t = getSquadEchangeText('fr') as Record<string, unknown>
    const rendus: string[] = []
    for (const valeur of Object.values(t)) {
      if (typeof valeur === 'string') rendus.push(valeur)
      else if (typeof valeur === 'function') {
        rendus.push(String((valeur as (...a: unknown[]) => string)(1, 1, 1, 1)))
      }
    }
    expect(rendus.length).toBeGreaterThan(20)
    const clesNonResolues = rendus.filter((r) => r.startsWith(PREFIXE))
    expect(clesNonResolues, `clés absentes du manifest : ${clesNonResolues.join(', ')}`).toEqual([])
  })

  it('ne déclare aucune clé ORPHELINE (déclarée au manifest, exposée par personne)', () => {
    // Une clé que `squadEchangeStrings` n'expose pas est une chaîne traduite deux
    // fois que personne ne verra jamais — et qui survivra à la suppression du
    // composant qui la justifiait.
    const orphelines = entrees
      .map(([k]) => k)
      .filter((k) => !accesseurSource.includes(`'${k}'`))
    expect(orphelines, `clés déclarées mais non servies : ${orphelines.join(', ')}`).toEqual([])
  })

  it('n’expose aucun accesseur que RIEN N’AFFICHE (correction W8)', () => {
    // Le volet precedent s'arretait a l'accesseur : `squad.echange.empty_description`
    // y passait — declaree, exposee par `emptyDescription`, et affichee par AUCUN
    // composant. Le garde annoncait donc « aucune orpheline » a tort. Il va desormais
    // jusqu'au bout de la chaine : manifest -> accesseur -> composant.
    const composants = [matrixCardSource, delaiCardSource, kpiSource, capCardSource].join(String.fromCharCode(10))
    const accesseurs = [...accesseurSource.matchAll(/^\s{4}([A-Za-z][A-Za-z0-9]*):/gm)].map(
      (m) => m[1],
    )
    expect(accesseurs.length, 'aucun accesseur détecté : le garde ne garde rien').toBeGreaterThan(20)
    const jamaisAffiches = accesseurs.filter((a) => !composants.includes(`t.${a}`))
    expect(
      jamaisAffiches,
      `accesseurs exposés mais affichés nulle part : ${jamaisAffiches.join(', ')}`,
    ).toEqual([])
  })
})
