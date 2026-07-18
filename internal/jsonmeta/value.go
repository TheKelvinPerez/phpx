package jsonmeta

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
)

const DefaultMaxDepth = 128

type Kind uint8

const (
	KindNull Kind = iota
	KindBoolean
	KindNumber
	KindString
	KindObject
	KindArray
)

type Member struct {
	Name  string
	Value *Value
}

type Value struct {
	kind     Kind
	boolean  bool
	number   json.Number
	text     string
	members  []Member
	elements []*Value
}

type DuplicateKeyError struct {
	Key string
}

func (err *DuplicateKeyError) Error() string {
	return fmt.Sprintf("duplicate object key %q", err.Key)
}

type RootTypeError struct{}

func (*RootTypeError) Error() string {
	return "root value is not an object"
}

type DepthError struct {
	Limit int
}

func (err *DepthError) Error() string {
	return fmt.Sprintf("JSON exceeds the %d level nesting limit", err.Limit)
}

func DecodeObject(content []byte, maxDepth int) (*Value, error) {
	if !utf8.Valid(content) {
		return nil, errors.New("JSON is not valid UTF 8")
	}
	if err := validateEscapedSurrogates(content); err != nil {
		return nil, err
	}
	if maxDepth <= 0 {
		maxDepth = DefaultMaxDepth
	}

	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.UseNumber()

	value, err := consumeValue(decoder, 0, maxDepth)
	if err != nil {
		return nil, err
	}
	if value.kind != KindObject {
		return nil, &RootTypeError{}
	}

	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("JSON contains more than one root value")
		}

		return nil, err
	}

	return value, nil
}

func validateEscapedSurrogates(content []byte) error {
	inString := false
	for index := 0; index < len(content); index++ {
		switch {
		case !inString:
			if content[index] == '"' {
				inString = true
			}
		case content[index] == '"':
			inString = false
		case content[index] == '\\':
			if index+1 >= len(content) {
				continue
			}
			if content[index+1] != 'u' {
				index++
				continue
			}

			codePoint, ok := escapedCodePoint(content, index)
			if !ok {
				return errors.New("JSON contains an invalid Unicode escape")
			}
			switch {
			case codePoint >= 0xd800 && codePoint <= 0xdbff:
				lowOffset := index + 6
				lowCodePoint, lowOK := escapedCodePoint(content, lowOffset)
				if !lowOK || lowCodePoint < 0xdc00 || lowCodePoint > 0xdfff {
					return errors.New("JSON contains an unpaired UTF 16 surrogate escape")
				}
				index = lowOffset + 5
			case codePoint >= 0xdc00 && codePoint <= 0xdfff:
				return errors.New("JSON contains an unpaired UTF 16 surrogate escape")
			default:
				index += 5
			}
		}
	}

	return nil
}

func escapedCodePoint(content []byte, offset int) (uint16, bool) {
	if offset < 0 ||
		offset+5 >= len(content) ||
		content[offset] != '\\' ||
		content[offset+1] != 'u' {
		return 0, false
	}

	var value uint16
	for _, digit := range content[offset+2 : offset+6] {
		value <<= 4
		switch {
		case digit >= '0' && digit <= '9':
			value |= uint16(digit - '0')
		case digit >= 'a' && digit <= 'f':
			value |= uint16(digit-'a') + 10
		case digit >= 'A' && digit <= 'F':
			value |= uint16(digit-'A') + 10
		default:
			return 0, false
		}
	}

	return value, true
}

func (value *Value) Kind() Kind {
	return value.kind
}

func (value *Value) Member(name string) (*Value, bool) {
	if value.kind != KindObject {
		return nil, false
	}
	for _, member := range value.members {
		if member.Name == name {
			return member.Value, true
		}
	}

	return nil, false
}

func (value *Value) Members() []Member {
	return append([]Member(nil), value.members...)
}

func (value *Value) Elements() []*Value {
	return append([]*Value(nil), value.elements...)
}

func (value *Value) StringValue() (string, bool) {
	return value.text, value.kind == KindString
}

func (value *Value) NumberValue() (json.Number, bool) {
	return value.number, value.kind == KindNumber
}

func (value *Value) BooleanValue() (bool, bool) {
	return value.boolean, value.kind == KindBoolean
}

func consumeValue(decoder *json.Decoder, depth int, maxDepth int) (*Value, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}

	switch typed := token.(type) {
	case nil:
		return &Value{kind: KindNull}, nil
	case bool:
		return &Value{kind: KindBoolean, boolean: typed}, nil
	case json.Number:
		return &Value{kind: KindNumber, number: typed}, nil
	case string:
		return &Value{kind: KindString, text: typed}, nil
	case json.Delim:
		if depth >= maxDepth {
			return nil, &DepthError{Limit: maxDepth}
		}
		switch typed {
		case '{':
			return consumeObject(decoder, depth+1, maxDepth)
		case '[':
			return consumeArray(decoder, depth+1, maxDepth)
		default:
			return nil, fmt.Errorf("unexpected JSON delimiter %q", typed)
		}
	default:
		return nil, fmt.Errorf("unsupported JSON token %T", token)
	}
}

func consumeObject(decoder *json.Decoder, depth int, maxDepth int) (*Value, error) {
	value := &Value{kind: KindObject}
	keys := make(map[string]struct{})

	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		key, ok := token.(string)
		if !ok {
			return nil, errors.New("JSON object key is not a string")
		}
		if _, exists := keys[key]; exists {
			return nil, &DuplicateKeyError{Key: key}
		}
		keys[key] = struct{}{}

		memberValue, err := consumeValue(decoder, depth, maxDepth)
		if err != nil {
			return nil, err
		}
		value.members = append(value.members, Member{
			Name:  key,
			Value: memberValue,
		})
	}

	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	if token != json.Delim('}') {
		return nil, errors.New("JSON object is not closed")
	}

	return value, nil
}

func consumeArray(decoder *json.Decoder, depth int, maxDepth int) (*Value, error) {
	value := &Value{kind: KindArray}
	for decoder.More() {
		element, err := consumeValue(decoder, depth, maxDepth)
		if err != nil {
			return nil, err
		}
		value.elements = append(value.elements, element)
	}

	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	if token != json.Delim(']') {
		return nil, errors.New("JSON array is not closed")
	}

	return value, nil
}
