package query

import (
	"context"
	"fmt"
	"io"

	"github.com/andrew-aiken/score/internal/sink"
)

// List prints each team's summed points over a time range
func List(w io.Writer, dbPath string, q sink.ScoreQuery) (err error) {
	db, err := sink.Open(dbPath)
	if err != nil {
		return fmt.Errorf("unable to open results database: %w", err)
	}
	defer func() {
		dbCloseErr := db.Close()
		if err == nil && dbCloseErr != nil {
			err = dbCloseErr
		}
	}()

	teamScores, err := db.TeamScores(context.Background(), q)
	if err != nil {
		return fmt.Errorf("query team scores: %w", err)
	}

	for _, score := range teamScores {
		_, err = fmt.Fprintf(w, "Team %d: %d points\n", score.TeamID, score.Points)
		if err != nil {
			return err
		}
	}

	return nil
}
