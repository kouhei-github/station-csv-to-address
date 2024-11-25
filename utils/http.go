package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Request interface {
	StationToAddress(url string) (*StationResponse, error)
	PostalCodeToPref(url string) (*AddressResponse, error)
}

type request struct {
	client *http.Client
}

func NewRequest() Request {
	return &request{client: &http.Client{}}
}

func (r *request) get(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return nil, err
	}
	res, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// Station represents a single station's information
type Station struct {
	Name       string  `json:"name"`
	Prefecture string  `json:"prefecture"`
	Line       string  `json:"line"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Postal     string  `json:"postal"`
	Prev       string  `json:"prev"`
	Next       string  `json:"next"`
}

type Response struct {
	Stations []Station `json:"station"`
}

// StationResponse represents the top-level response structure
type StationResponse struct {
	Response Response `json:"response"`
}

func (r *Response) GetLine(line string) (*Station, error) {
	for _, station := range r.Stations {
		if station.Line == line {
			return &station, nil
		}
		if station.Prefecture == line {
			return &station, nil
		}
	}
	return nil, nil
}

func (r *request) StationToAddress(url string) (*StationResponse, error) {
	data, err := r.get(url)
	if err != nil {
		return nil, err
	}
	var res StationResponse
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// AddressResponse は郵便番号検索の結果を表す構造体
type AddressResponse struct {
	PostalCode string    `json:"postalCode"`
	Addresses  []Address `json:"addresses"`
}

// Address は住所情報を表す構造体
type Address struct {
	PrefectureCode string        `json:"prefectureCode"`
	Japanese       AddressDetail `json:"ja"`
	Kana           AddressDetail `json:"kana"`
	English        AddressDetail `json:"en"`
}

// AddressDetail は各言語での住所の詳細を表す構造体
type AddressDetail struct {
	Prefecture string `json:"prefecture"`
	Address1   string `json:"address1"`
	Address2   string `json:"address2"`
	Address3   string `json:"address3"`
	Address4   string `json:"address4"`
}

// GetFormattedJapaneseAddress フォーマットされた住所文字列を取得するヘルパー関数を作成する場合：
func (response *AddressResponse) GetFormattedJapaneseAddress() string {
	if len(response.Addresses) == 0 {
		return ""
	}

	japanese := response.Addresses[0].Japanese
	return fmt.Sprintf("%s%s%s",
		japanese.Prefecture,
		japanese.Address1,
		japanese.Address2,
	)
}

func (r *request) PostalCodeToPref(url string) (*AddressResponse, error) {
	data, err := r.get(url)
	if err != nil {
		return nil, err
	}
	var res AddressResponse
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	return &res, nil
}
