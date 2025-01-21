package flows

import (
	"slices"

	"github.com/kube-agent/kuery/pkg/flows/steps"
)

// Chain is a chain of steps.
type Chain interface {
	// Next returns the next step in the chain.
	// If there is no next step, or the chain reached the end, it returns nil.
	Next() steps.Step
	// PushNext pushes a step to the immediate next of the current step.
	// That is, Chain::Next() will return the pushed step.
	// If temporary is true, the step will be removed after its execution.
	PushNext(step steps.Step, temporary bool) Chain
	// Push pushes steps to the end of the chain.
	Push(steps []steps.Step) Chain
	// Reset resets the chain to the beginning.
	Reset() Chain
}

// NewChain creates a new chain with the given steps.
func NewChain(steps []steps.Step) Chain {
	removableSteps := make([]removableStep, len(steps))
	for i, step := range steps {
		removableSteps[i] = removableStep{
			Step:      step,
			temporary: false,
		}
	}

	return &chain{
		steps: removableSteps,
	}
}

type chain struct {
	steps            []removableStep
	currentStepIndex int
}

type removableStep struct {
	steps.Step
	temporary bool
}

// Next returns the next step in the chain.
// If there is no next step, or the chain reached the end, it returns nil.
func (c *chain) Next() steps.Step {
	if c.currentStepIndex >= len(c.steps) {
		return nil
	}

	step := c.steps[c.currentStepIndex]
	c.currentStepIndex++

	return step
}

// PushNext pushes a step to the immediate next of the current step.
// That is, Chain::Next() will return the pushed step.
// If temporary is true, the step will be removed after its execution.
func (c *chain) PushNext(step steps.Step, temporary bool) Chain {
	c.steps = slices.Insert(c.steps, c.currentStepIndex, removableStep{
		Step:      step,
		temporary: temporary,
	})

	return c
}

// Push pushes steps to the end of the chain.
func (c *chain) Push(steps []steps.Step) Chain {
	for _, step := range steps {
		c.steps = append(c.steps, removableStep{
			Step:      step,
			temporary: false,
		})
	}

	return c
}

// Reset resets the chain to the beginning.
func (c *chain) Reset() Chain {
	c.currentStepIndex = 0
	// remove temporary steps
	for i := 0; i < len(c.steps); i++ {
		if c.steps[i].temporary {
			c.steps = append(c.steps[:i], c.steps[i+1:]...)
			i--
		}
	}

	return c
}
