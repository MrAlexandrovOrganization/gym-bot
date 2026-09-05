package main

import "testing"

type fakeDB struct {
	hasPoll bool
}

func (f *fakeDB) HasWeekPoll(int64, int, int) (bool, error)   { return f.hasPoll, nil }
func (f *fakeDB) SavePoll(int64, int, string, int, int) error { return nil }
func (f *fakeDB) GetWeekPoll(int64, int, int) (*Poll, error)  { return nil, nil }
func (f *fakeDB) Close() error                                { return nil }

func TestCheckWeekPoll(t *testing.T) {
	db := &fakeDB{hasPoll: true}
	hasPoll, err := checkWeekPoll(db, -100, 1, 2024)
	if err != nil {
		t.Fatalf("checkWeekPoll() error = %v", err)
	}
	if !hasPoll {
		t.Fatal("checkWeekPoll() = false, want true")
	}
}
