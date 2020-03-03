package main

import (
	"log"
	"os"
	"reflect"
	"testing"
)

func TestChunk(t *testing.T) {
	testText := "apple"
	expectedChunks := []string{"ap", "pl", "e"}

	actualChunks := chunkData(testText, 2)

	if len(actualChunks) != len(expectedChunks) {
		t.Errorf("Expected %d chunks, but got %d", len(expectedChunks), len(actualChunks))
	}

	for ind := range actualChunks {
		if actualChunks[ind] != expectedChunks[ind] {
			t.Errorf("index=%d:\nExpected chunk=%s, but got chunk=%s", ind, expectedChunks[ind], actualChunks[ind])
		}
	}
}

func TestGetStreamNameOverride(t *testing.T) {
	err := os.Setenv("STREAM_NAME", "foobar")
	if err != nil {
		t.Fatalf("Could not set env var: %v", err)
	}
	name := getStreamName()
	if *name != "foobar" {
		t.Fatalf("Stream name was %s", *name)
	}
}

func TestGetRegionOverride(t *testing.T) {
	err := os.Setenv("AWS_REGION", "foobar")
	if err != nil {
		t.Fatalf("Could not set env var: %v", err)
	}
	name := getRegion()
	if *name != "foobar" {
		t.Fatalf("Stream name was %s", *name)
	}
}

func TestBuildMessages(t *testing.T) {
	expected := KinesisMessage{
		Version: 4,
		Source:  "event_json",
		Event: EventChunk{
			ChunkNumber: 0,
			NumChunks:   1,
			UUID:        "",
			Data:        "foobar",
			IsUpstream:  true,
		},
	}
	messages, err := buildMessages("foobar")
	if err != nil {
		log.Fatalf("Error building Kinesis messages: %v", err)
	}
	if len(messages) != 1 {
		log.Fatalf("Expected 1 message, got %d", len(messages))
	}
	first := messages[0]
	expected.Event.UUID = first.Event.UUID
	if !reflect.DeepEqual(first, expected) {
		log.Fatalf("Kinesis message is not the expected, got %v", first)
	}
}

func TestFlushKinesisBuffer(t *testing.T) {
	wrapper := kinesisWrapperHandler{}
	wrapper.flushBuffer()
}