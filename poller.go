package main

import (
	"fmt"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"time"
	"regexp"
	"strconv"
	"strings"
	"os"
	"path/filepath"
)

var (
	WotcEventListUrl = "https://www.wizards.com/handlers/XMLListService.ashx?dir=mtgo&type=XMLFileInfo&start=%v"
	WotcEventPageUrl = "https://www.wizards.com/Magic/Digital/MagicOnlineTourn.aspx?x=mtg/digital/magiconline/tourn/%v"
	WotcDeckUrl = "https://www.wizards.com/magic/.dek?x=mtg/digital/magiconline/tourn/%v&decknum=%v"
	DeckHeaderReg, _ = regexp.Compile(`<heading>(\S+) \((.+)\)</heading>`)	
	CardListReg, _ = regexp.Compile(`(\d+) (.+)`)
	EventDateReg, _ = regexp.Compile(`(\d+)/(\d+)`)
)

func GetPageContent(url string) ([]byte, error) {
	resp, getErr := http.Get(url)
	if getErr != nil {
		fmt.Println("error retrieving page" + getErr.Error())
		return nil, getErr
	}
	defer resp.Body.Close()

	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		fmt.Println("error reading response" + readErr.Error())
		return nil, readErr
	}

	return body, nil
}

func GetEventList(daysToRetrieve int) (WotcEventList) {
	body, _ := GetPageContent(fmt.Sprintf(WotcEventListUrl, daysToRetrieve))

	var list WotcEventList
	parseErr := json.Unmarshal(body, &list)
	if parseErr != nil {
		fmt.Println("error parsing response" + parseErr.Error())
		return nil
	}

	return list
}

func GetNewEvents(knownEvents map[string] bool, daysToRetrieve int) ([]*Event) {
	evList := GetEventList(daysToRetrieve)
	newEvents := make([]*Event, 0, len(evList))
	for _, evLink := range evList {
		if knownEvents[evLink.Hyperlink] {
			continue
		}
		ev := new(Event)
		ev.Format = ParseEventFormat(evLink.Name)
		ev.Date = ParseEventDate(evLink.Date)
		ev.EventID = evLink.Hyperlink
		ev.Decks = GetEventDecks(ev)
		newEvents = append(newEvents, ev)
	}
	return newEvents
}

func GetEventDecks(ev *Event) ([]*Deck) {
	decks := GetDeckHeaders(ev.EventID)
	for i, deck := range decks {
		deck.Format = ev.Format
		deck.Date = ev.Date
		deck.EventID = ev.EventID
		listText := GetDeckList(ev.EventID, i+1)
		deck.MainDeck, deck.Sideboard = ParseDeckList(listText)
	}
	return decks
}

func GetDeckHeaders(eventID string) ([]*Deck) {
	body, _ := GetPageContent(fmt.Sprintf(WotcEventPageUrl, eventID))
	
	headers := DeckHeaderReg.FindAllStringSubmatch(string(body), -1)
	fmt.Printf("Event %v found %v deck headers\n", eventID, len(headers))
	decks := make([]*Deck, 0, len(headers))
	for _, match := range headers {
		if len(match) == 3 {
			deck := new(Deck)
			deck.Pilot = match[1]
			deck.Result = match[2]
			decks = append(decks, deck)
		}
	}

	return decks
}

func GetDeckList(eventID string, deckNum int) (string) {
	body, _ := GetPageContent(fmt.Sprintf(WotcDeckUrl, eventID, deckNum))
	return string(body)
}

func ParseDeckList(listText string) ([]Card, []Card) {
	maindeck := make([]Card, 0, 100)
	sideboard := make([]Card, 0, 50)

	inSideboard := false
	for _, line := range strings.Split(listText, "\r\n") {
		if len(line) == 0 { //blank line is separator between maindeck and sideboard
			inSideboard = true
			continue
		}

		m := CardListReg.FindStringSubmatch(line)
		if len(m) == 3 {
			num, _ := strconv.Atoi(m[1])
			c := Card { num, m[2] }
			if inSideboard {
				sideboard = append(sideboard, c)
			} else {
				maindeck = append(maindeck, c)
			}
		}
	}
	return maindeck, sideboard
}

func ParseEventFormat(eventName string) (string) {
	switch {
	case strings.Contains(eventName, "Standard"):
		return "Standard"
	case strings.Contains(eventName, "Modern"):
		return "Modern"
	case strings.Contains(eventName, "Pauper"):
		return "Pauper"
	case strings.Contains(eventName, "Classic"):
		return "Classic"
	case strings.Contains(eventName, "Sealed RTR Block"):
		return "RTR Block Sealed"
	case strings.Contains(eventName, "RTR Block"):
		return "RTR Block"
	}
	fmt.Printf("Unrecognized format: %v\n", eventName)
	return eventName
}

func ParseEventDate(eventDate string) (time.Time) {
	m := EventDateReg.FindStringSubmatch(eventDate) //this doesn't contain year?
	if len(m) == 3 {
		month, _ := strconv.Atoi(m[1])
		day, _ := strconv.Atoi(m[2])
		return time.Date(time.Now().Year(), time.Month(month), day, 0, 0, 0, 0, time.UTC)
	} 
	fmt.Println("Gave up on parsing date: ", eventDate)
	return time.Now()	
	
}

func CreateDirIfNecessary(path string) {
	_, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		dirCreateErr := os.MkdirAll(path, os.FileMode(0666))
		if dirCreateErr != nil {
			fmt.Println("Failed to create directory: ", dirCreateErr.Error())
		}
	}
}

func WriteEventToDisk(ev *Event) {	
	dirPath := filepath.Join("events", ev.Format)
	CreateDirIfNecessary(dirPath)
	filename := filepath.Join(dirPath, ev.EventID)
	fmt.Println("Saving to ", filename)
	serialized, jsonErr := json.MarshalIndent(ev, "", "\t")
	if jsonErr != nil {
		fmt.Println("event serialization failed: ", jsonErr.Error())
		return
	}
	createErr := ioutil.WriteFile(filename, serialized, os.FileMode(0666))
	if createErr != nil {
		fmt.Println("file create failed: ", createErr.Error())
		return
	}
}

func LoadEventsFromDisk() ([]*Event) {
	return nil
}

func main() {
	fmt.Println("starting")
	knownEvents := make(map[string] bool)
	events := GetNewEvents(knownEvents, 1)
	fmt.Printf("new events: %v", len(events))
	for _, ev := range events {
		WriteEventToDisk(ev)
	}
}