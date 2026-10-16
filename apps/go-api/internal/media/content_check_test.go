package media

import (
	"strings"
	"testing"
)

// En-têtes réels minimaux des formats envoyés.
var (
	headPNG  = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
	headJPEG = []byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00")
	headGIF  = []byte("GIF89a\x01\x00\x01\x00")
	headBMP  = []byte("BM\x36\x00\x00\x00\x00\x00\x00\x00\x36\x00\x00\x00")
	// MP4 (marque mp42), QuickTime (marque qt), MP4 sans marque mp4x (isom seule).
	headMP4  = []byte("\x00\x00\x00\x18ftypmp42\x00\x00\x00\x00mp42isom")
	headMOV  = []byte("\x00\x00\x00\x14ftypqt  \x00\x00\x02\x00qt  ")
	headISOM = []byte("\x00\x00\x00\x14ftypisom\x00\x00\x02\x00isom")
	headWebM = []byte("\x1a\x45\xdf\xa3\x9f\x42\x86\x81\x01\x42\xf7\x81\x01webm")
	headAVI  = []byte("RIFF\x00\x00\x00\x00AVI LIST")
)

func TestContentMatchesExtension_LegitimateUploads(t *testing.T) {
	cases := map[string][]byte{
		extPNG: headPNG, ".PNG": headPNG, ".jpg": headJPEG, ".jpeg": headJPEG, ".gif": headGIF,
		".bmp": headBMP, ".mp4": headMP4, ".mov": headMOV, ".webm": headWebM, ".mkv": headWebM,
		".avi": headAVI,
	}
	for ext, head := range cases {
		if !ContentMatchesExtension(ext, head) {
			t.Errorf("%s : contenu légitime refusé", ext)
		}
	}
	// Variantes qu'un outil de capture produit : MP4 isom seule (octet-stream pour Go),
	// capture PNG enregistrée en .jpg.
	if !ContentMatchesExtension(".mp4", headISOM) {
		t.Error(".mp4 de marque isom refusé")
	}
	if !ContentMatchesExtension(".jpg", headPNG) {
		t.Error("PNG sous .jpg refusé (même famille image)")
	}
}

func TestContentMatchesExtension_RefusesMismatches(t *testing.T) {
	html := []byte("<!DOCTYPE html><html><script>alert(1)</script></html>")
	svg := []byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"></svg>`)
	cases := []struct {
		ext  string
		head []byte
	}{
		{extPNG, html}, {".jpg", svg}, {".gif", []byte("hello")}, {".bmp", []byte("%PDF-1.7")},
		{".mp4", html}, {".mov", []byte("just text")}, {".webm", []byte("PK\x03\x04zip")},
		{".mkv", []byte("%PDF-1.4")}, {".avi", headPNG}, {".mp4", headJPEG},
		{extPNG, headMP4}, {".exe", headPNG}, {".html", html}, {".mp4", nil},
		{extPNG, []byte(strings.Repeat(" ", 10) + "<html>")},
	}
	for _, c := range cases {
		if ContentMatchesExtension(c.ext, c.head) {
			t.Errorf("%s / %q : contenu non concordant accepté", c.ext, c.head)
		}
	}
}
