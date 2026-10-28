package deskapp

import (
	"github.com/pplmx/aurora/desktop/internal/nativeui"

	"github.com/pplmx/aurora/internal/app/lotteryapp"
	"github.com/pplmx/aurora/internal/i18n"
)

// --- Lottery -------------------------------------------------------------

func (s *nativeServices) LotteryHistory(limit, offset int) ([]nativeui.DrawRow, error) {
	records, err := s.lotteryRepo.List(limit, offset)
	if err != nil {
		return nil, err
	}
	rows := make([]nativeui.DrawRow, 0, len(records))
	for _, r := range records {
		rows = append(rows, nativeui.DrawRow{
			ID: r.ID, BlockHeight: r.BlockHeight, Seed: r.Seed,
			Participants: r.Participants, Winners: r.Winners, Timestamp: r.Timestamp,
		})
	}
	return rows, nil
}

func (s *nativeServices) CreateDraw(participants []string, seed string, winners int) (nativeui.DrawRow, error) {
	uc := lotteryapp.NewCreate(s.lotteryRepo, s.chain)
	resp, err := uc.Execute(lotteryapp.CreateLotteryRequest{
		Participants: joinCSV(participants), Seed: seed, WinnerCount: winners,
	})
	if err != nil {
		return nativeui.DrawRow{}, err
	}
	return nativeui.DrawRow{
		ID: resp.ID, BlockHeight: resp.BlockHeight, Seed: resp.Seed,
		Participants: resp.Participants, Winners: resp.Winners, Timestamp: resp.Timestamp,
	}, nil
}

func (s *nativeServices) VerifyDraw(id string) (nativeui.VerifyResult, error) {
	report, err := lotteryapp.VerifyInput(s.lotteryRepo, id)
	if err != nil {
		return nativeui.VerifyResult{}, err
	}
	res := nativeui.VerifyResult{ID: report.Record.ID, Valid: report.IntegrityOK}
	for _, c := range report.Checks {
		res.Checks = append(res.Checks,
			i18n.LocalizeVerifyStatus(c.Status)+": "+i18n.LocalizeVerifyCheckMessage(c.Message))
	}
	return res, nil
}
