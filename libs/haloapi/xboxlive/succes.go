package xboxlive

// succes.go — DTO bruts de la réponse des succès Xbox Live (achievements v2) : des structs miroir
// du JSON reçu, sans interprétation.

// SuccesReponse : une page de la réponse des succès.
type SuccesReponse struct {
	Achievements []Succes `json:"achievements"`
	PagingInfo   struct {
		TotalRecords int `json:"totalRecords"`
		// ContinuationToken : non vide tant qu'il reste des pages (voir la pagination du `doc.go`).
		ContinuationToken string `json:"continuationToken"`
	} `json:"pagingInfo"`
}

// Succes : un succès tel que l'API le décrit.
type Succes struct {
	ID                string `json:"id"`
	ServiceConfigID   string `json:"serviceConfigId"`
	Name              string `json:"name"`
	TitleAssociations []struct {
		Name string `json:"name"`
		ID   int    `json:"id"`
	} `json:"titleAssociations"`
	ProgressState string `json:"progressState"` // "Achieved", "InProgress", "NotStarted"
	Progression   struct {
		Requirements []struct {
			ID                    string `json:"id"`
			Current               string `json:"current"`
			Target                string `json:"target"`
			OperationType         string `json:"operationType"`
			ValueType             string `json:"valueType"`
			RuleParticipationType string `json:"ruleParticipationType"`
		} `json:"requirements"`
		TimeUnlocked string `json:"timeUnlocked"`
	} `json:"progression"`
	MediaAssets []struct {
		Name string `json:"name"`
		Type string `json:"type"`
		URL  string `json:"url"`
	} `json:"mediaAssets"`
	Platforms         []string `json:"platforms"`
	IsSecret          bool     `json:"isSecret"`
	Description       string   `json:"description"`
	LockedDescription string   `json:"lockedDescription"`
	ProductID         string   `json:"productId"`
	ActivityID        string   `json:"activityId"`
	Rewards           []struct {
		Name        interface{} `json:"name"`
		Description interface{} `json:"description"`
		Value       string      `json:"value"`
		Type        string      `json:"type"`
		MediaAsset  interface{} `json:"mediaAsset"`
	} `json:"rewards"`
	EstimatedTime string `json:"estimatedTime"`
	Deeplink      string `json:"deeplink"`
	IsRevoked     bool   `json:"isRevoked"`
	Rarity        struct {
		CurrentCategory   string  `json:"currentCategory"`
		CurrentPercentage float64 `json:"currentPercentage"`
	} `json:"rarity"`
}
