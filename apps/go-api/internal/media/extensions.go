// Package media regroupe les helpers transverses au domaine média :
// classification des extensions vidéo (web-native vs remux requis) et
// streaming remux à la volée via ffmpeg pour les containers non lisibles
// nativement par les navigateurs (MKV, AVI).
//
// La logique d'indexation des médias reste dans internal/ops/media.go.
// Ce package est volontairement étroit : seul ServeMediaFile en dépend.
package media

import (
	"path/filepath"
	"strings"
)

// VideoExtensions liste toutes les extensions vidéo reconnues côté serving.
// Doit rester alignée avec internal/ops/media.go::supportedExtensions (subset
// kind=video). Si une nouvelle extension est ajoutée à l'indexation, l'ajouter
// ici aussi pour que le fallback resolution la trouve.
var VideoExtensions = []string{".mp4", ".webm", ".mov", ".avi", ".mkv"}

// extPNG : extension PNG, nommée pour la liste des images et ses tests.
const extPNG = ".png"

// ImageExtensions liste les formats d'image acceptés à l'envoi et indexés.
var ImageExtensions = []string{extPNG, ".jpg", ".jpeg", ".bmp", ".gif"}

// derivedExtensions liste les fichiers que le serveur produit à partir d'un média :
// miniatures (.webp ; .gif est déjà une image) et arbre HLS (playlists .m3u8, segments
// fMP4 .m4s, segments MPEG-TS .ts ; les init_*.mp4 sont des vidéos).
var derivedExtensions = []string{".webp", ".m3u8", ".m4s", ".ts"}

// IsImageExt retourne true si l'extension est un format d'image de ImageExtensions.
// La comparaison est case-insensitive.
func IsImageExt(ext string) bool {
	return containsExt(ImageExtensions, ext)
}

// IsServableExt retourne true si la route des médias peut servir un fichier de cette
// extension : vidéo, image, miniature ou fichier HLS. Toute autre extension (base
// DuckDB, WAL, JSON de réglage...) n'est jamais servie, quelle que soit la racine.
// La comparaison est case-insensitive.
func IsServableExt(ext string) bool {
	return IsVideoExt(ext) || IsImageExt(ext) || containsExt(derivedExtensions, ext)
}

func containsExt(list []string, ext string) bool {
	ext = strings.ToLower(ext)
	for _, v := range list {
		if v == ext {
			return true
		}
	}
	return false
}

// webNativeVideoExts : containers vidéo lisibles directement par les
// navigateurs modernes (Chrome/Firefox/Edge/Safari). Servis via http.ServeFile
// (supporte Range requests, donc le seek vidéo fonctionne).
var webNativeVideoExts = map[string]bool{
	".mp4":  true,
	".webm": true,
	".mov":  true, // QuickTime H.264 lisible Chrome/Safari ; Firefox partiel
}

// remuxRequiredVideoExts : containers non lisibles nativement. Doivent être
// remuxés à la volée vers WebM (zéro réencodage si codecs internes compatibles
// AV1/VP8/VP9/Opus).
var remuxRequiredVideoExts = map[string]bool{
	".mkv": true,
	".avi": true,
}

// IsWebNativeVideo retourne true si le navigateur peut lire le fichier
// directement, sans remux. La comparaison est case-insensitive.
func IsWebNativeVideo(ext string) bool {
	return webNativeVideoExts[strings.ToLower(ext)]
}

// RequiresRemux retourne true si l'extension doit passer par un remux ffmpeg
// avant d'être servie au navigateur. La comparaison est case-insensitive.
func RequiresRemux(ext string) bool {
	return remuxRequiredVideoExts[strings.ToLower(ext)]
}

// IsVideoExt retourne true si l'extension est un container vidéo reconnu
// (subset vidéo de VideoExtensions). Sert à pré-filtrer les candidats au
// transcoding HLS sans probe ffprobe (les images retournent false).
func IsVideoExt(ext string) bool {
	return containsExt(VideoExtensions, ext)
}

// StemAndExt sépare un chemin en (stem, ext). `ext` inclut le point et est
// en lowercase. `stem` est le basename sans extension.
//
// Exemple : "foo/Bar.MKV" → ("Bar", ".mkv")
func StemAndExt(p string) (stem, ext string) {
	base := filepath.Base(p)
	ext = strings.ToLower(filepath.Ext(base))
	stem = strings.TrimSuffix(base, filepath.Ext(base))
	return stem, ext
}
