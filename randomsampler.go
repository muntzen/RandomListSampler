package main

import (
	"fmt"
	"bufio"
	"os"
	"net/http"
	"strings"
	"log"
	"io"
	"encoding/json"
	"bytes"
	"strconv"
	"math"
	"math/rand"
	"time"
	"crypto/md5"
	"encoding/hex"
)

type Lists struct {
	Lists []List `json:"lists"`
	TotalCount int `json:"total_items"`
}

type List struct {
	ID string `json:"id"`
	Name string `json:"name"`
	WebID int `json:"web_id"`
	Stats ListStats `json:"stats"`
}

type ListStats struct {
	Count int `json:"member_count"`
}

type Members struct {
	Members []Member `json:"members"`
	TotalCount int `json:total_items`
	ListID string `json:list_id`
}

type Member struct {
	Email string `json:"email_address"`
	UniqueEmailID string `json:"unique_email_id"`
}

type TagList struct {
	Syncing bool `json:"is_syncing"`
	Tags []Tag `json:"tags"`
}

type TagSearchResults struct {
	Tags []Tag
}

type Tag struct {
	ID int `json:"id"`
	Name string `json:"name"`
	Status string `json:"status"`
}

var apiKey string
var apiDebug bool

func main() {
	fmt.Println("\n\n--------------------------------------\nRANDOM SAMPLER! LET'S TAG A RANDOM SAMPLE OF YOUR MAILCHIMP LIST\n--------------------------------------\n\n");

	apiDebug = false
	setApiKey()

	list := selectList()

	// now show the overall size and ask for a % of the overall list and a name for the tag
	fmt.Printf("\nOK, we're going to create a tag for a random percent of the list '%s', which has a list size of %d.\nWhat percent do you want to test?\n\n", list.Name, list.Stats.Count)
	percent, _ := strconv.ParseFloat(readFromStdin(), 64)
	numToTag := int(math.Round(float64(list.Stats.Count) * (percent / 100.0)))

	// get the name of the new tag
	currentTime := time.Now()
	suggestion := "Random " + fmt.Sprintf("%.2g", percent) + "% " + currentTime.Format("2006-01-02")
	fmt.Printf("\n\nWhat would you like to name this tag? (Suggestion: '%s', hit [enter] to use suggestion)\n\n", suggestion)
	tagName := readFromStdin()
	if strings.Trim(tagName, " ") == "" {
		tagName = suggestion
	}

	if numToTag > list.Stats.Count || numToTag <= 0 {
		fmt.Printf("\nNumber to tag %d is invalid, please try again\n", numToTag)
		os.Exit(0)
	}

	for {
		fmt.Printf("\nWe'll be creating a tag named '%s' and tagging %d members is that right? (Y/N)\n", tagName, numToTag)
		response := strings.ToLower(readFromStdin())
		if response == "y" {
			break;
		} else if response == "n" {
			fmt.Println("OK, re-run and try again")
			os.Exit(0)
		}
	}

	tagMembers(list, tagName, numToTag)

	// show the user how to segment based on the tag you just set and say woot
	var tagResults TagSearchResults
	params := map[string]string {"name": tagName}
	jsonStr := []byte (callApi("lists/" + list.ID + "/tag-search", "GET", params, nil))
	err := json.Unmarshal(jsonStr, &tagResults)
	if err != nil { log.Fatal(err) }
	apiParts := getApiParts()
	url := "https://" + apiParts[1] + ".admin.mailchimp.com/lists/segments/members/tags-filter?list_id=" + strconv.Itoa(list.WebID) + "&tag_ids[]=" + strconv.Itoa(tagResults.Tags[0].ID);
	fmt.Println("\n\nDONE! \nTo view your newly tagged list members, follow this link: \n\t" + url)
}

func selectList() List {
	// get the lists in the account to figure out which one to work with
	lists := getLists()

	// pick the list we're working with
	var list List;
	if (len(lists) > 1) {
		// TODO: display them all ask them to key in the list
		fmt.Printf("Your account has %d lists, we need to choose the right one...\n", len(lists))
		for i, list := range lists {
			fmt.Printf("[%d] '%s' (%d members)\n", i, list.Name, list.Stats.Count)
		}
		fmt.Printf("Enter the number next to the name of the list to use\n")
		if listNum, err := strconv.Atoi(readFromStdin()); err == nil && listNum >= 0 && listNum < len(lists) {
			list = lists[listNum];
		} else {
			fmt.Printf("\nInvalid input, restart the process and try again")
			os.Exit(0)
		}
	} else {
		list = lists[0];
	}
	
	return list
}

func setApiKey() {

	if _, err := os.Stat(".apikey"); err == nil {	
		data, err := os.ReadFile(".apikey");
		if err == nil {
			apiKey = strings.Trim(string(data), "\n");
			if (pingMailchimp()) {
				return;
			} else {
				fmt.Printf("API Key found in cache, but is invalid %s\n", data);
			}	
		}
	}

	fmt.Println("Enter your Mailchimp API Key to get started")
	fmt.Println("If you need to create a new API key, do so at https://admin.mailchimp.com/account/api/")
	fmt.Println("Enter your API Key below...\n")

	apiKey = readFromStdin()

	if (!pingMailchimp()) {
		fmt.Println("API call to /ping failed with this key, make sure your API key is properly formatted. Example: 123123123123-us10")
		os.Exit(1)
	} else {
		err := os.WriteFile(".apikey", []byte(apiKey), 0644)
		if err != nil {
			fmt.Println("(unable to save API key for subsequent runs, but we can keep going...)");
		}
	}
}

func readFromStdin() string {
	reader := bufio.NewReader(os.Stdin)
	str, _ := reader.ReadString('\n')
	str = strings.Trim(str, "\n")
	return str;
}

func tagMembers(list List, tagName string, numToTag int) {
	tags := TagList{Syncing: false, Tags: []Tag{ Tag{Name: tagName, Status: "active"} }}	
	tagged := make(map[string]bool)

	// loop through all members in the list until we've tagged the right amount
	total := 0
	batchSize := int(math.Min(1000, float64(list.Stats.Count)))
	currentBatch := 0
	for {
		// get the list of subscribers
		params := map[string]string { "offset":strconv.Itoa(currentBatch), "count":strconv.Itoa(batchSize), "status":"subscribed"}
		fmt.Println("\n\nCalling Mailchimp API for members to tag, this may take a few moments...\n")
		jsonStr := []byte (callApi("lists/" + list.ID + "/members", "GET", params, nil))

		var members Members
		err := json.Unmarshal(jsonStr, &members)
		if err != nil { log.Fatal(err) }
	
		for _, member := range members.Members {
			if err != nil {
				log.Fatal(err)
			}
			// roll a dice to see if we should tag this one or not
			if (!tagged[member.Email] && rand.Intn(list.Stats.Count) <= numToTag) {
				// if so, tag 'em			
				fmt.Printf("+")// just doing this so we get some updates
				jsonBody, err := json.Marshal(tags)					
				if err != nil { log.Fatal(err) }
				callApi("lists/" + list.ID + "/members/" + getMD5Hash(member.Email) + "/tags", "POST", nil, jsonBody);
				tagged[member.Email] = true
				total = total + 1;
			} else {
				fmt.Printf("-")// just doing this so we get some feedback
			}
			if (total == numToTag) {
				break				
			}
		}
		if (total == numToTag) {
			break				
		}
		currentBatch = currentBatch + 1
		if currentBatch > (list.Stats.Count / batchSize) {
			currentBatch = 0
		}
	}
}

func getMD5Hash(email string) string {
	hash := md5.Sum([]byte(email))
	return hex.EncodeToString(hash[:])
 }

func getLists() []List {
	jsonStr := []byte(callApi("lists", "GET", nil, nil))
	var lists Lists
	err := json.Unmarshal(jsonStr, &lists)
	if err != nil {
		log.Fatal(err)
	}
	return lists.Lists;
}

func pingMailchimp() bool {
	jsonStr := []byte(callApi("ping", "GET", nil, nil))
	var data map[string]interface{}
	err := json.Unmarshal(jsonStr, &data)
	if err != nil {
		log.Fatal(err)
	}
	return data["health_status"] == "Everything's Chimpy!"
}

func getApiParts() []string {
	return strings.Split(apiKey, "-");
}

func callApi(endpoint string, requestType string, params map[string]string, bodyJson []byte) string {
	if apiDebug {
		fmt.Println("apiEndpoint: ", endpoint)
	}
	apiParts := getApiParts()
	client := &http.Client {}
	var bodyBytes io.Reader

	// build the body if it's passed in
	if bodyJson != nil {
		if apiDebug { fmt.Println("Calling API with body: ", string(bodyJson)) }
		bodyBytes = bytes.NewBuffer(bodyJson)
	}

	req, err := http.NewRequest(requestType, "https://" + apiParts[1] + ".api.mailchimp.com/3.0/" + endpoint, bodyBytes)
	if err != nil {
		log.Fatal(err)
	}

	req.SetBasicAuth(apiParts[0], apiParts[0])
	
	// add query params
	q := req.URL.Query()
	if params != nil  {
		for key, val := range params {
			q.Add(key, val)
		}
	}
	req.URL.RawQuery = q.Encode()

	// run the request
	res, err := client.Do(req)	
	if err != nil {
		log.Fatal(err)
	}

	defer res.Body.Close()
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	if apiDebug {
		fmt.Println("Request: ", req.URL.String())
		fmt.Println("Mailchimp Response: ", string(resBody))				
	}
	return string (resBody)
}