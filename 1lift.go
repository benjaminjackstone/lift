//
// Lift simulator
// CS 3520
//
// Starter code
//

package main

import (
	"log"
	"math/rand"
	"time"
)

// the length of time to pause between floors and when opening doors
var Pause = time.Second

//
// Controller component
//

// A controller component.
// Use a pointer to this struct as your handle to a controller.
// Send a Step message to it by sending a destination floor across
// the Step channel. Only the lift should send this message.
//
// When NewController returns, the Lift field is nil. It should be
// set to a valid Lift before any messages are sent to the
// controller.
//
// The controller assumes that it starts on floor 1 with the motor
// stopped.

type Controller struct {
	Step chan int
	Lift *Lift
}

// Construct a new controller component.
// Returns a handle to the controller, which runs in its own
// goroutine.
func NewController() *Controller {
	elt := &Controller{make(chan int), nil}
	motorRunning := false
	floor := 1
	timerResponse := make(<-chan time.Time)
	go func() {
		for {
			if motorRunning {
				<-timerResponse
				elt.Lift.At <- floor
				motorRunning = false
			} else {
				destination := <-elt.Step
				if destination < 0 {
					return
				}
				switch {
				case floor == destination:
					return
				case floor < destination:
					timerResponse = time.After(Pause)
					motorRunning = true
					floor++
				case floor > destination:
					timerResponse = time.After(Pause)
					motorRunning = true
					floor--
				}
			}
		}
	}()
	return elt
}

//
// Lift component
//

// A helper function to append a new element to the end of a slice,
// but only if it does not match the existing last element of the
// slice.
func schedulelast(slice []int, elt int) []int {
	if len(slice) == 0 {
		slice = append(slice, elt)
	} else {
		if slice[len(slice)-1] == elt {
			//do nothing
		} else {
			slice = append(slice, elt)
		}
	}
	return slice
}

// A lift component.
// Use a pointer to this struct as your handle to a lift.

// Send an At message to it by sending a floor number across the At
// channel. This should only be sent by the lift's controller, and
// indicates that the lift has arrived at a new floor.

// Send a Call message to it by sending a new destination floor
// number across the Call channel. This can come from a floor
// component (when someone presses the call button on that floor) or
// directly from a user (when the user is inside the elevator and
// pushes a button).
//
// Send a negative value to Call to shut the lift down and end its
// goroutine.
type Lift struct {
	At   chan int
	Call chan int
}

// Construct a new lift component.
// Returns a handle to the lift, which runs in its own goroutine.
//
// number is the lift number (1-based numbering)
//
// controller is the controller for this lift.
//
// floors is a slice of all of the floors in the building. This is a
// 1-based list, so entry #0 is unused. It is okay to pass in a
// slice of nil values, as long as they are all filled in before
// any messages are sent to the lift.
func NewLift(number int, controller *Controller, floors []*Floor) *Lift {
	lift := &Lift{make(chan int), make(chan int)}
	floor := 1
	schedule := make([]int, 0)
	moving := false
	go func() {
		for {
			select {
			case destination := <-lift.Call:
				if destination < 0 {
					return
				} else {
					log.Println("Lift ", number, " has been called to floor", destination)
					if destination == floor && moving == false {
						ack := make(AckChan)
						floors[destination].Arrive <- ack
						<-ack
					} else {
						schedule = schedulelast(schedule, destination)

						if moving == false {
							moving = true
							controller.Step <- destination
						}
					}
				}
			case newFloor := <-lift.At:
				// respond to an At message
				floor = newFloor
				log.Println("Lift ", number, " has arrived at floor ", floor)
				if floor == schedule[0] {
					ack := make(AckChan)
					floors[floor].Arrive <- ack
					<-ack
					schedule = schedule[1:]
					if len(schedule) == 0 {
						moving = false
					} else {
						controller.Step <- schedule[0]
						moving = true
					}
				} else {
					controller.Step <- schedule[0]
				}
			}
		}
	}()
	return lift
}

//
// Floor component
//

// a convenience type representing an acknowledgement channel
type AckChan chan bool

// the states that a floor can be in
type FloorState int

const (
	Called    FloorState = iota // the floor is waiting for a lift to arrive
	NotCalled                   // the floor is idle
	DoorsOpen                   // at least one lift is at this floor with its doors open
)

// A floor component.
// Use a pointer to this struct as your handle to a floor.
//
// Send a Call message to it by sending a boolean value across the
// Call channel. This should be send by a user, i.e., it can come
// from anywhere.
//
// Send an Arrive message to the lift by sending an AckChan across
// the Arrive channel. This message indicates that a lift has
// arrived as the result of a user call. Its doors are immediately
// opened, and a boolean should be send across the AckChan when they
// are closed again. This message is only sent by lift components.
//
// Send a negative value to Call to shut the floor down and end its
// goroutine.
type Floor struct {
	Call   chan bool
	Arrive chan AckChan
}

// Construct a new floor component.
// Returns a handle to floor, which runs in its own goroutine.
//
// number is the floor number (1 based numbering)
//
// lifts is a slice containing all the lifts in the building. This
// is a 1-based list, so entry #0 is unused. It can contain nil
// values when NewFloor is called, as long as the lift values are
// filled in before any messages are send to the floor.
func NewFloor(number int, lifts []*Lift) *Floor {
	state := NotCalled
	floor := &Floor{make(chan bool), make(chan AckChan)}
	acknow := make([]AckChan, 0)
	go func() {
		for {
			if state == NotCalled {
				select {
				case ack := <-floor.Arrive:
					log.Println("Arrived at floor ", number)
					state = DoorsOpen
					<-time.After(Pause)
					acknow = append(acknow, ack)

				case <-floor.Call:

					elev := rand.Intn(len(lifts)-1) + 1
					log.Println("Calling elevator ", elev)
					lifts[elev].Call <- number
					state = Called
				}
			} else if state == Called {
				select {
				case ack := <-floor.Arrive:
					log.Println("Arrived at floor ", number)
					<-time.After(Pause)
					state = DoorsOpen
					acknow = append(acknow, ack)

				case <-floor.Call:
					// do nothing
				}
			} else if state == DoorsOpen {
				select {
				case <-time.After(Pause):
					log.Println("Doors are closing on floor ", number)
					for i := 0; i < len(acknow); i++ {
						acknow[i] <- false
					}
					state = NotCalled

				case ack := <-floor.Arrive:
					acknow = append(acknow, ack)

				case <-floor.Call:
					// do nothing
				}
			}

		}
	}()

	return floor
}

//
// Building component
//

// Construct a new building super-component.
// Returns the slice of floors and the slice of lifts, given the
// desired number of each. Note that these slices are 1-based, so
// entry #0 in each is unused.
func NewBuilding(nfloors, nlifts int) (floors []*Floor, lifts []*Lift) {
	lifts = make([]*Lift, nlifts+1)
	floors = make([]*Floor, nfloors+1)

	for i := 1; i <= nlifts; i++ {
		controller := NewController()
		lift := NewLift(i, controller, floors)
		controller.Lift = lift
		lifts[i] = lift
	}

	for i := 1; i <= nfloors; i++ {
		floors[i] = NewFloor(i, lifts)
	}

	return
}

func main() {
	rand.Seed(time.Now().UnixNano())
	log.Println("Insert your code here")

	// an example of how to use the components
	floors, lifts := NewBuilding(10, 2)
	floors[9].Call <- true
	time.Sleep(time.Second * 5 / 2)
	floors[10].Call <- true

	time.Sleep(time.Second / 2)
	lifts[1].Call <- 3
	lifts[2].Call <- 2
	lifts[1].Call <- 8
	lifts[2].Call <- 6

	time.Sleep(100 * time.Second)
}
