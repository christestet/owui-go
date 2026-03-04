package pipelines

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/christestet/owui-go/internal/api"
)

// Registration represents a normalized pipeline registration.
type Registration struct {
	RegistrationID string         `json:"registration_id"`
	URL            string         `json:"url"`
	URLIdx         int            `json:"url_idx"`
	PipeCount      int            `json:"pipe_count"`
	Raw            map[string]any `json:"raw"`
	PipesError     string         `json:"pipes_error,omitempty"`
}

// Pipe represents a normalized pipe.
type Pipe struct {
	PipeID          string         `json:"pipe_id"`
	Name            string         `json:"name"`
	Description     string         `json:"description,omitempty"`
	RegistrationID  string         `json:"registration_id"`
	RegistrationURL string         `json:"registration_url"`
	URLIdx          int            `json:"url_idx"`
	Raw             map[string]any `json:"raw"`
}

// Inventory is a normalized view over untyped pipelines API responses.
type Inventory struct {
	Registrations    []Registration            `json:"registrations"`
	Pipes            []Pipe                    `json:"pipes"`
	RawRegistrations any                       `json:"raw_registrations"`
	RawPipesByURLIdx map[string]any            `json:"raw_pipes_by_url_idx"`
	PipeIndex        map[string][]Pipe         `json:"-"`
	RegIndex         map[string][]Registration `json:"-"`
}

func toString(v any) string {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	default:
		return ""
	}
}

func toInt(v any) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case int32:
		return int(x), true
	case int64:
		return int(x), true
	case float64:
		return int(x), true
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(x))
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

func toMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func extractObjectArray(raw any) []map[string]any {
	switch x := raw.(type) {
	case []any:
		out := make([]map[string]any, 0, len(x))
		for _, item := range x {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	case map[string]any:
		for _, key := range []string{"items", "data", "pipelines", "pipes", "results"} {
			if arr, ok := x[key].([]any); ok {
				out := make([]map[string]any, 0, len(arr))
				for _, item := range arr {
					if m, ok := item.(map[string]any); ok {
						out = append(out, m)
					}
				}
				return out
			}
		}
		return []map[string]any{x}
	default:
		return nil
	}
}

func normalizeRegistration(raw map[string]any, fallbackIdx int) Registration {
	urlIdx, ok := toInt(raw["urlIdx"])
	if !ok {
		urlIdx, ok = toInt(raw["url_idx"])
	}
	if !ok {
		urlIdx, ok = toInt(raw["idx"])
	}
	if !ok {
		urlIdx = fallbackIdx
	}

	id := toString(raw["id"])
	if id == "" {
		id = toString(raw["pipeline_id"])
	}
	if id == "" {
		id = toString(raw["name"])
	}
	if id == "" {
		id = fmt.Sprintf("registration-%d", urlIdx)
	}

	url := toString(raw["url"])
	if url == "" {
		url = toString(raw["base_url"])
	}
	if url == "" {
		url = toString(raw["endpoint"])
	}

	return Registration{
		RegistrationID: id,
		URL:            url,
		URLIdx:         urlIdx,
		Raw:            raw,
	}
}

func normalizePipe(raw map[string]any, reg Registration) Pipe {
	pipeID := toString(raw["id"])
	if pipeID == "" {
		pipeID = toString(raw["pipe_id"])
	}
	if pipeID == "" {
		pipeID = toString(raw["pipeline_id"])
	}
	if pipeID == "" {
		pipeID = toString(raw["name"])
	}

	name := toString(raw["name"])
	if name == "" {
		name = toString(raw["title"])
	}
	if name == "" {
		name = pipeID
	}

	desc := toString(raw["description"])

	return Pipe{
		PipeID:          pipeID,
		Name:            name,
		Description:     desc,
		RegistrationID:  reg.RegistrationID,
		RegistrationURL: reg.URL,
		URLIdx:          reg.URLIdx,
		Raw:             raw,
	}
}

func buildInventory(ctx context.Context, client *api.Client, warn io.Writer) (*Inventory, error) {
	regsRaw, err := client.ListPipelineRegistrationsRaw(ctx)
	if err != nil {
		return nil, err
	}

	regsInput := extractObjectArray(regsRaw)
	inv := &Inventory{
		RawRegistrations: regsRaw,
		RawPipesByURLIdx: make(map[string]any),
		PipeIndex:        make(map[string][]Pipe),
		RegIndex:         make(map[string][]Registration),
	}

	for i, raw := range regsInput {
		reg := normalizeRegistration(raw, i)
		inv.Registrations = append(inv.Registrations, reg)
	}

	sort.Slice(inv.Registrations, func(i, j int) bool {
		if inv.Registrations[i].URLIdx == inv.Registrations[j].URLIdx {
			return inv.Registrations[i].RegistrationID < inv.Registrations[j].RegistrationID
		}
		return inv.Registrations[i].URLIdx < inv.Registrations[j].URLIdx
	})

	for i := range inv.Registrations {
		reg := inv.Registrations[i]
		pipesRaw, err := client.ListPipesByURLIdxRaw(ctx, reg.URLIdx)
		if err != nil {
			inv.Registrations[i].PipesError = err.Error()
			if warn != nil {
				fmt.Fprintf(warn, "Warning: failed to list pipes for registration %q (urlIdx=%d): %v\n", reg.RegistrationID, reg.URLIdx, err)
			}
			continue
		}
		inv.RawPipesByURLIdx[strconv.Itoa(reg.URLIdx)] = pipesRaw

		pipeMaps := extractObjectArray(pipesRaw)
		for _, rawPipe := range pipeMaps {
			pipe := normalizePipe(rawPipe, reg)
			if pipe.PipeID == "" {
				continue
			}
			inv.Pipes = append(inv.Pipes, pipe)
			inv.PipeIndex[pipe.PipeID] = append(inv.PipeIndex[pipe.PipeID], pipe)
			inv.Registrations[i].PipeCount++
		}
	}

	for _, reg := range inv.Registrations {
		inv.RegIndex[reg.RegistrationID] = append(inv.RegIndex[reg.RegistrationID], reg)
	}

	sort.Slice(inv.Pipes, func(i, j int) bool {
		if inv.Pipes[i].PipeID == inv.Pipes[j].PipeID {
			return inv.Pipes[i].URLIdx < inv.Pipes[j].URLIdx
		}
		return inv.Pipes[i].PipeID < inv.Pipes[j].PipeID
	})

	return inv, nil
}

func resolvePipe(inv *Inventory, pipeID string, urlIdx *int) (*Pipe, error) {
	candidates := inv.PipeIndex[pipeID]
	if len(candidates) == 0 {
		return nil, fmt.Errorf("pipe %q not found", pipeID)
	}
	if urlIdx != nil {
		for i := range candidates {
			if candidates[i].URLIdx == *urlIdx {
				return &candidates[i], nil
			}
		}
		return nil, fmt.Errorf("pipe %q not found for urlIdx=%d", pipeID, *urlIdx)
	}
	if len(candidates) > 1 {
		idxs := make([]string, 0, len(candidates))
		for _, c := range candidates {
			idxs = append(idxs, fmt.Sprintf("%d", c.URLIdx))
		}
		return nil, fmt.Errorf("pipe_id %q is ambiguous across urlIdx [%s]; pass --url-idx", pipeID, strings.Join(idxs, ","))
	}
	return &candidates[0], nil
}

func resolveRegistration(inv *Inventory, registrationID string, urlIdx *int) (*Registration, error) {
	candidates := inv.RegIndex[registrationID]
	if len(candidates) == 0 {
		return nil, fmt.Errorf("registration %q not found", registrationID)
	}
	if urlIdx != nil {
		for i := range candidates {
			if candidates[i].URLIdx == *urlIdx {
				return &candidates[i], nil
			}
		}
		return nil, fmt.Errorf("registration %q not found for urlIdx=%d", registrationID, *urlIdx)
	}
	if len(candidates) > 1 {
		idxs := make([]string, 0, len(candidates))
		for _, c := range candidates {
			idxs = append(idxs, fmt.Sprintf("%d", c.URLIdx))
		}
		return nil, fmt.Errorf("registration %q is ambiguous across urlIdx [%s]; pass --url-idx", registrationID, strings.Join(idxs, ","))
	}
	return &candidates[0], nil
}

func nextFreeURLIdx(inv *Inventory) int {
	if len(inv.Registrations) == 0 {
		return 0
	}
	max := inv.Registrations[0].URLIdx
	for _, r := range inv.Registrations[1:] {
		if r.URLIdx > max {
			max = r.URLIdx
		}
	}
	return max + 1
}

func filterInventory(inv *Inventory, q string) *Inventory {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return inv
	}

	filtered := &Inventory{
		RawRegistrations: inv.RawRegistrations,
		RawPipesByURLIdx: inv.RawPipesByURLIdx,
		PipeIndex:        make(map[string][]Pipe),
		RegIndex:         make(map[string][]Registration),
	}

	for _, r := range inv.Registrations {
		if strings.Contains(strings.ToLower(r.RegistrationID), q) || strings.Contains(strings.ToLower(r.URL), q) {
			filtered.Registrations = append(filtered.Registrations, r)
			filtered.RegIndex[r.RegistrationID] = append(filtered.RegIndex[r.RegistrationID], r)
		}
	}

	for _, p := range inv.Pipes {
		if strings.Contains(strings.ToLower(p.PipeID), q) || strings.Contains(strings.ToLower(p.Name), q) {
			filtered.Pipes = append(filtered.Pipes, p)
			filtered.PipeIndex[p.PipeID] = append(filtered.PipeIndex[p.PipeID], p)
		}
	}

	return filtered
}
