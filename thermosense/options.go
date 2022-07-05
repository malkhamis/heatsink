package thermosense

// Option is used to pass optional parameters to the Sensor factory function
type Option func(*Sensor)

// OptName sets the name of the sensor. if name is empty, it is set to the default value
//
// (default: filename)
func OptName(name string) Option {
	return func(s *Sensor) {
		if name != "" {
			s.name = name
		}
	}
}
