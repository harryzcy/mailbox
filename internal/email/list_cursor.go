package email

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/util/avutil"
)

var (
	ErrInvalidInputToUnmarshal = errors.New("invalid input to unmarshal")
	ErrInvalidInputToDecode    = errors.New("invalid input to decode")
)

type QueryInfo struct {
	Type  string `json:"type"`
	Year  string `json:"year"`
	Month string `json:"month"`
	Order string `json:"order"`
}

type Cursor struct {
	QueryInfo        QueryInfo        `json:"queryInfo"`
	LastEvaluatedKey LastEvaluatedKey `json:"lastEvaluatedKey"`
}

func (c Cursor) MarshalJSON() ([]byte, error) {
	var builder bytes.Buffer
	builder.WriteString("{")
	builder.WriteString(`"queryInfo":`)
	data, err := json.Marshal(c.QueryInfo)
	if err != nil {
		return nil, err
	}
	builder.Write(data)
	builder.WriteString(`,"lastEvaluatedKey":`)
	data, err = json.Marshal(c.LastEvaluatedKey)
	if err != nil {
		return nil, err
	}
	builder.Write(data)
	builder.WriteString("}")

	src := builder.Bytes()

	encoded := createdQuotedBase64Encoding(src)
	return encoded, nil
}

func (c *Cursor) UnmarshalJSON(data []byte) error {
	if len(data) < 2 || data[0] != '"' || data[len(data)-1] != '"' { // check both quotation marks
		return ErrInvalidInputToUnmarshal
	}
	data = data[1 : len(data)-1] // remove quotation marks
	if len(data) == 0 {
		return nil
	}

	return c.Bind(data)
}

func (c *Cursor) BindString(data string) error {
	return c.Bind([]byte(data))
}

func (c *Cursor) Bind(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if c == nil {
		c = &Cursor{}
	}

	dst, err := decodeBase64Encoding(data)
	if err != nil {
		return err
	}
	// dst should be in the format of {"queryInfo":{},"lastEvaluatedKey":{}}
	// we need to extract the lastEvaluatedKey

	dst = bytes.TrimPrefix(dst, []byte("{\"queryInfo\":"))
	dst = bytes.TrimSuffix(dst, []byte("}"))
	parts := bytes.SplitN(dst, []byte(",\"lastEvaluatedKey\":"), 2)
	if len(parts) != 2 {
		return ErrInvalidInputToUnmarshal
	}
	err = json.Unmarshal(parts[0], &c.QueryInfo)
	if err != nil {
		return err
	}
	err = json.Unmarshal(parts[1], &c.LastEvaluatedKey)
	if err != nil {
		return err
	}

	return nil
}

type LastEvaluatedKey map[string]types.AttributeValue

// MarshalJSON allows LastEvaluatedKey to be a Marshaler
func (k LastEvaluatedKey) MarshalJSON() ([]byte, error) {

	if len(k) == 0 {
		return []byte{'"', '"'}, nil
	}

	av := &types.AttributeValueMemberM{
		Value: k,
	}
	src := avutil.EncodeAttributeValue(av)

	encoded := createdQuotedBase64Encoding(src)

	return encoded, nil
}

// UnmarshalJSON allows LastEvaluatedKey to be an Unmarshaler
func (k *LastEvaluatedKey) UnmarshalJSON(data []byte) error {
	if len(data) < 2 || data[0] != '"' || data[len(data)-1] != '"' { // check both quotation marks
		return ErrInvalidInputToUnmarshal
	}
	data = data[1 : len(data)-1] // remove quotation marks
	if len(data) == 0 {
		return nil
	}

	return k.Bind(data)
}

// BindString binds a string input to LastEvaluatedKey
func (k *LastEvaluatedKey) BindString(data string) error {
	return k.Bind([]byte(data))
}

// BindString binds a byte array input to LastEvaluatedKey
func (k *LastEvaluatedKey) Bind(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	dst, err := decodeBase64Encoding(data)
	if err != nil {
		return err
	}

	av, err := avutil.DecodeAttributeValue(dst)
	if err != nil {
		return err
	}
	if m, ok := av.(*types.AttributeValueMemberM); ok {
		*k = m.Value
	} else {
		return ErrInvalidInputToDecode
	}

	return err
}

func createdQuotedBase64Encoding(src []byte) []byte {
	encoded := []byte{'"'}
	dst := make([]byte, base64.URLEncoding.EncodedLen(len(src)))
	base64.URLEncoding.Encode(dst, src)
	encoded = append(encoded, dst...)
	encoded = append(encoded, '"')

	return encoded
}

func decodeBase64Encoding(data []byte) ([]byte, error) {
	dst := make([]byte, base64.URLEncoding.DecodedLen(len(data)))
	l, err := base64.URLEncoding.Decode(dst, data)
	if err != nil {
		return nil, err
	}
	dst = dst[:l]
	return dst, nil
}
