package pathreliability

// BucketClass selects which aggregation bucket family to update.
type BucketClass uint8

const (
	BucketNone BucketClass = iota
	BucketCombined
)

// BucketForIngest maps a mode to its ingest bucket class.
// Modes not explicitly listed return BucketNone (no ingest).
func BucketForIngest(mode string) BucketClass {
	if policy, ok := modePolicy(mode); ok && policy.Ingest {
		return BucketCombined
	}
	return BucketNone
}
