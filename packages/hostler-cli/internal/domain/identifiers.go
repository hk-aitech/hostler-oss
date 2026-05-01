package domain

// TaskID — Task identifier value object (T001~T999 format).
type TaskID string

func (t TaskID) String() string { return string(t) }
func (t TaskID) IsZero() bool   { return t == "" }

// SprintID — Sprint identifier value object (sprint-NN format).
type SprintID string

func (s SprintID) String() string { return string(s) }
func (s SprintID) IsZero() bool   { return s == "" }
