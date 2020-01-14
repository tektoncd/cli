package deleter

import (
	"fmt"
	"strings"

	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/helper/names"
	"go.uber.org/multierr"
)

// Deleter encapsulates behaviour around deleting resources and their relations.
// While actually performing a deletion is left to calling code, this helper
// type standardizes the sequencing, messaging and error handling related to
// deletions.
type Deleter struct {
	errors                   []error
	successfulDeletes        []string
	successfulRelatedDeletes map[string][]string

	kind string

	delete func(string) error

	relatedKinds map[string]relatedFuncs
}

// relatedFuncs are the funcs needed when deleting related resources.
type relatedFuncs struct {
	list   func(string) ([]string, error)
	delete func(string) error
}

// New returns a Deleter that will delete resources of kind with the given
// delete func when Execute is called.
func New(kind string, deleteFunc func(string) error) *Deleter {
	return &Deleter{
		kind:                     kind,
		delete:                   deleteFunc,
		relatedKinds:             make(map[string]relatedFuncs),
		successfulRelatedDeletes: make(map[string][]string),
	}
}

// WithRelated tells this Deleter that it should also delete related resources
// when Execute is called. Related resources will be of given kind, the names of
// those resources must be provided by listFunc and each related resource will be
// passed to deleteFunc for deletion.
func (d *Deleter) WithRelated(kind string, listFunc func(string) ([]string, error), deleteFunc func(string) error) {
	d.relatedKinds[kind] = relatedFuncs{
		list:   listFunc,
		delete: deleteFunc,
	}
}

// Execute performs the deletion of resources and relations. Errors are aggregated
// and returned at the end of the func.
func (d *Deleter) Execute(streams *cli.Stream, resourceNames []string) error {
	for _, name := range resourceNames {
		if err := d.delete(name); err != nil {
			d.printAndAddError(streams, fmt.Errorf("failed to delete %s %q: %s", strings.ToLower(d.kind), name, err))
		} else {
			d.successfulDeletes = append(d.successfulDeletes, name)
		}
	}
	if len(d.relatedKinds) != 0 {
		for _, name := range d.successfulDeletes {
			d.deleteAllRelated(streams, name)
		}
	}
	d.printSuccesses(streams)
	return multierr.Combine(d.errors...)
}

// deleteAllRelated gets the list of each resource related to resourceName using the
// provided list func and then calls the delete func for each relation.
func (d *Deleter) deleteAllRelated(streams *cli.Stream, resourceName string) {
	for kind, funcs := range d.relatedKinds {
		if related, err := funcs.list(resourceName); err != nil {
			err = fmt.Errorf("fetching list of %ss related to %q failed: %s", strings.ToLower(kind), resourceName, err)
			d.printAndAddError(streams, err)
		} else {
			for _, subresource := range related {
				if err := funcs.delete(subresource); err != nil {
					err = fmt.Errorf("deleting relation of %q, failed to delete %s %q: %s", resourceName, strings.ToLower(kind), subresource, err)
					d.printAndAddError(streams, err)
				} else {
					d.successfulRelatedDeletes[kind] = append(d.successfulRelatedDeletes[kind], subresource)
				}
			}
		}
	}
}

// printSuccesses writes success messages to the provided stdout stream.
func (d *Deleter) printSuccesses(streams *cli.Stream) {
	if len(d.successfulRelatedDeletes) > 0 {
		for kind, relatedNames := range d.successfulRelatedDeletes {
			fmt.Fprintf(streams.Out, "%ss deleted: %s\n", kind, names.QuotedList(relatedNames))
		}
	}
	if len(d.successfulDeletes) > 0 {
		fmt.Fprintf(streams.Out, "%ss deleted: %s\n", d.kind, names.QuotedList(d.successfulDeletes))
	}
}

// printAndAddError prints the given error to the given stderr stream and
// adds that error to the list of accumulated errors that have occurred
// during execution.
func (d *Deleter) printAndAddError(streams *cli.Stream, err error) {
	d.errors = append(d.errors, err)
	fmt.Fprintf(streams.Err, "%s\n", err)
}
