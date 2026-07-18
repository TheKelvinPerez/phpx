package composer

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/elefantephp/elefante/internal/jsonmeta"
)

type hashMember struct {
	name     string
	value    *jsonmeta.Value
	platform *jsonmeta.Value
}

func composerContentHash(content []byte) (string, error) {
	root, err := jsonmeta.DecodeObject(content, jsonmeta.DefaultMaxDepth)
	if err != nil {
		return "", err
	}

	var members []hashMember
	for _, member := range root.Members() {
		if composerHashKey(member.Name) && member.Value.Kind() != jsonmeta.KindNull {
			members = append(members, hashMember{
				name:  member.Name,
				value: member.Value,
			})
		}
	}
	if config, exists := root.Member("config"); exists {
		if platform, exists := config.Member("platform"); exists &&
			platform.Kind() != jsonmeta.KindNull {
			members = append(members, hashMember{
				name:     "config",
				platform: platform,
			})
		}
	}
	sort.Slice(members, func(left int, right int) bool {
		return members[left].name < members[right].name
	})

	var encoded bytes.Buffer
	encoded.WriteByte('{')
	for index, member := range members {
		if index > 0 {
			encoded.WriteByte(',')
		}
		writePHPJSONString(&encoded, member.name)
		encoded.WriteByte(':')
		if member.platform != nil {
			encoded.WriteString(`{"platform":`)
			if err := writePHPJSONValue(&encoded, member.platform); err != nil {
				return "", err
			}
			encoded.WriteByte('}')
			continue
		}
		if err := writePHPJSONValue(&encoded, member.value); err != nil {
			return "", err
		}
	}
	encoded.WriteByte('}')

	sum := md5.Sum(encoded.Bytes())

	return hex.EncodeToString(sum[:]), nil
}

func composerHashKey(name string) bool {
	switch name {
	case "name",
		"version",
		"require",
		"require-dev",
		"conflict",
		"replace",
		"provide",
		"minimum-stability",
		"prefer-stable",
		"repositories",
		"extra":
		return true
	default:
		return false
	}
}

func writePHPJSONValue(buffer *bytes.Buffer, value *jsonmeta.Value) error {
	switch value.Kind() {
	case jsonmeta.KindNull:
		buffer.WriteString("null")
	case jsonmeta.KindBoolean:
		boolean, _ := value.BooleanValue()
		if boolean {
			buffer.WriteString("true")
		} else {
			buffer.WriteString("false")
		}
	case jsonmeta.KindNumber:
		number, _ := value.NumberValue()
		encoded, err := encodePHPJSONNumber(number.String())
		if err != nil {
			return err
		}
		buffer.WriteString(encoded)
	case jsonmeta.KindString:
		text, _ := value.StringValue()
		writePHPJSONString(buffer, text)
	case jsonmeta.KindObject:
		buffer.WriteByte('{')
		for index, member := range value.Members() {
			if index > 0 {
				buffer.WriteByte(',')
			}
			writePHPJSONString(buffer, member.Name)
			buffer.WriteByte(':')
			if err := writePHPJSONValue(buffer, member.Value); err != nil {
				return err
			}
		}
		buffer.WriteByte('}')
	case jsonmeta.KindArray:
		buffer.WriteByte('[')
		for index, element := range value.Elements() {
			if index > 0 {
				buffer.WriteByte(',')
			}
			if err := writePHPJSONValue(buffer, element); err != nil {
				return err
			}
		}
		buffer.WriteByte(']')
	default:
		return fmt.Errorf("unsupported JSON value kind %d", value.Kind())
	}

	return nil
}

func encodePHPJSONNumber(number string) (string, error) {
	if !strings.ContainsAny(number, ".eE") {
		if integer, err := strconv.ParseInt(number, 10, 64); err == nil {
			return strconv.FormatInt(integer, 10), nil
		}
	}

	float, err := strconv.ParseFloat(number, 64)
	if err != nil {
		return "", fmt.Errorf("parse JSON number: %w", err)
	}

	return encodePHPJSONFloat(float)
}

func encodePHPJSONFloat(value float64) (string, error) {
	scientific := strconv.FormatFloat(value, 'e', -1, 64)
	exponentOffset := strings.LastIndexByte(scientific, 'e')
	if exponentOffset < 0 {
		return "", fmt.Errorf("format JSON number %q", scientific)
	}

	mantissa := scientific[:exponentOffset]
	exponent, err := strconv.Atoi(scientific[exponentOffset+1:])
	if err != nil {
		return "", fmt.Errorf("parse JSON number exponent: %w", err)
	}

	sign := ""
	if strings.HasPrefix(mantissa, "-") {
		sign = "-"
		mantissa = strings.TrimPrefix(mantissa, "-")
	}
	digits := strings.ReplaceAll(mantissa, ".", "")
	decimalPoint := exponent + 1

	var encoded strings.Builder
	encoded.WriteString(sign)
	switch {
	case decimalPoint > 17 || decimalPoint < -3:
		encoded.WriteByte(digits[0])
		encoded.WriteByte('.')
		if len(digits) == 1 {
			encoded.WriteByte('0')
		} else {
			encoded.WriteString(digits[1:])
		}
		encoded.WriteByte('e')
		exponent = decimalPoint - 1
		if exponent >= 0 {
			encoded.WriteByte('+')
		}
		encoded.WriteString(strconv.Itoa(exponent))
	case decimalPoint <= 0:
		encoded.WriteString("0.")
		encoded.WriteString(strings.Repeat("0", -decimalPoint))
		encoded.WriteString(digits)
	default:
		if decimalPoint >= len(digits) {
			encoded.WriteString(digits)
			encoded.WriteString(strings.Repeat("0", decimalPoint-len(digits)))
		} else {
			encoded.WriteString(digits[:decimalPoint])
			encoded.WriteByte('.')
			encoded.WriteString(digits[decimalPoint:])
		}
	}

	return encoded.String(), nil
}

func writePHPJSONString(buffer *bytes.Buffer, value string) {
	buffer.WriteByte('"')
	for _, character := range value {
		switch character {
		case '"':
			buffer.WriteString(`\"`)
		case '\\':
			buffer.WriteString(`\\`)
		case '/':
			buffer.WriteString(`\/`)
		case '\b':
			buffer.WriteString(`\b`)
		case '\f':
			buffer.WriteString(`\f`)
		case '\n':
			buffer.WriteString(`\n`)
		case '\r':
			buffer.WriteString(`\r`)
		case '\t':
			buffer.WriteString(`\t`)
		default:
			switch {
			case character < 0x20:
				fmt.Fprintf(buffer, `\u%04x`, character)
			case character <= 0x7f:
				buffer.WriteRune(character)
			case character <= 0xffff:
				fmt.Fprintf(buffer, `\u%04x`, character)
			default:
				high, low := utf16.EncodeRune(character)
				fmt.Fprintf(buffer, `\u%04x\u%04x`, high, low)
			}
		}
	}
	buffer.WriteByte('"')
}
