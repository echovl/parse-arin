package main

type Remark struct {
	Title       string   `json:"title"`
	Description []string `json:"description"`
}

type Event struct {
	Action string `json:"eventAction"`
	Date   string `json:"eventDate"`
}

type Entity struct {
	VcardArray []any `json:"vcardArray"`
}

type RawDocument struct {
	StartAddress               string   `json:"startAddress"`
	EndAddress                 string   `json:"endAddress"`
	Name                       string   `json:"name"`
	Remarks                    []Remark `json:"remarks"`
	Events                     []Event  `json:"events"`
	Entities                   []Entity `json:"entities"`
	Status                     []string `json:"status"`
	ObjectClassName            string   `json:"objectClassName"`
	ArinOriginas0Originautnums []int    `json:"arin_originas0_originautnums"`
}

type ParsedDocument struct {
	CIDR         string   `json:"cidr"`
	NetName      string   `json:"netname"`
	ASN          int      `json:"asn"`
	Remarks      string   `json:"remarks"`
	Type         string   `json:"type"`
	Countries    []string `json:"countries"`
	Country      string   `json:"country"`
	LastModified string   `json:"last-modified"`
	Source       string   `json:"source"`
}
