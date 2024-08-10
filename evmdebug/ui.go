package evmdebug

import (
	"fmt"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// RunTerminalUI starts a UI that controls the Debugger and displays opcodes,
// memory, stack etc. Because of the current Debugger limitation of a single
// call frame, only that exact Contract can be displayed. The callData is
// assumed to be the same as passed to the execution environment.
//
// As the Debugger only has access via a vm.EVMLogger, it can't retrieve the
// final result. The `results` argument MUST return the returned buffer / error
// after d.Done() returns true.
func (d *Debugger) RunTerminalUI(callData []byte, results func() ([]byte, error), contract *vm.Contract) error {
	t := &termDBG{
		Debugger: d,
		results:  results,
	}
	t.initComponents()
	t.initApp()
	t.populateCallData(callData)
	t.populateCode(contract)
	return t.app.Run()
}

type termDBG struct {
	*Debugger
	app *tview.Application

	stack, memory    *tview.List
	callData, result *tview.TextView

	code         *tview.List
	pcToCodeItem map[uint64]int

	results func() ([]byte, error)
}

func (*termDBG) styleBox(b *tview.Box, title string) *tview.Box {
	return b.SetBorder(true).
		SetTitle(title).
		SetTitleAlign(tview.AlignLeft)
}

func (t *termDBG) initComponents() {
	const codeTitle = "Code"
	for title, l := range map[string]**tview.List{
		"Stack":   &t.stack,
		"Memory":  &t.memory,
		codeTitle: &t.code,
	} {
		*l = tview.NewList()
		(*l).ShowSecondaryText(false).
			SetSelectedFocusOnly(title != codeTitle)
		t.styleBox((*l).Box, title)
	}

	t.code.SetChangedFunc(func(int, string, string, rune) {
		t.onStep()
	})

	for title, v := range map[string]**tview.TextView{
		"calldata": &t.callData,
		"Result":   &t.result,
	} {
		*v = tview.NewTextView()
		t.styleBox((*v).Box, title)
	}
}

func (t *termDBG) initApp() {
	t.app = tview.NewApplication().SetRoot(t.createLayout(), true)
	t.app.SetInputCapture(t.inputCapture)
}

func (t *termDBG) createLayout() tview.Primitive {
	// Components have borders of 2, which need to be accounted for in absolute
	// dimensions.
	const (
		hStack = 2 + 16
		wStack = 2 + 5 + 64 // w/ 4-digit decimal label & space
		wMem   = 2 + 3 + 64 // w/ 2-digit hex offset & space
	)
	middle := tview.NewFlex().
		AddItem(t.code, 0, 1, false).
		AddItem(t.stack, wStack, 0, false).
		AddItem(t.memory, wMem, 0, false)

	root := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(t.callData, 0, 1, false).
		AddItem(middle, hStack, 0, false).
		AddItem(t.result, 0, 1, false)

	t.styleBox(root.Box, "SPEC0PS").SetTitleAlign(tview.AlignCenter)

	return root
}

func (t *termDBG) populateCallData(cd []byte) {
	t.callData.SetText(fmt.Sprintf("%x", cd))
}

func (t *termDBG) populateCode(c *vm.Contract) {
	t.pcToCodeItem = make(map[uint64]int)

	var skip int
	for i, o := range c.Code {
		if skip > 0 {
			skip--
			continue
		}

		var text string
		switch op := vm.OpCode(o); {
		case op == vm.PUSH0:
			text = op.String()

		case op.IsPush():
			skip += int(op - vm.PUSH0)
			text = fmt.Sprintf("%s %#x", op.String(), c.Code[i+1:i+1+skip])

		default:
			text = op.String()
		}

		t.pcToCodeItem[uint64(i)] = t.code.GetItemCount()
		t.code.AddItem(text, "", 0, nil)
	}

	t.code.AddItem("--- END ---", "", 0, nil)
}

func (t *termDBG) highlightPC() {
	t.code.SetCurrentItem(t.pcToCodeItem[t.State().PC] + 1)
}

// onStep is triggered by t.code's ChangedFunc.
func (t *termDBG) onStep() {
	if !t.Done() {
		return
	}
	t.result.SetText(t.resultToDisplay())
}

func (t *termDBG) resultToDisplay() string {
	out, err := t.results()
	if err != nil {
		return fmt.Sprintf("ERROR: %v", err)
	}
	return fmt.Sprintf("%x", out)
}

func (t *termDBG) inputCapture(ev *tcell.EventKey) *tcell.EventKey {
	var propagate bool

	switch ev.Key() {
	case tcell.KeyCtrlC:
		t.app.Stop()
		return ev

	case tcell.KeyEnd:
		t.FastForward()
		t.highlightPC()

	case tcell.KeyEscape:
		if t.Done() {
			t.app.Stop()
		}
	} // switch ev.Key()

	switch ev.Rune() {
	case ' ':
		if !t.Done() {
			t.Step()
			t.highlightPC()
		}

	case 'q':
		if t.Done() {
			t.app.Stop()
		}
	} // switch ev.Rune()

	if t.State().Context != nil {
		t.populateStack()
		t.populateMemory()
	}

	if propagate {
		return ev
	}
	return nil
}

func (t *termDBG) populateStack() {
	stack := t.State().Context.StackData()

	t.stack.Clear()
	for i, n := 0, len(stack); i < n; i++ {
		item := t.State().StackBack(i)
		buf := item.Bytes()
		if item.IsZero() {
			buf = []byte{0}
		}
		t.stack.AddItem(fmt.Sprintf("%4d %64x", len(stack)-i, buf), "", 0, nil)
	}

	// Empty lines so real values are at the bottom
	for t.stack.GetItemCount() < 16 {
		t.stack.InsertItem(0, "", "", 0, nil)
	}
}

func (t *termDBG) populateMemory() {
	mem := t.State().Context.MemoryData()

	t.memory.Clear()
	for i, n := 0, len(mem); i < n; i += 32 {
		t.memory.AddItem(fmt.Sprintf("%02x %x", i, mem[i:32]), "", 0, nil)
		mem = mem[n:]
	}
}
