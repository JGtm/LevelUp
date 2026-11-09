/**
 * oggOpusFile.ts — LA LECTURE D'UN SON LIVRÉ (Ogg Opus), pour les garde-rails d'assets sonores.
 *
 * Les sons de `static/sounds/` sont encodés en Opus 96 kbit/s en conteneur Ogg par
 * `cmd/sounds-encode`. Ces garde-rails les lisent hors navigateur :
 *
 *  - `opusInfo` lit les en-têtes (page `OpusHead`, position granulaire de la dernière page) :
 *    canaux, cadence d'origine et nombre d'échantillons EXACT, sans décoder. Le pré-décalage du
 *    codec (pre-skip) est retiré, comme le fait le décodeur d'un navigateur ;
 *  - `decodeOpus` décode en PCM (décodeur WASM `opus-decoder`, dépendance de développement) pour
 *    les garde-rails qui mesurent un niveau.
 *
 * Aucune tolérance : un fichier qui n'est pas un Ogg Opus bien formé est une erreur, jamais une
 * valeur devinée.
 */
/// <reference types="node" />
import { readFileSync } from 'node:fs'
import { OpusDecoder } from 'opus-decoder'

/** Cadence de décodage d'Opus : fixe, quelle que soit la cadence du fichier d'origine. */
export const OPUS_DECODE_RATE = 48_000

/** Ce que disent les en-têtes d'un Ogg Opus. */
export interface OpusInfo {
  canaux: number
  /** Cadence du fichier d'origine, telle que l'encodeur l'a notée dans `OpusHead`. */
  cadence: number
  /** Échantillons par canal après décodage (pré-décalage retiré). */
  echantillons: number
  /** `echantillons` / 48 000. */
  dureeS: number
}

interface OggPage {
  granule: bigint
  /** Les segments de la page ; un paquet peut continuer sur la page suivante. */
  segments: Buffer[]
}

/** Découpe un fichier Ogg en pages et segments. */
function pagesOgg(file: string, buf: Buffer): OggPage[] {
  const pages: OggPage[] = []
  for (let at = 0; at < buf.length; ) {
    if (at + 27 > buf.length || buf.toString('latin1', at, at + 4) !== 'OggS') {
      throw new Error(`${file} : page Ogg illisible à l'octet ${at}`)
    }
    const nSeg = buf[at + 26]
    const table = buf.subarray(at + 27, at + 27 + nSeg)
    let corps = at + 27 + nSeg
    const segments: Buffer[] = []
    for (const lg of table) {
      segments.push(buf.subarray(corps, corps + lg))
      corps += lg
    }
    pages.push({ granule: buf.readBigInt64LE(at + 6), segments })
    at = corps
  }
  return pages
}

/** Reconstitue les paquets Opus (un paquet = segments de 255 octets + un segment plus court). */
function paquets(pages: OggPage[]): Buffer[] {
  const out: Buffer[] = []
  let courant: Buffer[] = []
  for (const page of pages) {
    for (const seg of page.segments) {
      courant.push(seg)
      if (seg.length < 255) {
        out.push(Buffer.concat(courant))
        courant = []
      }
    }
  }
  return out
}

interface Entete {
  canaux: number
  preSkip: number
  cadence: number
}

function enteteOpus(file: string, premier: Buffer | undefined): Entete {
  if (!premier || premier.toString('latin1', 0, 8) !== 'OpusHead' || premier.length < 19) {
    throw new Error(`${file} : pas de page OpusHead (ce n'est pas un Ogg Opus)`)
  }
  if (premier[18] !== 0) throw new Error(`${file} : famille de canaux ${premier[18]} non prise en charge`)
  return { canaux: premier[9], preSkip: premier.readUInt16LE(10), cadence: premier.readUInt32LE(12) }
}

/** Canaux, cadence d'origine et durée exacte d'un Ogg Opus déjà lu en mémoire (`nom` : pour les erreurs). */
export function opusInfoDeOctets(nom: string, octets: Buffer): OpusInfo {
  const pages = pagesOgg(nom, octets)
  const { canaux, preSkip, cadence } = enteteOpus(nom, paquets(pages)[0])
  const derniere = pages[pages.length - 1].granule
  if (derniere < BigInt(preSkip)) throw new Error(`${nom} : position granulaire finale incohérente`)
  const echantillons = Number(derniere) - preSkip
  return { canaux, cadence, echantillons, dureeS: echantillons / OPUS_DECODE_RATE }
}

const cacheInfos = new Map<string, OpusInfo>()

/** Canaux, cadence d'origine et durée exacte d'un fichier Ogg Opus, lus dans ses en-têtes. */
export function opusInfo(file: string): OpusInfo {
  const memo = cacheInfos.get(file)
  if (memo) return memo
  const info = opusInfoDeOctets(file, readFileSync(file))
  cacheInfos.set(file, info)
  return info
}

/** Un son décodé : un tableau de flottants [-1 ; 1] par canal. */
export interface OpusPcm {
  canaux: number
  cadence: number
  data: Float32Array[]
}

/** Décode un Ogg Opus en PCM, pré-décalage retiré et fin rognée à la position granulaire. */
export async function decodeOpus(file: string): Promise<OpusPcm> {
  const pages = pagesOgg(file, readFileSync(file))
  const tous = paquets(pages)
  const { canaux, preSkip } = enteteOpus(file, tous[0])
  const total = Number(pages[pages.length - 1].granule) - preSkip
  const decodeur = new OpusDecoder({ sampleRate: OPUS_DECODE_RATE, channels: canaux })
  await decodeur.ready
  try {
    const blocs: Float32Array[][] = []
    for (const p of tous.slice(2)) {
      const r = decodeur.decodeFrame(new Uint8Array(p))
      if (r.errors.length > 0) throw new Error(`${file} : décodage Opus en erreur : ${r.errors[0].message}`)
      blocs.push(r.channelData)
    }
    const data = Array.from({ length: canaux }, (_, c) => {
      const plat = new Float32Array(blocs.reduce((n, b) => n + b[c].length, 0))
      let at = 0
      for (const b of blocs) { plat.set(b[c], at); at += b[c].length }
      return plat.subarray(preSkip, preSkip + total)
    })
    return { canaux, cadence: OPUS_DECODE_RATE, data }
  } finally {
    decodeur.free()
  }
}
