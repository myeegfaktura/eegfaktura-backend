package api

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/eegfaktura/eegfaktura-backend/model"
	"github.com/stretchr/testify/require"
)

func TestMarschaling(t *testing.T) {
	// Tariff numeric fields are plain numbers in JSON (matches prod-image
	// response after parity fix #45). businessNr is null.Int so accepts
	// integer or null.
	jsonStr := `{"id":"","name":"Mein Einspeise Traif","type":"EZP","useVat":false,"baseFee":0,"accountGrossAmount":0,"participantFee":0,"accountNetAmount":0,"billingPeriod":"monthly","businessNr":0,"centPerKWh":12,"discount":0,"freeKWH":0,"vatInPercent":0}`

	var r model.Tariff
	err := json.Unmarshal([]byte(jsonStr), &r)
	require.NoError(t, err)

	fmt.Printf("R: %+v\n", r)
}
