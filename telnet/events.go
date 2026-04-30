package telnet

// LoginAttemptEvent reports a failed or blocked pre-auth login attempt.
type LoginAttemptEvent struct {
	Action  string
	Reason  string
	Call    string
	Address string
	Detail  string
}

// ConnectionEvent reports telnet client connection lifecycle events.
type ConnectionEvent struct {
	Action  string
	Reason  string
	Call    string
	Address string
}
