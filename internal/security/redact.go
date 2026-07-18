package security

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

const Redacted = "[REDACTED]"

var sensitiveHeaderPattern = regexp.MustCompile(
	`(?i)\b(authorization|proxy-authorization|cookie|set-cookie):[^\r\n]*`,
)
var embeddedURLPattern = regexp.MustCompile(`https?://[^\s"'<>]+`)

type Redactor struct {
	secrets []string
}

func NewRedactor(secrets ...string) Redactor {
	unique := make(map[string]struct{}, len(secrets))
	filtered := make([]string, 0, len(secrets))
	for _, secret := range secrets {
		if secret == "" || secret == Redacted {
			continue
		}
		if _, exists := unique[secret]; exists {
			continue
		}
		unique[secret] = struct{}{}
		filtered = append(filtered, secret)
	}
	sort.Slice(filtered, func(left int, right int) bool {
		return len(filtered[left]) > len(filtered[right])
	})

	return Redactor{secrets: filtered}
}

func NewEnvironmentRedactor(environment []string) Redactor {
	var secrets []string
	for _, entry := range environment {
		separator := strings.Index(entry, "=")
		if separator <= 0 {
			continue
		}
		name := strings.TrimSpace(entry[:separator])
		if !sensitiveName(name) {
			continue
		}
		value := entry[separator+1:]
		if value == "" {
			continue
		}
		secrets = append(secrets, value)

		var structured any
		if json.Unmarshal([]byte(value), &structured) == nil {
			collectStringValues(structured, &secrets)
		}
	}

	return NewRedactor(secrets...)
}

func (redactor Redactor) Value(value any) (any, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode value for redaction: %w", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()

	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode value for redaction: %w", err)
	}

	return redactor.redactValue("", decoded), nil
}

func (redactor Redactor) Marshal(value any) ([]byte, error) {
	redacted := redactor.redactReflect(reflect.ValueOf(value), "")
	if !redacted.IsValid() {
		return json.Marshal(nil)
	}
	encoded, err := json.Marshal(redacted.Interface())
	if err != nil {
		return nil, fmt.Errorf("encode redacted value: %w", err)
	}

	return encoded, nil
}

func (redactor Redactor) Text(value string) string {
	redacted := redactEnvironmentAssignment(value)
	if redacted != value {
		return redacted
	}

	redacted = embeddedURLPattern.ReplaceAllStringFunc(redacted, redactURL)
	for _, secret := range redactor.secrets {
		redacted = strings.ReplaceAll(redacted, secret, Redacted)
	}
	redacted = sensitiveHeaderPattern.ReplaceAllStringFunc(
		redacted,
		func(header string) string {
			separator := strings.Index(header, ":")
			if separator < 0 {
				return Redacted
			}
			prefix := header[:separator+1]
			value := strings.TrimSpace(header[separator+1:])
			scheme := ""
			for _, candidate := range []string{"Basic", "Bearer"} {
				if strings.HasPrefix(strings.ToLower(value), strings.ToLower(candidate)+" ") {
					scheme = candidate + " "
					break
				}
			}

			return prefix + " " + scheme + Redacted
		},
	)

	return redacted
}

func (redactor Redactor) redactValue(key string, value any) any {
	if sensitiveName(key) {
		return Redacted
	}

	switch typed := value.(type) {
	case map[string]any:
		redacted := make(map[string]any, len(typed))
		pairIsSensitive := sensitivePair(typed)
		for childKey, childValue := range typed {
			if pairIsSensitive && valueName(childKey) {
				redacted[childKey] = Redacted
				continue
			}
			redacted[childKey] = redactor.redactValue(childKey, childValue)
		}

		return redacted
	case []any:
		redacted := make([]any, len(typed))
		for index, item := range typed {
			redacted[index] = redactor.redactValue("", item)
		}

		return redacted
	case string:
		return redactor.Text(typed)
	default:
		return value
	}
}

func (redactor Redactor) redactReflect(value reflect.Value, key string) reflect.Value {
	if !value.IsValid() {
		return value
	}
	if sensitiveName(key) {
		return redactedReflectValue(value.Type())
	}

	switch value.Kind() {
	case reflect.Interface:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		redacted := redactor.redactReflect(value.Elem(), key)
		result := reflect.New(value.Type()).Elem()
		result.Set(redacted)

		return result
	case reflect.Pointer:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		result := reflect.New(value.Type().Elem())
		result.Elem().Set(redactor.redactReflect(value.Elem(), key))

		return result
	case reflect.Struct:
		result := reflect.New(value.Type()).Elem()
		pairIsSensitive := reflectSensitivePair(value)
		for index := 0; index < value.NumField(); index++ {
			field := value.Type().Field(index)
			if !field.IsExported() {
				continue
			}
			fieldKey := jsonFieldName(field)
			if pairIsSensitive && valueName(fieldKey) {
				result.Field(index).Set(redactedReflectValue(field.Type))
				continue
			}
			result.Field(index).Set(redactor.redactReflect(value.Field(index), fieldKey))
		}

		return result
	case reflect.Map:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		result := reflect.MakeMapWithSize(value.Type(), value.Len())
		pairIsSensitive := reflectMapSensitivePair(value)
		iterator := value.MapRange()
		for iterator.Next() {
			mapKey := iterator.Key()
			childKey := ""
			if mapKey.Kind() == reflect.String {
				childKey = mapKey.String()
			}
			childValue := redactor.redactReflect(iterator.Value(), childKey)
			if pairIsSensitive && valueName(childKey) {
				childValue = redactedReflectValue(iterator.Value().Type())
			}
			result.SetMapIndex(mapKey, childValue)
		}

		return result
	case reflect.Slice:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		result := reflect.MakeSlice(value.Type(), value.Len(), value.Len())
		for index := 0; index < value.Len(); index++ {
			result.Index(index).Set(redactor.redactReflect(value.Index(index), ""))
		}

		return result
	case reflect.Array:
		result := reflect.New(value.Type()).Elem()
		for index := 0; index < value.Len(); index++ {
			result.Index(index).Set(redactor.redactReflect(value.Index(index), ""))
		}

		return result
	case reflect.String:
		result := reflect.New(value.Type()).Elem()
		result.SetString(redactor.Text(value.String()))

		return result
	default:
		return value
	}
}

func redactedReflectValue(valueType reflect.Type) reflect.Value {
	switch valueType.Kind() {
	case reflect.Interface:
		result := reflect.New(valueType).Elem()
		marker := reflect.ValueOf(Redacted)
		if marker.Type().AssignableTo(valueType) ||
			marker.Type().Implements(valueType) {
			result.Set(marker)
		}

		return result
	case reflect.String:
		result := reflect.New(valueType).Elem()
		result.SetString(Redacted)

		return result
	case reflect.Pointer:
		result := reflect.New(valueType.Elem())
		result.Elem().Set(redactedReflectValue(valueType.Elem()))

		return result
	default:
		return reflect.Zero(valueType)
	}
}

func jsonFieldName(field reflect.StructField) string {
	name := field.Tag.Get("json")
	if separator := strings.Index(name, ","); separator >= 0 {
		name = name[:separator]
	}
	if name == "" {
		return field.Name
	}

	return name
}

func sensitiveName(name string) bool {
	normalized := strings.ToLower(strings.TrimSpace(name))
	normalized = strings.NewReplacer("-", "_", ".", "_", " ", "_").Replace(normalized)
	for _, fragment := range []string{
		"authorization",
		"access_key",
		"api_key",
		"auth",
		"client_secret",
		"composer_auth",
		"cookie",
		"credential",
		"password",
		"passwd",
		"private_key",
		"signature",
		"sig",
		"secret",
		"token",
	} {
		if normalized == fragment ||
			strings.HasSuffix(normalized, "_"+fragment) ||
			strings.HasPrefix(normalized, fragment+"_") {
			return true
		}
	}

	return false
}

func sensitivePair(value map[string]any) bool {
	for key, child := range value {
		if !nameName(key) {
			continue
		}
		name, ok := child.(string)

		return ok && sensitiveName(name)
	}

	return false
}

func reflectSensitivePair(value reflect.Value) bool {
	for index := 0; index < value.NumField(); index++ {
		field := value.Type().Field(index)
		if !field.IsExported() || !nameName(jsonFieldName(field)) {
			continue
		}
		fieldValue := value.Field(index)
		if fieldValue.Kind() == reflect.String {
			return sensitiveName(fieldValue.String())
		}
	}

	return false
}

func reflectMapSensitivePair(value reflect.Value) bool {
	if value.Type().Key().Kind() != reflect.String {
		return false
	}
	iterator := value.MapRange()
	for iterator.Next() {
		if !nameName(iterator.Key().String()) {
			continue
		}
		child := iterator.Value()
		if child.Kind() == reflect.Interface && !child.IsNil() {
			child = child.Elem()
		}
		if child.Kind() == reflect.String {
			return sensitiveName(child.String())
		}
	}

	return false
}

func nameName(name string) bool {
	return strings.EqualFold(strings.TrimSpace(name), "name")
}

func valueName(name string) bool {
	return strings.EqualFold(strings.TrimSpace(name), "value")
}

func redactEnvironmentAssignment(value string) string {
	separator := strings.Index(value, "=")
	if separator <= 0 {
		return value
	}
	name := strings.TrimSpace(value[:separator])
	if !sensitiveName(name) {
		return value
	}

	return value[:separator+1] + Redacted
}

func redactURL(value string) string {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return value
	}
	if parsed.User != nil {
		parsed.User = url.User(Redacted)
	}
	query := parsed.Query()
	for name := range query {
		if sensitiveName(name) {
			query.Set(name, Redacted)
		}
	}
	parsed.RawQuery = query.Encode()

	return parsed.String()
}

func collectStringValues(value any, destination *[]string) {
	switch typed := value.(type) {
	case map[string]any:
		for _, child := range typed {
			collectStringValues(child, destination)
		}
	case []any:
		for _, child := range typed {
			collectStringValues(child, destination)
		}
	case string:
		if typed != "" {
			*destination = append(*destination, typed)
		}
	}
}
