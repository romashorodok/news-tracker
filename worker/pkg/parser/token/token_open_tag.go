package token

type OpenTag struct {
	Name string
	Attr map[string]string
}

func (t *OpenTag) unmarshalAttrKey(currReader *TokenCursorReader) {
	for {
		symbol := currReader.Byte()

		switch symbol {
		case SPACE, NEW_LINE, C_RETURN, TAB, FORM_FEED, SLASH:
			currReader.Backward()
			return
		case EQUALS:
			fallthrough
		case R_BRACKET:
			currReader.Backward()
			return
		}
	}
}

func (t *OpenTag) unmarshalAttrValue(currReader *TokenCursorReader) {
	for {
		symbol := currReader.Byte()
		if symbol != EQUALS {
			currReader.Backward()
		}

		quote := currReader.Byte()

		switch quote {
		case R_BRACKET:
			currReader.Backward()
			return
		case SINGLE_QUOTE, DOUBLE_QUOTE:
			currReader.SetStart(currReader.Cursor().End)

			for {
				symbol := currReader.Byte()
				if symbol == quote {
					currReader.Backward()
					return
				}
			}
		}

	}
}

func (t *OpenTag) unmarshalAttr(currReader *TokenCursorReader) {
	for {
		currReader.SetStart(currReader.Cursor().End)

		symbol := currReader.Byte()
		if symbol == '>' {
			return
		}

		if symbol == SPACE {
			currReader.SetStart(currReader.Cursor().Start + 1)
		}

		t.unmarshalAttrKey(currReader)
		attrKey := string(RemoveNewLine(currReader.Data()))
		currReader.SetStart(currReader.Cursor().End)

		t.unmarshalAttrValue(currReader)
		attrValue := string(RemoveNewLine(currReader.Data()))

		quote := currReader.Byte()

		switch quote {
		case SINGLE_QUOTE, DOUBLE_QUOTE:
		default:
			currReader.Backward()
		}

		if attrKey == "" || attrValue == "" {
			continue
		}

		t.Attr[attrKey] = attrValue
	}
}

func (t *OpenTag) unmarshalName(curReader *TokenCursorReader) {
	curReader.SetStart(curReader.Cursor().End)

loop:
	for {
		symbol := curReader.Byte()

		switch symbol {
		case SLASH, R_BRACKET, SPACE, NEW_LINE, C_RETURN, TAB, FORM_FEED:
			curReader.Backward()
			break loop
		default:
		}
	}

	t.Name = string(curReader.Data())
}

func (t *OpenTag) Unmarshal(data []byte) (err error) {
	currReader := NewTokenCursorReader(data)

	if symbol := currReader.Byte(); symbol != L_BRACKET {
		return ErrMssingLeftBracket
	}

	t.unmarshalName(currReader)
	if currReader.Cursor().End == currReader.Len() {
		return nil
	}

	t.Attr = make(map[string]string)

	t.unmarshalAttr(currReader)

	return err
}
