package lib

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/net/proxy"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var DebugWriter = ioutil.Discard

var httpClient = &http.Client{
	Transport: &http.Transport{
		DisableKeepAlives: true,
		Dial:              proxy.FromEnvironment().Dial,
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
	},
	Timeout: 15 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

type EsQueryDispatcher struct {
	Product              string
	Major                int
	ElasticsearchVersion string
	KibanaVersion        string
	BaseUrl              string
	LogOutput            io.Writer
}

func (disaptcher *EsQueryDispatcher) ESRequest(method, path string, target interface{}, body interface{}) (error, *http.Response) {
	switch {
	case disaptcher.Major >= 2:
		switch disaptcher.Product {
		case "elasticsearch":
			switch {
			case strings.ToLower(method) == "get":
				return disaptcher.GetJSONObject(disaptcher.BaseUrl+path, target)
			case strings.ToLower(method) == "post":
				return disaptcher.PostJsonObject(disaptcher.BaseUrl+path, target, body)
			}
		case "kibana":
			if disaptcher.Major >= 5 {
				kibanaUrl := fmt.Sprintf(disaptcher.BaseUrl+"/api/console/proxy?method=%s&path=%s", strings.ToUpper(method), url.QueryEscape(path))
				return disaptcher.PostJsonObject(kibanaUrl, target, body)
			}
			switch {
			case strings.ToLower(method) == "get":
				return disaptcher.GetJSONObject(disaptcher.BaseUrl+"/elasticsearch"+path, target)
			case strings.ToLower(method) == "post":
				return disaptcher.PostJsonObject(disaptcher.BaseUrl+"/elasticsearch"+path, target, body)
			}
		}
	}
	return errors.New("unsuported version"), nil
}

func (disaptcher *EsQueryDispatcher) GetJSONObject(url string, target interface{}) (err error, response *http.Response) {
	req, err := http.NewRequest("GET", url, nil)
	req.Header["User-Agent"] = []string{"estk/0.0.1 (+https://github.com/LeakIX/estk)"}
	req.Header["Accept"] = []string{"application/json"}
	req.Header["kbn-xsrf"] = []string{"true"}
	req.Header["kbn-version"] = []string{disaptcher.KibanaVersion}
	if err != nil {
		return err, nil
	}
	// use the http client to fetch the page
	resp, err := httpClient.Do(req)
	if err != nil {
		return err, nil
	}
	defer resp.Body.Close()
	ioTee := io.TeeReader(resp.Body, DebugWriter)
	jsonDecoder := json.NewDecoder(ioTee)
	return jsonDecoder.Decode(target), resp
}

func (disaptcher *EsQueryDispatcher) PostJsonObject(url string, target interface{}, body interface{}) (err error, response *http.Response) {
	jsonBody, err := json.Marshal(body)
	if err != nil || body == nil {
		jsonBody = nil
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return err, nil
	}
	req.Header["User-Agent"] = []string{"estk/0.0.1 (+https://github.com/LeakIX/estk)"}
	req.Header["Accept"] = []string{"application/json"}
	req.Header["kbn-xsrf"] = []string{"true"}
	req.Header["kbn-version"] = []string{disaptcher.KibanaVersion}
	req.Header["Content-Type"] = []string{"application/json"}

	// use the http client to fetch the page
	resp, err := httpClient.Do(req)
	if err != nil {
		return err, nil
	}
	defer resp.Body.Close()
	ioTee := io.TeeReader(resp.Body, DebugWriter)
	jsonDecoder := json.NewDecoder(ioTee)
	return jsonDecoder.Decode(target), resp
}

func (disaptcher *EsQueryDispatcher) DetectEsVersion() (err error) {
	log.Println("Detecting version...")
	disaptcher.Product = "elasticsearch"
	esGreetReply := &EsGreetReply{}
	dispatcher := &EsQueryDispatcher{}
	log.Println("Trying elasticsearch")
	err, _ = dispatcher.GetJSONObject(disaptcher.BaseUrl, esGreetReply)
	if len(esGreetReply.Version.Number) < 5 {
		log.Println("Trying Kibana")
		disaptcher.Product = "kibana"
		err, resp := dispatcher.PostJsonObject(disaptcher.BaseUrl+"/api/console/proxy?method=GET&path=/", esGreetReply, nil)
		if err != nil {
			return err
		}
		if len(disaptcher.KibanaVersion) < 1 {
			disaptcher.KibanaVersion = resp.Header.Get("kbn-version")
		}
		if len(esGreetReply.Version.Number) < 5 {
			log.Println("Trying Kibana < 5")
			err, resp = dispatcher.GetJSONObject(disaptcher.BaseUrl+"/elasticsearch/", esGreetReply)
			if len(esGreetReply.Version.Number) < 5 {
				return errors.New("couldn't detect endpoint")
			}
			if len(disaptcher.KibanaVersion) < 1 {
				disaptcher.KibanaVersion = resp.Header.Get("kbn-version")
			}
		}
	}
	disaptcher.Major, err = strconv.Atoi(esGreetReply.Version.Number[0:1])
	if err != nil {
		return err
	}
	dispatcher.ElasticsearchVersion = esGreetReply.Version.Number
	log.Printf("Found %s, major version %d", disaptcher.Product, disaptcher.Major)
	return nil
}

type EsGreetReply struct {
	Name        string `json:"name"`
	ClusterName string `json:"cluster_name"`
	Version     struct {
		Number string `json:"number"`
	} `json:"version"`
}
