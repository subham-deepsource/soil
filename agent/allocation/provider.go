package allocation

import (
	"encoding/json"
	"fmt"
	"github.com/da-moon/soil/manifest"
	"github.com/mitchellh/copystructure"
	"io"
	"strings"
)

const providerSpecPrefix = "### PROVIDER "

// Allocation providers
type ProviderSlice []*Provider

func (s *ProviderSlice) GetEmpty(paths SystemPaths) (empty Asset) {
	empty = &Provider{}
	return
}

func (s *ProviderSlice) GetVersionPrefix(v string) (p string) {
	p = providerSpecPrefix
	return
}

func (s *ProviderSlice) FromManifest(pod manifest.Pod, env manifest.FlatMap) (err error) {
	for _, decl := range pod.Providers {
		// clone provider
		v, _ := copystructure.Copy(decl)
		provider := Provider(v.(manifest.Provider))
		*s = append(*s, &provider)
	}
	return
}

func (s *ProviderSlice) AppendItem(v Asset) {
	*s = append(*s, v.(*Provider))
}

type Provider manifest.Provider

func (p *Provider) GetID(parent ...string) string {
	return strings.Join(append(parent, p.Name), ".")
}

// Restore state from header line
func (p *Provider) UnmarshalSpec(line string, spec Spec, paths SystemPaths) (err error) {
	err = json.Unmarshal([]byte(strings.TrimPrefix(line, providerSpecPrefix)), p)
	return
}

func (p *Provider) MarshalSpec(w io.Writer) (err error) {
	if _, err = fmt.Fprintf(w, "%s", providerSpecPrefix); err != nil {
		return
	}
	err = json.NewEncoder(w).Encode(p)
	return
}

func (p *Provider) Clone() (res *Provider) {
	r1, _ := copystructure.Copy(p)
	res = r1.(*Provider)
	return
}
