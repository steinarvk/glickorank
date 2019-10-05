package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/steinarvk/glickorank/lib/glicko2"
	"github.com/steinarvk/glickorank/lib/ratingfile"
)

var (
	tau = flag.Float64("tau", 0.5, "glicko2 system parameter tau")
)

func main() {
	universes, err := ratingfile.Read(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v", err)
		os.Exit(1)
	}

	sys := glicko2.System{
		Tau: *tau,
	}

	for _, v := range universes {
		updatedRatings, err := sys.Update(v.Ratings, v.Matches)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fatal: %v", err)
			os.Exit(1)
		}

		prefix := fmt.Sprintf("%d %s ", v.MaxTimestamp, v.Name)

		ratingfile.WriteRatings(os.Stdout, prefix, updatedRatings)
	}
	return
}
