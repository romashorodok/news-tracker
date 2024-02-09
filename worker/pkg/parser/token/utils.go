package token

import "bytes"

func RemoveNewLine(source []byte) []byte {
	source = bytes.Replace(source, []byte("  "), []byte(""), -1)
	source = bytes.Replace(source, []byte("\t"), []byte(""), -1)
	source = bytes.Replace(source, []byte{'\n'}, []byte{}, -1)
	source = bytes.Replace(source, []byte{'\r'}, []byte{}, -1)
	source = bytes.Replace(source, []byte("\r\n"), []byte{}, -1)
	return source
}
