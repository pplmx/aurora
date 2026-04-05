package voting

import (
	"time"

	"github.com/google/uuid"
)

var sessionStorage Storage

type VotingSession = DBVotingSession

func SetSessionStorage(s Storage) {
	sessionStorage = s
}

func CreateSession(title, description string, candidateIDs []string, startTime, endTime int64) (*VotingSession, error) {
	session := &VotingSession{
		ID:          uuid.New().String(),
		Title:       title,
		Description: description,
		StartTime:   startTime,
		EndTime:     endTime,
		Status:      "draft",
		Candidates:  candidateIDs,
		CreatedAt:   time.Now().Unix(),
	}

	if err := sessionStorage.SaveSession(session); err != nil {
		return nil, err
	}

	return session, nil
}

func GetSession(id string) (*VotingSession, error) {
	return sessionStorage.GetSession(id)
}

func ListSessions() ([]*VotingSession, error) {
	return sessionStorage.ListSessions()
}

func StartSession(id string) error {
	session, err := sessionStorage.GetSession(id)
	if err != nil {
		return err
	}
	if session == nil {
		return nil
	}

	session.Status = "active"
	session.StartTime = time.Now().Unix()

	return sessionStorage.UpdateSession(session)
}

func EndSession(id string) error {
	session, err := sessionStorage.GetSession(id)
	if err != nil {
		return err
	}
	if session == nil {
		return nil
	}

	session.Status = "ended"
	session.EndTime = time.Now().Unix()

	return sessionStorage.UpdateSession(session)
}

func GetSessionResults(sessionID string) (map[string]int, error) {
	session, err := sessionStorage.GetSession(sessionID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, nil
	}

	results := make(map[string]int)
	for _, cid := range session.Candidates {
		count, err := CountVotes(cid)
		if err != nil {
			return nil, err
		}
		results[cid] = count
	}

	return results, nil
}
