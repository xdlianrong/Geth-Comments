package utils

import (
)

type Purchase struct {
	PublicKey string `json:"publickey" form:"publickey" query:"publickey"`
	Amount int  `json:"amount" form:"amount" query:"amount"`
}

func CreateCM_v(amount int)  {
	return
}


