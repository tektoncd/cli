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
	successfulRelatedDeletes []string

	kind        string
	relatedKind string

	delete func(string) error

	listRelated   func(string) ([]string, error)
	deleteRelated func(string) error
}

// New returns a Deleter that will delete resources of kind with the given
// delete func when Execute is called.
func New(kind string, deleteFunc func(string) error) *Deleter {
	return &Deleter{
		kind:   kind,
		delete: deleteFunc,
	}
}

// WithRelated tells this Deleter that it should also delete related resources
// when Execute is called. Related resources will be of given kind, the names of
// those resources must be provided by listFunc and each related resource will be
// passed to deleteFunc for deletion.
func (d *Deleter) WithRelated(kind string, listFunc func(string) ([]string, error), deleteFunc func(string) error) {
	d.relatedKind = kind
	d.listRelated = listFunc
	d.deleteRelated = deleteFunc
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
	if d.relatedKind != "" && d.listRelated != nil && d.deleteRelated != nil {
		for _, name := range d.successfulDeletes {
			d.deleteRelatedList(streams, name)
		}
	}
	d.printSuccesses(streams)
	return multierr.Combine(d.errors...)
}

// deleteRelatedList gets the list of resources related to resourceName using the
// provided listFunc and then calls the deleteRelated func for each relation.
func (d *Deleter) deleteRelatedList(streams *cli.Stream, resourceName string) {
	if related, err := d.listRelated(resourceName); err != nil {
		d.printAndAddError(streams, err)
	} else {
		for _, subresource := range related {
			if err := d.deleteRelated(subresource); err != nil {
				err = fmt.Errorf("failed to delete %s %q: %s", strings.ToLower(d.relatedKind), subresource, err)
				d.printAndAddError(streams, err)
			} else {
				d.successfulRelatedDeletes = append(d.successfulRelatedDeletes, subresource)
			}
		}
	}
}

// printSuccesses writes success messages to the provided stdout stream.
func (d *Deleter) printSuccesses(streams *cli.Stream) {
	if len(d.successfulRelatedDeletes) > 0 {
		fmt.Fprintf(streams.Out, "%ss deleted: %s\n", d.relatedKind, names.QuotedList(d.successfulRelatedDeletes))
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
