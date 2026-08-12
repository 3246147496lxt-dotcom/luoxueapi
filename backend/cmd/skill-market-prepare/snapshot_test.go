package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestParseSkillsHTMLReadsOrderedNextFlightHydration(t *testing.T) {
	records := []rankingRecord{
		{Source: "owner/one", SkillID: "first", Name: "First", Installs: 300},
		{Source: "owner/two", SkillID: "second", Name: "Second", Installs: 200},
		{Source: "skills.example.com", SkillID: "third", Name: "Third", Installs: 100},
	}
	rawRecords, err := json.Marshal(records)
	if err != nil {
		t.Fatal(err)
	}
	payload := fmt.Sprintf(`4e:["$","component",null,{"initialSkills":%s,"other":true}]`, rawRecords)
	flight, err := json.Marshal([]any{1, payload})
	if err != nil {
		t.Fatal(err)
	}
	html := []byte(`<html><body><script>self.__next_f.push(` + string(flight) + `)</script></body></html>`)

	got, err := parseSkillsHTML(html, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d records, want 3", len(got))
	}
	for index := range records {
		if got[index] != records[index] {
			t.Fatalf("record %d = %#v, want %#v", index, got[index], records[index])
		}
	}
}

func TestParseSkillsHTMLEnforcesMinimumRecordCount(t *testing.T) {
	payload := `x:{"initialSkills":[{"source":"owner/repo","skillId":"one","name":"One","installs":1}]}`
	flight, _ := json.Marshal([]any{1, payload})
	_, err := parseSkillsHTML([]byte(`<script>self.__next_f.push(`+string(flight)+`)</script>`), 2)
	if err == nil || !strings.Contains(err.Error(), "need at least 2") {
		t.Fatalf("expected minimum-record error, got %v", err)
	}
}
