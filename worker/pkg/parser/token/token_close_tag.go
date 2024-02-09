package token

type CloseTag struct {
	Name string
}

func (t *CloseTag) unmarshalName(currReader *TokenCursorReader) {
	currReader.SetStart(currReader.Cursor().End)

loop:
	for {
		symbol := currReader.Byte()

		switch symbol {
		case R_BRACKET:
			currReader.Backward()
			break loop
		default:
		}
	}

	t.Name = string(currReader.Data())
}

func (t *CloseTag) Unmarshal(data []byte) (err error) {
	currReader := NewTokenCursorReader(data)

	if symbol := currReader.Byte(); symbol != L_BRACKET {
		return ErrMssingLeftBracket
	}

	if symbol := currReader.Byte(); symbol != SLASH {
		return ErrMssingLeftBracket
	}

	t.unmarshalName(currReader)

	return err
}
