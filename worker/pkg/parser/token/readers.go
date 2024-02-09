package token

type TokenCursorReader struct {
	data   []byte
	cursor Cursor
	Err    error
}

func (r *TokenCursorReader) Byte() TerminalSymbol {
	symbol := r.data[r.cursor.End]

	if r.cursor.End > len(r.data) {
		r.Err = ErrCursorEnd
		r.cursor.End = len(r.data) - 1
		return symbol
	}

	r.cursor.End++
	return symbol
}

func (r *TokenCursorReader) Data() []byte {
	return r.data[r.cursor.Start:r.cursor.End]
}

func (r *TokenCursorReader) Backward() {
	r.cursor.End--
}

func (r *TokenCursorReader) Len() int {
	return len(r.data)
}

func (r *TokenCursorReader) Cursor() Cursor {
	return r.cursor
}

func (r *TokenCursorReader) SetStart(n int) {
	r.cursor.Start = n
}

func (r *TokenCursorReader) SetEnd(n int) {
	r.cursor.End = n
}

func NewTokenCursorReader(data []byte) *TokenCursorReader {
	return &TokenCursorReader{
		data: data,
	}
}
