package core

import (
	"database/sql/driver"
	"encoding/json"
)

type Volume struct {
	Input  int64 `json:"input"`
	Output int64 `json:"output"`
}

func (v Volume) MarshalJSON() ([]byte, error) {
	type volume Volume
	return json.Marshal(struct {
		volume
		Balance int64 `json:"balance"`
	}{
		volume:  volume(v),
		Balance: v.Input - v.Output,
	})
}

func (v Volume) Balance() int64 {
	return v.Input - v.Output
}

type Balances map[string]int64
type Volumes map[string]Volume

func (v Volumes) Balances() Balances {
	balances := Balances{}
	for asset, vv := range v {
		balances[asset] = vv.Input - vv.Output
	}
	return balances
}

type AggregatedVolumes map[string]Volumes

type AggregatedBalances struct {
	Aggregated map[string]int64            `json:"aggregated"`
	Accounts   map[string]map[string]int64 `json:"accounts"`
}

// Scan - Implement the database/sql scanner interface
func (m *AggregatedVolumes) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	v, err := driver.String.ConvertValue(value)
	if err != nil {
		return err
	}

	*m = AggregatedVolumes{}
	switch vv := v.(type) {
	case []uint8:
		return json.Unmarshal(vv, m)
	case string:
		return json.Unmarshal([]byte(vv), m)
	default:
		panic("not handled type")
	}
}
