package media

import (
	"net/http"
	"strings"
)

// SniffLen est le nombre d'octets de tête que http.DetectContentType examine.
const SniffLen = 512

// ContentMatchesExtension dit si les premiers octets d'un fichier envoyé (head, au plus
// SniffLen octets) concordent avec la FAMILLE de son extension, selon
// http.DetectContentType :
//
//   - image (ImageExtensions) : le contenu détecté est une image (image/*). Go reconnaît
//     PNG, JPEG, GIF et BMP ; une image d'un autre de ces formats sous une extension
//     d'image reste acceptée (extension mal posée par l'outil de capture).
//   - vidéo (VideoExtensions) : le contenu détecté est une vidéo (video/* : MP4 de marque
//     mp4x, WebM et Matroska, AVI) ou application/octet-stream. Go ne reconnaît ni le
//     QuickTime (.mov, marque « qt  ») ni un MP4 sans marque compatible mp4x (isom seule) :
//     ils sortent en octet-stream, et ffprobe les valide en aval (transcodage HLS).
//
// Tout autre résultat (texte, HTML, XML, PDF, archive, image sous une extension vidéo...)
// est refusé : c'est ce qui empêche de déposer un document actif sous un nom de média.
// Une extension hors des deux familles n'est jamais acceptée.
func ContentMatchesExtension(ext string, head []byte) bool {
	detected := http.DetectContentType(head)
	switch {
	case IsImageExt(ext):
		return strings.HasPrefix(detected, "image/")
	case IsVideoExt(ext):
		return strings.HasPrefix(detected, "video/") || detected == "application/octet-stream"
	default:
		return false
	}
}
