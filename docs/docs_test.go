package docs

import (
	"os"
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSpecIsTheContractFile(t *testing.T) {
	onDisk, err := os.ReadFile("openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if string(Spec()) != string(onDisk) {
		t.Error("Spec() differs from docs/openapi.yaml")
	}
}

// A relative server makes the docs page call the address it was opened on,
// so it works from localhost and 127.0.0.1 alike.
func TestSpecServerIsRelative(t *testing.T) {
	var spec struct {
		Servers []struct {
			URL string `yaml:"url"`
		} `yaml:"servers"`
	}
	if err := yaml.Unmarshal(Spec(), &spec); err != nil {
		t.Fatal(err)
	}
	var urls []string
	for _, s := range spec.Servers {
		urls = append(urls, s.URL)
	}
	if want := []string{"/"}; !reflect.DeepEqual(urls, want) {
		t.Errorf("servers = %q, want %q", urls, want)
	}
}
