package lucene

// Error is the error type returned by the library. Op names the operation that
// failed (for example "add" or "parse") and Msg describes the problem.
type Error struct {
	Op  string
	Msg string
}

func (e *Error) Error() string {
	return "lucene: " + e.Op + ": " + e.Msg
}
