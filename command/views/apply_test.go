package views

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/command/arguments"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/states"
	"github.com/zclconf/go-cty/cty"
)

// This test is mostly because I am paranoid about having two consecutive
// boolean arguments.
func TestApply_new(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	defer done(t)
	v := NewApply(arguments.ViewHuman, false, true, NewView(streams))
	hv, ok := v.(*ApplyHuman)
	if !ok {
		t.Fatalf("unexpected return type %t", v)
	}

	if hv.destroy != false {
		t.Fatalf("unexpected destroy value")
	}

	if hv.inAutomation != true {
		t.Fatalf("unexpected inAutomation value")
	}
}

// Basic test coverage of Outputs, since most of its functionality is tested
// elsewhere.
func TestApplyHuman_outputs(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	v := NewApply(arguments.ViewHuman, false, false, NewView(streams))

	v.Outputs(map[string]*states.OutputValue{
		"foo": {Value: cty.StringVal("secret")},
	})

	got := done(t).Stdout()
	for _, want := range []string{"Outputs:", `foo = "secret"`} {
		if !strings.Contains(got, want) {
			t.Errorf("wrong result\ngot:  %q\nwant: %q", got, want)
		}
	}
}

// Outputs should do nothing if there are no outputs to render.
func TestApplyHuman_outputsEmpty(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	v := NewApply(arguments.ViewHuman, false, false, NewView(streams))

	v.Outputs(map[string]*states.OutputValue{})

	got := done(t).Stdout()
	if got != "" {
		t.Errorf("output should be empty, but got: %q", got)
	}
}

// Ensure that the correct view type and in-automation settings propagate to the
// Operation view.
func TestApplyHuman_operation(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	defer done(t)
	v := NewApply(arguments.ViewHuman, false, true, NewView(streams)).Operation()
	if hv, ok := v.(*OperationHuman); !ok {
		t.Fatalf("unexpected return type %t", v)
	} else if hv.inAutomation != true {
		t.Fatalf("unexpected inAutomation value on Operation view")
	}
}

// This view is used for both apply and destroy commands, so the help output
// needs to cover both.
func TestApplyHuman_help(t *testing.T) {
	testCases := map[string]bool{
		"apply":   false,
		"destroy": true,
	}

	for name, destroy := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			v := NewApply(arguments.ViewHuman, destroy, false, NewView(streams))
			v.HelpPrompt()
			got := done(t).Stderr()
			if !strings.Contains(got, name) {
				t.Errorf("wrong result\ngot:  %q\nwant: %q", got, name)
			}
		})
	}
}

// Hooks and ResourceCount are tangled up and easiest to test together.
func TestApplyHuman_resourceCount(t *testing.T) {
	testCases := map[string]struct {
		destroy bool
		want    string
	}{
		"apply": {
			false,
			"Apply complete! Resources: 1 added, 2 changed, 3 destroyed.",
		},
		"destroy": {
			true,
			"Destroy complete! Resources: 3 destroyed.",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			v := NewApply(arguments.ViewHuman, tc.destroy, false, NewView(streams))
			hooks := v.Hooks()

			var count *countHook
			for _, hook := range hooks {
				if ch, ok := hook.(*countHook); ok {
					count = ch
				}
			}
			if count == nil {
				t.Fatalf("expected Hooks to include a countHook: %#v", hooks)
			}

			count.Added = 1
			count.Changed = 2
			count.Removed = 3

			v.ResourceCount()

			got := done(t).Stdout()
			if !strings.Contains(got, tc.want) {
				t.Errorf("wrong result\ngot:  %q\nwant: %q", got, tc.want)
			}
		})
	}
}
