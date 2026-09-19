package query

import (
	"context"
	"fmt"
	"io"

	"server/internal/sink"
)

// List prints each team's summed points over a time range
func List(w io.Writer, dbPath string, q sink.ScoreQuery) error {
	db, err := sink.Open(dbPath)
	if err != nil {
		return fmt.Errorf("unable to open results database: %w", err)
	}
	defer db.Close()

	teamScores, err := db.TeamScores(context.Background(), q)
	if err != nil {
		return fmt.Errorf("query team scores: %w", err)
	}

	for _, score := range teamScores {
		fmt.Fprintf(w, "Team %d: %d points\n", score.TeamID, score.Points)
	}

	return nil
}
