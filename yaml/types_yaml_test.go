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
