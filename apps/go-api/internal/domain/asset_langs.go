// Package domain — asset_langs.go : constantes BCP-47 pour traductions multilingues assets.
//
// Sprint 54 : peuplement asset_translations depuis Discovery UGC API.
package domain

// TargetLanguages définit les 14 langues BCP-47 supportées par Halo Infinite.
// Ces codes sont utilisés dans les headers Accept-Language lors des appels à l'API Discovery UGC.
var TargetLanguages = []string{
	"en-US", // English (United States)
	"fr-FR", // French (France)
	"de-DE", // German (Germany)
	"es-ES", // Spanish (Spain)
	"es-MX", // Spanish (Mexico)
	"it-IT", // Italian (Italy)
	"ja-JP", // Japanese (Japan)
	"ko-KR", // Korean (South Korea)
	"pt-BR", // Portuguese (Brazil)
	"zh-CN", // Chinese (Simplified, China)
	"zh-TW", // Chinese (Traditional, Taiwan)
	"nl-NL", // Dutch (Netherlands)
	"pl-PL", // Polish (Poland)
	"ru-RU", // Russian (Russia)
}

// DefaultLanguage est la langue par défaut utilisée pour les assets non localisés.
const DefaultLanguage = "en-US"
