package ui

import (
	"context"

	"go.mau.fi/mauview"

	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
	"go.mau.fi/gomuks/tui/abstract"
)

type VerifySessionView struct {
	*mauview.Form
	Container *mauview.Centerer
	keyInput  *mauview.InputField
}

func NewVerifySessionView(ctx context.Context, app abstract.App) *VerifySessionView {
	vs := &VerifySessionView{
		Form:     mauview.NewForm(),
		keyInput: mauview.NewInputField(),
	}
	vs.Container = mauview.Center(mauview.NewBox(vs).SetTitle("Verify your device"), 64, 16)
	vs.Grid.SetColumns([]int{12, 1, 50}).SetRows([]int{1, 1})
	vs.AddFormItem(vs.keyInput, 2, 0, 1, 1).
		AddComponent(mauview.NewTextField().SetText("Recovery Key"), 0, 0, 1, 1).
		AddComponent(mauview.NewButton("Verify").SetOnClick(func() {
			app.Gmx().Log.Debug().Msg("Verifying session with recovery key")
			ok, err := app.Rpc().Verify(ctx, &jsoncmd.VerifyParams{RecoveryKey: vs.keyInput.GetText()})
			if err != nil {
				app.Gmx().Log.Error().Err(err).Msg("Failed to verify session")
				// todo: nobody's reading their log file while using gomuks
				return
			}
			if !ok {
				app.Gmx().Log.Error().Msg("Verification failed, please check your recovery key")
				return
			}
			app.Gmx().Log.Debug().Msg("Verification successful, starting sync")
			// And now control.go should swap us out.
		}), 0, 1, 1, 1).
		AddComponent(mauview.NewButton("Cancel").SetOnClick(func() {
			app.Gmx().Log.Debug().Msg("Verification cancelled")
			app.Gmx().Stop()
		}), 2, 1, 1, 1)
	vs.keyInput.SetPlaceholder("AAAA BBBB CCCC DDDD EEEE FFFF GGGG HHHH IIII JJJJ KKKK LLLL")
	return vs
}
