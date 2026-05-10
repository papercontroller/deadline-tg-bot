package main

import (
	"sync"
	"time"
)

type Step int

const (
	StepIdle Step = iota
	StepAddText
	StepAddDate
	StepAddTime
	StepUpdateSelect
	StepUpdateField
	StepUpdateText
	StepUpdateDate
	StepUpdateTime
	StepDeleteSelect
)

type UserState struct {
	Step        Step
	TempText    string
	TempDate    time.Time
	DeadlineID  int64
	CalYear     int
	CalMonth    time.Month
	SelectedIDs map[int64]bool
}

var userStates = struct {
	sync.Mutex
	m map[int64]UserState
}{m: make(map[int64]UserState)}

func getState(userID int64) UserState {
	userStates.Lock()
	defer userStates.Unlock()
	return userStates.m[userID]
}

func setState(userID int64, s UserState) {
	userStates.Lock()
	defer userStates.Unlock()
	userStates.m[userID] = s
}

func resetState(userID int64) {
	setState(userID, UserState{})
}
