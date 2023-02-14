package main

import (
	"sync"
)

// Multiplexer receives values from an input channel and sends them to each of its output channels.
type Multiplexer struct {
	input    <-chan UpdateActuatorMessage
	outputs  map[chan UpdateActuatorMessage]bool
	outputsM sync.Mutex
}

// NewMultiplexer creates a new Multiplexer that reads from the given input channel.
func NewMultiplexer(input <-chan UpdateActuatorMessage) *Multiplexer {
	mux := &Multiplexer{
		input:   input,
		outputs: make(map[chan UpdateActuatorMessage]bool),
	}
	go mux.run()
	return mux
}

// Subscribe creates a new output channel and returns it for the caller to read from.
func (mux *Multiplexer) Subscribe(output chan UpdateActuatorMessage) {
	mux.outputsM.Lock()
	mux.outputs[output] = true
	mux.outputsM.Unlock()
}

// Unsubscribe removes an output channel from the set of output channels.
func (mux *Multiplexer) Unsubscribe(output chan UpdateActuatorMessage) {
	mux.outputsM.Lock()
	delete(mux.outputs, output)
	mux.outputsM.Unlock()
	close(output)
}

// run is the main loop of the Multiplexer.

/*
	Here is the explanation for the code above:

1. mux.input is the channel that is used to send messages to the multiplexer.
2. mux.outputs is the channel that holds all the subscribers.
3. The for loop will run as long as the channel mux.input is open.
4. We lock the mux.outputsM mutex.
5. We send the message we received from the input channel to all the subscribers.
6. We unlock the mux.outputsM mutex.
7. If the input channel is closed, we lock the mux.outputsM mutex.
8. We unsubscribe all the subscribers.
9. We unlock the mux.outputsM mutex.
*/
func (mux *Multiplexer) run() {
	for val := range mux.input {
		mux.outputsM.Lock()
		for output := range mux.outputs {
			select {
			case output <- val:
			// unsubscribe if the output channel is closed only and not necessarily full
			case <-output:
				mux.Unsubscribe(output)
			}
		}
		mux.outputsM.Unlock()
	}
	mux.outputsM.Lock()
	// shouldn't get to this point unless the input channel is closed
	// then we unsubscribe all the output channels
	for output := range mux.outputs {
		mux.Unsubscribe(output)
	}
	mux.outputsM.Unlock()
}
