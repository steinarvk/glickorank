package ratingfile

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/google/shlex"

	"github.com/steinarvk/glickorank/lib/glicko2"
	"github.com/steinarvk/glickorank/lib/lines"
)

type Universe struct {
	Name             string
	Ratings          map[string]glicko2.Rating
	RatingTimestamps map[string]int64

	MatchTimestamps []int64
	Matches         []glicko2.Match

	MaxTimestamp int64
}

func parseRating(keyvals ...string) (*glicko2.Rating, error) {
	m := map[string]float64{}
	for _, keyval := range keyvals {
		if strings.Count(keyval, "=") != 1 {
			return nil, fmt.Errorf("invalid rating line clause: %q", keyval)
		}
		groups := strings.Split(keyval, "=")
		k := groups[0]
		v := groups[1]
		val, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, err
		}

		m[k] = val
	}

	r, ok := m["r"]
	if !ok {
		return nil, fmt.Errorf("invalid rating: missing 'r'")
	}

	rd, ok := m["rd"]
	if !ok {
		return nil, fmt.Errorf("invalid rating: missing 'rd'")
	}

	v, _ := m["v"]
	if v == 0 {
		v = glicko2.DefaultVolatility
	}

	return glicko2.NewRating(r, rd, v)
}

func parseGame(result, left, right string) (*glicko2.Match, error) {
	if strings.Count(result, "-") != 1 {
		return nil, fmt.Errorf("invalid game record: %q is not a proper game result", result)
	}

	resgroup := strings.Split(result, "-")
	resleft := resgroup[0]
	resright := resgroup[1]
	switch {
	case resleft == resright:
		return &glicko2.Match{
			Left:   left,
			Right:  right,
			Winner: "",
		}, nil
	case resleft == "0":
		return &glicko2.Match{
			Left:   left,
			Right:  right,
			Winner: right,
		}, nil
	case resright == "0":
		return &glicko2.Match{
			Left:   left,
			Right:  right,
			Winner: left,
		}, nil
	default:
		return nil, fmt.Errorf("invalid game result specification: %q", resgroup)
	}
}

func Read(r io.Reader) (map[string]*Universe, error) {
	multiverse := map[string]*Universe{}

	getUniverse := func(name string) *Universe {
		rv, ok := multiverse[name]
		if !ok {
			rv = &Universe{
				Name:             name,
				Ratings:          map[string]glicko2.Rating{},
				RatingTimestamps: map[string]int64{},
			}
			multiverse[name] = rv
		}
		return rv
	}

	if err := lines.OnLines(r, func(line string) (bool, error) {
		words, err := shlex.Split(line)
		if err != nil {
			return false, err
		}
		if len(words) == 0 {
			return true, nil
		}

		if len(words) < 5 {
			return false, fmt.Errorf("line too short (%d words): %q", len(words), line)
		}

		t, err := strconv.ParseInt(words[0], 10, 64)
		if err != nil {
			return false, fmt.Errorf("invalid timestamp %q: %v", words[0], err)
		}

		ns := words[1]

		univ := getUniverse(ns)

		if univ.MaxTimestamp < t {
			univ.MaxTimestamp = t
		}

		if strings.Contains(words[2], "-") {
			if len(words) > 5 {
				return false, fmt.Errorf("match line too long (%d words): %q", len(words), line)
			}
			match, err := parseGame(words[2], words[3], words[4])
			if err != nil {
				return false, fmt.Errorf("bad match line %q: %v", line, err)
			}

			univ.Matches = append(univ.Matches, *match)
			univ.MatchTimestamps = append(univ.MatchTimestamps, t)
		} else {
			playerName := words[3]

			rating, err := parseRating(append([]string{"r=" + words[2]}, words[4:]...)...)
			if err != nil {
				return false, fmt.Errorf("invalid rating line %q: %v", err, line)
			}

			if _, p := univ.Ratings[playerName]; p {
				return false, fmt.Errorf("duplicate rating provided for player %q", playerName)
			}

			univ.Ratings[playerName] = *rating
			univ.RatingTimestamps[playerName] = t
		}

		return true, nil
	}); err != nil {
		return nil, err
	}

	return multiverse, nil
}

func WriteRatings(w io.Writer, prefix string, m map[string]glicko2.Rating) error {
	var rv []string

	for k := range m {
		rv = append(rv, k)
	}

	sort.Slice(rv, func(i, j int) bool {
		return m[rv[i]].Rating > m[rv[j]].Rating
	})

	for _, k := range rv {
		v := m[k]
		_, err := fmt.Fprintf(w, "%s%.4f %s rd=%.4f v=%.4f\n", prefix, v.Rating, k, v.RatingDeviation, v.Volatility)
		if err != nil {
			return err
		}
	}

	return nil
}
