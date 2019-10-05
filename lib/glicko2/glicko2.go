package glicko2

import (
	"fmt"
	"math"
)

var (
	DefaultRating     float64 = 1500
	DefaultDeviation  float64 = 350
	DefaultVolatility float64 = 0.06
)

type Rating struct {
	Rating          float64
	RatingDeviation float64
	Volatility      float64
}

type Match struct {
	Left   string
	Right  string
	Winner string
}

type System struct {
	DefaultRating Rating
	Tau           float64
}

func (g Match) leftResult() GameResult {
	if g.Winner == "" {
		return Draw
	}
	if g.Left == g.Winner {
		return Win
	}
	return Loss
}

func (g Match) toInternalMatchAsLeft(opponentRating Rating) internalMatch {
	return internalMatch{
		Opponent: opponentRating.toInternal(),
		Result:   g.leftResult(),
	}
}

func (g Match) toInternalMatchAsRight(opponentRating Rating) internalMatch {
	return reverseMatch(g).toInternalMatchAsLeft(opponentRating)
}

func checkMatch(m Match) error {
	if m.Left == "" {
		return fmt.Errorf("missing left player")
	}
	if m.Right == "" {
		return fmt.Errorf("missing right player")
	}
	if m.Winner != "" && m.Winner != m.Left && m.Winner != m.Right {
		return fmt.Errorf("winner is non-player")
	}
	return nil
}

func reverseMatch(m Match) Match {
	return Match{
		Left:   m.Right,
		Right:  m.Left,
		Winner: m.Winner,
	}
}

func NewRating(r, rd, v float64) (*Rating, error) {
	rv := Rating{
		Rating:          r,
		RatingDeviation: rd,
		Volatility:      v,
	}
	if err := checkRating(rv); err != nil {
		return nil, err
	}
	return &rv, nil
}

func checkRating(r Rating) error {
	if r.RatingDeviation < 0 {
		return fmt.Errorf("rating deviation cannot be negative")
	}
	if r.Volatility < 0 {
		return fmt.Errorf("volatility cannot be negative")
	}
	return nil
}

func (s System) Update(oldRatings map[string]Rating, matches []Match) (map[string]Rating, error) {
	if s.Tau <= 0 {
		return nil, fmt.Errorf("invalid value for tau: %v", s.Tau)
	}

	if s.DefaultRating.Rating == 0 {
		s.DefaultRating.Rating = DefaultRating
	}
	if s.DefaultRating.RatingDeviation == 0 {
		s.DefaultRating.RatingDeviation = DefaultDeviation
	}
	if s.DefaultRating.Volatility == 0 {
		s.DefaultRating.Volatility = DefaultVolatility
	}

	getRating := func(k string) Rating {
		if oldRatings != nil {
			rv, ok := oldRatings[k]
			if ok {
				return rv
			}
		}
		return s.DefaultRating
	}

	players := map[string]struct{}{}

	if oldRatings != nil {
		for k, v := range oldRatings {
			if err := checkRating(v); err != nil {
				return nil, fmt.Errorf("invalid rating for %q: %v", k, err)
			}
			players[k] = struct{}{}
		}
	}

	for _, m := range matches {
		players[m.Left] = struct{}{}
		players[m.Right] = struct{}{}
	}

	playerMatches := map[string][]internalMatch{}

	for _, m := range matches {
		if err := checkMatch(m); err != nil {
			return nil, fmt.Errorf("bad match %v: %v", m, err)
		}
		asLeft := m.toInternalMatchAsLeft(getRating(m.Right))
		asRight := m.toInternalMatchAsRight(getRating(m.Left))
		playerMatches[m.Left] = append(playerMatches[m.Left], asLeft)
		playerMatches[m.Right] = append(playerMatches[m.Right], asRight)
	}

	rv := map[string]Rating{}

	for k := range players {
		prerat := getRating(k).toInternal()
		postrat := prerat.update(s.Tau, playerMatches[k])
		rv[k] = postrat.toRating()
	}

	return rv, nil
}

type internalRating struct {
	Mu    float64
	Phi   float64
	Sigma float64
}

func (g internalRating) toRating() Rating {
	return Rating{
		Rating:          173.7178*g.Mu + DefaultRating,
		RatingDeviation: 173.7178 * g.Phi,
		Volatility:      g.Sigma,
	}
}

func (r Rating) toInternal() internalRating {
	return internalRating{
		Mu:    (r.Rating - DefaultRating) / 173.7178,
		Phi:   r.RatingDeviation / 173.7178,
		Sigma: r.Volatility,
	}
}

type GameResult float64

var (
	Loss = GameResult(0)
	Draw = GameResult(0.5)
	Win  = GameResult(1)
)

type internalMatch struct {
	Opponent internalRating
	Result   GameResult
}

func (g internalRating) g() float64 {
	r := g.Phi / math.Pi
	return 1 / math.Sqrt(1+3*r*r)
}

func (g internalRating) estimatedResult(o internalRating) float64 {
	return 1 / (1 + math.Exp(-o.g()*(g.Mu-o.Mu)))
}

func (g internalRating) estimatedVariance(matches []internalMatch) float64 {
	var rv float64
	for _, m := range matches {
		og := m.Opponent.g()
		e := g.estimatedResult(m.Opponent)
		rv += og * og * e * (1 - e)
	}
	return 1.0 / rv
}

func (g internalRating) delta(v float64, matches []internalMatch) float64 {
	var rv float64
	for _, m := range matches {
		og := m.Opponent.g()
		e := g.estimatedResult(m.Opponent)
		rv += og * (float64(m.Result) - e)
	}
	return v * rv
}

const (
	epsilon = 0.000001
)

func (g internalRating) computeVolatility(v, delta, tau float64, matches []internalMatch) float64 {
	a := math.Log(g.Sigma * g.Sigma)

	f := func(x float64) float64 {
		ex := math.Exp(x)

		denom := (g.Phi*g.Phi + v + ex)
		denom *= denom * 2

		num := ex * (delta*delta - g.Phi*g.Phi - v - ex)

		return num/denom - (x-a)/(tau*tau)
	}

	A := math.Log(g.Sigma * g.Sigma)
	var B float64
	if (g.Sigma * g.Sigma) > (g.Phi*g.Phi + v) {
		B = math.Log(delta*delta - g.Phi*g.Phi - v)
	} else {
		k := 1
		for f(a-float64(k)*tau) < 0 {
			k++
		}
		B = a - float64(k)*tau
	}

	fa := f(A)
	fb := f(B)

	for math.Abs(B-A) > epsilon {
		C := A + (A-B)*fa/(fb-fa)
		fc := f(C)
		if (fc * fb) < 0 {
			A = B
			fa = fb
		} else {
			fa = fa / 2
		}
		B = C
		fb = fc
	}

	sigmaPrime := math.Exp(A / 2)

	return sigmaPrime
}

func (g internalRating) update(tau float64, matches []internalMatch) internalRating {
	v := g.estimatedVariance(matches)
	delta := g.delta(v, matches)

	newSigma := g.computeVolatility(v, delta, tau, matches)

	tempPhi := math.Sqrt(g.Phi*g.Phi + newSigma*newSigma)
	newPhi := 1 / math.Sqrt(1/(tempPhi*tempPhi)+1/v)

	newMu := g.Mu + g.delta(newPhi*newPhi, matches)

	return internalRating{
		Mu:    newMu,
		Phi:   newPhi,
		Sigma: newSigma,
	}
}
