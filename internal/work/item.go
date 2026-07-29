package work

// Item is a unit of work travelling through the pipeline.
// Attempt tracks how many times this app has failed, used to calculate retry backoff.
type Item struct {
	Apps    []int
	Attempt int
}
