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
 * explosions et équipements jusqu'à 4 s ; répliques et fanfares de FIN DE PARTIE entières
 * (lot C, 2026-08-27), la plus longue à 11,67 s. Cette règle-là s'efface toute seule — il
 * suffit qu'une re-livraison retronque tout à 1,2 s (c'est exactement ce que fait le script
 * de livraison des armes) pour que les explosions redeviennent « écourtées » sans qu'aucun
 * test ne bouge. Ici, elle devient rouge.
 */
import { describe, expect, it } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import {
  END_FFA_WIN_VOICE_STEMS,
  END_MUSIC_STEMS,
  END_VOICE_STEMS,
} from './endMatchSound'
import { EXPLOSION_SOUND_STEMS, THROW_SOUND_STEMS } from './grenadeSound'
import { SOUND_CUT_MAX_S } from './replayAudio'
import {
  EQUIPMENT_PLACEMENT_SOUND_STEMS,
  EQUIPMENT_SOUND_STEMS,
  GRAPPLE_SOUND_STEM,
  EQUIPMENT_PLACEMENT_SOUND_STEMS_END,
  KILL_SPRITE_SOUND_STEMS,
  OBJECTIVE_SOUND_STEMS,
  ROUND_OVER_SOUND_STEMS,
  SOUND_VARIANTS,
  WEAPON_SOUND_STEMS,
  ZONE_SOUND_STEMS,
  PAD_PICKUP_SOUND_STEM,
} from './replaySound'

/** Les stems d une entree d objectif : une PAIRE PEUT ETRE INCOMPLETE (le camp non design00e9 a
 *  l oreille reste muet), et le garde-rail ne doit pas r00e9clamer un fichier pour un stem absent. */
function stemsObjectif(v: { ally?: string; enemy?: string } | { any: string }): string[] {
  return 'any' in v ? [v.any] : [v.ally, v.enemy].filter((s): s is string => s !== undefined)
}

/** Tous les stems de ZONE_SOUND_STEMS, paires et stem seul confondus. */
function stemsZone(): string[] {
  return Object.values(ZONE_SOUND_STEMS).flatMap((v) =>
    typeof v === 'string' ? [v] : [v.ally, v.enemy],
  )
}

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

/**
 * LES SONS DE FIN DE PARTIE (lot C, 2026-08-27) : voix d'annonceur DANS LES DEUX LANGUES,
 * fanfares, et la réplique du FFA gagné. Les deux langues sont livrées ENSEMBLE et vérifiées
 * ENSEMBLE — un pack FR complet et un pack EN troué passerait inaperçu de tout ce qui monte
 * l'application en français.
 *
 * `end_victory_voice_en_01` figure DEUX fois dans ces tables (voix de victoire EN, et repli du
 * FFA gagné EN faute de « Winner » isolé dans le pack) : le `Set` du manifeste l'absorbe, et
 * c'est bien un seul fichier.
 */
const endMatchStems = [
  ...Object.values(END_VOICE_STEMS).flatMap((byLocale) => Object.values(byLocale).flat()),
  ...Object.values(END_MUSIC_STEMS),
  ...Object.values(END_FFA_WIN_VOICE_STEMS).flat(),
]

describe('garde-rail : manifeste sonore = dossier d assets', () => {
  const referenced = new Set([
    ...Object.values(WEAPON_SOUND_STEMS),
    ...THROW_SOUND_STEMS,
    // Les explosions de FIN DE VOL (lot R2-G, 2026-08-18) : mêmes fichiers que les quatre
    // explosions de kill, joints par le rang au lieu de la vignette.
    ...EXPLOSION_SOUND_STEMS,
    ...Object.values(KILL_SPRITE_SOUND_STEMS).map((v) => v.stem),
    // Les équipements (lot du 2026-08-16) : deux stems par famille mesurée.
    ...Object.values(EQUIPMENT_SOUND_STEMS).flatMap((s) => [s.activate, s.deactivate]),
    // Les POSES d'équipement (lot du 2026-08-18) : UN stem par famille — le geste de pose.
    ...Object.values(EQUIPMENT_PLACEMENT_SOUND_STEMS),
    // Et la FIN de pose, pour les familles qui ont un son d'extinction propre (2026-08-26).
    ...Object.values(EQUIPMENT_PLACEMENT_SOUND_STEMS_END),
    // Le TIR de grappin (lot G, 2026-08-20) : UN SEUL stem, aucune famille.
    GRAPPLE_SOUND_STEM,
    // Les ACTIONS D'OBJECTIF (lot du 2026-08-26) : un ou deux stems par statistique, selon
    // que le jeu distingue les camps.
    ...Object.values(OBJECTIVE_SOUND_STEMS).flatMap(stemsObjectif),
    // Les VARIANTES (lot du 2026-08-26) : un geste que le jeu tire dans un `RandomSequence`
    // livre TOUS ses fichiers, pas seulement le premier. Sans cette ligne, `grapple_fire_v2`
    // et `_v3` seraient des « assets morts » aux yeux du deuxième test — alors qu'ils sont
    // exactement ce que la table déclare jouer.
    ...Object.values(SOUND_VARIANTS).flat(),
    // Les sons d'ÉTAT DE ZONE (lot du 2026-08-27) : capture en cours, tic de domination,
    // déplacement de la colline. Source doc.zoneStates, pas doc.objectives — mais même
    // dossier d'assets et même garde-rail.
    ...stemsZone(),
    PAD_PICKUP_SOUND_STEM,
    // La FIN DE PARTIE (lot C, 2026-08-27) : voix FR et EN, fanfares, réplique du FFA gagné.
    ...endMatchStems,
    // Le son « MANCHE TERMINÉE » (2026-08-28) : voix d'annonceur FR et EN, sur la piste (daté à
    // la bascule de manche), les deux langues livrées et vérifiées ensemble.
    ...Object.values(ROUND_OVER_SOUND_STEMS),
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
  /**
   * LA BORNE DES SONS D'ÉVÉNEMENT, ex-plafond du lecteur (4 s) — elle est ÉCRITE ICI depuis le
   * lot C (2026-08-27) et ce n'est pas un doublon de `SOUND_CUT_MAX_S`.
   *
   * Jusqu'à ce lot, les deux nombres étaient le même : le plafond du lecteur bornait de fait
   * les explosions et les équipements, et un asset re-livré en pleine longueur devenait rouge.
   * Les fanfares de fin de partie ont fait monter le plafond à 12 s — sans cette borne-ci, une
   * explosion de grenade pourrait désormais passer à 9 s sans qu'aucun test ne bouge. Les deux
   * quantités ont cessé d'être la même chose le jour où le catalogue a porté deux familles de
   * durée : celle des ÉVÉNEMENTS du match, et celle de sa CONCLUSION.
   */
  // 4,0 -> 6,0 a la fusion du 2026-08-27 : la reconstitution des gestes Wwise livre des
  // EVENEMENTS jusqu'a 5,15 s (flag_taken_team 4,588 s, objective_zone_new 5,15 s) — la regle
  // « plafond = plus longue source livree, arrondie au-dessus » vaut aussi pour cette famille.
  const LONG_MAX_S = 6.0
  const dureeDe = (stem: string) => wavDurationS(resolve(SOUNDS_DIR, `${stem}.wav`))

  // Les catégories, reconstruites depuis le manifeste lui-même : la mêlée partage la table
  // des vignettes avec les explosions, et c'est son STEM qui l'en distingue (une seule
  // ligne CLASSE MELEE dans rules.tsv, cf. garde-rail ci-dessus).
  const courts = [
    ...Object.values(WEAPON_SOUND_STEMS),
    ...THROW_SOUND_STEMS,
    ...Object.values(KILL_SPRITE_SOUND_STEMS)
      .map((v) => v.stem)
      .filter((s) => s === 'melee_kill'),
  ]
  const longs = [
    ...EXPLOSION_SOUND_STEMS,
    // Le répulseur (catégorie ARME) tombe ici, PAS dans `courts` : le découpage court/long est
    // une propriété de la DURÉE DE LA SOURCE (lot R2, 2026-08-16), un axe différent de la
    // catégorie du filtre du tiroir. `repulsor_kill` (1,852 s, archive utilisateur) est de la
    // même famille de durée que les explosions et les équipements, pas des armes/lancers/mêlée.
    ...Object.values(KILL_SPRITE_SOUND_STEMS)
      .map((v) => v.stem)
      .filter((s) => s !== 'melee_kill'),
    ...Object.values(EQUIPMENT_SOUND_STEMS).flatMap((s) => [s.activate, s.deactivate]),
    // Les poses relèvent de la MÊME catégorie que les épisodes (Équipements) : elles gardent
    // la durée de leur source, jamais retronquée à la coupe des armes.
    ...Object.values(EQUIPMENT_PLACEMENT_SOUND_STEMS),
    ...Object.values(EQUIPMENT_PLACEMENT_SOUND_STEMS_END),
    // Le grappin : catégorie Équipement, même règle. SA SOURCE A CHANGÉ LE 2026-08-26 — le
    // fichier de l'archive utilisateur (1,687 s) est remplacé par le geste du JEU
    // (`play_007_abl_grapplinghook_deploy_player`, 0,745 s). Il passe donc SOUS la coupe des
    // armes et demande désormais une entrée `SOURCES_COURTES`, ce que l'ancienne rédaction de
    // ce commentaire déclarait inutile.
    GRAPPLE_SOUND_STEM,
    // Les ACTIONS D'OBJECTIF (lot du 2026-08-26) : catégorie Objectifs, même règle de durée que
    // les équipements — elles gardent la durée de leur source (1,31 à 3,41 s), jamais
    // retronquée. Un jingle de capture coupé à 1,2 s s'entendrait amputé de sa queue.
    ...Object.values(OBJECTIVE_SOUND_STEMS).flatMap(stemsObjectif),
    // Les sons d'ÉTAT DE ZONE : même catégorie, même règle de durée. Le tic de domination est
    // le seul son de la chaîne livré TRONQUÉ à dessein (1,2 s) et ATTÉNUÉ (-12 dBTP) : il se
    // joue une fois par seconde, un geste de 3,6 s s'y empilerait sur lui-même.
    ...stemsZone(),
    PAD_PICKUP_SOUND_STEM,
    // Le son « MANCHE TERMINÉE » : voix d'annonceur sur la piste (catégorie Objectifs), même
    // règle de durée que les autres annonceurs — il garde la durée de sa source (1,7 s), jamais
    // retronqué à la coupe des armes. Ce n'est PAS un son de FIN DE PARTIE : il vit sur la piste,
    // daté à la bascule, il tombe donc sous le plafond des ÉVÉNEMENTS (6 s), pas la conclusion.
    ...Object.values(ROUND_OVER_SOUND_STEMS),
    // Les VARIANTES d'un geste suivent la règle de leur geste : ce sont les autres tirages du
    // MÊME `RandomSequence`, pas d'autres sons.
    ...Object.values(SOUND_VARIANTS).flat(),
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
   * Chaque entrée porte donc la durée MESURÉE de sa source et la raison.
   *
   * `repair_field_activate` EST SORTI DE CETTE TABLE LE 2026-08-27, et il faut dire pourquoi
   * plutôt que de laisser une entrée périmée : le fichier livré n'est plus le même son. Il
   * portait le geste de POSE (`play_007_abl_repairfield_deploy_player`, 0,380 / 0,313 /
   * 0,349 s — un « pop » d'objet lâché) ; l'utilisateur, à l'écoute de la planche, a désigné
   * l'ACTIVATION (`play_007_abl_repairfield_activate`, 3,263 / 2,812 / 3,933 s) comme le son
   * qu'il veut entendre quand un joueur pose le champ. Les trois variantes livrées sont
   * désormais celles-là, et elles dépassent largement la coupe : plus de dispense à déclarer.
   *
   * `grapple_fire` et ses variantes (2026-08-26) : le geste du jeu
   * (`play_007_abl_grapplinghook_deploy_player`) fait 0,745 / 0,765 / 0,757 s. Il remplace le
   * fichier de l'archive utilisateur, qui faisait 1,687 s — d'où l'apparition de ces entrées.
   */
  const SOURCES_COURTES: Readonly<Record<string, number>> = {
    // LE SEUL SON LIVRÉ TRONQUÉ À DESSEIN, et il faut dire pourquoi : le tic de domination se
    // joue UNE FOIS PAR SECONDE tant qu un camp tient toutes les zones (règle produit du
    // 2026-08-27). Le geste du jeu dure 3,62 s côté allié et 4,36 s côté adverse — servi
    // entier, il s empilerait quatre fois sur lui-même. Il est donc coupé à 1,2 s avec un
    // fondu de 0,25 s, et atténué à -12 dBTP (« je les trouve un peu fort »).
    // DEUX SOURCES NATURELLEMENT COURTES, et ce ne sont pas des sons retronqués : le geste du
    // jeu fait cette durée-là. La zone contestée est « 1 couche, 1 son » (événement c3327c0b,
    // 1,180 s) ; le ramassage sur socle est « 1 parmi 2 » (événement c73036e4, 0,804 s).
    objective_zone_contested: 1.18,
    objective_pad_pickup: 0.804,
    objective_zone_tick_team: 1.2,
    objective_zone_tick_enemy: 1.2,
    grapple_fire: 0.745,
    grapple_fire_v2: 0.765,
    grapple_fire_v3: 0.757,
  }

  it('explosions et équipements gardent la durée de leur source, jamais retronquée à 1,2 s', () => {
    const retronques = longs
      .filter((s) => SOURCES_COURTES[s] === undefined)
      .map((s) => ({ stem: s, s: dureeDe(s) }))
      .filter((d) => d.s <= COURT_S)
    expect(retronques).toEqual([])
  })

  it('aucun son d ÉVÉNEMENT ne profite du plafond relevé par les fanfares', () => {
    const trop = [...courts, ...longs]
      .map((s) => ({ stem: s, s: dureeDe(s) }))
      .filter((d) => d.s > LONG_MAX_S + 0.001)
    expect(trop).toEqual([])
  })

  /**
   * LA CONCLUSION EST UNE TROISIÈME FAMILLE DE DURÉE, et elle n'a pas de coupe : une réplique
   * d'annonceur dure ce qu'elle dure (0,98 à 1,58 s), une fanfare ce qu'elle joue (9,93 à
   * 11,67 s). Ce qui se vérifie ici n'est donc pas une borne éditoriale mais la règle du
   * lecteur, « plafond = plus long fichier livré » : le jour où une fanfare plus longue
   * arriverait sans que `SOUND_CUT_MAX_S` bouge, elle serait tronquée EN SILENCE — et le jour
   * où le plafond monterait sans raison, il cesserait de protéger de quoi que ce soit.
   */
  it('les sons de fin de partie tiennent sous le plafond, qui les serre de près', () => {
    const durees = endMatchStems.map((s) => dureeDe(s))
    const plusLong = Math.max(...[...shipped].map((s) => dureeDe(s)))
    expect(Math.max(...durees)).toBeLessThanOrEqual(SOUND_CUT_MAX_S)
    expect(plusLong).toBeLessThanOrEqual(SOUND_CUT_MAX_S)
    expect(SOUND_CUT_MAX_S - plusLong, 'plafond décollé du plus long fichier livré').toBeLessThan(1)
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
   * geste partagé par tout l'arsenal, qu'aucune arme ne peut nommer ; NOM Repulsor = le
   * répulseur, identifié le 2026-08-25 (lot R6) par RE `himap` du `jpt!` 07104b31 (chaîne
   * eqip 7ca85adc -> sofa 6845f2b3 -> eqip frère 1e79ebda -> jpt!, propulseur écarté) — le
   * fil d'alerte posé par le lot R2.4 (2026-08-16, « le son du répulseur attend son
   * identification ») a fait exactement ce pour quoi il existait : il est devenu rouge
   * quand la règle killicon a existé, signalant qu'il fallait le retirer et brancher le son.
   */
  const attendu: Record<string, { genre: string; key: string }> = {
    'killfeed-46': { genre: 'GGGL', key: '0' },
    'killfeed-47': { genre: 'GGGL', key: '1' },
    'killfeed-48': { genre: 'GGGL', key: '2' },
    'killfeed-49': { genre: 'GGGL', key: '3' },
    'killfeed-65': { genre: 'CLASSE', key: 'MELEE' },
    'killfeed-56': { genre: 'NOM', key: 'Repulsor' },
    // LES QUATRE BOBINES (2026-08-27) : genre BANQUE, clé = le nom de banque ENTIER. C est la
    // nouveauté — la racine courte à trois segments les rendait toutes identiques
    // (`exp_single_small`), donc indistinguables et sans icône. Le resolveur essaie désormais
    // la racine longue d abord.
    'killfeed-42': { genre: 'BANQUE', key: 'exp_single_small_shock' },
    'killfeed-43': { genre: 'BANQUE', key: 'exp_single_small_hardlight' },
    'killfeed-44': { genre: 'BANQUE', key: 'exp_single_small_kineticunsc' },
    'killfeed-45': { genre: 'BANQUE', key: 'exp_single_small_plasma' },
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
      expect(KILL_SPRITE_SOUND_STEMS[sprite].stem, `rang ${rang}`).toBe(EXPLOSION_SOUND_STEMS[rang])
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
 * LE SON DU RÉPULSEUR : RÉSOLU (lot R6, 2026-08-25) — CE BLOC REMPLACE LE FIL D'ALERTE DU
 * LOT R2.4 (2026-08-16), QUI A FAIT EXACTEMENT SON TRAVAIL.
 *
 * HISTORIQUE. R2.4 avait mesuré ligne à ligne (2026-08-16) qu'aucune des 473 lignes de
 * `damagetag/data/labels.tsv` ne nommait le répulseur — la vignette (`killfeed-56`,
 * `nom_jeu: repulsor`, `static/weapons-assets/halo_infinite/jeu/index.json`) existait sans
 * source de dégât mesurée pour y mener, et écrire une règle `CLASSE INCONNU` aurait posé le
 * son sur 206 sources sans rapport. R2.4 avait donc livré un fil d'alerte plutôt qu'une
 * supposition — exactement la discipline que ce dépôt attend : une icône absente est un
 * repli, une icône fausse est un mensonge.
 *
 * RÉSOLUTION. Le lot R6 a identifié le `jpt!` par RE hors ligne (`himap`, tags du jeu
 * installé, aucune base ni re-cuisson) : `eqip 7ca85adc` (le répulseur, déjà nommé dans
 * `replay_labels.toml`) référence le `sofa 6845f2b3`, qui référence à son tour un second
 * `eqip` frère `1e79ebda` — et CET `eqip` porte directement une dépendance `jpt! 07104b31`.
 * Le propulseur a été écarté explicitement (ses deux `eqip` connus, 0x430dda48 et
 * 0xeef5d48d, sont des tags distincts). `07104b31` est désormais nommé "Repulsor",
 * classe ARME, statut SOUS_RESERVE (`damagetag/data/labels.tsv` — le même statut que TOUTE
 * arme nommée de cette table, cf. `TestCoherenceDesLignes` côté Go : le nom précis n'est
 * jamais garanti à 100 %, seule la classe l'est). `killicon/data/rules.tsv` porte la règle
 * `NOM Repulsor -> killfeed-56`, colonne weapon_key VIDE (arme hors registre, comme
 * Mutilator et Sandwich) — d'où le passage par `KILL_SPRITE_SOUND_STEMS` et non
 * `WEAPON_SOUND_STEMS` côté son. Le son livré est `EQUIPMENT/Repulser - Activate (On
 * Object)` (1,852 s, la variante d'impact SUR UNE CIBLE — seule cohérente avec un kill,
 * choix déjà écrit par R2.4), normalisé à la recette maison.
 *
 * CE QUE CE BLOC VÉRIFIE MAINTENANT : le sens inverse du fil de R2.4 — que la règle killicon
 * existe bien, UNE SEULE FOIS, et mène à la bonne vignette. Une régression (règle retirée,
 * dupliquée, ou déviée vers une autre vignette) redevient rouge ici.
 */
describe('garde-rail : le son du répulseur (résolu, lot R6 2026-08-25)', () => {
  const REPULSEUR = 'killfeed-56'

  it('exactement une règle killicon NOM Repulsor mène à killfeed-56', () => {
    const versRepulseur = killiconRules()
      .filter((r) => r.sprite === REPULSEUR)
      .map((r) => `${r.genre} ${r.key}`)
    expect(versRepulseur).toEqual(['NOM Repulsor'])
  })

  it('KILL_SPRITE_SOUND_STEMS répond pour le répulseur, catégorie ARME', () => {
    expect(KILL_SPRITE_SOUND_STEMS[REPULSEUR]).toEqual({ stem: 'repulsor_kill', category: 'weapon' })
  })
})
