package capacity

// ResourceType describes catalog constraints for a divisible or discrete resource.
type ResourceType struct {
	ID         int
	Name       string
	Unit       string
	MinAmount  int64
	StepAmount int64
	MaxAmount  *int64
	Discrete   bool
}

// ReserveLine is a validated resource allocation line within a reserve/create transaction.
type ReserveLine struct {
	ResourceTypeID int
	Name           string
	Amount         int64
}