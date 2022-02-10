package config

import (
	"context"
	"strings"
	"testing"

	"github.com/edaniels/golog"
	"go.viam.com/test"
)

func TestFromReaderValidate(t *testing.T) {
	logger := golog.NewTestLogger(t)
	_, err := FromReader(context.Background(), "somepath", strings.NewReader(""), logger)
	test.That(t, err, test.ShouldNotBeNil)
	test.That(t, err.Error(), test.ShouldContainSubstring, "unexpected end")

	_, err = FromReader(context.Background(), "somepath", strings.NewReader(`{"cloud": 1}`), logger)
	test.That(t, err, test.ShouldNotBeNil)
	test.That(t, err.Error(), test.ShouldContainSubstring, "unmarshal")

	conf, err := FromReader(context.Background(), "somepath", strings.NewReader(`{}`), logger)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, conf, test.ShouldResemble, &Config{
		ConfigFilePath: "somepath",
		Network:        NetworkConfig{NetworkConfigData: NetworkConfigData{BindAddress: "localhost:8080", BindAddressDefaultSet: true}},
	})

	_, err = FromReader(context.Background(), "somepath", strings.NewReader(`{"cloud": {}}`), logger)
	test.That(t, err, test.ShouldNotBeNil)
	test.That(t, err.Error(), test.ShouldContainSubstring, `"id" is required`)

	_, err = FromReader(context.Background(), "somepath", strings.NewReader(`{"components": [{}]}`), logger)
	test.That(t, err, test.ShouldNotBeNil)
	test.That(t, err.Error(), test.ShouldContainSubstring, `components.0`)
	test.That(t, err.Error(), test.ShouldContainSubstring, `"name" is required`)

	conf, err = FromReader(context.Background(), "somepath", strings.NewReader(`{"components": [{"name": "foo", "type": "arm"}]}`), logger)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, conf, test.ShouldResemble, &Config{
		ConfigFilePath: "somepath",
		Components: []Component{
			{
				Name: "foo",
				Type: ComponentTypeArm,
			},
		},
		Network: NetworkConfig{NetworkConfigData: NetworkConfigData{BindAddress: "localhost:8080", BindAddressDefaultSet: true}},
	})

	badComponentMapConverter := func() {
		RegisterComponentAttributeMapConverter(ComponentType("somecomponent"), "somemodel", func(attributes AttributeMap) (interface{}, error) {
			return &conf, nil
		}, nil)
	}
	test.That(t, badComponentMapConverter, test.ShouldPanic)

	badServiceMapConverter := func() {
		RegisterServiceAttributeMapConverter(ServiceType("someservice"), func(attributes AttributeMap) (interface{}, error) {
			return &conf, nil
		}, nil)
	}
	test.That(t, badServiceMapConverter, test.ShouldPanic)
}

func TestTransformAttributeMapToStruct(t *testing.T) {
	type myType struct {
		A          string            `json:"a"`
		B          string            `json:"b"`
		Attributes map[string]string `json:"attributes"`
	}

	var mt myType
	attrs := AttributeMap{
		"a": "1",
		"b": "2",
		"c": "3",
		"d": "4",
		"e": 5,
	}
	transformed, err := TransformAttributeMapToStruct(&mt, attrs)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, transformed, test.ShouldResemble, &myType{
		A: "1",
		B: "2",
		Attributes: map[string]string{
			"c": "3",
			"d": "4",
		},
	})

	mt = myType{Attributes: map[string]string{}}
	transformed, err = TransformAttributeMapToStruct(&mt, attrs)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, transformed, test.ShouldResemble, &myType{
		A: "1",
		B: "2",
		Attributes: map[string]string{
			"c": "3",
			"d": "4",
		},
	})

	type myExtendedType struct {
		A          string       `json:"a"`
		B          string       `json:"b"`
		Attributes AttributeMap `json:"attributes"`
	}

	var met myExtendedType
	transformed, err = TransformAttributeMapToStruct(&met, attrs)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, transformed, test.ShouldResemble, &myExtendedType{
		A: "1",
		B: "2",
		Attributes: AttributeMap{
			"c": "3",
			"d": "4",
			"e": 5,
		},
	})

	met = myExtendedType{Attributes: AttributeMap{}}
	transformed, err = TransformAttributeMapToStruct(&met, attrs)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, transformed, test.ShouldResemble, &myExtendedType{
		A: "1",
		B: "2",
		Attributes: AttributeMap{
			"c": "3",
			"d": "4",
			"e": 5,
		},
	})
}
