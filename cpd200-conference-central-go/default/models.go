package main

import (
	"fmt"
	"strings"
)

type Profile struct {
	//Profile -- User profile object
	DisplayName string	`json:"displayName"`
	MainEmail string	`json:"mainEmail"`
	TeeShirtSize string	`json:"teeShirtSize"`
}

type ProfileMiniForm struct {
	//ProfileMiniForm -- update Profile form message
	DisplayName string	`json:"displayName"`
	TeeShirtSize TeeShirtSize	`json:"teeShirtSize"`
}

type ProfileForm struct {
	//ProfileForm -- Profile outbound form message
	DisplayName  string	`json:"displayName"`
	MainEmail string	`json:"mainEmail"`
	TeeShirtSize TeeShirtSize	`json:"teeShirtSize"`
}

type TeeShirtSize int

const (
	//TeeShirtSize -- t-shirt size enumeration value
	NOT_SPECIFIED TeeShirtSize = iota
	XS_M
	XS_W
	S_M
	S_W
	M_M
	M_W
	L_M
	L_W
	XL_M
	XL_W
	XXL_M
	XXL_W
	XXXL_M
	XXXL_W
)

var teeShirtSizeEnumTypeNames = map[TeeShirtSize]string{
	NOT_SPECIFIED: "NOT_SPECIFIED",
	XS_M: "XS_M",
	XS_W: "XS_W",
	S_M: "S_M",
	S_W: "S_W",
	M_M: "M_M",
	M_W: "M_W",
	L_M: "L_M",
	L_W: "L_W",
	XL_M: "XL_M",
	XL_W: "XL_W",
	XXL_M: "XXL_M",
	XXL_W: "XXL_W",
	XXXL_M: "XXXL_M",
	XXXL_W: "XXXL_W",
}

func (m TeeShirtSize) MarshalJSON() ([]byte, error){
	str := fmt.Sprintf("\"%s\"", TeeShirtSizeToStringEnum(m))
    return []byte(str), nil
}

func (m *TeeShirtSize) UnmarshalJSON(value []byte) error {
	str := strings.Replace(string(value), "\"", "", -1)
	*m = StringEnumToTeeShirtSize(str)
	return nil
}

func StringEnumToTeeShirtSize(enum string) TeeShirtSize {
	for i, r := range teeShirtSizeEnumTypeNames {
		if enum == r {
			return TeeShirtSize(i)
		}
	}
	return NOT_SPECIFIED
}

func TeeShirtSizeToStringEnum(key TeeShirtSize) string {
	if val, ok := teeShirtSizeEnumTypeNames[key]; ok {
		return val
	}
	return teeShirtSizeEnumTypeNames[NOT_SPECIFIED]
}
