package spot

import "strings"

type EventMask uint64

const (
	EventLLOTA EventMask = 1 << iota
	EventIOTA
	EventPOTA
	EventSOTA
	EventWWFF
)

func SupportedEvents() []string {
	return CurrentTaxonomy().SupportedEvents()
}

func EventMaskForName(event string) EventMask {
	return CurrentTaxonomy().EventMaskForName(event)
}

func NormalizeEvent(event string) string {
	return CurrentTaxonomy().NormalizeEvent(event)
}

func EventNames(mask EventMask) []string {
	return CurrentTaxonomy().EventNames(mask)
}

func EventString(mask EventMask) string {
	return CurrentTaxonomy().EventString(mask)
}

func ParseEventString(value string) EventMask {
	return CurrentTaxonomy().ParseEventString(value)
}

func eventFromCommentToken(upperToken string) EventMask {
	return CurrentTaxonomy().EventFromCommentToken(strings.ToUpper(strings.TrimSpace(upperToken)))
}
