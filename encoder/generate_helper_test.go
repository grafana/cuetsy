package encoder

import (
	"fmt"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

func TestDefaultGenerationLevel(t *testing.T) {
	cases := []struct {
		name        string
		inputCueV   string
		outputLevel int
	}{
		{
			name:        "no struct object level should be 1",
			inputCueV:   `I2_Number: number`,
			outputLevel: 1,
		},
		{
			name:        "struct object with defaults, level should be correctly calculated",
			inputCueV:   `I2_Number: number`,
			outputLevel: 1,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := cuecontext.New()
			i := ctx.CompileString(c.inputCueV, cue.Filename(c.name+".cue"))
			result, err := GetStructDefaultGenerationLevel(i)
			if err != nil {
				t.Fatal(err)
			}
			if result != c.outputLevel {
				t.Fatal(fmt.Errorf("Test case fails: Expected generation level %d, but has %d", c.outputLevel, result))
			}
		})
	}
}
