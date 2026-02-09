package execution

// Scheduler distributes tests across workers
type Scheduler interface {
	Schedule(tests []string, workerCount int) [][]string
}

// RoundRobinScheduler distributes tests evenly across workers
type RoundRobinScheduler struct{}

// NewRoundRobinScheduler creates a new RoundRobinScheduler
func NewRoundRobinScheduler() *RoundRobinScheduler {
	return &RoundRobinScheduler{}
}

// Schedule distributes tests evenly across workers using round-robin
func (s *RoundRobinScheduler) Schedule(tests []string, workerCount int) [][]string {
	if workerCount <= 0 {
		workerCount = 1
	}

	distribution := make([][]string, workerCount)
	for i := range distribution {
		distribution[i] = make([]string, 0)
	}

	for i, test := range tests {
		workerIndex := i % workerCount
		distribution[workerIndex] = append(distribution[workerIndex], test)
	}

	return distribution
}

