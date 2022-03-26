package email

import (
	"encoding/base64"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/util/avutil"
)

type Cursor struct {
	LastEvaluatedKey map[string]types.AttributeValue `json:"lastEvaluatedKey"`
}

func (c Cursor) MarshalJSON() ([]byte, error) {
	if c.LastEvaluatedKey == nil || len(c.LastEvaluatedKey) == 0 {
		return []byte{'"', '"'}, nil
	}

	av := &types.AttributeValueMemberM{
		Value: c.LastEvaluatedKey,
	}
	src := avutil.EncodeAttributeValue(av)

	encoded := []byte{'"'}
	dst := make([]byte, base64.URLEncoding.EncodedLen(len(src)))
	base64.URLEncoding.Encode(dst, src)
	encoded = append(encoded, dst...)
	encoded = append(encoded, '"')

	return encoded, nil
}

func (c *Cursor) UnmarshalJSON(data []byte) error {
	if data[0] != '"' || data[len(data)-1] != '"' { // check both quotation marks
		return errors.New("invalid input to unmarshal")
	}
	data = data[1 : len(data)-1] // remove quotation marks
	if len(data) == 0 {
		return nil
	}

	dst := make([]byte, base64.URLEncoding.DecodedLen(len(data)))
	l, err := base64.URLEncoding.Decode(dst, data)
	if err != nil {
		return err
	}
	dst = dst[:l]

	av, err := avutil.DecodeAttributeValue(dst)
	if err != nil {
		return err
	}
	if m, ok := av.(*types.AttributeValueMemberM); ok {
		c.LastEvaluatedKey = m.Value
	} else {
		return errors.New("invalid input to decode")
	}

	return err
}
