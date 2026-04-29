package agenttab

var codenameWords = []string{
	"atlas",
	"beacon",
	"comet",
	"delta",
	"ember",
	"falcon",
	"glacier",
	"harbor",
	"ion",
	"juniper",
}

func codenameForIndex(i int) string {
	if i >= 0 && i < len(codenameWords) {
		return codenameWords[i]
	}
	return "candidate"
}
