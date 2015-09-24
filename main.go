package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/groob/vquery/axiom"
)

var (
	Version  = "unreleased"
	fVersion = flag.Bool("version", false, "display the version")
	fConfig  = flag.String("config", "", "configuration file to load")
	conf     config
	client   *axiom.Client
	wg       sync.WaitGroup
)

type report struct {
	Name string `toml:"name"`
	ID   int    `toml:"id"`
}
type config struct {
	Interval          time.Duration `toml:"interval"`
	VeracrossUsername string        `toml:"veracross_username"`
	VeracrossPassword string        `toml:"veracross_password"`
	VeracrossSchool   string        `toml:"veracross_school"`
	Reports           []report      `toml:"reports"`
	ReportsPath       string        `toml:"reports_path"`
}

func init() {
	flag.Parse()

	if *fVersion {
		fmt.Printf("axiom-query - version %s\n", Version)
		os.Exit(0)
	}

	if _, err := toml.DecodeFile(*fConfig, &conf); err != nil {
		log.Fatal(err)
	}

	c, err := axiom.NewClient(conf.VeracrossUsername, conf.VeracrossPassword,
		conf.VeracrossSchool,
	)
	if err != nil {
		log.Fatal(err)
	}
	client = c
}

func saveReport(jsonData []byte, name string) error {
	jsonFile, err := os.Create(conf.ReportsPath + "/" + name + ".json")
	if err != nil {
		return err
	}
	defer jsonFile.Close()
	_, err = jsonFile.Write(jsonData)
	if err != nil {
		return err
	}
	return nil
}

func runReport(reportID int, name string) {
	// runs a report from Veracross and saves the JSON file localy
	defer wg.Done()
	request := fmt.Sprintf("https://axiom.veracross.com/whitby/query/%v/result_data.json", reportID)
	req, err := http.NewRequest("POST", request, nil)
	req.Header.Set("x-csrf-token", client.Token)
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return
	}
	//save report
	err = saveReport(body, name)
	if err != nil {
		log.Println(err)
		return
	}
}

func run(done chan bool) {
	for _, report := range conf.Reports {
		wg.Add(1)
		go runReport(report.ID, report.Name)
	}
	wg.Wait()
	done <- true
}

func main() {
	done := make(chan bool)
	ticker := time.NewTicker(time.Minute * conf.Interval).C
	for {
		go run(done)
		<-done
		<-ticker
	}
}
