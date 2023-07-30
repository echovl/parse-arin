package main

import (
	"encoding/json"
	"testing"
)

func TestParsing(t *testing.T) {
	expected := `{"cidr":"217.147.184.0/21","netname":"SILVERSTAR-11","asn":26223,"remarks":"Geofeed https://raw.githubusercontent.com/SSC-DevOPS/SSC-Geofeed/main/geofeed.csv","type":"inetnum","countries":["US"],"country":"US","last-modified":"2023-05-22T17:10:08-04:00","source":"ARIN"}`

	documents, err := parseFile("./test_silverstar.json")
	if err != nil {
		t.Fatal(err)
	}

	got, err := json.Marshal(documents[0])
	if err != nil {
		t.Fatal(err)
	}

	if expected != string(got) {
		t.Fatalf("Invalid parsed manifest:\n\tgot %s\n\twant: %s", string(got), expected)
	}
}
