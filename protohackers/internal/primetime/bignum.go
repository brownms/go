package primetime

import (
	"fmt"
	"math/big"
	"strings"
)

type BigNumberJSON struct {
	IntVal   *big.Int
	FloatVal *big.Float
}

func (b BigNumberJSON) MarshalJSON() ([]byte, error) {
	if b.IntVal != nil {
		return []byte(b.IntVal.String()), nil
	}
	if b.FloatVal != nil {
		return []byte(b.FloatVal.String()), nil
	}
	return []byte(""), nil
}

func (b *BigNumberJSON) UnmarshalJSON(bytes []byte) error {
	string_val := string(bytes)
	if string_val == "null" {
		return nil
	}
	if strings.Contains(string_val, ".") {
		// JSON numbers can be floating point
		var value big.Float
		_, ok := value.SetString(string_val)
		if !ok {
			return fmt.Errorf("not a valid big float: %s", bytes)
		}
		b.FloatVal = &value
		return nil
	}

	var value big.Int
	_, ok := value.SetString(string_val, 10)
	if !ok {
		return fmt.Errorf("not a valid big integer: %s", bytes)
	}
	b.IntVal = &value
	return nil
}
