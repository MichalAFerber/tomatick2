package sessions_test

import (
	"fmt"

	"github.com/MichalAFerber/tomatick2/internal/sessions"
)

func ExampleParseDuration() {
	s, _ := sessions.ParseDuration("1h30m")
	fmt.Println(s)
	fmt.Println(sessions.FormatClock(s))
	// Output:
	// 5400
	// 1:30:00
}
