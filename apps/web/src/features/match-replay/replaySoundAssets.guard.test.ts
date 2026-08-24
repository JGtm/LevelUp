/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6) : le manifeste sonore du rejeu (replaySound.ts pour les armes,
 * la mêlée et les équipements ; grenadeSound.ts pour les deux formes de la grenade) et le
 * dossier d'assets `static/sounds/halo_infinite/` sont la MÊME liste, rejouée ici.
 *
 * POURQUOI. Le client ne sonde jamais le serveur pour savoir si un son existe (pas de
 * 404 exploratoire) : il croit le manifeste. Un stem sans fichier serait un silence
 * inexpliqué en production ; un fichier sans stem, un asset mort que rien ne joue. Les
 * deux dérives cassent CE test, pas l'écoute.
 *
 * LE SECOND GARDE-RAIL traverse la frontière Go/TS, et il le doit : le son d'un kill à la
 * grenade ou à la mêlée se joint par la VIGNETTE de la source de dégât (killfeed-NN), dont
 * la table d'autorité est `killicon/data/rules.tsv` côté Go. Cette table GRANDIT à chaque
 * saison. Si un index d'atlas bougeait, ou si une cinquième grenade apparaissait, la
 * jointure du son deviendrait fausse EN SILENCE — c'est-à-dire qu'on entendrait l'explosion
 * d'une autre grenade. Rejouer les cinq clés contre le fichier source rend cette dérive
 * rouge au lieu de la rendre inaudible.
 *
 * LE TROISIÈME GARDE-RAIL tient la RÈGLE DE DURÉE PAR CATÉGORIE (lot R2, 2026-08-16), qui
 * ne vit plus dans le lecteur mais dans les FICHIERS : armes, lancers et mêlée à 1,2 s ;
 * explosions et équipements jusqu'à 4 s. Cette règle-là s'efface toute seule — il suffit
 * qu'une re-livraison retronque tout à 1,2 s (c'est exactement ce que fait le script de
 * livraison des armes) pour que les explosions redeviennent « écourtées » sans qu'aucun
 * test ne bouge. Ici, elle devient rouge.
 */
import { describe, expect, it } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { EXPLOSION_SOUND_STEMS, THROW_SOUND_STEMS } from './grenadeSound'
import { SOUND_CUT_MAX_S } from './replayAudio'
import {
  EQUIPMENT_PLACEMENT_SOUND_STEMS,
  EQUIPMENT_SOUND_STEMS,
  GRAPPLE_SOUND_STEM,
  KILL_SPRITE_SOUND_STEMS,
  WEAPON_SOUND_STEMS,
} from './replaySound'

const REPO_ROOT = resolve(__dirname, '..', '..', '..', '..', '..')
const SOUNDS_DIR = resolve(REPO_ROOT, 'static', 'sounds', 'halo_infinite')
const KILLICON_RULES = resolve(
  REPO_ROOT,
  'apps/go-api/internal/games/halo_infinite/film/killicon/data/rules.tsv',
)

/** Les stems EFFECTIVEMENT présents dans le dossier d'assets. */
const shipped = new Set(
  readdirSync(SOUNDS_DIR)
    .filter((f) => f.endsWith('.wav'))
    .map((f) => f.slice(0, -'.wav'.length)),
)

describe('garde-rail : manifeste sonore = dossier d assets', () => {
  const referenced = new Set([
    ...Object.values(WEAPON_SOUND_STEMS),
    ...THROW_SOUND_STEMS,
    // Les explosions de FIN DE VOL (lot R2-G, 2026-08-18) : mêmes fichiers que les quatre
    // explosions de kill, joints par le rang au lieu de la vignette.
    ...EXPLOSION_SOUND_STEMS,
    ...Object.values(KILL_SPRITE_SOUND_STEMS),
    // Les équipements (lot du 2026-08-16) : deux stems par famille mesurée.
    ...Object.values(EQUIPMENT_SOUND_STEMS).flatMap((s) => [s.activate, s.deactivate]),
    // Les POSES d'équipement (lot du 2026-08-18) : UN stem par famille — le geste de pose.
    ...Object.values(EQUIPMENT_PLACEMENT_SOUND_STEMS),
    // Le TIR de grappin (lot G, 2026-08-20) : UN SEUL stem, aucune famille.
    GRAPPLE_SOUND_STEM,
  ])

  it('chaque stem du manifeste a son fichier .wav', () => {
    const missing = [...referenced].filter((s) => !shipped.has(s))
    expect(missing).toEqual([])
  })

  it('chaque fichier .wav livré est joué par un stem (0 asset mort)', () => {
    const orphans = [...shipped].filter((s) => !referenced.has(s))
    expect(orphans).toEqual([])
  })
})

/**
 * Durée d'un WAV, lue dans ses en-têtes RIFF (aucune dépendance : les fichiers livrés sont
 * du PCM produit par une recette fixe — `.ai/V7.5/RECETTE_SONS_ARMES.md` §7 et le lot R2 —
 * et un fichier qui ne serait plus du PCM canonique doit faire ÉCHOUER ce test, pas être
 * deviné). `data` / `byteRate` : les deux seules quantités nécessaires.
 */
function wavDurationS(file: string): number {
  const buf = readFileSync(file)
  expect(buf.toString('latin1', 0, 4), file).toBe('RIFF')
  expect(buf.toString('latin1', 8, 12), file).toBe('WAVE')
  let byteRate = 0
  let dataSize = 0
  for (let at = 12; at + 8 <= buf.length; ) {
    const id = buf.toString('latin1', at, at + 4)
    const size = buf.readUInt32LE(at + 4)
    if (id === 'fmt ') byteRate = buf.readUInt32LE(at + 16)
    if (id === 'data') dataSize = size
    at += 8 + size + (size % 2)
  }
  expect(byteRate, `${file} : en-tete fmt illisible`).toBeGreaterThan(0)
  return dataSize / byteRate
}

describe('garde-rail : durée livrée par catégorie', () => {
  /** La coupe historique du lot 5, celle que gardent les sons COURTS par décision produit. */
  const COURT_S = 1.2
  const dureeDe = (stem: string) => wavDurationS(resolve(SOUNDS_DIR, `${stem}.wav`))

  // Les catégories, reconstruites depuis le manifeste lui-même : la mêlée partage la table
  // des vignettes avec les explosions, et c'est son STEM qui l'en distingue (une seule
  // ligne CLASSE MELEE dans rules.tsv, cf. garde-rail ci-dessus).
  const courts = [
    ...Object.values(WEAPON_SOUND_STEMS),
    ...THROW_SOUND_STEMS,
    ...Object.values(KILL_SPRITE_SOUND_STEMS).filter((s) => s === 'melee_kill'),
  ]
  const longs = [
    ...EXPLOSION_SOUND_STEMS,
    ...Object.values(KILL_SPRITE_SOUND_STEMS).filter((s) => s !== 'melee_kill'),
    ...Object.values(EQUIPMENT_SOUND_STEMS).flatMap((s) => [s.activate, s.deactivate]),
    // Les poses relèvent de la MÊME catégorie que les épisodes (Équipements) : elles gardent
    // la durée de leur source, jamais retronquée à la coupe des armes.
    ...Object.values(EQUIPMENT_PLACEMENT_SOUND_STEMS),
    // Le grappin (lot G, 2026-08-20) : catégorie Équipement, même règle. Sa source fait
    // 1,687 s, déjà au-dessus de la coupe des armes — aucune entrée SOURCES_COURTES requise.
    GRAPPLE_SOUND_STEM,
  ]

  it('aucun son ne dépasse le plafond de sûreté du lecteur', () => {
    const trop = [...shipped]
      .map((s) => ({ stem: s, s: dureeDe(s) }))
      .filter((d) => d.s > SOUND_CUT_MAX_S + 0.001)
    expect(trop).toEqual([])
  })

  it('armes, lancers et mêlée restent tronqués à 1,2 s (la coupe validée du lot 5)', () => {
    // 1,2 s est un PLAFOND, pas une longueur : deux sons d'arme sont plus courts parce
    // que leur source l'est (MA40 AR 1,055 s, MA5K Avenger 1,051 s — mesure du 16/08).
    const hors = courts.map((s) => ({ stem: s, s: dureeDe(s) })).filter((d) => d.s > COURT_S + 0.001)
    expect(hors).toEqual([])
  })

  /**
   * SOURCES PLUS COURTES QUE LA COUPE — la seule échappatoire, et elle est nominative.
   *
   * La règle « un son d'équipement dépasse 1,2 s » est un PROXY : ce qu'elle traque, c'est une
   * re-livraison qui retronquerait tout à la coupe des armes. Un fichier dont la SOURCE fait
   * moins que la coupe ne peut pas en être la victime — mais il échoue quand même le proxy.
   *
   * Chaque entrée porte donc la durée MESURÉE de sa source et la raison. `repair_field_activate`
   * (2026-08-18) : les trois variantes du `RandomSequence` de la banque `5724312f` font 0,31 /
   * 0,35 / 0,38 s dans le jeu — c'est un « pop » de déploiement, pas un son écourté. Allonger le
   * fichier serait inventer de la matière ; l'exclure de la catégorie mentirait sur sa nature.
   */
  const SOURCES_COURTES: Readonly<Record<string, number>> = {
    repair_field_activate: 0.381,
  }

  it('explosions et équipements gardent la durée de leur source, jamais retronquée à 1,2 s', () => {
    const retronques = longs
      .filter((s) => SOURCES_COURTES[s] === undefined)
      .map((s) => ({ stem: s, s: dureeDe(s) }))
      .filter((d) => d.s <= COURT_S)
    expect(retronques).toEqual([])
  })

  it('une source déclarée courte a bien la durée déclarée (sinon la dispense ne vaut plus)', () => {
    for (const [stem, attendu] of Object.entries(SOURCES_COURTES)) {
      expect(dureeDe(stem), stem).toBeCloseTo(attendu, 2)
    }
  })
})

/** Les lignes utiles de rules.tsv : genre, clé, vignette (colonnes 0, 1 et 2). */
function killiconRules(): { genre: string; key: string; sprite: string }[] {
  return readFileSync(KILLICON_RULES, 'utf8')
    .split('\n')
    .map((l) => l.trimEnd())
    .filter((l) => l !== '' && !l.startsWith('#') && !l.startsWith('genre\t'))
    .map((l) => l.split('\t'))
    .map(([genre, key, sprite]) => ({ genre, key, sprite }))
}

describe('garde-rail : vignettes du son de kill = table killicon (Go)', () => {
  const rules = killiconRules()

  /**
   * CE QUE CHAQUE VIGNETTE DOIT DÉSIGNER, verbatim depuis rules.tsv. GGGL N = l'entrée N de
   * la liste des grenades du jeu (0 Frag, 1 Plasma, 2 Dynamo, 3 Spike) ; CLASSE MELEE = le
   * geste partagé par tout l'arsenal, qu'aucune arme ne peut nommer.
   */
  const attendu: Record<string, { genre: string; key: string }> = {
    'killfeed-46': { genre: 'GGGL', key: '0' },
    'killfeed-47': { genre: 'GGGL', key: '1' },
    'killfeed-48': { genre: 'GGGL', key: '2' },
    'killfeed-49': { genre: 'GGGL', key: '3' },
    'killfeed-65': { genre: 'CLASSE', key: 'MELEE' },
  }

  it('la table sonore couvre exactement les vignettes attendues', () => {
    expect(Object.keys(KILL_SPRITE_SOUND_STEMS).sort()).toEqual(Object.keys(attendu).sort())
  })

  it('chaque vignette est portée par UNE SEULE règle, et par celle qu on croit', () => {
    for (const [sprite, veut] of Object.entries(attendu)) {
      const trouvees = rules.filter((r) => r.sprite === sprite)
      expect(trouvees, `vignette ${sprite}`).toHaveLength(1)
      expect({ genre: trouvees[0].genre, key: trouvees[0].key }, `vignette ${sprite}`).toEqual(veut)
    }
  })

  /**
   * LES DEUX TABLES D'EXPLOSION NE PEUVENT PAS DIVERGER (lot R2-G, 2026-08-18). Depuis que
   * la fin de vol d'une grenade sonne, le même fichier est atteint par DEUX chemins : le
   * RANG que le film publie avec le lancer (EXPLOSION_SOUND_STEMS) et la VIGNETTE que le
   * backend publie avec le kill (KILL_SPRITE_SOUND_STEMS). Les deux ordres coïncident parce
   * que `rules.tsv` reporte GGGL 0..3 sur killfeed-46..49 — le test au-dessus le vérifie du
   * côté Go, celui-ci le vérifie du côté TS. Si l'un des deux bougeait seul, la même grenade
   * sonnerait deux fichiers différents selon qu'elle tue ou non, EN SILENCE.
   */
  const VIGNETTE_PAR_RANG = ['killfeed-46', 'killfeed-47', 'killfeed-48', 'killfeed-49']

  it('rang du lancer et vignette du kill nomment le MÊME fichier d explosion', () => {
    expect(EXPLOSION_SOUND_STEMS).toHaveLength(VIGNETTE_PAR_RANG.length)
    VIGNETTE_PAR_RANG.forEach((sprite, rang) => {
      expect(KILL_SPRITE_SOUND_STEMS[sprite], `rang ${rang}`).toBe(EXPLOSION_SOUND_STEMS[rang])
    })
  })

  it('les quatre entrées gggl du jeu sont toutes servies (aucune grenade sans son)', () => {
    const gggl = rules.filter((r) => r.genre === 'GGGL').map((r) => r.key).sort()
    expect(gggl).toEqual(['0', '1', '2', '3'])
    for (const r of rules.filter((x) => x.genre === 'GGGL')) {
      expect(KILL_SPRITE_SOUND_STEMS[r.sprite], `gggl ${r.key}`).toBeTruthy()
    }
  })
})

/**
 * LE SON DU RÉPULSEUR : DEMANDÉ, MESURÉ IMPOSSIBLE AUJOURD'HUI, ET CE TEST EST LE FIL QUI
 * PRÉVIENDRA LE JOUR OÙ IL DEVIENDRA POSSIBLE (lot R2.4, 2026-08-16).
 *
 * CE QUI EXISTE : la vignette. L'atlas kill feed du jeu porte bien un pictogramme de
 * répulseur — `killfeed-56`, `nom_jeu: repulsor`, `tags_weap: ["692390e9"]` dans
 * `static/weapons-assets/halo_infinite/jeu/index.json`. Le JEU sait donc afficher un kill au
 * répulseur dans son propre fil.
 *
 * CE QUI MANQUE : la SOURCE. Le son d'un kill se joint par la vignette, et la vignette se
 * résout depuis le `jpt!` de la mort via `damagetag/data/labels.tsv` — les seules clés que
 * `rules.tsv` sait écrire (NOM, GGGL, BANQUE, CLASSE) sortent toutes de cette table. Or,
 * mesuré ligne à ligne le 2026-08-16 : aucune des 473 lignes de `labels.tsv` ne nomme le
 * répulseur (0 occurrence), les 114 lignes qui portent un nom propre sont TOUTES de classe
 * ARME, et les 6 lignes dont l'effet est un `eqip` sont toutes de statut INCONNU et sans nom
 * (trois d'entre elles — bcabbe43, caaadcb0, a875923f — sont d'ailleurs identifiées par la
 * rétro-ingénierie comme des entrées de la liste `gggl`, c'est-à-dire des GRENADES).
 *
 * POURQUOI ON N'ÉCRIT PAS LA RÈGLE QUAND MÊME : la seule clé qui « attraperait » ces lignes
 * serait `CLASSE INCONNU`, qui poserait le pictogramme ET le son du répulseur sur 206 sources
 * sans rapport. C'est la faute que la table refuse explicitement (« une icône absente est un
 * repli, une icône fausse est un mensonge », en-tête de rules.tsv) et que `neutral.go` refuse
 * déjà pour cinq autres pictogrammes de l'atlas qu'aucune donnée mesurée n'atteint.
 * Conséquence : pas de fichier `.wav` de répulseur livré non plus — un asset que rien ne
 * joue casserait le garde-rail « 0 asset mort » ci-dessus.
 *
 * CE QUE CE TEST FAIT : il devient ROUGE le jour où une règle mène à `killfeed-56`. Ce
 * jour-là, il ne reste que deux gestes — livrer `EQUIPMENT/Repulser - Activate (On Object)`
 * (1,85 s, la variante d'impact SUR UNE CIBLE, seule cohérente avec un kill) coupé par la
 * recette du lot, et ajouter la ligne à `KILL_SPRITE_SOUND_STEMS`.
 */
describe('garde-rail : le son du répulseur attend son identification (R2.4)', () => {
  const REPULSEUR = 'killfeed-56'

  it('aucune règle killicon ne mène encore au répulseur — sinon, brancher son son', () => {
    const versRepulseur = killiconRules()
      .filter((r) => r.sprite === REPULSEUR)
      .map((r) => `${r.genre} ${r.key}`)
    expect(
      versRepulseur,
      'killicon resout desormais le repulseur : livrer le wav et l ajouter a KILL_SPRITE_SOUND_STEMS',
    ).toEqual([])
  })
})
