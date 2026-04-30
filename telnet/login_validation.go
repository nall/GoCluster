package telnet

import (
	"log"
	"strings"

	"dxcluster/spot"
	"dxcluster/uls"
)

type loginValidationReason string

const (
	loginValidationReasonSyntaxInvalid  loginValidationReason = "syntax_invalid"
	loginValidationReasonCTYUnavailable loginValidationReason = "cty_unavailable"
	loginValidationReasonCTYUnknown     loginValidationReason = "cty_unknown"
	loginValidationReasonUSUnlicensed   loginValidationReason = "us_unlicensed"
)

type loginValidationResult struct {
	valid    bool
	failOpen bool
	reason   loginValidationReason
}

func (s *Server) validateLoginCallsign(call string) loginValidationResult {
	if !isValidLoginCallsign(call) {
		return loginValidationResult{reason: loginValidationReasonSyntaxInvalid}
	}
	if s == nil || s.ctyLookup == nil {
		return loginValidationResult{valid: true}
	}
	db := s.ctyLookup()
	if db == nil {
		return loginValidationResult{valid: true, failOpen: true, reason: loginValidationReasonCTYUnavailable}
	}

	lookupCall := call
	testBase, isTestCall := loginTestCallBase(call)
	if isTestCall {
		lookupCall = testBase
	}
	info, ok := db.LookupCallsignPortable(lookupCall)
	if !ok {
		return loginValidationResult{reason: loginValidationReasonCTYUnknown}
	}
	if info.ADIF == 291 && !isTestCall {
		licenseCall := strings.TrimSpace(uls.NormalizeForLicense(call))
		if licenseCall == "" {
			licenseCall = call
		}
		if uls.AllowlistMatch(info.ADIF, licenseCall) {
			return loginValidationResult{valid: true}
		}
		if s.usLicenseCheck == nil {
			return loginValidationResult{valid: true, failOpen: true, reason: loginValidationReasonCTYUnavailable}
		}
		if !s.usLicenseCheck(licenseCall) {
			return loginValidationResult{reason: loginValidationReasonUSUnlicensed}
		}
	}
	return loginValidationResult{valid: true}
}

func isValidLoginCallsign(call string) bool {
	call = strings.TrimSpace(call)
	return spot.IsValidNormalizedCallsign(call) && containsASCIILetter(call) && !strings.Contains(call, "#")
}

func containsASCIILetter(s string) bool {
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch >= 'A' && ch <= 'Z' {
			return true
		}
	}
	return false
}

func loginTestCallBase(call string) (string, bool) {
	call = strings.TrimSpace(call)
	if call == "" || strings.Contains(call, "/") {
		return "", false
	}
	if strings.HasSuffix(call, "TEST") {
		return call, true
	}
	idx := strings.LastIndexByte(call, '-')
	if idx <= 0 || idx >= len(call)-1 {
		return "", false
	}
	for _, ch := range call[idx+1:] {
		if ch < '0' || ch > '9' {
			return "", false
		}
	}
	base := call[:idx]
	if strings.HasSuffix(base, "TEST") {
		return base, true
	}
	return "", false
}

func (s *Server) logLoginValidation(action string, reason loginValidationReason, call, address string) {
	if s == nil || reason == "" {
		return
	}
	if action != "skipped" {
		s.reportLoginAttempt(action, string(reason), call, address, "")
	}
	total, ok := s.loginValidationLog.Inc()
	if !ok {
		return
	}
	call = strings.TrimSpace(call)
	if call == "" {
		call = "(empty)"
	}
	log.Printf("Login callsign validation %s: reason=%s call=%s addr=%s total=%d", action, reason, call, address, total)
}

func (s *Server) reportLoginAttempt(action, reason, call, address, detail string) {
	if s == nil || s.loginAttemptReporter == nil {
		return
	}
	s.loginAttemptReporter(LoginAttemptEvent{
		Action:  action,
		Reason:  reason,
		Call:    call,
		Address: address,
		Detail:  detail,
	})
}

func (s *Server) reportConnection(action, reason, call, address string) {
	if s == nil || s.connectionReporter == nil {
		return
	}
	s.connectionReporter(ConnectionEvent{
		Action:  action,
		Reason:  reason,
		Call:    call,
		Address: address,
	})
}
