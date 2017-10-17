package yaml

import (
	"testing"

	"gopkg.in/yaml.v2"

	"github.com/stretchr/testify/assert"
)

type StructMemStringorInt struct {
	Foo MemStringorInt
}

func TestMemStringorIntYaml(t *testing.T) {
	for _, str := range []string{`{foo: 1048576}`, `{foo: "1048576"}`, `{foo: "1m"}`} {
		s := StructMemStringorInt{}
		yaml.Unmarshal([]byte(str), &s)

		assert.Equal(t, MemStringorInt(1048576), s.Foo)

		d, err := yaml.Marshal(&s)
		assert.Nil(t, err)

		s2 := StructMemStringorInt{}
		yaml.Unmarshal(d, &s2)

		assert.Equal(t, MemStringorInt(1048576), s2.Foo)
	}
}

type StructDurationStringorInt struct {
	Duration DurationStringorInt
}

func TestDurationStringorIntYAML(t *testing.T) {
	for _, str := range []string{`{duration: 90}`, `{duration: "90s"}`, `{duration: "1m30s"}`} {
		s := StructDurationStringorInt{}
		yaml.Unmarshal([]byte(str), &s)

		assert.Equal(t, DurationStringorInt(90), s.Duration)

		d, err := yaml.Marshal(&s)
		assert.Nil(t, err)

		s2 := StructDurationStringorInt{}
		yaml.Unmarshal(d, &s2)

		assert.Equal(t, DurationStringorInt(90), s2.Duration)
	}
}

type StructStringorOctalInteger struct {
	Mode StringorOctalInt
}

func TestStringorOctalIntYAML(t *testing.T) {
	testCases := map[string]string{
		`{mode: 1777}`:   "1777",
		`{mode: 0777}`:   "777",
		`{mode: "777"}`:  "777",
		`{mode: "0777"}`: "777",
		`{mode: "1777"}`: "1777",
	}
	failTestCases := map[string]string{
		`{mode: "292"}`:  "",
		`{mode: "0292"}`: "",
	}
	for test, expected := range testCases {
		s := StructStringorOctalInteger{}
		yaml.Unmarshal([]byte(test), &s)

		assert.Equal(t, StringorOctalInt(expected), s.Mode)

		d, err := yaml.Marshal(&s)
		assert.Nil(t, err)

		s2 := StructStringorOctalInteger{}
		yaml.Unmarshal(d, &s2)

		assert.Equal(t, StringorOctalInt(expected), s.Mode)
	}

	for test := range failTestCases {
		s := StructStringorOctalInteger{}
		err := yaml.Unmarshal([]byte(test), &s)
		assert.NotNil(t, err)
	}
}
