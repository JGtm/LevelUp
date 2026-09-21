/**
 * GARDE-RAIL i18n de la RIPOSTE — parité FR/EN et vocabulaire FR.
 *
 * Le build des manifests vérifie déjà que chaque clé porte ses deux langues ; ce
 * test-ci garde ce que le build ne peut pas voir :
 *
 *   1. aucune des deux langues n'est VIDE (une clé `en = ""` passe le build et rend
 *      une chaîne vide à l'écran) ;
 *   2. le vocabulaire FR arrêté par D19 est tenu : « riposte », « riposter », « taux de
 *      riposte ». Ni « échange », ni « vengeance », ni « trade » ;
 *   3. chaque clé déclarée est bien SERVIE par `getSquadRiposteText` — une clé
 *      orpheline dans le manifest est une chaîne que personne ne verra jamais, et
 *      une clé manquante s'afficherait telle quelle (formatMessage rend la clé) ;
 *   4. chaque accesseur exposé est AFFICHÉ par un composant — le volet 3 s'arrête à
 *      l'accesseur, celui-ci va jusqu'au bout de la chaîne.
 */
import { describe, expect, it } from 'vitest'

import { squadManifest } from '@/lib/i18n/generated/squad'

import carteSource from './SquadRiposteCard?raw'
import delaiSource from './SquadRiposteDelaiPanel?raw'
import matriceSource from './SquadRiposteMatricePanel?raw'
import pageSource from './SquadSynergiesPage?raw'
import accesseurSource from './squadRiposteStrings?raw'
import { getSquadRiposteText } from './squadRiposteStrings'

const PREFIXE = 'squad.riposte.'

/** Mots bannis des chaînes FR par D19 (et l'anglicisme historique du chantier). */
const BANNIS_FR = [/\btrades?\b/i, /\bheat\s?maps?\b/i, /\bkills?\b/i, /veng/i, /\béchanges?\b/i]

const entrees = Object.entries(squadManifest).filter(([k]) => k.startsWith(PREFIXE))

describe('manifest squad.riposte.*', () => {
  it('déclare des clés (sentinelle anti-vacuité)', () => {
    expect(entrees.length).toBeGreaterThan(20)
  })

  it('porte FR ET EN non vides sur chaque clé', () => {
    const vides = entrees.filter(([, v]) => !v.fr?.trim() || !v.en?.trim()).map(([k]) => k)
    expect(vides, `clés sans FR ou sans EN : ${vides.join(', ')}`).toEqual([])
  })

  it('tient le vocabulaire D19 : ni « vengeance », ni « échange », ni « trade » en FR', () => {
    const fautifs = entrees
      .filter(([, v]) => BANNIS_FR.some((re) => re.test(v.fr)))
      .map(([k, v]) => `${k} = « ${v.fr} »`)
    expect(fautifs, `vocabulaire FR arrêté : « riposte » — ${fautifs.join(' ; ')}`).toEqual([])
  })

  it('résout toutes les clés qu’il sert : aucune ne s’affiche telle quelle', () => {
    const t = getSquadRiposteText('fr') as Record<string, unknown>
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
    const orphelines = entrees.map(([k]) => k).filter((k) => !accesseurSource.includes(`'${k}'`))
    expect(orphelines, `clés déclarées mais non servies : ${orphelines.join(', ')}`).toEqual([])
  })

  it('n’expose aucun accesseur que RIEN N’AFFICHE', () => {
    const composants = [carteSource, delaiSource, matriceSource, pageSource].join('\n')
    const accesseurs = [...accesseurSource.matchAll(/^\s{4}([A-Za-z][A-Za-z0-9]*):/gm)].map(
      (m) => m[1],
    )
    expect(accesseurs.length, 'aucun accesseur détecté : le garde ne garde rien').toBeGreaterThan(20)
    // FRONTIÈRE DE MOT, et pas `includes` (correction R2 du 2026-09-06). Une recherche par
    // SOUS-CHAÎNE trouait le garde en silence : `t.delayBinOpen` couvrait `t.delayBin`.
    // Le PRÉFIXE d'un accesseur n'est pas cet accesseur.
    const affiche = (a: string) => new RegExp(`\\bt(?:Riposte)?\\.${a}\\b`).test(composants)
    const jamaisAffiches = accesseurs.filter((a) => !affiche(a))
    expect(
      jamaisAffiches,
      `accesseurs exposés mais affichés nulle part : ${jamaisAffiches.join(', ')}`,
    ).toEqual([])
  })
})
