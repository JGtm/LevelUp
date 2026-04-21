package assets

import (
	"encoding/json"
	"time"
)

// Source indique l'origine d'un asset résolu.
type Source uint8

const (
	// SourceLocalFile indique que l'asset a été servi depuis le filesystem local.
	SourceLocalFile Source = iota + 1
	// SourceLocalDB indique que l'asset a été servi depuis l'index DuckDB.
	SourceLocalDB
	// SourceRemote indique que l'asset a été fetché depuis une source distante (GameCMS, UGC).
	SourceRemote
)

// String retourne la représentation string de la source.
func (s Source) String() string {
	switch s {
	case SourceLocalFile:
		return "local_file"
	case SourceLocalDB:
		return "local_db"
	case SourceRemote:
		return "remote"
	}
	return "unknown"
}

// Payload est le contenu résolu d'un asset.
// Utiliser une assertion de type pour accéder au sous-type concret.
type Payload interface{ isPayload() }

// BinaryPayload contient des bytes binaires (image PNG, JPEG…).
type BinaryPayload struct {
	// ContentType est le MIME type (ex: "image/png").
	ContentType string
	// Bytes est le contenu brut.
	Bytes []byte
	// ETag est le hash SHA-256 hex (8 premiers octets) pour la validation de cache.
	ETag string
}

func (BinaryPayload) isPayload() {}

// URLPayload contient une URL de redirection vers le contenu.
// Le resolver renvoie ce type quand le contenu n'est pas encore en cache local
// mais que l'URL distante est connue (redirect 302).
type URLPayload struct {
	// URL est l'adresse de redirection.
	URL string
	// ContentType est le MIME type attendu.
	ContentType string
}

func (URLPayload) isPayload() {}

// JSONPayload contient des données JSON structurées (définitions, métadonnées…).
type JSONPayload struct {
	// RawJSON est le JSON brut tel que reçu de la source.
	RawJSON json.RawMessage
	// TypedValue est l'objet Go désérialisé (peut être nil si non demandé).
	TypedValue any
}

func (JSONPayload) isPayload() {}

// Resolved est le résultat complet d'une résolution d'asset.
type Resolved struct {
	// Payload est le contenu de l'asset.
	Payload Payload
	// Source indique l'origine du contenu.
	Source Source
	// FetchedAt est l'horodatage de dernière récupération.
	FetchedAt time.Time
	// ETag est le hash de contenu (pour validation de cache client).
	ETag string
}
