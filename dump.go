package main

import (
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
	"io"
	"log"
	"net/url"
	"os"
	"time"
)

type DumpCommand struct {
	Index        string    `required help:"Index filter" short:"i"`
	QueryString  string    `help:"Query string to filter results" short:"q"`
	Size         string    `help:"Bulk size" default:"100" short:"s"`
	OutputFile   string    `help:"Output file" short:"o"`
	OutputWriter io.Writer `kong:"-"`
}

func (cmd *DumpCommand) Run(dispatcher *EsQueryDispatcher) (err error) {
	log.Println("Dump starting...")
	log.Println("Endpoint : " + shellYellow(dispatcher.BaseUrl))
	log.Println("Index : " + shellYellow(cmd.Index))
	scrollUrl := "/%s/_search?scroll=5m"
	if len(cmd.QueryString) > 0 {
		scrollUrl += "&q=" + url.QueryEscape(cmd.QueryString)
	}
	if len(cmd.OutputFile) > 0 {
		cmd.OutputWriter, err = os.Create(cmd.OutputFile)
		if err != nil {
			return nil
		}
		log.Println("Output file : " + shellYellow(cmd.OutputFile))
	} else {
		cmd.OutputWriter = os.Stdout
		log.Println("Output to stdout")
	}
	scrollRequest := &ScrollRequest{
		Size: cmd.Size,
		Sort: "_doc",
	}
	scrollResponse := &ScrollResponse{}

	err, _ = dispatcher.ESRequest("POST", fmt.Sprintf(scrollUrl, cmd.Index), scrollResponse, scrollRequest)
	if err != nil {
		return err
	}
	log.Println("Got scrollId : " + shellYellow(scrollResponse.ScrollId))
	var totalHits int
	if parsedTotalHits, parsedInt := scrollResponse.Hits.Total.(float64); parsedInt {
		totalHits = int(parsedTotalHits)
	} else {
		totalHits = int(scrollResponse.Hits.Total.(map[string]interface{})["value"].(float64))
	}
	log.Println(fmt.Sprintf("Dumping %s documents :", shellYellow(totalHits)))
	// Progress bar, yeaaah
	bar := progressbar.New(totalHits)
	jsonEncoder := json.NewEncoder(cmd.OutputWriter)
	for _, hit := range scrollResponse.Hits.Hits {
		bar.Add(1)
		err = jsonEncoder.Encode(hit)
		if err != nil {
			return err
		}
	}
	scrollResumeRequest := &ScrollResume{
		Scroll:   "5m",
		ScrollId: scrollResponse.ScrollId,
	}
	// now we loop and do the same using our scrollID
	for {
		scrollResponse := &ScrollResponse{}
		err, _ = dispatcher.ESRequest("POST", "/_search/scroll", scrollResponse, scrollResumeRequest)
		if err != nil {
			log.Println("Timed out, continuing in 10s ...")
			time.Sleep(10 * time.Second)
			continue
		}
		if len(scrollResponse.Hits.Hits) < 1 {
			// Seems like there's no more results, exit
			log.Println("Dump completed")
			os.Exit(0)
		}
		for _, hit := range scrollResponse.Hits.Hits {
			bar.Add(1)
			err = jsonEncoder.Encode(hit)
			if err != nil {
				return err
			}
		}
	}
}

var shellYellow = color.New(color.FgYellow).SprintFunc()
var shellRed = color.New(color.FgRed).SprintFunc()

type ScrollResponse struct {
	Scroll   string `json:"scroll"`
	ScrollId string `json:"_scroll_id"`
	Hits     struct {
		Total interface{}   `json:"total"`
		Hits  []interface{} `json:"hits"`
	} `json:"hits"`
}

type ScrollRequest struct {
	Size string `json:"size"`
	Sort string `json:"sort"`
}

type ScrollResume struct {
	Scroll   string `json:"scroll"`
	ScrollId string `json:"scroll_id"`
}
