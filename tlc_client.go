package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type TLCResponse struct {
	States []string
	Keys   []int64
}

type TLCClient struct {
	ClientAddr string
}

func NewTLCClient(addr string) *TLCClient {
	return &TLCClient{
		ClientAddr: addr,
	}
}

func (c *TLCClient) SendTrace(trace *List[*Event]) ([]State, error) {
	trace.Append(&Event{Reset: true})
	data, err := json.Marshal(trace.AsList())
	if err != nil {
		return []State{}, fmt.Errorf("error marshalling json: %s", err)
	}
	res, err := http.Post("http://"+c.ClientAddr+"/execute", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return []State{}, fmt.Errorf("error sending trace to tlc: %s", err)
	}
	defer res.Body.Close()
	resData, err := io.ReadAll(res.Body)
	if err != nil {
		return []State{}, fmt.Errorf("error reading response from tlc: %s", err)
	}
	tlcResponse := &TLCResponse{}
	if err = json.Unmarshal(resData, tlcResponse); err != nil {
		return []State{}, fmt.Errorf("error parsing tlc response: %s", err)
	}
	result := make([]State, len(tlcResponse.States))
	for i, s := range tlcResponse.States {
		result[i] = State{Repr: s, Key: tlcResponse.Keys[i]}
	}
	return result, nil
}
