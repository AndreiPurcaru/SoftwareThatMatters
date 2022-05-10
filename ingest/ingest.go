package ingest

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

type OutputVersion struct {
	TimeStamp    time.Time         `json:"timestamp"`
	Dependencies map[string]string `json:"dependencies"`
}

type OutputFormat struct {
	Name     string                   `json:"name"`
	Versions map[string]OutputVersion `json:"versions"`
}

type VersionData struct {
	Version         string            `json:"version"`
	DevDependencies map[string]string `json:"devDependencies"`
	Dependencies    map[string]string `json:"dependencies"`
}

type Doc struct {
	Name     string                 `json:"name"`
	Versions map[string]VersionData `json:"versions"`
	Time     map[string]CreatedTime `json:"time"`
}

type Entry struct {
	Doc Doc `json:"doc"`
}

// Type alias so we can create a custom parser for time since it wasn't parsed correctly natively
type CreatedTime time.Time

// Function required to implement the JSON parser interface for CreatedTime
func (ct *CreatedTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return err
	}
	*ct = CreatedTime(t)
	return nil
}

// Function required to implement the JSON parser interface for CreatedTime
func (ct CreatedTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(ct)
}

// This function forces the JSON Unmarshaler to use the CreatedTime Unmarshaler
func (v *Version) UnmarshalJSON(b []byte) error {
	var dat map[string]interface{}

	if err := json.Unmarshal(b, &dat); err != nil {
		return err
	}
	date_string := "\"" + dat["published_at"].(string) + "\""
	date_json := []byte(date_string)
	var date CreatedTime

	if err := json.Unmarshal(date_json, &date); err != nil {
		return err
	}

	*v = Version{dat["number"].(string), date}
	return nil
}

func (v Version) MarshalJSON() ([]byte, error) {
	return json.Marshal(v)
}

func (ct CreatedTime) String() string {
	return time.Time(ct).Format(time.RFC3339Nano)
}

type Version struct {
	Number      string      `json:"number"`
	PublishedAt CreatedTime `json:"published_at"`
}

type VersionDependencies struct {
	Name           string
	Version        string
	VersionCreated time.Time
	Dependencies   []Dependency
}

type Dependency struct {
	Name            string
	RequiredVersion string
}

func (d Dependency) String() string {
	return fmt.Sprintf("%s:%s", d.Name, d.RequiredVersion)
}

func StreamParse(inPath string, jsonOutPathTemplate string) int {
	fmt.Println("Starting input JSON parser...")
	var wg sync.WaitGroup
	f, _ := os.Open(inPath)
	dec := json.NewDecoder(f)

	// versionPath := strings.Replace(outPath, ".", ".versions.", 1)
	// versionFile, err := os.OpenFile(versionPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)

	// if err != nil {
	// 	log.Fatal(err)
	// }

	// versionWriter := csv.NewWriter(versionFile)

	// Read opening bracket
	if _, err := dec.Token(); err != nil {
		log.Fatal(err)
	}
	// While the decoder says there is more to parse, parse a JSON entries and print them one-by-one
	i := 0
	for dec.More() {
		var e Entry

		if err := dec.Decode(&e); err != nil {
			log.Fatal(err)
		}
		timeStamps := e.Doc.Time

		var vds []VersionDependencies = make([]VersionDependencies, 0, len(e.Doc.Versions))

		for number, vd := range e.Doc.Versions {
			t := time.Time(timeStamps[number])
			deps, devDeps := vd.Dependencies, vd.DevDependencies
			allDependencies := make([]Dependency, 0, len(deps)+len(devDeps))
			for k, v := range deps {
				allDependencies = append(allDependencies, Dependency{k, v})
			}
			for k, v := range devDeps {
				allDependencies = append(allDependencies, Dependency{k, v})
			}

			vd := VersionDependencies{e.Doc.Name, number, t, allDependencies}
			vds = append(vds, vd)
		}
		jsonPath := fmt.Sprintf(jsonOutPathTemplate, fmt.Sprint(i)) // Append a number to filePath
		wg.Add(1)                                                   // Tell the WaitGroup it needs to wait for one more
		go func(vds *[]VersionDependencies, jsonPath string) {
			defer wg.Done() // Tell the WaitGroup this task is done after the function below is done
			writeToFileJSON(vds, jsonPath)
		}(&vds, jsonPath)

		i++
	}
	// Read closing bracket
	if _, err := dec.Token(); err != nil {
		log.Fatal(err)
	}
	wg.Wait() // Wait for all subroutines to be done
	fmt.Println("JSON parsing done")
	return i
}

func writeToFileJSON(vdAddr *[]VersionDependencies, outPath string) {
	outFile, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY, 0644)

	vds := *vdAddr

	if err != nil {
		log.Fatal(err)
	}
	defer outFile.Close()

	enc := json.NewEncoder(outFile)
	// Only when vds is non-empty
	if len(vds) > 0 {
		name := vds[0].Name
		versionMap := make(map[string]OutputVersion, len(vds))

		for _, vd := range vds {
			number := vd.Version
			timestamp := vd.VersionCreated
			deps := vd.Dependencies

			depMap := make(map[string]string, len(deps))

			for _, ver := range deps {
				depMap[ver.Name] = ver.RequiredVersion
			}

			outVersion := OutputVersion{timestamp, depMap}
			versionMap[number] = outVersion
		}

		out := OutputFormat{name, versionMap}

		// Error handling for encoding
		if err := enc.Encode(out); err != nil {
			log.Fatal(err)
		}
		// fmt.Printf("Wrote dependencies of %s to file \n", name)
	}
}

func MergeJSON(inPathTemplate string, amount int) {
	fmt.Println("Starting file merge process")
	var result []OutputFormat = make([]OutputFormat, 0, amount)
	outFile, err := os.OpenFile(fmt.Sprintf(inPathTemplate, "merged"), os.O_CREATE|os.O_WRONLY, 0644)
	enc := json.NewEncoder(outFile)

	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < amount; i++ {
		currentPath := fmt.Sprintf(inPathTemplate, fmt.Sprint(i))
		currentData, err := os.ReadFile(currentPath)

		// If the input file was empty, move on
		if len(currentData) < 1 {
			fmt.Printf("\tFile %d was empty\n", i)
			continue
		}

		if err != nil {
			log.Fatal(err)
		}

		var out OutputFormat
		if err := json.Unmarshal(currentData, &out); err != nil {
			log.Fatal(err)
		}
		result = append(result, out)
		//os.Remove(fmt.Sprintf(inPathTemplate, fmt.Sprint(i)))
	}

	enc.Encode(result)
	fmt.Println("Merged JSON files")
}
