package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	tea "github.com/charmbracelet/bubbletea"
)

const apiBase = "http://localhost:8000"

func cmdFetchPlayers() tea.Cmd {
	return func() tea.Msg {
		resp, err := http.Get(apiBase + "/users")
		if err != nil {
			return playerListMsg{err: err}
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)

		var raw []struct {
			Username        string `json:"username"`
			ScorecardsCount int    `json:"scorecards_count"`
			PlayerId				int    `json:"id"`
		}
		if err := json.Unmarshal(body, &raw); err != nil {
			return playerListMsg{err: err}
		}
		players := make([]playerEntry, len(raw))
		for i, u := range raw {
			players[i] = playerEntry{Name: u.Username, Rounds: u.ScorecardsCount, PlayerID: u.PlayerId}
		}
		return playerListMsg{players: players}
	}
}

func cmdFetchRounds(name string) tea.Cmd {
	return func() tea.Msg {
		resp, err := http.Get(apiBase + "/" + name + "/scorecards")
		if err != nil {
			return roundListMsg{err: err}
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)

		var rawRounds []struct {
			ID                int    `json:"id"`
			Course            string `json:"course"`
			DatePlayed        string `json:"date_played"`
			TotalScore        int    `json:"total_score"`
			TotalPar          int    `json:"total_par"`
			ScoreDifferential int    `json:"score_differential"`
		}
		if err := json.Unmarshal(body, &rawRounds); err != nil {
			return roundListMsg{err: err}
		}
		rounds := make([]roundEntry, len(rawRounds))
		for i, r := range rawRounds {
			date := r.DatePlayed
			if len(date) >= 10 {
				date = date[:10]
			}
			rounds[i] = roundEntry{
				ID:     r.ID,
				Course: r.Course,
				Date:   date,
				Score:  r.TotalScore,
				Par:    r.TotalPar,
				Diff:   r.ScoreDifferential,
			}
		}

		var stats playerStatsData
		resp2, err := http.Get(apiBase + "/stats/" + name)
		if err == nil {
			defer resp2.Body.Close()
			body2, _ := io.ReadAll(resp2.Body)
			var rawStats struct {
				TotalRounds      int     `json:"total_rounds"`
				AverageScore     float64 `json:"average_score"`
				BestScore        int     `json:"best_score"`
				WorstScore       int     `json:"worst_score"`
				HandicapEstimate float64 `json:"handicap_estimate"`
			}
			if json.Unmarshal(body2, &rawStats) == nil {
				stats = playerStatsData{
					TotalRounds: rawStats.TotalRounds,
					AvgScore:    rawStats.AverageScore,
					BestScore:   rawStats.BestScore,
					WorstScore:  rawStats.WorstScore,
					Handicap:    rawStats.HandicapEstimate,
				}
			}
		}

		return roundListMsg{rounds: rounds, stats: stats}
	}
}

func cmdFetchRoundDetail(name string, id int) tea.Cmd {
	return func() tea.Msg {
		url := fmt.Sprintf("%s/scorecards/%s/%d", apiBase, name, id)
		resp, err := http.Get(url)
		if err != nil {
			return roundDetailMsg{err: err}
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)

		var raw struct {
			ScorecardID     int    `json:"scorecardId"`
			User            string `json:"user"`
			Course          string `json:"course"`
			UserID          int    `json:"uiserId"`
			Holes           []struct {
				HoleNumber      int `json:"hole_number"`
				Score           int `json:"score"`
				Par             int `json:"par"`
			} `json:"holes"`
		}
		if err := json.Unmarshal(body, &raw); err != nil {
			return roundDetailMsg{err: err}
		}
		if len(raw.Holes) != 18 {
			return roundDetailMsg{err: fmt.Errorf("expected 18 holes, got %d", len(raw.Holes))}
		}

		sc := &scorecardData{
			CourseName: raw.Course,
			ScoreCardId: raw.ScorecardID,
		}
		pd := playerData{
			Name: raw.User,
			ID: raw.UserID,
		}
		for _, h := range raw.Holes {
			i := h.HoleNumber - 1
			if i >= 0 && i < 18 {
				sc.HolePars[i] = h.Par
				pd.Scores[i] = h.Score
			}
		}
		sc.Players = []playerData{pd}

		return roundDetailMsg{sc: sc, roundID: raw.ScorecardID}
	}
}

func cmdSaveScorecard(sc *scorecardData) tea.Cmd {
	type savePlayer struct {
		Name   string  `json:"name"`
		Scores [18]int `json:"scores"`
	}
	type saveCourse struct {
		Name     string  `json:"name"`
		HolePars [18]int `json:"holePars"`
	}
	type savePayload struct {
		Course    saveCourse   `json:"course"`
		Players   []savePlayer `json:"players"`
		ImagePath string       `json:"imagePath"`
	}

	payload := savePayload{
		Course:    saveCourse{Name: sc.CourseName, HolePars: sc.HolePars},
		ImagePath: sc.ImagePath,
	}
	for _, p := range sc.Players {
		payload.Players = append(payload.Players, savePlayer{Name: p.Name, Scores: p.Scores})
	}
	raw, _ := json.Marshal(payload)

	return func() tea.Msg {
		resp, err := http.Post(apiBase+"/scorecards", "application/json", bytes.NewBuffer(raw))
		if err != nil {
			return scorecardSavedMsg{err: err}
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			return scorecardSavedMsg{err: fmt.Errorf("API error %d: %s", resp.StatusCode, body)}
		}
		return scorecardSavedMsg{}
	}
}

func cmdUpdateRound(roundID int, sc *scorecardData) tea.Cmd {
	return func() tea.Msg {
		scores := sc.Players[0].Scores[:]
		body, _ := json.Marshal(scores)
		endpoint := fmt.Sprintf("%s/scorecards/%d?username=%s", apiBase, roundID, url.QueryEscape(sc.Players[0].Name))
		req, _ := http.NewRequest(http.MethodPut, endpoint, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return roundSavedMsg{err: err}
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			b, _ := io.ReadAll(resp.Body)
			return roundSavedMsg{err: fmt.Errorf("API error %d: %s", resp.StatusCode, b)}
		}
		return roundSavedMsg{}
	}
}

func cmdNukeDatabase() tea.Cmd {
	return func() tea.Msg {
		req, _ := http.NewRequest(http.MethodDelete, apiBase+"/nuke", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return cmdOutputMsg{err: err}
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			return cmdOutputMsg{err: fmt.Errorf("API error %d: %s", resp.StatusCode, body)}
		}
		return cmdOutputMsg{output: "Database cleared successfully."}
	}
}

func cmdDeleteRound(scorecard_id int, user_id int) tea.Cmd {
	return func() tea.Msg {
		url := fmt.Sprintf("%s/scorecards/%d/%d", apiBase, scorecard_id, user_id)
		req, _ := http.NewRequest(http.MethodDelete, url, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return cmdOutputMsg{err: err}
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			return cmdOutputMsg{err: fmt.Errorf("API error %d: %s", resp.StatusCode, body)}
		}
		return roundDeletedMsg{}
	}
}
