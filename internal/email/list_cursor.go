package email

import (
	"bytes"
	"encoding/base64"
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
	builder.WriteString(c.QueryInfo.Type)
	builder.WriteByte(',')
	builder.WriteString(c.QueryInfo.Year)
	builder.WriteByte(',')
	builder.WriteString(c.QueryInfo.Month)
	builder.WriteByte(',')
	builder.WriteString(c.QueryInfo.Order)
	builder.WriteByte(',')

	data, err := c.LastEvaluatedKey.Encode()
	if err != nil {
		return nil, err
	}
	builder.Write(data)

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
	// dst should be in the format of "type,year,month,order,lastEvaluatedKey"
	// we need to extract the lastEvaluatedKey

	parts := bytes.Split(dst, []byte(","))
	if len(parts) != 5 {
		return ErrInvalidInputToUnmarshal
	}
	c.QueryInfo.Type = string(parts[0])
	c.QueryInfo.Year = string(parts[1])
	c.QueryInfo.Month = string(parts[2])
	c.QueryInfo.Order = string(parts[3])

	err = c.LastEvaluatedKey.Decode(parts[4])
	if err != nil {
		return err
	}

	return nil
}

type LastEvaluatedKey map[string]types.AttributeValue

func (k LastEvaluatedKey) Encode() ([]byte, error) {
	if len(k) == 0 {
		return []byte{}, nil
	}

	av := &types.AttributeValueMemberM{
		Value: k,
	}

	encoded := avutil.EncodeAttributeValue(av)
	return encoded, nil
}

func (k *LastEvaluatedKey) Decode(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	av, err := avutil.DecodeAttributeValue(data)
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
