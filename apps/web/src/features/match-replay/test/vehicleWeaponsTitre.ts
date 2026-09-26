/// <reference types="node" />
/**
 * vehicleWeaponsTitre.ts — LE REGISTRE DES ARMES DE VÉHICULE DU TITRE, lu par les TESTS du rejeu
 * (retours du rejeu 2026-09-23, revue adverse du lot M4a, F7 et F8).
 *
 * # POURQUOI UN LECTEUR CÔTÉ TESTS
 *
 * Le registre (`config/titles/{slug}/mappings/vehicle_weapons.toml`) est RÉSOLU À LA REQUÊTE par le
 * serveur (`service/replay_vehicle_weapons.go`) : un artefact cuit lu sur disque n'en porte pas.
 * Deux usages de test en ont besoin sans passer par l'API :
 *  - le garde-rail « aucun tag d'arme de véhicule côté client » interdit CHAQUE tag du registre,
 *    pas seulement un gabarit (`vehicleWeaponRegistry.guard.test.ts`) ;
 *  - les instruments de mesure qui lisent des documents cuits y reposent la table telle que l'API
 *    la sert (`tourelleVisee.mesure.test.ts`).
 *
 * # CE QU'IL LIT, ET CE QU'IL NE LIT PAS
 *
 * Le SOUS-ENSEMBLE du format que ce fichier emploie : des blocs `[[weapons]]` / `[[unknown]]`, des
 * paires `clé = "chaîne"` et la table en ligne `mount = { aim = "…", ax = n, ay = n }`. La
 * VALIDATION du format (listes fermées, preuve, silence décidé) reste au chargeur Go
 * (`mappings/loader_vehicle_weapons.go`) et à ses tests : ce lecteur ne décide rien, il rend ce qui
 * est écrit. Un fichier qu'il ne sait pas lire le fait échouer (jamais une table vide en silence).
 */
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

import type { ReplayDocument } from '@/lib/api/types'

import { racineDuDepot } from './featureFiles'

type RegistreDuDocument = NonNullable<ReplayDocument['vehicleWeapons']>
type EntreeDuDocument = RegistreDuDocument[string]

/** Le registre d'un titre, tel que le dépôt le versionne. */
export interface RegistreTitre {
  /** Tous les tags du fichier (armes ET inconnus motivés), 8 chiffres hex en majuscules. */
  tags: string[]
  /** La table `vehicleWeapons` que l'API poserait pour un document qui tire TOUTES ces armes. */
  armes: RegistreDuDocument
}

const PAIRE = /^([a-z_]+)\s*=\s*(.+)$/
const CHAINE = /^"(.*)"$/
const MONTAGE = /^\{\s*aim\s*=\s*"(fixed|turret)"\s*,\s*ax\s*=\s*(-?[\d.]+)\s*,\s*ay\s*=\s*(-?[\d.]+)\s*\}$/

/** registreDuTitre lit le registre versionné d'un titre (`halo_infinite` par défaut). */
export function registreDuTitre(slug = 'halo_infinite'): RegistreTitre {
  const chemin = join(racineDuDepot(), 'config', 'titles', slug, 'mappings', 'vehicle_weapons.toml')
  const blocs = readFileSync(chemin, 'utf8').split(/^\[\[(weapons|unknown)\]\]\s*$/m)
  const out: RegistreTitre = { tags: [], armes: {} }
  // `split` avec un groupe capturant alterne [avant, genre, corps, genre, corps, …].
  for (let i = 1; i < blocs.length; i += 2) {
    const paires = pairesDe(blocs[i + 1])
    const tag = paires.get('tag')
    if (!tag || !/^[0-9A-F]{8}$/.test(tag)) throw new Error(`registre ${slug} : tag illisible (${tag})`)
    out.tags.push(tag)
    if (blocs[i] === 'weapons') out.armes[`0x${tag}00000000`] = entreeDe(paires)
  }
  if (out.tags.length === 0) throw new Error(`registre ${slug} : aucun tag lu dans ${chemin}`)
  return out
}

/** pairesDe rend les paires `clé = valeur` d'un bloc (chaînes dé-guillemetées). */
function pairesDe(corps: string): Map<string, string> {
  const out = new Map<string, string>()
  for (const ligne of corps.split(/\r?\n/)) {
    const m = PAIRE.exec(ligne.trim())
    if (!m) continue
    const chaine = CHAINE.exec(m[2].trim())
    out.set(m[1], chaine ? chaine[1] : m[2].trim())
  }
  return out
}

/** entreeDe projette un bloc `[[weapons]]` vers la forme publiée dans le document. */
function entreeDe(p: Map<string, string>): EntreeDuDocument {
  const e: EntreeDuDocument = {
    vehicle: p.get('vehicle') ?? '',
    en: p.get('en') ?? '',
    fr: p.get('fr') ?? '',
    fire: p.get('fire') ?? '',
    fx: p.get('fx') ?? '',
    tint: p.get('tint') ?? '',
  }
  const son = p.get('sound')
  if (son) e.sound = son
  const montage = p.get('mount')
  if (montage) {
    const m = MONTAGE.exec(montage)
    if (!m) throw new Error(`registre : montage illisible (${montage})`)
    e.mount = { aim: m[1], ax: Number(m[2]), ay: Number(m[3]) }
  }
  return e
}
