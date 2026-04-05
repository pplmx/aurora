package lottery

type CreateLotteryRequest struct {
	Participants string
	Seed         string
	WinnerCount  int
}

type LotteryResponse struct {
	ID              string
	BlockHeight     int64
	Seed            string
	Participants    []string
	Winners         []string
	WinnerAddresses []string
	VRFProof        string
	VRFOutput       string
	Timestamp       int64
	Verified        bool
}
