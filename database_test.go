package main

import (
	"slices"
	"testing"
)

func TestWebhookStorage(t *testing.T) {
	db := &memStore{}

	var urls []string
	var err error

	urls, err = getWebhooks(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(urls) != 0 {
		t.Fatalf("initial webhooks should be empty")
	}

	urls, err = addWebhook(db, "google.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(urls) != 1 || urls[0] != "google.com" {
		t.Fatal("failed to store webhook")
	}

	urls, err = addWebhook(db, "google.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(urls) != 1 || urls[0] != "google.com" {
		t.Fatal("should not store duplicate")
	}

	urls, err = addWebhook(db, "ebay.com")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(urls, "google.com") || !slices.Contains(urls, "ebay.com") {
		t.Fatal("failed to remember many webhooks")
	}

	urls, err = deleteWebhook(db, "google.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(urls) != 1 || urls[0] != "ebay.com" {
		t.Fatal("failed to delete webhook")
	}
}
