/**
 * replayRecording.ts — LE CHOIX DU CONTENEUR VIDÉO, et rien d'autre.
 *
 * POURQUOI CE CHOIX N'EST PAS ANODIN. `MediaRecorder` ne sait pas produire le même fichier
 * partout : Chrome sur Windows encode volontiers du MP4/H.264, Firefox ne connaît que WebM.
 * Un fichier MP4 s'ouvre dans n'importe quel lecteur et se dépose tel quel dans un montage ;
 * un WebM demande souvent une conversion. L'ordre ci-dessous descend donc du plus portable au
 * plus sûr, et l'EXTENSION SUIT LE TYPE RETENU — un `.mp4` qui contiendrait du WebM ne
 * s'ouvrirait nulle part, et c'est exactement ce que produit un nom de fichier écrit en dur.
 *
 * L'INTERROGATION EST UN PARAMÈTRE (`isSupported`), pas un appel direct à
 * `MediaRecorder.isTypeSupported` : la fonction reste ainsi une décision pure, testable sans
 * navigateur — jsdom n'a pas `MediaRecorder` du tout.
 */

/** Le conteneur retenu et l'extension qui va avec. Les deux voyagent ensemble, toujours. */
export interface VideoMimeChoice {
  mime: string
  ext: string
}

/**
 * L'ordre EXACT de la décision 5 du plan : MP4/H.264 d'abord (le plus portable), MP4 nu
 * ensuite, puis WebM/VP9 et WebM nu — le repli que tout navigateur qui enregistre sait faire.
 */
const CANDIDATES: readonly VideoMimeChoice[] = [
  { mime: 'video/mp4;codecs=avc1', ext: 'mp4' },
  { mime: 'video/mp4', ext: 'mp4' },
  { mime: 'video/webm;codecs=vp9', ext: 'webm' },
  { mime: 'video/webm', ext: 'webm' },
]

/**
 * CADENCE DE CAPTURE, en images par seconde.
 *
 * 30 et pas 60 : le rejeu se lit à la cadence de l'écran, mais ce qu'il montre — des points
 * qui glissent sur une carte — ne gagne rien à doubler le débit, et un clip de match pèserait
 * alors le double pour l'œil qui n'y verrait rien. Ce n'est PAS la vitesse de lecture : filmer
 * un rejeu passé en 2× donne un clip à 30 im/s où l'action va deux fois plus vite (décision 4).
 */
export const CAPTURE_FPS = 30

/** pickVideoMimeType rend le premier conteneur supporté, ou `null` si aucun ne l'est. */
export function pickVideoMimeType(isSupported: (type: string) => boolean): VideoMimeChoice | null {
  return CANDIDATES.find((c) => isSupported(c.mime)) ?? null
}

/**
 * isVideoTypeSupported interroge le navigateur courant.
 *
 * Rend `false` partout où l'enregistrement n'existe pas (jsdom, vieux navigateurs) plutôt que
 * de lever : c'est ce qui permet à `pickVideoMimeType` de conclure « aucun » proprement, et au
 * bouton de ne pas se rendre du tout (décision 7).
 */
export function isVideoTypeSupported(type: string): boolean {
  if (typeof MediaRecorder === 'undefined') return false
  if (typeof MediaRecorder.isTypeSupported !== 'function') return false
  return MediaRecorder.isTypeSupported(type)
}

/**
 * canRecordCanvas dit si CE navigateur sait filmer une toile.
 *
 * Les trois conditions sont indissociables : l'enregistreur, la capture de flux du canvas, et
 * un conteneur qu'il accepte. Il en manque une, il n'y a pas d'enregistrement possible — et
 * mieux vaut alors pas de bouton qu'un bouton qui ne rendrait jamais de fichier.
 */
export function canRecordCanvas(): boolean {
  if (typeof MediaRecorder === 'undefined') return false
  if (typeof HTMLCanvasElement === 'undefined') return false
  if (typeof HTMLCanvasElement.prototype.captureStream !== 'function') return false
  return pickVideoMimeType(isVideoTypeSupported) !== null
}
