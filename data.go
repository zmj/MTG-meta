package main

import (
	"time"
)

type Event struct {
	Format string
	Date time.Time
	EventID string
	Decks []*Deck
}

type Deck struct {
	Format string	
	Date time.Time
	EventID string
	Pilot string
	Result string
	MainDeck []Card
	Sideboard []Card
}

type Card struct {
	Number int
	Name string
}

type WotcEventList []WotcEvent

type WotcEvent struct {
	Date string
	Hyperlink string
	Name string
}