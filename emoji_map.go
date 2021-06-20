package nakama

import (
	"github.com/enescakir/emoji"
)

var emojiMap = func() map[string]struct{} {
	out := map[string]struct{}{}
	for _, emoji := range emoji.Map() {
		out[emoji] = struct{}{}
	}
	return out
}()
