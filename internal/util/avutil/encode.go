package avutil

import (
	"encoding/base64"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func EncodeAttributeValue(value types.AttributeValue) []byte {
	var content []byte

	switch value.(type) {
	case *types.AttributeValueMemberB:
		content = EncodeAttributeValueB(value)

	case *types.AttributeValueMemberBOOL:
		content = EncodeAttributeValueBOOL(value)

	case *types.AttributeValueMemberBS:
		content = EncodeAttributeValueBS(value)

	case *types.AttributeValueMemberL:
		content = EncodeAttributeValueL(value)

	case *types.AttributeValueMemberM:
		content = EncodeAttributeValueM(value)

	case *types.AttributeValueMemberN:
		content = EncodeAttributeValueN(value)

	case *types.AttributeValueMemberNS:
		content = EncodeAttributeValueNS(value)

	case *types.AttributeValueMemberNULL:
		content = EncodeAttributeValueNULL(value)

	case *types.AttributeValueMemberS:
		content = EncodeAttributeValueS(value)

	case *types.AttributeValueMemberSS:
		content = EncodeAttributeValueSS(value)

	}
	return content
}

func EncodeAttributeValueB(value types.AttributeValue) []byte {
	encoded := base64.StdEncoding.EncodeToString(value.(*types.AttributeValueMemberB).Value)
	content := fmt.Sprintf("{\"B\":\"%s\"}", encoded)
	return []byte(content)
}

func EncodeAttributeValueBOOL(value types.AttributeValue) []byte {
	content := fmt.Sprintf("{\"BOOL\":%t}", value.(*types.AttributeValueMemberBOOL).Value)
	return []byte(content)
}

func EncodeAttributeValueBS(value types.AttributeValue) []byte {
	result := []byte("{\"BS\":[")
	for i, item := range value.(*types.AttributeValueMemberBS).Value {
		if i != 0 {
			result = append(result, ',')
		}
		result = append(result, '"')
		encoded := base64.StdEncoding.EncodeToString(item)
		result = append(result, encoded...)
		result = append(result, '"')
	}
	result = append(result, []byte("]}")...)
	return result
}

func EncodeAttributeValueL(value types.AttributeValue) []byte {
	result := []byte("{\"L\":[")
	for _, v := range value.(*types.AttributeValueMemberL).Value {
		content := EncodeAttributeValue(v)
		result = append(result, content...)
	}
	result = append(result, ']', '}')
	return result
}

func EncodeAttributeValueM(value types.AttributeValue) []byte {
	result := []byte("{\"M\":{")

	first := true
	for k, v := range value.(*types.AttributeValueMemberM).Value {
		if !first {
			result = append(result, ',')
		} else {
			first = false
		}

		result = append(result, []byte("\""+k+"\":")...)

		content := EncodeAttributeValue(v)
		result = append(result, content...)
	}

	result = append(result, '}', '}')
	return result
}

func EncodeAttributeValueN(value types.AttributeValue) []byte {
	content := fmt.Sprintf("{\"N\":\"%s\"}", value.(*types.AttributeValueMemberN).Value)
	return []byte(content)
}

func EncodeAttributeValueNS(value types.AttributeValue) []byte {
	result := []byte("{\"NS\":[")
	for i, item := range value.(*types.AttributeValueMemberNS).Value {
		if i != 0 {
			result = append(result, ',')
		}
		result = append(result, '"')
		result = append(result, item...)
		result = append(result, '"')
	}
	result = append(result, ']', '}')

	return result
}

func EncodeAttributeValueNULL(value types.AttributeValue) []byte {
	return []byte("{\"NULL\":true}")
}

func EncodeAttributeValueS(value types.AttributeValue) []byte {
	return []byte("{\"S\":\"" + value.(*types.AttributeValueMemberS).Value + "\"}")
}

func EncodeAttributeValueSS(value types.AttributeValue) []byte {
	result := []byte("{\"SS\":[")
	for i, item := range value.(*types.AttributeValueMemberSS).Value {
		if i != 0 {
			result = append(result, ',')
		}
		result = append(result, '"')
		result = append(result, item...)
		result = append(result, '"')
	}
	result = append(result, ']', '}')

	return result
}
