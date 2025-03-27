package utils

import (
	"fmt"

	ethcommon "github.com/ethereum/go-ethereum/common"
)

func ParseAddress(addr string) (ethcommon.Address, error) {
	bytes, err := ethcommon.ParseHexOrString(addr)
	if err != nil {
		return ethcommon.Address{}, err
	}
	if len(bytes) != ethcommon.AddressLength {
		return ethcommon.Address{}, fmt.Errorf(
			"invalid address length: want %d, got %d", ethcommon.AddressLength, len(bytes),
		)
	}
	res := ethcommon.Address{}
	copy(res[:], bytes)
	return res, nil
}
