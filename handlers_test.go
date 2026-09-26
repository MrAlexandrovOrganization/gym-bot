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

func TestCreatePoll(t *testing.T) {
	p := createPoll(-1001234567890)

	if p.ChatID.ID != -1001234567890 {
		t.Fatalf("ChatID.ID = %d, want -1001234567890", p.ChatID.ID)
	}
	if p.Question != pollQuestion {
		t.Fatalf("Question = %q, want %q", p.Question, pollQuestion)
	}
	if len(p.Options) != len(weekdays) {
		t.Fatalf("len(Options) = %d, want %d", len(p.Options), len(weekdays))
	}
	for i, opt := range p.Options {
		if opt.Text != weekdays[i] {
			t.Errorf("Options[%d].Text = %q, want %q", i, opt.Text, weekdays[i])
		}
	}
	if p.IsAnonymous == nil || *p.IsAnonymous {
		t.Fatal("IsAnonymous = nil or true, want false")
	}
	if !p.AllowsMultipleAnswers {
		t.Fatal("AllowsMultipleAnswers = false, want true")
	}
}
