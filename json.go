package walkjson

import (
	"bytes"
	"fmt"
)

const runeEOF = '\uffff'

// -
const (
	Error = iota
	Bool
	Integer
	Float
	String
	List
	Group
	Null
)
const (
	unknown = iota
	lblock
	rblock
	lbr
	rbr
	comma
	colon
	number
	str
	tru
	fals
	float
	null
)

// TParser -
type TParser struct {
	src             *bytes.Reader
	offs, line, col int64
	curRune         rune
	curValue        string
	path            []string
	fn              func(typ int, path []string, key string, val interface{}) bool
	err             error
}

type tToken struct {
	Value string
	Type  int
}

// New -
func New() *TParser {
	return &TParser{}
}

// Reset -
func (o *TParser) Reset(src *bytes.Reader) {
	o.src = src
	o.offs = 0
	o.line = 0
	o.col = 0
	o.curValue = ""
	o.path = []string{}
	o.err = nil
	o.nextRune()
}

func (o *TParser) setError(err error) {
	if o.err == nil {
		o.err = err
	}
}

func (o *TParser) readRune() rune {
	r, w, err := o.src.ReadRune()
	if err != nil {
		return runeEOF
	}
	o.offs += int64(w)
	return r
}

func (o *TParser) nextRune() rune {
	if o.err != nil {
		return runeEOF
	}
	o.curRune = o.readRune()
	// fmt.Printf("%q\n", o.curRune)
	if o.curRune != runeEOF {
		o.col++
	}
	switch o.curRune {
	case '\x0a': // newline
		o.line++
		o.col = 0
	case '\x0d': // linefeed
		o.col = 0
	}
	return o.curRune
}

func (o *TParser) nextToken() tToken {
	t := tToken{}
	if o.err != nil {
		return t
	}
	for {
		r := o.curRune
		if r == runeEOF {
			o.setError(fmt.Errorf("Unexpected EOF"))
			return t
		}

		switch r {
		default:
			t.Value = string(r)
			o.setError(fmt.Errorf("Unknown symbol %q", r))
			return t
		case ' ', '\t', '\x0d', '\x0a':
			o.nextRune()
			continue
		case '{':
			t.Type = lblock
			t.Value = string(r)
			o.nextRune()
		case '}':
			t.Type = rblock
			t.Value = string(r)
			o.nextRune()
		case '[':
			t.Type = lbr
			t.Value = string(r)
			o.nextRune()
		case ']':
			t.Type = rbr
			t.Value = string(r)
			o.nextRune()
		case ',':
			t.Type = comma
			t.Value = string(r)
			o.nextRune()
		case ':':
			t.Type = colon
			t.Value = string(r)
			o.nextRune()
		case 'n':
			if o.nextRune() != 'u' || o.nextRune() != 'l' || o.nextRune() != 'l' {
				o.setError(fmt.Errorf("expected null"))
				return t
			}
			o.nextRune()
			t.Type = null
			t.Value = "null"
		case 't':
			if o.nextRune() != 'r' || o.nextRune() != 'u' || o.nextRune() != 'e' {
				o.setError(fmt.Errorf("expected true"))
				return t
			}
			o.nextRune()
			t.Type = tru
			t.Value = "true"
		case 'f':
			if o.nextRune() != 'a' || o.nextRune() != 'l' || o.nextRune() != 's' || o.nextRune() != 'e' {
				o.setError(fmt.Errorf("expected false"))
				return t
			}
			o.nextRune()
			t.Type = fals
			t.Value = "false"
		case '"':
			t.Type = str
			r = o.nextRune()
			for r != runeEOF && r != '"' {
				t.Value += string(r)
				r = o.nextRune()
				// fmt.Printf("str: %q\n", r)
			}
			o.nextRune()
		case '+', '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			t.Type = number
			if r == '+' || r == '-' {
				t.Value += string(r)
				r = o.nextRune()
			}
			for r != runeEOF && r >= '0' && r <= '9' {
				t.Value += string(r)
				r = o.nextRune()
			}
			if r != '.' && r != 'e' && r != 'E' {
				return t
			}
			fallthrough
		case '.':
			t.Type = float
			if r == '.' {
				t.Value += string(r)
				r = o.nextRune()
				if r < '0' || r > '9' {
					o.setError(fmt.Errorf("invalid floating point value (expectef digit after '.')"))
					t.Type = 0
					return t
				}
				for r != runeEOF && r >= '0' && r <= '9' {
					t.Value += string(r)
					r = o.nextRune()
				}
			}
			if r == 'e' || r == 'E' {
				t.Value += string(r)
				r = o.nextRune()
				if r == '+' || r == '-' {
					t.Value += string(r)
					r = o.nextRune()
				}
				if r < '0' || r > '9' {
					o.setError(fmt.Errorf("invalid floating point value (expectid digit after E)"))
					t.Type = 0
					return t
				}
				for r != runeEOF && r >= '0' && r <= '9' {
					t.Value += string(r)
					r = o.nextRune()
				}
			}
		} // case
		// fmt.Printf("token %v\n", t)
		return t
	} // for
}

func (o *TParser) mustBe(typ int) {
	if o.err != nil {
		return
	}
	t := o.nextToken()
	if t.Type != typ {
		o.setError(fmt.Errorf("unexpected token type %v (expected %v)", t.Type, typ))
	}
	return
}

func (o *TParser) readValType() int {
	if o.err != nil {
		return 0
	}
	typ := 0
	o.mustBe(colon)
	t := o.nextToken()
	o.curValue = t.Value
	switch t.Type {
	case lblock:
		typ = Group
	case lbr:
		typ = List
	case str:
		typ = String
	case number:
		typ = Integer
	case tru, fals:
		typ = Bool
	case float:
		typ = Float
	case null:
		typ = Null
	}
	// fmt.Printf("valType = %v\n", typ)
	return typ
}

func (o *TParser) readList() []string {
	list := []string{}
	if o.err != nil {
		return list
	}
	t := o.nextToken()
	for o.err == nil {
		switch t.Type {
		default:
			o.setError(fmt.Errorf("read list: unexpected %q", t.Value))
			return list
		case rbr:
			return list
		case number, str, tru, fals, float, null:
			list = append(list, t.Value)
			t = o.nextToken()
			switch t.Type {
			default:
				o.setError(fmt.Errorf("read list: unexpected %q (expected ',' or ']')", t.Value))
				return list
			case rbr:
				return list
			case comma:
				t = o.nextToken()
			}
		}
	}
	return list
}

func (o *TParser) readBlock() {
	if o.err != nil {
		return
	}
	// fmt.Printf("readblock\n")
	ok := false
	t := o.nextToken()
	for o.err == nil {
		// fmt.Printf("readblock token %v\n", t)
		switch t.Type {
		default:
			o.setError(fmt.Errorf("read block: unexpected %q", t.Value))
			return
		case rblock:
			return
		case str:
			key := t.Value
			typ := o.readValType()
			switch typ {
			case Integer, String, Bool, Float, Null:
				ok = o.fn(typ, o.path, key, o.curValue)
			case List:
				list := o.readList()
				ok = o.fn(List, o.path, key, list)
			case Group:
				o.path = append(o.path, t.Value)
				ok = o.fn(typ, o.path, key, nil)
				o.readBlock()
				o.path = o.path[:len(o.path)-1]
			}
			if !ok {
				o.setError(fmt.Errorf("walk function abort"))
				return
			}
			t = o.nextToken()
			switch t.Type {
			default:
				o.setError(fmt.Errorf("read block: unexpected %q (expected ',' or '}'", t.Value))
				return
			case comma:
				t = o.nextToken()
			case rblock:
				return
			}
		}
	}
	return
}

// Walk -
func (o *TParser) Walk(fn func(typ int, path []string, key string, val interface{}) bool) error {
	if fn == nil {
		return fmt.Errorf("walk function is null")
	}
	o.fn = fn
	if o.err != nil {
		return o.err
	}
	o.mustBe(lblock)
	o.readBlock()
	if o.err != nil {
		return fmt.Errorf("[0x%03x] (%v,%v) %v", o.offs-1, o.line+1, o.col, o.err)
	}
	return nil
}
