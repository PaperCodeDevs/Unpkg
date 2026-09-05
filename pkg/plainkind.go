package pkg

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"image/gif"
	"io"
	"path"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

type PlainKind string

const (
	PlainEmpty      PlainKind = "empty"
	PlainZip        PlainKind = "zip"
	PlainJSON       PlainKind = "json"
	PlainXML        PlainKind = "xml"
	PlainCSV        PlainKind = "csv"
	PlainText       PlainKind = "text"
	PlainLua        PlainKind = "lua"
	PlainLuaJIT     PlainKind = "luajit"
	PlainBinJSON    PlainKind = "binjson"
	PlainProto      PlainKind = "pbdesc"
	PlainVisual     PlainKind = "v1633v"
	PlainRegion     PlainKind = "region"
	PlainFlatBuf    PlainKind = "flatbuffer"
	PlainRainbow    PlainKind = "rainbow"
	PlainTimeline   PlainKind = "timeline"
	PlainServerList PlainKind = "serverlist"
	PlainOgg        PlainKind = "ogg"
	PlainMidi       PlainKind = "midi"
	PlainDLS        PlainKind = "dls"
	PlainGIF        PlainKind = "gif"
	PlainTTF        PlainKind = "ttf"
	PlainUnknown    PlainKind = "unknown"
)

type PlainOpt struct {
	Zip     ZipKeys
	Rainbow RainbowKeys
}

type PlainInfo struct {
	Kind   PlainKind
	Count  int
	Names  []string
	Detail string
}

var (
	utf8BOM          = []byte{0xEF, 0xBB, 0xBF}
	ErrCipherUnknown = errors.New("serverlist: 8 字节分组密文，密钥未知")
	plainMagics      = []struct {
		magic string
		kind  PlainKind
	}{
		{"PK\x03\x04", PlainZip}, {"\x1bLJ", PlainLuaJIT}, {"OggS", PlainOgg}, {"MThd", PlainMidi}, {"GIF8", PlainGIF},
		{"\x00\x01\x00\x00", PlainTTF}, {"Rainbow\x00", PlainRainbow}, {"v1633v", PlainVisual},
		{"\x5a\x12\xbe\x34\xc0\x18\xc3\x15", PlainServerList},
	}
	plainExts = map[string]PlainKind{
		".json": PlainJSON, ".cfg": PlainJSON, ".xml": PlainXML, ".csv": PlainCSV, ".txt": PlainText,
		".bil": PlainLua, ".lua": PlainLua, ".pb": PlainProto, ".r": PlainRegion, ".fb": PlainFlatBuf,
		".ttf": PlainTTF, ".otf": PlainTTF, ".uprefab": PlainBinJSON, ".uscene": PlainBinJSON,
	}
)

func ClassifyPlain(name string, body []byte) PlainKind {
	if len(body) == 0 {
		return PlainEmpty
	}
	for _, m := range plainMagics {
		if bytes.HasPrefix(body, []byte(m.magic)) {
			return m.kind
		}
	}
	if len(body) >= 12 && bytes.HasPrefix(body, []byte("RIFF")) && string(body[8:12]) == "DLS " {
		return PlainDLS
	}
	ext := strings.ToLower(path.Ext(strings.ReplaceAll(name, "\\", "/")))
	if k, ok := plainExts[ext]; ok {
		return k
	}
	if ext == ".asset" && !looksJSONText(body) {
		return PlainTimeline
	}
	t := bytes.TrimSpace(bytes.TrimPrefix(body, utf8BOM))
	switch {
	case !isTextual(body):
		return PlainUnknown
	case looksJSONText(body):
		return PlainJSON
	case len(t) > 0 && t[0] == '<':
		return PlainXML
	case IsLuaText(body):
		return PlainLua
	case utf8.Valid(body):
		return PlainText
	}
	return PlainUnknown
}

func looksJSONText(b []byte) bool {
	t := bytes.TrimSpace(bytes.TrimPrefix(b, utf8BOM))
	return len(t) > 0 && (t[0] == '{' || t[0] == '[')
}

func VerifyPlain(name string, body []byte, opt PlainOpt) (PlainInfo, error) {
	info := PlainInfo{Kind: ClassifyPlain(name, body)}
	var err error
	switch info.Kind {
	case PlainZip:
		info.Count, info.Names, err = verifyZipMembers(body, opt.Zip)
	case PlainJSON:
		lenient := JSONLenient(body)
		if !json.Valid(lenient) {
			err = errors.New("json: invalid")
		} else if !bytes.Equal(lenient, bytes.TrimPrefix(body, utf8BOM)) {
			info.Detail = "lenient"
		}
	case PlainXML:
		info.Count, err = walkXML(body)
	case PlainCSV, PlainText:
		info.Count, info.Detail, err = verifyText(body)
	case PlainLua:
		if !IsLuaText(body) {
			err = errors.New("lua: no source markers")
		}
	case PlainLuaJIT:
		err = luaJITHeaderOK(body)
	case PlainBinJSON:
		err = binJSONHeaderOK(body)
	case PlainProto:
		info.Names, err = PBDescriptorMessages(body)
		info.Count = len(info.Names)
	case PlainGIF:
		var g *gif.GIF
		if g, err = gif.DecodeAll(bytes.NewReader(body)); err == nil {
			info.Count = len(g.Image)
		}
	case PlainRainbow:
		if png := DecodeRainbow(body, opt.Rainbow); !bytes.HasPrefix(png, pngSignature) {
			err = errors.New("rainbow: xor result is not png")
		}
	case PlainRegion:
		info.Count, info.Detail, err = regionLocCount(body)
	case PlainFlatBuf:
		err = flatBufferRootOK(body)
	case PlainTimeline:
		info.Count, info.Detail, err = timelineHeader(body)
	case PlainServerList:
		info.Detail = "ecb8"
		err = ErrCipherUnknown
	case PlainVisual:
		_, err = base64.StdEncoding.DecodeString(strings.TrimSpace(string(body[6:])))
	case PlainUnknown:
		err = fmt.Errorf("unknown format head=% x", body[:hexHeadLen(body)])
	}
	return info, err
}

func JSONLenient(b []byte) []byte {
	b = bytes.TrimPrefix(b, utf8BOM)
	out := make([]byte, 0, len(b))
	inStr := false
	for i := 0; i < len(b); i++ {
		c := b[i]
		switch {
		case inStr:
			out = append(out, c)
			if c == '\\' && i+1 < len(b) {
				i++
				out = append(out, b[i])
			} else if c == '"' {
				inStr = false
			}
		case c == '"':
			inStr = true
			out = append(out, c)
		case c == '/' && i+1 < len(b) && b[i+1] == '/':
			for i+1 < len(b) && b[i+1] != '\n' {
				i++
			}
		case c == '/' && i+1 < len(b) && b[i+1] == '*':
			if end := bytes.Index(b[i+2:], []byte("*/")); end >= 0 {
				i += end + 3
			} else {
				i = len(b)
			}
		case c == ',' && nextNonSpaceCloses(b, i+1):
		default:
			out = append(out, c)
		}
	}
	return out
}

func nextNonSpaceCloses(b []byte, i int) bool {
	for i < len(b) {
		switch {
		case b[i] == ' ' || b[i] == '\t' || b[i] == '\r' || b[i] == '\n':
			i++
		case b[i] == '/' && i+1 < len(b) && b[i+1] == '/':
			for i < len(b) && b[i] != '\n' {
				i++
			}
		case b[i] == '/' && i+1 < len(b) && b[i+1] == '*':
			end := bytes.Index(b[i+2:], []byte("*/"))
			if end < 0 {
				return false
			}
			i += end + 4
		default:
			return b[i] == '}' || b[i] == ']'
		}
	}
	return false
}

func walkXML(body []byte) (int, error) {
	dec := xml.NewDecoder(bytes.NewReader(bytes.TrimPrefix(body, utf8BOM)))
	dec.CharsetReader = xmlCharsetReader
	n := 0
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return n, fmt.Errorf("xml: %w", err)
		}
		if _, ok := tok.(xml.StartElement); ok {
			n++
		}
	}
	if n == 0 {
		return 0, errors.New("xml: no elements")
	}
	return n, nil
}

func xmlCharsetReader(label string, in io.Reader) (io.Reader, error) {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case "", "utf-8", "utf8", "us-ascii", "ascii":
		return in, nil
	case "gbk", "gb2312", "gb18030", "cp936", "windows-936":
		return transform.NewReader(in, simplifiedchinese.GB18030.NewDecoder()), nil
	}
	return nil, fmt.Errorf("xml: charset %q", label)
}

func verifyText(body []byte) (int, string, error) {
	enc := "utf8"
	if !utf8.Valid(body) {
		if body = decodeCsvGBK(body); body == nil {
			return 0, "", errors.New("text: neither utf8 nor gb18030")
		}
		enc = "gb18030"
	}
	rows := bytes.Count(body, []byte{'\n'})
	if len(body) > 0 && body[len(body)-1] != '\n' {
		rows++
	}
	return rows, enc, nil
}

func IsLuaText(b []byte) bool {
	if LooksLikeLuaSource(b) {
		return true
	}
	head := string(b[:minInt(len(b), 512)])
	hits := 0
	for _, kw := range []string{"local ", "function", "return", "if ", " then", "end", "require", "--"} {
		if strings.Contains(head, kw) {
			hits++
		}
	}
	return hits >= 2 && utf8.Valid(b)
}
