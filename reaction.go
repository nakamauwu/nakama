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

func (cc *ReactionsCount) Dec(reaction string) {
	var wentZero bool
	for i, r := range *cc {
		if r.Reaction == reaction {
			(*cc)[i].Count--
			if (*cc)[i].Count == 0 {
				wentZero = true
			}
		}
	}

	if !wentZero {
		return
	}
	for {
		var removed bool
		for i, r := range *cc {
			if r.Reaction == reaction {
				(*cc)[i].Count--
				if (*cc)[i].Count == 0 {
					*cc = append((*cc)[:i], (*cc)[i+1:]...)
					removed = true
					break
				}
			}
		}
		if !removed {
			break
		}
	}
}
