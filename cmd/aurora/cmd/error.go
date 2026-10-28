package cmd

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/pplmx/aurora/internal/app/lotteryapp"
	"github.com/pplmx/aurora/internal/domain/lottery"
	"github.com/pplmx/aurora/internal/domain/nft"
	"github.com/pplmx/aurora/internal/domain/oracle"
	"github.com/pplmx/aurora/internal/domain/repoerr"
	"github.com/pplmx/aurora/internal/domain/token"
	"github.com/pplmx/aurora/internal/domain/voting"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/infra/blockchain"
)

// cliError keeps command context and its underlying cause separate until the
// error is rendered. Resolving the message at Error() time makes the same
// error follow a locale switch, while Unwrap preserves errors.Is/As for
// callers that classify domain failures.
type cliError struct {
	key   string
	args  []any
	cause error
}

func (e *cliError) Error() string {
	message := i18n.GetTextF(e.key, e.args...)
	if e.cause == nil {
		return message
	}
	return i18n.GetTextF("cli.error.with_cause", message, localizedErrorText(e.cause))
}

func (e *cliError) Unwrap() error {
	return e.cause
}

func newCLIError(key string, args ...any) error {
	return &cliError{key: key, args: args}
}

func wrapCLIError(key string, cause error) error {
	return &cliError{key: key, cause: cause}
}

func wrapCLIErrorf(key string, cause error, args ...any) error {
	return &cliError{key: key, args: args, cause: cause}
}

// localizedErrorText translates known domain sentinels even when an app-layer
// wrapper hides them behind fmt.Errorf. Unknown technical errors stay intact:
// inventing a translation would hide the actionable operating-system or
// database detail. English preserves every existing error string verbatim.
func localizedErrorText(err error) string {
	if err == nil {
		return ""
	}
	if localized, ok := err.(*cliError); ok {
		return localized.Error()
	}
	if i18n.GetTranslator().GetLocale() != "en" {
		if translated, ok := localizeCobraErrorText(err.Error()); ok {
			return translated
		}
		for _, entry := range cliErrorTranslations {
			if i18n.HasText(entry.key) && errors.Is(err, entry.err) {
				return i18n.GetText(entry.key)
			}
		}
	}
	return err.Error()
}

var (
	requiredFlagsPattern    = regexp.MustCompile(`^required flag\(s\) (.+) not set$`)
	unknownFlagPattern      = regexp.MustCompile(`^unknown flag: (.+)$`)
	unknownShorthandPattern = regexp.MustCompile(`^unknown shorthand flag: '([^']+)' in (.+)$`)
	flagArgumentPattern     = regexp.MustCompile(`^flag needs an argument: (.+)$`)
	invalidFlagValuePattern = regexp.MustCompile(`^invalid argument ("(?:[^"\\]|\\.)*") for ("(?:[^"\\]|\\.)*") flag: (.+)$`)
	unknownCommandPattern   = regexp.MustCompile(`^unknown command ("(?:[^"\\]|\\.)*") for ("(?:[^"\\]|\\.)*")$`)
	exactArgsPattern        = regexp.MustCompile(`^accepts (\d+) arg\(s\), received (\d+)$`)
	atMostArgsPattern       = regexp.MustCompile(`^accepts at most (\d+) arg\(s\), received (\d+)$`)
	betweenArgsPattern      = regexp.MustCompile(`^accepts between (\d+) and (\d+) arg\(s\), received (\d+)$`)
	atLeastArgsPattern      = regexp.MustCompile(`^requires at least (\d+) arg\(s\), only received (\d+)$`)
)

// Cobra returns plain errors for flag parsing, required flags, command lookup
// and positional-argument validation, without typed error values. Keep this
// display-only translation narrowly anchored to Cobra's documented wording;
// the underlying error and exit behavior are untouched.
func localizeCobraErrorText(message string) (string, bool) {
	line, suffix := message, ""
	if at := strings.IndexByte(message, '\n'); at >= 0 {
		line, suffix = message[:at], message[at:]
	}

	if match := requiredFlagsPattern.FindStringSubmatch(line); match != nil {
		return "缺少必填参数：" + match[1] + localizeSuggestionSuffix(suffix), true
	}
	if match := unknownFlagPattern.FindStringSubmatch(line); match != nil {
		return "未知参数：" + match[1] + localizeSuggestionSuffix(suffix), true
	}
	if match := unknownShorthandPattern.FindStringSubmatch(line); match != nil {
		return fmt.Sprintf("未知短参数：'%s'（位于 %s）", match[1], match[2]) +
			localizeSuggestionSuffix(suffix), true
	}
	if match := flagArgumentPattern.FindStringSubmatch(line); match != nil {
		return "参数 " + match[1] + " 缺少值" + localizeSuggestionSuffix(suffix), true
	}
	if match := invalidFlagValuePattern.FindStringSubmatch(line); match != nil {
		detail := localizeFlagParseDetail(match[3])
		return fmt.Sprintf("参数 %s 的值 %s 无效：%s", match[2], match[1], detail) +
			localizeSuggestionSuffix(suffix), true
	}
	if match := unknownCommandPattern.FindStringSubmatch(line); match != nil {
		return fmt.Sprintf("未知命令：%s（命令路径：%s）", match[1], match[2]) +
			localizeSuggestionSuffix(suffix), true
	}
	if match := exactArgsPattern.FindStringSubmatch(line); match != nil {
		return fmt.Sprintf("需要 %s 个参数，实际收到 %s 个", match[1], match[2]) +
			localizeSuggestionSuffix(suffix), true
	}
	if match := atMostArgsPattern.FindStringSubmatch(line); match != nil {
		return fmt.Sprintf("最多接受 %s 个参数，实际收到 %s 个", match[1], match[2]) +
			localizeSuggestionSuffix(suffix), true
	}
	if match := betweenArgsPattern.FindStringSubmatch(line); match != nil {
		return fmt.Sprintf("需要 %s 到 %s 个参数，实际收到 %s 个", match[1], match[2], match[3]) +
			localizeSuggestionSuffix(suffix), true
	}
	if match := atLeastArgsPattern.FindStringSubmatch(line); match != nil {
		return fmt.Sprintf("至少需要 %s 个参数，实际只收到 %s 个", match[1], match[2]) +
			localizeSuggestionSuffix(suffix), true
	}
	return "", false
}

func localizeFlagParseDetail(detail string) string {
	switch {
	case strings.Contains(detail, "strconv.ParseBool"):
		return "必须是 true 或 false"
	case strings.Contains(detail, "strconv.ParseInt"),
		strings.Contains(detail, "strconv.ParseUint"),
		strings.Contains(detail, "strconv.Atoi"):
		if strings.Contains(detail, "value out of range") {
			return "超出整数范围"
		}
		return "不是有效整数"
	case strings.HasPrefix(detail, "time: invalid duration"):
		return "时间格式无效"
	default:
		return detail
	}
}

func localizeSuggestionSuffix(suffix string) string {
	return strings.Replace(suffix, "Did you mean this?", "你是不是想输入：", 1)
}

type cliErrorTranslation struct {
	err error
	key string
}

var cliErrorTranslations = []cliErrorTranslation{
	{token.ErrTokenNotFound, "error.token_not_found"},
	{token.ErrInsufficientBalance, "error.insufficient_balance"},
	{token.ErrInsufficientAllowance, "error.insufficient_allowance"},
	{token.ErrInvalidSignature, "error.invalid_signature"},
	{token.ErrAmountMustBePositive, "error.amount_positive"},
	{token.ErrNotTokenOwner, "error.not_token_owner"},
	{token.ErrZeroOwner, "error.zero_owner"},
	{token.ErrTokenNotMintable, "error.token_not_mintable"},
	{token.ErrTokenNotBurnable, "error.token_not_burnable"},
	{token.ErrUnauthorized, "error.unauthorized"},
	{token.ErrTransferToZero, "error.transfer_to_zero"},
	{token.ErrInvalidAmount, "error.invalid_amount"},
	{token.ErrDuplicateTransfer, "error.duplicate_transfer"},
	{token.ErrTokenExists, "error.token_exists"},
	{token.ErrTokenNameRequired, "error.token_name_required"},
	{token.ErrTokenNameTooLong, "error.token_name_too_long"},
	{token.ErrTokenSymbolRequired, "error.token_symbol_required"},
	{token.ErrTokenSymbolTooLong, "error.token_symbol_too_long"},
	{token.ErrPublicKeyRequired, "error.public_key_required"},
	{token.ErrInvalidPublicKeyLength, "error.invalid_public_key_length"},
	{token.ErrPrivateKeyRequired, "error.private_key_required"},
	{token.ErrInvalidPrivateKeyLength, "error.invalid_private_key_length"},
	{token.ErrInvalidBase64, "error.invalid_base64"},
	{token.ErrTokenDecimalsInvalid, "error.invalid_decimals"},
	{token.ErrAmountTooLarge, "error.amount_too_large"},
	{token.ErrAuditPublishFailed, "error.audit_publish_failed"},

	{nft.ErrNameRequired, "error.nft_name_required"},
	{nft.ErrOwnerRequired, "error.nft_owner_required"},
	{nft.ErrNotOwner, "error.not_nft_owner"},
	{nft.ErrNFTNotFound, "error.nft_not_found"},
	{nft.ErrZeroAddress, "error.zero_address"},
	{nft.ErrStorageNotInit, "error.storage_not_initialized"},
	{nft.ErrInvalidSignature, "error.invalid_signature"},
	{nft.ErrInvalidPrivateKey, "error.invalid_private_key_length"},
	{nft.ErrInvalidPublicKey, "error.invalid_public_key_length"},
	{nft.ErrInvalidBase64, "error.invalid_base64"},
	{nft.ErrKeyMismatch, "error.key_mismatch"},
	{nft.ErrNameTooLong, "error.nft_name_too_long"},
	{nft.ErrDescriptionTooLong, "error.nft_description_too_long"},
	{nft.ErrImageURLTooLong, "error.nft_image_url_too_long"},
	{nft.ErrTokenURITooLong, "error.nft_token_uri_too_long"},

	{voting.ErrSessionNotFound, "error.session_not_found"},
	{voting.ErrSessionNotStarted, "error.session_not_started"},
	{voting.ErrSessionEnded, "error.session_ended"},
	{voting.ErrVoterNotRegistered, "error.voter_not_registered"},
	{voting.ErrCandidateNotFound, "error.candidate_not_found"},
	{voting.ErrCandidateNotInSession, "error.candidate_not_in_session"},
	{voting.ErrAlreadyVoted, "error.already_voted"},
	{voting.ErrInvalidSignature, "error.invalid_vote_signature"},
	{voting.ErrInvalidBase64, "error.invalid_base64"},
	{voting.ErrInvalidPrivateKey, "error.invalid_private_key_length"},
	{voting.ErrVoterNameRequired, "error.voter_name_required"},
	{voting.ErrCandidateNameRequired, "error.candidate_name_required"},
	{voting.ErrSessionTitleRequired, "error.session_title_required"},
	{voting.ErrInvalidSessionTime, "error.invalid_session_time"},
	{voting.ErrCandidatesRequired, "error.candidates_required"},
	{voting.ErrDuplicateCandidate, "error.duplicate_candidate"},
	{voting.ErrVoterNameTooLong, "error.voter_name_too_long"},
	{voting.ErrCandidateNameTooLong, "error.candidate_name_too_long"},
	{voting.ErrCandidatePartyTooLong, "error.candidate_party_too_long"},
	{voting.ErrCandidateProgramTooLong, "error.candidate_program_too_long"},
	{voting.ErrSessionTitleTooLong, "error.session_title_too_long"},
	{voting.ErrSessionDescriptionTooLong, "error.session_description_too_long"},

	{oracle.ErrInvalidSource, "error.invalid_source"},
	{oracle.ErrSourceNotFound, "error.source_not_found"},
	{oracle.ErrSourceDisabled, "error.source_disabled"},
	{oracle.ErrRateLimited, "error.rate_limited"},

	{lottery.ErrNotFound, "error.lottery_not_found"},
	{lottery.ErrEmptyParticipantName, "error.empty_participant_name"},
	{lottery.ErrParticipantNameTooLong, "error.participant_name_too_long"},
	{lottery.ErrInvalidParticipantName, "error.invalid_participant_name"},
	{lottery.ErrSeedTooShort, "error.seed_too_short"},
	{lottery.ErrSeedTooLong, "error.seed_too_long"},
	{lottery.ErrNoParticipants, "error.no_participants"},
	{lottery.ErrTooManyParticipants, "error.too_many_participants"},
	{lottery.ErrDuplicateParticipant, "error.duplicate_participant"},
	{lottery.ErrWinnerCountNotPositive, "error.winner_count_not_positive"},
	{lottery.ErrTooManyWinners, "error.too_many_winners"},
	{lottery.ErrWinnersExceedParticipants, "error.winners_exceed_participants"},
	{lotteryapp.ErrVerificationFailed, "error.integrity_failed"},

	{repoerr.ErrNotFound, "error.record_not_found"},
	{blockchain.ErrLedgerCorruption, "error.ledger_corruption"},
	{blockchain.ErrHeightConflict, "error.height_conflict"},
}
