package card

type CardType string

const (
	CardTypeConcept       CardType = "concept"
	CardTypeSignature     CardType = "signature"
	CardTypeTrace         CardType = "trace"
	CardTypeLineOrder     CardType = "line_order"
	CardTypeChooseSnippet CardType = "choose_snippet"
	CardTypeFixBug        CardType = "fix_bug"
)

func DefaultCardType() CardType {
	return CardTypeConcept
}

func (t CardType) Valid() bool {
	switch t {
	case CardTypeConcept, CardTypeSignature, CardTypeTrace, CardTypeLineOrder, CardTypeChooseSnippet, CardTypeFixBug:
		return true
	default:
		return false
	}
}

func IsBlockCardType(t CardType) bool {
	switch t {
	case CardTypeLineOrder, CardTypeChooseSnippet, CardTypeFixBug:
		return true
	default:
		return false
	}
}
