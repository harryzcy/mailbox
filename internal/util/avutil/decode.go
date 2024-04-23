package avutil

import (
	"bytes"
	"encoding/base64"
	"errors"

	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var (
	// ErrDecodeError is returned when decoding failed
	ErrDecodeError = errors.New("decoding error")
)

//gocyclo:ignore
func DecodeAttributeValue(src []byte) (dynamodbTypes.AttributeValue, error) {
	if len(src) < 1 || src[0] != '{' || src[len(src)-1] != '}' || src[1] != '"' {
		return nil, ErrDecodeError
	}

	inner := src[2 : len(src)-1]

	i := bytes.Index(inner, []byte{'"'})
	if inner[i+1] != ':' {
		return nil, ErrDecodeError
	}

	name := string(inner[:i])

	switch name {
	case "B":
		return DecodeAttributeValueB(src)
	case "BOOL":
		return DecodeAttributeValueBOOL(src)
	case "BS":
		return DecodeAttributeValueBS(src)
	case "L":
		return DecodeAttributeValueL(src)
	case "M":
		return DecodeAttributeValueM(src)
	case "N":
		return DecodeAttributeValueN(src)
	case "NS":
		return DecodeAttributeValueNS(src)
	case "NULL":
		return DecodeAttributeValueNULL(src)
	case "S":
		return DecodeAttributeValueS(src)
	case "SS":
		return DecodeAttributeValueSS(src)

	}
	return nil, ErrDecodeError
}

func DecodeAttributeValueB(src []byte) (dynamodbTypes.AttributeValue, error) {
	if !bytes.HasPrefix(src, []byte("{\"B\":\"")) || !bytes.HasSuffix(src, []byte("\"}")) {
		return nil, ErrDecodeError
	}
	value := bytes.TrimPrefix(src, []byte("{\"B\":\""))
	value = bytes.TrimSuffix(value, []byte("\"}"))

	dst := make([]byte, base64.StdEncoding.DecodedLen(len(value)))
	length, err := base64.StdEncoding.Decode(dst, value)
	if err != nil {
		return nil, err
	}
	dst = dst[:length]

	return &dynamodbTypes.AttributeValueMemberB{Value: dst}, nil
}

func DecodeAttributeValueBOOL(src []byte) (dynamodbTypes.AttributeValue, error) {
	if !bytes.HasPrefix(src, []byte("{\"BOOL\":")) || src[len(src)-1] != '}' {
		return nil, ErrDecodeError
	}
	value := bytes.TrimPrefix(src, []byte("{\"BOOL\":"))
	value = value[:len(value)-1]

	var result bool
	switch {
	case bytes.Equal(value, []byte("true")):
		result = true
	case bytes.Equal(value, []byte("false")):
		result = false
	default:
		return nil, ErrDecodeError
	}

	return &dynamodbTypes.AttributeValueMemberBOOL{Value: result}, nil
}

func DecodeAttributeValueBS(src []byte) (dynamodbTypes.AttributeValue, error) {
	if !bytes.HasPrefix(src, []byte("{\"BS\":[")) || !bytes.HasSuffix(src, []byte("]}")) {
		return nil, ErrDecodeError
	}
	value := bytes.TrimPrefix(src, []byte("{\"BS\":["))
	value = bytes.TrimSuffix(value, []byte("]}"))

	sources := bytes.Split(value, []byte{','})
	results := make([][]byte, len(sources))
	for i, b := range sources {
		if len(b) < 1 || b[0] != '"' || b[len(b)-1] != '"' {
			return nil, ErrDecodeError
		}
		b = b[1 : len(b)-1]

		dst := make([]byte, base64.StdEncoding.DecodedLen(len(b)))
		length, err := base64.StdEncoding.Decode(dst, b)
		if err != nil {
			return nil, err
		}
		dst = dst[:length]

		results[i] = dst
	}

	return &dynamodbTypes.AttributeValueMemberBS{Value: results}, nil
}

func DecodeAttributeValueL(src []byte) (dynamodbTypes.AttributeValue, error) {
	if !bytes.HasPrefix(src, []byte("{\"L\":[")) || !bytes.HasSuffix(src, []byte("]}")) {
		return nil, ErrDecodeError
	}
	value := bytes.TrimPrefix(src, []byte("{\"L\":["))
	value = bytes.TrimSuffix(value, []byte("]}"))

	sources := bytes.Split(value, []byte{','})
	results := make([]dynamodbTypes.AttributeValue, len(sources))

	var err error
	for i, src := range sources {
		results[i], err = DecodeAttributeValue(src)
		if err != nil {
			return nil, err
		}
	}

	return &dynamodbTypes.AttributeValueMemberL{Value: results}, nil
}

func DecodeAttributeValueM(src []byte) (dynamodbTypes.AttributeValue, error) {
	if !bytes.HasPrefix(src, []byte("{\"M\":{")) || !bytes.HasSuffix(src, []byte("}}")) {
		return nil, ErrDecodeError
	}
	value := bytes.TrimPrefix(src, []byte("{\"M\":{"))
	value = bytes.TrimSuffix(value, []byte("}}"))

	sources := bytes.Split(value, []byte{','})
	results := make(map[string]dynamodbTypes.AttributeValue, len(sources))

	for _, item := range sources {
		parts := bytes.SplitN(item, []byte{':'}, 2)
		if len(parts) < 2 {
			return nil, ErrDecodeError
		}

		k := parts[0]
		v := parts[1]

		if len(k) < 1 || k[0] != '"' || k[len(k)-1] != '"' {
			return nil, ErrDecodeError
		}
		k = k[1 : len(k)-1]

		var err error
		results[string(k)], err = DecodeAttributeValue(v)
		if err != nil {
			return nil, ErrDecodeError
		}
	}

	return &dynamodbTypes.AttributeValueMemberM{Value: results}, nil
}

func DecodeAttributeValueN(src []byte) (dynamodbTypes.AttributeValue, error) {
	if !bytes.HasPrefix(src, []byte("{\"N\":\"")) || !bytes.HasSuffix(src, []byte("\"}")) {
		return nil, ErrDecodeError
	}
	value := bytes.TrimPrefix(src, []byte("{\"N\":\""))
	value = bytes.TrimSuffix(value, []byte("\"}"))

	return &dynamodbTypes.AttributeValueMemberN{Value: string(value)}, nil
}

func DecodeAttributeValueNS(src []byte) (dynamodbTypes.AttributeValue, error) {
	if !bytes.HasPrefix(src, []byte("{\"NS\":[")) || !bytes.HasSuffix(src, []byte("]}")) {
		return nil, ErrDecodeError
	}
	value := bytes.TrimPrefix(src, []byte("{\"NS\":["))
	value = bytes.TrimSuffix(value, []byte("]}"))

	sources := bytes.Split(value, []byte{','})
	results := make([]string, len(sources))

	for i, item := range sources {
		if len(item) < 1 || item[0] != '"' || item[len(item)-1] != '"' {
			return nil, ErrDecodeError
		}
		item = item[1 : len(item)-1]
		results[i] = string(item)
	}

	return &dynamodbTypes.AttributeValueMemberNS{Value: results}, nil
}

func DecodeAttributeValueNULL(src []byte) (dynamodbTypes.AttributeValue, error) {
	if !bytes.HasPrefix(src, []byte("{\"NULL\":")) || src[len(src)-1] != '}' {
		return nil, ErrDecodeError
	}
	value := bytes.TrimPrefix(src, []byte("{\"NULL\":"))
	value = value[:len(value)-1]
	if string(value) != "true" {
		return nil, ErrDecodeError
	}

	return &dynamodbTypes.AttributeValueMemberNULL{Value: true}, nil
}

func DecodeAttributeValueS(src []byte) (dynamodbTypes.AttributeValue, error) {
	if !bytes.HasPrefix(src, []byte("{\"S\":\"")) || !bytes.HasSuffix(src, []byte("\"}")) {
		return nil, ErrDecodeError
	}
	value := bytes.TrimPrefix(src, []byte("{\"S\":\""))
	value = bytes.TrimSuffix(value, []byte("\"}"))

	return &dynamodbTypes.AttributeValueMemberS{Value: string(value)}, nil
}

func DecodeAttributeValueSS(src []byte) (dynamodbTypes.AttributeValue, error) {
	if !bytes.HasPrefix(src, []byte("{\"SS\":[")) || !bytes.HasSuffix(src, []byte("]}")) {
		return nil, ErrDecodeError
	}
	value := bytes.TrimPrefix(src, []byte("{\"SS\":["))
	value = bytes.TrimSuffix(value, []byte("]}"))

	sources := bytes.Split(value, []byte{','})
	results := make([]string, len(sources))

	for i, item := range sources {
		if len(item) < 1 || item[0] != '"' || item[len(item)-1] != '"' {
			return nil, ErrDecodeError
		}
		item = item[1 : len(item)-1]
		results[i] = string(item)
	}

	return &dynamodbTypes.AttributeValueMemberSS{Value: results}, nil
}
