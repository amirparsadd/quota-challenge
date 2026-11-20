package capacity

// ProblemTypeURI is an RFC 9457 type URI (relative to the gateway).
func ProblemTypeURI(slug string) string {
	return "/v1/problems/" + slug
}

// ProblemDocument is RFC 9457 problem JSON stored for idempotent replays.
func ProblemDocument(slug, title string, status int, detail string) map[string]any {
	return map[string]any{
		"type":   ProblemTypeURI(slug),
		"title":  title,
		"status": status,
		"detail": detail,
	}
}

// ReplayHTTPError is returned when an idempotency key replays a stored HTTP error body.
type ReplayHTTPError struct {
	Status int
	Body   []byte
}

func (e *ReplayHTTPError) Error() string { return "idempotency rejection replay" }