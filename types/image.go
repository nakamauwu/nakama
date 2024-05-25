package types

type Image struct {
	Path      string `json:"path"`
	Width     uint   `json:"width"`
	Height    uint   `json:"height"`
	Thumbhash string `json:"thumbhash"`
}
