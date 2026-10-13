package lexers

import (
	"regexp"
)

var zoneAnalyserRe = regexp.MustCompile(`(?m)^@\s+IN\s+SOA\s+`)

func analyseDNS() {
	GlobalLexerRegistry.Get("dns").SetAnalyser(func(text string) float32 {
		if zoneAnalyserRe.FindString(text) != "" {
			return 1.0
		}
		return 0.0
	})
}
