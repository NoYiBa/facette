package library

import (
	"fmt"
	"github.com/nu7hatch/gouuid"
	"regexp"
	"strings"
)

const (
	// GraphTypeArea represents an area graph type.
	GraphTypeArea = iota
	// GraphTypeLine represents a line graph type.
	GraphTypeLine
)

const (
	// StackModeNone represents a null stack mode.
	StackModeNone = iota
	// StackModeNormal represents a normal stack mode.
	StackModeNormal
	// StackModePercent represents a percentage stack mode.
	StackModePercent
)

// Serie represents a instance of a Graph serie.
type Serie struct {
	Name   string `json:"name"`
	Origin string `json:"origin"`
	Source string `json:"source"`
	Metric string `json:"metric"`
}

// OperGroup represents a subset of Stack entry.
type OperGroup struct {
	Name    string            `json:"name"`
	Type    int               `json:"type"`
	Series  []*Serie          `json:"series"`
	Options map[string]string `json:"options"`
}

// Stack represents a set of OperGroup entries in a Graph instance.
type Stack struct {
	Name   string       `json:"name"`
	Groups []*OperGroup `json:"groups"`
}

// Graph represents a graph entry in a Library.
type Graph struct {
	Item
	Type      int      `json:"type"`
	StackMode int      `json:"stack_mode"`
	Stacks    []*Stack `json:"stacks"`
	Volatile  bool     `json:"-"`
}

func (library *Library) getTemplateID(origin, name string) (string, error) {
	var (
		err error
		id  *uuid.UUID
	)

	if id, err = uuid.NewV3(uuid.NamespaceURL, []byte(origin+name)); err != nil {
		return "", err
	}

	return id.String(), nil
}

// GetGraphTemplate gets a graph item.
func (library *Library) GetGraphTemplate(origin, source, template, filter string) (*Graph, error) {
	var (
		graph *Graph
		group *OperGroup
		id    string
		re    *regexp.Regexp
		stack *Stack
	)

	id = origin + "\x30" + template + "\x30" + filter

	if _, ok := library.Config.Origins[origin]; !ok {
		return nil, fmt.Errorf("unknown `%s' origin", origin)
	} else if _, ok := library.Config.Origins[origin].Templates[template]; !ok {
		return nil, fmt.Errorf("unknown `%s' template for `%s' origin", template, origin)
	}

	// Load template from filesystem if needed
	if !library.ItemExists(id, LibraryItemGraphTemplate) {
		graph = &Graph{
			Item:      Item{Name: template, Modified: library.Config.Origins[origin].Modified},
			StackMode: library.Config.Origins[origin].Templates[template].StackMode,
		}

		for i, tmplStack := range library.Config.Origins[origin].Templates[template].Stacks {
			stack = &Stack{Name: fmt.Sprintf("stack%d", i)}

			for groupName, tmplGroup := range tmplStack.Groups {
				if filter != "" {
					re = regexp.MustCompile(strings.Replace(tmplGroup.Pattern, "%s", regexp.QuoteMeta(filter), 1))
				} else {
					re = regexp.MustCompile(tmplGroup.Pattern)
				}

				group = &OperGroup{Name: groupName, Type: tmplGroup.Type}

				for metricName := range library.Catalog.Origins[origin].Sources[source].Metrics {
					if !re.MatchString(metricName) {
						continue
					}

					group.Series = append(group.Series, &Serie{
						Name:   metricName,
						Origin: origin,
						Source: source,
						Metric: metricName,
					})
				}

				if len(group.Series) == 1 {
					group.Series[0].Name = group.Name
				}

				stack.Groups = append(stack.Groups, group)
			}

			graph.Stacks = append(graph.Stacks, stack)
		}

		graph.ID = id
		library.TemplateGraphs[id] = graph
	}

	return library.TemplateGraphs[id], nil
}
