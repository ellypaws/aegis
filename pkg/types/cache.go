package types

type CachedRole struct {
	ID      string `gorm:"primaryKey" json:"id"`
	Name    string `json:"name"`
	Color   int    `json:"color"`
	Managed bool   `json:"managed"`
}

type CachedChannel struct {
	ID   string `gorm:"primaryKey" json:"id"`
	Name string `json:"name"`
	Type int    `json:"type"`
}
