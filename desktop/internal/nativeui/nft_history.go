package nativeui

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

func (a *App) readNFTHistory() {
	id := strings.TrimSpace(a.nft.historyID)
	if rejectIncomplete(&a.nft.status, "NFT ID is required for this read.", id) {
		return
	}
	a.clearNFTHistoryForAnotherSubject(id)
	if err := a.loadNFTHistoryPage(id, 0); err != nil {
		a.nft.status = errStatus(err)
	} else {
		a.nft.status = statusLine{}
	}
}

// loadNFTHistory reads one asset's operation log. The id is the whole key: a
// failure keeps the rows only when they describe the same asset.
func (a *App) loadNFTHistory(id string) error {
	offset := a.nft.historyOffset
	if a.nft.historyLoadedID != id {
		offset = 0
	}
	return a.loadNFTHistoryPage(id, offset)
}

// loadNFTHistoryPage commits the subject, offset and rows only after a
// successful read. A failed refresh or pager move therefore leaves the visible
// trail and its position on the page they actually describe.
func (a *App) loadNFTHistoryPage(id string, offset int) error {
	if offset < 0 {
		offset = 0
	}
	ops, err := a.services.NFTHistory(id, nftHistoryPageSize+1, offset)
	if err != nil {
		a.nft.historyErr = err
		a.nft.historyLoadedID = id
		return err
	}
	if len(ops) == 0 && offset > 0 {
		return a.loadNFTHistoryPage(id, 0)
	}
	a.nft.historyOffset = offset
	a.nft.historyHasMore = len(ops) > nftHistoryPageSize
	if a.nft.historyHasMore {
		ops = ops[:nftHistoryPageSize]
	}
	a.nft.history = ops
	a.nft.historyLoadedID = id
	a.nft.historyErr = nil
	return nil
}

func (a *App) clearNFTHistoryForAnotherSubject(id string) {
	if a.nft.historyLoadedID == id {
		return
	}
	a.nft.history = nil
	a.nft.historyErr = nil
	a.nft.historyLoadedID = ""
	a.nft.historyOffset = 0
	a.nft.historyHasMore = false
}

func (a *App) nftHistoryTable(c *ui.Context) {
	if len(a.nft.history) == 0 {
		emptyNote(c, "No operations recorded for this NFT.")
		return
	}
	if previous, next := collectionPager(c, "NFT operation history", a.nft.historyOffset, len(a.nft.history), a.nft.historyHasMore); previous || next {
		target := a.nft.historyOffset
		if previous {
			target = max(0, target-nftHistoryPageSize)
		} else {
			target += nftHistoryPageSize
		}
		_ = a.loadNFTHistoryPage(a.nft.historyLoadedID, target)
		return
	}
	collectionList(c, "NFT operation history",
		collectionRecords(a.nft.history, nftOperationRecord), -1, nil)
}

// nftOperationRecord projects one recorded operation onto a record. The type
// is the title because this trail is read as "what happened, when"; both
// account keys are fields so neither is ever cut to a prefix.
func nftOperationRecord(operation NFTOpRow) collectionRecord {
	return collectionRecord{
		title: tr(operation.Type),
		facts: []collectionFact{
			{label: "Block", value: strconv.FormatInt(operation.BlockHeight, 10)},
		},
		fields: []collectionField{
			{label: "From", value: valueOr(operation.From)},
			{label: "To", value: valueOr(operation.To)},
		},
	}
}
