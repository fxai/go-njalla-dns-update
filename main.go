package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type UpdateListEnv struct {
	Update []struct {
		Sub    string `json:"sub"`
		Domain string `json:"domain"`
	} `json:"update"`
}

type Params struct {
	Domain   string `json:"domain"`
	RecordID int    `json:"id,omitempty"`
	Content  string `json:"content,omitempty"`
}

type Payload struct {
	Jsonrpc   string `json:"jsonrpc"`
	Method    string `json:"method"`
	Params    Params `json:"params,omitempty"`
	RequestID string `json:"id"`
}

type ResponseListRecords struct {
	Result struct {
		Records []struct {
			ID      int    `json:"id"`
			Name    string `json:"name"`
			Type    string `json:"type"`
			Content string `json:"content"`
			TTL     int    `json:"ttl"`
		} `json:"records"`
	} `json:"result"`
	ID      string `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
}

type ResponseEditRecord struct {
	Result struct {
		ID      int    `json:"id"`
		Name    string `json:"name"`
		Type    string `json:"type"`
		Content string `json:"content"`
		TTL     int    `json:"ttl"`
	} `json:"result"`
	ID      string `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
}

var NjallaToken = ""

func main() {
	log.Println("Starting Update Service")

	oldIP := "0.0.0.0"

	data, err := ioutil.ReadFile("/vault/secrets/api.txt")
	if err != nil {
		log.Println("API token reading error", err)
		return
	}

	NjallaToken = strings.TrimSuffix(string(data), "\n")

	//read the defined update interval from env variabale and convert it to some magic numbers
	a := os.Getenv("njalla_update_interval")
	interval, err := strconv.Atoi(a)
	if err != nil {
		log.Println(err.Error())
	}

	//main loop
	for _ = range time.Tick(time.Second * time.Duration(interval)) {
		newIP, err := getOwnIP()
		if err != nil {
			log.Println(err.Error())
		} else if oldIP != newIP {
			log.Println("I looked up following IP: " + newIP)
			log.Println("New IP detected. Init DNS updates")
			//no error handling this might be bad
			initUpdate(newIP)
			oldIP = newIP
		} else {
			log.Println("Same as old IP, no update needed, chill")
		}
	}

}

func getOwnIP() (string, error) {

	type OwnIP struct {
		IP string `json:"ip"`
	}

	//get ip from ipfy
	response, err := http.Get("https://api.ipify.org?format=json")
	if err != nil {
		return "-1", err
	}

	defer response.Body.Close()

	responseFromIpfy, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "-1", err
	}

	var ipFromIpfy = new(OwnIP)
	err = json.Unmarshal(responseFromIpfy, &ipFromIpfy)
	if err != nil {
		return "-1", err
	}

	//get IP from ipinfo

	response, err = http.Get("https://ipinfo.io")
	if err != nil {
		return "-1", err
	}

	defer response.Body.Close()

	responseFromIpinfo, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "-1", err
	}

	var ipFromIpinfo = new(OwnIP)
	err = json.Unmarshal(responseFromIpinfo, &ipFromIpinfo)
	if err != nil {
		return "-1", err
	}

	if ipFromIpinfo.IP != ipFromIpfy.IP {
		return "-1", errors.New("Something strange, external IP's do not match. Ipinfo: " + ipFromIpinfo.IP + " Ipfy: " + ipFromIpfy.IP)
	}
	return ipFromIpinfo.IP, nil
}

func initUpdate(newIP string) {

	//save the environment variable $njalla_update into updatelist varible of struct UpdateListenv
	updatelist, err := parsToUpdate()
	if err != nil {
		log.Println(err.Error())
	}

	//for each sub/domain entry in the updatelist grab the list of subdomains from njalla
	for _, s := range updatelist.Update {

		//returns the list of subdomains in a domain
		list, err := listRecords(s.Domain)
		if err != nil {
			log.Println(err.Error())
		}

		//for each returned subdomain, check if it matches the subdomain in the updatelist. Invoke update if true.
		for _, sub := range list.Result.Records {
			if sub.Name == s.Sub {
				log.Println("update record for " + sub.Name + "." + s.Domain)
				editRecordResponse, err := editRecord(s.Domain, sub.ID, newIP)
				if err != nil {
					log.Println(err.Error())
				}
				log.Println("Njalla Response for update request:")
				log.Println(editRecordResponse.Result)
			}
		}

	}

}

func parsToUpdate() (*UpdateListEnv, error) {

	var updatelist = new(UpdateListEnv)
	byt := []byte(os.Getenv("njalla_update"))

	err := json.Unmarshal(byt, &updatelist)
	if err != nil {
		return nil, err
	}

	return updatelist, nil

}

func listRecords(domain string) (*ResponseListRecords, error) {

	data := Payload{
		Jsonrpc:   "2.0",
		Method:    "list-records",
		Params:    Params{domain, 0, ""},
		RequestID: "123",
	}

	resp, err := request(data)
	if err != nil {
		return nil, err
	}

	var s = new(ResponseListRecords)
	err = json.Unmarshal(resp, &s)
	if err != nil {
		return nil, err
	}

	return s, err

}

func editRecord(domain string, id int, content string) (*ResponseEditRecord, error) {

	data := Payload{
		Jsonrpc:   "2.0",
		Method:    "edit-record",
		Params:    Params{domain, id, content},
		RequestID: "123",
	}

	resp, err := request(data)
	if err != nil {
		return nil, err
	}

	var s = new(ResponseEditRecord)
	err = json.Unmarshal(resp, &s)
	if err != nil {
		return nil, err
	}

	return s, err

}

func request(data Payload) ([]byte, error) {

	payloadBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", "https://njal.la/api/1/", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Njalla "+NjallaToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	response, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	return response, nil

}
