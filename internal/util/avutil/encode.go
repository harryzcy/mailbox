package avutil

import (
	"encoding/base64"
	"fmt"

	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func EncodeAttributeValue(value dynamodbTypes.AttributeValue) []byte {
	var content []byte

	switch value.(type) {
	case *dynamodbTypes.AttributeValueMemberB:
		content = EncodeAttributeValueB(value)

	case *dynamodbTypes.AttributeValueMemberBOOL:
		content = EncodeAttributeValueBOOL(value)

	case *dynamodbTypes.AttributeValueMemberBS:
		content = EncodeAttributeValueBS(value)

	case *dynamodbTypes.AttributeValueMemberL:
		content = EncodeAttributeValueL(value)

	case *dynamodbTypes.AttributeValueMemberM:
		content = EncodeAttributeValueM(value)

	case *dynamodbTypes.AttributeValueMemberN:
		content = EncodeAttributeValueN(value)

	case *dynamodbTypes.AttributeValueMemberNS:
		content = EncodeAttributeValueNS(value)

	case *dynamodbTypes.AttributeValueMemberNULL:
		content = EncodeAttributeValueNULL(value)

	case *dynamodbTypes.AttributeValueMemberS:
		content = EncodeAttributeValueS(value)

	case *dynamodbTypes.AttributeValueMemberSS:
		content = EncodeAttributeValueSS(value)

	}
	return content
}

func EncodeAttributeValueB(value dynamodbTypes.AttributeValue) []byte {
	encoded := base64.StdEncoding.EncodeToString(value.(*dynamodbTypes.AttributeValueMemberB).Value)
	content := fmt.Sprintf("{\"B\":\"%s\"}", encoded)
	return []byte(content)
}

func EncodeAttributeValueBOOL(value dynamodbTypes.AttributeValue) []byte {
	content := fmt.Sprintf("{\"BOOL\":%t}", value.(*dynamodbTypes.AttributeValueMemberBOOL).Value)
	return []byte(content)
}

func EncodeAttributeValueBS(value dynamodbTypes.AttributeValue) []byte {
	result := []byte("{\"BS\":[")
	for i, item := range value.(*dynamodbTypes.AttributeValueMemberBS).Value {
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

func EncodeAttributeValueL(value dynamodbTypes.AttributeValue) []byte {
	result := []byte("{\"L\":[")
	for _, v := range value.(*dynamodbTypes.AttributeValueMemberL).Value {
		content := EncodeAttributeValue(v)
		result = append(result, content...)
	}
	result = append(result, ']', '}')
	return result
}

func EncodeAttributeValueM(value dynamodbTypes.AttributeValue) []byte {
	result := []byte("{\"M\":{")

	first := true
	for k, v := range value.(*dynamodbTypes.AttributeValueMemberM).Value {
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

func EncodeAttributeValueN(value dynamodbTypes.AttributeValue) []byte {
	content := fmt.Sprintf("{\"N\":\"%s\"}", value.(*dynamodbTypes.AttributeValueMemberN).Value)
	return []byte(content)
}

func EncodeAttributeValueNS(value dynamodbTypes.AttributeValue) []byte {
	result := []byte("{\"NS\":[")
	for i, item := range value.(*dynamodbTypes.AttributeValueMemberNS).Value {
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

func EncodeAttributeValueNULL(value dynamodbTypes.AttributeValue) []byte {
	if value.(*dynamodbTypes.AttributeValueMemberNULL).Value {
		return []byte("{\"NULL\":true}")
	}
	return []byte("{\"NULL\":false}")
}

func EncodeAttributeValueS(value dynamodbTypes.AttributeValue) []byte {
	return []byte("{\"S\":\"" + value.(*dynamodbTypes.AttributeValueMemberS).Value + "\"}")
}

func EncodeAttributeValueSS(value dynamodbTypes.AttributeValue) []byte {
	result := []byte("{\"SS\":[")
	for i, item := range value.(*dynamodbTypes.AttributeValueMemberSS).Value {
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
