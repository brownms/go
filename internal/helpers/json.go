package helpers

import (
	"fmt"
	"math/big"
	"strings"
)

type BigNumberJSON struct {
	int_val   *big.Int
	float_val *big.Float
}

func (b BigNumberJSON) MarshalJSON() ([]byte, error) {
	if b.int_val != nil {
		return []byte(b.int_val.String()), nil
	}
	if b.float_val != nil {
		return []byte(b.float_val.String()), nil
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
		b.float_val = &value
		return nil
	}

	var value big.Int
	_, ok := value.SetString(string_val, 10)
	if !ok {
		return fmt.Errorf("not a valid big integer: %s", bytes)
	}
	b.int_val = &value
	return nil
}
