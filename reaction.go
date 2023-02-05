package nakama

type ReactionsCountItem struct {
	Reaction string `json:"reaction"`
	Count    uint   `json:"count"`
}

type ReactionsCount []ReactionsCountItem

func (cc *ReactionsCount) Inc(reaction string) {
	var done bool
	for i, r := range *cc {
		if r.Reaction == reaction {
			(*cc)[i].Count++
			done = true
		}
	}

	if done {
		return
	}

	*cc = append(*cc, ReactionsCountItem{Reaction: reaction, Count: 1})
}
