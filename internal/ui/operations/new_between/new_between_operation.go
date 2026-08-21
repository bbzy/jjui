package new_between

import (
	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/render"
)

var _ operations.Operation = (*Operation)(nil)
var _ common.Focusable = (*Operation)(nil)
var _ common.ScopeProvider = (*Operation)(nil)
var _ common.ScopeHandler = (*Operation)(nil)

type Operation struct {
	context     *appContext.MainContext
	insertAfter jj.SelectedRevisions
	targets     jj.SelectedRevisions
	target      intents.NewBetweenTarget
	current     *jj.Commit
}

func (o *Operation) IsFocused() bool {
	return true
}

func (o *Operation) Scopes() []common.Scope {
	return []common.Scope{
		{
			Name:    actions.ScopeNewBetween,
			Leak:    common.LeakAll,
			Handler: o,
		},
	}
}

func (o *Operation) Init() tea.Cmd {
	return nil
}

func (o *Operation) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case common.SelectionChangedMsg:
		selected, ok := msg.Item.(common.SelectedRevision)
		if !ok {
			return nil
		}
		o.current = &jj.Commit{ChangeId: selected.ChangeId, CommitId: selected.CommitId}
		return nil
	case intents.Intent:
		cmd, _ := o.HandleIntent(msg)
		return cmd
	default:
		return nil
	}
}

func (o *Operation) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch intent := intent.(type) {
	case intents.Cancel:
		return common.Close, true
	case intents.Apply:
		targets := o.effectiveTargets()
		if len(o.insertAfter.Revisions) == 0 && len(targets.Revisions) == 0 {
			return nil, true
		}
		return tea.Sequence(
			common.Close,
			o.context.RunCommand(o.newCommand(targets), common.RefreshAndSelect("@")),
		), true
	case intents.NewBetweenToggleInsertBefore:
		o.targets = o.targets.Toggle(o.current)
		return nil, true
	case intents.NewBetweenSetTarget:
		if intent.Target == intents.NewBetweenTargetAfter || intent.Target == intents.NewBetweenTargetBefore {
			o.target = intent.Target
		}
		return nil, true
	}
	return nil, false
}

func (o *Operation) ViewRect(dl *render.DisplayContext, box layout.Box) {}

func (o *Operation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	if pos != operations.RenderBeforeChangeId {
		return ""
	}

	sourceMarkerStyle := common.DefaultPalette.Get("new", "", "source_marker", false)
	targetMarkerStyle := common.DefaultPalette.Get("new", "", "target_marker", false)
	if o.insertAfter.Contains(commit) {
		return sourceMarkerStyle.Render("<< after this >>")
	}
	if !o.effectiveTargets().Contains(commit) {
		return ""
	}
	if o.target == intents.NewBetweenTargetAfter {
		return targetMarkerStyle.Render("<< after this >>")
	}
	return targetMarkerStyle.Render("<< before this >>")
}

func (o *Operation) Name() string {
	return "new.between"
}

func New(context *appContext.MainContext, insertAfter jj.SelectedRevisions, current *jj.Commit) *Operation {
	return &Operation{
		context:     context,
		insertAfter: insertAfter,
		target:      intents.NewBetweenTargetBefore,
		current:     current,
	}
}

func (o *Operation) effectiveTargets() jj.SelectedRevisions {
	if len(o.targets.Revisions) > 0 {
		return o.targets
	}
	return jj.NewSelectedRevisions(o.current)
}

func (o *Operation) newCommand(targets jj.SelectedRevisions) jj.CommandArgs {
	// When the fixed anchor and target are the same revision, either target direction
	// describes a normal child commit. Using --insert-after would also rebase its children.
	if len(o.insertAfter.Revisions) == 1 && len(targets.Revisions) == 1 && o.insertAfter.Revisions[0].Equal(targets.Revisions[0]) {
		return jj.New(o.insertAfter)
	}

	if o.target == intents.NewBetweenTargetAfter {
		return jj.NewInsert(combineRevisions(o.insertAfter, targets), jj.NewSelectedRevisions())
	}
	return jj.NewInsert(o.insertAfter, targets)
}

func combineRevisions(first jj.SelectedRevisions, second jj.SelectedRevisions) jj.SelectedRevisions {
	combined := jj.NewSelectedRevisions(first.Revisions...)
	for _, revision := range second.Revisions {
		if !combined.Contains(revision) {
			combined = combined.Add(revision)
		}
	}
	return combined
}
