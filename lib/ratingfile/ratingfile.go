package ratingfile

import "strings"

type Player struct {
	Name string
	R    float64
	RD   float64
	V    float64
}

type Result struct {
	Left  string
	Right string
}

type Entry struct {
	Timestamp int64
	Namespace string

	Result  *Result
	Player *Player

	Comment string
}

const (
	timestampRE = `([0-9]+)`
	tokenRE = `([A-Za-z0-9-]+)`
	resultRE = `([0-9.]+-[0-9.]+)`
	keyvalRE = `([A-Za-z0-9]+=)`
)

var playerDefRE = regexp.MustCompile(strings.Join([]string{
	timestampRE,
	tokenRE,
	numberRE,
	tokenRE,
}, `\s+`)
	`([0-9]+)\s+`+
	`([a-z0-9-]+)\s(

func parseLine(s string) (*Entry, error) {
	var comment string
	if i := strings.Index(s, "#"); i != -1 {
		s = s[:i]
		comment = strings.TrimLeft(s[i:], "# ")
	}

	s = strings.TrimSpace(s)

	if s == "" {
		return nil, nil
	}
}
