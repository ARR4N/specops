package jump

// A Table is a slice of JUMPDEST labels. If passed to specops.PUSH(), the
// respective code locations will be concatenated and pushed. If all locations
// can be represented as a single byte then they will be, otherwise all will be
// represented as two bytes for concatenation.
type Table []Dest
