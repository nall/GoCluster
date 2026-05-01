package pathreliability

import "strings"

// ReceiverIdentityHash returns a stable non-cryptographic hash for a normalized
// receiving station identity. A zero return means the identity is blank and
// should be treated as unattributed for capped trust.
func ReceiverIdentityHash(identity string) uint64 {
	identity = strings.TrimSpace(identity)
	if identity == "" {
		return 0
	}
	const (
		offset64 = 14695981039346656037
		prime64  = 1099511628211
	)
	hash := uint64(offset64)
	for i := 0; i < len(identity); i++ {
		b := identity[i]
		if b >= 'a' && b <= 'z' {
			b -= 'a' - 'A'
		}
		hash ^= uint64(b)
		hash *= prime64
	}
	if hash == 0 {
		return 1
	}
	return hash
}
