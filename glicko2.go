package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/steinarvk/glickorank/lib/glicko2"
	"github.com/steinarvk/glickorank/lib/ratingfile"
)

var (
	tau         = flag.Float64("tau", 0.5, "glicko2 system parameter tau")
	batch       = flag.Int("batch", 0, "split matches into batches")
	repetitions = flag.Int("repetitions", 0, "repeat entire match dataset")
)

func main() {
	flag.Parse()

	universes, err := ratingfile.Read(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v", err)
		os.Exit(1)
	}

	sys := glicko2.System{
		Tau: *tau,
	}

	var keys []string
	for k := range universes {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := universes[k]

		ratings := v.Ratings

		nTimes := 1 + *repetitions

		for i := 0; i < nTimes; i++ {
			matches := v.Matches

			for len(matches) > 0 {
				var restOfMatches []glicko2.Match

				if *batch > 0 && len(matches) > *batch {
					restOfMatches = matches[*batch:]
					matches = matches[:*batch]
				}

				updatedRatings, err := sys.Update(ratings, matches)
				if err != nil {
					fmt.Fprintf(os.Stderr, "fatal: %v", err)
					os.Exit(1)
				}

				ratings = updatedRatings
				matches = restOfMatches
			}
		}

		prefix := fmt.Sprintf("%d %s ", v.MaxTimestamp, v.Name)
		ratingfile.WriteRatings(os.Stdout, prefix, ratings)
	}
	return
}
