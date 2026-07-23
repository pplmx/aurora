package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/pplmx/aurora/internal/domain/lottery"
	nfterrors "github.com/pplmx/aurora/internal/domain/nft"
	"github.com/pplmx/aurora/internal/domain/oracle"
	tokenerrors "github.com/pplmx/aurora/internal/domain/token"
)

type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// errorClassification maps domain sentinel errors to HTTP status codes
// and machine-readable error codes. Errors not in this map default to
// 500 Internal Server Error, preserving the existing catch-all behaviour
// for genuinely unexpected failures while giving clients actionable
// feedback for recognised domain conditions.
var errorClassification = []struct {
	err        error
	statusCode int
	code       string
}{
	// Token domain errors
	{tokenerrors.ErrTokenNotFound, http.StatusNotFound, "TOKEN_NOT_FOUND"},
	{tokenerrors.ErrInsufficientBalance, http.StatusBadRequest, "INSUFFICIENT_BALANCE"},
	{tokenerrors.ErrInsufficientAllowance, http.StatusBadRequest, "INSUFFICIENT_ALLOWANCE"},
	{tokenerrors.ErrInvalidSignature, http.StatusUnauthorized, "INVALID_SIGNATURE"},
	{tokenerrors.ErrNonceTooLow, http.StatusConflict, "NONCE_TOO_LOW"},
	{tokenerrors.ErrAmountMustBePositive, http.StatusBadRequest, "AMOUNT_MUST_BE_POSITIVE"},
	{tokenerrors.ErrNotTokenOwner, http.StatusForbidden, "NOT_TOKEN_OWNER"},
	{tokenerrors.ErrTokenNotMintable, http.StatusBadRequest, "TOKEN_NOT_MINTABLE"},
	{tokenerrors.ErrTokenNotBurnable, http.StatusBadRequest, "TOKEN_NOT_BURNABLE"},
	{tokenerrors.ErrUnauthorized, http.StatusUnauthorized, "UNAUTHORIZED"},
	{tokenerrors.ErrTransferToZero, http.StatusBadRequest, "TRANSFER_TO_ZERO"},
	{tokenerrors.ErrInvalidAmount, http.StatusBadRequest, "INVALID_AMOUNT"},
	{tokenerrors.ErrDuplicateTransfer, http.StatusConflict, "DUPLICATE_TRANSFER"},

	// NFT domain errors
	{nfterrors.ErrNFTNotFound, http.StatusNotFound, "NFT_NOT_FOUND"},
	{nfterrors.ErrNameRequired, http.StatusBadRequest, "NAME_REQUIRED"},
	{nfterrors.ErrOwnerRequired, http.StatusBadRequest, "OWNER_REQUIRED"},
	{nfterrors.ErrNotOwner, http.StatusForbidden, "NOT_NFT_OWNER"},
	{nfterrors.ErrInvalidSignature, http.StatusUnauthorized, "INVALID_SIGNATURE"},

	// Lottery domain errors
	{lottery.ErrNotFound, http.StatusNotFound, "LOTTERY_NOT_FOUND"},

	// Oracle domain errors
	{oracle.ErrInvalidSource, http.StatusBadRequest, "INVALID_SOURCE"},
	{oracle.ErrSourceNotFound, http.StatusNotFound, "SOURCE_NOT_FOUND"},
	{oracle.ErrSourceDisabled, http.StatusBadRequest, "SOURCE_DISABLED"},
}

// classifyError maps a domain error to an HTTP status code and error code.
// Unrecognised errors default to 500 Internal Server Error.
func classifyError(err error) (int, string) {
	if err == nil {
		return http.StatusOK, "OK"
	}
	for _, ec := range errorClassification {
		if errors.Is(err, ec.err) {
			return ec.statusCode, ec.code
		}
	}
	return http.StatusInternalServerError, "INTERNAL_ERROR"
}

func writeError(w http.ResponseWriter, message string, code string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Error: message,
		Code:  code,
	})
}

// writeUseCaseError maps a use case error to the appropriate HTTP response.
// It inspects the error chain for known domain sentinel errors and returns
// the corresponding status code and machine-readable code. Unknown errors
// fall through to 500.
func writeUseCaseError(w http.ResponseWriter, err error) {
	statusCode, code := classifyError(err)
	writeError(w, err.Error(), code, statusCode)
}

func writeInternalError(w http.ResponseWriter) {
	writeError(w, "internal server error", "INTERNAL_ERROR", http.StatusInternalServerError)
}

func writeBadRequest(w http.ResponseWriter, message string) {
	writeError(w, message, "INVALID_REQUEST", http.StatusBadRequest)
}
