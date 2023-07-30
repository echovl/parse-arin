package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/biter777/countries"
	"github.com/mattn/go-isatty"
	"github.com/mikioh/ipaddr"
)

const lastChangedAction = "last changed"
const ipNetworkClass = "ip network"
const ipv4NetType = "inetnum"
const ipv6NetType = "inet6num"
const defaultCountryCode = "ZZ"

func getCountryCodes(doc *RawDocument) []string {
	var codes []string
	for _, ent := range doc.Entities {
		for _, vcard := range ent.VcardArray {
			vcard, ok := vcard.([]any)
			if !ok {
				continue
			}

			for _, vcardInfo := range vcard {
				vcardInfo, ok := vcardInfo.([]any)
				if !ok {
					continue
				}

				if len(vcardInfo) == 0 {
					continue
				}

				kind, ok := vcardInfo[0].(string)
				if !ok || kind != "adr" {
					continue
				}

				infoMap, ok := vcardInfo[1].(map[string]any)
				if !ok {
					continue
				}

				label, ok := infoMap["label"]
				if !ok {
					continue
				}

				addr, ok := label.(string)
				if !ok {
					continue
				}

				addrLines := strings.Split(addr, "\n")
				if len(addrLines) == 0 {
					continue
				}

				countryCode := countries.ByName(addrLines[len(addrLines)-1])
				if countryCode == countries.Unknown {
					continue
				}

				codes = append(codes, countryCode.Alpha2())
			}
		}
	}

	return codes
}

func parseFile(filename string) ([]ParsedDocument, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var rawDoc RawDocument
	var documents []ParsedDocument
	var remarks string
	var netType string
	var asn int
	var lastModified string
	var countries []string
	var mainCountry string

	if err := json.NewDecoder(f).Decode(&rawDoc); err != nil {
		return nil, err
	}

	if rawDoc.ObjectClassName != ipNetworkClass {
		return documents, nil
	}

	ipStart := net.ParseIP(rawDoc.StartAddress)
	ipEnd := net.ParseIP(rawDoc.EndAddress)

	cidrs := ipaddr.Summarize(ipStart, ipEnd)

	for i, remark := range rawDoc.Remarks {
		remarks += strings.Join(remark.Description, "\n")
		if i < len(rawDoc.Remarks)-1 {
			remarks += "\n"
		}
	}

	if ip := ipStart.To4(); ip != nil {
		netType = ipv4NetType
	} else if ip := ipStart.To16(); ip != nil {
		netType = ipv6NetType
	} else {
		return nil, errors.New("Invalid IP version")
	}

	for _, ev := range rawDoc.Events {
		if ev.Action == lastChangedAction {
			lastModified = ev.Date
		}
	}

	countryCodes := getCountryCodes(&rawDoc)

CountryLoop:
	for _, cc := range countryCodes {
		// Filter out duplicates, using an array is faster vs a map for small arrays
		for _, c := range countries {
			if c == cc {
				continue CountryLoop
			}
		}
		countries = append(countries, cc)
	}

	if len(countries) > 0 {
		mainCountry = countries[0]
	} else {
		mainCountry = defaultCountryCode
	}

	if len(rawDoc.ArinOriginas0Originautnums) > 0 {
		asn = rawDoc.ArinOriginas0Originautnums[0]
	}

	for _, cidr := range cidrs {
		documents = append(documents, ParsedDocument{
			CIDR:         cidr.String(),
			NetName:      rawDoc.Name,
			ASN:          asn,
			Remarks:      remarks,
			Type:         netType,
			Countries:    countries,
			Country:      mainCountry,
			LastModified: lastModified,
			Source:       "ARIN",
		})
	}

	return documents, nil
}

func main() {
	testFilename := flag.String("test-file", "", "Optional test file to parse")
	targetDir := flag.String("target-dir", "./.cache/arin-rir/", "Target directory with the files to parse")
	flag.Parse()

	if *testFilename != "" {
		testFileInfo, err := os.Stat(*testFilename)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("%s file does not exist\n", *testFilename)
				os.Exit(1)
			}
		}

		if testFileInfo.IsDir() {
			fmt.Printf("%s is a directory\n", *testFilename)
			os.Exit(1)
		}
	}

	targetInfo, err := os.Stat(*targetDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("%s directory does not exist\n", *targetDir)
			os.Exit(1)
		}
	}

	if !targetInfo.IsDir() {
		fmt.Printf("%s is not a directory\n", *targetDir)
		os.Exit(1)
	}

	if *testFilename != "" {
		testDoc, err := parseFile(*testFilename)
		if err != nil {
			panic(err)
		}

		for _, d := range testDoc {
			doc, _ := json.Marshal(d)
			fmt.Println(string(doc))
		}
	}

	isTerminal := isatty.IsTerminal(os.Stdout.Fd())

	if isTerminal {
		var execute string
		fmt.Println("WARNING: This script will output a large amount of data. Continue? [y/N]")
		fmt.Scanf("%s", &execute)
		if execute != "y" {
			return
		}
	}

	// Number of workers
	poolSize := runtime.NumCPU()

	// Files to parse
	files := make(chan string)

	abort := make(chan struct{})

	// Parsed documents in JSON format
	documents := []string{}

	var mu sync.Mutex
	for poolIdx := 0; poolIdx < poolSize; poolIdx++ {
		go func(index int, files <-chan string, abort chan<- struct{}) {
			for filename := range files {
				fmt.Printf("Worker #%d: Processing file %s\n", index, filename)
				rawDocs, err := parseFile(filename)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error parsing '%s'", filename)
					fmt.Fprintf(os.Stderr, "The error was: %s", err)
					abort <- struct{}{}
				}

				batch := make([]string, len(rawDocs))
				for i, d := range rawDocs {
					doc, _ := json.Marshal(d)
					batch[i] = string(doc)

				}

				mu.Lock()
				for _, b := range batch {
					documents = append(documents, b)
				}
				mu.Unlock()
			}
		}(poolIdx, files, abort)
	}

	finished := make(chan struct{})

	go func(files chan<- string, abort <-chan struct{}, finished chan<- struct{}) {
		filepath.WalkDir(*targetDir, func(path string, d fs.DirEntry, err error) error {
			if d.IsDir() {
				return nil
			}

			select {
			case files <- path:
				return nil
			case <-abort:
				close(files)
				return filepath.SkipAll
			}
		})
		finished <- struct{}{}
	}(files, abort, finished)

	<-finished
	for _, doc := range documents {
		fmt.Println(doc)
		if isTerminal {
			time.Sleep(100 * time.Millisecond)
		}
	}
}
