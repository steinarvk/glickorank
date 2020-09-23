# glickorank

A simple implementation of the Glicko-2 rating system.

Input is on stdin and output is on stdout.

Ratings are of the form:

```
   [timestamp] [game/class] [rating] [player] rd=[ratingdeviation] v=[volatility]
```

Match results are of the form:

```
   [timestamp] [game/class] [1-0/0-1/1-1] [player1] [player2]
```

Input consists of an optional section of pre-existing ratings,
followed by a section of match results.

Output consists of ratings.

The `game/class` feature allows one to track several parallel games at once;
for instance chess and go, or different game modes in a video game.
At any rate, the ratings for the different game classes will be completely
independent.
